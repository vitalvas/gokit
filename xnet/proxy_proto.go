package xnet

import (
	"bufio"
	"errors"
	"net"
	"time"
)

var (
	ErrProxyProtoRequired     = errors.New("proxy protocol header required")
	ErrProxyProtoInvalid      = errors.New("invalid proxy protocol header")
	ErrProxyProtoUntrusted    = errors.New("proxy protocol not allowed from this source")
	ErrProxyProtoReadTimeout  = errors.New("proxy protocol header read timeout")
	ErrProxyProtoUnknownProto = errors.New("unknown proxy protocol version")
)

// ProxyProtoMode defines the operating mode for PROXY protocol handling.
type ProxyProtoMode int

const (
	// ProxyProtoModeStrict requires all connections to have a valid PROXY protocol header.
	// Connections without a valid header are rejected.
	ProxyProtoModeStrict ProxyProtoMode = iota

	// ProxyProtoModeAuto auto-detects PROXY protocol headers.
	// When TrustedProxies is configured: connections from trusted sources require PROXY header,
	// connections from other sources are accepted as regular connections.
	// When TrustedProxies is empty: attempts to detect PROXY header, falls back to regular connection.
	ProxyProtoModeAuto
)

// ProxyProtoConfig contains configuration for the PROXY protocol listener.
type ProxyProtoConfig struct {
	// Mode defines strict or auto-detect behavior.
	// Default (zero value) is ProxyProtoModeStrict.
	Mode ProxyProtoMode

	// HeaderTimeout is the maximum duration for reading the PROXY protocol header.
	// Zero value means no timeout.
	HeaderTimeout time.Duration

	// TrustedProxies contains CIDRs of trusted proxy sources.
	// In strict mode: if empty, PROXY headers are accepted from any source.
	// In auto mode: connections from trusted sources require PROXY header,
	// other connections are accepted as regular connections.
	TrustedProxies []net.IPNet
}

// ProxyProtoListen creates a new listener on the specified network address with PROXY protocol support.
func ProxyProtoListen(network, address string, config ProxyProtoConfig) (net.Listener, error) {
	listener, err := net.Listen(network, address)
	if err != nil {
		return nil, err
	}

	return NewProxyProtoListener(listener, config), nil
}

// NewProxyProtoListener wraps an existing listener with PROXY protocol support.
func NewProxyProtoListener(listener net.Listener, config ProxyProtoConfig) net.Listener {
	var matcher *CIDRMatcher
	if len(config.TrustedProxies) > 0 {
		matcher = NewCIDRMatcher(config.TrustedProxies)
	}

	return &proxyProtoListener{
		listener: listener,
		config:   config,
		matcher:  matcher,
	}
}

// proxyProtoListener wraps a net.Listener to handle PROXY protocol headers.
type proxyProtoListener struct {
	listener net.Listener
	config   ProxyProtoConfig
	matcher  *CIDRMatcher
}

// Accept waits for and returns the next connection with PROXY protocol handling.
func (l *proxyProtoListener) Accept() (net.Conn, error) {
	conn, err := l.listener.Accept()
	if err != nil {
		return nil, err
	}

	isTrusted := l.isTrustedSource(conn)

	if l.config.Mode == ProxyProtoModeStrict {
		if l.matcher != nil && !isTrusted {
			conn.Close()
			return nil, ErrProxyProtoUntrusted
		}

		proxyConn, err := l.handleProxyProtocol(conn, true)
		if err != nil {
			conn.Close()
			return nil, err
		}

		return proxyConn, nil
	}

	if l.matcher != nil && !isTrusted {
		return l.wrapRegularConn(conn), nil
	}

	proxyConn, err := l.handleProxyProtocol(conn, false)
	if err != nil {
		conn.Close()
		return nil, err
	}

	return proxyConn, nil
}

func (l *proxyProtoListener) isTrustedSource(conn net.Conn) bool {
	if l.matcher == nil {
		return true
	}

	remoteAddr, ok := conn.RemoteAddr().(*net.TCPAddr)
	if !ok {
		return false
	}

	return l.matcher.Contains(remoteAddr.IP)
}

func (l *proxyProtoListener) wrapRegularConn(conn net.Conn) *ProxyProtoConn {
	return &ProxyProtoConn{
		Conn:   conn,
		reader: bufio.NewReader(conn),
		header: nil,
	}
}

// Close closes the underlying listener.
func (l *proxyProtoListener) Close() error {
	return l.listener.Close()
}

// Addr returns the listener's network address.
func (l *proxyProtoListener) Addr() net.Addr {
	return l.listener.Addr()
}

