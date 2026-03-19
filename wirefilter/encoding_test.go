package wirefilter

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarshalUnmarshal(t *testing.T) {
	expressions := []struct {
		name string
		expr string
	}{
		{"simple equality", `name == "test"`},
		{"integer comparison", `status >= 400`},
		{"logical and", `a == 1 and b == 2`},
		{"logical or", `a == 1 or b == 2`},
		{"logical xor", `a xor b`},
		{"not", `not active`},
		{"not in", `ip not in $nets`},
		{"not contains", `name not contains "admin"`},
		{"in array", `status in {200, 201, 204}`},
		{"in range", `port in {80..100}`},
		{"in CIDR string", `ip in "10.0.0.0/8"`},
		{"in CIDR native", `ip in 192.168.0.0/24`},
		{"contains", `path contains "/api"`},
		{"matches", `ua matches "^Mozilla"`},
		{"wildcard", `host wildcard "*.example.com"`},
		{"strict wildcard", `host strict wildcard "*.Example.com"`},
		{"all equal", `tags === "prod"`},
		{"any not equal", `tags !== "dev"`},
		{"field index string", `data["key"] == "val"`},
		{"field index int", `items[0] == "first"`},
		{"array unpack", `tags[*] == "prod"`},
		{"list ref", `ip in $blocked_ips`},
		{"function lower", `lower(name) == "test"`},
		{"function upper", `upper(name) == "TEST"`},
		{"function len", `len(name) > 5`},
		{"function starts_with", `starts_with(name, "pre")`},
		{"function ends_with", `ends_with(name, ".com")`},
		{"function concat", `concat("a", "b") == "ab"`},
		{"function substring", `substring(name, 0, 3) == "tes"`},
		{"function split", `split(name, ",")[0] == "a"`},
		{"function join", `join(tags, ",") == "a,b"`},
		{"function has_key", `has_key(data, "key")`},
		{"function has_value", `has_value(tags, "a")`},
		{"function url_decode", `url_decode(query) contains "test"`},
		{"function cidr", `cidr(ip, 24) == "10.0.0.0"`},
		{"function cidr6", `cidr6(ip, 64) == "2001:db8::"`},
		{"function any", `any(tags[*] == "prod")`},
		{"function all", `all(tags[*] contains "a")`},
		{"complex nested", `(lower(name) == "admin" or status >= 500) and ip not in $blocked`},
		{"bool literal true", `active == true`},
		{"bool literal false", `active == false`},
		{"negative int", `count > -1`},
		{"float literal", `score > 3.14`},
		{"float equality", `score == 99.5`},
		{"negative float", `temp > -10.5`},
		{"float in set", `score in {1.5, 2.5, 3.5}`},
		{"IP literal", `ip == 192.168.1.1`},
		{"empty array", `x in {}`},
		{"mixed array", `port in {80, 443, 8000..9000}`},
		{"table lookup scalar", `$geo[ip] == "US"`},
		{"table lookup with in", `name in $allowed[dept]`},
		{"table lookup literal key", `$config["mode"] == "prod"`},
		{"udf no args", `maintenance() == true`},
		{"udf with arg", `get_score(name) > 5.0`},
		{"udf with ip arg", `is_tor(ip) == true`},
		{"udf in operator", `ip in get_cidrs(name)`},
		{"udf combined", `is_tor(ip) and get_score(name) > 3.0`},
	}

	for _, tt := range expressions {
		t.Run(tt.name, func(t *testing.T) {
			original, err := Compile(tt.expr, nil)
			require.NoError(t, err)

			data, err := original.MarshalBinary()
			require.NoError(t, err)
			assert.True(t, len(data) > 3, "encoded data should have header + body")

			restored := &Filter{}
			err = restored.UnmarshalBinary(data)
			require.NoError(t, err)

			// Verify both filters produce the same results
			ctx := NewExecutionContext().
				SetStringField("name", "test").
				SetStringField("host", "api.example.com").
				SetStringField("path", "/api/v1/users").
				SetStringField("ua", "Mozilla/5.0").
				SetStringField("query", "search%20term").
				SetIntField("status", 500).
				SetIntField("port", 443).
				SetIntField("count", 10).
				SetIntField("x", 201).
				SetFloatField("score", 99.5).
				SetFloatField("temp", -5.0).
				SetBoolField("active", true).
				SetBoolField("a", true).
				SetBoolField("b", false).
				SetIPField("ip", "192.168.1.1").
				SetArrayField("tags", []string{"prod", "v2"}).
				SetArrayField("items", []string{"first", "second"}).
				SetMapField("data", map[string]string{"key": "val"}).
				SetList("names", []string{"admin", "user"}).
				SetIPList("blocked_ips", []string{"10.0.0.1", "192.168.0.0/16"}).
				SetIPList("blocked", []string{"10.0.0.0/8"}).
				SetIPList("nets", []string{"10.0.0.0/8", "172.16.0.0/12"}).
				SetTable("geo", map[string]string{"192.168.1.1": "US"}).
				SetTable("config", map[string]string{"mode": "prod"}).
				SetTableList("allowed", map[string][]string{"eng": {"dev", "sre"}}).
				SetStringField("dept", "eng").
				SetStringField("name", "dev").
				SetFunc("maintenance", func(_ []Value) (Value, error) {
					return BoolValue(true), nil
				}).
				SetFunc("get_score", func(_ []Value) (Value, error) {
					return FloatValue(7.5), nil
				}).
				SetFunc("is_tor", func(_ []Value) (Value, error) {
					return BoolValue(false), nil
				}).
				SetFunc("get_cidrs", func(_ []Value) (Value, error) {
					_, ipNet, _ := net.ParseCIDR("192.168.0.0/16")
					return ArrayValue{CIDRValue{IPNet: ipNet}}, nil
				})

			origResult, origErr := original.Execute(ctx)
			restoredResult, restoredErr := restored.Execute(ctx)

			assert.Equal(t, origErr, restoredErr)
			assert.Equal(t, origResult, restoredResult)
		})
	}
}

