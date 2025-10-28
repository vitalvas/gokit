package xnet

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCIDRMatcher(t *testing.T) {
	t.Run("empty matcher", func(t *testing.T) {
		matcher := NewCIDRMatcher([]net.IPNet{})
		assert.False(t, matcher.Contains(net.IPv4(192, 168, 1, 1)))
	})

	t.Run("single IPv4 network", func(t *testing.T) {
		_, network, _ := net.ParseCIDR("192.168.1.0/24")
		matcher := NewCIDRMatcher([]net.IPNet{*network})

		assert.True(t, matcher.Contains(net.IPv4(192, 168, 1, 1)))
		assert.True(t, matcher.Contains(net.IPv4(192, 168, 1, 255)))
		assert.False(t, matcher.Contains(net.IPv4(192, 168, 2, 1)))
		assert.False(t, matcher.Contains(net.IPv4(10, 0, 0, 1)))
	})

	t.Run("multiple IPv4 networks", func(t *testing.T) {
		nets := []net.IPNet{}
		_, n1, _ := net.ParseCIDR("192.168.1.0/24")
		_, n2, _ := net.ParseCIDR("10.0.0.0/8")
		_, n3, _ := net.ParseCIDR("172.16.0.0/12")
		nets = append(nets, *n1, *n2, *n3)

		matcher := NewCIDRMatcher(nets)

		// Test matches
		assert.True(t, matcher.Contains(net.IPv4(192, 168, 1, 50)))
		assert.True(t, matcher.Contains(net.IPv4(10, 1, 2, 3)))
		assert.True(t, matcher.Contains(net.IPv4(172, 16, 0, 1)))
		assert.True(t, matcher.Contains(net.IPv4(172, 31, 255, 255)))

		// Test non-matches
		assert.False(t, matcher.Contains(net.IPv4(192, 168, 2, 1)))
		assert.False(t, matcher.Contains(net.IPv4(11, 0, 0, 1)))
		assert.False(t, matcher.Contains(net.IPv4(172, 32, 0, 1)))
	})

	t.Run("overlapping networks", func(t *testing.T) {
		nets := []net.IPNet{}
		_, n1, _ := net.ParseCIDR("10.0.0.0/8")
		_, n2, _ := net.ParseCIDR("10.1.0.0/16")
		nets = append(nets, *n1, *n2)

		matcher := NewCIDRMatcher(nets)

		// Both should match IPs in 10.1.0.0/16
		assert.True(t, matcher.Contains(net.IPv4(10, 1, 2, 3)))
		// Only /8 should match this
		assert.True(t, matcher.Contains(net.IPv4(10, 2, 3, 4)))
	})

	t.Run("IPv6 single network", func(t *testing.T) {
		_, network, _ := net.ParseCIDR("2001:db8::/32")
		matcher := NewCIDRMatcher([]net.IPNet{*network})

		assert.True(t, matcher.Contains(net.ParseIP("2001:db8::1")))
		assert.True(t, matcher.Contains(net.ParseIP("2001:db8:ffff::1")))
		assert.False(t, matcher.Contains(net.ParseIP("2001:db9::1")))
		assert.False(t, matcher.Contains(net.ParseIP("2002::1")))
	})

	t.Run("mixed IPv4 and IPv6", func(t *testing.T) {
		nets := []net.IPNet{}
		_, n1, _ := net.ParseCIDR("192.168.0.0/16")
		_, n2, _ := net.ParseCIDR("2001:db8::/32")
		nets = append(nets, *n1, *n2)

		matcher := NewCIDRMatcher(nets)

		// IPv4 matches
		assert.True(t, matcher.Contains(net.IPv4(192, 168, 1, 1)))
		assert.False(t, matcher.Contains(net.IPv4(192, 169, 1, 1)))

		// IPv6 matches
		assert.True(t, matcher.Contains(net.ParseIP("2001:db8::1")))
		assert.False(t, matcher.Contains(net.ParseIP("2002::1")))
	})

	t.Run("host routes /32 and /128", func(t *testing.T) {
		nets := []net.IPNet{}
		_, n1, _ := net.ParseCIDR("192.168.1.100/32")
		_, n2, _ := net.ParseCIDR("2001:db8::1/128")
		nets = append(nets, *n1, *n2)

		matcher := NewCIDRMatcher(nets)

		// Exact IPv4 match
		assert.True(t, matcher.Contains(net.IPv4(192, 168, 1, 100)))
		assert.False(t, matcher.Contains(net.IPv4(192, 168, 1, 101)))

		// Exact IPv6 match
		assert.True(t, matcher.Contains(net.ParseIP("2001:db8::1")))
		assert.False(t, matcher.Contains(net.ParseIP("2001:db8::2")))
	})

	t.Run("add networks incrementally", func(t *testing.T) {
		matcher := NewCIDRMatcher([]net.IPNet{})

		// Initially empty
		assert.False(t, matcher.Contains(net.IPv4(192, 168, 1, 1)))

		// Add first network
		_, n1, _ := net.ParseCIDR("192.168.1.0/24")
		matcher.Add(*n1)
		assert.True(t, matcher.Contains(net.IPv4(192, 168, 1, 1)))
		assert.False(t, matcher.Contains(net.IPv4(10, 0, 0, 1)))

		// Add second network
		_, n2, _ := net.ParseCIDR("10.0.0.0/8")
		matcher.Add(*n2)
		assert.True(t, matcher.Contains(net.IPv4(192, 168, 1, 1)))
		assert.True(t, matcher.Contains(net.IPv4(10, 0, 0, 1)))
	})
}

