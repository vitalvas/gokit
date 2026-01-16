package wirefilter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestASTNodeInterfaces(t *testing.T) {
	t.Run("BinaryExpr implements Node and Expression", func(t *testing.T) {
		expr := &BinaryExpr{}
		expr.node()
		expr.expression()
		var _ Node = expr
		var _ Expression = expr
		assert.NotNil(t, expr)
	})

	t.Run("UnaryExpr implements Node and Expression", func(t *testing.T) {
		expr := &UnaryExpr{}
		expr.node()
		expr.expression()
		var _ Node = expr
		var _ Expression = expr
		assert.NotNil(t, expr)
	})

	t.Run("FieldExpr implements Node and Expression", func(t *testing.T) {
		expr := &FieldExpr{}
		expr.node()
		expr.expression()
		var _ Node = expr
		var _ Expression = expr
		assert.NotNil(t, expr)
	})

	t.Run("LiteralExpr implements Node and Expression", func(t *testing.T) {
		expr := &LiteralExpr{}
		expr.node()
		expr.expression()
		var _ Node = expr
		var _ Expression = expr
		assert.NotNil(t, expr)
	})

	t.Run("ArrayExpr implements Node and Expression", func(t *testing.T) {
		expr := &ArrayExpr{}
		expr.node()
		expr.expression()
		var _ Node = expr
		var _ Expression = expr
		assert.NotNil(t, expr)
	})

	t.Run("RangeExpr implements Node and Expression", func(t *testing.T) {
		expr := &RangeExpr{}
		expr.node()
		expr.expression()
		var _ Node = expr
		var _ Expression = expr
		assert.NotNil(t, expr)
	})
}
