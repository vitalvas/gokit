package wirefilter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExprString(t *testing.T) {
	t.Run("binary expr", func(t *testing.T) {
		expr := &BinaryExpr{
			Left:     &FieldExpr{Name: "x"},
			Operator: TokenEq,
			Right:    &LiteralExpr{Value: IntValue(1)},
		}
		assert.Equal(t, "(x == 1)", exprString(expr))
	})

	t.Run("unary expr", func(t *testing.T) {
		expr := &UnaryExpr{
			Operator: TokenNot,
			Operand:  &FieldExpr{Name: "active"},
		}
		assert.Equal(t, "(not active)", exprString(expr))
	})

	t.Run("field expr", func(t *testing.T) {
		assert.Equal(t, "http.host", exprString(&FieldExpr{Name: "http.host"}))
	})

	t.Run("literal nil", func(t *testing.T) {
		assert.Equal(t, "nil", exprString(&LiteralExpr{Value: nil}))
	})

	t.Run("literal string", func(t *testing.T) {
		assert.Equal(t, "test", exprString(&LiteralExpr{Value: StringValue("test")}))
	})

	t.Run("function call", func(t *testing.T) {
		expr := &FunctionCallExpr{
			Name:      "lower",
			Arguments: []Expression{&FieldExpr{Name: "name"}},
		}
		assert.Equal(t, "lower(name)", exprString(expr))
	})

	t.Run("function call no args", func(t *testing.T) {
		expr := &FunctionCallExpr{Name: "now", Arguments: []Expression{}}
		assert.Equal(t, "now()", exprString(expr))
	})

	t.Run("array expr", func(t *testing.T) {
		expr := &ArrayExpr{Elements: []Expression{
			&LiteralExpr{Value: IntValue(1)},
			&LiteralExpr{Value: IntValue(2)},
		}}
		assert.Equal(t, "{...2}", exprString(expr))
	})

	t.Run("index expr", func(t *testing.T) {
		expr := &IndexExpr{
			Object: &FieldExpr{Name: "data"},
			Index:  &LiteralExpr{Value: StringValue("key")},
		}
		assert.Equal(t, "data[key]", exprString(expr))
	})

	t.Run("unpack expr", func(t *testing.T) {
		expr := &UnpackExpr{Array: &FieldExpr{Name: "tags"}}
		assert.Equal(t, "tags[*]", exprString(expr))
	})

	t.Run("list ref expr", func(t *testing.T) {
		assert.Equal(t, "$blocked", exprString(&ListRefExpr{Name: "blocked"}))
	})

	t.Run("range expr", func(t *testing.T) {
		expr := &RangeExpr{
			Start: &LiteralExpr{Value: IntValue(1)},
			End:   &LiteralExpr{Value: IntValue(10)},
		}
		assert.Equal(t, "1..10", exprString(expr))
	})

	t.Run("unknown expr", func(t *testing.T) {
		assert.Equal(t, "?", exprString(nil))
	})
}

func TestExprStringIntegration(t *testing.T) {
	t.Run("exprString through tracing", func(t *testing.T) {
		filter, _ := Compile(`x in {1..5} and tags[*] == "a" and $list contains "b"`, nil)
		ctx := NewExecutionContext().
			EnableTrace().
			SetIntField("x", 3).
			SetArrayField("tags", []string{"a"}).
			SetList("list", []string{"b"})
		_, _ = filter.Execute(ctx)
		trace := ctx.Trace()
		assert.NotNil(t, trace)
		assert.NotEmpty(t, trace.Children)
	})
}
