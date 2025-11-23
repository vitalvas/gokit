package wirefilter

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStringValue(t *testing.T) {
	t.Run("type and string", func(t *testing.T) {
		sv := StringValue("test")
		assert.Equal(t, TypeString, sv.Type())
		assert.Equal(t, "test", sv.String())
	})

	t.Run("equality", func(t *testing.T) {
		sv1 := StringValue("test")
		sv2 := StringValue("test")
		sv3 := StringValue("different")
		assert.True(t, sv1.Equal(sv2))
		assert.False(t, sv1.Equal(sv3))
	})
}

func TestIntValue(t *testing.T) {
	t.Run("type and string", func(t *testing.T) {
		iv := IntValue(42)
		assert.Equal(t, TypeInt, iv.Type())
		assert.Equal(t, "42", iv.String())
	})

	t.Run("equality", func(t *testing.T) {
		iv1 := IntValue(42)
		iv2 := IntValue(42)
		iv3 := IntValue(100)
		assert.True(t, iv1.Equal(iv2))
		assert.False(t, iv1.Equal(iv3))
	})
}

func TestBoolValue(t *testing.T) {
	t.Run("type and string", func(t *testing.T) {
		bv := BoolValue(true)
		assert.Equal(t, TypeBool, bv.Type())
		assert.Equal(t, "true", bv.String())
	})

	t.Run("equality", func(t *testing.T) {
		bv1 := BoolValue(true)
		bv2 := BoolValue(true)
		bv3 := BoolValue(false)
		assert.True(t, bv1.Equal(bv2))
		assert.False(t, bv1.Equal(bv3))
	})
}

func TestIPValue(t *testing.T) {
	t.Run("type and string", func(t *testing.T) {
		ip := IPValue{IP: net.ParseIP("192.168.1.1")}
		assert.Equal(t, TypeIP, ip.Type())
		assert.Equal(t, "192.168.1.1", ip.String())
	})

	t.Run("equality", func(t *testing.T) {
		ip1 := IPValue{IP: net.ParseIP("192.168.1.1")}
		ip2 := IPValue{IP: net.ParseIP("192.168.1.1")}
		ip3 := IPValue{IP: net.ParseIP("10.0.0.1")}
		assert.True(t, ip1.Equal(ip2))
		assert.False(t, ip1.Equal(ip3))
	})
}

func TestBytesValue(t *testing.T) {
	t.Run("type and string", func(t *testing.T) {
		bv := BytesValue([]byte("test"))
		assert.Equal(t, TypeBytes, bv.Type())
		assert.Equal(t, "test", bv.String())
	})

	t.Run("equality", func(t *testing.T) {
		bv1 := BytesValue([]byte("test"))
		bv2 := BytesValue([]byte("test"))
		bv3 := BytesValue([]byte("different"))
		assert.True(t, bv1.Equal(bv2))
		assert.False(t, bv1.Equal(bv3))
	})
}

func TestArrayValue(t *testing.T) {
	t.Run("type and string", func(t *testing.T) {
		av := ArrayValue([]Value{IntValue(1), IntValue(2), IntValue(3)})
		assert.Equal(t, TypeArray, av.Type())
		assert.Equal(t, "[1, 2, 3]", av.String())
	})

	t.Run("equality", func(t *testing.T) {
		av1 := ArrayValue([]Value{IntValue(1), IntValue(2)})
		av2 := ArrayValue([]Value{IntValue(1), IntValue(2)})
		av3 := ArrayValue([]Value{IntValue(1), IntValue(3)})
		assert.True(t, av1.Equal(av2))
		assert.False(t, av1.Equal(av3))
	})

	t.Run("contains", func(t *testing.T) {
		av := ArrayValue([]Value{IntValue(1), IntValue(2), IntValue(3)})
		assert.True(t, av.Contains(IntValue(2)))
		assert.False(t, av.Contains(IntValue(5)))
	})
}

func TestIPInCIDR(t *testing.T) {
	t.Run("ip in cidr", func(t *testing.T) {
		ip := net.ParseIP("192.168.1.100")
		result, err := IPInCIDR(ip, "192.168.1.0/24")
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("ip not in cidr", func(t *testing.T) {
		ip := net.ParseIP("10.0.0.1")
		result, err := IPInCIDR(ip, "192.168.1.0/24")
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("invalid cidr", func(t *testing.T) {
		ip := net.ParseIP("192.168.1.1")
		_, err := IPInCIDR(ip, "invalid")
		assert.Error(t, err)
	})
}

func TestMatchesRegex(t *testing.T) {
	t.Run("matches pattern", func(t *testing.T) {
		result, err := MatchesRegex("hello world", "^hello.*")
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("does not match pattern", func(t *testing.T) {
		result, err := MatchesRegex("hello world", "^goodbye.*")
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("invalid regex", func(t *testing.T) {
		_, err := MatchesRegex("test", "[invalid")
		assert.Error(t, err)
	})
}

func TestContainsString(t *testing.T) {
	t.Run("contains substring", func(t *testing.T) {
		result := ContainsString("hello world", "world")
		assert.True(t, result)
	})

	t.Run("does not contain substring", func(t *testing.T) {
		result := ContainsString("hello world", "goodbye")
		assert.False(t, result)
	})
}

func TestIPv6Support(t *testing.T) {
	t.Run("parse ipv6 address", func(t *testing.T) {
		ip := IPValue{IP: net.ParseIP("2001:db8::1")}
		assert.Equal(t, TypeIP, ip.Type())
		assert.NotNil(t, ip.IP)
	})

	t.Run("ipv6 equality", func(t *testing.T) {
		ip1 := IPValue{IP: net.ParseIP("2001:db8::1")}
		ip2 := IPValue{IP: net.ParseIP("2001:db8::1")}
		ip3 := IPValue{IP: net.ParseIP("2001:db8::2")}
		assert.True(t, ip1.Equal(ip2))
		assert.False(t, ip1.Equal(ip3))
	})

	t.Run("ipv6 in cidr", func(t *testing.T) {
		ip := net.ParseIP("2001:db8::1")
		result, err := IPInCIDR(ip, "2001:db8::/32")
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("ipv6 not in cidr", func(t *testing.T) {
		ip := net.ParseIP("2001:db9::1")
		result, err := IPInCIDR(ip, "2001:db8::/32")
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("is ipv6", func(t *testing.T) {
		ipv6 := net.ParseIP("2001:db8::1")
		ipv4 := net.ParseIP("192.168.1.1")
		assert.True(t, IsIPv6(ipv6))
		assert.False(t, IsIPv6(ipv4))
	})

	t.Run("is ipv4", func(t *testing.T) {
		ipv4 := net.ParseIP("192.168.1.1")
		ipv6 := net.ParseIP("2001:db8::1")
		assert.True(t, IsIPv4(ipv4))
		assert.False(t, IsIPv4(ipv6))
	})

	t.Run("ipv6 loopback", func(t *testing.T) {
		ip := net.ParseIP("::1")
		assert.NotNil(t, ip)
		assert.True(t, IsIPv6(ip))
	})

	t.Run("ipv6 mixed notation", func(t *testing.T) {
		ip := net.ParseIP("::ffff:192.168.1.1")
		assert.NotNil(t, ip)
	})
}
