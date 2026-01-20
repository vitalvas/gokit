package wirefilter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSchemaFunctionControl(t *testing.T) {
	t.Run("default mode is blocklist - all functions allowed", func(t *testing.T) {
		schema := NewSchema().AddField("name", TypeString)

		_, err := Compile(`lower(name) == "test"`, schema)
		assert.NoError(t, err)

		_, err = Compile(`upper(name) == "TEST"`, schema)
		assert.NoError(t, err)

		_, err = Compile(`len(name) > 0`, schema)
		assert.NoError(t, err)
	})

	t.Run("blocklist mode - disable specific functions", func(t *testing.T) {
		schema := NewSchema().
			AddField("name", TypeString).
			DisableFunctions("lower")

		_, err := Compile(`lower(name) == "test"`, schema)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "function not allowed: lower")

		// Other functions still work
		_, err = Compile(`upper(name) == "TEST"`, schema)
		assert.NoError(t, err)
	})

	t.Run("blocklist mode - disable multiple functions", func(t *testing.T) {
		schema := NewSchema().
			AddField("name", TypeString).
			DisableFunctions("lower", "upper", "len")

		_, err := Compile(`lower(name) == "test"`, schema)
		assert.Error(t, err)

		_, err = Compile(`upper(name) == "TEST"`, schema)
		assert.Error(t, err)

		_, err = Compile(`len(name) > 0`, schema)
		assert.Error(t, err)

		// starts_with still works
		_, err = Compile(`starts_with(name, "test")`, schema)
		assert.NoError(t, err)
	})

	t.Run("allowlist mode - only enabled functions work", func(t *testing.T) {
		schema := NewSchema().
			AddField("name", TypeString).
			SetFunctionMode(FunctionModeAllowlist).
			EnableFunctions("lower")

		_, err := Compile(`lower(name) == "test"`, schema)
		assert.NoError(t, err)

		_, err = Compile(`upper(name) == "TEST"`, schema)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "function not allowed: upper")
	})

	t.Run("allowlist mode - enable multiple functions", func(t *testing.T) {
		schema := NewSchema().
			AddField("name", TypeString).
			SetFunctionMode(FunctionModeAllowlist).
			EnableFunctions("lower", "upper", "len")

		_, err := Compile(`lower(name) == "test"`, schema)
		assert.NoError(t, err)

		_, err = Compile(`upper(name) == "TEST"`, schema)
		assert.NoError(t, err)

		_, err = Compile(`len(name) > 0`, schema)
		assert.NoError(t, err)

		// starts_with is not enabled
		_, err = Compile(`starts_with(name, "test")`, schema)
		assert.Error(t, err)
	})

	t.Run("function names are case-insensitive", func(t *testing.T) {
		schema := NewSchema().
			AddField("name", TypeString).
			DisableFunctions("LOWER")

		_, err := Compile(`lower(name) == "test"`, schema)
		assert.Error(t, err)

		_, err = Compile(`LOWER(name) == "test"`, schema)
		assert.Error(t, err)

		_, err = Compile(`Lower(name) == "test"`, schema)
		assert.Error(t, err)
	})

	t.Run("enable after disable re-enables function", func(t *testing.T) {
		schema := NewSchema().
			AddField("name", TypeString).
			DisableFunctions("lower").
			EnableFunctions("lower")

		_, err := Compile(`lower(name) == "test"`, schema)
		assert.NoError(t, err)
	})

	t.Run("disable after enable disables function", func(t *testing.T) {
		schema := NewSchema().
			AddField("name", TypeString).
			SetFunctionMode(FunctionModeAllowlist).
			EnableFunctions("lower").
			DisableFunctions("lower")

		_, err := Compile(`lower(name) == "test"`, schema)
		assert.Error(t, err)
	})

	t.Run("allowlist mode with no functions enabled", func(t *testing.T) {
		schema := NewSchema().
			AddField("name", TypeString).
			SetFunctionMode(FunctionModeAllowlist)

		_, err := Compile(`lower(name) == "test"`, schema)
		assert.Error(t, err)

		// Non-function expressions still work
		_, err = Compile(`name == "test"`, schema)
		assert.NoError(t, err)
	})

	t.Run("IsFunctionAllowed - blocklist mode", func(t *testing.T) {
		schema := NewSchema().DisableFunctions("lower")

		assert.False(t, schema.IsFunctionAllowed("lower"))
		assert.False(t, schema.IsFunctionAllowed("LOWER"))
		assert.True(t, schema.IsFunctionAllowed("upper"))
	})

	t.Run("IsFunctionAllowed - allowlist mode", func(t *testing.T) {
		schema := NewSchema().
			SetFunctionMode(FunctionModeAllowlist).
			EnableFunctions("lower")

		assert.True(t, schema.IsFunctionAllowed("lower"))
		assert.True(t, schema.IsFunctionAllowed("LOWER"))
		assert.False(t, schema.IsFunctionAllowed("upper"))
	})

	t.Run("nested function calls respect rules", func(t *testing.T) {
		schema := NewSchema().
			AddField("name", TypeString).
			DisableFunctions("lower")

		// len(lower(name)) should fail because lower is disabled
		_, err := Compile(`len(lower(name)) > 0`, schema)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "function not allowed: lower")

		// len(upper(name)) should work
		_, err = Compile(`len(upper(name)) > 0`, schema)
		assert.NoError(t, err)
	})

	t.Run("nil schema allows all functions", func(t *testing.T) {
		_, err := Compile(`lower(name) == "test"`, nil)
		assert.NoError(t, err)
	})
}
