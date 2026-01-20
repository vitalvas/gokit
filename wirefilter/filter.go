// Package wirefilter implements a filtering expression language and execution engine.
// It allows you to compile and evaluate filter expressions against runtime data.
//
// The filter language supports:
//   - Logical operators: and, or, not, xor, &&, ||, !, ^^
//   - Comparison operators: ==, !=, <, >, <=, >=
//   - Array operators: === (all equal), !== (any not equal)
//   - Membership operators: in, contains, matches (~)
//   - Wildcard matching: wildcard, strict wildcard
//   - Range expressions: {1..10}
//   - Multiple data types: string, int, bool, IP, bytes, arrays, maps
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
	"net/url"
	"regexp"
	"strings"
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
	case *UnpackExpr:
		return f.evaluateUnpackExpr(e, ctx)
	case *ListRefExpr:
		return f.evaluateListRefExpr(e, ctx)
	case *FunctionCallExpr:
		return f.evaluateFunctionCall(e, ctx)
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

	// Map access with string key
	if object.Type() == TypeMap && index.Type() == TypeString {
		mapVal := object.(MapValue)
		key := string(index.(StringValue))
		if val, ok := mapVal.Get(key); ok {
			return val, nil
		}
		return nil, nil
	}

	// Array access with integer index
	if object.Type() == TypeArray && index.Type() == TypeInt {
		arr := object.(ArrayValue)
		idx := int(index.(IntValue))
		if idx < 0 || idx >= len(arr) {
			return nil, nil // Out of bounds
		}
		return arr[idx], nil
	}

	return nil, nil
}

func (f *Filter) evaluateUnpackExpr(expr *UnpackExpr, ctx *ExecutionContext) (Value, error) {
	arr, err := f.evaluate(expr.Array, ctx)
	if err != nil {
		return nil, err
	}
	if arr == nil {
		return nil, nil
	}

	if arr.Type() != TypeArray {
		return nil, nil
	}

	return UnpackedArrayValue{Array: arr.(ArrayValue)}, nil
}

func (f *Filter) evaluateListRefExpr(expr *ListRefExpr, ctx *ExecutionContext) (Value, error) {
	list, ok := ctx.GetList(expr.Name)
	if !ok {
		return nil, nil
	}
	return list, nil
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

	// Handle UnpackedArrayValue - apply operation to each element (ANY semantics)
	if uv, ok := left.(UnpackedArrayValue); ok {
		return f.evaluateUnpackedBinaryExpr(uv, expr.Operator, right)
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

	case TokenXor:
		leftTruthy := left != nil && left.IsTruthy()
		rightTruthy := right != nil && right.IsTruthy()
		return BoolValue(leftTruthy != rightTruthy), nil

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

	case TokenWildcard:
		return f.evaluateWildcard(left, right, false)

	case TokenStrictWildcard:
		return f.evaluateWildcard(left, right, true)
	}

	return BoolValue(false), nil
}

