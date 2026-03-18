package wirefilter

import (
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
}
