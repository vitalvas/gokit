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
	"encoding/hex"
	"fmt"
	"hash/fnv"
	"math"
	"net"
	"net/url"
	"regexp"
	"strings"
	"sync"
)

// Filter represents a compiled filter expression that can be executed against an execution context.
// Filter is safe for concurrent use across goroutines.
type Filter struct {
	expr       Expression
	schema     *Schema
	regexCache map[string]*regexp.Regexp
	regexMu    sync.RWMutex
	cidrCache  map[string]*net.IPNet
	cidrMu     sync.RWMutex
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

// Hash returns a hex-encoded hash of the compiled filter's canonical AST representation.
// Two expressions that are semantically identical produce the same hash, even if they
// differ in whitespace, operator aliases (and vs &&), or formatting.
// This can be used to deduplicate filter expressions.
func (f *Filter) Hash() string {
	data, err := f.MarshalBinary()
	if err != nil {
		return ""
	}

	h := fnv.New128a()
	h.Write(data)

	return hex.EncodeToString(h.Sum(nil))
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

	if start == nil || end == nil || start.Type() != TypeInt || end.Type() != TypeInt {
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

	// Map access with string key (or any type converted to string)
	if object.Type() == TypeMap {
		mapVal := object.(MapValue)
		var key string
		if index.Type() == TypeString {
			key = string(index.(StringValue))
		} else {
			key = index.String()
		}
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
	if list, ok := ctx.GetList(expr.Name); ok {
		return list, nil
	}
	if table, ok := ctx.GetTable(expr.Name); ok {
		return table, nil
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

// evaluateLogicalOp handles short-circuit evaluation for logical operators (and, or, xor).
// Returns (result, handled, error) where handled=true if this was a logical operator.
func (f *Filter) evaluateLogicalOp(expr *BinaryExpr, left Value, ctx *ExecutionContext) (Value, bool, error) {
	switch expr.Operator {
	case TokenAnd:
		leftTruthy := left != nil && left.IsTruthy()
		if !leftTruthy {
			return BoolValue(false), true, nil // Short-circuit: false and X = false
		}
		right, err := f.evaluate(expr.Right, ctx)
		if err != nil {
			return nil, true, err
		}
		rightTruthy := right != nil && right.IsTruthy()
		return BoolValue(rightTruthy), true, nil

	case TokenOr:
		leftTruthy := left != nil && left.IsTruthy()
		if leftTruthy {
			return BoolValue(true), true, nil // Short-circuit: true or X = true
		}
		right, err := f.evaluate(expr.Right, ctx)
		if err != nil {
			return nil, true, err
		}
		rightTruthy := right != nil && right.IsTruthy()
		return BoolValue(rightTruthy), true, nil

	case TokenXor:
		// XOR cannot short-circuit - both sides needed
		right, err := f.evaluate(expr.Right, ctx)
		if err != nil {
			return nil, true, err
		}
		leftTruthy := left != nil && left.IsTruthy()
		rightTruthy := right != nil && right.IsTruthy()
		return BoolValue(leftTruthy != rightTruthy), true, nil
	}
	return nil, false, nil
}

func (f *Filter) evaluateBinaryExpr(expr *BinaryExpr, ctx *ExecutionContext) (Value, error) {
	left, err := f.evaluate(expr.Left, ctx)
	if err != nil {
		return nil, err
	}

	// Handle logical operators with short-circuit evaluation
	if result, handled, err := f.evaluateLogicalOp(expr, left, ctx); handled {
		return result, err
	}

	// For non-logical operators, evaluate right side
	right, err := f.evaluate(expr.Right, ctx)
	if err != nil {
		return nil, err
	}

	// Handle UnpackedArrayValue - apply operation to each element (ANY semantics)
	if uv, ok := left.(UnpackedArrayValue); ok {
		return f.evaluateUnpackedBinaryExpr(uv, expr.Operator, right)
	}

	switch expr.Operator {
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

	case TokenPlus, TokenMinus, TokenAsterisk, TokenDiv, TokenMod:
		return f.evaluateArithmetic(left, right, expr.Operator)
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
	switch {
	case left.Type() == TypeIP && right.Type() == TypeString:
		ip := net.ParseIP(string(right.(StringValue)))
		if ip == nil {
			return BoolValue(false), nil
		}
		right = IPValue{IP: ip}
	case left.Type() == TypeString && right.Type() == TypeIP:
		ip := net.ParseIP(string(left.(StringValue)))
		if ip == nil {
			return BoolValue(false), nil
		}
		left = IPValue{IP: ip}
	case left.Type() == TypeCIDR && right.Type() == TypeString:
		_, ipNet, err := net.ParseCIDR(string(right.(StringValue)))
		if err != nil {
			return BoolValue(false), nil
		}
		right = CIDRValue{IPNet: ipNet}
	case left.Type() == TypeString && right.Type() == TypeCIDR:
		_, ipNet, err := net.ParseCIDR(string(left.(StringValue)))
		if err != nil {
			return BoolValue(false), nil
		}
		left = CIDRValue{IPNet: ipNet}
	}
	return BoolValue(left.Equal(right)), nil
}

func (f *Filter) evaluateComparison(left, right Value, cmp func(int64, int64) bool) (Value, error) {
	if left == nil || right == nil {
		return BoolValue(false), nil
	}

	// Handle Float and mixed Int/Float comparisons
	if left.Type() == TypeFloat || right.Type() == TypeFloat {
		leftF, leftOk := toFloat64(left)
		rightF, rightOk := toFloat64(right)
		if !leftOk || !rightOk {
			return BoolValue(false), nil
		}
		// Map the int64 comparator to float64 by comparing equivalent sign values
		return BoolValue(cmp(floatSign(leftF-rightF), 0)), nil
	}

	if left.Type() != TypeInt || right.Type() != TypeInt {
		return BoolValue(false), nil
	}
	return BoolValue(cmp(int64(left.(IntValue)), int64(right.(IntValue)))), nil
}

// toFloat64 converts Int or Float values to float64 for mixed comparisons.
func toFloat64(v Value) (float64, bool) {
	switch val := v.(type) {
	case FloatValue:
		return float64(val), true
	case IntValue:
		return float64(val), true
	}
	return 0, false
}

func (f *Filter) evaluateArithmetic(left, right Value, op TokenType) (Value, error) {
	if left == nil || right == nil {
		return nil, nil
	}

	// If either operand is a float, do float arithmetic
	if left.Type() == TypeFloat || right.Type() == TypeFloat {
		lf, lok := toFloat64(left)
		rf, rok := toFloat64(right)
		if !lok || !rok {
			return nil, nil
		}
		switch op {
		case TokenPlus:
			return FloatValue(lf + rf), nil
		case TokenMinus:
			return FloatValue(lf - rf), nil
		case TokenAsterisk:
			return FloatValue(lf * rf), nil
		case TokenDiv:
			if rf == 0 {
				return nil, nil
			}
			return FloatValue(lf / rf), nil
		case TokenMod:
			if rf == 0 {
				return nil, nil
			}
			return FloatValue(math.Mod(lf, rf)), nil
		}
		return nil, nil
	}

	// Integer arithmetic
	if left.Type() != TypeInt || right.Type() != TypeInt {
		return nil, nil
	}
	li := int64(left.(IntValue))
	ri := int64(right.(IntValue))

	switch op {
	case TokenPlus:
		return IntValue(li + ri), nil
	case TokenMinus:
		return IntValue(li - ri), nil
	case TokenAsterisk:
		return IntValue(li * ri), nil
	case TokenDiv:
		if ri == 0 {
			return nil, nil
		}
		return IntValue(li / ri), nil
	case TokenMod:
		if ri == 0 {
			return nil, nil
		}
		return IntValue(li % ri), nil
	}
	return nil, nil
}

// floatSign returns -1, 0, or 1 as an int64 based on the sign of f.
func floatSign(f float64) int64 {
	if f < 0 {
		return -1
	}
	if f > 0 {
		return 1
	}
	return 0
}

func (f *Filter) evaluateContains(left, right Value) (Value, error) {
	if left == nil || right == nil {
		return BoolValue(false), nil
	}
	if left.Type() == TypeString && right.Type() == TypeString {
		return BoolValue(strings.Contains(string(left.(StringValue)), string(right.(StringValue)))), nil
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
	f.regexMu.RLock()
	if re, ok := f.regexCache[pattern]; ok {
		f.regexMu.RUnlock()
		return re, nil
	}
	f.regexMu.RUnlock()

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}

	f.regexMu.Lock()
	f.regexCache[pattern] = re
	f.regexMu.Unlock()
	return re, nil
}

func (f *Filter) evaluateIn(left, right Value) (Value, error) {
	if left == nil || right == nil {
		return BoolValue(false), nil
	}

	// Handle IP in CIDR directly: ip.src in 192.168.0.0/24
	if left.Type() == TypeIP && right.Type() == TypeCIDR {
		ipVal := left.(IPValue)
		cidrVal := right.(CIDRValue)
		return BoolValue(cidrVal.Contains(ipVal.IP)), nil
	}

	if right.Type() == TypeArray {
		rightArr := right.(ArrayValue)

		// IP in Array: check if IP matches any element (IP equality or CIDR containment)
		if left.Type() == TypeIP {
			ipVal := left.(IPValue)
			for _, elem := range rightArr {
				if elem == nil {
					continue
				}
				switch elem.Type() {
				case TypeIP:
					if ipVal.IP.Equal(elem.(IPValue).IP) {
						return BoolValue(true), nil
					}
				case TypeCIDR:
					if elem.(CIDRValue).Contains(ipVal.IP) {
						return BoolValue(true), nil
					}
				}
			}
			return BoolValue(false), nil
		}

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

	// Legacy: IP in CIDR as string (keep for backwards compatibility)
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
	f.cidrMu.RLock()
	if ipNet, ok := f.cidrCache[cidr]; ok {
		f.cidrMu.RUnlock()
		return ipNet, nil
	}
	f.cidrMu.RUnlock()

	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}

	f.cidrMu.Lock()
	f.cidrCache[cidr] = ipNet
	f.cidrMu.Unlock()
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
		regexPattern = fmt.Sprintf("(?i)%s", regexPattern)
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
	name := strings.ToLower(expr.Name)

	// Special-case functions that need raw expressions (not evaluated args)
	switch name {
	case "any":
		return f.fnAny(expr.Arguments, ctx)
	case "all":
		return f.fnAll(expr.Arguments, ctx)
	}

	// Evaluate all arguments for standard functions
	args := make([]Value, len(expr.Arguments))
	for i, arg := range expr.Arguments {
		val, err := f.evaluate(arg, ctx)
		if err != nil {
			return nil, err
		}
		args[i] = val
	}

	if fn, ok := f.builtinFuncs()[name]; ok {
		return fn(args)
	}

	// Check user-defined functions in the execution context
	if fn, ok := ctx.GetFunc(name); ok {
		return fn(args)
	}

	return nil, nil
}

// builtinFuncs returns the map of built-in function implementations.
func (f *Filter) builtinFuncs() map[string]func([]Value) (Value, error) {
	return map[string]func([]Value) (Value, error){
		"lower":         f.fnLower,
		"upper":         f.fnUpper,
		"len":           f.fnLen,
		"starts_with":   f.fnStartsWith,
		"ends_with":     f.fnEndsWith,
		"concat":        f.fnConcat,
		"substring":     f.fnSubstring,
		"split":         f.fnSplit,
		"join":          f.fnJoin,
		"has_key":       f.fnHasKey,
		"has_value":     f.fnHasValue,
		"url_decode":    f.fnURLDecode,
		"cidr":          f.fnCIDR,
		"cidr6":         f.fnCIDR6,
		"regex_replace": f.fnRegexReplace,
		"trim":          f.fnTrim,
		"trim_left":     f.fnTrimLeft,
		"trim_right":    f.fnTrimRight,
		"replace":       f.fnReplace,
		"count":         f.fnCount,
		"coalesce":      f.fnCoalesce,
		"contains_word": f.fnContainsWord,
		"abs":           f.fnAbs,
		"ceil":          f.fnCeil,
		"floor":         f.fnFloor,
		"round":         f.fnRound,
		"is_ipv4":       f.fnIsIPv4,
		"is_ipv6":       f.fnIsIPv6,
		"is_loopback":   f.fnIsLoopback,
		"regex_extract": f.fnRegexExtract,
		"intersection":  f.fnIntersection,
		"union":         f.fnUnion,
		"difference":    f.fnDifference,
		"contains_any":  f.fnContainsAny,
		"contains_all":  f.fnContainsAll,
	}
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

// cidr(IP, Int) -> IP
// Applies CIDR masking for IPv4: ipv4_bits (1-32)
// For IPv6 addresses, applies the same mask value (capped at 128)
func (f *Filter) fnCIDR(args []Value) (Value, error) {
	if len(args) != 2 || args[0] == nil || args[1] == nil {
		return nil, nil
	}
	if args[0].Type() != TypeIP || args[1].Type() != TypeInt {
		return nil, nil
	}

	ipVal := args[0].(IPValue)
	ipv4Bits := int(args[1].(IntValue))

	ip4 := ipVal.IP.To4()
	if ip4 != nil {
		if ipv4Bits < 0 {
			ipv4Bits = 0
		}
		if ipv4Bits > 32 {
			ipv4Bits = 32
		}
		mask := net.CIDRMask(ipv4Bits, 32)
		return CIDRValue{IPNet: &net.IPNet{IP: ip4.Mask(mask), Mask: mask}}, nil
	}

	return nil, nil
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

// applyCIDRMask applies CIDR mask to an IP address and returns a CIDRValue.
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
		return CIDRValue{IPNet: &net.IPNet{IP: ip4.Mask(mask), Mask: mask}}
	}

	// IPv6 address
	if ipv6Bits < 0 {
		ipv6Bits = 0
	}
	if ipv6Bits > 128 {
		ipv6Bits = 128
	}
	mask := net.CIDRMask(ipv6Bits, 128)
	return CIDRValue{IPNet: &net.IPNet{IP: ip.Mask(mask), Mask: mask}}
}

// regex_replace(String, String, String) -> String
func (f *Filter) fnRegexReplace(args []Value) (Value, error) {
	if len(args) != 3 || args[0] == nil || args[1] == nil || args[2] == nil {
		return nil, nil
	}
	if args[0].Type() != TypeString || args[1].Type() != TypeString || args[2].Type() != TypeString {
		return nil, nil
	}

	str := string(args[0].(StringValue))
	pattern := string(args[1].(StringValue))
	replacement := string(args[2].(StringValue))

	re, err := f.getCompiledRegex(pattern)
	if err != nil {
		return nil, err
	}

	return StringValue(re.ReplaceAllString(str, replacement)), nil
}

// trim(String) -> String
func (f *Filter) fnTrim(args []Value) (Value, error) {
	if len(args) != 1 || args[0] == nil {
		return nil, nil
	}
	if args[0].Type() != TypeString {
		return nil, nil
	}
	return StringValue(strings.TrimSpace(string(args[0].(StringValue)))), nil
}

// trim_left(String) -> String
func (f *Filter) fnTrimLeft(args []Value) (Value, error) {
	if len(args) != 1 || args[0] == nil {
		return nil, nil
	}
	if args[0].Type() != TypeString {
		return nil, nil
	}
	return StringValue(strings.TrimLeft(string(args[0].(StringValue)), " \t\n\r")), nil
}

// trim_right(String) -> String
func (f *Filter) fnTrimRight(args []Value) (Value, error) {
	if len(args) != 1 || args[0] == nil {
		return nil, nil
	}
	if args[0].Type() != TypeString {
		return nil, nil
	}
	return StringValue(strings.TrimRight(string(args[0].(StringValue)), " \t\n\r")), nil
}

// replace(String, String, String) -> String
func (f *Filter) fnReplace(args []Value) (Value, error) {
	if len(args) != 3 || args[0] == nil || args[1] == nil || args[2] == nil {
		return nil, nil
	}
	if args[0].Type() != TypeString || args[1].Type() != TypeString || args[2].Type() != TypeString {
		return nil, nil
	}

	str := string(args[0].(StringValue))
	old := string(args[1].(StringValue))
	newStr := string(args[2].(StringValue))

	return StringValue(strings.ReplaceAll(str, old, newStr)), nil
}

// count(Array) -> Int
func (f *Filter) fnCount(args []Value) (Value, error) {
	if len(args) != 1 || args[0] == nil {
		return nil, nil
	}
	if args[0].Type() != TypeArray {
		return nil, nil
	}

	arr := args[0].(ArrayValue)
	count := 0
	for _, elem := range arr {
		if elem != nil && elem.IsTruthy() {
			count++
		}
	}
	return IntValue(count), nil
}

// coalesce(Value...) -> Value
func (f *Filter) fnCoalesce(args []Value) (Value, error) {
	for _, arg := range args {
		if arg != nil {
			return arg, nil
		}
	}
	return nil, nil
}

// contains_word(String, String) -> Bool
func (f *Filter) fnContainsWord(args []Value) (Value, error) {
	if len(args) != 2 || args[0] == nil || args[1] == nil {
		return BoolValue(false), nil
	}
	if args[0].Type() != TypeString || args[1].Type() != TypeString {
		return BoolValue(false), nil
	}

	str := string(args[0].(StringValue))
	word := string(args[1].(StringValue))
	pattern := fmt.Sprintf(`\b%s\b`, regexp.QuoteMeta(word))

	re, err := f.getCompiledRegex(pattern)
	if err != nil {
		return BoolValue(false), err
	}

	return BoolValue(re.MatchString(str)), nil
}

// abs(Int|Float) -> Int|Float
func (f *Filter) fnAbs(args []Value) (Value, error) {
	if len(args) != 1 || args[0] == nil {
		return nil, nil
	}
	switch v := args[0].(type) {
	case IntValue:
		if int64(v) < 0 {
			return IntValue(-int64(v)), nil
		}
		return v, nil
	case FloatValue:
		return FloatValue(math.Abs(float64(v))), nil
	}
	return nil, nil
}

// ceil(Float) -> Int
func (f *Filter) fnCeil(args []Value) (Value, error) {
	if len(args) != 1 || args[0] == nil {
		return nil, nil
	}
	switch v := args[0].(type) {
	case FloatValue:
		return IntValue(int64(math.Ceil(float64(v)))), nil
	case IntValue:
		return v, nil
	}
	return nil, nil
}

// floor(Float) -> Int
func (f *Filter) fnFloor(args []Value) (Value, error) {
	if len(args) != 1 || args[0] == nil {
		return nil, nil
	}
	switch v := args[0].(type) {
	case FloatValue:
		return IntValue(int64(math.Floor(float64(v)))), nil
	case IntValue:
		return v, nil
	}
	return nil, nil
}

// round(Float) -> Int
func (f *Filter) fnRound(args []Value) (Value, error) {
	if len(args) != 1 || args[0] == nil {
		return nil, nil
	}
	switch v := args[0].(type) {
	case FloatValue:
		return IntValue(int64(math.Round(float64(v)))), nil
	case IntValue:
		return v, nil
	}
	return nil, nil
}

// is_ipv4(IP) -> Bool
func (f *Filter) fnIsIPv4(args []Value) (Value, error) {
	if len(args) != 1 || args[0] == nil {
		return BoolValue(false), nil
	}
	if args[0].Type() != TypeIP {
		return BoolValue(false), nil
	}
	ip := args[0].(IPValue).IP
	return BoolValue(ip.To4() != nil), nil
}

// is_ipv6(IP) -> Bool
func (f *Filter) fnIsIPv6(args []Value) (Value, error) {
	if len(args) != 1 || args[0] == nil {
		return BoolValue(false), nil
	}
	if args[0].Type() != TypeIP {
		return BoolValue(false), nil
	}
	ip := args[0].(IPValue).IP
	return BoolValue(ip.To4() == nil && len(ip) == 16), nil
}

// is_loopback(IP) -> Bool
func (f *Filter) fnIsLoopback(args []Value) (Value, error) {
	if len(args) != 1 || args[0] == nil {
		return BoolValue(false), nil
	}
	if args[0].Type() != TypeIP {
		return BoolValue(false), nil
	}
	ip := args[0].(IPValue).IP
	return BoolValue(ip.IsLoopback()), nil
}

// regex_extract(String, String) -> String
func (f *Filter) fnRegexExtract(args []Value) (Value, error) {
	if len(args) != 2 || args[0] == nil || args[1] == nil {
		return nil, nil
	}
	if args[0].Type() != TypeString || args[1].Type() != TypeString {
		return nil, nil
	}

	str := string(args[0].(StringValue))
	pattern := string(args[1].(StringValue))

	re, err := f.getCompiledRegex(pattern)
	if err != nil {
		return nil, err
	}

	match := re.FindString(str)
	if match == "" {
		return StringValue(""), nil
	}
	return StringValue(match), nil
}

// intersection(Array, Array) -> Array
func (f *Filter) fnIntersection(args []Value) (Value, error) {
	if len(args) != 2 || args[0] == nil || args[1] == nil {
		return nil, nil
	}
	if args[0].Type() != TypeArray || args[1].Type() != TypeArray {
		return nil, nil
	}

	left := args[0].(ArrayValue)
	right := args[1].(ArrayValue)

	var result ArrayValue
	for _, lElem := range left {
		if right.Contains(lElem) {
			result = append(result, lElem)
		}
	}

	if result == nil {
		return ArrayValue{}, nil
	}
	return result, nil
}

// union(Array, Array) -> Array
func (f *Filter) fnUnion(args []Value) (Value, error) {
	if len(args) != 2 || args[0] == nil || args[1] == nil {
		return nil, nil
	}
	if args[0].Type() != TypeArray || args[1].Type() != TypeArray {
		return nil, nil
	}

	left := args[0].(ArrayValue)
	right := args[1].(ArrayValue)

	result := make(ArrayValue, len(left))
	copy(result, left)

	for _, rElem := range right {
		if !result.Contains(rElem) {
			result = append(result, rElem)
		}
	}

	return result, nil
}

// difference(Array, Array) -> Array
func (f *Filter) fnDifference(args []Value) (Value, error) {
	if len(args) != 2 || args[0] == nil || args[1] == nil {
		return nil, nil
	}
	if args[0].Type() != TypeArray || args[1].Type() != TypeArray {
		return nil, nil
	}

	left := args[0].(ArrayValue)
	right := args[1].(ArrayValue)

	var result ArrayValue
	for _, lElem := range left {
		if !right.Contains(lElem) {
			result = append(result, lElem)
		}
	}

	if result == nil {
		return ArrayValue{}, nil
	}
	return result, nil
}

// contains_any(Array, Array) -> Bool
func (f *Filter) fnContainsAny(args []Value) (Value, error) {
	if len(args) != 2 || args[0] == nil || args[1] == nil {
		return BoolValue(false), nil
	}
	if args[0].Type() != TypeArray || args[1].Type() != TypeArray {
		return BoolValue(false), nil
	}

	left := args[0].(ArrayValue)
	right := args[1].(ArrayValue)

	for _, rElem := range right {
		if left.Contains(rElem) {
			return BoolValue(true), nil
		}
	}
	return BoolValue(false), nil
}

// contains_all(Array, Array) -> Bool
func (f *Filter) fnContainsAll(args []Value) (Value, error) {
	if len(args) != 2 || args[0] == nil || args[1] == nil {
		return BoolValue(false), nil
	}
	if args[0].Type() != TypeArray || args[1].Type() != TypeArray {
		return BoolValue(false), nil
	}

	left := args[0].(ArrayValue)
	right := args[1].(ArrayValue)

	for _, rElem := range right {
		if !left.Contains(rElem) {
			return BoolValue(false), nil
		}
	}
	return BoolValue(true), nil
}