func TestMarshalBinaryCompactness(t *testing.T) {
	filter, err := Compile(`name == "test"`, nil)
	require.NoError(t, err)

	data, err := filter.MarshalBinary()
	require.NoError(t, err)

	// Header: "WF" (2) + version (1) = 3
	// BinaryExpr tag (1) + operator (1) = 2
	// FieldExpr tag (1) + name len varint (1) + "name" (4) = 6
	// LiteralExpr tag (1) + string val tag (1) + len varint (1) + "test" (4) = 7
	// Total: 3 + 2 + 6 + 7 = 18
	assert.Equal(t, 18, len(data))
}

func TestUnmarshalBinaryErrors(t *testing.T) {
	t.Run("empty data", func(t *testing.T) {
		f := &Filter{}
		err := f.UnmarshalBinary([]byte{})
		assert.Error(t, err)
	})

	t.Run("invalid magic", func(t *testing.T) {
		f := &Filter{}
		err := f.UnmarshalBinary([]byte("XX\x01"))
		assert.ErrorIs(t, err, errInvalidMagic)
	})

	t.Run("wrong version", func(t *testing.T) {
		f := &Filter{}
		err := f.UnmarshalBinary([]byte("WF\x99"))
		assert.ErrorIs(t, err, errInvalidVersion)
	})

	t.Run("truncated after header", func(t *testing.T) {
		f := &Filter{}
		err := f.UnmarshalBinary([]byte("WF\x01"))
		assert.Error(t, err)
	})

	t.Run("invalid node tag", func(t *testing.T) {
		f := &Filter{}
		err := f.UnmarshalBinary([]byte("WF\x01\xFF"))
		assert.Error(t, err)
	})

	t.Run("invalid value tag", func(t *testing.T) {
		f := &Filter{}
		// LiteralExpr node (0x04) followed by invalid value tag (0xFF)
		err := f.UnmarshalBinary([]byte("WF\x01\x04\xFF"))
		assert.Error(t, err)
	})

	t.Run("truncated string", func(t *testing.T) {
		f := &Filter{}
		// FieldExpr (0x03) + string length 10, but no data
		err := f.UnmarshalBinary([]byte("WF\x01\x03\x0A"))
		assert.Error(t, err)
	})
}

