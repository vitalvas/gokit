package xnet

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCIDRSplit(t *testing.T) {
	t.Run("split /16 to /24", func(t *testing.T) {
		_, network, err := net.ParseCIDR("192.168.0.0/16")
		require.NoError(t, err)

		result, err := CIDRSplit(*network, 24)
		require.NoError(t, err)
		require.Len(t, result, 256)

		// Check first subnet
		assert.Equal(t, "192.168.0.0/24", result[0].String())
		// Check last subnet
		assert.Equal(t, "192.168.255.0/24", result[255].String())
		// Check some middle subnets
		assert.Equal(t, "192.168.1.0/24", result[1].String())
		assert.Equal(t, "192.168.128.0/24", result[128].String())
	})

	t.Run("split /24 to /25", func(t *testing.T) {
		_, network, err := net.ParseCIDR("10.0.0.0/24")
		require.NoError(t, err)

		result, err := CIDRSplit(*network, 25)
		require.NoError(t, err)
		require.Len(t, result, 2)

		assert.Equal(t, "10.0.0.0/25", result[0].String())
		assert.Equal(t, "10.0.0.128/25", result[1].String())
	})

	t.Run("split /24 to /26", func(t *testing.T) {
		_, network, err := net.ParseCIDR("10.0.0.0/24")
		require.NoError(t, err)

		result, err := CIDRSplit(*network, 26)
		require.NoError(t, err)
		require.Len(t, result, 4)

		assert.Equal(t, "10.0.0.0/26", result[0].String())
		assert.Equal(t, "10.0.0.64/26", result[1].String())
		assert.Equal(t, "10.0.0.128/26", result[2].String())
		assert.Equal(t, "10.0.0.192/26", result[3].String())
	})

	t.Run("split /19 to /32", func(t *testing.T) {
		_, network, err := net.ParseCIDR("10.0.0.0/19")
		require.NoError(t, err)

		result, err := CIDRSplit(*network, 32)
		require.NoError(t, err)
		require.Len(t, result, 8192)

		// Check first IP
		assert.Equal(t, "10.0.0.0/32", result[0].String())
		// Check last IP
		assert.Equal(t, "10.0.31.255/32", result[8191].String())
	})

	t.Run("IPv6 split /32 to /48", func(t *testing.T) {
		_, network, err := net.ParseCIDR("2001:db8::/32")
		require.NoError(t, err)

		result, err := CIDRSplit(*network, 48)
		require.NoError(t, err)
		require.Len(t, result, 65536)

		// Check first subnet
		assert.Equal(t, "2001:db8::/48", result[0].String())
		// Check last subnet
		assert.Equal(t, "2001:db8:ffff::/48", result[65535].String())
		// Check some middle subnets
		assert.Equal(t, "2001:db8:1::/48", result[1].String())
		assert.Equal(t, "2001:db8:100::/48", result[256].String())
	})

	t.Run("IPv6 split /48 to /64", func(t *testing.T) {
		_, network, err := net.ParseCIDR("2001:db8:1::/48")
		require.NoError(t, err)

		result, err := CIDRSplit(*network, 64)
		require.NoError(t, err)
		require.Len(t, result, 65536)

		// Check first subnet
		assert.Equal(t, "2001:db8:1::/64", result[0].String())
		// Check last subnet
		assert.Equal(t, "2001:db8:1:ffff::/64", result[65535].String())
	})

	t.Run("same prefix returns original", func(t *testing.T) {
		_, network, err := net.ParseCIDR("192.168.1.0/24")
		require.NoError(t, err)

		result, err := CIDRSplit(*network, 24)
		require.NoError(t, err)
		require.Len(t, result, 1)
		assert.Equal(t, "192.168.1.0/24", result[0].String())
	})

	t.Run("error on larger prefix", func(t *testing.T) {
		_, network, err := net.ParseCIDR("192.168.1.0/24")
		require.NoError(t, err)

		result, err := CIDRSplit(*network, 16)
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidPrefixSize, err)
		assert.Nil(t, result)
	})

	t.Run("error on prefix larger than max", func(t *testing.T) {
		_, network, err := net.ParseCIDR("192.168.1.0/24")
		require.NoError(t, err)

		result, err := CIDRSplit(*network, 33)
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidPrefixSize, err)
		assert.Nil(t, result)
	})

	t.Run("error on IPv6 prefix larger than max", func(t *testing.T) {
		_, network, err := net.ParseCIDR("2001:db8::/32")
		require.NoError(t, err)

		result, err := CIDRSplit(*network, 129)
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidPrefixSize, err)
		assert.Nil(t, result)
	})

	t.Run("split /8 to /16", func(t *testing.T) {
		_, network, err := net.ParseCIDR("10.0.0.0/8")
		require.NoError(t, err)

		result, err := CIDRSplit(*network, 16)
		require.NoError(t, err)
		require.Len(t, result, 256)

		// Check coverage
		assert.Equal(t, "10.0.0.0/16", result[0].String())
		assert.Equal(t, "10.1.0.0/16", result[1].String())
		assert.Equal(t, "10.255.0.0/16", result[255].String())
	})

	t.Run("split /16 to /30", func(t *testing.T) {
		_, network, err := net.ParseCIDR("192.168.0.0/16")
		require.NoError(t, err)

		result, err := CIDRSplit(*network, 30)
		require.NoError(t, err)
		// /16 to /30 = 2^(30-16) = 2^14 = 16384 subnets
		require.Len(t, result, 16384)

		// Check first subnet
		assert.Equal(t, "192.168.0.0/30", result[0].String())
		// Check second subnet (4 IPs per /30)
		assert.Equal(t, "192.168.0.4/30", result[1].String())
		// Check last subnet
		assert.Equal(t, "192.168.255.252/30", result[16383].String())
	})

	t.Run("IPv6 split /56 to /64", func(t *testing.T) {
		_, network, err := net.ParseCIDR("2001:db8:1::/56")
		require.NoError(t, err)

		result, err := CIDRSplit(*network, 64)
		require.NoError(t, err)
		// /56 to /64 = 2^(64-56) = 2^8 = 256 subnets
		require.Len(t, result, 256)

		// Check first subnet
		assert.Equal(t, "2001:db8:1::/64", result[0].String())
		// Check second subnet
		assert.Equal(t, "2001:db8:1:1::/64", result[1].String())
		// Check last subnet
		assert.Equal(t, "2001:db8:1:ff::/64", result[255].String())
	})
}

