package xflags

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

var durationType = reflect.TypeFor[time.Duration]()

// intBase returns the radix to use for integer parsing. Base 10 is the default;
// base 0 lets strconv detect 0x/0o/0b prefixes.
func (opt *Option) intBase() int {
	if !opt.baseSet {
		return 10
	}
	return opt.base
}

// applyString sets an option's value from a single raw string token, honoring
// the repeated-option policy. It records that the option was set.
func (opt *Option) applyString(raw string) error {
	// A bool switch with no inline value means true.
	if opt.isBool && raw == "" {
		raw = "true"
	}

	// Store into the destination value when present.
	if opt.value.IsValid() {
		if err := opt.storeValue(raw); err != nil {
			return err
		}
	}

	// Standalone callbacks react to each occurrence. Paired callbacks validate
	// the resolved destination during finalization.
	if opt.callback.IsValid() && !opt.value.IsValid() {
		if err := opt.applyCallback(raw); err != nil {
			return err
		}
	}
	opt.set = true
	opt.count++
	return nil
}

// storeValue writes raw into the option's destination honoring its policy.
func (opt *Option) storeValue(raw string) error {
	switch opt.choice {
	case choiceCount:
		return opt.applyCount(raw)
	case choiceAppend:
		return opt.appendValue(raw)
	case choiceMerge:
		return opt.mergeValue(raw)
	default:
		return opt.setScalar(opt.value, raw)
	}
}

// applyCallback invokes a func-typed option. For a func() error option the raw
// value is ignored; for func(T) error the raw value is converted to T.
func (opt *Option) applyCallback(raw string) error {
	if opt.callback.IsNil() {
		return nil
	}
	ft := opt.callback.Type()
	if ft.NumIn() == 0 {
		return callError(opt.callback.Call(nil))
	}

	arg := reflect.New(ft.In(0)).Elem()
	if err := opt.setScalar(arg, raw); err != nil {
		return err
	}
	return callError(opt.callback.Call([]reflect.Value{arg}))
}

// callError extracts the error return from a reflect Call result.
func callError(out []reflect.Value) error {
	if err := out[0].Interface(); err != nil {
		return err.(error)
	}
	return nil
}

// applyCount increments an integer value. A bare boolean-style occurrence
// increments by one; an explicit numeric value sets the count.
func (opt *Option) applyCount(raw string) error {
	v := opt.value
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if raw != "" {
			return opt.setScalar(v, raw)
		}
		n := v.Int() + 1
		if n < v.Int() || v.OverflowInt(n) {
			return fmt.Errorf("option %s: count overflows %s", optionName(opt), v.Type())
		}
		v.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if raw != "" {
			return opt.setScalar(v, raw)
		}
		n := v.Uint() + 1
		if n == 0 || v.OverflowUint(n) {
			return fmt.Errorf("option %s: count overflows %s", optionName(opt), v.Type())
		}
		v.SetUint(n)
	default:
		return fmt.Errorf("option %s: choice count requires an integer", optionName(opt))
	}
	return nil
}

// appendValue appends a converted element to a slice value.
func (opt *Option) appendValue(raw string) error {
	if opt.value.Kind() != reflect.Slice {
		return fmt.Errorf("option %s: choice append requires a slice", optionName(opt))
	}
	elem := reflect.New(opt.value.Type().Elem()).Elem()
	if err := opt.setScalar(elem, raw); err != nil {
		return err
	}
	opt.value.Set(reflect.Append(opt.value, elem))
	return nil
}

// mergeValue inserts a key=value pair into a map value.
func (opt *Option) mergeValue(raw string) error {
	if opt.value.Kind() != reflect.Map {
		return fmt.Errorf("option %s: choice merge requires a map", optionName(opt))
	}
	if opt.value.Type().Key().Kind() != reflect.String {
		return fmt.Errorf("option %s: map keys must be strings", optionName(opt))
	}

	key, val, ok := strings.Cut(raw, "=")
	if !ok {
		return fmt.Errorf("option %s: expected key=value, got %q", optionName(opt), raw)
	}
	if key == "" {
		return fmt.Errorf("option %s: empty map key in %q", optionName(opt), raw)
	}

	if opt.value.IsNil() {
		opt.value.Set(reflect.MakeMap(opt.value.Type()))
	}
	elem := reflect.New(opt.value.Type().Elem()).Elem()
	if err := opt.setScalar(elem, val); err != nil {
		return err
	}
	mapKey := reflect.New(opt.value.Type().Key()).Elem()
	mapKey.SetString(key)
	opt.value.SetMapIndex(mapKey, elem)
	return nil
}

