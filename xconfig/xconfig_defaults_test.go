package xconfig

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Multi-level test structures
type MultiLevelConfig struct {
	Name  string         `yaml:"name" json:"name"`
	Level LevelOneConfig `yaml:"level" json:"level"`
}

func (c *MultiLevelConfig) Default() {
	*c = MultiLevelConfig{
		Name: "multi-level",
	}
}

type LevelOneConfig struct {
	Value string         `yaml:"value" json:"value"`
	Level LevelTwoConfig `yaml:"level" json:"level"`
}

func (c *LevelOneConfig) Default() {
	*c = LevelOneConfig{
		Value: "level-one",
	}
}

type LevelTwoConfig struct {
	Setting string `yaml:"setting" json:"setting"`
}

func (c *LevelTwoConfig) Default() {
	*c = LevelTwoConfig{
		Setting: "level-two",
	}
}

// Default tag test structures
type DefaultTagConfig struct {
	StringField  string  `yaml:"string_field" default:"default_string"`
	IntField     int     `yaml:"int_field" default:"42"`
	BoolField    bool    `yaml:"bool_field" default:"true"`
	FloatField   float64 `yaml:"float_field" default:"3.14"`
	UintField    uint    `yaml:"uint_field" default:"100"`
	PointerField *string `yaml:"pointer_field" default:"default_pointer"`
}

type NestedDefaultConfig struct {
	Parent ParentConfig `yaml:"parent"`
}

type ParentConfig struct {
	Name  string      `yaml:"name" default:"parent_name"`
	Child ChildConfig `yaml:"child"`
}

type ChildConfig struct {
	Value string `yaml:"value" default:"child_value"`
}

type DurationDefaultTagConfig struct {
	Timeout    time.Duration  `yaml:"timeout" default:"30s"`
	RetryDelay time.Duration  `yaml:"retry_delay" default:"5m"`
	MaxWait    time.Duration  `yaml:"max_wait" default:"1h"`
	Optional   *time.Duration `yaml:"optional" default:"15s"`
}

type InvalidDefaultConfig struct {
	BadBool bool `yaml:"bad_bool" default:"invalid_bool"`
}

type InvalidDurationConfig struct {
	BadDuration time.Duration `yaml:"bad_duration" default:"invalid_duration"`
}

type UnsupportedTypeConfig struct {
	UnsupportedField []string `yaml:"unsupported_field" default:"should_fail"`
}

func TestDefaults(t *testing.T) {
	t.Run("defaults only", func(t *testing.T) {
		var cfg TestConfig
		err := Load(&cfg)
		require.NoError(t, err)

		assert.Equal(t, "info", cfg.Logger.Level)
		assert.Equal(t, ":8080", cfg.Health.Address)
		assert.True(t, cfg.Health.Auth.Enabled)
		assert.Equal(t, "secret", cfg.Health.Auth.Secret)
		assert.Equal(t, "localhost", cfg.DB.Host)
		assert.Equal(t, 5432, cfg.DB.Port)
		assert.Equal(t, "postgres", cfg.DB.Username)
		assert.False(t, cfg.DB.SSL)
	})

	t.Run("multi level defaults with custom default", func(t *testing.T) {
		customDefault := MultiLevelConfig{
			Name: "custom-name",
			Level: LevelOneConfig{
				Value: "custom-level-one",
			},
		}

		var cfg MultiLevelConfig
		err := Load(&cfg, WithDefault(customDefault))
		require.NoError(t, err)

		assert.Equal(t, "custom-name", cfg.Name)
		assert.Equal(t, "custom-level-one", cfg.Level.Value)
		assert.Equal(t, "", cfg.Level.Level.Setting)
	})

	t.Run("custom defaults override struct defaults", func(t *testing.T) {
		customDefault := TestConfig{
			Logger: LoggerConfig{Level: "error"},
			Health: HealthConfig{Address: ":6666"},
		}

		var cfg TestConfig
		err := Load(&cfg, WithDefault(customDefault))
		require.NoError(t, err)

		assert.Equal(t, "error", cfg.Logger.Level)
		assert.Equal(t, ":6666", cfg.Health.Address)
		assert.False(t, cfg.Health.Auth.Enabled)
		assert.Equal(t, "", cfg.Health.Auth.Secret)
		assert.Equal(t, "", cfg.DB.Host)
		assert.Equal(t, 0, cfg.DB.Port)
	})

	t.Run("custom default type mismatch", func(t *testing.T) {
		type DifferentConfig struct {
			Value string `yaml:"value"`
		}

		customDefault := DifferentConfig{Value: "test"}
		var cfg TestConfig

		err := Load(&cfg, WithDefault(customDefault))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not match config type")
	})

	t.Run("custom defaults", func(t *testing.T) {
		customDefaults := ComprehensiveDurationConfig{
			FileTimeout:          1*time.Hour + 30*time.Minute,
			CustomDefaultTimeout: 5 * time.Minute,
		}

		var cfg ComprehensiveDurationConfig
		err := Load(&cfg, WithDefault(customDefaults))
		require.NoError(t, err)

		assert.Equal(t, 30*time.Second, cfg.DefaultTagTimeout)
		assert.Equal(t, 1*time.Hour+30*time.Minute, cfg.FileTimeout)
		assert.Equal(t, time.Duration(0), cfg.EnvTimeout)
		assert.Equal(t, 5*time.Minute, cfg.CustomDefaultTimeout)
		require.NotNil(t, cfg.PointerTimeout)
		assert.Equal(t, 15*time.Second, *cfg.PointerTimeout)
	})
}

