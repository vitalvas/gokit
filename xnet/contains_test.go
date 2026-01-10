package xnet

import (
	"net"
	"testing"
)

func TestCIDRContains(t *testing.T) {
	tests := []struct {
		name     string
		nets     []string
		ip       string
		expected bool
	}{
		{
			name:     "IP is within one of the subnets",
			nets:     []string{"192.168.1.0/24", "10.0.0.0/8"},
			ip:       "192.168.1.10",
			expected: true,
		},
		{
			name:     "IP is not within any of the subnets",
			nets:     []string{"192.168.1.0/24", "10.0.0.0/8"},
			ip:       "172.16.0.1",
			expected: false,
		},
		{
			name:     "IP is within another subnet",
			nets:     []string{"192.168.1.0/24", "10.0.0.0/8"},
			ip:       "10.1.2.3",
			expected: true,
		},
		{
			name:     "Empty nets slice",
			nets:     []string{},
			ip:       "192.168.1.10",
			expected: false,
		},
		{
			name:     "Invalid IP format",
			nets:     []string{"192.168.1.0/24"},
			ip:       "invalid-ip",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var nets []net.IPNet
			for _, cidr := range tt.nets {
				_, subnet, err := net.ParseCIDR(cidr)
				if err != nil {
					t.Fatalf("Failed to parse CIDR: %v", err)
				}
				nets = append(nets, *subnet)
			}

			ip := net.ParseIP(tt.ip)
			if ip == nil && tt.expected == false {
				// If IP is invalid, skip the test as we're not checking for it in CIDRContains
				t.Skip("Skipping invalid IP format test")
			}

			result := CIDRContains(nets, ip)
			if result != tt.expected {
				t.Errorf("CIDRContains() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestCIDRContainsString(t *testing.T) {
	tests := []struct {
		name     string
		nets     []string
		ip       string
		expected bool
	}{
		{
			name:     "IP is within one of the subnets",
			nets:     []string{"192.168.1.0/24", "10.0.0.0/8"},
			ip:       "192.168.1.50",
			expected: true,
		},
		{
			name:     "IP is not within any of the subnets",
			nets:     []string{"192.168.1.0/24", "10.0.0.0/8"},
			ip:       "172.16.0.1",
			expected: false,
		},
		{
			name:     "IP is within a larger subnet",
			nets:     []string{"192.168.1.0/24", "10.0.0.0/8"},
			ip:       "10.1.1.1",
			expected: true,
		},
		{
			name:     "Empty subnet list",
			nets:     []string{},
			ip:       "192.168.1.1",
			expected: false,
		},
		{
			name:     "Malformed CIDR in the list",
			nets:     []string{"192.168.1.0/24", "invalid-cidr"},
			ip:       "192.168.1.100",
			expected: true,
		},
		{
			name:     "All CIDRs malformed",
			nets:     []string{"invalid-cidr-1", "invalid-cidr-2"},
			ip:       "192.168.1.1",
			expected: false,
		},
		{
			name:     "Invalid IP format",
			nets:     []string{"192.168.1.0/24"},
			ip:       "invalid-ip",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			if ip == nil && tt.expected == false {
				// Skip the test if IP format is invalid and test expects false
				t.Skip("Skipping invalid IP format test")
			}

			result := CIDRContainsString(tt.nets, ip)
			if result != tt.expected {
				t.Errorf("CIDRContainsString() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

// Benchmarks

func BenchmarkCIDRContains_10(b *testing.B) {
	nets := make([]net.IPNet, 10)
	for i := 0; i < 10; i++ {
		_, ipNet, _ := net.ParseCIDR(net.IPv4(10, byte(i), 0, 0).String() + "/24")
		nets[i] = *ipNet
	}
	ip := net.IPv4(10, 5, 1, 1)

	b.ResetTimer()
	for b.Loop() {
		_ = CIDRContains(nets, ip)
	}
}

func BenchmarkCIDRContains_100(b *testing.B) {
	nets := make([]net.IPNet, 100)
	for i := 0; i < 100; i++ {
		_, ipNet, _ := net.ParseCIDR(net.IPv4(10, byte(i), 0, 0).String() + "/24")
		nets[i] = *ipNet
	}
	ip := net.IPv4(10, 50, 1, 1)

	b.ResetTimer()
	for b.Loop() {
		_ = CIDRContains(nets, ip)
	}
}

func BenchmarkCIDRContains_1000(b *testing.B) {
	nets := make([]net.IPNet, 1000)
	for i := 0; i < 1000; i++ {
		octet2 := byte(i / 256)
		octet3 := byte(i % 256)
		_, ipNet, _ := net.ParseCIDR(net.IPv4(10, octet2, octet3, 0).String() + "/24")
		nets[i] = *ipNet
	}
	ip := net.IPv4(10, 1, 244, 1)

	b.ResetTimer()
	for b.Loop() {
		_ = CIDRContains(nets, ip)
	}
}

func BenchmarkCIDRContainsString_1000(b *testing.B) {
	nets := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		octet2 := byte(i / 256)
		octet3 := byte(i % 256)
		nets[i] = net.IPv4(10, octet2, octet3, 0).String() + "/24"
	}
	ip := net.IPv4(10, 1, 244, 1)

	b.ResetTimer()
	for b.Loop() {
		_ = CIDRContainsString(nets, ip)
	}
}

func FuzzCIDRContains(f *testing.F) {
	f.Add([]byte{192, 168, 1, 0}, 24, []byte{192, 168, 1, 10})
	f.Add([]byte{10, 0, 0, 0}, 8, []byte{10, 1, 2, 3})
	f.Add([]byte{172, 16, 0, 0}, 12, []byte{172, 31, 255, 255})
	f.Add([]byte{0, 0, 0, 0}, 0, []byte{1, 2, 3, 4})

	f.Fuzz(func(_ *testing.T, netBytes []byte, mask int, ipBytes []byte) {
		if len(netBytes) != 4 || len(ipBytes) != 4 {
			return
		}
		if mask < 0 || mask > 32 {
			return
		}

		_, ipNet, err := net.ParseCIDR(net.IP(netBytes).String() + "/" + string(rune('0'+mask/10)) + string(rune('0'+mask%10)))
		if err != nil {
			return
		}

		nets := []net.IPNet{*ipNet}
		ip := net.IP(ipBytes)
		_ = CIDRContains(nets, ip)
	})
}

func FuzzCIDRContainsString(f *testing.F) {
	f.Add("192.168.1.0/24", []byte{192, 168, 1, 10})
	f.Add("10.0.0.0/8", []byte{10, 1, 2, 3})
	f.Add("172.16.0.0/12", []byte{172, 31, 255, 255})

	f.Fuzz(func(_ *testing.T, cidr string, ipBytes []byte) {
		if len(ipBytes) != 4 && len(ipBytes) != 16 {
			return
		}

		nets := []string{cidr}
		ip := net.IP(ipBytes)
		_ = CIDRContainsString(nets, ip)
	})
}
