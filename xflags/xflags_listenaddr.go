package xflags

import (
	"fmt"
	"net/netip"
	"strings"
)

// ListenAddr is a network listen address that accepts the ":port" shorthand in
// addition to a full host:port address. The shorthand binds the dual-stack
// wildcard "[::]:port", so the listener serves both native IPv6 and IPv4-mapped
// clients. It implements encoding.TextUnmarshaler and encoding.TextMarshaler, so
// it works as an xflags option type and round-trips through text.
type ListenAddr netip.AddrPort

// ParseListenAddr parses a listen address. A value beginning with ":" is treated
// as a port-only shorthand and bound to the dual-stack wildcard "[::]:port";
// any other value is parsed as a full host:port address.
func ParseListenAddr(s string) (ListenAddr, error) {
	candidate := s
	if port, ok := strings.CutPrefix(s, ":"); ok {
		// ":8428" binds the dual-stack wildcard so the listener serves both
		// native IPv6 and IPv4-mapped clients.
		candidate = fmt.Sprintf("[::]:%s", port)
	}
	addr, err := netip.ParseAddrPort(candidate)
	if err != nil {
		return ListenAddr{}, fmt.Errorf("invalid listen address %q: %w", s, err)
	}
	return ListenAddr(addr), nil
}

// AddrPort returns the underlying netip.AddrPort.
func (l ListenAddr) AddrPort() netip.AddrPort {
	return netip.AddrPort(l)
}

// UnmarshalText parses a listen address, implementing encoding.TextUnmarshaler.
func (l *ListenAddr) UnmarshalText(text []byte) error {
	v, err := ParseListenAddr(string(text))
	if err != nil {
		return err
	}
	*l = v
	return nil
}

// MarshalText renders the resolved address as host:port, implementing
// encoding.TextMarshaler.
func (l ListenAddr) MarshalText() ([]byte, error) {
	return []byte(l.String()), nil
}

// String renders the resolved address as host:port.
func (l ListenAddr) String() string {
	return netip.AddrPort(l).String()
}