func (l *proxyProtoListener) handleProxyProtocol(conn net.Conn, strict bool) (*ProxyProtoConn, error) {
	if l.config.HeaderTimeout > 0 {
		if err := conn.SetReadDeadline(time.Now().Add(l.config.HeaderTimeout)); err != nil {
			return nil, err
		}
	}

	reader := bufio.NewReader(conn)

	peek, err := reader.Peek(proxyV2SignatureLen)
	if err != nil {
		if strict {
			return nil, ErrProxyProtoRequired
		}

		return l.wrapReaderConn(conn, reader), nil
	}

	var header *ProxyHeader

	switch {
	case isProxyV2(peek):
		header, err = parseProxyV2(reader)
	case isProxyV1(peek):
		header, err = parseProxyV1(reader)
	default:
		if strict {
			return nil, ErrProxyProtoUnknownProto
		}

		return l.wrapReaderConn(conn, reader), nil
	}

	if err != nil {
		return nil, err
	}

	if l.config.HeaderTimeout > 0 {
		if err := conn.SetReadDeadline(time.Time{}); err != nil {
			return nil, err
		}
	}

	return &ProxyProtoConn{
		Conn:   conn,
		reader: reader,
		header: header,
	}, nil
}

func (l *proxyProtoListener) wrapReaderConn(conn net.Conn, reader *bufio.Reader) *ProxyProtoConn {
	if l.config.HeaderTimeout > 0 {
		conn.SetReadDeadline(time.Time{})
	}

	return &ProxyProtoConn{
		Conn:   conn,
		reader: reader,
		header: nil,
	}
}

// ProxyTransport identifies the transport protocol carried by a PROXY header.
type ProxyTransport int

const (
	// ProxyTransportUnspec means the transport protocol is unknown or unspecified
	// (PROXY v1 UNKNOWN, PROXY v2 LOCAL/UNSPEC, or Unix sockets).
	ProxyTransportUnspec ProxyTransport = iota

	// ProxyTransportTCP means the proxied connection is a TCP stream.
	ProxyTransportTCP

	// ProxyTransportUDP means the proxied connection is a UDP datagram flow.
	ProxyTransportUDP
)

// PROXY protocol v2 TLV types.
const (
	// PP2TypeAWS is the AWS vendor-specific TLV type (PP2_TYPE_AWS).
	PP2TypeAWS byte = 0xEA
)

// AWS PP2_TYPE_AWS subtypes (the first octet of the TLV value).
const (
	// PP2SubtypeAWSVPCEID identifies the VPC endpoint ID subtype
	// (PP2_SUBTYPE_AWS_VPCE_ID).
	PP2SubtypeAWSVPCEID byte = 0x01
)

// ProxyTLV is a single PROXY protocol v2 Type-Length-Value vector. Value holds
// the raw bytes following the type and length fields.
type ProxyTLV struct {
	Type  byte
	Value []byte
}

// ProxyHeader contains parsed PROXY protocol header information.
//
// SourceAddr and DestAddr hold the original client and destination addresses.
// Their concrete type matches Transport: *net.TCPAddr for TCP and *net.UDPAddr
// for UDP. They are nil for headers without address information (PROXY v1
// UNKNOWN, PROXY v2 LOCAL/UNSPEC, or Unix sockets).
type ProxyHeader struct {
	SourceAddr net.Addr
	DestAddr   net.Addr

	// Transport is the transport protocol declared by the header.
	Transport ProxyTransport

	// TLVs holds the additional Type-Length-Value vectors carried by a
	// PROXY protocol v2 header, in the order they appear. It is nil for v1
	// headers and for v2 headers without TLVs.
	TLVs []ProxyTLV
}

// TLV returns the value of the first TLV with the given type and whether such a
// TLV is present.
func (h *ProxyHeader) TLV(typ byte) ([]byte, bool) {
	for _, tlv := range h.TLVs {
		if tlv.Type == typ {
			return tlv.Value, true
		}
	}

	return nil, false
}

// VPCEndpointID returns the AWS VPC endpoint ID carried by the PP2_TYPE_AWS TLV
// with the PP2_SUBTYPE_AWS_VPCE_ID subtype, and whether it is present. Network
// Load Balancers fronting a VPC endpoint service add this TLV to the PROXY
// protocol v2 header.
func (h *ProxyHeader) VPCEndpointID() (string, bool) {
	value, ok := h.TLV(PP2TypeAWS)
	if !ok || len(value) < 1 || value[0] != PP2SubtypeAWSVPCEID {
		return "", false
	}

	return string(value[1:]), true
}

// ProxyProtoConn wraps a net.Conn with PROXY protocol information.
type ProxyProtoConn struct {
	net.Conn
	reader *bufio.Reader
	header *ProxyHeader
}

// Read reads data from the connection.
func (c *ProxyProtoConn) Read(b []byte) (int, error) {
	return c.reader.Read(b)
}

// RemoteAddr returns the remote address from the PROXY protocol header.
func (c *ProxyProtoConn) RemoteAddr() net.Addr {
	if c.header != nil && c.header.SourceAddr != nil {
		return c.header.SourceAddr
	}

	return c.Conn.RemoteAddr()
}

// LocalAddr returns the destination address from the PROXY protocol header.
func (c *ProxyProtoConn) LocalAddr() net.Addr {
	if c.header != nil && c.header.DestAddr != nil {
		return c.header.DestAddr
	}

	return c.Conn.LocalAddr()
}

// ProxyHeader returns the parsed PROXY protocol header.
func (c *ProxyProtoConn) ProxyHeader() *ProxyHeader {
	return c.header
}

// RealRemoteAddr returns the original remote address of the connection (proxy address).
func (c *ProxyProtoConn) RealRemoteAddr() net.Addr {
	return c.Conn.RemoteAddr()
}
