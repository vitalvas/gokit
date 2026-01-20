package wirefilter

import (
	"testing"

	"github.com/stretchr/testify/assert"
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

func TestFilter(t *testing.T) {
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

	t.Run("all equal operator - non-array value", func(t *testing.T) {
		schema := NewSchema().
			AddField("name", TypeString)

		filter, err := Compile(`name === "test"`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetStringField("name", "test")

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("any not equal operator - non-array value", func(t *testing.T) {
		schema := NewSchema().
			AddField("name", TypeString)

		filter, err := Compile(`name !== "test"`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetStringField("name", "test")

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
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

	t.Run("context SetBytesField", func(t *testing.T) {
		ctx := NewExecutionContext().
			SetBytesField("data", []byte("test data"))

		val, ok := ctx.GetField("data")
		assert.True(t, ok)
		assert.Equal(t, TypeBytes, val.Type())
		assert.Equal(t, "test data", val.String())
	})

	t.Run("compile without schema", func(t *testing.T) {
		filter, err := Compile(`http.host == "test"`, nil)
		assert.NoError(t, err)
		assert.NotNil(t, filter)
	})

	t.Run("execute returns error on nil result", func(t *testing.T) {
		filter, err := Compile(`http.host`, nil)
		assert.NoError(t, err)

		ctx := NewExecutionContext()
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("unary not on non-existent field", func(t *testing.T) {
		filter, err := Compile(`not http.host`, nil)
		assert.NoError(t, err)

		ctx := NewExecutionContext()
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
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
			{true, true, false},   // T xor T = F
			{true, false, true},   // T xor F = T
			{false, true, true},   // F xor T = T
			{false, false, false}, // F xor F = F
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
			{"WWW.EXAMPLE.COM", true}, // case insensitive
			{"Api.Example.Com", true}, // case insensitive
			{"example.com", false},    // no prefix
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
			{"/API/V1/USERS/789", true}, // case insensitive
			{"/api/users/123", false},   // missing version segment
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
			{"www.example.com", false}, // case sensitive - lowercase fails
			{"WWW.EXAMPLE.COM", false}, // case sensitive - uppercase fails
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
			{"abc", true}, // case insensitive
			{"AC", false}, // missing char
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

		// Dot should be literal, not regex wildcard
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

		// Missing field
		ctx := NewExecutionContext()

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("wildcard with non-string types", func(t *testing.T) {
		schema := NewSchema().
			AddField("count", TypeInt)

		filter, err := Compile(`count wildcard "123"`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetIntField("count", 123)

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result) // Non-string types should return false
	})

	t.Run("xor with nil values", func(t *testing.T) {
		schema := NewSchema().
			AddField("a", TypeBool).
			AddField("b", TypeBool)

		filter, err := Compile(`a xor b`, schema)
		assert.NoError(t, err)

		// Only a is set
		ctx := NewExecutionContext().
			SetBoolField("a", true)

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result) // true xor nil(false) = true
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

	t.Run("array index - first element", func(t *testing.T) {
		schema := NewSchema().
			AddField("tags", TypeArray)

		filter, err := Compile(`tags[0] == "first"`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetArrayField("tags", []string{"first", "second", "third"})

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("array index - middle element", func(t *testing.T) {
		schema := NewSchema().
			AddField("tags", TypeArray)

		filter, err := Compile(`tags[1] == "second"`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetArrayField("tags", []string{"first", "second", "third"})

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("array index - out of bounds", func(t *testing.T) {
		schema := NewSchema().
			AddField("tags", TypeArray)

		filter, err := Compile(`tags[10] == "test"`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetArrayField("tags", []string{"first", "second"})

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("array index - negative index", func(t *testing.T) {
		schema := NewSchema().
			AddField("tags", TypeArray)

		filter, err := Compile(`tags[-1] == "test"`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetArrayField("tags", []string{"first", "second"})

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("array index - integer array", func(t *testing.T) {
		schema := NewSchema().
			AddField("ports", TypeArray)

		filter, err := Compile(`ports[0] == 80`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetIntArrayField("ports", []int64{80, 443, 8080})

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("array unpack - any element equals", func(t *testing.T) {
		schema := NewSchema().
			AddField("tags", TypeArray)

		filter, err := Compile(`tags[*] == "admin"`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetArrayField("tags", []string{"user", "admin", "guest"})

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)

		ctx2 := NewExecutionContext().
			SetArrayField("tags", []string{"user", "guest"})

		result2, err := filter.Execute(ctx2)
		assert.NoError(t, err)
		assert.False(t, result2)
	})

	t.Run("array unpack - any element contains", func(t *testing.T) {
		schema := NewSchema().
			AddField("tags", TypeArray)

		filter, err := Compile(`tags[*] contains "test"`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetArrayField("tags", []string{"foo", "testing", "bar"})

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)

		ctx2 := NewExecutionContext().
			SetArrayField("tags", []string{"foo", "bar"})

		result2, err := filter.Execute(ctx2)
		assert.NoError(t, err)
		assert.False(t, result2)
	})

	t.Run("array unpack - any element matches", func(t *testing.T) {
		schema := NewSchema().
			AddField("emails", TypeArray)

		filter, err := Compile(`emails[*] matches ".*@example\\.com$"`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetArrayField("emails", []string{"foo@other.com", "bar@example.com"})

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("array unpack - comparison operators", func(t *testing.T) {
		schema := NewSchema().
			AddField("ports", TypeArray)

		filter, err := Compile(`ports[*] > 1000`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetIntArrayField("ports", []int64{80, 443, 8080})

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)

		ctx2 := NewExecutionContext().
			SetIntArrayField("ports", []int64{80, 443})

		result2, err := filter.Execute(ctx2)
		assert.NoError(t, err)
		assert.False(t, result2)
	})

	t.Run("array unpack - empty array", func(t *testing.T) {
		schema := NewSchema().
			AddField("tags", TypeArray)

		filter, err := Compile(`tags[*] == "test"`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetArrayField("tags", []string{})

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("array unpack - not equal", func(t *testing.T) {
		schema := NewSchema().
			AddField("tags", TypeArray)

		filter, err := Compile(`tags[*] != "banned"`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetArrayField("tags", []string{"admin", "user"})

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("array unpack - wildcard", func(t *testing.T) {
		schema := NewSchema().
			AddField("hosts", TypeArray)

		filter, err := Compile(`hosts[*] wildcard "*.example.com"`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetArrayField("hosts", []string{"other.com", "www.example.com"})

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("array unpack - in operator", func(t *testing.T) {
		schema := NewSchema().
			AddField("roles", TypeArray)

		filter, err := Compile(`roles[*] in {"admin", "superuser"}`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetArrayField("roles", []string{"user", "admin"})

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
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

	t.Run("array unpack - non-array field", func(t *testing.T) {
		schema := NewSchema().
			AddField("name", TypeString)

		filter, err := Compile(`name[*] == "test"`, schema)
		assert.NoError(t, err)

		ctx := NewExecutionContext().
			SetStringField("name", "test")

		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("array unpack - missing field", func(t *testing.T) {
		schema := NewSchema().
			AddField("tags", TypeArray)

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

	// Function tests
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
		// concat with no args returns empty string
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
		// Start beyond string length
		filter, err := Compile(`substring(name, 100) == ""`, nil)
		assert.NoError(t, err)

		ctx := NewExecutionContext().SetStringField("name", "hello")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)

		// Negative start
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
		assert.True(t, result) // Returns original on decode error
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
		filter, err := Compile(`cidr(ip, 24, 64) == "192.168.1.0"`, nil)
		assert.NoError(t, err)

		ctx := NewExecutionContext().SetIPField("ip", "192.168.1.100")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)

		// Different subnet
		ctx2 := NewExecutionContext().SetIPField("ip", "192.168.2.100")
		result2, err := filter.Execute(ctx2)
		assert.NoError(t, err)
		assert.False(t, result2)
	})

	t.Run("function cidr - IPv4 /16", func(t *testing.T) {
		filter, err := Compile(`cidr(ip, 16, 64) == "192.168.0.0"`, nil)
		assert.NoError(t, err)

		ctx := NewExecutionContext().SetIPField("ip", "192.168.100.50")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("function cidr - IPv6", func(t *testing.T) {
		filter, err := Compile(`cidr(ip, 24, 64) == "2001:db8::"`, nil)
		assert.NoError(t, err)

		ctx := NewExecutionContext().SetIPField("ip", "2001:db8::1234")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)

		// Different /64 subnet
		ctx2 := NewExecutionContext().SetIPField("ip", "2001:db8:1::1234")
		result2, err := filter.Execute(ctx2)
		assert.NoError(t, err)
		assert.False(t, result2)
	})

	t.Run("function cidr - edge cases", func(t *testing.T) {
		// /32 mask (full IP)
		filter, err := Compile(`cidr(ip, 32, 128) == "192.168.1.100"`, nil)
		assert.NoError(t, err)

		ctx := NewExecutionContext().SetIPField("ip", "192.168.1.100")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)

		// /0 mask (all zeros)
		filter2, err := Compile(`cidr(ip, 0, 0) == "0.0.0.0"`, nil)
		assert.NoError(t, err)

		result2, err := filter2.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result2)
	})

	t.Run("function cidr - wrong types", func(t *testing.T) {
		filter, err := Compile(`cidr(name, 24, 64) == "192.168.1.0"`, nil)
		assert.NoError(t, err)

		ctx := NewExecutionContext().SetStringField("name", "not an ip")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("function cidr6 - IPv4", func(t *testing.T) {
		// cidr6 with IPv4 caps at 32
		filter, err := Compile(`cidr6(ip, 24) == "192.168.1.0"`, nil)
		assert.NoError(t, err)

		ctx := NewExecutionContext().SetIPField("ip", "192.168.1.100")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("function cidr6 - IPv4 with bits > 32", func(t *testing.T) {
		// cidr6 with bits > 32 for IPv4 should cap at 32
		filter, err := Compile(`cidr6(ip, 64) == "192.168.1.100"`, nil)
		assert.NoError(t, err)

		ctx := NewExecutionContext().SetIPField("ip", "192.168.1.100")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("function cidr6 - IPv6", func(t *testing.T) {
		filter, err := Compile(`cidr6(ip, 64) == "2001:db8::"`, nil)
		assert.NoError(t, err)

		ctx := NewExecutionContext().SetIPField("ip", "2001:db8::abcd:1234")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("function cidr6 - wrong types", func(t *testing.T) {
		filter, err := Compile(`cidr6(name, 64) == "2001:db8::"`, nil)
		assert.NoError(t, err)

		ctx := NewExecutionContext().SetStringField("name", "not an ip")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("function cidr - nil arguments", func(t *testing.T) {
		filter, err := Compile(`cidr(ip, 24, 64) == "192.168.1.0"`, nil)
		assert.NoError(t, err)

		ctx := NewExecutionContext()
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("function cidr6 - nil arguments", func(t *testing.T) {
		filter, err := Compile(`cidr6(ip, 64) == "2001:db8::"`, nil)
		assert.NoError(t, err)

		ctx := NewExecutionContext()
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	// Additional coverage tests
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

	t.Run("index with non-literal index rejected", func(t *testing.T) {
		// Non-literal indices like tags[idx] are rejected at parse time
		_, err := Compile(`tags[idx] == "test"`, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "index must be a string or integer literal")
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

	// Tests for report.md fixes

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
		// false and (error expression) should not evaluate right side
		filter, err := Compile(`false and (name matches "[")`, nil)
		assert.NoError(t, err)

		ctx := NewExecutionContext().SetStringField("name", "test")
		result, err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("short-circuit or - true or error", func(t *testing.T) {
		// true or (error expression) should not evaluate right side
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
}
