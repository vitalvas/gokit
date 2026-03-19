package wirefilter

import (
	"fmt"
	"math"
	"net"
	"net/url"
	"regexp"
	"strings"
)

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

	// Check user-defined functions in the execution context (with optional caching)
	if fn, ok := ctx.GetFunc(name); ok {
		key := cacheKey(name, args)
		if cached, ok := ctx.getCached(key); ok {
			return cached, nil
		}
		result, err := fn(args)
		if err != nil {
			return nil, err
		}
		ctx.setCache(key, result)
		return result, nil
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
		"exists":        f.fnExists,
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

// exists(Value) -> Bool
// Returns true if the argument is not nil (field is set in context).
func (f *Filter) fnExists(args []Value) (Value, error) {
	if len(args) != 1 {
		return BoolValue(false), nil
	}
	return BoolValue(args[0] != nil), nil
}
