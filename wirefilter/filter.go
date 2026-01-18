// Package wirefilter implements a filtering expression language and execution engine.
// It allows you to compile and evaluate filter expressions against runtime data.
//
// The filter language supports:
//   - Logical operators: and, or, not, &&, ||
//   - Comparison operators: ==, !=, <, >, <=, >=
//   - Array operators: === (all equal), !== (any not equal)
//   - Membership operators: in, contains, matches
//   - Range expressions: {1..10}
//   - Multiple data types: string, int, bool, IP, bytes, arrays
//
// Example:
//
//	schema := wirefilter.NewSchema().
//	    AddField("http.host", wirefilter.TypeString).
//	    AddField("http.status", wirefilter.TypeInt)
//
//	filter, err := wirefilter.Compile(`http.host == "example.com" and http.status >= 400`, schema)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	ctx := wirefilter.NewExecutionContext().
//	    SetStringField("http.host", "example.com").
//	    SetIntField("http.status", 500)
//
//	result, err := filter.Execute(ctx)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(result) // true
package wirefilter

import (
	"net"
	"regexp"
)

// Filter represents a compiled filter expression that can be executed against an execution context.
type Filter struct {
	expr       Expression
	schema     *Schema
	regexCache map[string]*regexp.Regexp
	cidrCache  map[string]*net.IPNet
}

// Compile parses and compiles a filter expression string into an executable Filter.
// If a schema is provided, it validates that all fields referenced in the expression exist in the schema.
// Returns an error if the expression is malformed or references unknown fields.
func Compile(filterStr string, schema *Schema) (*Filter, error) {
	lexer := NewLexer(filterStr)
	parser := NewParser(lexer)

	expr, err := parser.Parse()
	if err != nil {
		return nil, err
	}

	if schema != nil {
		if err := schema.Validate(expr); err != nil {
			return nil, err
		}
	}

	return &Filter{
		expr:       expr,
		schema:     schema,
		regexCache: make(map[string]*regexp.Regexp),
		cidrCache:  make(map[string]*net.IPNet),
	}, nil
}

// Execute evaluates the compiled filter against the provided execution context.
// Returns true if the filter matches, false otherwise.
// Returns an error if evaluation fails.
func (f *Filter) Execute(ctx *ExecutionContext) (bool, error) {
	result, err := f.evaluate(f.expr, ctx)
	if err != nil {
		return false, err
	}

	if result == nil {
		return false, nil
	}

	return result.IsTruthy(), nil
}

func (f *Filter) evaluate(expr Expression, ctx *ExecutionContext) (Value, error) {
	switch e := expr.(type) {
	case *BinaryExpr:
		return f.evaluateBinaryExpr(e, ctx)
	case *UnaryExpr:
		return f.evaluateUnaryExpr(e, ctx)
	case *FieldExpr:
		return f.evaluateFieldExpr(e, ctx)
	case *LiteralExpr:
		return e.Value, nil
	case *ArrayExpr:
		return f.evaluateArrayExpr(e, ctx)
	case *RangeExpr:
		return f.evaluateRangeExpr(e, ctx)
	case *IndexExpr:
		return f.evaluateIndexExpr(e, ctx)
	}
	return nil, nil
}

func (f *Filter) evaluateArrayExpr(expr *ArrayExpr, ctx *ExecutionContext) (Value, error) {
	values := make([]Value, 0, len(expr.Elements))
	for _, elem := range expr.Elements {
		if rangeExpr, ok := elem.(*RangeExpr); ok {
			rangeVals, err := f.evaluateRangeExpr(rangeExpr, ctx)
			if err != nil {
				return nil, err
			}
			if arr, ok := rangeVals.(ArrayValue); ok {
				values = append(values, arr...)
			}
		} else {
			val, err := f.evaluate(elem, ctx)
			if err != nil {
				return nil, err
			}
			values = append(values, val)
		}
	}
	return ArrayValue(values), nil
}

func (f *Filter) evaluateRangeExpr(expr *RangeExpr, ctx *ExecutionContext) (Value, error) {
	start, err := f.evaluate(expr.Start, ctx)
	if err != nil {
		return nil, err
	}

	end, err := f.evaluate(expr.End, ctx)
	if err != nil {
		return nil, err
	}

	if start.Type() != TypeInt || end.Type() != TypeInt {
		return ArrayValue([]Value{}), nil
	}

	startInt := int64(start.(IntValue))
	endInt := int64(end.(IntValue))

	if startInt > endInt {
		return ArrayValue([]Value{}), nil
	}

	values := make([]Value, 0, endInt-startInt+1)
	for i := startInt; i <= endInt; i++ {
		values = append(values, IntValue(i))
	}

	return ArrayValue(values), nil
}

