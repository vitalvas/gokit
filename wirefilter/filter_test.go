package wirefilter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func BenchmarkCompile(b *testing.B) {
	schema := NewSchema().
		AddField("http.host", TypeString).
		AddField("http.status", TypeInt).
		AddField("ip.src", TypeIP)

	tests := []struct {
		name       string
		expression string
	}{
		{
			name:       "simple equality",
			expression: `http.host == "example.com"`,
		},
		{
			name:       "multiple conditions",
			expression: `http.host == "example.com" and http.status >= 400`,
		},
		{
			name:       "complex expression",
			expression: `(http.host == "example.com" or http.host == "test.com") and http.status >= 200 and http.status < 300`,
		},
		{
			name:       "ip in cidr",
			expression: `ip.src in "192.168.0.0/16"`,
		},
		{
			name:       "array membership",
			expression: `http.status in {200, 201, 204, 301, 302, 304}`,
		},
		{
			name:       "range expression",
			expression: `http.status in {200..299, 400..499}`,
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				_, err := Compile(tt.expression, schema)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkExecute(b *testing.B) {
	schema := NewSchema().
		AddField("http.host", TypeString).
		AddField("http.status", TypeInt).
		AddField("http.path", TypeString).
		AddField("ip.src", TypeIP)

	tests := []struct {
		name       string
		expression string
		setup      func() *ExecutionContext
	}{
		{
			name:       "simple equality",
			expression: `http.host == "example.com"`,
			setup: func() *ExecutionContext {
				return NewExecutionContext().
					SetStringField("http.host", "example.com")
			},
		},
		{
			name:       "multiple conditions",
			expression: `http.host == "example.com" and http.status >= 400`,
			setup: func() *ExecutionContext {
				return NewExecutionContext().
					SetStringField("http.host", "example.com").
					SetIntField("http.status", 500)
			},
		},
		{
			name:       "complex boolean logic",
			expression: `(http.host == "example.com" or http.host == "test.com") and http.status >= 200 and http.status < 300`,
			setup: func() *ExecutionContext {
				return NewExecutionContext().
					SetStringField("http.host", "example.com").
					SetIntField("http.status", 200)
			},
		},
		{
			name:       "string contains",
			expression: `http.path contains "/api"`,
			setup: func() *ExecutionContext {
				return NewExecutionContext().
					SetStringField("http.path", "/api/v1/users")
			},
		},
		{
			name:       "regex match",
			expression: `http.host matches "^example\\..*"`,
			setup: func() *ExecutionContext {
				return NewExecutionContext().
					SetStringField("http.host", "example.com")
			},
		},
		{
			name:       "ip in cidr",
			expression: `ip.src in "192.168.0.0/16"`,
			setup: func() *ExecutionContext {
				return NewExecutionContext().
					SetIPField("ip.src", "192.168.1.1")
			},
		},
		{
			name:       "array membership",
			expression: `http.status in {200, 201, 204, 301, 302, 304}`,
			setup: func() *ExecutionContext {
				return NewExecutionContext().
					SetIntField("http.status", 200)
			},
		},
		{
			name:       "range expression",
			expression: `http.status in {200..299}`,
			setup: func() *ExecutionContext {
				return NewExecutionContext().
					SetIntField("http.status", 250)
			},
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			filter, err := Compile(tt.expression, schema)
			if err != nil {
				b.Fatal(err)
			}

			ctx := tt.setup()

			b.ResetTimer()
			b.ReportAllocs()
			for b.Loop() {
				_, err := filter.Execute(ctx)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func FuzzCompile(f *testing.F) {
	f.Add(`http.host == "example.com"`)
	f.Add(`http.status >= 400`)
	f.Add(`http.host == "example.com" and http.status >= 400`)
	f.Add(`(http.host == "test.com" or http.path contains "/api") and http.status < 500`)
	f.Add(`http.status in {200, 201, 204, 301, 302, 304}`)
	f.Add(`port in {80..100, 443, 8000..9000}`)
	f.Add(`ip.src in "192.168.0.0/16"`)
	f.Add(`http.path matches "^/api/v[0-9]+/"`)
	f.Add(`not http.host == "blocked.com"`)
	f.Add(`true and false`)
	f.Add(`ip not in $blocked`)
	f.Add(`$geo[ip] == "US"`)
	f.Add(`role in $allowed[dept]`)
	f.Add(`$config["key"] == "val"`)
	f.Add(`name not contains "admin"`)
	f.Add(`cidr(ip, 24) == 10.0.0.0/24`)
	f.Add(`cidr6(ip, 64) == 2001:db8::/64`)
	f.Add(`lower(name) == "test"`)
	f.Add(`tags[*] == "prod"`)
	f.Add(`all(tags[*] contains "a")`)
	f.Add(`any(ports[*] > 80)`)
	f.Add(`data["key"] == "val"`)
	f.Add(`a xor b`)
	f.Add(`name wildcard "*.com"`)
	f.Add(`name strict wildcard "*.COM"`)
	f.Add(`tags === "a"`)
	f.Add(`tags !== "b"`)
	f.Add(`ip.src in 192.168.0.0/24`)
	f.Add(`concat("a", "b") == "ab"`)
	f.Add(`split(name, ",")[0] == "a"`)
	f.Add(`join(tags, ",") == "a,b"`)
	f.Add(`has_key(data, "key")`)
	f.Add(`has_value(tags, "a")`)
	f.Add(`starts_with(name, "test")`)
	f.Add(`ends_with(name, ".com")`)
	f.Add(`len(name) > 0`)
	f.Add(`url_decode(name) == "a b"`)
	f.Add(`substring(name, 0, 3) == "tes"`)
	f.Add(`trim(name) == "test"`)
	f.Add(`replace(name, "a", "b") == "b"`)
	f.Add(`regex_replace(name, "[0-9]+", "X") == "X"`)
	f.Add(`regex_extract(name, "[0-9]+") == "123"`)
	f.Add(`contains_word(name, "test")`)
	f.Add(`count(tags) > 0`)
	f.Add(`coalesce(a, b) == "x"`)
	f.Add(`abs(x) > 0`)
	f.Add(`ceil(x) == 4`)
	f.Add(`floor(x) == 3`)
	f.Add(`round(x) == 4`)
	f.Add(`is_ipv4(ip) == true`)
	f.Add(`is_loopback(ip) == true`)
	f.Add(`intersection(a, b)`)
	f.Add(`union(a, b)`)
	f.Add(`difference(a, b)`)
	f.Add(`contains_any(a, b)`)
	f.Add(`contains_all(a, b)`)
	f.Add(`custom_func() == true`)
	f.Add(`get_score(name) > 5.0`)
	f.Add(`is_tor(ip) and name == "test"`)
	f.Add(`ip in get_cidrs(name)`)
	f.Add(`exists(name)`)
	f.Add(`not exists(missing)`)
	f.Add(`exists(name) and name == "test"`)
	f.Add(`x + 1 > 5`)
	f.Add(`x * 2 == 10`)
	f.Add(`x / 3 == 1`)
	f.Add(`x % 2 == 0`)

	f.Fuzz(func(_ *testing.T, input string) {
		_, _ = Compile(input, nil)
	})
}

func FuzzExecute(f *testing.F) {
	f.Add(`http.host == "example.com"`, "example.com", int64(200))
	f.Add(`http.status >= 400`, "test.com", int64(500))
	f.Add(`http.host == "example.com" and http.status >= 400`, "example.com", int64(404))
	f.Add(`http.status in {200, 201, 204}`, "test.com", int64(200))
	f.Add(`http.host contains "example"`, "example.com", int64(200))
	f.Add(`http.status < 300`, "test.com", int64(250))
	f.Add(`not http.host == "blocked"`, "allowed.com", int64(200))

	schema := NewSchema().
		AddField("http.host", TypeString).
		AddField("http.status", TypeInt)

	f.Fuzz(func(_ *testing.T, expression string, host string, status int64) {
		filter, err := Compile(expression, schema)
		if err != nil {
			return
		}

		ctx := NewExecutionContext().
			SetStringField("http.host", host).
			SetIntField("http.status", status)

		_, _ = filter.Execute(ctx)
	})
}

func FuzzExecuteMultiType(f *testing.F) {
	f.Add(`name == value`, "test", "test", int64(0), "10.0.0.1")
	f.Add(`name contains value`, "hello world", "world", int64(0), "10.0.0.1")
	f.Add(`count > 5`, "x", "x", int64(10), "10.0.0.1")
	f.Add(`ip == "10.0.0.1"`, "x", "x", int64(0), "10.0.0.1")
	f.Add(`name not contains "admin"`, "user", "admin", int64(0), "10.0.0.1")
	f.Add(`count in {1..100}`, "x", "x", int64(50), "10.0.0.1")
	f.Add(`lower(name) == value`, "TEST", "test", int64(0), "10.0.0.1")
	f.Add(`len(name) > count`, "hello", "x", int64(3), "10.0.0.1")
	f.Add(`$geo[ip] == "US"`, "x", "US", int64(0), "10.0.0.1")
	f.Add(`name in $allowed[value]`, "dev", "eng", int64(0), "10.0.0.1")
	f.Add(`custom_func() == true`, "x", "x", int64(0), "10.0.0.1")
	f.Add(`get_score(name) > 5.0`, "test", "x", int64(0), "10.0.0.1")

	f.Fuzz(func(_ *testing.T, expression, strVal1, strVal2 string, intVal int64, ipVal string) {
		filter, err := Compile(expression, nil)
		if err != nil {
			return
		}

		ctx := NewExecutionContext().
			SetStringField("name", strVal1).
			SetStringField("value", strVal2).
			SetIntField("count", intVal).
			SetIPField("ip", ipVal).
			SetBoolField("active", intVal > 0).
			SetArrayField("tags", []string{strVal1, strVal2}).
			SetIntArrayField("ports", []int64{intVal, intVal + 1}).
			SetMapField("data", map[string]string{"key": strVal1}).
			SetList("names", []string{strVal1, strVal2}).
			SetIPList("nets", []string{"10.0.0.0/8", "192.168.0.0/16"}).
			SetTable("geo", map[string]string{ipVal: "US", strVal1: strVal2}).
			SetTableList("allowed", map[string][]string{strVal1: {strVal2}}).
			SetTableIPList("blocked", map[string][]string{"office": {"10.0.0.0/8"}}).
			SetFunc("custom_func", func(_ []Value) (Value, error) {
				return BoolValue(true), nil
			}).
			SetFunc("get_score", func(_ []Value) (Value, error) {
				return FloatValue(7.5), nil
			})

		_, _ = filter.Execute(ctx)
	})
}

func FuzzIPListOperations(f *testing.F) {
	f.Add("10.0.0.1", "10.0.0.0/8")
	f.Add("192.168.1.100", "192.168.0.0/16")
	f.Add("172.16.5.1", "172.16.0.0/12")
	f.Add("8.8.8.8", "8.8.8.0/24")
	f.Add("2001:db8::1", "2001:db8::/32")
	f.Add("fe80::1", "fe80::/10")
	f.Add("invalid", "invalid/cidr")

	f.Fuzz(func(_ *testing.T, ipStr, cidrStr string) {
		filter, err := Compile(`ip not in $nets`, nil)
		if err != nil {
			return
		}

		ctx := NewExecutionContext().
			SetIPField("ip", ipStr).
			SetIPList("nets", []string{cidrStr})

		_, _ = filter.Execute(ctx)
	})
}

func FuzzFunctions(f *testing.F) {
	f.Add(`lower(name)`, "HELLO")
	f.Add(`upper(name)`, "hello")
	f.Add(`len(name)`, "test")
	f.Add(`starts_with(name, "he")`, "hello")
	f.Add(`ends_with(name, "lo")`, "hello")
	f.Add(`concat(name, "!")`, "hello")
	f.Add(`substring(name, 0, 3)`, "hello")
	f.Add(`split(name, ",")`, "a,b,c")
	f.Add(`url_decode(name)`, "hello%20world")
	f.Add(`cidr(ip, 24)`, "192.168.1.100")
	f.Add(`cidr6(ip, 64)`, "2001:db8::1")

	f.Fuzz(func(_ *testing.T, expression, value string) {
		filter, err := Compile(expression, nil)
		if err != nil {
			return
		}

		ctx := NewExecutionContext().
			SetStringField("name", value).
			SetIPField("ip", value).
			SetIntField("n", int64(len(value)))

		_, _ = filter.Execute(ctx)
	})
}

func FuzzSchemaValidation(f *testing.F) {
	f.Add(`name == "test"`)
	f.Add(`name contains "test"`)
	f.Add(`count > 5`)
	f.Add(`ip in $blocked`)
	f.Add(`tags[*] == "a"`)
	f.Add(`data["key"] == "val"`)
	f.Add(`lower(name) == "test"`)
	f.Add(`name not in {"a", "b"}`)
	f.Add(`unknown_field == "x"`)

	schema := NewSchema().
		AddField("name", TypeString).
		AddField("count", TypeInt).
		AddField("ip", TypeIP).
		AddField("tags", TypeArray).
		AddField("data", TypeMap)

	f.Fuzz(func(_ *testing.T, expression string) {
		_, _ = Compile(expression, schema)
	})
}

func TestFilterCompile(t *testing.T) {
	t.Run("compile without schema", func(t *testing.T) {
		filter, err := Compile(`http.host == "test"`, nil)
		assert.NoError(t, err)
		assert.NotNil(t, filter)
	})

	t.Run("parse error - invalid expression", func(t *testing.T) {
		_, err := Compile(`http.host ==`, nil)
		assert.Error(t, err)
	})

	t.Run("parse error - unclosed parenthesis", func(t *testing.T) {
		_, err := Compile(`(http.host == "test"`, nil)
		assert.Error(t, err)
	})

	t.Run("parse error - unclosed brace", func(t *testing.T) {
		_, err := Compile(`status in {200, 201`, nil)
		assert.Error(t, err)
	})

	t.Run("schema validation - unknown field", func(t *testing.T) {
		schema := NewSchema().
			AddField("http.host", TypeString)

		_, err := Compile(`http.unknown == "test"`, schema)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown field")
	})

	t.Run("schema validation - nested unknown field", func(t *testing.T) {
		schema := NewSchema().
			AddField("http.host", TypeString)

		_, err := Compile(`http.host == "test" and http.unknown == "test"`, schema)
		assert.Error(t, err)
	})

	t.Run("schema validation - unary expression", func(t *testing.T) {
		schema := NewSchema().
			AddField("http.host", TypeString)

		_, err := Compile(`not http.unknown`, schema)
		assert.Error(t, err)
	})

	t.Run("schema with initial fields map", func(t *testing.T) {
		fields := map[string]Type{
			"http.host":   TypeString,
			"http.status": TypeInt,
			"http.secure": TypeBool,
		}

		schema := NewSchema(fields)

		filter, err := Compile(`http.host == "example.com" and http.status == 200 and http.secure == true`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetStringField("http.host", "example.com").
			SetIntField("http.status", 200).
			SetBoolField("http.secure", true)

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)

		field, ok := schema.GetField("http.host")
		assert.True(t, ok)
		assert.Equal(t, "http.host", field.Name)
		assert.Equal(t, TypeString, field.Type)
	})

	t.Run("schema with multiple field maps", func(t *testing.T) {
		httpFields := map[string]Type{
			"http.host":   TypeString,
			"http.status": TypeInt,
		}

		ipFields := map[string]Type{
			"ip.src": TypeIP,
			"ip.dst": TypeIP,
		}

		schema := NewSchema(httpFields, ipFields)

		filter, err := Compile(`http.host == "example.com" and ip.src in "192.168.0.0/16"`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetStringField("http.host", "example.com").
			SetIPField("ip.src", "192.168.1.1")

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)

		httpField, ok := schema.GetField("http.host")
		assert.True(t, ok)
		assert.Equal(t, TypeString, httpField.Type)

		ipField, ok := schema.GetField("ip.src")
		assert.True(t, ok)
		assert.Equal(t, TypeIP, ipField.Type)
	})

	t.Run("schema validation with range in array", func(t *testing.T) {
		schema := NewSchema().
			AddField("status", TypeInt)

		filter, err := Compile(`status in {200..299}`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetIntField("status", 250)

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("schema validation - unpack expression", func(t *testing.T) {
		schema := NewSchema().
			AddField("tags", TypeArray)

		_, err := Compile(`tags[*] == "test"`, schema)
		assert.NoError(t, err)

		// Unknown field in unpack expression
		_, err = Compile(`unknown[*] == "test"`, schema)
		assert.Error(t, err)
	})

	t.Run("schema validation - list reference", func(t *testing.T) {
		schema := NewSchema().
			AddField("role", TypeString)

		// List references are validated at runtime, not compile time
		_, err := Compile(`role in $any_list`, schema)
		assert.NoError(t, err)
	})

	t.Run("all equal operator - non-array value rejected by schema", func(t *testing.T) {
		schema := NewSchema().
			AddField("name", TypeString)

		_, err := Compile(`name === "test"`, schema)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not valid for field type")
	})

	t.Run("any not equal operator - non-array value rejected by schema", func(t *testing.T) {
		schema := NewSchema().
			AddField("name", TypeString)

		_, err := Compile(`name !== "test"`, schema)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not valid for field type")
	})

	t.Run("wildcard with non-string types rejected by schema", func(t *testing.T) {
		schema := NewSchema().
			AddField("count", TypeInt)

		_, err := Compile(`count wildcard "123"`, schema)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not valid for field type")
	})

	t.Run("grouped expression", func(t *testing.T) {
		filter, err := Compile(`(http.status == 200 or http.status == 201) and http.host == "test"`, nil)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetIntField("http.status", 200).
			SetStringField("http.host", "test")

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("execute returns error on nil result", func(t *testing.T) {
		filter, err := Compile(`http.host`, nil)
		assert.NoError(t, err)

		ctx := NewExecutionContext()
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("raw string - no escape processing", func(t *testing.T) {
		schema := NewSchema().
			AddField("path", TypeString)

		filter, err := Compile(`path matches r"^C:\\Users\\.*"`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetStringField("path", `C:\Users\admin`)

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("raw string - regex pattern", func(t *testing.T) {
		schema := NewSchema().
			AddField("email", TypeString)

		filter, err := Compile(`email matches r"^\w+@\w+\.\w+$"`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetStringField("email", "user@example.com")

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("raw string - empty", func(t *testing.T) {
		filter, err := Compile(`field == r""`, nil)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetStringField("field", "")

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("lexer error - unterminated string", func(t *testing.T) {
		_, err := Compile(`name == "unterminated`, nil)
		assert.Error(t, err)
	})

	t.Run("lexer error - integer overflow", func(t *testing.T) {
		_, err := Compile(`x == 99999999999999999999999999999`, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "integer overflow")
	})

	t.Run("lexer error - unknown character", func(t *testing.T) {
		// A single @ at the start triggers lexer error
		_, err := Compile(`@`, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unexpected character")
	})

	t.Run("unterminated raw string", func(t *testing.T) {
		_, err := Compile(`name == r"unterminated`, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unterminated raw string")
	})

	t.Run("trailing garbage - single ampersand", func(t *testing.T) {
		// "a & b" should fail - single & is not a valid operator
		_, err := Compile(`a & b`, nil)
		assert.Error(t, err)
	})

	t.Run("trailing garbage - unterminated string after valid expr", func(t *testing.T) {
		// Should fail with lexer error in trailing position
		_, err := Compile(`a "unterminated`, nil)
		assert.Error(t, err)
	})

	t.Run("trailing garbage - extra identifier", func(t *testing.T) {
		_, err := Compile(`a b`, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unexpected trailing token")
	})

	t.Run("function result indexing is valid", func(t *testing.T) {
		// split(x, ",")[0] should be valid - indexing function result
		filter, err := Compile(`split(name, ",")[0] == "a"`, nil)
		assert.NoError(t, err)

		ctx := NewExecutionContext().SetStringField("name", "a,b,c")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("schema validates range expr with unknown field", func(t *testing.T) {
		schema := NewSchema().AddField("x", TypeInt)
		_, err := Compile(`x in {unknown_start..10}`, schema)
		assert.Error(t, err)
	})

	t.Run("schema validates range expr end with unknown field", func(t *testing.T) {
		schema := NewSchema().AddField("x", TypeInt)
		_, err := Compile(`x in {1..unknown_end}`, schema)
		assert.Error(t, err)
	})

	t.Run("schema validates array elements with unknown field", func(t *testing.T) {
		schema := NewSchema().AddField("x", TypeInt)
		_, err := Compile(`x in {unknown, 1}`, schema)
		assert.Error(t, err)
	})

	t.Run("lexer unterminated string with escape", func(t *testing.T) {
		lexer := NewLexer(`"test\`)
		tok := lexer.NextToken()
		assert.Equal(t, TokenError, tok.Type)
	})

	t.Run("lexer CIDR in number context", func(t *testing.T) {
		lexer := NewLexer(`192.168.0.0/24`)
		tok := lexer.NextToken()
		assert.Equal(t, TokenCIDR, tok.Type)
	})

	t.Run("lexer IPv6 CIDR", func(t *testing.T) {
		lexer := NewLexer(`2001:db8::/32`)
		tok := lexer.NextToken()
		assert.Equal(t, TokenCIDR, tok.Type)
	})

	t.Run("lexer negative number overflow", func(t *testing.T) {
		lexer := NewLexer(`-99999999999999999999999`)
		tok := lexer.NextToken()
		assert.Equal(t, TokenError, tok.Type)
	})
}

func TestFilterHash(t *testing.T) {
	t.Run("identical expressions produce same hash", func(t *testing.T) {
		f1, err := Compile(`name == "test"`, nil)
		require.NoError(t, err)
		f2, err := Compile(`name == "test"`, nil)
		require.NoError(t, err)

		assert.Equal(t, f1.Hash(), f2.Hash())
		assert.Len(t, f1.Hash(), 32) // 128-bit FNV = 16 bytes = 32 hex chars
	})

	t.Run("extra whitespace ignored", func(t *testing.T) {
		f1, err := Compile(`name=="test"`, nil)
		require.NoError(t, err)
		f2, err := Compile(`name   ==   "test"`, nil)
		require.NoError(t, err)

		assert.Equal(t, f1.Hash(), f2.Hash())
	})

	t.Run("tabs and newlines ignored", func(t *testing.T) {
		f1, err := Compile(`name == "test"`, nil)
		require.NoError(t, err)
		f2, err := Compile("name\t==\n\"test\"", nil)
		require.NoError(t, err)

		assert.Equal(t, f1.Hash(), f2.Hash())
	})

	t.Run("operator aliases produce same hash", func(t *testing.T) {
		f1, err := Compile(`a and b`, nil)
		require.NoError(t, err)
		f2, err := Compile(`a && b`, nil)
		require.NoError(t, err)

		assert.Equal(t, f1.Hash(), f2.Hash())
	})

	t.Run("or alias", func(t *testing.T) {
		f1, err := Compile(`a or b`, nil)
		require.NoError(t, err)
		f2, err := Compile(`a || b`, nil)
		require.NoError(t, err)

		assert.Equal(t, f1.Hash(), f2.Hash())
	})

	t.Run("not alias", func(t *testing.T) {
		f1, err := Compile(`not a`, nil)
		require.NoError(t, err)
		f2, err := Compile(`! a`, nil)
		require.NoError(t, err)

		assert.Equal(t, f1.Hash(), f2.Hash())
	})

	t.Run("xor alias", func(t *testing.T) {
		f1, err := Compile(`a xor b`, nil)
		require.NoError(t, err)
		f2, err := Compile(`a ^^ b`, nil)
		require.NoError(t, err)

		assert.Equal(t, f1.Hash(), f2.Hash())
	})

	t.Run("matches alias", func(t *testing.T) {
		f1, err := Compile(`name matches "^test"`, nil)
		require.NoError(t, err)
		f2, err := Compile(`name ~ "^test"`, nil)
		require.NoError(t, err)

		assert.Equal(t, f1.Hash(), f2.Hash())
	})

	t.Run("different expressions produce different hash", func(t *testing.T) {
		f1, err := Compile(`name == "test"`, nil)
		require.NoError(t, err)
		f2, err := Compile(`name == "other"`, nil)
		require.NoError(t, err)

		assert.NotEqual(t, f1.Hash(), f2.Hash())
	})

	t.Run("different operators produce different hash", func(t *testing.T) {
		f1, err := Compile(`x == 1`, nil)
		require.NoError(t, err)
		f2, err := Compile(`x != 1`, nil)
		require.NoError(t, err)

		assert.NotEqual(t, f1.Hash(), f2.Hash())
	})

	t.Run("different fields produce different hash", func(t *testing.T) {
		f1, err := Compile(`name == "test"`, nil)
		require.NoError(t, err)
		f2, err := Compile(`host == "test"`, nil)
		require.NoError(t, err)

		assert.NotEqual(t, f1.Hash(), f2.Hash())
	})

	t.Run("complex expression with aliases", func(t *testing.T) {
		f1, err := Compile(`name == "test" and status >= 400 or not active`, nil)
		require.NoError(t, err)
		f2, err := Compile(`name == "test" && status >= 400 || ! active`, nil)
		require.NoError(t, err)

		assert.Equal(t, f1.Hash(), f2.Hash())
	})
}

func TestFilterHashStable(t *testing.T) {
	expected := map[string]string{
		`name == "test"`:                "c2889f4a7ccca7ff44f3d705ede3a9d2",
		`status >= 400`:                 "4d0d67a73f751e14aacca5bb3502c749",
		`a and b`:                       "8d37d268ba0d433e9085647d4515db7e",
		`a or b`:                        "8cc4a49ad50d433e83bc73aaacd7db57",
		`not a`:                         "c8194b89c2659af17cda0cbf1bbcba23",
		`a xor b`:                       "ac37ffb8bf0d433e7b23fe3a6bcf6e85",
		`name matches "^test"`:          "be80d3f9b61f58a7a706ad74e2763340",
		`ip in "10.0.0.0/8"`:            "a90ffc4474bd91cc1e8539ee6c34dbee",
		`ip not in $blocked`:            "39d433affcd7e7a207116d845e2a9a90",
		`tags[*] == "prod"`:             "ff16911f0e06efe44d5eb8288add6bef",
		`lower(name) == "admin"`:        "ee93a7331fa511f603da995bab8a3ad5",
		`cidr(ip, 24) == 10.0.0.0/24`:   "7b88e911d95eb856e3d2bcc7a7da10a5",
		`x in {1..100}`:                 "22fa482e0fc63ac8541c43d27a475328",
		`name not contains "admin"`:     "4074d01cb7420c78dea8b032e8307b1f",
		`data["key"] == "val"`:          "3368e39bf8c533c6cff6b3a987a43ef5",
		`(a == 1 or b == 2) and c == 3`: "6acb7ec503e4dfdf1a5391f787845042",
	}

	for expr, wantHash := range expected {
		t.Run(expr, func(t *testing.T) {
			f, err := Compile(expr, nil)
			require.NoError(t, err)
			assert.Equal(t, wantHash, f.Hash())
		})
	}
}

func TestRuleMeta(t *testing.T) {
	t.Run("set and get meta", func(t *testing.T) {
		filter, _ := Compile(`name == "test"`, nil)
		filter.SetMeta(RuleMeta{
			ID:   "WAF-1001",
			Tags: map[string]string{"severity": "high", "category": "xss"},
		})

		meta := filter.Meta()
		assert.Equal(t, "WAF-1001", meta.ID)
		assert.Equal(t, "high", meta.Tags["severity"])
		assert.Equal(t, "xss", meta.Tags["category"])
	})

	t.Run("default meta is empty", func(t *testing.T) {
		filter, _ := Compile(`name == "test"`, nil)
		meta := filter.Meta()
		assert.Empty(t, meta.ID)
		assert.Nil(t, meta.Tags)
	})

	t.Run("chaining", func(t *testing.T) {
		filter, _ := Compile(`name == "test"`, nil)
		f := filter.SetMeta(RuleMeta{ID: "R1"})
		assert.Equal(t, "R1", f.Meta().ID)
	})
}
