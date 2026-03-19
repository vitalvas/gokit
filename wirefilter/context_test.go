package wirefilter

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecutionContext(t *testing.T) {
	t.Run("set and get string field", func(t *testing.T) {
		ctx := NewExecutionContext().SetStringField("name", "test")
		val, ok := ctx.GetField("name")
		assert.True(t, ok)
		assert.Equal(t, StringValue("test"), val)
	})

	t.Run("set and get int field", func(t *testing.T) {
		ctx := NewExecutionContext().SetIntField("count", 42)
		val, ok := ctx.GetField("count")
		assert.True(t, ok)
		assert.Equal(t, IntValue(42), val)
	})

	t.Run("set and get bool field", func(t *testing.T) {
		ctx := NewExecutionContext().SetBoolField("active", true)
		val, ok := ctx.GetField("active")
		assert.True(t, ok)
		assert.Equal(t, BoolValue(true), val)
	})

	t.Run("set and get float field", func(t *testing.T) {
		ctx := NewExecutionContext().SetFloatField("score", 3.14)
		val, ok := ctx.GetField("score")
		assert.True(t, ok)
		assert.Equal(t, TypeFloat, val.Type())
		assert.Equal(t, FloatValue(3.14), val)
	})

	t.Run("set and get IP field", func(t *testing.T) {
		ctx := NewExecutionContext().SetIPField("ip", "192.168.1.1")
		val, ok := ctx.GetField("ip")
		assert.True(t, ok)
		assert.Equal(t, TypeIP, val.Type())
	})

	t.Run("set IP field with invalid IP", func(t *testing.T) {
		ctx := NewExecutionContext().SetIPField("ip", "invalid")
		_, ok := ctx.GetField("ip")
		assert.False(t, ok)
	})

	t.Run("set and get bytes field", func(t *testing.T) {
		ctx := NewExecutionContext().SetBytesField("data", []byte{1, 2, 3})
		val, ok := ctx.GetField("data")
		assert.True(t, ok)
		assert.Equal(t, BytesValue([]byte{1, 2, 3}), val)
	})

	t.Run("set and get map field", func(t *testing.T) {
		ctx := NewExecutionContext().SetMapField("headers", map[string]string{"host": "example.com"})
		val, ok := ctx.GetField("headers")
		assert.True(t, ok)
		assert.Equal(t, TypeMap, val.Type())
		m := val.(MapValue)
		v, ok := m.Get("host")
		assert.True(t, ok)
		assert.Equal(t, StringValue("example.com"), v)
	})

	t.Run("set and get map field values", func(t *testing.T) {
		ctx := NewExecutionContext().SetMapFieldValues("data", map[string]Value{
			"count": IntValue(5),
		})
		val, ok := ctx.GetField("data")
		assert.True(t, ok)
		m := val.(MapValue)
		v, ok := m.Get("count")
		assert.True(t, ok)
		assert.Equal(t, IntValue(5), v)
	})

	t.Run("set and get generic field", func(t *testing.T) {
		ctx := NewExecutionContext().SetField("custom", StringValue("value"))
		val, ok := ctx.GetField("custom")
		assert.True(t, ok)
		assert.Equal(t, StringValue("value"), val)
	})

	t.Run("get missing field", func(t *testing.T) {
		ctx := NewExecutionContext()
		_, ok := ctx.GetField("missing")
		assert.False(t, ok)
	})

	t.Run("set and get array field", func(t *testing.T) {
		ctx := NewExecutionContext().SetArrayField("tags", []string{"a", "b"})
		val, ok := ctx.GetField("tags")
		assert.True(t, ok)
		arr := val.(ArrayValue)
		assert.Len(t, arr, 2)
		assert.Equal(t, StringValue("a"), arr[0])
		assert.Equal(t, StringValue("b"), arr[1])
	})

	t.Run("set and get int array field", func(t *testing.T) {
		ctx := NewExecutionContext().SetIntArrayField("ports", []int64{80, 443})
		val, ok := ctx.GetField("ports")
		assert.True(t, ok)
		arr := val.(ArrayValue)
		assert.Len(t, arr, 2)
		assert.Equal(t, IntValue(80), arr[0])
		assert.Equal(t, IntValue(443), arr[1])
	})

	t.Run("set and get string list", func(t *testing.T) {
		ctx := NewExecutionContext().SetList("roles", []string{"admin", "user"})
		list, ok := ctx.GetList("roles")
		assert.True(t, ok)
		assert.Len(t, list, 2)
		assert.Equal(t, StringValue("admin"), list[0])
		assert.Equal(t, StringValue("user"), list[1])
	})

	t.Run("set and get IP list with plain IPs", func(t *testing.T) {
		ctx := NewExecutionContext().SetIPList("ips", []string{"10.0.0.1", "192.168.1.1"})
		list, ok := ctx.GetList("ips")
		assert.True(t, ok)
		assert.Len(t, list, 2)
		assert.Equal(t, TypeIP, list[0].Type())
		assert.Equal(t, TypeIP, list[1].Type())
	})

	t.Run("set and get IP list with CIDR", func(t *testing.T) {
		ctx := NewExecutionContext().SetIPList("nets", []string{"10.0.0.0/8", "192.168.1.1"})
		list, ok := ctx.GetList("nets")
		assert.True(t, ok)
		assert.Len(t, list, 2)
		assert.Equal(t, TypeCIDR, list[0].Type())
		assert.Equal(t, TypeIP, list[1].Type())

		cidr := list[0].(CIDRValue)
		assert.True(t, cidr.Contains(net.ParseIP("10.50.0.1")))
		assert.False(t, cidr.Contains(net.ParseIP("192.168.0.1")))
	})

	t.Run("set IP list skips invalid entries", func(t *testing.T) {
		ctx := NewExecutionContext().SetIPList("ips", []string{"10.0.0.1", "invalid", "192.168.1.1"})
		list, ok := ctx.GetList("ips")
		assert.True(t, ok)
		assert.Len(t, list, 2)
	})

	t.Run("get missing list", func(t *testing.T) {
		ctx := NewExecutionContext()
		_, ok := ctx.GetList("missing")
		assert.False(t, ok)
	})

	t.Run("set and get table", func(t *testing.T) {
		ctx := NewExecutionContext().
			SetTable("geo", map[string]string{"10.0.0.1": "US", "8.8.8.8": "DE"})
		table, ok := ctx.GetTable("geo")
		assert.True(t, ok)
		assert.Len(t, table, 2)
		v, ok := table.Get("10.0.0.1")
		assert.True(t, ok)
		assert.Equal(t, StringValue("US"), v)
	})

	t.Run("set and get table values", func(t *testing.T) {
		ctx := NewExecutionContext().
			SetTableValues("limits", map[string]Value{
				"admin": IntValue(1000),
				"user":  IntValue(100),
			})
		table, ok := ctx.GetTable("limits")
		assert.True(t, ok)
		v, ok := table.Get("admin")
		assert.True(t, ok)
		assert.Equal(t, IntValue(1000), v)
	})

	t.Run("set and get table list", func(t *testing.T) {
		ctx := NewExecutionContext().
			SetTableList("roles_by_dept", map[string][]string{
				"eng":   {"dev", "sre"},
				"sales": {"account", "manager"},
			})
		table, ok := ctx.GetTable("roles_by_dept")
		assert.True(t, ok)
		v, ok := table.Get("eng")
		assert.True(t, ok)
		arr := v.(ArrayValue)
		assert.Len(t, arr, 2)
		assert.Equal(t, StringValue("dev"), arr[0])
	})

	t.Run("set and get table IP list", func(t *testing.T) {
		ctx := NewExecutionContext().
			SetTableIPList("nets_by_office", map[string][]string{
				"hq":     {"10.0.0.0/8"},
				"branch": {"192.168.1.0/24", "172.16.0.1"},
			})
		table, ok := ctx.GetTable("nets_by_office")
		assert.True(t, ok)
		v, ok := table.Get("branch")
		assert.True(t, ok)
		arr := v.(ArrayValue)
		assert.Len(t, arr, 2)
		assert.Equal(t, TypeCIDR, arr[0].Type())
		assert.Equal(t, TypeIP, arr[1].Type())
	})

	t.Run("get missing table", func(t *testing.T) {
		ctx := NewExecutionContext()
		_, ok := ctx.GetTable("missing")
		assert.False(t, ok)
	})

	t.Run("method chaining", func(t *testing.T) {
		ctx := NewExecutionContext().
			SetStringField("name", "test").
			SetIntField("count", 1).
			SetBoolField("active", true).
			SetIPField("ip", "10.0.0.1").
			SetList("roles", []string{"admin"}).
			SetIPList("nets", []string{"10.0.0.0/8"}).
			SetTable("geo", map[string]string{"10.0.0.1": "US"})

		_, ok := ctx.GetField("name")
		assert.True(t, ok)
		_, ok = ctx.GetField("count")
		assert.True(t, ok)
		_, ok = ctx.GetField("active")
		assert.True(t, ok)
		_, ok = ctx.GetField("ip")
		assert.True(t, ok)
		_, ok = ctx.GetList("roles")
		assert.True(t, ok)
		_, ok = ctx.GetList("nets")
		assert.True(t, ok)
		_, ok = ctx.GetTable("geo")
		assert.True(t, ok)
	})

	t.Run("set and get func", func(t *testing.T) {
		handler := func(_ []Value) (Value, error) {
			return BoolValue(true), nil
		}
		ctx := NewExecutionContext().SetFunc("is_admin", handler)
		fn, ok := ctx.GetFunc("is_admin")
		assert.True(t, ok)
		assert.NotNil(t, fn)

		result, err := fn(nil)
		assert.NoError(t, err)
		assert.Equal(t, BoolValue(true), result)
	})

	t.Run("set and get map array field with strings", func(t *testing.T) {
		ctx := NewExecutionContext().
			SetMapArrayField("headers", map[string][]Value{
				"Accept":       {StringValue("text/html"), StringValue("application/json")},
				"Content-Type": {StringValue("application/json")},
			})
		val, ok := ctx.GetField("headers")
		assert.True(t, ok)
		assert.Equal(t, TypeMap, val.Type())

		m := val.(MapValue)
		accept, ok := m.Get("Accept")
		assert.True(t, ok)
		arr := accept.(ArrayValue)
		assert.Len(t, arr, 2)
		assert.Equal(t, StringValue("text/html"), arr[0])
		assert.Equal(t, StringValue("application/json"), arr[1])
	})

	t.Run("set and get map array field with mixed types", func(t *testing.T) {
		_, ipNet, _ := net.ParseCIDR("10.0.0.0/8")
		ctx := NewExecutionContext().
			SetMapArrayField("rules", map[string][]Value{
				"ports":  {IntValue(80), IntValue(443)},
				"scores": {FloatValue(1.5), FloatValue(2.5)},
				"flags":  {BoolValue(true), BoolValue(false)},
				"nets":   {CIDRValue{IPNet: ipNet}},
			})
		val, ok := ctx.GetField("rules")
		assert.True(t, ok)

		m := val.(MapValue)
		ports, ok := m.Get("ports")
		assert.True(t, ok)
		arr := ports.(ArrayValue)
		assert.Len(t, arr, 2)
		assert.Equal(t, IntValue(80), arr[0])

		nets, ok := m.Get("nets")
		assert.True(t, ok)
		netArr := nets.(ArrayValue)
		assert.Equal(t, TypeCIDR, netArr[0].Type())
	})

	t.Run("get missing func", func(t *testing.T) {
		ctx := NewExecutionContext()
		_, ok := ctx.GetFunc("missing")
		assert.False(t, ok)
	})

	t.Run("with context", func(t *testing.T) {
		goCtx := context.Background()
		ctx := NewExecutionContext().WithContext(goCtx)
		assert.NoError(t, ctx.checkContext())
	})

	t.Run("with cancelled context", func(t *testing.T) {
		goCtx, cancel := context.WithCancel(context.Background())
		cancel()
		ctx := NewExecutionContext().WithContext(goCtx)
		assert.ErrorIs(t, ctx.checkContext(), context.Canceled)
	})

	t.Run("check context without context set", func(t *testing.T) {
		ctx := NewExecutionContext()
		assert.NoError(t, ctx.checkContext())
	})

	t.Run("enable trace", func(t *testing.T) {
		ctx := NewExecutionContext().EnableTrace()
		assert.True(t, ctx.traceEnabled())
		assert.NotNil(t, ctx.Trace())
		assert.Equal(t, "root", ctx.Trace().Expression)
	})

	t.Run("trace disabled by default", func(t *testing.T) {
		ctx := NewExecutionContext()
		assert.False(t, ctx.traceEnabled())
		assert.Nil(t, ctx.Trace())
	})

	t.Run("enable cache", func(t *testing.T) {
		ctx := NewExecutionContext().EnableCache()
		assert.Equal(t, defaultCacheMaxSize, ctx.cacheMaxSize)
		assert.Equal(t, 0, ctx.CacheLen())
	})

	t.Run("set cache max size", func(t *testing.T) {
		ctx := NewExecutionContext().EnableCache().SetCacheMaxSize(100)
		assert.Equal(t, 100, ctx.cacheMaxSize)
	})

	t.Run("set cache max size zero resets to default", func(t *testing.T) {
		ctx := NewExecutionContext().EnableCache().SetCacheMaxSize(0)
		assert.Equal(t, defaultCacheMaxSize, ctx.cacheMaxSize)
	})

	t.Run("reset cache", func(t *testing.T) {
		ctx := NewExecutionContext().EnableCache()
		ctx.setCache("key", StringValue("val"))
		assert.Equal(t, 1, ctx.CacheLen())
		ctx.ResetCache()
		assert.Equal(t, 0, ctx.CacheLen())
	})

	t.Run("cache get and set", func(t *testing.T) {
		ctx := NewExecutionContext().EnableCache()
		ctx.setCache("fn:arg", IntValue(42))
		val, ok := ctx.getCached("fn:arg")
		assert.True(t, ok)
		assert.Equal(t, IntValue(42), val)
	})

	t.Run("cache miss", func(t *testing.T) {
		ctx := NewExecutionContext().EnableCache()
		_, ok := ctx.getCached("missing")
		assert.False(t, ok)
	})

	t.Run("cache disabled get returns false", func(t *testing.T) {
		ctx := NewExecutionContext()
		_, ok := ctx.getCached("key")
		assert.False(t, ok)
	})

	t.Run("cache disabled set is no-op", func(t *testing.T) {
		ctx := NewExecutionContext()
		ctx.setCache("key", IntValue(1))
		assert.Equal(t, 0, ctx.CacheLen())
	})

	t.Run("cache respects max size", func(t *testing.T) {
		ctx := NewExecutionContext().EnableCache().SetCacheMaxSize(2)
		ctx.setCache("a", IntValue(1))
		ctx.setCache("b", IntValue(2))
		ctx.setCache("c", IntValue(3)) // should be dropped
		assert.Equal(t, 2, ctx.CacheLen())
		_, ok := ctx.getCached("a")
		assert.True(t, ok)
		_, ok = ctx.getCached("c")
		assert.False(t, ok)
	})

	t.Run("reset cache on nil cache", func(t *testing.T) {
		ctx := NewExecutionContext()
		ctx.ResetCache() // should not panic
		assert.Equal(t, 0, ctx.CacheLen())
	})

	t.Run("trace push and pop", func(t *testing.T) {
		ctx := NewExecutionContext().EnableTrace()
		ctx.pushTrace("a == 1")
		ctx.popTrace(BoolValue(true), 0)
		trace := ctx.Trace()
		assert.Len(t, trace.Children, 1)
		assert.Equal(t, "a == 1", trace.Children[0].Expression)
		assert.Equal(t, "true", trace.Children[0].Result)
	})

	t.Run("trace pop with nil result", func(t *testing.T) {
		ctx := NewExecutionContext().EnableTrace()
		ctx.pushTrace("missing")
		ctx.popTrace(nil, 0)
		assert.Nil(t, ctx.Trace().Children[0].Result)
	})

	t.Run("cache key", func(t *testing.T) {
		key1 := cacheKey("fn", []Value{StringValue("a"), IntValue(1)})
		key2 := cacheKey("fn", []Value{StringValue("a"), IntValue(1)})
		key3 := cacheKey("fn", []Value{StringValue("b"), IntValue(1)})
		assert.Equal(t, key1, key2)
		assert.NotEqual(t, key1, key3)
	})

	t.Run("cache key with nil", func(t *testing.T) {
		key := cacheKey("fn", []Value{nil})
		assert.Contains(t, key, "nil")
	})
}