func TestCIDRSplitString(t *testing.T) {
	t.Run("valid split", func(t *testing.T) {
		result, err := CIDRSplitString("192.168.0.0/24", 26)
		require.NoError(t, err)
		require.Len(t, result, 4)

		assert.Equal(t, "192.168.0.0/26", result[0])
		assert.Equal(t, "192.168.0.64/26", result[1])
		assert.Equal(t, "192.168.0.128/26", result[2])
		assert.Equal(t, "192.168.0.192/26", result[3])
	})

	t.Run("invalid CIDR", func(t *testing.T) {
		result, err := CIDRSplitString("invalid-cidr", 24)
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidCIDR, err)
		assert.Nil(t, result)
	})

	t.Run("invalid prefix size", func(t *testing.T) {
		result, err := CIDRSplitString("192.168.0.0/24", 16)
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidPrefixSize, err)
		assert.Nil(t, result)
	})

	t.Run("IPv6 split", func(t *testing.T) {
		result, err := CIDRSplitString("2001:db8::/32", 40)
		require.NoError(t, err)
		require.Len(t, result, 256)

		assert.Equal(t, "2001:db8::/40", result[0])
		assert.Equal(t, "2001:db8:ff00::/40", result[255])
	})
}

func TestCIDRSplitAndMerge(t *testing.T) {
	t.Run("split and merge IPv4 /16 to /24", func(t *testing.T) {
		_, network, err := net.ParseCIDR("192.168.0.0/16")
		require.NoError(t, err)

		// Split into /24 subnets
		subnets, err := CIDRSplit(*network, 24)
		require.NoError(t, err)
		require.Len(t, subnets, 256)

		// Merge back
		merged := CIDRMerge(subnets)
		require.Len(t, merged, 1)
		assert.Equal(t, "192.168.0.0/16", merged[0].String())
	})

	t.Run("split and merge IPv6 /32 to /40", func(t *testing.T) {
		_, network, err := net.ParseCIDR("2001:db8::/32")
		require.NoError(t, err)

		// Split into /40 subnets
		subnets, err := CIDRSplit(*network, 40)
		require.NoError(t, err)
		require.Len(t, subnets, 256)

		// Merge back
		merged := CIDRMerge(subnets)
		require.Len(t, merged, 1)
		assert.Equal(t, "2001:db8::/32", merged[0].String())
	})

	t.Run("split and merge IPv4 /19 to /32", func(t *testing.T) {
		_, network, err := net.ParseCIDR("10.0.0.0/19")
		require.NoError(t, err)

		// Split into /32 (individual IPs)
		subnets, err := CIDRSplit(*network, 32)
		require.NoError(t, err)
		require.Len(t, subnets, 8192)

		// Merge back
		merged := CIDRMerge(subnets)
		require.Len(t, merged, 1)
		assert.Equal(t, "10.0.0.0/19", merged[0].String())
	})
}