// checkBinding rejects incompatible policies and paired callback signatures
// while registration can still return an error without parsing any arguments.
func (opt *Option) checkBinding() error {
	if !opt.value.IsValid() {
		return nil
	}
	kind := opt.value.Kind()
	switch opt.choice {
	case choiceCount:
		if kind < reflect.Int || kind > reflect.Uint64 || opt.value.Type() == durationType {
			return fmt.Errorf("option %s: choice count requires an integer", optionName(opt))
		}
	case choiceAppend:
		if kind != reflect.Slice {
			return fmt.Errorf("option %s: choice append requires a slice", optionName(opt))
		}
	case choiceMerge:
		if kind != reflect.Map || opt.value.Type().Key().Kind() != reflect.String {
			return fmt.Errorf("option %s: choice merge requires a map with string keys", optionName(opt))
		}
	}
	scalar := opt.value.Type()
	if opt.choice == choiceAppend || opt.choice == choiceMerge {
		scalar = scalar.Elem()
	}
	if !supportedScalar(scalar) {
		return fmt.Errorf("option %s: unsupported value type %s", optionName(opt), scalar)
	}
	if opt.callback.IsValid() {
		ft := opt.callback.Type()
		if ft.NumIn() == 1 && !opt.value.Type().AssignableTo(ft.In(0)) {
			return fmt.Errorf("option %s: callback argument must accept %s", optionName(opt), opt.value.Type())
		}
	}
	return nil
}

func supportedScalar(t reflect.Type) bool {
	seen := make(map[reflect.Type]bool)
	for t.Kind() == reflect.Pointer {
		if seen[t] {
			return false
		}
		seen[t] = true
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.Bool, reflect.String,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	default:
		return false
	}
}

// cloneValue snapshots supported destinations so repeated Parse calls start
// from the same initial configuration, including pointer/container values.
func cloneValue(v reflect.Value) reflect.Value {
	out := reflect.New(v.Type()).Elem()
	switch v.Kind() {
	case reflect.Pointer:
		if !v.IsNil() {
			out.Set(reflect.New(v.Type().Elem()))
			out.Elem().Set(cloneValue(v.Elem()))
		}
	case reflect.Slice:
		if !v.IsNil() {
			out.Set(reflect.MakeSlice(v.Type(), v.Len(), v.Len()))
			for i := 0; i < v.Len(); i++ {
				out.Index(i).Set(cloneValue(v.Index(i)))
			}
		}
	case reflect.Map:
		if !v.IsNil() {
			out.Set(reflect.MakeMap(v.Type()))
			iter := v.MapRange()
			for iter.Next() {
				out.SetMapIndex(iter.Key(), cloneValue(iter.Value()))
			}
		}
	default:
		out.Set(v)
	}
	return out
}

// setScalar converts a raw string into a single non-container value.
func (opt *Option) setScalar(dst reflect.Value, raw string) error {
	if dst.Kind() == reflect.Pointer {
		if dst.IsNil() {
			dst.Set(reflect.New(dst.Type().Elem()))
		}
		return opt.setScalar(dst.Elem(), raw)
	}

	if dst.Type() == durationType {
		d, err := time.ParseDuration(raw)
		if err != nil {
			return fmt.Errorf("option %s: invalid duration %q", optionName(opt), raw)
		}
		dst.SetInt(int64(d))
		return nil
	}

	switch dst.Kind() {
	case reflect.String:
		dst.SetString(raw)
	case reflect.Bool:
		b, err := strconv.ParseBool(raw)
		if err != nil {
			return fmt.Errorf("option %s: invalid bool %q", optionName(opt), raw)
		}
		dst.SetBool(b)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(raw, opt.intBase(), 64)
		if err != nil {
			return fmt.Errorf("option %s: invalid integer %q", optionName(opt), raw)
		}
		if dst.OverflowInt(n) {
			return fmt.Errorf("option %s: integer %q overflows %s", optionName(opt), raw, dst.Kind())
		}
		dst.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(raw, opt.intBase(), 64)
		if err != nil {
			return fmt.Errorf("option %s: invalid unsigned integer %q", optionName(opt), raw)
		}
		if dst.OverflowUint(n) {
			return fmt.Errorf("option %s: value %q overflows %s", optionName(opt), raw, dst.Kind())
		}
		dst.SetUint(n)
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return fmt.Errorf("option %s: invalid float %q", optionName(opt), raw)
		}
		if dst.OverflowFloat(f) {
			return fmt.Errorf("option %s: value %q overflows %s", optionName(opt), raw, dst.Kind())
		}
		dst.SetFloat(f)
	default:
		return fmt.Errorf("option %s: unsupported type %s", optionName(opt), dst.Kind())
	}
	return nil
}
