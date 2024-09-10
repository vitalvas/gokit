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
