package wirefilter

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
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