func (f *Filter) evaluateUnpackedBinaryExpr(uv UnpackedArrayValue, op TokenType, right Value) (Value, error) {
	if len(uv.Array) == 0 {
		return BoolValue(false), nil
	}

	// Apply operation to each element, return true if ANY matches
	for _, elem := range uv.Array {
		var result Value
		var err error

		switch op {
		case TokenEq:
			result, err = f.evaluateEquality(elem, right)
		case TokenNe:
			eqResult, eqErr := f.evaluateEquality(elem, right)
			if eqErr != nil {
				return nil, eqErr
			}
			result = BoolValue(!bool(eqResult.(BoolValue)))
		case TokenLt:
			result, err = f.evaluateComparison(elem, right, func(a, b int64) bool { return a < b })
		case TokenGt:
			result, err = f.evaluateComparison(elem, right, func(a, b int64) bool { return a > b })
		case TokenLe:
			result, err = f.evaluateComparison(elem, right, func(a, b int64) bool { return a <= b })
		case TokenGe:
			result, err = f.evaluateComparison(elem, right, func(a, b int64) bool { return a >= b })
		case TokenContains:
			result, err = f.evaluateContains(elem, right)
		case TokenMatches:
			result, err = f.evaluateMatches(elem, right)
		case TokenWildcard:
			result, err = f.evaluateWildcard(elem, right, false)
		case TokenStrictWildcard:
			result, err = f.evaluateWildcard(elem, right, true)
		case TokenIn:
			result, err = f.evaluateIn(elem, right)
		default:
			continue
		}

		if err != nil {
			return nil, err
		}
		if result != nil && result.IsTruthy() {
			return BoolValue(true), nil
		}
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

func (f *Filter) evaluateWildcard(left, right Value, caseSensitive bool) (Value, error) {
	if left == nil || right == nil {
		return BoolValue(false), nil
	}
	if left.Type() != TypeString || right.Type() != TypeString {
		return BoolValue(false), nil
	}

	pattern := string(right.(StringValue))
	text := string(left.(StringValue))

	regexPattern := globToRegex(pattern)
	if !caseSensitive {
		regexPattern = "(?i)" + regexPattern
	}

	re, err := f.getCompiledRegex(regexPattern)
	if err != nil {
		return BoolValue(false), err
	}
	return BoolValue(re.MatchString(text)), nil
}

func globToRegex(glob string) string {
	var result []byte
	result = append(result, '^')

	for i := 0; i < len(glob); i++ {
		ch := glob[i]
		switch ch {
		case '*':
			result = append(result, '.', '*')
		case '?':
			result = append(result, '.')
		case '.', '+', '^', '$', '(', ')', '[', ']', '{', '}', '|', '\\':
			result = append(result, '\\', ch)
		default:
			result = append(result, ch)
		}
	}

	result = append(result, '$')
	return string(result)
}

func (f *Filter) evaluateFunctionCall(expr *FunctionCallExpr, ctx *ExecutionContext) (Value, error) {
	// Evaluate all arguments first
	args := make([]Value, len(expr.Arguments))
	for i, arg := range expr.Arguments {
		val, err := f.evaluate(arg, ctx)
		if err != nil {
			return nil, err
		}
		args[i] = val
	}

	name := strings.ToLower(expr.Name)

	switch name {
	case "lower":
		return f.fnLower(args)
	case "upper":
		return f.fnUpper(args)
	case "len":
		return f.fnLen(args)
	case "starts_with":
		return f.fnStartsWith(args)
	case "ends_with":
		return f.fnEndsWith(args)
	case "any":
		return f.fnAny(expr.Arguments, ctx)
	case "all":
		return f.fnAll(expr.Arguments, ctx)
	case "concat":
		return f.fnConcat(args)
	case "substring":
		return f.fnSubstring(args)
	case "split":
		return f.fnSplit(args)
	case "join":
		return f.fnJoin(args)
	case "has_key":
		return f.fnHasKey(args)
	case "has_value":
		return f.fnHasValue(args)
	case "url_decode":
		return f.fnURLDecode(args)
	case "cidr":
		return f.fnCIDR(args)
	case "cidr6":
		return f.fnCIDR6(args)
	}

	return nil, nil
}

// lower(String) -> String
func (f *Filter) fnLower(args []Value) (Value, error) {
	if len(args) != 1 || args[0] == nil {
		return nil, nil
	}
	if args[0].Type() != TypeString {
		return nil, nil
	}
	return StringValue(strings.ToLower(string(args[0].(StringValue)))), nil
}

// upper(String) -> String
func (f *Filter) fnUpper(args []Value) (Value, error) {
	if len(args) != 1 || args[0] == nil {
		return nil, nil
	}
	if args[0].Type() != TypeString {
		return nil, nil
	}
	return StringValue(strings.ToUpper(string(args[0].(StringValue)))), nil
}

// len(String|Array|Map|Bytes) -> Int
func (f *Filter) fnLen(args []Value) (Value, error) {
	if len(args) != 1 || args[0] == nil {
		return nil, nil
	}
	switch v := args[0].(type) {
	case StringValue:
		return IntValue(len(v)), nil
	case ArrayValue:
		return IntValue(len(v)), nil
	case MapValue:
		return IntValue(len(v)), nil
	case BytesValue:
		return IntValue(len(v)), nil
	}
	return nil, nil
}

// starts_with(String, String) -> Bool
func (f *Filter) fnStartsWith(args []Value) (Value, error) {
	if len(args) != 2 || args[0] == nil || args[1] == nil {
		return BoolValue(false), nil
	}
	if args[0].Type() != TypeString || args[1].Type() != TypeString {
		return BoolValue(false), nil
	}
	str := string(args[0].(StringValue))
	prefix := string(args[1].(StringValue))
	return BoolValue(strings.HasPrefix(str, prefix)), nil
}

// ends_with(String, String) -> Bool
func (f *Filter) fnEndsWith(args []Value) (Value, error) {
	if len(args) != 2 || args[0] == nil || args[1] == nil {
		return BoolValue(false), nil
	}
	if args[0].Type() != TypeString || args[1].Type() != TypeString {
		return BoolValue(false), nil
	}
	str := string(args[0].(StringValue))
	suffix := string(args[1].(StringValue))
	return BoolValue(strings.HasSuffix(str, suffix)), nil
}

// any(expression) -> Bool - returns true if any element matches
func (f *Filter) fnAny(args []Expression, ctx *ExecutionContext) (Value, error) {
	if len(args) != 1 {
		return BoolValue(false), nil
	}

	// The argument should be a binary expression with unpacked array on left
	result, err := f.evaluate(args[0], ctx)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return BoolValue(false), nil
	}

	return BoolValue(result.IsTruthy()), nil
}

// all(expression) -> Bool - returns true if all elements match
func (f *Filter) fnAll(args []Expression, ctx *ExecutionContext) (Value, error) {
	if len(args) != 1 {
		return BoolValue(false), nil
	}

	// Evaluate the inner expression
	arg := args[0]

	// If it's a binary expression with unpacked array, we need ALL semantics
	if binExpr, ok := arg.(*BinaryExpr); ok {
		left, err := f.evaluate(binExpr.Left, ctx)
		if err != nil {
			return nil, err
		}

		if uv, ok := left.(UnpackedArrayValue); ok {
			if len(uv.Array) == 0 {
				return BoolValue(false), nil
			}

			right, err := f.evaluate(binExpr.Right, ctx)
			if err != nil {
				return nil, err
			}

			// Apply operation to each element, return true only if ALL match
			for _, elem := range uv.Array {
				var result Value

				switch binExpr.Operator {
				case TokenEq:
					result, err = f.evaluateEquality(elem, right)
				case TokenNe:
					eqResult, eqErr := f.evaluateEquality(elem, right)
					if eqErr != nil {
						return nil, eqErr
					}
					result = BoolValue(!bool(eqResult.(BoolValue)))
				case TokenContains:
					result, err = f.evaluateContains(elem, right)
				case TokenMatches:
					result, err = f.evaluateMatches(elem, right)
				case TokenIn:
					result, err = f.evaluateIn(elem, right)
				case TokenLt:
					result, err = f.evaluateComparison(elem, right, func(a, b int64) bool { return a < b })
				case TokenGt:
					result, err = f.evaluateComparison(elem, right, func(a, b int64) bool { return a > b })
				case TokenLe:
					result, err = f.evaluateComparison(elem, right, func(a, b int64) bool { return a <= b })
				case TokenGe:
					result, err = f.evaluateComparison(elem, right, func(a, b int64) bool { return a >= b })
				default:
					continue
				}

				if err != nil {
					return nil, err
				}
				if result == nil || !result.IsTruthy() {
					return BoolValue(false), nil
				}
			}
			return BoolValue(true), nil
		}
	}

	// Fallback: just evaluate the expression
	result, err := f.evaluate(arg, ctx)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return BoolValue(false), nil
	}
	return BoolValue(result.IsTruthy()), nil
}

