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

	t.Run("parser errors method", func(t *testing.T) {
		input := `field ==`
		lexer := NewLexer(input)
		parser := NewParser(lexer)

		_, err := parser.Parse()
		assert.Error(t, err)
		errors := parser.Errors()
		assert.NotEmpty(t, errors)
	})

	t.Run("empty array expression", func(t *testing.T) {
		input := `field in {}`
		lexer := NewLexer(input)
		parser := NewParser(lexer)

		expr, err := parser.Parse()
		assert.NoError(t, err)
		assert.NotNil(t, expr)

		binExpr, ok := expr.(*BinaryExpr)
		assert.True(t, ok)

		arrayExpr, ok := binExpr.Right.(*ArrayExpr)
		assert.True(t, ok)
		assert.Empty(t, arrayExpr.Elements)
	})

	t.Run("range expression in array", func(t *testing.T) {
		input := `status in {200..299}`
		lexer := NewLexer(input)
		parser := NewParser(lexer)

		expr, err := parser.Parse()
		assert.NoError(t, err)
		assert.NotNil(t, expr)

		binExpr, ok := expr.(*BinaryExpr)
		assert.True(t, ok)

		arrayExpr, ok := binExpr.Right.(*ArrayExpr)
		assert.True(t, ok)
		assert.Equal(t, 1, len(arrayExpr.Elements))

		rangeExpr, ok := arrayExpr.Elements[0].(*RangeExpr)
		assert.True(t, ok)
		assert.NotNil(t, rangeExpr.Start)
		assert.NotNil(t, rangeExpr.End)
	})

	t.Run("mixed array with ranges", func(t *testing.T) {
		input := `status in {100, 200..299, 400}`
		lexer := NewLexer(input)
		parser := NewParser(lexer)

		expr, err := parser.Parse()
		assert.NoError(t, err)
		assert.NotNil(t, expr)

		binExpr, ok := expr.(*BinaryExpr)
		assert.True(t, ok)

		arrayExpr, ok := binExpr.Right.(*ArrayExpr)
		assert.True(t, ok)
		assert.Equal(t, 3, len(arrayExpr.Elements))
	})

	t.Run("ip token in expression", func(t *testing.T) {
		input := `ip.src == ip.dst`
		lexer := NewLexer(input)
		parser := NewParser(lexer)

		expr, err := parser.Parse()
		assert.NoError(t, err)
		assert.NotNil(t, expr)
	})

	t.Run("precedence - or lower than and", func(t *testing.T) {
		input := `a == 1 or b == 2 and c == 3`
		lexer := NewLexer(input)
		parser := NewParser(lexer)

		expr, err := parser.Parse()
		assert.NoError(t, err)
		assert.NotNil(t, expr)

		binExpr, ok := expr.(*BinaryExpr)
		assert.True(t, ok)
		assert.Equal(t, TokenOr, binExpr.Operator)
	})

	t.Run("boolean literal true", func(t *testing.T) {
		input := `active == true`
		lexer := NewLexer(input)
		parser := NewParser(lexer)

		expr, err := parser.Parse()
		assert.NoError(t, err)

		binExpr := expr.(*BinaryExpr)
		literal := binExpr.Right.(*LiteralExpr)
		assert.Equal(t, BoolValue(true), literal.Value)
	})

	t.Run("boolean literal false", func(t *testing.T) {
		input := `active == false`
		lexer := NewLexer(input)
		parser := NewParser(lexer)

		expr, err := parser.Parse()
		assert.NoError(t, err)

		binExpr := expr.(*BinaryExpr)
		literal := binExpr.Right.(*LiteralExpr)
		assert.Equal(t, BoolValue(false), literal.Value)
	})

	t.Run("contains with array", func(t *testing.T) {
		input := `tags contains {"a", "b"}`
		lexer := NewLexer(input)
		parser := NewParser(lexer)

		expr, err := parser.Parse()
		assert.NoError(t, err)
		assert.NotNil(t, expr)

		binExpr, ok := expr.(*BinaryExpr)
		assert.True(t, ok)
		assert.Equal(t, TokenContains, binExpr.Operator)

		arrayExpr, ok := binExpr.Right.(*ArrayExpr)
		assert.True(t, ok)
		assert.Equal(t, 2, len(arrayExpr.Elements))
	})

	t.Run("integer literal", func(t *testing.T) {
		input := `status == 200`
		lexer := NewLexer(input)
		parser := NewParser(lexer)

		expr, err := parser.Parse()
		assert.NoError(t, err)

		binExpr := expr.(*BinaryExpr)
		literal := binExpr.Right.(*LiteralExpr)
		assert.Equal(t, IntValue(200), literal.Value)
	})

	t.Run("string literal", func(t *testing.T) {
		input := `name == "test"`
		lexer := NewLexer(input)
		parser := NewParser(lexer)

		expr, err := parser.Parse()
		assert.NoError(t, err)

		binExpr := expr.(*BinaryExpr)
		literal := binExpr.Right.(*LiteralExpr)
		assert.Equal(t, StringValue("test"), literal.Value)
	})

	t.Run("in with string cidr", func(t *testing.T) {
		input := `ip.src in "192.168.0.0/16"`
		lexer := NewLexer(input)
		parser := NewParser(lexer)

		expr, err := parser.Parse()
		assert.NoError(t, err)

		binExpr, ok := expr.(*BinaryExpr)
		assert.True(t, ok)
		assert.Equal(t, TokenIn, binExpr.Operator)

		literal, ok := binExpr.Right.(*LiteralExpr)
		assert.True(t, ok)
		assert.Equal(t, StringValue("192.168.0.0/16"), literal.Value)
	})

	t.Run("contains with string", func(t *testing.T) {
		input := `path contains "/api"`
		lexer := NewLexer(input)
		parser := NewParser(lexer)

		expr, err := parser.Parse()
		assert.NoError(t, err)

		binExpr, ok := expr.(*BinaryExpr)
		assert.True(t, ok)
		assert.Equal(t, TokenContains, binExpr.Operator)

		literal, ok := binExpr.Right.(*LiteralExpr)
		assert.True(t, ok)
		assert.Equal(t, StringValue("/api"), literal.Value)
	})

	t.Run("xor expression with symbol", func(t *testing.T) {
		input := `a == 1 ^^ b == 2`
		lexer := NewLexer(input)
		parser := NewParser(lexer)

		expr, err := parser.Parse()
		assert.NoError(t, err)
		assert.NotNil(t, expr)

		binExpr, ok := expr.(*BinaryExpr)
		assert.True(t, ok)
		assert.Equal(t, TokenXor, binExpr.Operator)
	})

	t.Run("xor expression with keyword", func(t *testing.T) {
		input := `a == 1 xor b == 2`
		lexer := NewLexer(input)
		parser := NewParser(lexer)

		expr, err := parser.Parse()
		assert.NoError(t, err)
		assert.NotNil(t, expr)

		binExpr, ok := expr.(*BinaryExpr)
		assert.True(t, ok)
		assert.Equal(t, TokenXor, binExpr.Operator)
	})

	t.Run("precedence - xor between and and or", func(t *testing.T) {
		// a or b xor c and d should parse as: a or ((b xor c) and d)
		// which becomes: a or (b xor (c and d))
		// Actually with OR < XOR < AND: a or (b xor (c and d))
		input := `a == 1 or b == 2 xor c == 3 and d == 4`
		lexer := NewLexer(input)
		parser := NewParser(lexer)

		expr, err := parser.Parse()
		assert.NoError(t, err)
		assert.NotNil(t, expr)

		// Top level should be OR (lowest precedence)
		binExpr, ok := expr.(*BinaryExpr)
		assert.True(t, ok)
		assert.Equal(t, TokenOr, binExpr.Operator)

		// Right side of OR should be XOR
		rightBin, ok := binExpr.Right.(*BinaryExpr)
		assert.True(t, ok)
		assert.Equal(t, TokenXor, rightBin.Operator)

		// Right side of XOR should be AND (highest precedence among these)
		rightRightBin, ok := rightBin.Right.(*BinaryExpr)
		assert.True(t, ok)
		assert.Equal(t, TokenAnd, rightRightBin.Operator)
	})

	t.Run("matches with tilde alias", func(t *testing.T) {
		input := `email ~ "^.*@example\\.com$"`
		lexer := NewLexer(input)
		parser := NewParser(lexer)

		expr, err := parser.Parse()
		assert.NoError(t, err)
		assert.NotNil(t, expr)

		binExpr, ok := expr.(*BinaryExpr)
		assert.True(t, ok)
		assert.Equal(t, TokenMatches, binExpr.Operator)
	})

	t.Run("not with exclamation alias", func(t *testing.T) {
		input := `! active`
		lexer := NewLexer(input)
		parser := NewParser(lexer)

		expr, err := parser.Parse()
		assert.NoError(t, err)
		assert.NotNil(t, expr)

		unaryExpr, ok := expr.(*UnaryExpr)
		assert.True(t, ok)
		assert.Equal(t, TokenNot, unaryExpr.Operator)
	})

	t.Run("wildcard expression", func(t *testing.T) {
		input := `host wildcard "*.example.com"`
		lexer := NewLexer(input)
		parser := NewParser(lexer)

		expr, err := parser.Parse()
		assert.NoError(t, err)
		assert.NotNil(t, expr)

		binExpr, ok := expr.(*BinaryExpr)
		assert.True(t, ok)
		assert.Equal(t, TokenWildcard, binExpr.Operator)
	})

	t.Run("strict wildcard expression", func(t *testing.T) {
		input := `host strict wildcard "*.Example.com"`
		lexer := NewLexer(input)
		parser := NewParser(lexer)

		expr, err := parser.Parse()
		assert.NoError(t, err)
		assert.NotNil(t, expr)

		binExpr, ok := expr.(*BinaryExpr)
		assert.True(t, ok)
		assert.Equal(t, TokenStrictWildcard, binExpr.Operator)
	})
}
