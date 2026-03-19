package wirefilter

import (
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFnRegexReplace(t *testing.T) {
	t.Run("basic replace", func(t *testing.T) {
		f, err := Compile(`regex_replace(name, "[0-9]+", "X") == "userX"`, nil)
		require.NoError(t, err)
		ctx := NewExecutionContext().SetStringField("name", "user123")
		result, _ := f.Execute(ctx)
		assert.True(t, result)
	})

	t.Run("no match", func(t *testing.T) {
		f, err := Compile(`regex_replace(name, "[0-9]+", "X") == "hello"`, nil)
		require.NoError(t, err)
		ctx := NewExecutionContext().SetStringField("name", "hello")
		result, _ := f.Execute(ctx)
		assert.True(t, result)
	})

	t.Run("wrong arg count", func(t *testing.T) {
		f, err := Compile(`regex_replace(name, "[0-9]+") == "x"`, nil)
		require.NoError(t, err)
		ctx := NewExecutionContext().SetStringField("name", "test")
		result, _ := f.Execute(ctx)
		assert.False(t, result)
	})

	t.Run("invalid regex", func(t *testing.T) {
		f, err := Compile(`regex_replace(name, "[invalid", "X") == "x"`, nil)
		require.NoError(t, err)
		ctx := NewExecutionContext().SetStringField("name", "test")
		_, err = f.Execute(ctx)
		assert.Error(t, err)
	})
}

func TestFnTrim(t *testing.T) {
	t.Run("trim", func(t *testing.T) {
		f, _ := Compile(`trim(name) == "hello"`, nil)
		ctx := NewExecutionContext().SetStringField("name", "  hello  ")
		result, _ := f.Execute(ctx)
		assert.True(t, result)
	})

	t.Run("trim_left", func(t *testing.T) {
		f, _ := Compile(`trim_left(name) == "hello  "`, nil)
		ctx := NewExecutionContext().SetStringField("name", "  hello  ")
		result, _ := f.Execute(ctx)
		assert.True(t, result)
	})

	t.Run("trim_right", func(t *testing.T) {
		f, _ := Compile(`trim_right(name) == "  hello"`, nil)
		ctx := NewExecutionContext().SetStringField("name", "  hello  ")
		result, _ := f.Execute(ctx)
		assert.True(t, result)
	})

	t.Run("nil arg", func(t *testing.T) {
		f, _ := Compile(`trim(missing) == ""`, nil)
		ctx := NewExecutionContext()
		result, _ := f.Execute(ctx)
		assert.False(t, result)
	})

	t.Run("wrong type", func(t *testing.T) {
		f, _ := Compile(`trim(count) == ""`, nil)
		ctx := NewExecutionContext().SetIntField("count", 42)
		result, _ := f.Execute(ctx)
		assert.False(t, result)
	})
}

func TestFnReplace(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		f, _ := Compile(`replace(name, "world", "go") == "hello go"`, nil)
		ctx := NewExecutionContext().SetStringField("name", "hello world")
		result, _ := f.Execute(ctx)
		assert.True(t, result)
	})

	t.Run("multiple occurrences", func(t *testing.T) {
		f, _ := Compile(`replace(name, "a", "b") == "bbb"`, nil)
		ctx := NewExecutionContext().SetStringField("name", "aaa")
		result, _ := f.Execute(ctx)
		assert.True(t, result)
	})

	t.Run("wrong arg count", func(t *testing.T) {
		f, _ := Compile(`replace(name, "a") == "x"`, nil)
		ctx := NewExecutionContext().SetStringField("name", "test")
		result, _ := f.Execute(ctx)
		assert.False(t, result)
	})
}

func TestFnCount(t *testing.T) {
	t.Run("count truthy elements", func(t *testing.T) {
		f, _ := Compile(`count(tags) > 0`, nil)
		ctx := NewExecutionContext().SetArrayField("tags", []string{"a", "b"})
		result, _ := f.Execute(ctx)
		assert.True(t, result)
	})

	t.Run("count value", func(t *testing.T) {
		f, _ := Compile(`count(tags) == 3`, nil)
		ctx := NewExecutionContext().SetArrayField("tags", []string{"a", "b", "c"})
		result, _ := f.Execute(ctx)
		assert.True(t, result)
	})

	t.Run("nil arg", func(t *testing.T) {
		f, _ := Compile(`count(missing) == 0`, nil)
		ctx := NewExecutionContext()
		result, _ := f.Execute(ctx)
		assert.False(t, result)
	})

	t.Run("non-array", func(t *testing.T) {
		f, _ := Compile(`count(name) == 0`, nil)
		ctx := NewExecutionContext().SetStringField("name", "test")
		result, _ := f.Execute(ctx)
		assert.False(t, result)
	})
}

func TestFnCoalesce(t *testing.T) {
	t.Run("first non-nil", func(t *testing.T) {
		f, _ := Compile(`coalesce(missing, name) == "hello"`, nil)
		ctx := NewExecutionContext().SetStringField("name", "hello")
		result, _ := f.Execute(ctx)
		assert.True(t, result)
	})

	t.Run("first arg non-nil", func(t *testing.T) {
		f, _ := Compile(`coalesce(name, other) == "hello"`, nil)
		ctx := NewExecutionContext().SetStringField("name", "hello").SetStringField("other", "world")
		result, _ := f.Execute(ctx)
		assert.True(t, result)
	})

	t.Run("all nil", func(t *testing.T) {
		f, _ := Compile(`coalesce(a, b, c) == "x"`, nil)
		ctx := NewExecutionContext()
		result, _ := f.Execute(ctx)
		assert.False(t, result)
	})

	t.Run("no args", func(t *testing.T) {
		f, _ := Compile(`coalesce() == "x"`, nil)
		ctx := NewExecutionContext()
		result, _ := f.Execute(ctx)
		assert.False(t, result)
	})
}

