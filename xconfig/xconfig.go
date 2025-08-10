package xconfig

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"gopkg.in/yaml.v3"
)

var (
	envMacroRegex = regexp.MustCompile(`\$\{env:([^}]+)\}`)
)

type Options struct {
	files         []string
	dirs          []string
	envPrefix     string
	customDefault interface{}
}

type Option func(*Options)

func WithFiles(filenames ...string) Option {
	return func(o *Options) {
		o.files = append(o.files, filenames...)
	}
}

func WithDirs(dirnames ...string) Option {
	return func(o *Options) {
		o.dirs = append(o.dirs, dirnames...)
	}
}

func WithEnv(prefix string) Option {
	return func(o *Options) {
		o.envPrefix = prefix
	}
}

func WithDefault(defaultConfig interface{}) Option {
	return func(o *Options) {
		o.customDefault = defaultConfig
	}
}

func Load(config interface{}, options ...Option) error {
	opts := &Options{}
	for _, option := range options {
		option(opts)
	}

	// Apply default tags first
	configElem, err := validateConfigPointer(config)
	if err != nil {
		return fmt.Errorf("failed to validate config: %w", err)
	}
	if err := applyDefaultTagsRecursive(configElem); err != nil {
		return fmt.Errorf("failed to apply default tags: %w", err)
	}

	// Then call Default() methods (only if no custom defaults provided)
	if opts.customDefault == nil {
		if err := callDefaultMethodsRecursive(configElem); err != nil {
			return fmt.Errorf("failed to call default methods: %w", err)
		}
	}

	// Then apply custom defaults (which completely override Default() methods)
	if opts.customDefault != nil {
		if err := applyCustomDefaults(config, opts.customDefault); err != nil {
			return fmt.Errorf("failed to apply custom defaults: %w", err)
		}
	}

	// Load from directories first, then files
	if len(opts.dirs) > 0 {
		if err := loadFromDirs(config, opts.dirs); err != nil {
			return fmt.Errorf("failed to load from directories: %w", err)
		}
	}

	if len(opts.files) > 0 {
		if err := loadFromFiles(config, opts.files); err != nil {
			return fmt.Errorf("failed to load from files: %w", err)
		}
	}

	// Expand macros in loaded configuration (if any files were loaded)
	if len(opts.dirs) > 0 || len(opts.files) > 0 {
		configValue := reflect.ValueOf(config).Elem()
		expandMacrosInValue(configValue)
	}

	if opts.envPrefix != "" {
		if err := loadFromEnv(config, opts.envPrefix); err != nil {
			return fmt.Errorf("failed to load from environment: %w", err)
		}
	}

	return nil
}

func expandMacros(value string) string {
	return envMacroRegex.ReplaceAllStringFunc(value, func(match string) string {
		// Extract the environment variable name from ${env:VAR_NAME}
		envVar := envMacroRegex.FindStringSubmatch(match)[1]
		if envValue := os.Getenv(envVar); envValue != "" {
			return envValue
		}
		// Return original if env var is not set or empty
		return match
	})
}

func expandMacrosInValue(v reflect.Value) {
	if !v.CanSet() {
		return
	}

	switch v.Kind() {
	case reflect.String:
		if v.String() != "" {
			v.SetString(expandMacros(v.String()))
		}
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if field := v.Field(i); field.CanSet() {
				expandMacrosInValue(field)
			}
		}
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			if elem := v.Index(i); elem.CanSet() {
				expandMacrosInValue(elem)
			}
		}
	case reflect.Map:
		for _, key := range v.MapKeys() {
			mapValue := v.MapIndex(key)
			if mapValue.Kind() == reflect.String && mapValue.String() != "" {
				v.SetMapIndex(key, reflect.ValueOf(expandMacros(mapValue.String())))
			} else if mapValue.CanInterface() {
				newValue := reflect.New(mapValue.Type()).Elem()
				newValue.Set(mapValue)
				expandMacrosInValue(newValue)
				v.SetMapIndex(key, newValue)
			}
		}
	case reflect.Ptr:
		if !v.IsNil() {
			expandMacrosInValue(v.Elem())
		}
	case reflect.Interface:
		if !v.IsNil() && v.Elem().CanSet() {
			expandMacrosInValue(v.Elem())
		}
	}
}

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

func loadFromFile(config interface{}, filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".json":
		return unmarshalJSON(data, config)
	case ".yaml", ".yml":
		return yaml.Unmarshal(data, config)
	default:
		return fmt.Errorf("unsupported file extension %s for file %s", ext, filename)
	}
}

func unmarshalJSON(data []byte, config interface{}) error {
	// First, unmarshal into a map to find duration fields
	var rawData map[string]interface{}
	if err := json.Unmarshal(data, &rawData); err != nil {
		return err
	}

	// Process duration fields recursively
	if err := processDurationFields(rawData, reflect.ValueOf(config).Elem()); err != nil {
		return err
	}

	// Convert back to JSON and unmarshal normally
	processedData, err := json.Marshal(rawData)
	if err != nil {
		return fmt.Errorf("failed to marshal processed data: %w", err)
	}

	return json.Unmarshal(processedData, config)
}