// concat(String...) -> String
func (f *Filter) fnConcat(args []Value) (Value, error) {
	var sb strings.Builder
	for _, arg := range args {
		if arg == nil {
			continue
		}
		if arg.Type() == TypeString {
			sb.WriteString(string(arg.(StringValue)))
		} else {
			sb.WriteString(arg.String())
		}
	}
	return StringValue(sb.String()), nil
}

// substring(String, Int, Int) -> String
func (f *Filter) fnSubstring(args []Value) (Value, error) {
	if len(args) < 2 || args[0] == nil || args[1] == nil {
		return nil, nil
	}
	if args[0].Type() != TypeString || args[1].Type() != TypeInt {
		return nil, nil
	}

	str := string(args[0].(StringValue))
	start := int(args[1].(IntValue))

	if start < 0 {
		start = 0
	}
	if start >= len(str) {
		return StringValue(""), nil
	}

	end := len(str)
	if len(args) >= 3 && args[2] != nil && args[2].Type() == TypeInt {
		end = int(args[2].(IntValue))
		if end > len(str) {
			end = len(str)
		}
		if end < start {
			end = start
		}
	}

	return StringValue(str[start:end]), nil
}

// split(String, String) -> Array
func (f *Filter) fnSplit(args []Value) (Value, error) {
	if len(args) != 2 || args[0] == nil || args[1] == nil {
		return nil, nil
	}
	if args[0].Type() != TypeString || args[1].Type() != TypeString {
		return nil, nil
	}

	str := string(args[0].(StringValue))
	sep := string(args[1].(StringValue))
	parts := strings.Split(str, sep)

	result := make(ArrayValue, len(parts))
	for i, part := range parts {
		result[i] = StringValue(part)
	}
	return result, nil
}