func TestFnContainsWord(t *testing.T) {
	t.Run("word found", func(t *testing.T) {
		f, _ := Compile(`contains_word(msg, "admin") == true`, nil)
		ctx := NewExecutionContext().SetStringField("msg", "hello admin world")
		result, _ := f.Execute(ctx)
		assert.True(t, result)
	})

	t.Run("substring but not word", func(t *testing.T) {
		f, _ := Compile(`contains_word(msg, "admin") == true`, nil)
		ctx := NewExecutionContext().SetStringField("msg", "sysadmin access")
		result, _ := f.Execute(ctx)
		assert.False(t, result)
	})

	t.Run("word at start", func(t *testing.T) {
		f, _ := Compile(`contains_word(msg, "hello") == true`, nil)
		ctx := NewExecutionContext().SetStringField("msg", "hello world")
		result, _ := f.Execute(ctx)
		assert.True(t, result)
	})

	t.Run("word at end", func(t *testing.T) {
		f, _ := Compile(`contains_word(msg, "world") == true`, nil)
		ctx := NewExecutionContext().SetStringField("msg", "hello world")
		result, _ := f.Execute(ctx)
		assert.True(t, result)
	})

	t.Run("wrong arg count", func(t *testing.T) {
		f, _ := Compile(`contains_word(msg) == true`, nil)
		ctx := NewExecutionContext().SetStringField("msg", "test")
		result, _ := f.Execute(ctx)
		assert.False(t, result)
	})
}

func TestFnMath(t *testing.T) {
	t.Run("abs int positive", func(t *testing.T) {
		f, _ := Compile(`abs(x) == 5`, nil)
		ctx := NewExecutionContext().SetIntField("x", 5)
		result, _ := f.Execute(ctx)
		assert.True(t, result)
	})

	t.Run("abs int negative", func(t *testing.T) {
		f, _ := Compile(`abs(x) == 5`, nil)
		ctx := NewExecutionContext().SetIntField("x", -5)
		result, _ := f.Execute(ctx)
		assert.True(t, result)
	})

	t.Run("abs float", func(t *testing.T) {
		f, _ := Compile(`abs(x) == 3.14`, nil)
		ctx := NewExecutionContext().SetFloatField("x", -3.14)
		result, _ := f.Execute(ctx)
		assert.True(t, result)
	})

	t.Run("ceil", func(t *testing.T) {
		f, _ := Compile(`ceil(x) == 4`, nil)
		ctx := NewExecutionContext().SetFloatField("x", 3.2)
		result, _ := f.Execute(ctx)
		assert.True(t, result)
	})

	t.Run("ceil int passthrough", func(t *testing.T) {
		f, _ := Compile(`ceil(x) == 5`, nil)
		ctx := NewExecutionContext().SetIntField("x", 5)
		result, _ := f.Execute(ctx)
		assert.True(t, result)
	})

	t.Run("floor", func(t *testing.T) {
		f, _ := Compile(`floor(x) == 3`, nil)
		ctx := NewExecutionContext().SetFloatField("x", 3.9)
		result, _ := f.Execute(ctx)
		assert.True(t, result)
	})

	t.Run("round", func(t *testing.T) {
		f, _ := Compile(`round(x) == 4`, nil)
		ctx := NewExecutionContext().SetFloatField("x", 3.6)
		result, _ := f.Execute(ctx)
		assert.True(t, result)
	})

	t.Run("round down", func(t *testing.T) {
		f, _ := Compile(`round(x) == 3`, nil)
		ctx := NewExecutionContext().SetFloatField("x", 3.4)
		result, _ := f.Execute(ctx)
		assert.True(t, result)
	})

	t.Run("wrong type", func(t *testing.T) {
		f, _ := Compile(`abs(name) == 0`, nil)
		ctx := NewExecutionContext().SetStringField("name", "test")
		result, _ := f.Execute(ctx)
		assert.False(t, result)
	})
}

func TestFnIPClassification(t *testing.T) {
	t.Run("is_ipv4 true", func(t *testing.T) {
		f, _ := Compile(`is_ipv4(ip) == true`, nil)
		ctx := NewExecutionContext().SetIPField("ip", "192.168.1.1")
		result, _ := f.Execute(ctx)
		assert.True(t, result)
	})

	t.Run("is_ipv4 false for ipv6", func(t *testing.T) {
		f, _ := Compile(`is_ipv4(ip) == true`, nil)
		ctx := NewExecutionContext().SetIPField("ip", "2001:db8::1")
		result, _ := f.Execute(ctx)
		assert.False(t, result)
	})

	t.Run("is_ipv6 true", func(t *testing.T) {
		f, _ := Compile(`is_ipv6(ip) == true`, nil)
		ctx := NewExecutionContext().SetIPField("ip", "2001:db8::1")
		result, _ := f.Execute(ctx)
		assert.True(t, result)
	})

	t.Run("is_ipv6 false for ipv4", func(t *testing.T) {
		f, _ := Compile(`is_ipv6(ip) == true`, nil)
		ctx := NewExecutionContext().SetIPField("ip", "192.168.1.1")
		result, _ := f.Execute(ctx)
		assert.False(t, result)
	})

	t.Run("is_loopback ipv4", func(t *testing.T) {
		f, _ := Compile(`is_loopback(ip) == true`, nil)
		ctx := NewExecutionContext().SetIPField("ip", "127.0.0.1")
		result, _ := f.Execute(ctx)
		assert.True(t, result)
	})

	t.Run("is_loopback ipv6", func(t *testing.T) {
		f, _ := Compile(`is_loopback(ip) == true`, nil)
		ctx := NewExecutionContext().SetIPField("ip", "::1")
		result, _ := f.Execute(ctx)
		assert.True(t, result)
	})

	t.Run("is_loopback false", func(t *testing.T) {
		f, _ := Compile(`is_loopback(ip) == true`, nil)
		ctx := NewExecutionContext().SetIPField("ip", "8.8.8.8")
		result, _ := f.Execute(ctx)
		assert.False(t, result)
	})

	t.Run("nil arg", func(t *testing.T) {
		f, _ := Compile(`is_ipv4(missing) == true`, nil)
		ctx := NewExecutionContext()
		result, _ := f.Execute(ctx)
		assert.False(t, result)
	})

	t.Run("wrong type", func(t *testing.T) {
		f, _ := Compile(`is_ipv4(name) == true`, nil)
		ctx := NewExecutionContext().SetStringField("name", "192.168.1.1")
		result, _ := f.Execute(ctx)
		assert.False(t, result)
	})
}

