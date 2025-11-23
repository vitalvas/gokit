package wirefilter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