// join(Array, String) -> String
func (f *Filter) fnJoin(args []Value) (Value, error) {
	if len(args) != 2 || args[0] == nil || args[1] == nil {
		return nil, nil
	}
	if args[0].Type() != TypeArray || args[1].Type() != TypeString {
		return nil, nil
	}

	arr := args[0].(ArrayValue)
	sep := string(args[1].(StringValue))

	parts := make([]string, len(arr))
	for i, elem := range arr {
		switch {
		case elem == nil:
			parts[i] = ""
		case elem.Type() == TypeString:
			parts[i] = string(elem.(StringValue))
		default:
			parts[i] = elem.String()
		}
	}
	return StringValue(strings.Join(parts, sep)), nil
}

// has_key(Map, String) -> Bool
func (f *Filter) fnHasKey(args []Value) (Value, error) {
	if len(args) != 2 || args[0] == nil || args[1] == nil {
		return BoolValue(false), nil
	}
	if args[0].Type() != TypeMap || args[1].Type() != TypeString {
		return BoolValue(false), nil
	}

	m := args[0].(MapValue)
	key := string(args[1].(StringValue))
	_, ok := m.Get(key)
	return BoolValue(ok), nil
}

// has_value(Array, Value) -> Bool
func (f *Filter) fnHasValue(args []Value) (Value, error) {
	if len(args) != 2 || args[0] == nil || args[1] == nil {
		return BoolValue(false), nil
	}
	if args[0].Type() != TypeArray {
		return BoolValue(false), nil
	}

	arr := args[0].(ArrayValue)
	return BoolValue(arr.Contains(args[1])), nil
}

// url_decode(String) -> String
func (f *Filter) fnURLDecode(args []Value) (Value, error) {
	if len(args) != 1 || args[0] == nil {
		return nil, nil
	}
	if args[0].Type() != TypeString {
		return nil, nil
	}

	str := string(args[0].(StringValue))
	decoded, err := url.QueryUnescape(str)
	if err != nil {
		return StringValue(str), nil // Return original on error
	}
	return StringValue(decoded), nil
}

// cidr(IP, Int, Int) -> IP
// Applies CIDR masking: ipv4_bits for IPv4 (1-32), ipv6_bits for IPv6 (1-128)
func (f *Filter) fnCIDR(args []Value) (Value, error) {
	if len(args) != 3 || args[0] == nil || args[1] == nil || args[2] == nil {
		return nil, nil
	}
	if args[0].Type() != TypeIP || args[1].Type() != TypeInt || args[2].Type() != TypeInt {
		return nil, nil
	}

	ipVal := args[0].(IPValue)
	ipv4Bits := int(args[1].(IntValue))
	ipv6Bits := int(args[2].(IntValue))

	return applyCIDRMask(ipVal.IP, ipv4Bits, ipv6Bits), nil
}

// cidr6(IP, Int) -> IP
// Applies CIDR masking for IPv6: ipv6_bits (1-128)
// For IPv4 addresses, applies the same mask value (capped at 32)
func (f *Filter) fnCIDR6(args []Value) (Value, error) {
	if len(args) != 2 || args[0] == nil || args[1] == nil {
		return nil, nil
	}
	if args[0].Type() != TypeIP || args[1].Type() != TypeInt {
		return nil, nil
	}

	ipVal := args[0].(IPValue)
	ipv6Bits := int(args[1].(IntValue))

	// For cidr6, use ipv6_bits for both (IPv4 capped at 32)
	ipv4Bits := ipv6Bits
	if ipv4Bits > 32 {
		ipv4Bits = 32
	}

	return applyCIDRMask(ipVal.IP, ipv4Bits, ipv6Bits), nil
}

// applyCIDRMask applies CIDR mask to an IP address
func applyCIDRMask(ip net.IP, ipv4Bits, ipv6Bits int) Value {
	// Determine if IPv4 or IPv6
	ip4 := ip.To4()
	if ip4 != nil {
		// IPv4 address
		if ipv4Bits < 0 {
			ipv4Bits = 0
		}
		if ipv4Bits > 32 {
			ipv4Bits = 32
		}
		mask := net.CIDRMask(ipv4Bits, 32)
		masked := ip4.Mask(mask)
		return IPValue{IP: masked}
	}

	// IPv6 address
	if ipv6Bits < 0 {
		ipv6Bits = 0
	}
	if ipv6Bits > 128 {
		ipv6Bits = 128
	}
	mask := net.CIDRMask(ipv6Bits, 128)
	masked := ip.Mask(mask)
	return IPValue{IP: masked}
}
