package xconfig

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

func loadFromFile(config interface{}, filename string, strict bool) error {
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
		return unmarshalJSON(data, config, strict)
	case ".yaml", ".yml":
		return unmarshalYAML(data, config, strict)
	default:
		return fmt.Errorf("unsupported file extension %s for file %s", ext, filename)
	}
}

func unmarshalYAML(data []byte, config interface{}, strict bool) error {
	dec := yaml.NewDecoder(strings.NewReader(string(data)))
	if strict {
		dec.KnownFields(true)
	}
	return dec.Decode(config)
}

func unmarshalJSON(data []byte, config interface{}, strict bool) error {
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

	if strict {
		dec := json.NewDecoder(strings.NewReader(string(processedData)))
		dec.DisallowUnknownFields()
		return dec.Decode(config)
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

func isConfigFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ext == ".json" || ext == ".yaml" || ext == ".yml"
}

func loadFromFiles(config interface{}, filenames []string, strict bool) error {
	for _, filename := range filenames {
		if err := loadFromFile(config, filename, strict); err != nil {
			return fmt.Errorf("failed to load file %s: %w", filename, err)
		}
	}
	return nil
}
