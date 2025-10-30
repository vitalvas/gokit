package xconfig

import (
	"fmt"
	"reflect"
	"strconv"
	"time"
)

func validateConfigPointer(config interface{}) (reflect.Value, error) {
	configValue := reflect.ValueOf(config)
	if configValue.Kind() != reflect.Ptr || configValue.IsNil() {
		return reflect.Value{}, fmt.Errorf("config must be a non-nil pointer")
	}
	configElem := configValue.Elem()
	if !configElem.CanSet() {
		return reflect.Value{}, fmt.Errorf("config is not settable")
	}
	return configElem, nil
}

func applyCustomDefaults(config, customDefault interface{}) error {
	configElem, err := validateConfigPointer(config)
	if err != nil {
		return err
	}

	defaultValue := reflect.ValueOf(customDefault)
	if configElem.Type() != defaultValue.Type() {
		return fmt.Errorf("custom default type %s does not match config type %s",
			defaultValue.Type(), configElem.Type())
	}

	return copyValues(configElem, defaultValue)
}

func copyValues(dst, src reflect.Value) error {
	if dst.Type() != src.Type() {
		return fmt.Errorf("type mismatch: %s != %s", dst.Type(), src.Type())
	}

	switch dst.Kind() {
	case reflect.Struct:
		for i := 0; i < dst.NumField(); i++ {
			dstField := dst.Field(i)
			srcField := src.Field(i)

			if !dstField.CanSet() {
				continue
			}

			// Only copy non-zero values from custom defaults
			if !srcField.IsZero() {
				// For struct fields, do complete replacement to override Default() methods
				if srcField.Kind() == reflect.Struct {
					dstField.Set(srcField)
				} else {
					if err := copyValues(dstField, srcField); err != nil {
						return err
					}
				}
			}
		}
	case reflect.Slice:
		if !src.IsNil() {
			newSlice := reflect.MakeSlice(dst.Type(), src.Len(), src.Cap())
			reflect.Copy(newSlice, src)
			dst.Set(newSlice)
		}
	case reflect.Map:
		if !src.IsNil() {
			newMap := reflect.MakeMap(dst.Type())
			for _, key := range src.MapKeys() {
				newMap.SetMapIndex(key, src.MapIndex(key))
			}
			dst.Set(newMap)
		}
	case reflect.Ptr:
		if !src.IsNil() {
			if dst.IsNil() {
				dst.Set(reflect.New(dst.Type().Elem()))
			}
			if err := copyValues(dst.Elem(), src.Elem()); err != nil {
				return err
			}
		}
	case reflect.Interface:
		if !src.IsNil() {
			dst.Set(src)
		}
	default:
		if src.IsValid() {
			dst.Set(src)
		}
	}

	return nil
}

func applyDefaultTagsRecursive(v reflect.Value) error {
	if !v.CanSet() {
		return nil
	}

	switch v.Kind() {
	case reflect.Struct:
		t := v.Type()

		// Apply default tag values to struct fields
		for i := 0; i < v.NumField(); i++ {
			field := v.Field(i)
			fieldType := t.Field(i)

			if !field.CanSet() {
				continue
			}

			if err := applyDefaultTag(field, fieldType); err != nil {
				return fmt.Errorf("failed to apply default tag to field %s: %w", fieldType.Name, err)
			}
		}

		// Process nested fields recursively
		for i := 0; i < v.NumField(); i++ {
			field := v.Field(i)
			if !field.CanSet() {
				continue
			}
			if err := applyDefaultTagsRecursive(field); err != nil {
				return err
			}
		}
	case reflect.Slice:
		// Process each element in the slice
		for i := 0; i < v.Len(); i++ {
			elem := v.Index(i)
			if elem.CanSet() {
				if err := applyDefaultTagsRecursive(elem); err != nil {
					return err
				}
			}
		}
	case reflect.Ptr:
		if v.IsNil() && v.CanSet() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		if !v.IsNil() {
			return applyDefaultTagsRecursive(v.Elem())
		}
	}

	return nil
}

func callDefaultMethodsRecursive(v reflect.Value) error {
	if !v.CanSet() {
		return nil
	}

	switch v.Kind() {
	case reflect.Struct:
		// Call Default method if it exists
		if method := v.Addr().MethodByName("Default"); method.IsValid() {
			method.Call(nil)
		}

		// Process nested fields recursively
		for i := 0; i < v.NumField(); i++ {
			field := v.Field(i)
			if !field.CanSet() {
				continue
			}
			if err := callDefaultMethodsRecursive(field); err != nil {
				return err
			}
		}
	case reflect.Slice:
		// Process each element in the slice
		for i := 0; i < v.Len(); i++ {
			elem := v.Index(i)
			if elem.CanSet() {
				if err := callDefaultMethodsRecursive(elem); err != nil {
					return err
				}
			}
		}
	case reflect.Ptr:
		if v.IsNil() && v.CanSet() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		if !v.IsNil() {
			return callDefaultMethodsRecursive(v.Elem())
		}
	}

	return nil
}

func applyDefaultTag(field reflect.Value, fieldType reflect.StructField) error {
	defaultValue := fieldType.Tag.Get("default")
	if defaultValue == "" {
		return nil
	}

	// Only apply default if field is zero value
	if !field.IsZero() {
		return nil
	}

	// Handle time.Duration specifically
	if field.Type() == reflect.TypeOf(time.Duration(0)) {
		duration, err := time.ParseDuration(defaultValue)
		if err != nil {
			return fmt.Errorf("invalid duration default value %q for field %s: %w", defaultValue, fieldType.Name, err)
		}
		field.SetInt(int64(duration))
		return nil
	}

	switch field.Kind() {
	case reflect.String:
		field.SetString(defaultValue)
	case reflect.Bool:
		val, err := strconv.ParseBool(defaultValue)
		if err != nil {
			return fmt.Errorf("invalid boolean default value %q for field %s", defaultValue, fieldType.Name)
		}
		field.SetBool(val)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		val, err := strconv.ParseInt(defaultValue, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid integer default value %q for field %s", defaultValue, fieldType.Name)
		}
		if field.OverflowInt(val) {
			return fmt.Errorf("integer default value %q overflows field %s of type %s", defaultValue, fieldType.Name, field.Type())
		}
		field.SetInt(val)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		val, err := strconv.ParseUint(defaultValue, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid unsigned integer default value %q for field %s", defaultValue, fieldType.Name)
		}
		if field.OverflowUint(val) {
			return fmt.Errorf("unsigned integer default value %q overflows field %s of type %s", defaultValue, fieldType.Name, field.Type())
		}
		field.SetUint(val)
	case reflect.Float32, reflect.Float64:
		val, err := strconv.ParseFloat(defaultValue, 64)
		if err != nil {
			return fmt.Errorf("invalid float default value %q for field %s", defaultValue, fieldType.Name)
		}
		if field.OverflowFloat(val) {
			return fmt.Errorf("float default value %q overflows field %s of type %s", defaultValue, fieldType.Name, field.Type())
		}
		field.SetFloat(val)
	case reflect.Ptr:
		if field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}
		return applyDefaultTag(field.Elem(), fieldType)
	default:
		return fmt.Errorf("unsupported field type %s for default tag on field %s", field.Kind(), fieldType.Name)
	}

	return nil
}
