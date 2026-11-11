package xflags

import (
	"fmt"
	"reflect"
)

// numericKind reports whether k is an integer, unsigned, or float kind, the
// kinds that support min and max bounds.
func numericKind(k reflect.Kind) bool {
	switch k {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	default:
		return false
	}
}

// checkBoundTypes validates the min and max tags at registration time: the
// scalar must be a numeric kind, each bound must parse as that type, and a lower
// bound may not exceed an upper bound.
func (opt *Option) checkBoundTypes(scalar reflect.Type) error {
	if !opt.minBoundSet && !opt.maxBoundSet {
		return nil
	}
	elem := scalar
	for elem.Kind() == reflect.Pointer {
		elem = elem.Elem()
	}
	if !numericKind(elem.Kind()) {
		return fmt.Errorf("option %s: min/max require a numeric type, got %s", optionName(opt), scalar)
	}

	var minVal, maxVal reflect.Value
	if opt.minBoundSet {
		v, err := opt.parseBound(scalar, opt.minBound, "min")
		if err != nil {
			return err
		}
		minVal = v
	}
	if opt.maxBoundSet {
		v, err := opt.parseBound(scalar, opt.maxBound, "max")
		if err != nil {
			return err
		}
		maxVal = v
	}
	if opt.minBoundSet && opt.maxBoundSet && compareNumeric(maxVal, minVal) < 0 {
		return fmt.Errorf("option %s: max %q is less than min %q", optionName(opt), opt.maxBound, opt.minBound)
	}
	return nil
}

// parseBound converts a bound tag value into the scalar type, reporting a
// registration error that names which bound failed.
func (opt *Option) parseBound(scalar reflect.Type, raw, which string) (reflect.Value, error) {
	dst := reflect.New(scalar).Elem()
	if err := opt.setScalar(dst, raw); err != nil {
		return reflect.Value{}, fmt.Errorf("option %s: invalid %s bound %q", optionName(opt), which, raw)
	}
	return dst, nil
}

// checkBounds enforces the min and max tags against the resolved value. A nil
// pointer destination (an unset optional) is skipped.
func (opt *Option) checkBounds() error {
	if !opt.minBoundSet && !opt.maxBoundSet || !opt.value.IsValid() {
		return nil
	}
	v := opt.value
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}
	scalar := v.Type()
	if opt.minBoundSet {
		bound := reflect.New(scalar).Elem()
		if err := opt.setScalar(bound, opt.minBound); err != nil {
			return err
		}
		if compareNumeric(v, bound) < 0 {
			return fmt.Errorf("option %s: value %v is less than minimum %s", optionName(opt), v.Interface(), opt.minBound)
		}
	}
	if opt.maxBoundSet {
		bound := reflect.New(scalar).Elem()
		if err := opt.setScalar(bound, opt.maxBound); err != nil {
			return err
		}
		if compareNumeric(v, bound) > 0 {
			return fmt.Errorf("option %s: value %v is greater than maximum %s", optionName(opt), v.Interface(), opt.maxBound)
		}
	}
	return nil
}

// compareNumeric compares two numeric reflect values of the same kind family,
// returning -1, 0, or 1. Pointers are dereferenced first.
func compareNumeric(a, b reflect.Value) int {
	for a.Kind() == reflect.Pointer {
		a, b = a.Elem(), b.Elem()
	}
	switch a.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return cmpOrdered(a.Int(), b.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return cmpOrdered(a.Uint(), b.Uint())
	case reflect.Float32, reflect.Float64:
		return cmpOrdered(a.Float(), b.Float())
	default:
		return 0
	}
}

func cmpOrdered[T int64 | uint64 | float64](a, b T) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}