func TestMarshalUnmarshalRoundtrip(t *testing.T) {
	expr := `lower(name) == "admin" and ip not in $blocked and status in {400..599}`

	filter1, err := Compile(expr, nil)
	require.NoError(t, err)

	// Marshal
	data1, err := filter1.MarshalBinary()
	require.NoError(t, err)

	// Unmarshal
	filter2 := &Filter{}
	err = filter2.UnmarshalBinary(data1)
	require.NoError(t, err)

	// Marshal again
	data2, err := filter2.MarshalBinary()
	require.NoError(t, err)

	// Binary output should be identical
	assert.Equal(t, data1, data2)
}

func BenchmarkMarshalBinary(b *testing.B) {
	filter, _ := Compile(
		`(lower(http.host) == "example.com" or http.host wildcard "*.example.com") and http.status >= 400 and ip.src not in $blocked_ips`,
		nil,
	)

	b.ReportAllocs()
	for b.Loop() {
		_, _ = filter.MarshalBinary()
	}
}

func BenchmarkUnmarshalBinary(b *testing.B) {
	filter, _ := Compile(
		`(lower(http.host) == "example.com" or http.host wildcard "*.example.com") and http.status >= 400 and ip.src not in $blocked_ips`,
		nil,
	)
	data, _ := filter.MarshalBinary()

	b.ReportAllocs()
	for b.Loop() {
		f := &Filter{}
		_ = f.UnmarshalBinary(data)
	}
}

func BenchmarkCompileVsUnmarshal(b *testing.B) {
	expr := `(lower(http.host) == "example.com" or http.host wildcard "*.example.com") and http.status >= 400 and ip.src not in $blocked_ips`

	filter, _ := Compile(expr, nil)
	data, _ := filter.MarshalBinary()

	b.Run("compile", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_, _ = Compile(expr, nil)
		}
	})

	b.Run("unmarshal", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			f := &Filter{}
			_ = f.UnmarshalBinary(data)
		}
	})
}

func FuzzMarshalUnmarshal(f *testing.F) {
	f.Add(`name == "test"`)
	f.Add(`status >= 400`)
	f.Add(`ip in $blocked`)
	f.Add(`tags[*] contains "prod"`)
	f.Add(`cidr(ip, 24) == "10.0.0.0"`)
	f.Add(`a and b or not c`)
	f.Add(`x in {1..100}`)
	f.Add(`lower(name) not contains "admin"`)
	f.Add(`data["key"] == "val"`)
	f.Add(`$geo[ip] == "US"`)
	f.Add(`role in $allowed[dept]`)
	f.Add(`$config["mode"] == "prod"`)
	f.Add(`maintenance() == true`)
	f.Add(`get_score(name) > 5.0`)
	f.Add(`is_tor(ip) and name == "test"`)

	f.Fuzz(func(t *testing.T, expr string) {
		filter, err := Compile(expr, nil)
		if err != nil {
			return
		}

		data, err := filter.MarshalBinary()
		if err != nil {
			t.Fatalf("MarshalBinary failed for %q: %v", expr, err)
		}

		restored := &Filter{}
		if err := restored.UnmarshalBinary(data); err != nil {
			t.Fatalf("UnmarshalBinary failed for %q: %v", expr, err)
		}

		// Re-marshal should produce identical bytes
		data2, err := restored.MarshalBinary()
		if err != nil {
			t.Fatalf("second MarshalBinary failed for %q: %v", expr, err)
		}

		if len(data) != len(data2) {
			t.Fatalf("roundtrip mismatch for %q: %d vs %d bytes", expr, len(data), len(data2))
		}
	})
}
