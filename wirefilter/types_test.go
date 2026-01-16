package wirefilter

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func BenchmarkValueOperations(b *testing.B) {
	b.Run("string equality", func(b *testing.B) {
		v1 := StringValue("example.com")
		v2 := StringValue("example.com")
		b.ReportAllocs()
		for b.Loop() {
			v1.Equal(v2)
		}
	})

	b.Run("int comparison", func(b *testing.B) {
		v1 := IntValue(200)
		v2 := IntValue(200)
		b.ReportAllocs()
		for b.Loop() {
			v1.Equal(v2)
		}
	})

	b.Run("ip equality", func(b *testing.B) {
		v1 := IPValue{IP: []byte{192, 168, 1, 1}}
		v2 := IPValue{IP: []byte{192, 168, 1, 1}}
		b.ReportAllocs()
		for b.Loop() {
			v1.Equal(v2)
		}
	})

	b.Run("array contains", func(b *testing.B) {
		arr := ArrayValue{IntValue(1), IntValue(2), IntValue(3), IntValue(4), IntValue(5)}
		val := IntValue(3)
		b.ReportAllocs()
		for b.Loop() {
			arr.Contains(val)
		}
	})
}

func BenchmarkIPOperations(b *testing.B) {
	b.Run("ipv4 in cidr", func(b *testing.B) {
		ip := []byte{192, 168, 1, 1}
		cidr := "192.168.0.0/16"
		b.ReportAllocs()
		for b.Loop() {
			_, err := IPInCIDR(ip, cidr)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("ipv6 in cidr", func(b *testing.B) {
		ip := []byte{0x20, 0x01, 0x0d, 0xb8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}
		cidr := "2001:db8::/32"
		b.ReportAllocs()
		for b.Loop() {
			_, err := IPInCIDR(ip, cidr)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkStringOperations(b *testing.B) {
	b.Run("contains", func(b *testing.B) {
		haystack := "this is a long string that contains some text"
		needle := "contains"
		b.ReportAllocs()
		for b.Loop() {
			ContainsString(haystack, needle)
		}
	})

	b.Run("regex match", func(b *testing.B) {
		value := "example.com"
		pattern := "^example\\..*"
		b.ReportAllocs()
		for b.Loop() {
			_, err := MatchesRegex(value, pattern)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func FuzzIPInCIDR(f *testing.F) {
	f.Add([]byte{192, 168, 1, 1}, "192.168.0.0/16")
	f.Add([]byte{10, 0, 0, 1}, "10.0.0.0/8")
	f.Add([]byte{172, 16, 0, 1}, "172.16.0.0/12")
	f.Add([]byte{8, 8, 8, 8}, "8.8.8.0/24")
	f.Add([]byte{0x20, 0x01, 0x0d, 0xb8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}, "2001:db8::/32")

	f.Fuzz(func(_ *testing.T, ipBytes []byte, cidr string) {
		if len(ipBytes) != 4 && len(ipBytes) != 16 {
			return
		}
		_, _ = IPInCIDR(ipBytes, cidr)
	})
}

func FuzzMatchesRegex(f *testing.F) {
	f.Add("example.com", "^example\\..*")
	f.Add("test123", "[a-z]+[0-9]+")
	f.Add("/api/v1/users", "^/api/v[0-9]+/")
	f.Add("hello world", "\\bworld\\b")
	f.Add("abc", ".*")

	f.Fuzz(func(_ *testing.T, value, pattern string) {
		_, _ = MatchesRegex(value, pattern)
	})
}

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

func TestValueTypeMismatch(t *testing.T) {
	t.Run("string equal to non-string", func(t *testing.T) {
		sv := StringValue("test")
		iv := IntValue(42)
		assert.False(t, sv.Equal(iv))
	})

	t.Run("int equal to non-int", func(t *testing.T) {
		iv := IntValue(42)
		sv := StringValue("42")
		assert.False(t, iv.Equal(sv))
	})

	t.Run("bool equal to non-bool", func(t *testing.T) {
		bv := BoolValue(true)
		sv := StringValue("true")
		assert.False(t, bv.Equal(sv))
	})

	t.Run("ip equal to non-ip", func(t *testing.T) {
		ip := IPValue{IP: net.ParseIP("192.168.1.1")}
		sv := StringValue("192.168.1.1")
		assert.False(t, ip.Equal(sv))
	})

	t.Run("bytes equal to non-bytes", func(t *testing.T) {
		bv := BytesValue([]byte("test"))
		sv := StringValue("test")
		assert.False(t, bv.Equal(sv))
	})

	t.Run("array equal to non-array", func(t *testing.T) {
		av := ArrayValue{IntValue(1), IntValue(2)}
		iv := IntValue(1)
		assert.False(t, av.Equal(iv))
	})

	t.Run("array equal different lengths", func(t *testing.T) {
		av1 := ArrayValue{IntValue(1), IntValue(2)}
		av2 := ArrayValue{IntValue(1)}
		assert.False(t, av1.Equal(av2))
	})

	t.Run("bytes equal different lengths", func(t *testing.T) {
		bv1 := BytesValue([]byte("test"))
		bv2 := BytesValue([]byte("te"))
		assert.False(t, bv1.Equal(bv2))
	})
}

func TestValueIsTruthy(t *testing.T) {
	t.Run("string is truthy", func(t *testing.T) {
		sv := StringValue("test")
		assert.True(t, sv.IsTruthy())
	})

	t.Run("empty string is truthy", func(t *testing.T) {
		sv := StringValue("")
		assert.True(t, sv.IsTruthy())
	})

	t.Run("int is truthy", func(t *testing.T) {
		iv := IntValue(42)
		assert.True(t, iv.IsTruthy())
	})

	t.Run("zero int is truthy", func(t *testing.T) {
		iv := IntValue(0)
		assert.True(t, iv.IsTruthy())
	})

	t.Run("bool true is truthy", func(t *testing.T) {
		bv := BoolValue(true)
		assert.True(t, bv.IsTruthy())
	})

	t.Run("bool false is not truthy", func(t *testing.T) {
		bv := BoolValue(false)
		assert.False(t, bv.IsTruthy())
	})

	t.Run("ip is truthy", func(t *testing.T) {
		ip := IPValue{IP: net.ParseIP("192.168.1.1")}
		assert.True(t, ip.IsTruthy())
	})

	t.Run("bytes is truthy", func(t *testing.T) {
		bv := BytesValue([]byte("test"))
		assert.True(t, bv.IsTruthy())
	})

	t.Run("empty bytes is truthy", func(t *testing.T) {
		bv := BytesValue([]byte{})
		assert.True(t, bv.IsTruthy())
	})

	t.Run("array is truthy", func(t *testing.T) {
		av := ArrayValue{IntValue(1)}
		assert.True(t, av.IsTruthy())
	})

	t.Run("empty array is truthy", func(t *testing.T) {
		av := ArrayValue{}
		assert.True(t, av.IsTruthy())
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

func TestBytesValueEqualSameLengthDifferentContent(t *testing.T) {
	bv1 := BytesValue([]byte("test"))
	bv2 := BytesValue([]byte("best"))
	assert.False(t, bv1.Equal(bv2))
}
