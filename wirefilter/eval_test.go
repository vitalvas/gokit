package wirefilter

import (
	"math"
	"net"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestArithmeticOperators(t *testing.T) {
	t.Run("addition", func(t *testing.T) {
		f, _ := Compile(`x + 1 == 6`, nil)
		ctx := NewExecutionContext().SetIntField("x", 5)
		result, _ := f.Execute(ctx)
		assert.True(t, result)
	})

	t.Run("subtraction", func(t *testing.T) {
		f, _ := Compile(`x - 3 == 2`, nil)
		ctx := NewExecutionContext().SetIntField("x", 5)
		result, _ := f.Execute(ctx)
		assert.True(t, result)
	})

	t.Run("multiplication", func(t *testing.T) {
		f, _ := Compile(`x * 3 == 15`, nil)
		ctx := NewExecutionContext().SetIntField("x", 5)
		result, _ := f.Execute(ctx)
		assert.True(t, result)
	})

	t.Run("division", func(t *testing.T) {
		f, _ := Compile(`x / 2 == 5`, nil)
		ctx := NewExecutionContext().SetIntField("x", 10)
		result, _ := f.Execute(ctx)
		assert.True(t, result)
	})

	t.Run("modulo", func(t *testing.T) {
		f, _ := Compile(`x % 3 == 1`, nil)
		ctx := NewExecutionContext().SetIntField("x", 10)
		result, _ := f.Execute(ctx)
		assert.True(t, result)
	})

	t.Run("division by zero returns nil", func(t *testing.T) {
		f, _ := Compile(`x / 0 == 0`, nil)
		ctx := NewExecutionContext().SetIntField("x", 10)
		result, _ := f.Execute(ctx)
		assert.False(t, result)
	})

	t.Run("modulo by zero returns nil", func(t *testing.T) {
		f, _ := Compile(`x % 0 == 0`, nil)
		ctx := NewExecutionContext().SetIntField("x", 10)
		result, _ := f.Execute(ctx)
		assert.False(t, result)
	})

	t.Run("float addition", func(t *testing.T) {
		f, _ := Compile(`x + 1.5 == 3.5`, nil)
		ctx := NewExecutionContext().SetFloatField("x", 2.0)
		result, _ := f.Execute(ctx)
		assert.True(t, result)
	})

	t.Run("float multiplication", func(t *testing.T) {
		f, _ := Compile(`x * 2.0 == 6.28`, nil)
		ctx := NewExecutionContext().SetFloatField("x", 3.14)
		result, _ := f.Execute(ctx)
		assert.True(t, result)
	})

	t.Run("float division", func(t *testing.T) {
		f, _ := Compile(`x / 2.0 == 5.0`, nil)
		ctx := NewExecutionContext().SetFloatField("x", 10.0)
		result, _ := f.Execute(ctx)
		assert.True(t, result)
	})

	t.Run("float division by zero", func(t *testing.T) {
		f, _ := Compile(`x / 0.0 == 0`, nil)
		ctx := NewExecutionContext().SetFloatField("x", 10.0)
		result, _ := f.Execute(ctx)
		assert.False(t, result)
	})

	t.Run("mixed int and float", func(t *testing.T) {
		f, _ := Compile(`x + 1 == 3.14`, nil)
		ctx := NewExecutionContext().SetFloatField("x", 2.14)
		result, _ := f.Execute(ctx)
		assert.True(t, result)
	})

	t.Run("precedence mul before add", func(t *testing.T) {
		f, _ := Compile(`x + y * 2 == 7`, nil)
		ctx := NewExecutionContext().SetIntField("x", 1).SetIntField("y", 3)
		result, _ := f.Execute(ctx)
		assert.True(t, result)
	})

	t.Run("arithmetic in comparison", func(t *testing.T) {
		f, _ := Compile(`x * 2 > 5`, nil)
		ctx := NewExecutionContext().SetIntField("x", 3)
		result, _ := f.Execute(ctx)
		assert.True(t, result)
	})

	t.Run("arithmetic with non-numeric", func(t *testing.T) {
		f, _ := Compile(`name + 1 == 1`, nil)
		ctx := NewExecutionContext().SetStringField("name", "test")
		result, _ := f.Execute(ctx)
		assert.False(t, result)
	})

	t.Run("nil operand", func(t *testing.T) {
		f, _ := Compile(`missing + 1 == 1`, nil)
		ctx := NewExecutionContext()
		result, _ := f.Execute(ctx)
		assert.False(t, result)
	})

	t.Run("complex expression", func(t *testing.T) {
		f, _ := Compile(`(x + y) * 2 == 10`, nil)
		ctx := NewExecutionContext().SetIntField("x", 2).SetIntField("y", 3)
		result, _ := f.Execute(ctx)
		assert.True(t, result)
	})

	t.Run("arithmetic with logical", func(t *testing.T) {
		f, _ := Compile(`x + 1 > 5 and y * 2 < 10`, nil)
		ctx := NewExecutionContext().SetIntField("x", 5).SetIntField("y", 4)
		result, _ := f.Execute(ctx)
		assert.True(t, result)
	})

	t.Run("schema type validation", func(t *testing.T) {
		schema := NewSchema().
			AddField("x", TypeInt).
			AddField("name", TypeString)

		_, err := Compile(`x + 1 == 6`, schema)
		assert.NoError(t, err)

		_, err = Compile(`name + 1 == 6`, schema)
		assert.Error(t, err)
	})
}

func TestEvalCoverageEdgeCases(t *testing.T) {
	t.Run("arithmetic float mod", func(t *testing.T) {
		f, _ := Compile(`x % 2.0 == 1.5`, nil)
		ctx := NewExecutionContext().SetFloatField("x", 5.5)
		result, _ := f.Execute(ctx)
		assert.True(t, result)
	})

	t.Run("arithmetic float subtraction", func(t *testing.T) {
		f, _ := Compile(`x - 1.5 == 1.5`, nil)
		ctx := NewExecutionContext().SetFloatField("x", 3.0)
		result, _ := f.Execute(ctx)
		assert.True(t, result)
	})

	t.Run("arithmetic float mod by zero", func(t *testing.T) {
		f, _ := Compile(`x % 0.0 == 0`, nil)
		ctx := NewExecutionContext().SetFloatField("x", 5.0)
		result, _ := f.Execute(ctx)
		assert.False(t, result)
	})

	t.Run("arithmetic float div by zero", func(t *testing.T) {
		f, _ := Compile(`x / 0.0 == 0`, nil)
		ctx := NewExecutionContext().SetFloatField("x", 5.0)
		result, _ := f.Execute(ctx)
		assert.False(t, result)
	})

	t.Run("arithmetic with non-numeric float", func(t *testing.T) {
		f, _ := Compile(`x + 1.5 == 0`, nil)
		ctx := NewExecutionContext().SetStringField("x", "hello")
		result, _ := f.Execute(ctx)
		assert.False(t, result)
	})

	t.Run("equality CIDR with string left", func(t *testing.T) {
		f, _ := Compile(`"10.0.0.0/8" == cidr(ip, 8)`, nil)
		ctx := NewExecutionContext().SetIPField("ip", "10.1.2.3")
		result, _ := f.Execute(ctx)
		assert.True(t, result)
	})

	t.Run("equality CIDR with invalid string", func(t *testing.T) {
		f, _ := Compile(`cidr(ip, 24) == "not-a-cidr"`, nil)
		ctx := NewExecutionContext().SetIPField("ip", "10.0.0.1")
		result, _ := f.Execute(ctx)
		assert.False(t, result)
	})

	t.Run("equality string left invalid CIDR", func(t *testing.T) {
		f, _ := Compile(`"not-a-cidr" == cidr(ip, 24)`, nil)
		ctx := NewExecutionContext().SetIPField("ip", "10.0.0.1")
		result, _ := f.Execute(ctx)
		assert.False(t, result)
	})
}

func TestFilterEval(t *testing.T) {
	t.Run("simple string equality", func(t *testing.T) {
		schema := NewSchema().
			AddField("method", TypeString)

		filter, err := Compile(`method == "GET"`, schema)
		assert.NoError(t, err)
		assert.NotNil(t, filter)

		ctx := NewExecutionContext().
			SetStringField("method", "GET")

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)

		ctx2 := NewExecutionContext().
			SetStringField("method", "POST")

		result2, err := filter.Execute(ctx2)
		assert.NoError(t, err)
		assert.False(t, result2)
	})

	t.Run("integer comparison", func(t *testing.T) {
		schema := NewSchema().
			AddField("status", TypeInt)

		filter, err := Compile(`status >= 200 && status < 300`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetIntField("status", 200)

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)

		ctx2 := NewExecutionContext().
			SetIntField("status", 404)

		result2, err := filter.Execute(ctx2)
		assert.NoError(t, err)
		assert.False(t, result2)
	})

	t.Run("boolean logic", func(t *testing.T) {
		schema := NewSchema().
			AddField("active", TypeBool).
			AddField("verified", TypeBool)

		filter, err := Compile(`active == true && verified == true`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetBoolField("active", true).
			SetBoolField("verified", true)

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)

		ctx2 := NewExecutionContext().
			SetBoolField("active", true).
			SetBoolField("verified", false)

		result2, err := filter.Execute(ctx2)
		assert.NoError(t, err)
		assert.False(t, result2)
	})

	t.Run("not operator", func(t *testing.T) {
		schema := NewSchema().
			AddField("blocked", TypeBool)

		filter, err := Compile(`not blocked`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetBoolField("blocked", false)

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)

		ctx2 := NewExecutionContext().
			SetBoolField("blocked", true)

		result2, err := filter.Execute(ctx2)
		assert.NoError(t, err)
		assert.False(t, result2)
	})

	t.Run("contains operator", func(t *testing.T) {
		schema := NewSchema().
			AddField("message", TypeString)

		filter, err := Compile(`message contains "error"`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetStringField("message", "An error occurred")

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)

		ctx2 := NewExecutionContext().
			SetStringField("message", "Success")

		result2, err := filter.Execute(ctx2)
		assert.NoError(t, err)
		assert.False(t, result2)
	})

	t.Run("matches operator", func(t *testing.T) {
		schema := NewSchema().
			AddField("email", TypeString)

		filter, err := Compile(`email matches "^.*@example\\.com$"`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetStringField("email", "user@example.com")

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)

		ctx2 := NewExecutionContext().
			SetStringField("email", "user@other.com")

		result2, err := filter.Execute(ctx2)
		assert.NoError(t, err)
		assert.False(t, result2)
	})

	t.Run("in operator with array", func(t *testing.T) {
		schema := NewSchema().
			AddField("port", TypeInt)

		filter, err := Compile(`port in {80, 443, 8080}`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetIntField("port", 443)

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)

		ctx2 := NewExecutionContext().
			SetIntField("port", 3000)

		result2, err := filter.Execute(ctx2)
		assert.NoError(t, err)
		assert.False(t, result2)
	})

	t.Run("ip in cidr", func(t *testing.T) {
		schema := NewSchema().
			AddField("ip", TypeIP)

		filter, err := Compile(`ip in "192.168.1.0/24"`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetIPField("ip", "192.168.1.100")

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)

		ctx2 := NewExecutionContext().
			SetIPField("ip", "10.0.0.1")

		result2, err := filter.Execute(ctx2)
		assert.NoError(t, err)
		assert.False(t, result2)
	})

	t.Run("complex expression", func(t *testing.T) {
		schema := NewSchema().
			AddField("method", TypeString).
			AddField("status", TypeInt).
			AddField("path", TypeString)

		filter, err := Compile(`method == "GET" && status == 200 && path contains "/api/"`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetStringField("method", "GET").
			SetIntField("status", 200).
			SetStringField("path", "/api/users")

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)

		ctx2 := NewExecutionContext().
			SetStringField("method", "POST").
			SetIntField("status", 200).
			SetStringField("path", "/api/users")

		result2, err := filter.Execute(ctx2)
		assert.NoError(t, err)
		assert.False(t, result2)
	})

	t.Run("or expression", func(t *testing.T) {
		schema := NewSchema().
			AddField("status", TypeInt)

		filter, err := Compile(`status == 404 || status == 500`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetIntField("status", 404)

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)

		ctx2 := NewExecutionContext().
			SetIntField("status", 200)

		result2, err := filter.Execute(ctx2)
		assert.NoError(t, err)
		assert.False(t, result2)
	})

	t.Run("range membership", func(t *testing.T) {
		schema := NewSchema().
			AddField("port", TypeInt)

		filter, err := Compile(`port in {80, 443, 8000..9000}`, schema)
		assert.NoError(t, err)

		testCases := []struct {
			port     int64
			expected bool
		}{
			{80, true},
			{443, true},
			{8000, true},
			{8500, true},
			{9000, true},
			{9001, false},
			{100, false},
			{7999, false},
		}

		for _, tc := range testCases {
			ctx := NewExecutionContext().
				SetIntField("port", tc.port)

			result, err := filter.Execute(ctx)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, result, "port %d", tc.port)
		}
	})

	t.Run("multiple ranges", func(t *testing.T) {
		schema := NewSchema().
			AddField("port", TypeInt)

		filter, err := Compile(`port in {1..10, 20..30, 100}`, schema)
		assert.NoError(t, err)

		testCases := []struct {
			port     int64
			expected bool
		}{
			{1, true},
			{5, true},
			{10, true},
			{15, false},
			{20, true},
			{25, true},
			{30, true},
			{50, false},
			{100, true},
		}

		for _, tc := range testCases {
			ctx := NewExecutionContext().
				SetIntField("port", tc.port)

			result, err := filter.Execute(ctx)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, result, "port %d", tc.port)
		}
	})

	t.Run("ipv6 in cidr filter", func(t *testing.T) {
		schema := NewSchema().
			AddField("ip", TypeIP)

		filter, err := Compile(`ip in "2001:db8::/32"`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetIPField("ip", "2001:db8::1")

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)

		ctx2 := NewExecutionContext().
			SetIPField("ip", "2001:db9::1")

		result2, err := filter.Execute(ctx2)
		assert.NoError(t, err)
		assert.False(t, result2)
	})

	t.Run("ipv6 equality", func(t *testing.T) {
		schema := NewSchema().
			AddField("ip", TypeIP)

		filter, err := Compile(`ip == "2001:db8::1"`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetIPField("ip", "2001:db8::1")

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("all equal operator - all elements match", func(t *testing.T) {
		schema := NewSchema().
			AddField("tags", TypeArray)

		filter, err := Compile(`tags === "test"`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetField("tags", ArrayValue{
				StringValue("test"),
				StringValue("test"),
				StringValue("test"),
			})

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("all equal operator - some elements do not match", func(t *testing.T) {
		schema := NewSchema().
			AddField("tags", TypeArray)

		filter, err := Compile(`tags === "test"`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetField("tags", ArrayValue{
				StringValue("test"),
				StringValue("other"),
				StringValue("test"),
			})

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("all equal operator - empty array", func(t *testing.T) {
		schema := NewSchema().
			AddField("tags", TypeArray)

		filter, err := Compile(`tags === "test"`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetField("tags", ArrayValue{})

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("all equal operator - integer array", func(t *testing.T) {
		schema := NewSchema().
			AddField("values", TypeArray)

		filter, err := Compile(`values === 5`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetField("values", ArrayValue{
				IntValue(5),
				IntValue(5),
				IntValue(5),
			})

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)

		ctx2 := NewExecutionContext().
			SetField("values", ArrayValue{
				IntValue(5),
				IntValue(6),
				IntValue(5),
			})

		result2, err := filter.Execute(ctx2)
		assert.NoError(t, err)
		assert.False(t, result2)
	})

	t.Run("any not equal operator - all elements match", func(t *testing.T) {
		schema := NewSchema().
			AddField("tags", TypeArray)

		filter, err := Compile(`tags !== "test"`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetField("tags", ArrayValue{
				StringValue("test"),
				StringValue("test"),
				StringValue("test"),
			})

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("any not equal operator - some elements do not match", func(t *testing.T) {
		schema := NewSchema().
			AddField("tags", TypeArray)

		filter, err := Compile(`tags !== "test"`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetField("tags", ArrayValue{
				StringValue("test"),
				StringValue("other"),
				StringValue("test"),
			})

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("any not equal operator - empty array", func(t *testing.T) {
		schema := NewSchema().
			AddField("tags", TypeArray)

		filter, err := Compile(`tags !== "test"`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetField("tags", ArrayValue{})

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("any not equal operator - integer array", func(t *testing.T) {
		schema := NewSchema().
			AddField("values", TypeArray)

		filter, err := Compile(`values !== 5`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetField("values", ArrayValue{
				IntValue(5),
				IntValue(5),
				IntValue(5),
			})

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)

		ctx2 := NewExecutionContext().
			SetField("values", ArrayValue{
				IntValue(5),
				IntValue(6),
				IntValue(5),
			})

		result2, err := filter.Execute(ctx2)
		assert.NoError(t, err)
		assert.True(t, result2)
	})

	t.Run("all equal operator - non-array value without schema", func(t *testing.T) {
		filter, err := Compile(`name === "test"`, nil)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetStringField("name", "test")

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("any not equal operator - non-array value without schema", func(t *testing.T) {
		filter, err := Compile(`name !== "test"`, nil)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetStringField("name", "test")

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("field presence - string field present", func(t *testing.T) {
		schema := NewSchema().
			AddField("http.host", TypeString)

		filter, err := Compile(`http.host`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetStringField("http.host", "example.com")

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("field presence - string field absent", func(t *testing.T) {
		schema := NewSchema().
			AddField("http.host", TypeString)

		filter, err := Compile(`http.host`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext()

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("field presence - int field present with zero", func(t *testing.T) {
		schema := NewSchema().
			AddField("http.status", TypeInt)

		filter, err := Compile(`http.status`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetIntField("http.status", 0)

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("field presence - int field absent", func(t *testing.T) {
		schema := NewSchema().
			AddField("http.status", TypeInt)

		filter, err := Compile(`http.status`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext()

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("field presence - bool field present with false", func(t *testing.T) {
		schema := NewSchema().
			AddField("http.secure", TypeBool)

		filter, err := Compile(`http.secure`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetBoolField("http.secure", false)

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("field presence - bool field present with true", func(t *testing.T) {
		schema := NewSchema().
			AddField("http.secure", TypeBool)

		filter, err := Compile(`http.secure`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetBoolField("http.secure", true)

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("field absence - not operator on absent field", func(t *testing.T) {
		schema := NewSchema().
			AddField("http.error", TypeString)

		filter, err := Compile(`not http.error`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext()

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("field absence - not operator on present field", func(t *testing.T) {
		schema := NewSchema().
			AddField("http.error", TypeString)

		filter, err := Compile(`not http.error`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetStringField("http.error", "not found")

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("field presence with and operator", func(t *testing.T) {
		schema := NewSchema().
			AddField("http.host", TypeString).
			AddField("http.status", TypeInt)

		filter, err := Compile(`http.host and http.status == 200`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetStringField("http.host", "example.com").
			SetIntField("http.status", 200)

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)

		ctx2 := NewExecutionContext().
			SetIntField("http.status", 200)

		result2, err := filter.Execute(ctx2)
		assert.NoError(t, err)
		assert.False(t, result2)
	})

	t.Run("field presence with or operator", func(t *testing.T) {
		schema := NewSchema().
			AddField("http.host", TypeString).
			AddField("http.error", TypeString)

		filter, err := Compile(`http.host or http.error`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetStringField("http.host", "example.com")

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)

		ctx2 := NewExecutionContext()

		result2, err := filter.Execute(ctx2)
		assert.NoError(t, err)
		assert.False(t, result2)
	})

	t.Run("field presence - IP field present", func(t *testing.T) {
		schema := NewSchema().
			AddField("ip.src", TypeIP)

		filter, err := Compile(`ip.src`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetIPField("ip.src", "192.168.1.1")

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("field presence - IP field absent", func(t *testing.T) {
		schema := NewSchema().
			AddField("ip.src", TypeIP)

		filter, err := Compile(`ip.src`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext()

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("combined presence and absence check", func(t *testing.T) {
		schema := NewSchema().
			AddField("http.host", TypeString).
			AddField("http.error", TypeString)

		filter, err := Compile(`http.host and not http.error`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetStringField("http.host", "example.com")

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)

		ctx2 := NewExecutionContext().
			SetStringField("http.host", "example.com").
			SetStringField("http.error", "not found")

		result2, err := filter.Execute(ctx2)
		assert.NoError(t, err)
		assert.False(t, result2)
	})

	t.Run("field presence - empty string is present", func(t *testing.T) {
		schema := NewSchema().
			AddField("http.host", TypeString)

		filter, err := Compile(`http.host`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetStringField("http.host", "")

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("array in array - OR logic - match found", func(t *testing.T) {
		schema := NewSchema().
			AddField("user.groups", TypeArray)

		filter, err := Compile(`user.groups in {"guest", "test"}`, schema)
		assert.NoError(t, err)

		groups := ArrayValue{
			StringValue("admin"),
			StringValue("guest"),
			StringValue("user"),
		}
		ctx := NewExecutionContext().
			SetField("user.groups", groups)

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("array in array - OR logic - no match", func(t *testing.T) {
		schema := NewSchema().
			AddField("user.groups", TypeArray)

		filter, err := Compile(`user.groups in {"foo", "bar"}`, schema)
		assert.NoError(t, err)

		groups := ArrayValue{
			StringValue("admin"),
			StringValue("guest"),
			StringValue("user"),
		}
		ctx := NewExecutionContext().
			SetField("user.groups", groups)

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("array in array - OR logic - empty left array", func(t *testing.T) {
		schema := NewSchema().
			AddField("user.groups", TypeArray)

		filter, err := Compile(`user.groups in {"guest", "test"}`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetField("user.groups", ArrayValue{})

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("array contains array - AND logic - all match", func(t *testing.T) {
		schema := NewSchema().
			AddField("user.groups", TypeArray)

		filter, err := Compile(`user.groups contains {"guest", "user"}`, schema)
		assert.NoError(t, err)

		groups := ArrayValue{
			StringValue("admin"),
			StringValue("guest"),
			StringValue("user"),
		}
		ctx := NewExecutionContext().
			SetField("user.groups", groups)

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("array contains array - AND logic - partial match", func(t *testing.T) {
		schema := NewSchema().
			AddField("user.groups", TypeArray)

		filter, err := Compile(`user.groups contains {"guest", "test"}`, schema)
		assert.NoError(t, err)

		groups := ArrayValue{
			StringValue("admin"),
			StringValue("guest"),
			StringValue("user"),
		}
		ctx := NewExecutionContext().
			SetField("user.groups", groups)

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("array contains array - AND logic - empty right array", func(t *testing.T) {
		schema := NewSchema().
			AddField("user.groups", TypeArray)

		filter, err := Compile(`user.groups contains {}`, schema)
		assert.NoError(t, err)

		groups := ArrayValue{
			StringValue("admin"),
			StringValue("guest"),
		}
		ctx := NewExecutionContext().
			SetField("user.groups", groups)

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("array in array - OR logic - int values", func(t *testing.T) {
		schema := NewSchema().
			AddField("ports", TypeArray)

		filter, err := Compile(`ports in {80, 443, 8080}`, schema)
		assert.NoError(t, err)

		ports := ArrayValue{
			IntValue(22),
			IntValue(443),
			IntValue(3306),
		}
		ctx := NewExecutionContext().
			SetField("ports", ports)

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("array contains array - AND logic - int values", func(t *testing.T) {
		schema := NewSchema().
			AddField("ports", TypeArray)

		filter, err := Compile(`ports contains {22, 443}`, schema)
		assert.NoError(t, err)

		ports := ArrayValue{
			IntValue(22),
			IntValue(443),
			IntValue(3306),
		}
		ctx := NewExecutionContext().
			SetField("ports", ports)

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("range expression - non-int start", func(t *testing.T) {
		filter, err := Compile(`status in {"a".."b"}`, nil)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetIntField("status", 200)

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("range expression - start greater than end", func(t *testing.T) {
		filter, err := Compile(`status in {10..1}`, nil)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetIntField("status", 5)

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("invalid regex pattern", func(t *testing.T) {
		filter, err := Compile(`http.path matches "[invalid"`, nil)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetStringField("http.path", "/api/test")

		_, err = filter.Execute(ctx)
		assert.Error(t, err)
	})

	t.Run("invalid CIDR", func(t *testing.T) {
		filter, err := Compile(`ip.src in "invalid-cidr"`, nil)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetIPField("ip.src", "192.168.1.1")

		_, err = filter.Execute(ctx)
		assert.Error(t, err)
	})

	t.Run("comparison with non-int types", func(t *testing.T) {
		filter, err := Compile(`http.host > "test"`, nil)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetStringField("http.host", "example.com")

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("contains with non-string non-array", func(t *testing.T) {
		filter, err := Compile(`status contains 200`, nil)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetIntField("status", 200)

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("matches with non-string types", func(t *testing.T) {
		filter, err := Compile(`status matches "200"`, nil)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetIntField("status", 200)

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("ip equality with string", func(t *testing.T) {
		schema := NewSchema().
			AddField("ip.src", TypeIP)

		filter, err := Compile(`ip.src == "192.168.1.1"`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetIPField("ip.src", "192.168.1.1")

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("string equality with ip", func(t *testing.T) {
		schema := NewSchema().
			AddField("ip.src", TypeIP).
			AddField("str", TypeString)

		filter, err := Compile(`str == ip.src`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetIPField("ip.src", "192.168.1.1").
			SetStringField("str", "192.168.1.1")

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("ip equality with invalid string", func(t *testing.T) {
		schema := NewSchema().
			AddField("ip.src", TypeIP)

		filter, err := Compile(`ip.src == "not-an-ip"`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetIPField("ip.src", "192.168.1.1")

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("string equality with invalid ip", func(t *testing.T) {
		schema := NewSchema().
			AddField("ip.src", TypeIP).
			AddField("str", TypeString)

		filter, err := Compile(`str == ip.src`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetIPField("ip.src", "192.168.1.1").
			SetStringField("str", "not-an-ip")

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("in with non-array non-cidr", func(t *testing.T) {
		filter, err := Compile(`status in 200`, nil)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetIntField("status", 200)

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("all equal with empty array", func(t *testing.T) {
		schema := NewSchema().
			AddField("tags", TypeArray)

		filter, err := Compile(`tags === "test"`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetField("tags", ArrayValue{})

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("any not equal with empty array", func(t *testing.T) {
		schema := NewSchema().
			AddField("tags", TypeArray)

		filter, err := Compile(`tags !== "test"`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetField("tags", ArrayValue{})

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("CIDR without quotes", func(t *testing.T) {
		filter, err := Compile(`ip.src in 192.168.0.0/16`, nil)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetIPField("ip.src", "192.168.1.1")

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)

		ctx2 := NewExecutionContext().
			SetIPField("ip.src", "10.0.0.1")

		result2, err := filter.Execute(ctx2)
		assert.NoError(t, err)
		assert.False(t, result2)
	})

	t.Run("IP without quotes", func(t *testing.T) {
		filter, err := Compile(`ip.src == 192.168.1.1`, nil)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetIPField("ip.src", "192.168.1.1")

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)

		ctx2 := NewExecutionContext().
			SetIPField("ip.src", "192.168.1.2")

		result2, err := filter.Execute(ctx2)
		assert.NoError(t, err)
		assert.False(t, result2)
	})

	t.Run("nil values in and operation", func(t *testing.T) {
		filter, err := Compile(`http.host and http.status == 200`, nil)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetIntField("http.status", 200)

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("nil values in or operation", func(t *testing.T) {
		filter, err := Compile(`http.host or http.status == 200`, nil)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetIntField("http.status", 200)

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("nil left in equality", func(t *testing.T) {
		filter, err := Compile(`http.host == "test"`, nil)
		assert.NoError(t, err)

		ctx := NewExecutionContext()

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("nil in contains", func(t *testing.T) {
		filter, err := Compile(`http.host contains "test"`, nil)
		assert.NoError(t, err)

		ctx := NewExecutionContext()

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("nil in matches", func(t *testing.T) {
		filter, err := Compile(`http.host matches "test"`, nil)
		assert.NoError(t, err)

		ctx := NewExecutionContext()

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("nil in comparison", func(t *testing.T) {
		filter, err := Compile(`http.status > 200`, nil)
		assert.NoError(t, err)

		ctx := NewExecutionContext()

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("nil in array membership", func(t *testing.T) {
		filter, err := Compile(`http.status in {200, 201}`, nil)
		assert.NoError(t, err)

		ctx := NewExecutionContext()

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("nil in all equal", func(t *testing.T) {
		filter, err := Compile(`tags === "test"`, nil)
		assert.NoError(t, err)

		ctx := NewExecutionContext()

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("nil in any not equal", func(t *testing.T) {
		filter, err := Compile(`tags !== "test"`, nil)
		assert.NoError(t, err)

		ctx := NewExecutionContext()

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("bytes contains", func(t *testing.T) {
		schema := NewSchema().
			AddField("data", TypeBytes)

		filter, err := Compile(`data contains "test"`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetBytesField("data", []byte("this is test data"))

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("invalid IP field", func(t *testing.T) {
		ctx := NewExecutionContext().
			SetIPField("ip.src", "invalid-ip")

		_, ok := ctx.GetField("ip.src")
		assert.False(t, ok)
	})

	t.Run("array expr with range error", func(t *testing.T) {
		filter, err := Compile(`status in {100, 200..299}`, nil)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetIntField("status", 250)

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("inequality operator", func(t *testing.T) {
		filter, err := Compile(`status != 200`, nil)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetIntField("status", 404)

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("less than operator", func(t *testing.T) {
		filter, err := Compile(`status < 300`, nil)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetIntField("status", 200)

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("less than or equal operator", func(t *testing.T) {
		filter, err := Compile(`status <= 200`, nil)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetIntField("status", 200)

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("array all equal true case", func(t *testing.T) {
		schema := NewSchema().
			AddField("tags", TypeArray)

		filter, err := Compile(`tags === "test"`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetField("tags", ArrayValue{StringValue("test"), StringValue("test")})

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("array any not equal true case", func(t *testing.T) {
		schema := NewSchema().
			AddField("tags", TypeArray)

		filter, err := Compile(`tags !== "test"`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetField("tags", ArrayValue{StringValue("test"), StringValue("other")})

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("array contains with string haystack", func(t *testing.T) {
		filter, err := Compile(`tags contains "test"`, nil)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetField("tags", ArrayValue{StringValue("test"), StringValue("other")})

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("contains with nil right array", func(t *testing.T) {
		filter, err := Compile(`tags contains otherfield`, nil)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetField("tags", ArrayValue{StringValue("test")})

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("map field access with bracket notation", func(t *testing.T) {
		schema := NewSchema().
			AddField("user.attributes", TypeMap)

		filter, err := Compile(`user.attributes["region"] == "us-west"`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetMapField("user.attributes", map[string]string{"region": "us-west"})

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)

		ctx2 := NewExecutionContext().
			SetMapField("user.attributes", map[string]string{"region": "us-east"})

		result2, err := filter.Execute(ctx2)
		assert.NoError(t, err)
		assert.False(t, result2)
	})

	t.Run("field-to-field comparison with map access", func(t *testing.T) {
		schema := NewSchema().
			AddField("user.attributes", TypeMap).
			AddField("device.vars", TypeMap)

		filter, err := Compile(`user.attributes["region"] == device.vars["region"]`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetMapField("user.attributes", map[string]string{"region": "us-west"}).
			SetMapField("device.vars", map[string]string{"region": "us-west"})

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)

		ctx2 := NewExecutionContext().
			SetMapField("user.attributes", map[string]string{"region": "us-west"}).
			SetMapField("device.vars", map[string]string{"region": "us-east"})

		result2, err := filter.Execute(ctx2)
		assert.NoError(t, err)
		assert.False(t, result2)
	})

	t.Run("field-to-field equality without bracket notation", func(t *testing.T) {
		schema := NewSchema().
			AddField("user.region", TypeString).
			AddField("device.region", TypeString)

		filter, err := Compile(`user.region == device.region`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetStringField("user.region", "us-west").
			SetStringField("device.region", "us-west")

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)

		ctx2 := NewExecutionContext().
			SetStringField("user.region", "us-west").
			SetStringField("device.region", "us-east")

		result2, err := filter.Execute(ctx2)
		assert.NoError(t, err)
		assert.False(t, result2)
	})

	t.Run("field-to-field comparison with int values", func(t *testing.T) {
		schema := NewSchema().
			AddField("user.age", TypeInt).
			AddField("limit.age", TypeInt)

		filter, err := Compile(`user.age >= limit.age`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetIntField("user.age", 25).
			SetIntField("limit.age", 18)

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)

		ctx2 := NewExecutionContext().
			SetIntField("user.age", 15).
			SetIntField("limit.age", 18)

		result2, err := filter.Execute(ctx2)
		assert.NoError(t, err)
		assert.False(t, result2)
	})

	t.Run("complex expression with field-to-field and map access", func(t *testing.T) {
		schema := NewSchema().
			AddField("user.attributes", TypeMap).
			AddField("device.vars", TypeMap).
			AddField("user.active", TypeBool)

		filter, err := Compile(`user.attributes["region"] == device.vars["region"] and user.active == true`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetMapField("user.attributes", map[string]string{"region": "us-west"}).
			SetMapField("device.vars", map[string]string{"region": "us-west"}).
			SetBoolField("user.active", true)

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)

		ctx2 := NewExecutionContext().
			SetMapField("user.attributes", map[string]string{"region": "us-west"}).
			SetMapField("device.vars", map[string]string{"region": "us-west"}).
			SetBoolField("user.active", false)

		result2, err := filter.Execute(ctx2)
		assert.NoError(t, err)
		assert.False(t, result2)
	})

	t.Run("map access with missing key returns false", func(t *testing.T) {
		schema := NewSchema().
			AddField("user.attributes", TypeMap)

		filter, err := Compile(`user.attributes["region"] == "us-west"`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetMapField("user.attributes", map[string]string{"other": "value"})

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("map access with missing field returns false", func(t *testing.T) {
		schema := NewSchema().
			AddField("user.attributes", TypeMap)

		filter, err := Compile(`user.attributes["region"] == "us-west"`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext()

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("map value equality", func(t *testing.T) {
		ctx := NewExecutionContext().
			SetMapField("attrs", map[string]string{"a": "1", "b": "2"})

		val, ok := ctx.GetField("attrs")
		assert.True(t, ok)
		assert.Equal(t, TypeMap, val.Type())

		mapVal := val.(MapValue)
		v, exists := mapVal.Get("a")
		assert.True(t, exists)
		assert.Equal(t, StringValue("1"), v)

		_, exists = mapVal.Get("missing")
		assert.False(t, exists)
	})

	t.Run("map truthiness", func(t *testing.T) {
		// Maps are truthy when present (field presence semantics)
		emptyMap := MapValue{}
		assert.True(t, emptyMap.IsTruthy())

		nonEmptyMap := MapValue{"key": StringValue("value")}
		assert.True(t, nonEmptyMap.IsTruthy())
	})

	t.Run("map with int values", func(t *testing.T) {
		schema := NewSchema().
			AddField("config", TypeMap)

		filter, err := Compile(`config["port"] == 8080`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetMapFieldValues("config", map[string]Value{
				"port": IntValue(8080),
				"host": StringValue("localhost"),
			})

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("map with mixed value types comparison", func(t *testing.T) {
		schema := NewSchema().
			AddField("user.settings", TypeMap).
			AddField("default.settings", TypeMap)

		filter, err := Compile(`user.settings["timeout"] == default.settings["timeout"]`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetMapFieldValues("user.settings", map[string]Value{
				"timeout": IntValue(30),
			}).
			SetMapFieldValues("default.settings", map[string]Value{
				"timeout": IntValue(30),
			})

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)

		ctx2 := NewExecutionContext().
			SetMapFieldValues("user.settings", map[string]Value{
				"timeout": IntValue(60),
			}).
			SetMapFieldValues("default.settings", map[string]Value{
				"timeout": IntValue(30),
			})

		result2, err := filter.Execute(ctx2)
		assert.NoError(t, err)
		assert.False(t, result2)
	})

	t.Run("map with bool values", func(t *testing.T) {
		schema := NewSchema().
			AddField("flags", TypeMap)

		filter, err := Compile(`flags["enabled"] == true`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetMapFieldValues("flags", map[string]Value{
				"enabled": BoolValue(true),
			})

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("map equality", func(t *testing.T) {
		map1 := MapValue{"a": StringValue("1"), "b": StringValue("2")}
		map2 := MapValue{"a": StringValue("1"), "b": StringValue("2")}
		map3 := MapValue{"a": StringValue("1"), "b": StringValue("3")}
		map4 := MapValue{"a": StringValue("1")}

		assert.True(t, map1.Equal(map2))
		assert.False(t, map1.Equal(map3))
		assert.False(t, map1.Equal(map4))
		assert.False(t, map1.Equal(StringValue("test")))
	})

	t.Run("map string representation", func(t *testing.T) {
		m := MapValue{"key": StringValue("value")}
		str := m.String()
		assert.Contains(t, str, "key")
		assert.Contains(t, str, "value")
	})

	t.Run("xor operator - truth table", func(t *testing.T) {
		schema := NewSchema().
			AddField("a", TypeBool).
			AddField("b", TypeBool)

		filter, err := Compile(`a xor b`, schema)
		assert.NoError(t, err)

		testCases := []struct {
			a, b     bool
			expected bool
		}{
			{true, true, false},
			{true, false, true},
			{false, true, true},
			{false, false, false},
		}

		for _, tc := range testCases {
			ctx := NewExecutionContext().
				SetBoolField("a", tc.a).
				SetBoolField("b", tc.b)

			result, err := filter.Execute(ctx)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, result, "a=%v xor b=%v", tc.a, tc.b)
		}
	})

	t.Run("xor operator with symbol", func(t *testing.T) {
		schema := NewSchema().
			AddField("a", TypeBool).
			AddField("b", TypeBool)

		filter, err := Compile(`a ^^ b`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetBoolField("a", true).
			SetBoolField("b", false)

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("wildcard operator - case insensitive", func(t *testing.T) {
		schema := NewSchema().
			AddField("host", TypeString)

		filter, err := Compile(`host wildcard "*.example.com"`, schema)
		assert.NoError(t, err)

		testCases := []struct {
			host     string
			expected bool
		}{
			{"www.example.com", true},
			{"api.example.com", true},
			{"WWW.EXAMPLE.COM", true},
			{"Api.Example.Com", true},
			{"example.com", false},
			{"www.other.com", false},
			{"www.example.org", false},
		}

		for _, tc := range testCases {
			ctx := NewExecutionContext().
				SetStringField("host", tc.host)

			result, err := filter.Execute(ctx)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, result, "host=%s", tc.host)
		}
	})

	t.Run("wildcard operator - multiple wildcards", func(t *testing.T) {
		schema := NewSchema().
			AddField("path", TypeString)

		filter, err := Compile(`path wildcard "/api/*/users/*"`, schema)
		assert.NoError(t, err)

		testCases := []struct {
			path     string
			expected bool
		}{
			{"/api/v1/users/123", true},
			{"/api/v2/users/456", true},
			{"/API/V1/USERS/789", true},
			{"/api/users/123", false},
			{"/web/v1/users/123", false},
		}

		for _, tc := range testCases {
			ctx := NewExecutionContext().
				SetStringField("path", tc.path)

			result, err := filter.Execute(ctx)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, result, "path=%s", tc.path)
		}
	})

	t.Run("strict wildcard operator - case sensitive", func(t *testing.T) {
		schema := NewSchema().
			AddField("host", TypeString)

		filter, err := Compile(`host strict wildcard "*.Example.com"`, schema)
		assert.NoError(t, err)

		testCases := []struct {
			host     string
			expected bool
		}{
			{"www.Example.com", true},
			{"api.Example.com", true},
			{"www.example.com", false},
			{"WWW.EXAMPLE.COM", false},
			{"www.Example.org", false},
		}

		for _, tc := range testCases {
			ctx := NewExecutionContext().
				SetStringField("host", tc.host)

			result, err := filter.Execute(ctx)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, result, "host=%s", tc.host)
		}
	})

	t.Run("wildcard with question mark", func(t *testing.T) {
		schema := NewSchema().
			AddField("code", TypeString)

		filter, err := Compile(`code wildcard "A?C"`, schema)
		assert.NoError(t, err)

		testCases := []struct {
			code     string
			expected bool
		}{
			{"ABC", true},
			{"A1C", true},
			{"abc", true},
			{"AC", false},
			{"ABBC", false},
		}

		for _, tc := range testCases {
			ctx := NewExecutionContext().
				SetStringField("code", tc.code)

			result, err := filter.Execute(ctx)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, result, "code=%s", tc.code)
		}
	})

	t.Run("wildcard with special regex chars escaped", func(t *testing.T) {
		schema := NewSchema().
			AddField("path", TypeString)

		filter, err := Compile(`path wildcard "/api/v1.0/*"`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetStringField("path", "/api/v1.0/users")

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)

		ctx2 := NewExecutionContext().
			SetStringField("path", "/api/v1X0/users")

		result2, err := filter.Execute(ctx2)
		assert.NoError(t, err)
		assert.False(t, result2)
	})

	t.Run("matches with tilde alias", func(t *testing.T) {
		schema := NewSchema().
			AddField("email", TypeString)

		filter, err := Compile(`email ~ "^.*@example\\.com$"`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetStringField("email", "user@example.com")

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)

		ctx2 := NewExecutionContext().
			SetStringField("email", "user@other.com")

		result2, err := filter.Execute(ctx2)
		assert.NoError(t, err)
		assert.False(t, result2)
	})

	t.Run("not with exclamation alias", func(t *testing.T) {
		schema := NewSchema().
			AddField("blocked", TypeBool)

		filter, err := Compile(`! blocked`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetBoolField("blocked", false)

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)

		ctx2 := NewExecutionContext().
			SetBoolField("blocked", true)

		result2, err := filter.Execute(ctx2)
		assert.NoError(t, err)
		assert.False(t, result2)
	})

	t.Run("wildcard with nil values", func(t *testing.T) {
		schema := NewSchema().
			AddField("host", TypeString)

		filter, err := Compile(`host wildcard "*.example.com"`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext()

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("xor with nil values", func(t *testing.T) {
		schema := NewSchema().
			AddField("a", TypeBool).
			AddField("b", TypeBool)

		filter, err := Compile(`a xor b`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetBoolField("a", true)

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("globToRegex function", func(t *testing.T) {
		testCases := []struct {
			glob     string
			expected string
		}{
			{"*", "^.*$"},
			{"?", "^.$"},
			{"*.txt", "^.*\\.txt$"},
			{"file[1]", "^file\\[1\\]$"},
			{"a+b", "^a\\+b$"},
			{"test$var", "^test\\$var$"},
		}

		for _, tc := range testCases {
			result := globToRegex(tc.glob)
			assert.Equal(t, tc.expected, result, "glob=%s", tc.glob)
		}
	})

	t.Run("cached regex pattern", func(t *testing.T) {
		filter, err := Compile(`http.path matches "^/api/"`, nil)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetStringField("http.path", "/api/v1")

		result1, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result1)

		result2, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result2)
	})

	t.Run("cached CIDR pattern", func(t *testing.T) {
		filter, err := Compile(`ip.src in "192.168.0.0/16"`, nil)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetIPField("ip.src", "192.168.1.1")

		result1, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result1)

		result2, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result2)
	})

	t.Run("unary not on non-existent field", func(t *testing.T) {
		filter, err := Compile(`not http.host`, nil)
		assert.NoError(t, err)

		ctx := NewExecutionContext()
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("not in operator", func(t *testing.T) {
		filter, err := Compile(`ip.src not in {192.168.1.0/24, 10.0.0.0/8}`, nil)
		assert.NoError(t, err)

		ctx := NewExecutionContext().SetIPField("ip.src", "192.168.1.50")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)

		ctx2 := NewExecutionContext().SetIPField("ip.src", "8.8.8.8")
		result2, err := filter.Execute(ctx2)
		assert.NoError(t, err)
		assert.True(t, result2)
	})

	t.Run("not in with list ref", func(t *testing.T) {
		filter, err := Compile(`device.ip not in $management_nets`, nil)
		assert.NoError(t, err)

		nets := []string{"10.255.0.0/16", "172.16.0.0/12"}

		ctx := NewExecutionContext().
			SetIPField("device.ip", "10.255.1.50").
			SetIPList("management_nets", nets)
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)

		ctx2 := NewExecutionContext().
			SetIPField("device.ip", "8.8.8.8").
			SetIPList("management_nets", nets)
		result2, err := filter.Execute(ctx2)
		assert.NoError(t, err)
		assert.True(t, result2)
	})

	t.Run("not contains operator", func(t *testing.T) {
		filter, err := Compile(`name not contains "admin"`, nil)
		assert.NoError(t, err)

		ctx := NewExecutionContext().SetStringField("name", "superadmin")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)

		ctx2 := NewExecutionContext().SetStringField("name", "user")
		result2, err := filter.Execute(ctx2)
		assert.NoError(t, err)
		assert.True(t, result2)
	})

	t.Run("not in with logical operators", func(t *testing.T) {
		filter, err := Compile(
			`user.groups contains "network-admins" and device.ip not in $management_nets`,
			nil,
		)
		assert.NoError(t, err)

		nets := []string{"10.255.0.0/16", "172.16.0.0/12"}

		ctx := NewExecutionContext().
			SetArrayField("user.groups", []string{"network-admins"}).
			SetIPField("device.ip", "8.8.8.8").
			SetIPList("management_nets", nets)
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)

		ctx2 := NewExecutionContext().
			SetArrayField("user.groups", []string{"network-admins"}).
			SetIPField("device.ip", "10.255.1.1").
			SetIPList("management_nets", nets)
		result2, err := filter.Execute(ctx2)
		assert.NoError(t, err)
		assert.False(t, result2)

		ctx3 := NewExecutionContext().
			SetArrayField("user.groups", []string{"users"}).
			SetIPField("device.ip", "8.8.8.8").
			SetIPList("management_nets", nets)
		result3, err := filter.Execute(ctx3)
		assert.NoError(t, err)
		assert.False(t, result3)
	})

	t.Run("array index - first element", func(t *testing.T) {
		schema := NewSchema().AddField("tags", TypeArray)
		filter, err := Compile(`tags[0] == "first"`, schema)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetArrayField("tags", []string{"first", "second", "third"})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("array index - middle element", func(t *testing.T) {
		schema := NewSchema().AddField("tags", TypeArray)
		filter, err := Compile(`tags[1] == "second"`, schema)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetArrayField("tags", []string{"first", "second", "third"})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("array index - out of bounds", func(t *testing.T) {
		schema := NewSchema().AddField("tags", TypeArray)
		filter, err := Compile(`tags[10] == "test"`, schema)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetArrayField("tags", []string{"first", "second"})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("array index - negative index", func(t *testing.T) {
		schema := NewSchema().AddField("tags", TypeArray)
		filter, err := Compile(`tags[-1] == "test"`, schema)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetArrayField("tags", []string{"first", "second"})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("array index - integer array", func(t *testing.T) {
		schema := NewSchema().AddField("ports", TypeArray)
		filter, err := Compile(`ports[0] == 80`, schema)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetIntArrayField("ports", []int64{80, 443, 8080})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("array unpack - any element equals", func(t *testing.T) {
		schema := NewSchema().AddField("tags", TypeArray)
		filter, err := Compile(`tags[*] == "admin"`, schema)
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

	t.Run("array unpack - any element contains", func(t *testing.T) {
		schema := NewSchema().AddField("tags", TypeArray)
		filter, err := Compile(`tags[*] contains "test"`, schema)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetArrayField("tags", []string{"foo", "testing", "bar"})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
		ctx2 := NewExecutionContext().SetArrayField("tags", []string{"foo", "bar"})
		result2, err := filter.Execute(ctx2)
		assert.NoError(t, err)
		assert.False(t, result2)
	})

	t.Run("array unpack - any element matches", func(t *testing.T) {
		schema := NewSchema().AddField("emails", TypeArray)
		filter, err := Compile(`emails[*] matches ".*@example\\.com$"`, schema)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetArrayField("emails", []string{"foo@other.com", "bar@example.com"})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("array unpack - comparison operators", func(t *testing.T) {
		schema := NewSchema().AddField("ports", TypeArray)
		filter, err := Compile(`ports[*] > 1000`, schema)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetIntArrayField("ports", []int64{80, 443, 8080})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
		ctx2 := NewExecutionContext().SetIntArrayField("ports", []int64{80, 443})
		result2, err := filter.Execute(ctx2)
		assert.NoError(t, err)
		assert.False(t, result2)
	})

	t.Run("array unpack - empty array", func(t *testing.T) {
		schema := NewSchema().AddField("tags", TypeArray)
		filter, err := Compile(`tags[*] == "test"`, schema)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetArrayField("tags", []string{})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("array unpack - not equal", func(t *testing.T) {
		schema := NewSchema().AddField("tags", TypeArray)
		filter, err := Compile(`tags[*] != "banned"`, schema)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetArrayField("tags", []string{"admin", "user"})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("array unpack - wildcard", func(t *testing.T) {
		schema := NewSchema().AddField("hosts", TypeArray)
		filter, err := Compile(`hosts[*] wildcard "*.example.com"`, schema)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetArrayField("hosts", []string{"other.com", "www.example.com"})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("array unpack - in operator", func(t *testing.T) {
		schema := NewSchema().AddField("roles", TypeArray)
		filter, err := Compile(`roles[*] in {"admin", "superuser"}`, schema)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetArrayField("roles", []string{"user", "admin"})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("array unpack - non-array field", func(t *testing.T) {
		schema := NewSchema().AddField("name", TypeString)
		filter, err := Compile(`name[*] == "test"`, schema)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetStringField("name", "test")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("array unpack - missing field", func(t *testing.T) {
		schema := NewSchema().AddField("tags", TypeArray)
		filter, err := Compile(`tags[*] == "test"`, schema)
		assert.NoError(t, err)
		ctx := NewExecutionContext()
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("UnpackedArrayValue methods", func(t *testing.T) {
		uv := UnpackedArrayValue{Array: ArrayValue{StringValue("a"), StringValue("b")}}
		assert.Equal(t, TypeArray, uv.Type())
		assert.True(t, uv.IsTruthy())
		assert.Contains(t, uv.String(), "a")
		emptyUv := UnpackedArrayValue{Array: ArrayValue{}}
		assert.False(t, emptyUv.IsTruthy())
		uv2 := UnpackedArrayValue{Array: ArrayValue{StringValue("a"), StringValue("b")}}
		assert.True(t, uv.Equal(uv2))
		arr := ArrayValue{StringValue("a"), StringValue("b")}
		assert.True(t, uv.Equal(arr))
	})

	t.Run("range with non-int types", func(t *testing.T) {
		filter, err := Compile(`status in {200..299}`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetStringField("status", "250")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("unary not with nil", func(t *testing.T) {
		filter, err := Compile(`not missing`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext()
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("index on nil object", func(t *testing.T) {
		filter, err := Compile(`missing[0] == "test"`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext()
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("index with field reference key", func(t *testing.T) {
		filter, err := Compile(`data[key] == "val"`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().
			SetMapField("data", map[string]string{"mykey": "val"}).
			SetStringField("key", "mykey")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("unpack on nil array", func(t *testing.T) {
		filter, err := Compile(`missing[*] == "test"`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext()
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("unpack on non-array", func(t *testing.T) {
		filter, err := Compile(`name[*] == "test"`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetStringField("name", "test")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("list ref not found", func(t *testing.T) {
		filter, err := Compile(`role in $missing_list`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetStringField("role", "admin")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("range with nil start value", func(t *testing.T) {
		filter, err := Compile(`x in {missing..10}`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetIntField("x", 5)
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("range with nil end value", func(t *testing.T) {
		filter, err := Compile(`x in {1..missing}`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetIntField("x", 5)
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("short-circuit and - false and error", func(t *testing.T) {
		filter, err := Compile(`false and (name matches "[")`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetStringField("name", "test")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("short-circuit or - true or error", func(t *testing.T) {
		filter, err := Compile(`true or (name matches "[")`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetStringField("name", "test")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("xor evaluates both sides", func(t *testing.T) {
		filter, err := Compile(`true xor false`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext()
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("array with nil element string", func(t *testing.T) {
		arr := ArrayValue{StringValue("a"), nil, StringValue("c")}
		str := arr.String()
		assert.Contains(t, str, "nil")
	})

	t.Run("array equal with nil elements", func(t *testing.T) {
		arr1 := ArrayValue{StringValue("a"), nil}
		arr2 := ArrayValue{StringValue("a"), nil}
		assert.True(t, arr1.Equal(arr2))
		arr3 := ArrayValue{StringValue("a"), StringValue("b")}
		assert.False(t, arr1.Equal(arr3))
	})

	t.Run("array contains nil", func(t *testing.T) {
		arr := ArrayValue{StringValue("a"), nil, StringValue("c")}
		assert.True(t, arr.Contains(nil))
		assert.True(t, arr.Contains(StringValue("a")))
		assert.False(t, arr.Contains(StringValue("b")))
	})

	t.Run("map with nil value string", func(t *testing.T) {
		m := MapValue{"a": StringValue("x"), "b": nil}
		str := m.String()
		assert.Contains(t, str, "nil")
	})

	t.Run("map equal with nil values", func(t *testing.T) {
		m1 := MapValue{"a": nil}
		m2 := MapValue{"a": nil}
		assert.True(t, m1.Equal(m2))
		m3 := MapValue{"a": StringValue("x")}
		assert.False(t, m1.Equal(m3))
	})

	t.Run("array unpack with ne operator", func(t *testing.T) {
		filter, err := Compile(`tags[*] != "banned"`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetArrayField("tags", []string{"a", "b", "c"})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("array unpack with lt operator", func(t *testing.T) {
		filter, err := Compile(`nums[*] < 10`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetIntArrayField("nums", []int64{5, 15, 25})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("array unpack with le operator", func(t *testing.T) {
		filter, err := Compile(`nums[*] <= 5`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetIntArrayField("nums", []int64{5, 15, 25})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("array unpack with ge operator", func(t *testing.T) {
		filter, err := Compile(`nums[*] >= 25`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetIntArrayField("nums", []int64{5, 15, 25})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("array unpack with strict wildcard", func(t *testing.T) {
		filter, err := Compile(`hosts[*] strict wildcard "*.Example.com"`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetArrayField("hosts", []string{
			"api.Example.com",
			"www.other.com",
		})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})
}

func TestFilterFloat(t *testing.T) {
	t.Run("float literal comparison", func(t *testing.T) {
		filter, err := Compile(`score > 3.14`, nil)
		require.NoError(t, err)

		ctx := NewExecutionContext().SetFloatField("score", 4.0)
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)

		ctx2 := NewExecutionContext().SetFloatField("score", 2.0)
		result2, err := filter.Execute(ctx2)
		assert.NoError(t, err)
		assert.False(t, result2)
	})

	t.Run("float equality", func(t *testing.T) {
		filter, err := Compile(`score == 3.14`, nil)
		require.NoError(t, err)

		ctx := NewExecutionContext().SetFloatField("score", 3.14)
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("float inequality", func(t *testing.T) {
		filter, err := Compile(`score != 3.14`, nil)
		require.NoError(t, err)

		ctx := NewExecutionContext().SetFloatField("score", 2.71)
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("float less than", func(t *testing.T) {
		filter, err := Compile(`score < 10.5`, nil)
		require.NoError(t, err)

		ctx := NewExecutionContext().SetFloatField("score", 9.9)
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("float greater than or equal", func(t *testing.T) {
		filter, err := Compile(`score >= 5.0`, nil)
		require.NoError(t, err)

		ctx := NewExecutionContext().SetFloatField("score", 5.0)
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("float less than or equal", func(t *testing.T) {
		filter, err := Compile(`score <= 5.0`, nil)
		require.NoError(t, err)

		ctx := NewExecutionContext().SetFloatField("score", 5.0)
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)

		ctx2 := NewExecutionContext().SetFloatField("score", 5.1)
		result2, err := filter.Execute(ctx2)
		assert.NoError(t, err)
		assert.False(t, result2)
	})

	t.Run("mixed int and float comparison", func(t *testing.T) {
		filter, err := Compile(`score > 3`, nil)
		require.NoError(t, err)

		ctx := NewExecutionContext().SetFloatField("score", 3.5)
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("int field compared with float literal", func(t *testing.T) {
		filter, err := Compile(`count > 2.5`, nil)
		require.NoError(t, err)

		ctx := NewExecutionContext().SetIntField("count", 3)
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)

		ctx2 := NewExecutionContext().SetIntField("count", 2)
		result2, err := filter.Execute(ctx2)
		assert.NoError(t, err)
		assert.False(t, result2)
	})

	t.Run("negative float", func(t *testing.T) {
		filter, err := Compile(`temp > -10.5`, nil)
		require.NoError(t, err)

		ctx := NewExecutionContext().SetFloatField("temp", -5.0)
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("float in set", func(t *testing.T) {
		filter, err := Compile(`score in {1.5, 2.5, 3.5}`, nil)
		require.NoError(t, err)

		ctx := NewExecutionContext().SetFloatField("score", 2.5)
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)

		ctx2 := NewExecutionContext().SetFloatField("score", 4.0)
		result2, err := filter.Execute(ctx2)
		assert.NoError(t, err)
		assert.False(t, result2)
	})

	t.Run("float schema type validation", func(t *testing.T) {
		schema := NewSchema().AddField("score", TypeFloat)

		_, err := Compile(`score > 3.14`, schema)
		assert.NoError(t, err)

		_, err = Compile(`score contains "x"`, schema)
		assert.Error(t, err)
	})

	t.Run("float marshal unmarshal", func(t *testing.T) {
		filter, err := Compile(`score > 3.14`, nil)
		require.NoError(t, err)

		data, err := filter.MarshalBinary()
		require.NoError(t, err)

		restored := &Filter{}
		err = restored.UnmarshalBinary(data)
		require.NoError(t, err)

		ctx := NewExecutionContext().SetFloatField("score", 4.0)
		r1, _ := filter.Execute(ctx)
		r2, _ := restored.Execute(ctx)
		assert.Equal(t, r1, r2)
	})

	t.Run("float truthiness", func(t *testing.T) {
		assert.True(t, FloatValue(3.14).IsTruthy())
		assert.True(t, FloatValue(0.0).IsTruthy())
		assert.Equal(t, "3.14", FloatValue(3.14).String())
		assert.Equal(t, TypeFloat, FloatValue(0).Type())
	})

	t.Run("float equal", func(t *testing.T) {
		a, b := FloatValue(3.14), FloatValue(3.14)
		assert.True(t, a.Equal(b))
		assert.False(t, FloatValue(3.14).Equal(FloatValue(2.71)))
		assert.False(t, FloatValue(3.14).Equal(IntValue(3)))
	})

	t.Run("float type string", func(t *testing.T) {
		assert.Equal(t, "Float", TypeFloat.String())
	})

	t.Run("float comparison with non-numeric", func(t *testing.T) {
		filter, err := Compile(`x > 3.14`, nil)
		require.NoError(t, err)

		ctx := NewExecutionContext().SetStringField("x", "hello")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("float le and ge edge cases", func(t *testing.T) {
		filterLe, err := Compile(`x <= 3.14`, nil)
		require.NoError(t, err)

		filterGe, err := Compile(`x >= 3.14`, nil)
		require.NoError(t, err)

		ctx := NewExecutionContext().SetFloatField("x", 3.14)
		r1, _ := filterLe.Execute(ctx)
		assert.True(t, r1)
		r2, _ := filterGe.Execute(ctx)
		assert.True(t, r2)

		ctxLess := NewExecutionContext().SetFloatField("x", 2.0)
		r3, _ := filterLe.Execute(ctxLess)
		assert.True(t, r3)
		r4, _ := filterGe.Execute(ctxLess)
		assert.False(t, r4)
	})

	t.Run("float lt edge", func(t *testing.T) {
		filter, err := Compile(`x < 3.14`, nil)
		require.NoError(t, err)

		ctx := NewExecutionContext().SetFloatField("x", 3.14)
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("float max value", func(t *testing.T) {
		filter, err := Compile(`x > 0`, nil)
		require.NoError(t, err)

		ctx := NewExecutionContext().SetFloatField("x", math.MaxFloat64)
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("float max value equality", func(t *testing.T) {
		ctx := NewExecutionContext().SetFloatField("x", math.MaxFloat64)

		fv, ok := ctx.GetField("x")
		require.True(t, ok)
		assert.Equal(t, FloatValue(math.MaxFloat64), fv)
	})

	t.Run("float smallest positive", func(t *testing.T) {
		filter, err := Compile(`x > 0`, nil)
		require.NoError(t, err)

		ctx := NewExecutionContext().SetFloatField("x", math.SmallestNonzeroFloat64)
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("float negative max", func(t *testing.T) {
		filter, err := Compile(`x < 0`, nil)
		require.NoError(t, err)

		ctx := NewExecutionContext().SetFloatField("x", -math.MaxFloat64)
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("float infinity", func(t *testing.T) {
		filter, err := Compile(`x > 0`, nil)
		require.NoError(t, err)

		ctx := NewExecutionContext().SetFloatField("x", math.Inf(1))
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)

		ctx2 := NewExecutionContext().SetFloatField("x", math.Inf(-1))
		result2, err := filter.Execute(ctx2)
		assert.NoError(t, err)
		assert.False(t, result2)
	})

	t.Run("float NaN", func(t *testing.T) {
		filter, err := Compile(`x > 0`, nil)
		require.NoError(t, err)

		ctx := NewExecutionContext().SetFloatField("x", math.NaN())
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("float NaN equality", func(t *testing.T) {
		filter, err := Compile(`x == x`, nil)
		require.NoError(t, err)

		ctx := NewExecutionContext().SetFloatField("x", math.NaN())
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("float marshal roundtrip extreme values", func(t *testing.T) {
		extremes := []float64{
			math.MaxFloat64,
			-math.MaxFloat64,
			math.SmallestNonzeroFloat64,
			math.Inf(1),
			math.Inf(-1),
			0,
			-0,
		}

		for _, val := range extremes {
			filter, err := Compile(`x > 0`, nil)
			require.NoError(t, err)

			data, err := filter.MarshalBinary()
			require.NoError(t, err)

			restored := &Filter{}
			require.NoError(t, restored.UnmarshalBinary(data))

			ctx := NewExecutionContext().SetFloatField("x", val)
			r1, _ := filter.Execute(ctx)
			r2, _ := restored.Execute(ctx)
			assert.Equal(t, r1, r2)
		}
	})
}

func TestFilterEvalCoverageGaps(t *testing.T) {
	t.Run("range with start greater than end", func(t *testing.T) {
		filter, err := Compile(`x in {10..5}`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetIntField("x", 7)
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("range with nil values", func(t *testing.T) {
		filter, err := Compile(`x in {a..b}`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetIntField("x", 5)
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("range with non-int types", func(t *testing.T) {
		filter, err := Compile(`x in {a..b}`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().
			SetIntField("x", 5).
			SetStringField("a", "hello").
			SetStringField("b", "world")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("index on non-array non-map", func(t *testing.T) {
		filter, err := Compile(`data["key"] == "val"`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetStringField("data", "not a map")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("index on nil object", func(t *testing.T) {
		filter, err := Compile(`data["key"] == "val"`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext()
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("map access with missing key", func(t *testing.T) {
		filter, err := Compile(`data["missing"] == "val"`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetMapField("data", map[string]string{"key": "val"})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("array index out of bounds", func(t *testing.T) {
		filter, err := Compile(`tags[5] == "a"`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetArrayField("tags", []string{"a", "b"})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("array index negative", func(t *testing.T) {
		filter, err := Compile(`tags[0] == "a"`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetArrayField("tags", []string{"a"})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("unpack on non-array", func(t *testing.T) {
		filter, err := Compile(`name[*] == "a"`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetStringField("name", "test")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("unpack on nil field", func(t *testing.T) {
		filter, err := Compile(`tags[*] == "a"`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext()
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("not with nil operand", func(t *testing.T) {
		filter, err := Compile(`not missing`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext()
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("xor operator", func(t *testing.T) {
		filter, err := Compile(`a xor b`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetBoolField("a", true).SetBoolField("b", false)
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
		ctx2 := NewExecutionContext().SetBoolField("a", true).SetBoolField("b", true)
		result2, err := filter.Execute(ctx2)
		assert.NoError(t, err)
		assert.False(t, result2)
		ctx3 := NewExecutionContext().SetBoolField("a", false).SetBoolField("b", false)
		result3, err := filter.Execute(ctx3)
		assert.NoError(t, err)
		assert.False(t, result3)
	})

	t.Run("wildcard with nil left", func(t *testing.T) {
		filter, err := Compile(`name wildcard "*.com"`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext()
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("wildcard with non-string types", func(t *testing.T) {
		filter, err := Compile(`x wildcard "*.com"`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetIntField("x", 42)
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("all equal with nil left", func(t *testing.T) {
		filter, err := Compile(`tags === "a"`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext()
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("all equal with non-array", func(t *testing.T) {
		filter, err := Compile(`name === "a"`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetStringField("name", "a")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("all equal with empty array", func(t *testing.T) {
		filter, err := Compile(`tags === "a"`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetField("tags", ArrayValue{})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("any not equal with nil left", func(t *testing.T) {
		filter, err := Compile(`tags !== "a"`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext()
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("any not equal with non-array", func(t *testing.T) {
		filter, err := Compile(`name !== "a"`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetStringField("name", "a")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("any not equal with empty array", func(t *testing.T) {
		filter, err := Compile(`tags !== "a"`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetField("tags", ArrayValue{})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("in with nil left", func(t *testing.T) {
		filter, err := Compile(`x in {1, 2, 3}`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext()
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("in with non-array non-cidr right", func(t *testing.T) {
		filter, err := Compile(`x in y`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetIntField("x", 1).SetIntField("y", 1)
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("unknown function returns nil", func(t *testing.T) {
		filter, err := Compile(`unknown_func("test") == "test"`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext()
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("any with wrong arg count", func(t *testing.T) {
		filter := &Filter{
			expr:       &FunctionCallExpr{Name: "any", Arguments: []Expression{}},
			regexCache: make(map[string]*regexp.Regexp),
			cidrCache:  make(map[string]*net.IPNet),
		}
		ctx := NewExecutionContext()
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("all with wrong arg count", func(t *testing.T) {
		filter := &Filter{
			expr:       &FunctionCallExpr{Name: "all", Arguments: []Expression{}},
			regexCache: make(map[string]*regexp.Regexp),
			cidrCache:  make(map[string]*net.IPNet),
		}
		ctx := NewExecutionContext()
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("array in array - OR semantics", func(t *testing.T) {
		filter, err := Compile(`tags in allowed`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().
			SetArrayField("tags", []string{"x", "a"}).
			SetArrayField("allowed", []string{"a", "b", "c"})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
		ctx2 := NewExecutionContext().
			SetArrayField("tags", []string{"x", "y"}).
			SetArrayField("allowed", []string{"a", "b", "c"})
		result2, err := filter.Execute(ctx2)
		assert.NoError(t, err)
		assert.False(t, result2)
	})

	t.Run("IP in array with CIDR element nil skip", func(t *testing.T) {
		filter, err := Compile(`ip in ips`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().
			SetIPField("ip", "10.0.0.1").
			SetField("ips", ArrayValue{nil, IPValue{IP: nil}})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("equality with nil values", func(t *testing.T) {
		filter, err := Compile(`x == "test"`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext()
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("equality both nil", func(t *testing.T) {
		filter, err := Compile(`x == y`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext()
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("IP equality with string right", func(t *testing.T) {
		filter, err := Compile(`ip == "192.168.1.1"`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetIPField("ip", "192.168.1.1")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("IP equality with invalid string", func(t *testing.T) {
		filter, err := Compile(`ip == "not-an-ip"`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetIPField("ip", "192.168.1.1")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("string left equality with IP right", func(t *testing.T) {
		filter, err := Compile(`name == ip`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().
			SetStringField("name", "192.168.1.1").
			SetIPField("ip", "192.168.1.1")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("string left equality with IP right invalid string", func(t *testing.T) {
		filter, err := Compile(`name == ip`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().
			SetStringField("name", "not-an-ip").
			SetIPField("ip", "192.168.1.1")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("matches with nil left", func(t *testing.T) {
		filter, err := Compile(`name matches "^test"`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext()
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("matches with non-string types", func(t *testing.T) {
		filter, err := Compile(`x matches "^test"`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetIntField("x", 42)
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("contains with nil operands", func(t *testing.T) {
		filter, err := Compile(`name contains "test"`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext()
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("contains with non-string non-array", func(t *testing.T) {
		filter, err := Compile(`x contains "test"`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetIntField("x", 42)
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("comparison with nil operands", func(t *testing.T) {
		filter, err := Compile(`x > 5`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext()
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("comparison with non-int types", func(t *testing.T) {
		filter, err := Compile(`x > 5`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetStringField("x", "hello")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("unpacked array with wildcard operator", func(t *testing.T) {
		filter, err := Compile(`names[*] wildcard "*.com"`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetArrayField("names", []string{"example.com", "test.org"})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
		ctx2 := NewExecutionContext().SetArrayField("names", []string{"test.org", "test.net"})
		result2, err := filter.Execute(ctx2)
		assert.NoError(t, err)
		assert.False(t, result2)
	})

	t.Run("unpacked array with strict wildcard operator", func(t *testing.T) {
		filter, err := Compile(`names[*] strict wildcard "*.COM"`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetArrayField("names", []string{"test.COM", "test.org"})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
		ctx2 := NewExecutionContext().SetArrayField("names", []string{"test.com"})
		result2, err := filter.Execute(ctx2)
		assert.NoError(t, err)
		assert.False(t, result2)
	})

	t.Run("unpacked array with in operator", func(t *testing.T) {
		filter, err := Compile(`ports[*] in {80, 443}`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetIntArrayField("ports", []int64{8080, 80})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("unpacked array empty", func(t *testing.T) {
		filter, err := Compile(`tags[*] == "a"`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetField("tags", ArrayValue{})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("unpacked array with ne operator", func(t *testing.T) {
		filter, err := Compile(`tags[*] != "bad"`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetArrayField("tags", []string{"good", "bad"})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("unpacked array with lt operator", func(t *testing.T) {
		filter, err := Compile(`vals[*] < 5`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetIntArrayField("vals", []int64{10, 3})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("unpacked array with gt operator", func(t *testing.T) {
		filter, err := Compile(`vals[*] > 5`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetIntArrayField("vals", []int64{1, 10})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("unpacked array with le operator", func(t *testing.T) {
		filter, err := Compile(`vals[*] <= 5`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetIntArrayField("vals", []int64{10, 5})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("unpacked array with ge operator", func(t *testing.T) {
		filter, err := Compile(`vals[*] >= 5`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetIntArrayField("vals", []int64{1, 5})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("unpacked array with contains operator", func(t *testing.T) {
		filter, err := Compile(`tags[*] contains "test"`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetArrayField("tags", []string{"no", "testing"})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("unpacked array with matches operator", func(t *testing.T) {
		filter, err := Compile(`tags[*] matches "^test"`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetArrayField("tags", []string{"no", "testing"})
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("logical or short circuit true", func(t *testing.T) {
		filter, err := Compile(`a or b`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetBoolField("a", true)
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("logical and short circuit false", func(t *testing.T) {
		filter, err := Compile(`a and b`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetBoolField("a", false)
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("cidr with negative bits via direct construction", func(t *testing.T) {
		filter := &Filter{
			expr: &BinaryExpr{
				Left: &FunctionCallExpr{
					Name: "cidr",
					Arguments: []Expression{
						&FieldExpr{Name: "ip"},
						&LiteralExpr{Value: IntValue(-5)},
					},
				},
				Operator: TokenEq,
				Right:    &LiteralExpr{Value: StringValue("0.0.0.0/0")},
			},
			regexCache: make(map[string]*regexp.Regexp),
			cidrCache:  make(map[string]*net.IPNet),
		}
		ctx := NewExecutionContext().SetIPField("ip", "192.168.1.100")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("cidr6 with negative bits via direct construction", func(t *testing.T) {
		filter := &Filter{
			expr: &BinaryExpr{
				Left: &FunctionCallExpr{
					Name: "cidr6",
					Arguments: []Expression{
						&FieldExpr{Name: "ip"},
						&LiteralExpr{Value: IntValue(-5)},
					},
				},
				Operator: TokenEq,
				Right:    &LiteralExpr{Value: StringValue("::/0")},
			},
			regexCache: make(map[string]*regexp.Regexp),
			cidrCache:  make(map[string]*net.IPNet),
		}
		ctx := NewExecutionContext().SetIPField("ip", "2001:db8::1")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("cidr6 with bits over 128", func(t *testing.T) {
		filter := &Filter{
			expr: &BinaryExpr{
				Left: &FunctionCallExpr{
					Name: "cidr6",
					Arguments: []Expression{
						&FieldExpr{Name: "ip"},
						&LiteralExpr{Value: IntValue(200)},
					},
				},
				Operator: TokenEq,
				Right:    &LiteralExpr{Value: StringValue("2001:db8::1/128")},
			},
			regexCache: make(map[string]*regexp.Regexp),
			cidrCache:  make(map[string]*net.IPNet),
		}
		ctx := NewExecutionContext().SetIPField("ip", "2001:db8::1")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("any with error in evaluation", func(t *testing.T) {
		filter, err := Compile(`any(tags[*] matches pattern)`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().
			SetArrayField("tags", []string{"test"}).
			SetStringField("pattern", "[invalid")
		_, err = filter.Execute(ctx)
		assert.Error(t, err)
	})

	t.Run("all with error in matches evaluation", func(t *testing.T) {
		filter, err := Compile(`all(tags[*] matches pattern)`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().
			SetArrayField("tags", []string{"test"}).
			SetStringField("pattern", "[invalid")
		_, err = filter.Execute(ctx)
		assert.Error(t, err)
	})

	t.Run("all with binary expr and non-unpacked left", func(t *testing.T) {
		filter, err := Compile(`all(x == 1)`, nil)
		assert.NoError(t, err)
		ctx := NewExecutionContext().SetIntField("x", 1)
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})
}
