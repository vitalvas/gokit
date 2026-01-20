package wirefilter

import (
	"fmt"
	"strings"
)

// FunctionMode defines how function availability is controlled.
type FunctionMode int

const (
	// FunctionModeBlocklist allows all functions except those explicitly disabled.
	// This is the default mode.
	FunctionModeBlocklist FunctionMode = iota
	// FunctionModeAllowlist allows only functions that are explicitly enabled.
	FunctionModeAllowlist
)

// Field represents a named field with a specific type in a schema.
type Field struct {
	Name string
	Type Type
}

// Schema defines the structure of fields that can be used in filter expressions.
// It provides validation to ensure that filter expressions only reference defined fields.
type Schema struct {
	fields        map[string]Field
	functionMode  FunctionMode
	functionRules map[string]bool // true = enabled, false = disabled
}

// NewSchema creates a new schema.
// If fields are provided, initializes the schema with those fields.
// Multiple field maps can be provided and will be merged.
// Otherwise, creates an empty schema.
// Default function mode is Blocklist (all functions allowed).
func NewSchema(fields ...map[string]Type) *Schema {
	s := &Schema{
		fields:        make(map[string]Field),
		functionMode:  FunctionModeBlocklist,
		functionRules: make(map[string]bool),
	}
	for _, fieldMap := range fields {
		for name, fieldType := range fieldMap {
			s.fields[name] = Field{
				Name: name,
				Type: fieldType,
			}
		}
	}
	return s
}

// SetFunctionMode sets the function availability mode.
// In Blocklist mode (default), all functions are allowed except those disabled.
// In Allowlist mode, only explicitly enabled functions are allowed.
// Returns the schema to allow method chaining.
func (s *Schema) SetFunctionMode(mode FunctionMode) *Schema {
	s.functionMode = mode
	return s
}

// EnableFunctions enables one or more functions by name.
// In Allowlist mode, this allows the functions to be used.
// In Blocklist mode, this removes the functions from the disabled list.
// Function names are case-insensitive.
// Returns the schema to allow method chaining.
func (s *Schema) EnableFunctions(names ...string) *Schema {
	for _, name := range names {
		s.functionRules[strings.ToLower(name)] = true
	}
	return s
}

// DisableFunctions disables one or more functions by name.
// In Blocklist mode, this prevents the functions from being used.
// In Allowlist mode, this removes the functions from the enabled list.
// Function names are case-insensitive.
// Returns the schema to allow method chaining.
func (s *Schema) DisableFunctions(names ...string) *Schema {
	for _, name := range names {
		s.functionRules[strings.ToLower(name)] = false
	}
	return s
}

// IsFunctionAllowed checks if a function is allowed based on the current mode and rules.
// Function names are case-insensitive.
func (s *Schema) IsFunctionAllowed(name string) bool {
	name = strings.ToLower(name)
	enabled, hasRule := s.functionRules[name]

	switch s.functionMode {
	case FunctionModeAllowlist:
		// In allowlist mode, function must be explicitly enabled
		return hasRule && enabled
	case FunctionModeBlocklist:
		// In blocklist mode, function is allowed unless explicitly disabled
		if hasRule {
			return enabled
		}
		return true
	}
	return true
}

// AddField adds a field to the schema with the specified name and type.
// Returns the schema to allow method chaining.
func (s *Schema) AddField(name string, fieldType Type) *Schema {
	s.fields[name] = Field{
		Name: name,
		Type: fieldType,
	}
	return s
}

// GetField retrieves a field from the schema by name.
// Returns the field and true if found, or an empty field and false if not found.
func (s *Schema) GetField(name string) (Field, bool) {
	field, ok := s.fields[name]
	return field, ok
}

// Validate checks that all field references in the expression exist in the schema.
// Returns an error if any field is not defined in the schema.
func (s *Schema) Validate(expr Expression) error {
	return s.validateExpression(expr)
}

func (s *Schema) validateExpression(expr Expression) error {
	switch e := expr.(type) {
	case *BinaryExpr:
		if err := s.validateExpression(e.Left); err != nil {
			return err
		}
		if err := s.validateExpression(e.Right); err != nil {
			return err
		}
	case *UnaryExpr:
		if err := s.validateExpression(e.Operand); err != nil {
			return err
		}
	case *FieldExpr:
		if _, ok := s.GetField(e.Name); !ok {
			return fmt.Errorf("unknown field: %s", e.Name)
		}
	case *ArrayExpr:
		for _, elem := range e.Elements {
			if err := s.validateExpression(elem); err != nil {
				return err
			}
		}
	case *RangeExpr:
		if err := s.validateExpression(e.Start); err != nil {
			return err
		}
		if err := s.validateExpression(e.End); err != nil {
			return err
		}
	case *IndexExpr:
		if err := s.validateExpression(e.Object); err != nil {
			return err
		}
	case *UnpackExpr:
		if err := s.validateExpression(e.Array); err != nil {
			return err
		}
	case *ListRefExpr:
		// List references are validated at runtime
	case *FunctionCallExpr:
		if !s.IsFunctionAllowed(e.Name) {
			return fmt.Errorf("function not allowed: %s", e.Name)
		}
		for _, arg := range e.Arguments {
			if err := s.validateExpression(arg); err != nil {
				return err
			}
		}
	}
	return nil
}
