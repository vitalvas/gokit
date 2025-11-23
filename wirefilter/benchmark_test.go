package wirefilter

import (
	"testing"
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
			for i := 0; i < b.N; i++ {
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
			for i := 0; i < b.N; i++ {
				_, err := filter.Execute(ctx)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkLexer(b *testing.B) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "simple expression",
			input: `http.host == "example.com"`,
		},
		{
			name:  "complex expression",
			input: `http.host == "example.com" and http.status >= 400 or http.path contains "/api"`,
		},
		{
			name:  "array expression",
			input: `http.status in {200, 201, 204, 301, 302, 304}`,
		},
		{
			name:  "range expression",
			input: `port in {80..100, 443, 8000..9000}`,
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				lexer := NewLexer(tt.input)
				for {
					tok := lexer.NextToken()
					if tok.Type == TokenEOF {
						break
					}
				}
			}
		})
	}
}

func BenchmarkParser(b *testing.B) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "simple expression",
			input: `http.host == "example.com"`,
		},
		{
			name:  "complex expression",
			input: `http.host == "example.com" and http.status >= 400 or http.path contains "/api"`,
		},
		{
			name:  "nested parentheses",
			input: `((http.host == "example.com" and http.status == 200) or (http.host == "test.com" and http.status == 404))`,
		},
		{
			name:  "array expression",
			input: `http.status in {200, 201, 204, 301, 302, 304}`,
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				lexer := NewLexer(tt.input)
				parser := NewParser(lexer)
				_, err := parser.Parse()
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkValueOperations(b *testing.B) {
	b.Run("string equality", func(b *testing.B) {
		v1 := StringValue("example.com")
		v2 := StringValue("example.com")
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			v1.Equal(v2)
		}
	})

	b.Run("int comparison", func(b *testing.B) {
		v1 := IntValue(200)
		v2 := IntValue(200)
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			v1.Equal(v2)
		}
	})

	b.Run("ip equality", func(b *testing.B) {
		v1 := IPValue{IP: []byte{192, 168, 1, 1}}
		v2 := IPValue{IP: []byte{192, 168, 1, 1}}
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			v1.Equal(v2)
		}
	})

	b.Run("array contains", func(b *testing.B) {
		arr := ArrayValue{IntValue(1), IntValue(2), IntValue(3), IntValue(4), IntValue(5)}
		val := IntValue(3)
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			arr.Contains(val)
		}
	})
}

func BenchmarkIPOperations(b *testing.B) {
	b.Run("ipv4 in cidr", func(b *testing.B) {
		ip := []byte{192, 168, 1, 1}
		cidr := "192.168.0.0/16"
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
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
		for i := 0; i < b.N; i++ {
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
		for i := 0; i < b.N; i++ {
			ContainsString(haystack, needle)
		}
	})

	b.Run("regex match", func(b *testing.B) {
		value := "example.com"
		pattern := "^example\\..*"
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, err := MatchesRegex(value, pattern)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
