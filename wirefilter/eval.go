package wirefilter

import (
	"fmt"
	"math"
	"net"
	"regexp"
	"strings"
)

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