func TestAddToIP(t *testing.T) {
	t.Run("IPv4 add offset", func(t *testing.T) {
		ip := net.IPv4(192, 168, 0, 0).To4()
		addToIP(ip, 1, 24, 26)
		assert.Equal(t, "192.168.0.64", ip.String())
	})

	t.Run("IPv4 add larger offset", func(t *testing.T) {
		ip := net.IPv4(192, 168, 0, 0).To4()
		addToIP(ip, 255, 16, 24)
		assert.Equal(t, "192.168.255.0", ip.String())
	})

	t.Run("IPv6 add offset", func(t *testing.T) {
		ip := net.ParseIP("2001:db8::")
		addToIP(ip, 1, 32, 48)
		assert.Equal(t, "2001:db8:1::", ip.String())
	})
}

func TestSetBit(t *testing.T) {
	t.Run("set bit in IPv4", func(t *testing.T) {
		ip := net.IPv4(0, 0, 0, 0).To4()
		setBit(ip, 0) // Set first bit
		assert.Equal(t, byte(128), ip[0])
	})

	t.Run("set multiple bits in IPv4", func(t *testing.T) {
		ip := net.IPv4(0, 0, 0, 0).To4()
		setBit(ip, 8)  // Set first bit of second byte
		setBit(ip, 16) // Set first bit of third byte
		assert.Equal(t, byte(128), ip[1])
		assert.Equal(t, byte(128), ip[2])
	})
}

// Benchmarks

func BenchmarkCIDRSplit_IPv4_16to24(b *testing.B) {
	_, network, _ := net.ParseCIDR("192.168.0.0/16")

	b.ResetTimer()
	for b.Loop() {
		result, err := CIDRSplit(*network, 24)
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 256 {
			b.Fatalf("expected 256 results, got %d", len(result))
		}
	}
}

func BenchmarkCIDRSplit_IPv4_19to32(b *testing.B) {
	_, network, _ := net.ParseCIDR("10.0.0.0/19")

	b.ResetTimer()
	for b.Loop() {
		result, err := CIDRSplit(*network, 32)
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 8192 {
			b.Fatalf("expected 8192 results, got %d", len(result))
		}
	}
}

func BenchmarkCIDRSplit_IPv6_32to48(b *testing.B) {
	_, network, _ := net.ParseCIDR("2001:db8::/32")

	b.ResetTimer()
	for b.Loop() {
		result, err := CIDRSplit(*network, 48)
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 65536 {
			b.Fatalf("expected 65536 results, got %d", len(result))
		}
	}
}

func BenchmarkCIDRSplit_IPv6_115to128(b *testing.B) {
	_, network, _ := net.ParseCIDR("2001:db8::/115")

	b.ResetTimer()
	for b.Loop() {
		result, err := CIDRSplit(*network, 128)
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 8192 {
			b.Fatalf("expected 8192 results, got %d", len(result))
		}
	}
}

func BenchmarkCIDRSplitString_Small(b *testing.B) {
	b.ResetTimer()
	for b.Loop() {
		result, err := CIDRSplitString("10.0.0.0/24", 26)
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 4 {
			b.Fatalf("expected 4 results, got %d", len(result))
		}
	}
}

func BenchmarkCIDRSplit_IPv4_8to16(b *testing.B) {
	_, network, _ := net.ParseCIDR("10.0.0.0/8")

	b.ResetTimer()
	for b.Loop() {
		result, err := CIDRSplit(*network, 16)
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 256 {
			b.Fatalf("expected 256 results, got %d", len(result))
		}
	}
}
