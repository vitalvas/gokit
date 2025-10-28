package xnet

import (
	"net"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCIDRMerge(t *testing.T) {
	t.Run("empty input", func(t *testing.T) {
		result := CIDRMerge([]net.IPNet{})
		assert.Empty(t, result)
	})

	t.Run("single CIDR", func(t *testing.T) {
		_, ipNet, err := net.ParseCIDR("192.168.1.0/24")
		require.NoError(t, err)

		result := CIDRMerge([]net.IPNet{*ipNet})
		assert.Len(t, result, 1)
		assert.Equal(t, "192.168.1.0/24", result[0].String())
	})

	t.Run("non-overlapping CIDRs", func(t *testing.T) {
		cidrs := []string{"192.168.1.0/24", "10.0.0.0/8", "172.16.0.0/12"}
		nets := parseCIDRs(t, cidrs)

		result := CIDRMerge(nets)
		assert.Len(t, result, 3)

		// Sort for consistent comparison
		resultStrs := netIPNetsToStrings(result)
		sort.Strings(resultStrs)
		expected := []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.1.0/24"}
		assert.Equal(t, expected, resultStrs)
	})

	t.Run("adjacent CIDRs merge into larger block", func(t *testing.T) {
		cidrs := []string{"192.168.0.0/25", "192.168.0.128/25"}
		nets := parseCIDRs(t, cidrs)

		result := CIDRMerge(nets)
		require.Len(t, result, 1)
		assert.Equal(t, "192.168.0.0/24", result[0].String())
	})

	t.Run("four adjacent /26 networks merge into /24", func(t *testing.T) {
		cidrs := []string{
			"10.0.0.0/26",
			"10.0.0.64/26",
			"10.0.0.128/26",
			"10.0.0.192/26",
		}
		nets := parseCIDRs(t, cidrs)

		result := CIDRMerge(nets)
		require.Len(t, result, 1)
		assert.Equal(t, "10.0.0.0/24", result[0].String())
	})

	t.Run("overlapping CIDRs - one contains another", func(t *testing.T) {
		cidrs := []string{"10.0.0.0/8", "10.1.0.0/16", "10.1.1.0/24"}
		nets := parseCIDRs(t, cidrs)

		result := CIDRMerge(nets)
		require.Len(t, result, 1)
		assert.Equal(t, "10.0.0.0/8", result[0].String())
	})

	t.Run("duplicate CIDRs", func(t *testing.T) {
		cidrs := []string{"192.168.1.0/24", "192.168.1.0/24", "192.168.1.0/24"}
		nets := parseCIDRs(t, cidrs)

		result := CIDRMerge(nets)
		require.Len(t, result, 1)
		assert.Equal(t, "192.168.1.0/24", result[0].String())
	})

	t.Run("mixed IPv4 and IPv6", func(t *testing.T) {
		cidrs := []string{
			"192.168.1.0/24",
			"2001:db8::/32",
			"10.0.0.0/8",
			"2001:db8:1::/48",
		}
		nets := parseCIDRs(t, cidrs)

		result := CIDRMerge(nets)
		assert.Len(t, result, 3)

		resultStrs := netIPNetsToStrings(result)
		assert.Contains(t, resultStrs, "10.0.0.0/8")
		assert.Contains(t, resultStrs, "192.168.1.0/24")
		assert.Contains(t, resultStrs, "2001:db8::/32")
	})

	t.Run("IPv6 adjacent networks merge", func(t *testing.T) {
		cidrs := []string{
			"2001:db8::/33",
			"2001:db8:8000::/33",
		}
		nets := parseCIDRs(t, cidrs)

		result := CIDRMerge(nets)
		require.Len(t, result, 1)
		assert.Equal(t, "2001:db8::/32", result[0].String())
	})

	t.Run("partial overlap - different mask sizes", func(t *testing.T) {
		cidrs := []string{
			"192.168.0.0/24",
			"192.168.1.0/24",
			"192.168.2.0/23",
		}
		nets := parseCIDRs(t, cidrs)

		result := CIDRMerge(nets)
		require.Len(t, result, 1)
		assert.Equal(t, "192.168.0.0/22", result[0].String())
	})

	t.Run("complex merging scenario", func(t *testing.T) {
		cidrs := []string{
			"10.0.0.0/24",
			"10.0.1.0/24",
			"10.0.2.0/24",
			"10.0.3.0/24",
			"10.0.4.0/24",
			"10.0.5.0/24",
			"10.0.6.0/24",
			"10.0.7.0/24",
		}
		nets := parseCIDRs(t, cidrs)

		result := CIDRMerge(nets)
		require.Len(t, result, 1)
		assert.Equal(t, "10.0.0.0/21", result[0].String())
	})

	t.Run("unmerged networks remain separate", func(t *testing.T) {
		cidrs := []string{
			"192.168.0.0/24",
			"192.168.2.0/24",
			"192.168.4.0/24",
		}
		nets := parseCIDRs(t, cidrs)

		result := CIDRMerge(nets)
		assert.Len(t, result, 3)

		resultStrs := netIPNetsToStrings(result)
		sort.Strings(resultStrs)
		assert.Equal(t, cidrs, resultStrs)
	})

	t.Run("split 192.168.0.0/16 to /24 and merge back", func(t *testing.T) {
		// Generate all 256 /24 subnets from 192.168.0.0/16
		cidrs := make([]string, 256)
		for i := 0; i < 256; i++ {
			cidrs[i] = net.IPv4(192, 168, byte(i), 0).String() + "/24"
		}

		nets := parseCIDRs(t, cidrs)
		require.Len(t, nets, 256)

		// Merge all /24 networks back into /16
		result := CIDRMerge(nets)
		require.Len(t, result, 1)
		assert.Equal(t, "192.168.0.0/16", result[0].String())
	})

	t.Run("split 2001:db8::/32 to /48 and merge back", func(t *testing.T) {
		// Generate all 65536 /48 subnets from 2001:db8::/32
		// /32 to /48 means 16 bits of variation (2^16 = 65536 subnets)
		cidrs := make([]string, 65536)
		for i := 0; i < 65536; i++ {
			// 2001:db8:XXXX::/48 where XXXX varies from 0000 to ffff
			ip := net.IP{
				0x20, 0x01, 0x0d, 0xb8, // 2001:db8
				byte(i >> 8), byte(i & 0xff), 0x00, 0x00, // third hextet varies
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			}
			cidrs[i] = ip.String() + "/48"
		}

		nets := parseCIDRs(t, cidrs)
		require.Len(t, nets, 65536)

		// Merge all /48 networks back into /32
		result := CIDRMerge(nets)
		require.Len(t, result, 1)
		assert.Equal(t, "2001:db8::/32", result[0].String())
	})

	t.Run("IPv4 with gaps - partial merge", func(t *testing.T) {
		// Generate /24 subnets from 192.168.0.0/16 with gaps
		// Include: 0-63, skip 64-127, include 128-191, skip 192-255
		// This should merge into two /18 blocks
		cidrs := make([]string, 0, 128)
		for i := 0; i < 64; i++ {
			cidrs = append(cidrs, net.IPv4(192, 168, byte(i), 0).String()+"/24")
		}
		for i := 128; i < 192; i++ {
			cidrs = append(cidrs, net.IPv4(192, 168, byte(i), 0).String()+"/24")
		}

		nets := parseCIDRs(t, cidrs)
		require.Len(t, nets, 128)

		result := CIDRMerge(nets)
		require.Len(t, result, 2)

		resultStrs := netIPNetsToStrings(result)
		sort.Strings(resultStrs)
		assert.Equal(t, []string{"192.168.0.0/18", "192.168.128.0/18"}, resultStrs)
	})

	t.Run("IPv4 with small gaps - multiple blocks", func(t *testing.T) {
		// Generate /24 subnets with small gaps that prevent full merging
		// Include: 0-3, skip 4-7, include 8-11, skip 12-15
		// Each group of 4 consecutive /24s should merge into /22
		cidrs := []string{
			"10.0.0.0/24", "10.0.1.0/24", "10.0.2.0/24", "10.0.3.0/24",
			"10.0.8.0/24", "10.0.9.0/24", "10.0.10.0/24", "10.0.11.0/24",
		}

		nets := parseCIDRs(t, cidrs)
		result := CIDRMerge(nets)
		require.Len(t, result, 2)

		resultStrs := netIPNetsToStrings(result)
		sort.Strings(resultStrs)
		assert.Equal(t, []string{"10.0.0.0/22", "10.0.8.0/22"}, resultStrs)
	})

	t.Run("IPv6 with gaps - partial merge", func(t *testing.T) {
		// Generate /48 subnets from 2001:db8::/32 with gaps
		// Include first 256 (0x0000-0x00ff) and skip the rest
		// These 256 should merge into 2001:db8::/40
		cidrs := make([]string, 256)
		for i := 0; i < 256; i++ {
			ip := net.IP{
				0x20, 0x01, 0x0d, 0xb8, // 2001:db8
				0x00, byte(i), 0x00, 0x00, // third hextet 0x00XX
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			}
			cidrs[i] = ip.String() + "/48"
		}

		nets := parseCIDRs(t, cidrs)
		require.Len(t, nets, 256)

		result := CIDRMerge(nets)
		require.Len(t, result, 1)
		assert.Equal(t, "2001:db8::/40", result[0].String())
	})

	t.Run("IPv6 with multiple gaps - several blocks", func(t *testing.T) {
		// Generate /48 subnets with gaps creating multiple blocks
		// Block 1: 2001:db8:0::/48 to 2001:db8:3::/48 (4 subnets -> /46)
		// Gap
		// Block 2: 2001:db8:100::/48 to 2001:db8:103::/48 (4 subnets -> /46)
		cidrs := make([]string, 0, 8)
		for i := 0; i < 4; i++ {
			ip := net.IP{
				0x20, 0x01, 0x0d, 0xb8,
				0x00, byte(i), 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			}
			cidrs = append(cidrs, ip.String()+"/48")
		}
		for i := 0x100; i < 0x104; i++ {
			ip := net.IP{
				0x20, 0x01, 0x0d, 0xb8,
				byte(i >> 8), byte(i & 0xff), 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			}
			cidrs = append(cidrs, ip.String()+"/48")
		}

		nets := parseCIDRs(t, cidrs)
		require.Len(t, nets, 8)

		result := CIDRMerge(nets)
		require.Len(t, result, 2)

		resultStrs := netIPNetsToStrings(result)
		sort.Strings(resultStrs)
		assert.Equal(t, []string{"2001:db8:100::/46", "2001:db8::/46"}, resultStrs)
	})
}

func TestCIDRMergeString(t *testing.T) {
	t.Run("empty input", func(t *testing.T) {
		result, err := CIDRMergeString([]string{})
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("valid CIDRs merge successfully", func(t *testing.T) {
		cidrs := []string{"192.168.0.0/25", "192.168.0.128/25"}
		result, err := CIDRMergeString(cidrs)
		require.NoError(t, err)
		require.Len(t, result, 1)
		assert.Equal(t, "192.168.0.0/24", result[0])
	})

	t.Run("invalid CIDR returns error", func(t *testing.T) {
		cidrs := []string{"192.168.0.0/25", "invalid-cidr"}
		result, err := CIDRMergeString(cidrs)
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidCIDR, err)
		assert.Nil(t, result)
	})

	t.Run("mixed IPv4 and IPv6 strings", func(t *testing.T) {
		cidrs := []string{
			"10.0.0.0/24",
			"10.0.1.0/24",
			"2001:db8::/33",
			"2001:db8:8000::/33",
		}
		result, err := CIDRMergeString(cidrs)
		require.NoError(t, err)
		assert.Len(t, result, 2)

		sort.Strings(result)
		assert.Equal(t, []string{"10.0.0.0/23", "2001:db8::/32"}, result)
	})

	t.Run("complex real-world scenario", func(t *testing.T) {
		cidrs := []string{
			"192.168.0.0/24",
			"192.168.1.0/24",
			"10.0.0.0/24",
			"10.0.1.0/24",
			"10.0.2.0/24",
			"10.0.3.0/24",
			"172.16.0.0/16",
		}
		result, err := CIDRMergeString(cidrs)
		require.NoError(t, err)
		assert.Len(t, result, 3)

		sort.Strings(result)
		expected := []string{"10.0.0.0/22", "172.16.0.0/16", "192.168.0.0/23"}
		assert.Equal(t, expected, result)
	})

	t.Run("all invalid CIDRs", func(t *testing.T) {
		cidrs := []string{"invalid1", "invalid2", "not-a-cidr"}
		result, err := CIDRMergeString(cidrs)
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidCIDR, err)
		assert.Nil(t, result)
	})
}

func TestIsSubnetOf(t *testing.T) {
	t.Run("subnet is contained in network", func(t *testing.T) {
		_, network, _ := net.ParseCIDR("10.0.0.0/8")
		_, subnet, _ := net.ParseCIDR("10.1.0.0/16")

		assert.True(t, isSubnetOf(*subnet, *network))
	})

	t.Run("subnet is not contained in network", func(t *testing.T) {
		_, network, _ := net.ParseCIDR("10.0.0.0/8")
		_, subnet, _ := net.ParseCIDR("192.168.0.0/16")

		assert.False(t, isSubnetOf(*subnet, *network))
	})

	t.Run("network cannot be subnet of smaller network", func(t *testing.T) {
		_, network, _ := net.ParseCIDR("10.0.0.0/16")
		_, largerNet, _ := net.ParseCIDR("10.0.0.0/8")

		assert.False(t, isSubnetOf(*largerNet, *network))
	})

	t.Run("identical networks", func(t *testing.T) {
		_, network1, _ := net.ParseCIDR("10.0.0.0/24")
		_, network2, _ := net.ParseCIDR("10.0.0.0/24")

		assert.True(t, isSubnetOf(*network1, *network2))
	})
}

func TestTryMerge(t *testing.T) {
	t.Run("adjacent networks merge successfully", func(t *testing.T) {
		_, net1, _ := net.ParseCIDR("192.168.0.0/25")
		_, net2, _ := net.ParseCIDR("192.168.0.128/25")

		merged, ok := tryMerge(*net1, *net2)
		assert.True(t, ok)
		assert.Equal(t, "192.168.0.0/24", merged.String())
	})

	t.Run("non-adjacent networks do not merge", func(t *testing.T) {
		_, net1, _ := net.ParseCIDR("192.168.0.0/24")
		_, net2, _ := net.ParseCIDR("192.168.2.0/24")

		_, ok := tryMerge(*net1, *net2)
		assert.False(t, ok)
	})

	t.Run("different mask sizes do not merge", func(t *testing.T) {
		_, net1, _ := net.ParseCIDR("192.168.0.0/24")
		_, net2, _ := net.ParseCIDR("192.168.1.0/25")

		_, ok := tryMerge(*net1, *net2)
		assert.False(t, ok)
	})

	t.Run("IPv6 adjacent networks merge", func(t *testing.T) {
		_, net1, _ := net.ParseCIDR("2001:db8::/33")
		_, net2, _ := net.ParseCIDR("2001:db8:8000::/33")

		merged, ok := tryMerge(*net1, *net2)
		assert.True(t, ok)
		assert.Equal(t, "2001:db8::/32", merged.String())
	})
}

func TestIPToInt(t *testing.T) {
	t.Run("IPv4 conversion", func(t *testing.T) {
		ip := net.ParseIP("192.168.1.1")
		result := ipToInt(ip)
		expected := uint64(192)<<24 | uint64(168)<<16 | uint64(1)<<8 | uint64(1)
		assert.Equal(t, expected, result)
	})

	t.Run("IPv6 conversion uses first 64 bits", func(t *testing.T) {
		ip := net.ParseIP("2001:db8::1")
		result := ipToInt(ip)
		assert.Greater(t, result, uint64(0))
	})

	t.Run("zero IP addresses", func(t *testing.T) {
		ipv4 := net.ParseIP("0.0.0.0")
		ipv6 := net.ParseIP("::")

		assert.Equal(t, uint64(0), ipToInt(ipv4))
		assert.Equal(t, uint64(0), ipToInt(ipv6))
	})
}

// Helper functions

func parseCIDRs(t *testing.T, cidrs []string) []net.IPNet {
	t.Helper()
	nets := make([]net.IPNet, 0, len(cidrs))
	for _, cidr := range cidrs {
		_, ipNet, err := net.ParseCIDR(cidr)
		require.NoError(t, err, "failed to parse CIDR: %s", cidr)
		nets = append(nets, *ipNet)
	}
	return nets
}

func netIPNetsToStrings(nets []net.IPNet) []string {
	result := make([]string, len(nets))
	for i, n := range nets {
		result[i] = n.String()
	}
	return result
}

// Benchmarks

func BenchmarkCIDRMerge_IPv4_19to32(b *testing.B) {
	// Generate all /32 (individual IPs) from 10.0.0.0/19
	// /19 to /32 = 13 bits = 8192 individual IPs
	cidrs := make([]net.IPNet, 8192)
	baseIP := net.IPv4(10, 0, 0, 0).To4()

	for i := 0; i < 8192; i++ {
		ip := make(net.IP, 4)
		copy(ip, baseIP)
		// Spread the 13 bits across the last two octets
		ip[2] = byte(i >> 8)
		ip[3] = byte(i & 0xff)
		cidrs[i] = net.IPNet{
			IP:   ip,
			Mask: net.CIDRMask(32, 32),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := CIDRMerge(cidrs)
		if len(result) != 1 {
			b.Fatalf("expected 1 result, got %d", len(result))
		}
	}
}

func BenchmarkCIDRMerge_IPv6_115to128(b *testing.B) {
	// Generate all /128 (individual IPs) from 2001:db8::/115
	// /115 to /128 = 13 bits = 8192 individual IPs
	cidrs := make([]net.IPNet, 8192)

	for i := 0; i < 8192; i++ {
		ip := net.IP{
			0x20, 0x01, 0x0d, 0xb8, // 2001:db8
			0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00,
			0x00, 0x00,
			byte(i >> 8), byte(i & 0xff), // last 13 bits vary
		}
		cidrs[i] = net.IPNet{
			IP:   ip,
			Mask: net.CIDRMask(128, 128),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := CIDRMerge(cidrs)
		if len(result) != 1 {
			b.Fatalf("expected 1 result, got %d", len(result))
		}
	}
}

func BenchmarkCIDRMerge_IPv4_16to24(b *testing.B) {
	// Generate all /24 subnets from 192.168.0.0/16
	// 256 subnets
	cidrs := make([]net.IPNet, 256)
	for i := 0; i < 256; i++ {
		_, ipNet, _ := net.ParseCIDR(net.IPv4(192, 168, byte(i), 0).String() + "/24")
		cidrs[i] = *ipNet
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := CIDRMerge(cidrs)
		if len(result) != 1 {
			b.Fatalf("expected 1 result, got %d", len(result))
		}
	}
}

func BenchmarkCIDRMerge_IPv6_32to48(b *testing.B) {
	// Generate all /48 subnets from 2001:db8::/32
	// 65536 subnets
	cidrs := make([]net.IPNet, 65536)
	for i := 0; i < 65536; i++ {
		ip := net.IP{
			0x20, 0x01, 0x0d, 0xb8,
			byte(i >> 8), byte(i & 0xff), 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		}
		cidrs[i] = net.IPNet{
			IP:   ip,
			Mask: net.CIDRMask(48, 128),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := CIDRMerge(cidrs)
		if len(result) != 1 {
			b.Fatalf("expected 1 result, got %d", len(result))
		}
	}
}

func BenchmarkCIDRMergeString_Small(b *testing.B) {
	// Small benchmark with 8 CIDRs that merge into 1
	cidrs := []string{
		"10.0.0.0/24",
		"10.0.1.0/24",
		"10.0.2.0/24",
		"10.0.3.0/24",
		"10.0.4.0/24",
		"10.0.5.0/24",
		"10.0.6.0/24",
		"10.0.7.0/24",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := CIDRMergeString(cidrs)
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 1 {
			b.Fatalf("expected 1 result, got %d", len(result))
		}
	}
}

func BenchmarkCIDRMerge_WithGaps(b *testing.B) {
	// Benchmark with gaps - should result in 2 blocks
	cidrs := make([]net.IPNet, 128)
	// First 64 /24s
	for i := 0; i < 64; i++ {
		_, ipNet, _ := net.ParseCIDR(net.IPv4(192, 168, byte(i), 0).String() + "/24")
		cidrs[i] = *ipNet
	}
	// Skip 64-127, add 128-191
	for i := 128; i < 192; i++ {
		_, ipNet, _ := net.ParseCIDR(net.IPv4(192, 168, byte(i), 0).String() + "/24")
		cidrs[i-64] = *ipNet
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := CIDRMerge(cidrs)
		if len(result) != 2 {
			b.Fatalf("expected 2 results, got %d", len(result))
		}
	}
}