func TestFnRegexExtract(t *testing.T) {
	t.Run("extract match", func(t *testing.T) {
		f, _ := Compile(`regex_extract(path, "/api/v([0-9]+)/") == "/api/v2/"`, nil)
		require.NotNil(t, f)
		ctx := NewExecutionContext().SetStringField("path", "/api/v2/users")
		result, _ := f.Execute(ctx)
		assert.True(t, result)
	})

	t.Run("no match", func(t *testing.T) {
		f, _ := Compile(`regex_extract(path, "/api/v([0-9]+)/") == ""`, nil)
		ctx := NewExecutionContext().SetStringField("path", "/home")
		result, _ := f.Execute(ctx)
		assert.True(t, result)
	})

	t.Run("wrong arg count", func(t *testing.T) {
		f, _ := Compile(`regex_extract(path) == ""`, nil)
		ctx := NewExecutionContext().SetStringField("path", "test")
		result, _ := f.Execute(ctx)
		assert.False(t, result)
	})

	t.Run("invalid regex", func(t *testing.T) {
		f, _ := Compile(`regex_extract(path, "[invalid") == ""`, nil)
		ctx := NewExecutionContext().SetStringField("path", "test")
		_, err := f.Execute(ctx)
		assert.Error(t, err)
	})
}

func TestFnSetOperations(t *testing.T) {
	t.Run("intersection", func(t *testing.T) {
		f, _ := Compile(`len(intersection(a, b)) == 2`, nil)
		ctx := NewExecutionContext().
			SetArrayField("a", []string{"x", "y", "z"}).
			SetArrayField("b", []string{"y", "z", "w"})
		result, _ := f.Execute(ctx)
		assert.True(t, result)
	})

	t.Run("intersection empty", func(t *testing.T) {
		f, _ := Compile(`len(intersection(a, b)) == 0`, nil)
		ctx := NewExecutionContext().
			SetArrayField("a", []string{"x"}).
			SetArrayField("b", []string{"y"})
		result, _ := f.Execute(ctx)
		assert.True(t, result)
	})

	t.Run("union", func(t *testing.T) {
		f, _ := Compile(`len(union(a, b)) == 3`, nil)
		ctx := NewExecutionContext().
			SetArrayField("a", []string{"x", "y"}).
			SetArrayField("b", []string{"y", "z"})
		result, _ := f.Execute(ctx)
		assert.True(t, result)
	})

	t.Run("difference", func(t *testing.T) {
		f, _ := Compile(`len(difference(a, b)) == 1`, nil)
		ctx := NewExecutionContext().
			SetArrayField("a", []string{"x", "y", "z"}).
			SetArrayField("b", []string{"y", "z"})
		result, _ := f.Execute(ctx)
		assert.True(t, result)
	})

	t.Run("wrong types", func(t *testing.T) {
		f, _ := Compile(`len(intersection(a, b)) == 0`, nil)
		ctx := NewExecutionContext().
			SetStringField("a", "hello").
			SetStringField("b", "world")
		result, _ := f.Execute(ctx)
		assert.False(t, result)
	})

	t.Run("wrong arg count", func(t *testing.T) {
		f, _ := Compile(`len(intersection(a)) == 0`, nil)
		ctx := NewExecutionContext().SetArrayField("a", []string{"x"})
		result, _ := f.Execute(ctx)
		assert.False(t, result)
	})
}

func TestFnContainsAnyAll(t *testing.T) {
	t.Run("contains_any true", func(t *testing.T) {
		f, _ := Compile(`contains_any(tags, roles) == true`, nil)
		ctx := NewExecutionContext().
			SetArrayField("tags", []string{"admin", "user"}).
			SetArrayField("roles", []string{"guest", "admin"})
		result, _ := f.Execute(ctx)
		assert.True(t, result)
	})

	t.Run("contains_any false", func(t *testing.T) {
		f, _ := Compile(`contains_any(tags, roles) == true`, nil)
		ctx := NewExecutionContext().
			SetArrayField("tags", []string{"admin", "user"}).
			SetArrayField("roles", []string{"guest", "visitor"})
		result, _ := f.Execute(ctx)
		assert.False(t, result)
	})

	t.Run("contains_all true", func(t *testing.T) {
		f, _ := Compile(`contains_all(tags, required) == true`, nil)
		ctx := NewExecutionContext().
			SetArrayField("tags", []string{"admin", "user", "guest"}).
			SetArrayField("required", []string{"admin", "user"})
		result, _ := f.Execute(ctx)
		assert.True(t, result)
	})

	t.Run("contains_all false", func(t *testing.T) {
		f, _ := Compile(`contains_all(tags, required) == true`, nil)
		ctx := NewExecutionContext().
			SetArrayField("tags", []string{"admin"}).
			SetArrayField("required", []string{"admin", "user"})
		result, _ := f.Execute(ctx)
		assert.False(t, result)
	})

	t.Run("wrong arg count", func(t *testing.T) {
		f, _ := Compile(`contains_any(tags) == true`, nil)
		ctx := NewExecutionContext().SetArrayField("tags", []string{"x"})
		result, _ := f.Execute(ctx)
		assert.False(t, result)
	})
}

