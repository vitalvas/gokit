package xnet

import (
	"net"
	"sync"
	"time"
)

// ProxyProtoUDPConfig contains configuration for the PROXY protocol UDP wrapper.
type ProxyProtoUDPConfig struct {
	// Mode defines strict or auto-detect behavior.
	// Default (zero value) is ProxyProtoModeStrict.
	Mode ProxyProtoMode

	// TrustedProxies contains CIDRs of trusted proxy sources.
	// In strict mode: if empty, PROXY headers are accepted from any source.
	// In auto mode: datagrams from trusted sources require a PROXY header,
	// datagrams from other sources are delivered as regular datagrams.
	TrustedProxies []net.IPNet

	// MaxDatagramSize is the buffer size used to read datagrams from the
	// underlying connection. When zero, defaultUDPDatagramSize is used.
	MaxDatagramSize int
}

// defaultUDPDatagramSize is large enough for a full IPv6 PROXY v2 header plus a
// jumbo UDP payload while staying within a single read.
const defaultUDPDatagramSize = 65535

// ProxyProtoListenPacket creates a PROXY protocol aware UDP packet connection on
// the given network address. The network must be a UDP network ("udp", "udp4"
// or "udp6").
func ProxyProtoListenPacket(network, address string, config ProxyProtoUDPConfig) (net.PacketConn, error) {
	conn, err := net.ListenPacket(network, address)
	if err != nil {
		return nil, err
	}

	return NewProxyProtoPacketConn(conn, config), nil
}

// NewProxyProtoPacketConn wraps an existing packet connection with PROXY
// protocol v2 datagram support. Each datagram is expected to be prefixed with a
// PROXY protocol v2 header (the only version defined for datagrams).
func NewProxyProtoPacketConn(conn net.PacketConn, config ProxyProtoUDPConfig) net.PacketConn {
	var matcher *CIDRMatcher
	if len(config.TrustedProxies) > 0 {
		matcher = NewCIDRMatcher(config.TrustedProxies)
	}

	size := config.MaxDatagramSize
	if size <= 0 {
		size = defaultUDPDatagramSize
	}

	return &ProxyProtoPacketConn{
		conn:    conn,
		config:  config,
		matcher: matcher,
		bufSize: size,
		routes:  make(map[string]net.Addr),
	}
}

// ProxyProtoPacketConn wraps a net.PacketConn to handle PROXY protocol v2
// headers on a per-datagram basis.
type ProxyProtoPacketConn struct {
	conn    net.PacketConn
	config  ProxyProtoUDPConfig
	matcher *CIDRMatcher
	bufSize int

	mu     sync.Mutex
	routes map[string]net.Addr
}

// ReadFrom reads a single datagram and copies its payload into p. When the
// datagram carries a PROXY protocol header, addr is the original client address
// declared by that header; otherwise addr is the address of the datagram sender.
func (c *ProxyProtoPacketConn) ReadFrom(p []byte) (int, net.Addr, error) {
	n, _, src, err := c.readFromProxy(p)
	return n, src, err
}

// ReadFromProxy behaves like ReadFrom but additionally returns the parsed PROXY
// protocol header (nil for regular datagrams) and the real address of the
// datagram sender (the proxy when a header is present).
func (c *ProxyProtoPacketConn) ReadFromProxy(p []byte) (n int, header *ProxyHeader, src net.Addr, err error) {
	return c.readFromProxy(p)
}

func (c *ProxyProtoPacketConn) readFromProxy(p []byte) (int, *ProxyHeader, net.Addr, error) {
	buf := make([]byte, c.bufSize)

	n, sender, err := c.conn.ReadFrom(buf)
	if err != nil {
		return 0, nil, sender, err
	}

	datagram := buf[:n]
	trusted := c.isTrustedSource(sender)

	if c.config.Mode == ProxyProtoModeStrict {
		if c.matcher != nil && !trusted {
			return 0, nil, sender, ErrProxyProtoUntrusted
		}

		header, payload, err := c.decode(datagram)
		if err != nil {
			return 0, nil, sender, err
		}

		c.recordRoute(header, sender)

		return copyPayload(p, payload), header, c.clientAddr(header, sender), nil
	}

	if c.matcher != nil && !trusted {
		return copyPayload(p, datagram), nil, sender, nil
	}

	header, payload, ok := c.tryDecode(datagram)
	if !ok {
		return copyPayload(p, datagram), nil, sender, nil
	}

	c.recordRoute(header, sender)

	return copyPayload(p, payload), header, c.clientAddr(header, sender), nil
}