func TestDefaultTag(t *testing.T) {
	t.Run("basic types", func(t *testing.T) {
		t.Run("all fields use default tags", func(t *testing.T) {
			var cfg DefaultTagConfig
			err := Load(&cfg)
			require.NoError(t, err)

			assert.Equal(t, "default_string", cfg.StringField)
			assert.Equal(t, 42, cfg.IntField)
			assert.True(t, cfg.BoolField)
			assert.Equal(t, 3.14, cfg.FloatField)
			assert.Equal(t, uint(100), cfg.UintField)
			require.NotNil(t, cfg.PointerField)
			assert.Equal(t, "default_pointer", *cfg.PointerField)
		})

		t.Run("some fields already set", func(t *testing.T) {
			cfg := DefaultTagConfig{
				StringField: "already_set",
				IntField:    99,
			}
			err := Load(&cfg)
			require.NoError(t, err)

			assert.Equal(t, "already_set", cfg.StringField)
			assert.Equal(t, 99, cfg.IntField)
			assert.True(t, cfg.BoolField)
			assert.Equal(t, 3.14, cfg.FloatField)
			assert.Equal(t, uint(100), cfg.UintField)
		})
	})

	t.Run("duration types", func(t *testing.T) {
		t.Run("all duration fields use default tags", func(t *testing.T) {
			var cfg DurationDefaultTagConfig
			err := Load(&cfg)
			require.NoError(t, err)

			assert.Equal(t, 30*time.Second, cfg.Timeout)
			assert.Equal(t, 5*time.Minute, cfg.RetryDelay)
			assert.Equal(t, 1*time.Hour, cfg.MaxWait)
			require.NotNil(t, cfg.Optional)
			assert.Equal(t, 15*time.Second, *cfg.Optional)
		})

		t.Run("invalid duration format", func(t *testing.T) {
			var cfg InvalidDurationConfig
			err := Load(&cfg)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "invalid duration default value")
			assert.Contains(t, err.Error(), "BadDuration")
		})
	})

	t.Run("invalid values", func(t *testing.T) {
		t.Run("invalid boolean", func(t *testing.T) {
			var cfg InvalidDefaultConfig
			err := Load(&cfg)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "invalid boolean default value")
		})
	})

	t.Run("unsupported type", func(t *testing.T) {
		t.Run("unsupported field type for default tag", func(t *testing.T) {
			var cfg UnsupportedTypeConfig
			err := Load(&cfg)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "unsupported field type slice for default tag")
		})
	})

	t.Run("nested structs", func(t *testing.T) {
		t.Run("nested default tags", func(t *testing.T) {
			var cfg NestedDefaultConfig
			err := Load(&cfg)
			require.NoError(t, err)

			assert.Equal(t, "parent_name", cfg.Parent.Name)
			assert.Equal(t, "child_value", cfg.Parent.Child.Value)
		})
	})

	t.Run("integer types", func(t *testing.T) {
		type IntegerConfig struct {
			Int8Field   int8   `yaml:"int8" default:"127"`
			Int16Field  int16  `yaml:"int16" default:"32767"`
			Int32Field  int32  `yaml:"int32" default:"2147483647"`
			Uint8Field  uint8  `yaml:"uint8" default:"255"`
			Uint16Field uint16 `yaml:"uint16" default:"65535"`
			Uint32Field uint32 `yaml:"uint32" default:"4294967295"`
			Uint64Field uint64 `yaml:"uint64" default:"18446744073709551615"`
		}

		var cfg IntegerConfig
		err := Load(&cfg)
		require.NoError(t, err)

		assert.Equal(t, int8(127), cfg.Int8Field)
		assert.Equal(t, int16(32767), cfg.Int16Field)
		assert.Equal(t, int32(2147483647), cfg.Int32Field)
		assert.Equal(t, uint8(255), cfg.Uint8Field)
		assert.Equal(t, uint16(65535), cfg.Uint16Field)
		assert.Equal(t, uint32(4294967295), cfg.Uint32Field)
		assert.Equal(t, uint64(18446744073709551615), cfg.Uint64Field)
	})

	t.Run("float types", func(t *testing.T) {
		type FloatConfig struct {
			Float32Field float32 `yaml:"float32" default:"3.14159"`
			Float64Field float64 `yaml:"float64" default:"2.71828"`
		}

		var cfg FloatConfig
		err := Load(&cfg)
		require.NoError(t, err)

		assert.InDelta(t, float32(3.14159), cfg.Float32Field, 0.00001)
		assert.InDelta(t, 2.71828, cfg.Float64Field, 0.00001)
	})

	t.Run("overflow detection", func(t *testing.T) {
		type OverflowConfig struct {
			SmallInt int8 `yaml:"small_int" default:"999"` // overflow
		}

		var cfg OverflowConfig
		err := Load(&cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "overflows")
	})

	t.Run("priority with files", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "default-priority-*.yaml")
		require.NoError(t, err)
		defer func() { _ = os.Remove(tmpFile.Name()) }()

		content := `string_field: "from_file"
int_field: 999`

		_, err = tmpFile.WriteString(content)
		require.NoError(t, err)
		require.NoError(t, tmpFile.Close())

		var cfg DefaultTagConfig
		err = Load(&cfg, WithFiles(tmpFile.Name()))
		require.NoError(t, err)

		// File should override default tags
		assert.Equal(t, "from_file", cfg.StringField)
		assert.Equal(t, 999, cfg.IntField)
		// Fields not in file should use default tags
		assert.True(t, cfg.BoolField)
		assert.Equal(t, 3.14, cfg.FloatField)
	})

	t.Run("priority with environment", func(t *testing.T) {
		require.NoError(t, os.Setenv("TEST_STRING_FIELD", "from_env"))
		require.NoError(t, os.Setenv("TEST_INT_FIELD", "777"))
		defer func() {
			_ = os.Unsetenv("TEST_STRING_FIELD")
			_ = os.Unsetenv("TEST_INT_FIELD")
		}()

		var cfg DefaultTagConfig
		err := Load(&cfg, WithEnv("TEST"))
		require.NoError(t, err)

		// Environment should override default tags
		assert.Equal(t, "from_env", cfg.StringField)
		assert.Equal(t, 777, cfg.IntField)
		// Fields not in env should use default tags
		assert.True(t, cfg.BoolField)
		assert.Equal(t, 3.14, cfg.FloatField)
	})
}