func TestExecuteWithContext(t *testing.T) {
	t.Run("normal execution with context", func(t *testing.T) {
		filter, _ := Compile(`name == "test"`, nil)
		goCtx := context.Background()
		ctx := NewExecutionContext().
			WithContext(goCtx).
			SetStringField("name", "test")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("cancelled context returns error", func(t *testing.T) {
		filter, _ := Compile(`name == "test"`, nil)
		goCtx, cancel := context.WithCancel(context.Background())
		cancel() // cancel immediately
		ctx := NewExecutionContext().
			WithContext(goCtx).
			SetStringField("name", "test")
		_, err := filter.Execute(ctx)
		assert.Error(t, err)
		assert.ErrorIs(t, err, context.Canceled)
	})

	t.Run("timeout context", func(t *testing.T) {
		filter, _ := Compile(`name == "test"`, nil)
		goCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		ctx := NewExecutionContext().
			WithContext(goCtx).
			SetStringField("name", "test")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("timeout with slow UDF", func(t *testing.T) {
		filter, _ := Compile(`slow_func() == true`, nil)
		goCtx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()
		ctx := NewExecutionContext().
			WithContext(goCtx).
			SetFunc("slow_func", func(_ []Value) (Value, error) {
				time.Sleep(100 * time.Millisecond)
				return BoolValue(true), nil
			})
		// The UDF sleeps 100ms, context expires at 10ms.
		// After UDF returns, the next evaluate call detects the expired context.
		_, err := filter.Execute(ctx)
		assert.ErrorIs(t, err, context.DeadlineExceeded)
	})

	t.Run("no context is fine", func(t *testing.T) {
		filter, _ := Compile(`name == "test"`, nil)
		ctx := NewExecutionContext().SetStringField("name", "test")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})
}

func TestExecuteWithTrace(t *testing.T) {
	t.Run("simple trace", func(t *testing.T) {
		filter, _ := Compile(`name == "test"`, nil)
		ctx := NewExecutionContext().
			EnableTrace().
			SetStringField("name", "test")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)

		trace := ctx.Trace()
		assert.NotNil(t, trace)
		assert.Equal(t, "root", trace.Expression)
		assert.NotEmpty(t, trace.Children)
	})

	t.Run("trace shows sub-expressions", func(t *testing.T) {
		filter, _ := Compile(`name == "test" and status > 200`, nil)
		ctx := NewExecutionContext().
			EnableTrace().
			SetStringField("name", "test").
			SetIntField("status", 500)
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)

		trace := ctx.Trace()
		assert.NotNil(t, trace)
		// Root should have the top-level AND expression as a child
		assert.NotEmpty(t, trace.Children)
	})

	t.Run("trace without enable returns nil", func(t *testing.T) {
		ctx := NewExecutionContext()
		assert.Nil(t, ctx.Trace())
	})

	t.Run("trace has duration", func(t *testing.T) {
		filter, _ := Compile(`name == "test"`, nil)
		ctx := NewExecutionContext().
			EnableTrace().
			SetStringField("name", "test")
		_, _ = filter.Execute(ctx)

		trace := ctx.Trace()
		assert.NotNil(t, trace)
		if len(trace.Children) > 0 {
			assert.GreaterOrEqual(t, int64(trace.Children[0].Duration), int64(0))
		}
	})
}

func TestExecuteWithCache(t *testing.T) {
	t.Run("caches UDF results", func(t *testing.T) {
		callCount := 0
		filter, _ := Compile(`get_score(name) > 5 and get_score(name) < 100`, nil)
		ctx := NewExecutionContext().
			EnableCache().
			SetStringField("name", "test").
			SetFunc("get_score", func(_ []Value) (Value, error) {
				callCount++
				return FloatValue(50.0), nil
			})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
		// With caching, get_score("test") should only be called once
		assert.Equal(t, 1, callCount)
	})

	t.Run("without cache calls multiple times", func(t *testing.T) {
		callCount := 0
		filter, _ := Compile(`get_score(name) > 5 and get_score(name) < 100`, nil)
		ctx := NewExecutionContext().
			SetStringField("name", "test").
			SetFunc("get_score", func(_ []Value) (Value, error) {
				callCount++
				return FloatValue(50.0), nil
			})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
		assert.Equal(t, 2, callCount)
	})

	t.Run("different args not cached", func(t *testing.T) {
		callCount := 0
		filter, _ := Compile(`get_score(a) > 5 and get_score(b) > 5`, nil)
		ctx := NewExecutionContext().
			EnableCache().
			SetStringField("a", "foo").
			SetStringField("b", "bar").
			SetFunc("get_score", func(_ []Value) (Value, error) {
				callCount++
				return FloatValue(50.0), nil
			})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
		// Different args = different cache keys = 2 calls
		assert.Equal(t, 2, callCount)
	})

	t.Run("cache does not affect builtins", func(t *testing.T) {
		filter, _ := Compile(`lower(name) == "test" and lower(name) == "test"`, nil)
		ctx := NewExecutionContext().
			EnableCache().
			SetStringField("name", "TEST")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("cache max size limits entries", func(t *testing.T) {
		callCount := 0
		ctx := NewExecutionContext().
			EnableCache().
			SetCacheMaxSize(2).
			SetFunc("score", func(_ []Value) (Value, error) {
				callCount++
				return FloatValue(1.0), nil
			})

		// Fill cache with 2 entries
		f1, _ := Compile(`score("a") > 0`, nil)
		f2, _ := Compile(`score("b") > 0`, nil)
		f3, _ := Compile(`score("c") > 0`, nil)

		_, _ = f1.Execute(ctx)
		_, _ = f2.Execute(ctx)
		assert.Equal(t, 2, ctx.CacheLen())
		assert.Equal(t, 2, callCount)

		// Third entry should not be cached (cache full)
		_, _ = f3.Execute(ctx)
		assert.Equal(t, 2, ctx.CacheLen())
		assert.Equal(t, 3, callCount)

		// But "a" and "b" are still cached
		_, _ = f1.Execute(ctx)
		_, _ = f2.Execute(ctx)
		assert.Equal(t, 3, callCount) // no new calls
	})

	t.Run("cache reset clears entries", func(t *testing.T) {
		callCount := 0
		ctx := NewExecutionContext().
			EnableCache().
			SetFunc("score", func(_ []Value) (Value, error) {
				callCount++
				return FloatValue(1.0), nil
			})

		f, _ := Compile(`score("a") > 0`, nil)
		_, _ = f.Execute(ctx)
		assert.Equal(t, 1, callCount)
		assert.Equal(t, 1, ctx.CacheLen())

		ctx.ResetCache()
		assert.Equal(t, 0, ctx.CacheLen())

		_, _ = f.Execute(ctx)
		assert.Equal(t, 2, callCount) // called again after reset
	})

	t.Run("cache persists across rules", func(t *testing.T) {
		callCount := 0
		ctx := NewExecutionContext().
			EnableCache().
			SetStringField("domain", "test.com").
			SetFunc("get_score", func(_ []Value) (Value, error) {
				callCount++
				return FloatValue(8.0), nil
			})

		f1, _ := Compile(`get_score(domain) > 5`, nil)
		f2, _ := Compile(`get_score(domain) > 3`, nil)

		_, _ = f1.Execute(ctx)
		_, _ = f2.Execute(ctx)
		// Same function + same args across two rules = 1 call
		assert.Equal(t, 1, callCount)
	})

	t.Run("set cache max size zero uses default", func(t *testing.T) {
		ctx := NewExecutionContext().EnableCache().SetCacheMaxSize(0)
		assert.Equal(t, defaultCacheMaxSize, ctx.cacheMaxSize)
	})
}

func TestCacheCoverageEdgeCases(t *testing.T) {
	t.Run("cache key with nil arg", func(t *testing.T) {
		callCount := 0
		filter, _ := Compile(`get_val(missing) == get_val(missing)`, nil)
		ctx := NewExecutionContext().
			EnableCache().
			SetFunc("get_val", func(_ []Value) (Value, error) {
				callCount++
				return IntValue(1), nil
			})
		result, _ := filter.Execute(ctx)
		assert.True(t, result)
		assert.Equal(t, 1, callCount)
	})
}

func TestFilterContext(t *testing.T) {
	t.Run("context SetBytesField", func(t *testing.T) {
		ctx := NewExecutionContext().
			SetBytesField("data", []byte("test data"))

		val, ok := ctx.GetField("data")
		assert.True(t, ok)
		assert.Equal(t, TypeBytes, val.Type())
		assert.Equal(t, "test data", val.String())
	})

	t.Run("context SetArrayField", func(t *testing.T) {
		ctx := NewExecutionContext().
			SetArrayField("tags", []string{"a", "b", "c"})

		val, ok := ctx.GetField("tags")
		assert.True(t, ok)
		assert.Equal(t, TypeArray, val.Type())

		arr := val.(ArrayValue)
		assert.Len(t, arr, 3)
		assert.Equal(t, StringValue("a"), arr[0])
		assert.Equal(t, StringValue("b"), arr[1])
		assert.Equal(t, StringValue("c"), arr[2])
	})

	t.Run("context SetIntArrayField", func(t *testing.T) {
		ctx := NewExecutionContext().
			SetIntArrayField("ports", []int64{80, 443})

		val, ok := ctx.GetField("ports")
		assert.True(t, ok)
		assert.Equal(t, TypeArray, val.Type())

		arr := val.(ArrayValue)
		assert.Len(t, arr, 2)
		assert.Equal(t, IntValue(80), arr[0])
		assert.Equal(t, IntValue(443), arr[1])
	})

	t.Run("context GetList", func(t *testing.T) {
		ctx := NewExecutionContext().
			SetList("roles", []string{"admin", "user"})

		list, ok := ctx.GetList("roles")
		assert.True(t, ok)
		assert.Len(t, list, 2)
		assert.Equal(t, StringValue("admin"), list[0])

		_, ok = ctx.GetList("undefined")
		assert.False(t, ok)
	})

	t.Run("custom list - string list", func(t *testing.T) {
		schema := NewSchema().
			AddField("role", TypeString)

		filter, err := Compile(`role in $admin_roles`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetStringField("role", "superuser").
			SetList("admin_roles", []string{"admin", "superuser", "root"})

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)

		ctx2 := NewExecutionContext().
			SetStringField("role", "guest").
			SetList("admin_roles", []string{"admin", "superuser", "root"})

		result2, err := filter.Execute(ctx2)
		assert.NoError(t, err)
		assert.False(t, result2)
	})

	t.Run("custom list - undefined list", func(t *testing.T) {
		schema := NewSchema().
			AddField("role", TypeString)

		filter, err := Compile(`role in $undefined_list`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetStringField("role", "admin")

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("custom list - empty list", func(t *testing.T) {
		schema := NewSchema().
			AddField("role", TypeString)

		filter, err := Compile(`role in $empty_list`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetStringField("role", "admin").
			SetList("empty_list", []string{})

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("custom list - IP list", func(t *testing.T) {
		schema := NewSchema().
			AddField("ip.src", TypeIP)

		filter, err := Compile(`ip.src in $blocked_ips`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetIPField("ip.src", "192.168.1.100").
			SetIPList("blocked_ips", []string{"10.0.0.1", "192.168.1.100", "172.16.0.1"})

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)

		ctx2 := NewExecutionContext().
			SetIPField("ip.src", "8.8.8.8").
			SetIPList("blocked_ips", []string{"10.0.0.1", "192.168.1.100", "172.16.0.1"})

		result2, err := filter.Execute(ctx2)
		assert.NoError(t, err)
		assert.False(t, result2)
	})

	t.Run("custom list - IP list with CIDR", func(t *testing.T) {
		schema := NewSchema().
			AddField("device.ip", TypeIP)

		filter, err := Compile(`not device.ip in $management_nets`, schema)
		assert.NoError(t, err)

		nets := []string{"10.255.0.0/16", "172.16.0.0/12"}

		ctx := NewExecutionContext().
			SetIPField("device.ip", "10.255.1.50").
			SetIPList("management_nets", nets)
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)

		ctx2 := NewExecutionContext().
			SetIPField("device.ip", "172.20.5.1").
			SetIPList("management_nets", nets)
		result2, err := filter.Execute(ctx2)
		assert.NoError(t, err)
		assert.False(t, result2)

		ctx3 := NewExecutionContext().
			SetIPField("device.ip", "192.168.1.1").
			SetIPList("management_nets", nets)
		result3, err := filter.Execute(ctx3)
		assert.NoError(t, err)
		assert.True(t, result3)
	})

	t.Run("custom list - mixed IPv4 and IPv6 with CIDR", func(t *testing.T) {
		filter, err := Compile(`ip.src in $nets`, nil)
		assert.NoError(t, err)

		nets := []string{
			"10.0.0.0/8",
			"192.168.1.1",
			"2001:db8::/32",
			"fd00::1",
		}

		ctx := NewExecutionContext().SetIPField("ip.src", "10.50.0.1").SetIPList("nets", nets)
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)

		ctx2 := NewExecutionContext().SetIPField("ip.src", "192.168.1.1").SetIPList("nets", nets)
		result2, err := filter.Execute(ctx2)
		assert.NoError(t, err)
		assert.True(t, result2)

		ctx3 := NewExecutionContext().SetIPField("ip.src", "2001:db8::abcd").SetIPList("nets", nets)
		result3, err := filter.Execute(ctx3)
		assert.NoError(t, err)
		assert.True(t, result3)

		ctx4 := NewExecutionContext().SetIPField("ip.src", "fd00::1").SetIPList("nets", nets)
		result4, err := filter.Execute(ctx4)
		assert.NoError(t, err)
		assert.True(t, result4)

		ctx5 := NewExecutionContext().SetIPField("ip.src", "8.8.8.8").SetIPList("nets", nets)
		result5, err := filter.Execute(ctx5)
		assert.NoError(t, err)
		assert.False(t, result5)

		ctx6 := NewExecutionContext().SetIPField("ip.src", "fe80::1").SetIPList("nets", nets)
		result6, err := filter.Execute(ctx6)
		assert.NoError(t, err)
		assert.False(t, result6)
	})
}

func TestFilterLookupTable(t *testing.T) {
	t.Run("scalar table lookup", func(t *testing.T) {
		filter, err := Compile(`$geo[ip.src] == "US"`, nil)
		require.NoError(t, err)

		ctx := NewExecutionContext().
			SetIPField("ip.src", "10.0.0.1").
			SetTable("geo", map[string]string{"10.0.0.1": "US", "8.8.8.8": "DE"})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)

		ctx2 := NewExecutionContext().
			SetIPField("ip.src", "8.8.8.8").
			SetTable("geo", map[string]string{"10.0.0.1": "US", "8.8.8.8": "DE"})
		result2, err := filter.Execute(ctx2)
		assert.NoError(t, err)
		assert.False(t, result2)
	})

	t.Run("table lookup with int values", func(t *testing.T) {
		filter, err := Compile(`$rate_limits[user.role] >= 100`, nil)
		require.NoError(t, err)

		ctx := NewExecutionContext().
			SetStringField("user.role", "admin").
			SetTableValues("rate_limits", map[string]Value{
				"admin": IntValue(1000),
				"user":  IntValue(50),
			})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)

		ctx2 := NewExecutionContext().
			SetStringField("user.role", "user").
			SetTableValues("rate_limits", map[string]Value{
				"admin": IntValue(1000),
				"user":  IntValue(50),
			})
		result2, err := filter.Execute(ctx2)
		assert.NoError(t, err)
		assert.False(t, result2)
	})

	t.Run("table lookup key not found", func(t *testing.T) {
		filter, err := Compile(`$geo[ip.src] == "US"`, nil)
		require.NoError(t, err)

		ctx := NewExecutionContext().
			SetIPField("ip.src", "1.2.3.4").
			SetTable("geo", map[string]string{"10.0.0.1": "US"})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("table not defined", func(t *testing.T) {
		filter, err := Compile(`$missing[ip.src] == "US"`, nil)
		require.NoError(t, err)

		ctx := NewExecutionContext().SetIPField("ip.src", "10.0.0.1")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("array table lookup with in", func(t *testing.T) {
		filter, err := Compile(`user.role in $allowed_roles[department]`, nil)
		require.NoError(t, err)

		ctx := NewExecutionContext().
			SetStringField("user.role", "dev").
			SetStringField("department", "eng").
			SetTableList("allowed_roles", map[string][]string{
				"eng":   {"dev", "sre", "lead"},
				"sales": {"account", "manager"},
			})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)

		ctx2 := NewExecutionContext().
			SetStringField("user.role", "dev").
			SetStringField("department", "sales").
			SetTableList("allowed_roles", map[string][]string{
				"eng":   {"dev", "sre", "lead"},
				"sales": {"account", "manager"},
			})
		result2, err := filter.Execute(ctx2)
		assert.NoError(t, err)
		assert.False(t, result2)
	})

	t.Run("IP table lookup with in", func(t *testing.T) {
		filter, err := Compile(`ip.src in $blocked_nets[region]`, nil)
		require.NoError(t, err)

		ctx := NewExecutionContext().
			SetIPField("ip.src", "10.0.0.5").
			SetStringField("region", "office").
			SetTableIPList("blocked_nets", map[string][]string{
				"office": {"10.0.0.0/8"},
				"vpn":    {"172.16.0.0/12"},
			})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)

		ctx2 := NewExecutionContext().
			SetIPField("ip.src", "192.168.1.1").
			SetStringField("region", "office").
			SetTableIPList("blocked_nets", map[string][]string{
				"office": {"10.0.0.0/8"},
				"vpn":    {"172.16.0.0/12"},
			})
		result2, err := filter.Execute(ctx2)
		assert.NoError(t, err)
		assert.False(t, result2)
	})

	t.Run("IP table lookup with not in", func(t *testing.T) {
		filter, err := Compile(`ip.src not in $allowed_nets[zone]`, nil)
		require.NoError(t, err)

		ctx := NewExecutionContext().
			SetIPField("ip.src", "192.168.1.1").
			SetStringField("zone", "dmz").
			SetTableIPList("allowed_nets", map[string][]string{
				"dmz":      {"10.0.0.0/8"},
				"internal": {"192.168.0.0/16"},
			})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("table lookup with string field key", func(t *testing.T) {
		filter, err := Compile(`$config[env] == "production"`, nil)
		require.NoError(t, err)

		ctx := NewExecutionContext().
			SetStringField("env", "mode").
			SetTable("config", map[string]string{"mode": "production", "debug": "false"})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("table lookup combined with logical operators", func(t *testing.T) {
		filter, err := Compile(
			`$geo[ip.src] == "US" and user.role in $allowed_roles[department]`,
			nil,
		)
		require.NoError(t, err)

		ctx := NewExecutionContext().
			SetIPField("ip.src", "10.0.0.1").
			SetStringField("user.role", "dev").
			SetStringField("department", "eng").
			SetTable("geo", map[string]string{"10.0.0.1": "US"}).
			SetTableList("allowed_roles", map[string][]string{
				"eng": {"dev", "sre"},
			})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("table marshal unmarshal", func(t *testing.T) {
		filter, err := Compile(`$geo[ip.src] == "US"`, nil)
		require.NoError(t, err)

		data, err := filter.MarshalBinary()
		require.NoError(t, err)

		restored := &Filter{}
		err = restored.UnmarshalBinary(data)
		require.NoError(t, err)

		ctx := NewExecutionContext().
			SetIPField("ip.src", "10.0.0.1").
			SetTable("geo", map[string]string{"10.0.0.1": "US"})

		r1, _ := filter.Execute(ctx)
		r2, _ := restored.Execute(ctx)
		assert.Equal(t, r1, r2)
	})
}