// WriteTo writes a regular datagram to addr without a PROXY header. When addr is
// an original client address previously seen via ReadFrom, the datagram is sent
// to the proxy that delivered it so the reply reaches the client.
func (c *ProxyProtoPacketConn) WriteTo(p []byte, addr net.Addr) (int, error) {
	target := c.resolveRoute(addr)

	if _, err := c.conn.WriteTo(p, target); err != nil {
		return 0, err
	}

	return len(p), nil
}

// WriteToProxy writes p to addr prefixed with a PROXY protocol v2 header that
// declares src as the original source and addr as the destination. Use this to
// emit PROXY protocol datagrams towards a backend.
func (c *ProxyProtoPacketConn) WriteToProxy(p []byte, src, addr net.Addr) (int, error) {
	header := encodeProxyV2(src, addr)

	datagram := make([]byte, len(header)+len(p))
	copy(datagram, header)
	copy(datagram[len(header):], p)

	if _, err := c.conn.WriteTo(datagram, addr); err != nil {
		return 0, err
	}

	return len(p), nil
}

// Close closes the underlying packet connection.
func (c *ProxyProtoPacketConn) Close() error {
	return c.conn.Close()
}

// LocalAddr returns the local network address of the underlying connection.
func (c *ProxyProtoPacketConn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

// SetDeadline sets the read and write deadlines on the underlying connection.
func (c *ProxyProtoPacketConn) SetDeadline(t time.Time) error {
	return c.conn.SetDeadline(t)
}

// SetReadDeadline sets the read deadline on the underlying connection.
func (c *ProxyProtoPacketConn) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}

// SetWriteDeadline sets the write deadline on the underlying connection.
func (c *ProxyProtoPacketConn) SetWriteDeadline(t time.Time) error {
	return c.conn.SetWriteDeadline(t)
}

// decode parses a required PROXY v2 header from a datagram, returning the header
// and the payload that follows it.
func (c *ProxyProtoPacketConn) decode(datagram []byte) (*ProxyHeader, []byte, error) {
	if !isProxyV2(datagram) {
		return nil, nil, ErrProxyProtoUnknownProto
	}

	header, offset, err := parseProxyV2Datagram(datagram)
	if err != nil {
		return nil, nil, err
	}

	return header, datagram[offset:], nil
}

// tryDecode attempts to parse a PROXY v2 header, reporting ok=false when the
// datagram is not a valid PROXY v2 datagram.
func (c *ProxyProtoPacketConn) tryDecode(datagram []byte) (*ProxyHeader, []byte, bool) {
	if !isProxyV2(datagram) {
		return nil, nil, false
	}

	header, offset, err := parseProxyV2Datagram(datagram)
	if err != nil {
		return nil, nil, false
	}

	return header, datagram[offset:], true
}

func (c *ProxyProtoPacketConn) isTrustedSource(addr net.Addr) bool {
	if c.matcher == nil {
		return true
	}

	udpAddr, ok := addr.(*net.UDPAddr)
	if !ok {
		return false
	}

	return c.matcher.Contains(udpAddr.IP)
}

// clientAddr returns the original client address from the header when present,
// falling back to the datagram sender.
func (c *ProxyProtoPacketConn) clientAddr(header *ProxyHeader, sender net.Addr) net.Addr {
	if header != nil && header.SourceAddr != nil {
		return header.SourceAddr
	}

	return sender
}

// recordRoute remembers which proxy delivered datagrams for a given client so
// replies addressed to the client can be routed back through that proxy.
func (c *ProxyProtoPacketConn) recordRoute(header *ProxyHeader, sender net.Addr) {
	if header == nil || header.SourceAddr == nil {
		return
	}

	c.mu.Lock()
	c.routes[header.SourceAddr.String()] = sender
	c.mu.Unlock()
}

// resolveRoute maps an original client address back to the proxy that delivered
// its datagrams, or returns addr unchanged when no route is known.
func (c *ProxyProtoPacketConn) resolveRoute(addr net.Addr) net.Addr {
	c.mu.Lock()
	target, ok := c.routes[addr.String()]
	c.mu.Unlock()

	if ok {
		return target
	}

	return addr
}

// copyPayload copies payload into p, returning the number of bytes copied.
func copyPayload(p, payload []byte) int {
	return copy(p, payload)
}