func TestNewCIDRMatcherFromStrings(t *testing.T) {
	t.Run("valid CIDRs", func(t *testing.T) {
		cidrs := []string{
			"192.168.1.0/24",
			"10.0.0.0/8",
			"2001:db8::/32",
		}

		matcher, err := NewCIDRMatcherFromStrings(cidrs)
		require.NoError(t, err)

		assert.True(t, matcher.Contains(net.IPv4(192, 168, 1, 1)))
		assert.True(t, matcher.Contains(net.IPv4(10, 1, 2, 3)))
		assert.True(t, matcher.Contains(net.ParseIP("2001:db8::1")))
		assert.False(t, matcher.Contains(net.IPv4(172, 16, 0, 1)))
	})

	t.Run("error on invalid CIDRs", func(t *testing.T) {
		cidrs := []string{
			"192.168.1.0/24",
			"invalid-cidr",
			"10.0.0.0/8",
			"not-a-network",
		}

		matcher, err := NewCIDRMatcherFromStrings(cidrs)
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidCIDR, err)
		assert.Nil(t, matcher)
	})

	t.Run("all invalid CIDRs", func(t *testing.T) {
		cidrs := []string{
			"invalid1",
			"invalid2",
		}

		matcher, err := NewCIDRMatcherFromStrings(cidrs)
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidCIDR, err)
		assert.Nil(t, matcher)
	})
}

func TestGetBit(t *testing.T) {
	t.Run("IPv4 bits", func(t *testing.T) {
		ip := net.IPv4(192, 168, 1, 100).To4() // 11000000.10101000.00000001.01100100

		// First byte: 192 = 11000000
		assert.Equal(t, byte(1), getBit(ip, 0))
		assert.Equal(t, byte(1), getBit(ip, 1))
		assert.Equal(t, byte(0), getBit(ip, 2))

		// Second byte: 168 = 10101000
		assert.Equal(t, byte(1), getBit(ip, 8))
		assert.Equal(t, byte(0), getBit(ip, 9))
		assert.Equal(t, byte(1), getBit(ip, 10))
	})

	t.Run("IPv6 bits", func(t *testing.T) {
		ip := net.ParseIP("2001:db8::1") // First 16 bits: 0010000000000001

		assert.Equal(t, byte(0), getBit(ip, 0))
		assert.Equal(t, byte(0), getBit(ip, 1))
		assert.Equal(t, byte(1), getBit(ip, 2))
		assert.Equal(t, byte(0), getBit(ip, 3))
	})
}

// Benchmarks

func BenchmarkCIDRMatcher_Build_10(b *testing.B) {
	nets := make([]net.IPNet, 10)
	for i := 0; i < 10; i++ {
		_, ipNet, _ := net.ParseCIDR(net.IPv4(10, byte(i), 0, 0).String() + "/24")
		nets[i] = *ipNet
	}

	b.ResetTimer()
	for b.Loop() {
		_ = NewCIDRMatcher(nets)
	}
}

func BenchmarkCIDRMatcher_Build_100(b *testing.B) {
	nets := make([]net.IPNet, 100)
	for i := 0; i < 100; i++ {
		_, ipNet, _ := net.ParseCIDR(net.IPv4(10, byte(i), 0, 0).String() + "/24")
		nets[i] = *ipNet
	}

	b.ResetTimer()
	for b.Loop() {
		_ = NewCIDRMatcher(nets)
	}
}

func BenchmarkCIDRMatcher_Build_1000(b *testing.B) {
	nets := make([]net.IPNet, 1000)
	for i := 0; i < 1000; i++ {
		octet2 := byte(i / 256)
		octet3 := byte(i % 256)
		_, ipNet, _ := net.ParseCIDR(net.IPv4(10, octet2, octet3, 0).String() + "/24")
		nets[i] = *ipNet
	}

	b.ResetTimer()
	for b.Loop() {
		_ = NewCIDRMatcher(nets)
	}
}

func BenchmarkCIDRMatcher_Contains_10(b *testing.B) {
	nets := make([]net.IPNet, 10)
	for i := 0; i < 10; i++ {
		_, ipNet, _ := net.ParseCIDR(net.IPv4(10, byte(i), 0, 0).String() + "/24")
		nets[i] = *ipNet
	}
	matcher := NewCIDRMatcher(nets)
	ip := net.IPv4(10, 5, 1, 1)

	b.ResetTimer()
	for b.Loop() {
		_ = matcher.Contains(ip)
	}
}

func BenchmarkCIDRMatcher_Contains_100(b *testing.B) {
	nets := make([]net.IPNet, 100)
	for i := 0; i < 100; i++ {
		_, ipNet, _ := net.ParseCIDR(net.IPv4(10, byte(i), 0, 0).String() + "/24")
		nets[i] = *ipNet
	}
	matcher := NewCIDRMatcher(nets)
	ip := net.IPv4(10, 50, 1, 1)

	b.ResetTimer()
	for b.Loop() {
		_ = matcher.Contains(ip)
	}
}

func BenchmarkCIDRMatcher_Contains_1000(b *testing.B) {
	nets := make([]net.IPNet, 1000)
	for i := 0; i < 1000; i++ {
		octet2 := byte(i / 256)
		octet3 := byte(i % 256)
		_, ipNet, _ := net.ParseCIDR(net.IPv4(10, octet2, octet3, 0).String() + "/24")
		nets[i] = *ipNet
	}
	matcher := NewCIDRMatcher(nets)
	ip := net.IPv4(10, 1, 244, 1)

	b.ResetTimer()
	for b.Loop() {
		_ = matcher.Contains(ip)
	}
}
