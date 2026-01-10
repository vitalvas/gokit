package wirefilter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
			for b.Loop() {
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

func FuzzParser(f *testing.F) {
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
	f.Add(`((a == 1) or (b == 2)) and c == 3`)

	f.Fuzz(func(_ *testing.T, input string) {
		lexer := NewLexer(input)
		parser := NewParser(lexer)
		_, _ = parser.Parse()
	})
}

func TestParser(t *testing.T) {
	t.Run("simple equality", func(t *testing.T) {
		input := `name == "test"`
		lexer := NewLexer(input)
		parser := NewParser(lexer)

		expr, err := parser.Parse()
		assert.NoError(t, err)
		assert.NotNil(t, expr)

		binExpr, ok := expr.(*BinaryExpr)
		assert.True(t, ok)
		assert.Equal(t, TokenEq, binExpr.Operator)

		field, ok := binExpr.Left.(*FieldExpr)
		assert.True(t, ok)
		assert.Equal(t, "name", field.Name)

		literal, ok := binExpr.Right.(*LiteralExpr)
		assert.True(t, ok)
		assert.Equal(t, StringValue("test"), literal.Value)
	})

	t.Run("logical and", func(t *testing.T) {
		input := `age > 18 && active == true`
		lexer := NewLexer(input)
		parser := NewParser(lexer)

		expr, err := parser.Parse()
		assert.NoError(t, err)
		assert.NotNil(t, expr)

		binExpr, ok := expr.(*BinaryExpr)
		assert.True(t, ok)
		assert.Equal(t, TokenAnd, binExpr.Operator)
	})

	t.Run("not expression", func(t *testing.T) {
		input := `not active`
		lexer := NewLexer(input)
		parser := NewParser(lexer)

		expr, err := parser.Parse()
		assert.NoError(t, err)
		assert.NotNil(t, expr)

		unaryExpr, ok := expr.(*UnaryExpr)
		assert.True(t, ok)
		assert.Equal(t, TokenNot, unaryExpr.Operator)
	})

	t.Run("grouped expression", func(t *testing.T) {
		input := `(a == 1 || b == 2) && c == 3`
		lexer := NewLexer(input)
		parser := NewParser(lexer)

		expr, err := parser.Parse()
		assert.NoError(t, err)
		assert.NotNil(t, expr)

		binExpr, ok := expr.(*BinaryExpr)
		assert.True(t, ok)
		assert.Equal(t, TokenAnd, binExpr.Operator)
	})

	t.Run("in expression with array", func(t *testing.T) {
		input := `port in {80, 443, 8080}`
		lexer := NewLexer(input)
		parser := NewParser(lexer)

		expr, err := parser.Parse()
		assert.NoError(t, err)
		assert.NotNil(t, expr)

		binExpr, ok := expr.(*BinaryExpr)
		assert.True(t, ok)
		assert.Equal(t, TokenIn, binExpr.Operator)

		arrayExpr, ok := binExpr.Right.(*ArrayExpr)
		assert.True(t, ok)
		assert.Equal(t, 3, len(arrayExpr.Elements))
	})

	t.Run("contains expression", func(t *testing.T) {
		input := `message contains "error"`
		lexer := NewLexer(input)
		parser := NewParser(lexer)

		expr, err := parser.Parse()
		assert.NoError(t, err)
		assert.NotNil(t, expr)

		binExpr, ok := expr.(*BinaryExpr)
		assert.True(t, ok)
		assert.Equal(t, TokenContains, binExpr.Operator)
	})

	t.Run("matches expression", func(t *testing.T) {
		input := `email matches "^.*@example\\.com$"`
		lexer := NewLexer(input)
		parser := NewParser(lexer)

		expr, err := parser.Parse()
		assert.NoError(t, err)
		assert.NotNil(t, expr)

		binExpr, ok := expr.(*BinaryExpr)
		assert.True(t, ok)
		assert.Equal(t, TokenMatches, binExpr.Operator)
	})
}
