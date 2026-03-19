package wirefilter

import (
	"fmt"
	"strings"
)

// exprString returns a short string representation of an expression for tracing.
func exprString(expr Expression) string {
	switch e := expr.(type) {
	case *BinaryExpr:
		return fmt.Sprintf("(%s %s %s)", exprString(e.Left), e.Operator, exprString(e.Right))
	case *UnaryExpr:
		return fmt.Sprintf("(%s %s)", e.Operator, exprString(e.Operand))
	case *FieldExpr:
		return e.Name
	case *LiteralExpr:
		if e.Value == nil {
			return "nil"
		}
		return e.Value.String()
	case *FunctionCallExpr:
		args := make([]string, len(e.Arguments))
		for i, arg := range e.Arguments {
			args[i] = exprString(arg)
		}
		return fmt.Sprintf("%s(%s)", e.Name, strings.Join(args, ", "))
	case *ArrayExpr:
		return fmt.Sprintf("{...%d}", len(e.Elements))
	case *IndexExpr:
		return fmt.Sprintf("%s[%s]", exprString(e.Object), exprString(e.Index))
	case *UnpackExpr:
		return fmt.Sprintf("%s[*]", exprString(e.Array))
	case *ListRefExpr:
		return fmt.Sprintf("$%s", e.Name)
	case *RangeExpr:
		return fmt.Sprintf("%s..%s", exprString(e.Start), exprString(e.End))
	}
	return "?"
}
