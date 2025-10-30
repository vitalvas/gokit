package xconfig

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode"
)

func camelToSnake(s string) string {
	var result strings.Builder
	runes := []rune(s)

	for i, r := range runes {
		if i > 0 && unicode.IsUpper(r) {
			// Add underscore before uppercase letters, except when:
			// 1. Previous character is also uppercase (part of acronym)
			// 2. Next character is lowercase (end of acronym)
			prevUpper := i > 0 && unicode.IsUpper(runes[i-1])
			nextLower := i < len(runes)-1 && unicode.IsLower(runes[i+1])

			if !prevUpper || nextLower {
				result.WriteByte('_')
			}
		}
		result.WriteRune(unicode.ToLower(r))
	}
	return result.String()
}

func getFieldTagName(fieldType reflect.StructField) string {
	envTag := fieldType.Tag.Get("env")
	if envTag != "" && envTag != "-" {
		return strings.Split(envTag, ",")[0]
	}

	yamlTag := fieldType.Tag.Get("yaml")
	if yamlTag != "" && yamlTag != "-" {
		return strings.Split(yamlTag, ",")[0]
	}

	jsonTag := fieldType.Tag.Get("json")
	if jsonTag != "" && jsonTag != "-" {
		return strings.Split(jsonTag, ",")[0]
	}

	// Use field name converted to snake_case if no tags are present
	return camelToSnake(fieldType.Name)
}

func loadFromEnv(config interface{}, prefix string) error {
	configElem, err := validateConfigPointer(config)
	if err != nil {
		return err
	}
	// Handle special case where prefix is EnvSkipPrefix to not prepend any prefix
	if prefix == EnvSkipPrefix {
		return loadFromEnvRecursive(configElem, "")
	}
	return loadFromEnvRecursive(configElem, strings.ToUpper(prefix))
}

func loadFromEnvRecursive(v reflect.Value, prefix string) error {
	if !v.CanSet() {
		return nil
	}

	t := v.Type()

	if v.Kind() == reflect.Struct {
		for i := 0; i < v.NumField(); i++ {
			field := v.Field(i)
			fieldType := t.Field(i)

			if !field.CanSet() {
				continue
			}

			tagName := getFieldTagName(fieldType)
			if tagName == "" {
				continue
			}

			// Check if env tag is used - if so, use original prefix, not nested prefix
			envTag := fieldType.Tag.Get("env")
			var envKey string
			if envTag != "" && envTag != "-" {
				// For env tags, use root prefix + env tag name (skip intermediate prefixes)
				if prefix == "" {
					// No prefix case (WithEnv(EnvSkipPrefix))
					envKey = strings.ToUpper(strings.Split(envTag, ",")[0])
				} else {
					rootPrefix := strings.Split(prefix, "_")[0] // Get the original prefix (e.g., "TEST")
					envKey = rootPrefix + "_" + strings.ToUpper(strings.Split(envTag, ",")[0])
				}
			} else {
				// Use standard prefix + tag name
				if prefix == "" {
					// No prefix case (WithEnv(EnvSkipPrefix))
					envKey = strings.ToUpper(tagName)
				} else {
					envKey = prefix + "_" + strings.ToUpper(tagName)
				}
			}

			if err := setFieldFromEnv(field, fieldType, envKey); err != nil {
				return fmt.Errorf("failed to set field %s: %w", fieldType.Name, err)
			}

			if field.Kind() == reflect.Struct {
				if err := loadFromEnvRecursive(field, envKey); err != nil {
					return err
				}
			} else if field.Kind() == reflect.Ptr && !field.IsNil() {
				if err := loadFromEnvRecursive(field.Elem(), envKey); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func getEnvSeparator(fieldType reflect.StructField) string {
	separator := fieldType.Tag.Get("envSeparator")
	if separator == "" {
		return "," // Default separator
	}
	return separator
}

func parseSeparated(envValue, separator string) []string {
	if envValue == "" {
		return nil
	}
	var result []string
	for _, value := range strings.Split(envValue, separator) {
		if value = strings.TrimSpace(value); value != "" {
			result = append(result, value)
		}
	}
	return result
}

// parseCommaSeparated is deprecated, use parseSeparated instead.
// Kept for backward compatibility.
func parseCommaSeparated(envValue string) []string {
	return parseSeparated(envValue, ",")
}

func setSliceFromEnv(field reflect.Value, envValue, envKey, separator string) error {
	if envValue == "" {
		return nil
	}

	values := parseSeparated(envValue, separator)
	elemType := field.Type().Elem()
	slice := reflect.MakeSlice(field.Type(), 0, len(values))

	for _, value := range values {
		elem := reflect.New(elemType).Elem()
		if err := setValueFromString(elem, value, envKey, "in slice"); err != nil {
			return err
		}
		slice = reflect.Append(slice, elem)
	}

	field.Set(slice)
	return nil
}

func setValueFromString(elem reflect.Value, value, envKey, context string) error {
	switch elem.Kind() {
	case reflect.String:
		elem.SetString(value)
	case reflect.Bool:
		val, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid boolean value %s for %s: %s", context, envKey, value)
		}
		elem.SetBool(val)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		val, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid integer value %s for %s: %s", context, envKey, value)
		}
		elem.SetInt(val)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		val, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid unsigned integer value %s for %s: %s", context, envKey, value)
		}
		elem.SetUint(val)
	case reflect.Float32, reflect.Float64:
		val, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("invalid float value %s for %s: %s", context, envKey, value)
		}
		elem.SetFloat(val)
	default:
		return fmt.Errorf("unsupported type %s %s for %s", elem.Kind(), context, envKey)
	}
	return nil
}

