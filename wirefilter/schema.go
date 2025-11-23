package wirefilter

import "fmt"

// Field represents a named field with a specific type in a schema.
type Field struct {
	Name string
	Type Type
}

// Schema defines the structure of fields that can be used in filter expressions.
// It provides validation to ensure that filter expressions only reference defined fields.
type Schema struct {
	fields map[string]Field
}

// NewSchema creates a new schema.
// If fields are provided, initializes the schema with those fields.
// Multiple field maps can be provided and will be merged.
// Otherwise, creates an empty schema.
func NewSchema(fields ...map[string]Type) *Schema {
	s := &Schema{
		fields: make(map[string]Field),
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
	}
	return nil
}
