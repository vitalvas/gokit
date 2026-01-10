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

// ProxyHeader contains parsed PROXY protocol header information.
type ProxyHeader struct {
	SourceAddr *net.TCPAddr
	DestAddr   *net.TCPAddr
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