func processDurationFields(data map[string]interface{}, configValue reflect.Value) error {
	if !configValue.IsValid() || configValue.Kind() != reflect.Struct {
		return nil
	}

	configType := configValue.Type()
	for i := 0; i < configValue.NumField(); i++ {
		field := configValue.Field(i)
		fieldType := configType.Field(i)

		// Skip unexported fields
		if !fieldType.IsExported() {
			continue
		}

		// Get the JSON field name
		jsonFieldName := getJSONFieldName(fieldType)
		if jsonFieldName == "" || jsonFieldName == "-" {
			continue
		}

		// Check if this field exists in the data
		fieldData, exists := data[jsonFieldName]
		if !exists {
			continue
		}

		// Handle different field types
		switch {
		case field.Type() == reflect.TypeOf(time.Duration(0)):
			// Handle time.Duration fields
			if strVal, ok := fieldData.(string); ok {
				duration, err := time.ParseDuration(strVal)
				if err != nil {
					return fmt.Errorf("invalid duration value %q for field %s: %w", strVal, fieldType.Name, err)
				}
				// Convert to nanoseconds (int64) for JSON unmarshaling
				data[jsonFieldName] = int64(duration)
			}
		case field.Kind() == reflect.Ptr && field.Type().Elem() == reflect.TypeOf(time.Duration(0)):
			// Handle *time.Duration fields
			if strVal, ok := fieldData.(string); ok {
				duration, err := time.ParseDuration(strVal)
				if err != nil {
					return fmt.Errorf("invalid duration value %q for field %s: %w", strVal, fieldType.Name, err)
				}
				// Convert to nanoseconds (int64) for JSON unmarshaling
				data[jsonFieldName] = int64(duration)
			}
		case field.Kind() == reflect.Struct:
			// Handle nested structs
			if nestedMap, ok := fieldData.(map[string]interface{}); ok {
				if err := processDurationFields(nestedMap, field); err != nil {
					return err
				}
			}
		case field.Kind() == reflect.Ptr && field.Type().Elem().Kind() == reflect.Struct:
			// Handle pointers to structs
			if nestedMap, ok := fieldData.(map[string]interface{}); ok {
				// Create a new struct value to process
				newStruct := reflect.New(field.Type().Elem()).Elem()
				if err := processDurationFields(nestedMap, newStruct); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func getJSONFieldName(fieldType reflect.StructField) string {
	jsonTag := fieldType.Tag.Get("json")
	if jsonTag != "" && jsonTag != "-" {
		return strings.Split(jsonTag, ",")[0]
	}
	// If no json tag, use the field name as-is (Go's default behavior)
	return fieldType.Name
}

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

func isConfigFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ext == ".json" || ext == ".yaml" || ext == ".yml"
}

func scanDirectory(dirname string) ([]string, error) {
	var configFiles []string

	entries, err := os.ReadDir(dirname)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // Directory doesn't exist, return empty list
		}
		return nil, fmt.Errorf("failed to read directory %s: %w", dirname, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue // Skip subdirectories
		}

		filename := entry.Name()
		if isConfigFile(filename) {
			fullPath := filepath.Join(dirname, filename)
			configFiles = append(configFiles, fullPath)
		}
	}

	// Sort files for deterministic loading order
	sort.Strings(configFiles)
	return configFiles, nil
}

func loadFromDirs(config interface{}, dirnames []string) error {
	var allFiles []string

	for _, dirname := range dirnames {
		files, err := scanDirectory(dirname)
		if err != nil {
			return fmt.Errorf("failed to scan directory %s: %w", dirname, err)
		}
		allFiles = append(allFiles, files...)
	}

	return loadFromFiles(config, allFiles)
}

func loadFromFiles(config interface{}, filenames []string) error {
	for _, filename := range filenames {
		if err := loadFromFile(config, filename); err != nil {
			return fmt.Errorf("failed to load file %s: %w", filename, err)
		}
	}
	return nil
}

func loadFromEnv(config interface{}, prefix string) error {
	configElem, err := validateConfigPointer(config)
	if err != nil {
		return err
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

			envKey := prefix + "_" + strings.ToUpper(tagName)

			if err := setFieldFromEnv(field, envKey); err != nil {
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

func parseCommaSeparated(envValue string) []string {
	if envValue == "" {
		return nil
	}
	var result []string
	for _, value := range strings.Split(envValue, ",") {
		if value = strings.TrimSpace(value); value != "" {
			result = append(result, value)
		}
	}
	return result
}

func setSliceFromEnv(field reflect.Value, envValue, envKey string) error {
	if envValue == "" {
		return nil
	}

	values := parseCommaSeparated(envValue)
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

func setMapFromEnv(field reflect.Value, envValue, envKey string) error {
	if envValue == "" {
		return nil
	}

	pairs := parseCommaSeparated(envValue)
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

func setFieldFromEnv(field reflect.Value, envKey string) error {
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

	switch field.Kind() {
	case reflect.Ptr:
		if field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}
		return setFieldFromEnv(field.Elem(), envKey)
	case reflect.Slice:
		return setSliceFromEnv(field, envValue, envKey)
	case reflect.Map:
		return setMapFromEnv(field, envValue, envKey)
	default:
		return setValueFromString(field, envValue, envKey, "")
	}
}