func setMapFromEnv(field reflect.Value, envValue, envKey, separator string) error {
	if envValue == "" {
		return nil
	}

	pairs := parseSeparated(envValue, separator)
	keyType := field.Type().Key()
	if keyType.Kind() != reflect.String {
		return fmt.Errorf("unsupported map key type %s for %s, only string keys are supported", keyType.Kind(), envKey)
	}

	mapValue := reflect.MakeMap(field.Type())
	valueType := field.Type().Elem()

	for _, pair := range pairs {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid map pair format for %s: %s (expected key=value)", envKey, pair)
		}

		key := strings.TrimSpace(parts[0])
		if key == "" {
			return fmt.Errorf("empty key in map pair for %s: %s", envKey, pair)
		}

		valueValue := reflect.New(valueType).Elem()
		value := strings.TrimSpace(parts[1])
		if err := setValueFromString(valueValue, value, envKey, "in map"); err != nil {
			return fmt.Errorf("invalid map value for key %s in %s: %w", key, envKey, err)
		}

		mapValue.SetMapIndex(reflect.ValueOf(key), valueValue)
	}

	field.Set(mapValue)
	return nil
}

func setFieldFromEnv(field reflect.Value, fieldType reflect.StructField, envKey string) error {
	envValue := os.Getenv(envKey)
	if envValue == "" {
		return nil
	}

	// Handle time.Duration specifically
	if field.Type() == reflect.TypeOf(time.Duration(0)) {
		duration, err := time.ParseDuration(envValue)
		if err != nil {
			return fmt.Errorf("invalid duration value for %s: %s", envKey, envValue)
		}
		field.SetInt(int64(duration))
		return nil
	}

	separator := getEnvSeparator(fieldType)

	switch field.Kind() {
	case reflect.Ptr:
		if field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}
		return setFieldFromEnv(field.Elem(), fieldType, envKey)
	case reflect.Slice:
		return setSliceFromEnv(field, envValue, envKey, separator)
	case reflect.Map:
		return setMapFromEnv(field, envValue, envKey, separator)
	default:
		return setValueFromString(field, envValue, envKey, "")
	}
}