func (f *Filter) evaluateFieldExpr(expr *FieldExpr, ctx *ExecutionContext) (Value, error) {
	val, ok := ctx.GetField(expr.Name)
	if !ok {
		return nil, nil
	}
	return val, nil
}

func (f *Filter) evaluateIndexExpr(expr *IndexExpr, ctx *ExecutionContext) (Value, error) {
	object, err := f.evaluate(expr.Object, ctx)
	if err != nil {
		return nil, err
	}
	if object == nil {
		return nil, nil
	}

	index, err := f.evaluate(expr.Index, ctx)
	if err != nil {
		return nil, err
	}
	if index == nil {
		return nil, nil
	}

	if object.Type() == TypeMap && index.Type() == TypeString {
		mapVal := object.(MapValue)
		key := string(index.(StringValue))
		if val, ok := mapVal.Get(key); ok {
			return val, nil
		}
		return nil, nil
	}

	return nil, nil
}

func (f *Filter) evaluateUnaryExpr(expr *UnaryExpr, ctx *ExecutionContext) (Value, error) {
	operand, err := f.evaluate(expr.Operand, ctx)
	if err != nil {
		return nil, err
	}

	if expr.Operator == TokenNot {
		if operand == nil {
			return BoolValue(true), nil
		}
		return BoolValue(!operand.IsTruthy()), nil
	}

	return nil, nil
}

func (f *Filter) evaluateBinaryExpr(expr *BinaryExpr, ctx *ExecutionContext) (Value, error) {
	left, err := f.evaluate(expr.Left, ctx)
	if err != nil {
		return nil, err
	}

	right, err := f.evaluate(expr.Right, ctx)
	if err != nil {
		return nil, err
	}

	switch expr.Operator {
	case TokenAnd:
		leftTruthy := left != nil && left.IsTruthy()
		rightTruthy := right != nil && right.IsTruthy()
		return BoolValue(leftTruthy && rightTruthy), nil

	case TokenOr:
		leftTruthy := left != nil && left.IsTruthy()
		rightTruthy := right != nil && right.IsTruthy()
		return BoolValue(leftTruthy || rightTruthy), nil

	case TokenEq:
		return f.evaluateEquality(left, right)

	case TokenNe:
		result, err := f.evaluateEquality(left, right)
		if err != nil {
			return nil, err
		}
		return BoolValue(!bool(result.(BoolValue))), nil

	case TokenAllEq:
		return f.evaluateAllEqual(left, right)

	case TokenAnyNe:
		return f.evaluateAnyNotEqual(left, right)

	case TokenLt:
		return f.evaluateComparison(left, right, func(a, b int64) bool { return a < b })

	case TokenGt:
		return f.evaluateComparison(left, right, func(a, b int64) bool { return a > b })

	case TokenLe:
		return f.evaluateComparison(left, right, func(a, b int64) bool { return a <= b })

	case TokenGe:
		return f.evaluateComparison(left, right, func(a, b int64) bool { return a >= b })

	case TokenContains:
		return f.evaluateContains(left, right)

	case TokenMatches:
		return f.evaluateMatches(left, right)

	case TokenIn:
		return f.evaluateIn(left, right)
	}

	return BoolValue(false), nil
}

func (f *Filter) evaluateEquality(left, right Value) (Value, error) {
	if left == nil || right == nil {
		return BoolValue(left == nil && right == nil), nil
	}
	if left.Type() == TypeIP && right.Type() == TypeString {
		ip := net.ParseIP(string(right.(StringValue)))
		if ip == nil {
			return BoolValue(false), nil
		}
		right = IPValue{IP: ip}
	} else if left.Type() == TypeString && right.Type() == TypeIP {
		ip := net.ParseIP(string(left.(StringValue)))
		if ip == nil {
			return BoolValue(false), nil
		}
		left = IPValue{IP: ip}
	}
	return BoolValue(left.Equal(right)), nil
}