func TestUserDefinedFunctions(t *testing.T) {
	t.Run("simple bool function", func(t *testing.T) {
		filter, err := Compile(`maintenance() == true`, nil)
		require.NoError(t, err)

		ctx := NewExecutionContext().
			SetFunc("maintenance", func(_ []Value) (Value, error) {
				return BoolValue(true), nil
			})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("function with string arg returning float", func(t *testing.T) {
		filter, err := Compile(`get_score(domain) > 5.0`, nil)
		require.NoError(t, err)

		scores := map[string]float64{"example.com": 8.5, "spam.com": 2.0}
		ctx := NewExecutionContext().
			SetStringField("domain", "example.com").
			SetFunc("get_score", func(args []Value) (Value, error) {
				domain := string(args[0].(StringValue))
				return FloatValue(scores[domain]), nil
			})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("function with IP arg returning bool", func(t *testing.T) {
		filter, err := Compile(`is_tor(src.ip) == true`, nil)
		require.NoError(t, err)

		ctx := NewExecutionContext().
			SetIPField("src.ip", "1.2.3.4").
			SetFunc("is_tor", func(_ []Value) (Value, error) {
				return BoolValue(true), nil
			})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("function returning array for in operator", func(t *testing.T) {
		filter, err := Compile(`ip.src in get_allowed_ips(zone)`, nil)
		require.NoError(t, err)

		ctx := NewExecutionContext().
			SetIPField("ip.src", "10.0.0.5").
			SetStringField("zone", "office").
			SetFunc("get_allowed_ips", func(_ []Value) (Value, error) {
				return ArrayValue{
					CIDRValue{IPNet: mustParseCIDR("10.0.0.0/8")},
				}, nil
			})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("function returning CIDR for in operator", func(t *testing.T) {
		filter, err := Compile(`ip.src in get_network(zone)`, nil)
		require.NoError(t, err)

		ctx := NewExecutionContext().
			SetIPField("ip.src", "192.168.1.50").
			SetStringField("zone", "lan").
			SetFunc("get_network", func(_ []Value) (Value, error) {
				return CIDRValue{IPNet: mustParseCIDR("192.168.0.0/16")}, nil
			})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("function not registered returns nil", func(t *testing.T) {
		filter, err := Compile(`unknown_func("test") == "x"`, nil)
		require.NoError(t, err)

		ctx := NewExecutionContext()
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("function returning error", func(t *testing.T) {
		filter, err := Compile(`failing() == true`, nil)
		require.NoError(t, err)

		ctx := NewExecutionContext().
			SetFunc("failing", func(_ []Value) (Value, error) {
				return nil, fmt.Errorf("database unavailable")
			})
		_, err = filter.Execute(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database unavailable")
	})

	t.Run("combined with arithmetic", func(t *testing.T) {
		filter, err := Compile(`get_score(domain) * 2 > 10`, nil)
		require.NoError(t, err)

		ctx := NewExecutionContext().
			SetStringField("domain", "test.com").
			SetFunc("get_score", func(_ []Value) (Value, error) {
				return FloatValue(6.0), nil
			})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("combined with logical operators", func(t *testing.T) {
		filter, err := Compile(`is_tor(ip) and get_score(domain) < 3.0`, nil)
		require.NoError(t, err)

		ctx := NewExecutionContext().
			SetIPField("ip", "1.2.3.4").
			SetStringField("domain", "spam.com").
			SetFunc("is_tor", func(_ []Value) (Value, error) {
				return BoolValue(true), nil
			}).
			SetFunc("get_score", func(_ []Value) (Value, error) {
				return FloatValue(1.5), nil
			})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("schema validates registered function", func(t *testing.T) {
		schema := NewSchema().
			AddField("domain", TypeString).
			RegisterFunction("get_score", TypeFloat, []Type{TypeString})

		_, err := Compile(`get_score(domain) > 5.0`, schema)
		assert.NoError(t, err)
	})

	t.Run("schema rejects unregistered function in allowlist mode", func(t *testing.T) {
		schema := NewSchema().
			AddField("domain", TypeString).
			SetFunctionMode(FunctionModeAllowlist).
			RegisterFunction("get_score", TypeFloat, nil)

		_, err := Compile(`get_score(domain) > 5.0`, schema)
		assert.NoError(t, err)

		_, err = Compile(`unknown(domain) > 5.0`, schema)
		assert.Error(t, err)
	})

	t.Run("schema validates argument count", func(t *testing.T) {
		schema := NewSchema().
			AddField("domain", TypeString).
			RegisterFunction("get_score", TypeFloat, []Type{TypeString})

		_, err := Compile(`get_score(domain, domain) > 5.0`, schema)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expects 1 arguments, got 2")
	})

	t.Run("schema validates argument type", func(t *testing.T) {
		schema := NewSchema().
			AddField("domain", TypeString).
			AddField("count", TypeInt).
			RegisterFunction("get_score", TypeFloat, []Type{TypeString})

		_, err := Compile(`get_score(count) > 5.0`, schema)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expected String, got Int")
	})

	t.Run("schema allows nil arg types (skip validation)", func(t *testing.T) {
		schema := NewSchema().
			RegisterFunction("maintenance", TypeBool, nil)

		_, err := Compile(`maintenance() == true`, schema)
		assert.NoError(t, err)
	})

	t.Run("no args function", func(t *testing.T) {
		filter, err := Compile(`maintenance()`, nil)
		require.NoError(t, err)

		ctx := NewExecutionContext().
			SetFunc("maintenance", func(_ []Value) (Value, error) {
				return BoolValue(false), nil
			})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})
}

func mustParseCIDR(s string) *net.IPNet {
	_, ipNet, err := net.ParseCIDR(s)
	if err != nil {
		panic(err)
	}
	return ipNet
}

func TestExists(t *testing.T) {
	t.Run("field exists", func(t *testing.T) {
		filter, _ := Compile(`exists(name)`, nil)
		ctx := NewExecutionContext().SetStringField("name", "hello")
		result, _ := filter.Execute(ctx)
		assert.True(t, result)
	})

	t.Run("field missing", func(t *testing.T) {
		filter, _ := Compile(`exists(name)`, nil)
		ctx := NewExecutionContext()
		result, _ := filter.Execute(ctx)
		assert.False(t, result)
	})

	t.Run("field with empty string exists", func(t *testing.T) {
		filter, _ := Compile(`exists(name)`, nil)
		ctx := NewExecutionContext().SetStringField("name", "")
		result, _ := filter.Execute(ctx)
		assert.True(t, result)
	})

	t.Run("field with zero int exists", func(t *testing.T) {
		filter, _ := Compile(`exists(count)`, nil)
		ctx := NewExecutionContext().SetIntField("count", 0)
		result, _ := filter.Execute(ctx)
		assert.True(t, result)
	})

	t.Run("combined with logic", func(t *testing.T) {
		filter, _ := Compile(`exists(referer) and name == "test"`, nil)
		ctx := NewExecutionContext().
			SetStringField("name", "test").
			SetStringField("referer", "https://example.com")
		result, _ := filter.Execute(ctx)
		assert.True(t, result)

		ctx2 := NewExecutionContext().SetStringField("name", "test")
		result2, _ := filter.Execute(ctx2)
		assert.False(t, result2)
	})

	t.Run("not exists", func(t *testing.T) {
		filter, _ := Compile(`not exists(referer)`, nil)
		ctx := NewExecutionContext()
		result, _ := filter.Execute(ctx)
		assert.True(t, result)
	})

	t.Run("wrong arg count", func(t *testing.T) {
		filter, _ := Compile(`exists(a, b)`, nil)
		ctx := NewExecutionContext().SetStringField("a", "x").SetStringField("b", "y")
		result, _ := filter.Execute(ctx)
		assert.False(t, result)
	})
}

func TestFnCoverageEdgeCases(t *testing.T) {
	t.Run("trim_left nil", func(t *testing.T) {
		f, _ := Compile(`trim_left(missing) == ""`, nil)
		ctx := NewExecutionContext()
		result, _ := f.Execute(ctx)
		assert.False(t, result)
	})

	t.Run("trim_left wrong type", func(t *testing.T) {
		f, _ := Compile(`trim_left(x) == ""`, nil)
		ctx := NewExecutionContext().SetIntField("x", 42)
		result, _ := f.Execute(ctx)
		assert.False(t, result)
	})

	t.Run("trim_right nil", func(t *testing.T) {
		f, _ := Compile(`trim_right(missing) == ""`, nil)
		ctx := NewExecutionContext()
		result, _ := f.Execute(ctx)
		assert.False(t, result)
	})

	t.Run("trim_right wrong type", func(t *testing.T) {
		f, _ := Compile(`trim_right(x) == ""`, nil)
		ctx := NewExecutionContext().SetIntField("x", 42)
		result, _ := f.Execute(ctx)
		assert.False(t, result)
	})

	t.Run("contains_word nil args", func(t *testing.T) {
		f, _ := Compile(`contains_word(missing, "test") == true`, nil)
		ctx := NewExecutionContext()
		result, _ := f.Execute(ctx)
		assert.False(t, result)
	})

	t.Run("contains_word wrong types", func(t *testing.T) {
		f, _ := Compile(`contains_word(x, "test") == true`, nil)
		ctx := NewExecutionContext().SetIntField("x", 42)
		result, _ := f.Execute(ctx)
		assert.False(t, result)
	})

	t.Run("regex_replace nil args", func(t *testing.T) {
		f, _ := Compile(`regex_replace(missing, "a", "b") == ""`, nil)
		ctx := NewExecutionContext()
		result, _ := f.Execute(ctx)
		assert.False(t, result)
	})
}

func TestFilterFunctions(t *testing.T) {
	t.Run("function lower", func(t *testing.T) {
		filter, err := Compile(`lower(name) == "hello"`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetStringField("name", "HELLO")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
		ctx2 := NewExecutionContext().SetStringField("name", "World")
		result2, err := filter.Execute(ctx2)
		assert.NoError(t, err)
		assert.False(t, result2)
	})

	t.Run("function upper", func(t *testing.T) {
		filter, err := Compile(`upper(name) == "HELLO"`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetStringField("name", "hello")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("function len - string", func(t *testing.T) {
		filter, err := Compile(`len(name) == 5`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetStringField("name", "hello")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("function len - array", func(t *testing.T) {
		filter, err := Compile(`len(tags) == 3`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetArrayField("tags", []string{"a", "b", "c"})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("function len - map", func(t *testing.T) {
		filter, err := Compile(`len(attrs) == 2`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetMapField("attrs", map[string]string{"a": "1", "b": "2"})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("function starts_with", func(t *testing.T) {
		filter, err := Compile(`starts_with(path, "/api/")`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetStringField("path", "/api/users")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
		ctx2 := NewExecutionContext().SetStringField("path", "/web/users")
		result2, err := filter.Execute(ctx2)
		assert.NoError(t, err)
		assert.False(t, result2)
	})

	t.Run("function ends_with", func(t *testing.T) {
		filter, err := Compile(`ends_with(file, ".json")`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetStringField("file", "data.json")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
		ctx2 := NewExecutionContext().SetStringField("file", "data.xml")
		result2, err := filter.Execute(ctx2)
		assert.NoError(t, err)
		assert.False(t, result2)
	})

	t.Run("function any", func(t *testing.T) {
		filter, err := Compile(`any(tags[*] == "admin")`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetArrayField("tags", []string{"user", "admin", "guest"})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
		ctx2 := NewExecutionContext().SetArrayField("tags", []string{"user", "guest"})
		result2, err := filter.Execute(ctx2)
		assert.NoError(t, err)
		assert.False(t, result2)
	})

	t.Run("function all", func(t *testing.T) {
		filter, err := Compile(`all(ports[*] > 0)`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetIntArrayField("ports", []int64{80, 443, 8080})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
		ctx2 := NewExecutionContext().SetIntArrayField("ports", []int64{80, 0, 443})
		result2, err := filter.Execute(ctx2)
		assert.NoError(t, err)
		assert.False(t, result2)
	})

	t.Run("function all - empty array", func(t *testing.T) {
		filter, err := Compile(`all(tags[*] == "admin")`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetArrayField("tags", []string{})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("function concat", func(t *testing.T) {
		filter, err := Compile(`concat(scheme, "://", host) == "https://example.com"`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().
			SetStringField("scheme", "https").
			SetStringField("host", "example.com")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("function substring - with end", func(t *testing.T) {
		filter, err := Compile(`substring(path, 0, 4) == "/api"`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetStringField("path", "/api/users")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("function substring - without end", func(t *testing.T) {
		filter, err := Compile(`substring(path, 4) == "/users"`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetStringField("path", "/api/users")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("function split", func(t *testing.T) {
		filter, err := Compile(`split(header, ",")[0] == "value1"`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetStringField("header", "value1,value2,value3")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("function join", func(t *testing.T) {
		filter, err := Compile(`join(tags, ",") == "a,b,c"`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetArrayField("tags", []string{"a", "b", "c"})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("function has_key", func(t *testing.T) {
		filter, err := Compile(`has_key(attrs, "region")`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetMapField("attrs", map[string]string{"region": "us-west"})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
		ctx2 := NewExecutionContext().SetMapField("attrs", map[string]string{"zone": "a"})
		result2, err := filter.Execute(ctx2)
		assert.NoError(t, err)
		assert.False(t, result2)
	})

	t.Run("function has_value", func(t *testing.T) {
		filter, err := Compile(`has_value(tags, "admin")`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetArrayField("tags", []string{"user", "admin"})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
		ctx2 := NewExecutionContext().SetArrayField("tags", []string{"user", "guest"})
		result2, err := filter.Execute(ctx2)
		assert.NoError(t, err)
		assert.False(t, result2)
	})

	t.Run("function url_decode", func(t *testing.T) {
		filter, err := Compile(`url_decode(query) contains "admin"`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetStringField("query", "user%3Dadmin%26role%3Dsuper")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("function case insensitive", func(t *testing.T) {
		filter, err := Compile(`LOWER(name) == "hello"`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetStringField("name", "HELLO")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("function nested", func(t *testing.T) {
		filter, err := Compile(`len(lower(name)) == 5`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetStringField("name", "HELLO")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("function in expression", func(t *testing.T) {
		filter, err := Compile(`lower(name) == "hello" and len(name) == 5`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetStringField("name", "HELLO")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("function unknown", func(t *testing.T) {
		filter, err := Compile(`unknown_func(name) == "hello"`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetStringField("name", "hello")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("function with nil argument", func(t *testing.T) {
		filter, err := Compile(`lower(missing) == "hello"`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext()
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("function schema validation", func(t *testing.T) {
		schema := NewSchema().AddField("name", TypeString)
		_, err := Compile(`lower(name) == "hello"`, schema)
		assert.NoError(t, err)
		_, err = Compile(`lower(unknown) == "hello"`, schema)
		assert.Error(t, err)
	})

	t.Run("function empty arguments", func(t *testing.T) {
		filter, err := Compile(`concat() == ""`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext()
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("function lower with wrong type", func(t *testing.T) {
		filter, err := Compile(`lower(count) == "hello"`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetIntField("count", 123)
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("function substring edge cases", func(t *testing.T) {
		filter, err := Compile(`substring(name, 100) == ""`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetStringField("name", "hello")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
		filter2, err := Compile(`substring(name, -5, 3) == "hel"`, nil)
		assert.NoError(t, err)
		result2, err := filter2.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result2)
	})

	t.Run("function url_decode invalid", func(t *testing.T) {
		filter, err := Compile(`url_decode(query) == "%invalid"`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetStringField("query", "%invalid")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("function all with contains", func(t *testing.T) {
		filter, err := Compile(`all(emails[*] contains "@")`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetArrayField("emails", []string{"a@b.com", "c@d.com"})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
		ctx2 := NewExecutionContext().SetArrayField("emails", []string{"a@b.com", "invalid"})
		result2, err := filter.Execute(ctx2)
		assert.NoError(t, err)
		assert.False(t, result2)
	})

	t.Run("function parsing with multiple args", func(t *testing.T) {
		filter, err := Compile(`concat("a", "b", "c") == "abc"`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext()
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("function cidr - IPv4", func(t *testing.T) {
		filter, err := Compile(`cidr(ip, 24) == 192.168.1.0/24`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetIPField("ip", "192.168.1.100")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
		ctx2 := NewExecutionContext().SetIPField("ip", "192.168.2.100")
		result2, err := filter.Execute(ctx2)
		assert.NoError(t, err)
		assert.False(t, result2)
	})

	t.Run("function cidr - IPv4 /16", func(t *testing.T) {
		filter, err := Compile(`cidr(ip, 16) == 192.168.0.0/16`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetIPField("ip", "192.168.100.50")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("function cidr - IPv6 returns nil", func(t *testing.T) {
		filter, err := Compile(`cidr(ip, 24) == 2001:db8::/24`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetIPField("ip", "2001:db8::1234")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("function cidr - edge cases", func(t *testing.T) {
		filter, err := Compile(`cidr(ip, 32) == 192.168.1.100/32`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetIPField("ip", "192.168.1.100")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
		filter2, err := Compile(`cidr(ip, 0) == 0.0.0.0/0`, nil)
		assert.NoError(t, err)
		result2, err := filter2.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result2)
	})

	t.Run("function cidr - wrong types", func(t *testing.T) {
		filter, err := Compile(`cidr(name, 24) == 192.168.1.0/24`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetStringField("name", "not an ip")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("function cidr6 - IPv4", func(t *testing.T) {
		filter, err := Compile(`cidr6(ip, 24) == 192.168.1.0/24`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetIPField("ip", "192.168.1.100")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("function cidr6 - IPv4 with bits > 32", func(t *testing.T) {
		filter, err := Compile(`cidr6(ip, 64) == 192.168.1.100/32`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetIPField("ip", "192.168.1.100")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("function cidr6 - IPv6", func(t *testing.T) {
		filter, err := Compile(`cidr6(ip, 64) == 2001:db8::/64`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetIPField("ip", "2001:db8::abcd:1234")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("function cidr6 - wrong types", func(t *testing.T) {
		filter, err := Compile(`cidr6(name, 64) == 2001:db8::/64`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetStringField("name", "not an ip")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("function cidr - nil arguments", func(t *testing.T) {
		filter, err := Compile(`cidr(ip, 24) == 192.168.1.0/24`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext()
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("function cidr6 - nil arguments", func(t *testing.T) {
		filter, err := Compile(`cidr6(ip, 64) == 2001:db8::/64`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext()
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("function upper with wrong type", func(t *testing.T) {
		filter, err := Compile(`upper(count) == "TEST"`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetIntField("count", 123)
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("function upper with nil", func(t *testing.T) {
		filter, err := Compile(`upper(missing) == "TEST"`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext()
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("function len with bytes", func(t *testing.T) {
		filter, err := Compile(`len(data) == 5`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetField("data", BytesValue([]byte("hello")))
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("function len with nil", func(t *testing.T) {
		filter, err := Compile(`len(missing) == 0`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext()
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("function len with wrong type", func(t *testing.T) {
		filter, err := Compile(`len(flag) == 0`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetBoolField("flag", true)
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("function starts_with with nil source", func(t *testing.T) {
		filter, err := Compile(`starts_with(missing, "test")`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext()
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("function starts_with with wrong type", func(t *testing.T) {
		filter, err := Compile(`starts_with(count, "test")`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetIntField("count", 123)
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("function ends_with with nil source", func(t *testing.T) {
		filter, err := Compile(`ends_with(missing, "test")`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext()
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("function ends_with with wrong type", func(t *testing.T) {
		filter, err := Compile(`ends_with(count, "test")`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetIntField("count", 123)
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("function any with non-expression", func(t *testing.T) {
		filter, err := Compile(`any(flag)`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetBoolField("flag", true)
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("function any with nil result", func(t *testing.T) {
		filter, err := Compile(`any(missing)`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext()
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("function all with non-binary expression", func(t *testing.T) {
		filter, err := Compile(`all(flag)`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetBoolField("flag", true)
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("function all with nil result", func(t *testing.T) {
		filter, err := Compile(`all(missing)`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext()
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("function all with ne operator", func(t *testing.T) {
		filter, err := Compile(`all(tags[*] != "banned")`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetArrayField("tags", []string{"a", "b", "c"})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
		ctx2 := NewExecutionContext().SetArrayField("tags", []string{"a", "banned", "c"})
		result2, err := filter.Execute(ctx2)
		assert.NoError(t, err)
		assert.False(t, result2)
	})

	t.Run("function all with in operator", func(t *testing.T) {
		filter, err := Compile(`all(tags[*] in {"a", "b", "c"})`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetArrayField("tags", []string{"a", "b"})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
		ctx2 := NewExecutionContext().SetArrayField("tags", []string{"a", "x"})
		result2, err := filter.Execute(ctx2)
		assert.NoError(t, err)
		assert.False(t, result2)
	})

	t.Run("function all with matches operator", func(t *testing.T) {
		filter, err := Compile(`all(tags[*] matches "^[a-z]+$")`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetArrayField("tags", []string{"abc", "def"})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
		ctx2 := NewExecutionContext().SetArrayField("tags", []string{"abc", "123"})
		result2, err := filter.Execute(ctx2)
		assert.NoError(t, err)
		assert.False(t, result2)
	})

	t.Run("function concat with nil args", func(t *testing.T) {
		filter, err := Compile(`concat(a, b, c) == "ac"`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().
			SetStringField("a", "a").
			SetStringField("c", "c")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("function concat with non-string", func(t *testing.T) {
		filter, err := Compile(`concat(name, count) == "test123"`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().
			SetStringField("name", "test").
			SetIntField("count", 123)
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("function substring with nil", func(t *testing.T) {
		filter, err := Compile(`substring(missing, 0, 4) == ""`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext()
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("function substring with wrong type", func(t *testing.T) {
		filter, err := Compile(`substring(count, 0, 4) == ""`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetIntField("count", 123)
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("function substring end less than start", func(t *testing.T) {
		filter, err := Compile(`substring(name, 5, 2) == ""`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetStringField("name", "hello world")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("function substring end beyond length", func(t *testing.T) {
		filter, err := Compile(`substring(name, 0, 100) == "hello"`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetStringField("name", "hello")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("function split with nil", func(t *testing.T) {
		filter, err := Compile(`split(missing, ",")[0] == ""`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext()
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("function split with wrong type", func(t *testing.T) {
		filter, err := Compile(`split(count, ",")[0] == ""`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetIntField("count", 123)
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("function join with nil array", func(t *testing.T) {
		filter, err := Compile(`join(missing, ",") == ""`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext()
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("function join with wrong type", func(t *testing.T) {
		filter, err := Compile(`join(name, ",") == ""`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetStringField("name", "test")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("function join with nil elements", func(t *testing.T) {
		filter, err := Compile(`join(tags, ",")`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetField("tags", ArrayValue{
			StringValue("a"),
			nil,
			StringValue("c"),
		})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("function join with non-string elements", func(t *testing.T) {
		filter, err := Compile(`join(items, ",") == "1,2,3"`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetIntArrayField("items", []int64{1, 2, 3})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("function has_key with nil", func(t *testing.T) {
		filter, err := Compile(`has_key(missing, "key")`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext()
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("function has_key with wrong type", func(t *testing.T) {
		filter, err := Compile(`has_key(name, "key")`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetStringField("name", "test")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("function has_value with nil", func(t *testing.T) {
		filter, err := Compile(`has_value(missing, "value")`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext()
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("function has_value with wrong type", func(t *testing.T) {
		filter, err := Compile(`has_value(name, "value")`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetStringField("name", "test")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("function url_decode with nil", func(t *testing.T) {
		filter, err := Compile(`url_decode(missing) == ""`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext()
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("function url_decode with wrong type", func(t *testing.T) {
		filter, err := Compile(`url_decode(count) == ""`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetIntField("count", 123)
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("cidr with out of range bits", func(t *testing.T) {
		filter, err := Compile(`cidr(ip, 50) == 192.168.1.100/32`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetIPField("ip", "192.168.1.100")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("cidr with negative bits", func(t *testing.T) {
		filter, err := Compile(`cidr(ip, 0) == 0.0.0.0/0`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetIPField("ip", "192.168.1.100")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("cidr6 with IPv6 negative bits", func(t *testing.T) {
		filter, err := Compile(`cidr6(ip, 0) == "::/0"`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetIPField("ip", "2001:db8::1")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("cidr6 with IPv6 max bits", func(t *testing.T) {
		filter, err := Compile(`cidr6(ip, 128) == 2001:db8::1/128`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetIPField("ip", "2001:db8::1")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("any with nil result", func(t *testing.T) {
		filter, err := Compile(`any(missing == "x")`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext()
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("all with non-binary expression coverage", func(t *testing.T) {
		filter, err := Compile(`all(active)`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetBoolField("active", true)
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
		ctx2 := NewExecutionContext()
		result2, err := filter.Execute(ctx2)
		assert.NoError(t, err)
		assert.False(t, result2)
	})

	t.Run("all with empty unpacked array", func(t *testing.T) {
		filter, err := Compile(`all(tags[*] == "a")`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetField("tags", ArrayValue{})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("all with contains operator coverage", func(t *testing.T) {
		filter, err := Compile(`all(tags[*] contains "a")`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetArrayField("tags", []string{"apple", "avocado"})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
		ctx2 := NewExecutionContext().SetArrayField("tags", []string{"apple", "berry"})
		result2, err := filter.Execute(ctx2)
		assert.NoError(t, err)
		assert.False(t, result2)
	})

	t.Run("all with matches operator coverage", func(t *testing.T) {
		filter, err := Compile(`all(tags[*] matches "^a")`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetArrayField("tags", []string{"apple", "avocado"})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("all with in operator coverage", func(t *testing.T) {
		filter, err := Compile(`all(ports[*] in {80, 443})`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetIntArrayField("ports", []int64{80, 443})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
		ctx2 := NewExecutionContext().SetIntArrayField("ports", []int64{80, 8080})
		result2, err := filter.Execute(ctx2)
		assert.NoError(t, err)
		assert.False(t, result2)
	})

	t.Run("all with comparison operators coverage", func(t *testing.T) {
		filter, err := Compile(`all(vals[*] > 0)`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetIntArrayField("vals", []int64{1, 2, 3})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
		ctx2 := NewExecutionContext().SetIntArrayField("vals", []int64{0, 1, 2})
		result2, err := filter.Execute(ctx2)
		assert.NoError(t, err)
		assert.False(t, result2)
	})

	t.Run("all with lt operator", func(t *testing.T) {
		filter, err := Compile(`all(vals[*] < 10)`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetIntArrayField("vals", []int64{1, 5, 9})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("all with le operator", func(t *testing.T) {
		filter, err := Compile(`all(vals[*] <= 10)`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetIntArrayField("vals", []int64{1, 5, 10})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("all with ge operator", func(t *testing.T) {
		filter, err := Compile(`all(vals[*] >= 0)`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetIntArrayField("vals", []int64{0, 1, 2})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("all with ne operator coverage", func(t *testing.T) {
		filter, err := Compile(`all(tags[*] != "bad")`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetArrayField("tags", []string{"good", "ok"})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
		ctx2 := NewExecutionContext().SetArrayField("tags", []string{"good", "bad"})
		result2, err := filter.Execute(ctx2)
		assert.NoError(t, err)
		assert.False(t, result2)
	})
}
