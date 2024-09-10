package xnet

import (
	"errors"
	"net"
)

var (
	ErrInvalidIPAddress    = errors.New("invalid IP address")
	ErrInvalidIPv4MaskSize = errors.New("invalid IPv4 mask size")
	ErrInvalidIPv6MaskSize = errors.New("invalid IPv6 mask size")
	ErrInvalidIPMaskSize   = errors.New("both IPv4 and IPv6 mask sizes cannot be zero")
	ErrNonStandardIP       = errors.New("this is a non-standard IP or IPv6 address")
)

func GetStripedAddress(addr net.IP, ipv4, ipv6 int) (net.IP, error) {
	if addr == nil {
		return nil, ErrInvalidIPAddress
	}

	if ipv4 < 0 || ipv4 > 32 {
		return nil, ErrInvalidIPv4MaskSize
	}

	if ipv6 < 0 || ipv6 > 128 {
		return nil, ErrInvalidIPv6MaskSize
	}

	if ipv4 == 0 && ipv6 == 0 {
		return nil, ErrInvalidIPMaskSize
	}

	if addr.To4() != nil && ipv4 > 0 {
		mask := net.CIDRMask(ipv4, 32)
		network := addr.Mask(mask)

		return network, nil
	}

	if addr.To16() != nil && ipv6 > 0 {
		mask := net.CIDRMask(ipv6, 128)
		network := addr.Mask(mask)

		return network, nil
	}

	return nil, ErrNonStandardIP
}
