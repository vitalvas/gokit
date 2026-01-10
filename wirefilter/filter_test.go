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
}