func (f *Filter) evaluateComparison(left, right Value, cmp func(int64, int64) bool) (Value, error) {
	if left == nil || right == nil {
		return BoolValue(false), nil
	}
	if left.Type() != TypeInt || right.Type() != TypeInt {
		return BoolValue(false), nil
	}
	return BoolValue(cmp(int64(left.(IntValue)), int64(right.(IntValue)))), nil
}

func (f *Filter) evaluateContains(left, right Value) (Value, error) {
	if left == nil || right == nil {
		return BoolValue(false), nil
	}
	if left.Type() == TypeString && right.Type() == TypeString {
		return BoolValue(ContainsString(string(left.(StringValue)), string(right.(StringValue)))), nil
	}
	if left.Type() == TypeArray {
		leftArr := left.(ArrayValue)
		// Array contains Array: AND logic - all elements from right exist in left
		if right.Type() == TypeArray {
			rightArr := right.(ArrayValue)
			if len(rightArr) == 0 {
				return BoolValue(true), nil
			}
			for _, rightElem := range rightArr {
				if !leftArr.Contains(rightElem) {
					return BoolValue(false), nil
				}
			}
			return BoolValue(true), nil
		}
		// Array contains single value
		return BoolValue(leftArr.Contains(right)), nil
	}
	return BoolValue(false), nil
}

func (f *Filter) evaluateMatches(left, right Value) (Value, error) {
	if left == nil || right == nil {
		return BoolValue(false), nil
	}
	if left.Type() != TypeString || right.Type() != TypeString {
		return BoolValue(false), nil
	}
	pattern := string(right.(StringValue))
	re, err := f.getCompiledRegex(pattern)
	if err != nil {
		return BoolValue(false), err
	}
	return BoolValue(re.MatchString(string(left.(StringValue)))), nil
}

func (f *Filter) getCompiledRegex(pattern string) (*regexp.Regexp, error) {
	if re, ok := f.regexCache[pattern]; ok {
		return re, nil
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	f.regexCache[pattern] = re
	return re, nil
}

func (f *Filter) evaluateIn(left, right Value) (Value, error) {
	if left == nil || right == nil {
		return BoolValue(false), nil
	}
	if right.Type() == TypeArray {
		rightArr := right.(ArrayValue)
		// Array in Array: OR logic - any element from left exists in right
		if left.Type() == TypeArray {
			leftArr := left.(ArrayValue)
			for _, leftElem := range leftArr {
				if rightArr.Contains(leftElem) {
					return BoolValue(true), nil
				}
			}
			return BoolValue(false), nil
		}
		// Single value in Array
		return BoolValue(rightArr.Contains(left)), nil
	}

	if left.Type() == TypeIP && right.Type() == TypeString {
		ipVal := left.(IPValue)
		cidr := string(right.(StringValue))
		ipNet, err := f.getParsedCIDR(cidr)
		if err != nil {
			return BoolValue(false), err
		}
		return BoolValue(ipNet.Contains(ipVal.IP)), nil
	}

	return BoolValue(false), nil
}

func (f *Filter) getParsedCIDR(cidr string) (*net.IPNet, error) {
	if ipNet, ok := f.cidrCache[cidr]; ok {
		return ipNet, nil
	}
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}
	f.cidrCache[cidr] = ipNet
	return ipNet, nil
}

func (f *Filter) evaluateAllEqual(left, right Value) (Value, error) {
	if left == nil || right == nil {
		return BoolValue(false), nil
	}
	if left.Type() != TypeArray {
		return BoolValue(false), nil
	}

	arr := left.(ArrayValue)
	if len(arr) == 0 {
		return BoolValue(false), nil
	}

	for _, elem := range arr {
		result, err := f.evaluateEquality(elem, right)
		if err != nil {
			return nil, err
		}
		if !result.IsTruthy() {
			return BoolValue(false), nil
		}
	}

	return BoolValue(true), nil
}

func (f *Filter) evaluateAnyNotEqual(left, right Value) (Value, error) {
	if left == nil || right == nil {
		return BoolValue(false), nil
	}
	if left.Type() != TypeArray {
		return BoolValue(false), nil
	}

	arr := left.(ArrayValue)
	if len(arr) == 0 {
		return BoolValue(false), nil
	}

	for _, elem := range arr {
		result, err := f.evaluateEquality(elem, right)
		if err != nil {
			return nil, err
		}
		if !result.IsTruthy() {
			return BoolValue(true), nil
		}
	}

	return BoolValue(false), nil
}
