package xnet

import "net"

func CIDRContains(nets []net.IPNet, ip net.IP) bool {
	for _, net := range nets {
		if net.Contains(ip) {
			return true
		}
	}

	return false
}

func CIDRContainsString(nets []string, ip net.IP) bool {
	for _, n := range nets {
		_, ipNet, err := net.ParseCIDR(n)
		if err != nil {
			continue
		}

		if ipNet.Contains(ip) {
			return true
		}
	}

	return false
}
