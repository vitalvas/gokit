package xconfig

import (
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadFromEnv(t *testing.T) {
	t.Run("basic environment variables", func(t *testing.T) {
		envVars := map[string]string{
			"TEST_LOGGER_LEVEL":   "error",
			"TEST_HEALTH_ADDRESS": ":3000",
			"TEST_DB_HOST":        "envhost",
			"TEST_DB_PORT":        "5433",
		}

		for key, value := range envVars {
			require.NoError(t, os.Setenv(key, value))
		}
		defer func() {
			for key := range envVars {
				_ = os.Unsetenv(key)
			}
		}()

		var cfg TestConfig
		err := Load(&cfg, WithEnv("TEST"))
		require.NoError(t, err)

		assert.Equal(t, "error", cfg.Logger.Level)
		assert.Equal(t, ":3000", cfg.Health.Address)
		assert.Equal(t, "envhost", cfg.DB.Host)
		assert.Equal(t, 5433, cfg.DB.Port)
	})

	t.Run("environment variables without prefix", func(t *testing.T) {
		type SimpleConfig struct {
			Host string `yaml:"host"`
			Port int    `yaml:"port"`
		}

		envVars := map[string]string{
			"HOST": "localhost",
			"PORT": "8080",
		}

		for key, value := range envVars {
			require.NoError(t, os.Setenv(key, value))
		}
		defer func() {
			for key := range envVars {
				_ = os.Unsetenv(key)
			}
		}()

		var cfg SimpleConfig
		err := Load(&cfg, WithEnv(EnvSkipPrefix))
		require.NoError(t, err)

		assert.Equal(t, "localhost", cfg.Host)
		assert.Equal(t, 8080, cfg.Port)
	})

	t.Run("slices and maps from environment", func(t *testing.T) {
		type ConfigWithCollections struct {
			Hosts  []string          `yaml:"hosts"`
			Ports  []int             `yaml:"ports"`
			Labels map[string]string `yaml:"labels"`
		}

		envVars := map[string]string{
			"TEST_HOSTS":  "host1,host2,host3",
			"TEST_PORTS":  "8080,8081,8082",
			"TEST_LABELS": "env=prod,region=us-east",
		}

		for key, value := range envVars {
			require.NoError(t, os.Setenv(key, value))
		}
		defer func() {
			for key := range envVars {
				_ = os.Unsetenv(key)
			}
		}()

		var cfg ConfigWithCollections
		err := Load(&cfg, WithEnv("TEST"))
		require.NoError(t, err)

		assert.Equal(t, []string{"host1", "host2", "host3"}, cfg.Hosts)
		assert.Equal(t, []int{8080, 8081, 8082}, cfg.Ports)
		assert.Equal(t, map[string]string{"env": "prod", "region": "us-east"}, cfg.Labels)
	})

	t.Run("duration from environment", func(t *testing.T) {
		type ConfigWithDuration struct {
			Timeout time.Duration `yaml:"timeout"`
		}

		require.NoError(t, os.Setenv("TEST_TIMEOUT", "30s"))
		defer func() { _ = os.Unsetenv("TEST_TIMEOUT") }()

		var cfg ConfigWithDuration
		err := Load(&cfg, WithEnv("TEST"))
		require.NoError(t, err)

		assert.Equal(t, 30*time.Second, cfg.Timeout)
	})
}

func TestEnvTagSupport(t *testing.T) {
	t.Run("env tag takes precedence", func(t *testing.T) {
		type ConfigWithEnvTag struct {
			APIKey string `env:"CUSTOM_API_KEY" yaml:"api_key"`
		}

		require.NoError(t, os.Setenv("TEST_CUSTOM_API_KEY", "secret123"))
		defer func() { _ = os.Unsetenv("TEST_CUSTOM_API_KEY") }()

		var cfg ConfigWithEnvTag
		err := Load(&cfg, WithEnv("TEST"))
		require.NoError(t, err)

		assert.Equal(t, "secret123", cfg.APIKey)
	})

	t.Run("env tag with special characters", func(t *testing.T) {
		type SpecialEnvConfig struct {
			DatabaseURL string `env:"DB_URL" yaml:"database_url"`
			APIToken    string `env:"API_TOKEN" yaml:"api_token"`
		}

		envVars := map[string]string{
			"TEST_DB_URL":    "postgres://localhost:5432/db",
			"TEST_API_TOKEN": "token-123-abc",
		}

		for key, value := range envVars {
			require.NoError(t, os.Setenv(key, value))
		}
		defer func() {
			for key := range envVars {
				_ = os.Unsetenv(key)
			}
		}()

		var cfg SpecialEnvConfig
		err := Load(&cfg, WithEnv("TEST"))
		require.NoError(t, err)

		assert.Equal(t, "postgres://localhost:5432/db", cfg.DatabaseURL)
		assert.Equal(t, "token-123-abc", cfg.APIToken)
	})
}

func TestCamelToSnake(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"SimpleField", "simple_field"},
		{"HTTPServer", "http_server"},
		{"APIKey", "api_key"},
		{"DBHost", "db_host"},
		{"URLPath", "url_path"},
		{"ID", "id"},
		{"MyHTTPSConnection", "my_https_connection"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := camelToSnake(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseCommaSeparated(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"simple values", "a,b,c", []string{"a", "b", "c"}},
		{"with spaces", " a , b , c ", []string{"a", "b", "c"}},
		{"empty string", "", nil},
		{"single value", "single", []string{"single"}},
		{"trailing comma", "a,b,", []string{"a", "b"}},
		{"multiple spaces", "a  ,  b  ,  c", []string{"a", "b", "c"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseCommaSeparated(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSetValueFromString(t *testing.T) {
	t.Run("string value", func(t *testing.T) {
		var s string
		v := reflect.ValueOf(&s).Elem()
		err := setValueFromString(v, "hello", "TEST_KEY", "")
		require.NoError(t, err)
		assert.Equal(t, "hello", s)
	})

	t.Run("boolean value", func(t *testing.T) {
		var b bool
		v := reflect.ValueOf(&b).Elem()
		err := setValueFromString(v, "true", "TEST_KEY", "")
		require.NoError(t, err)
		assert.True(t, b)
	})

	t.Run("integer value", func(t *testing.T) {
		var i int
		v := reflect.ValueOf(&i).Elem()
		err := setValueFromString(v, "42", "TEST_KEY", "")
		require.NoError(t, err)
		assert.Equal(t, 42, i)
	})

	t.Run("float value", func(t *testing.T) {
		var f float64
		v := reflect.ValueOf(&f).Elem()
		err := setValueFromString(v, "3.14", "TEST_KEY", "")
		require.NoError(t, err)
		assert.Equal(t, 3.14, f)
	})

	t.Run("invalid boolean", func(t *testing.T) {
		var b bool
		v := reflect.ValueOf(&b).Elem()
		err := setValueFromString(v, "not-a-bool", "TEST_KEY", "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid boolean value")
	})

	t.Run("invalid integer", func(t *testing.T) {
		var i int
		v := reflect.ValueOf(&i).Elem()
		err := setValueFromString(v, "not-an-int", "TEST_KEY", "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid integer value")
	})
}

func TestInvalidEnvironmentValues(t *testing.T) {
	t.Run("invalid slice values", func(t *testing.T) {
		type ConfigWithIntSlice struct {
			Ports []int `yaml:"ports"`
		}

		require.NoError(t, os.Setenv("TEST_PORTS", "8080,invalid,8082"))
		defer func() { _ = os.Unsetenv("TEST_PORTS") }()

		var cfg ConfigWithIntSlice
		err := Load(&cfg, WithEnv("TEST"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid integer value")
	})

	t.Run("invalid map values", func(t *testing.T) {
		type ConfigWithIntMap struct {
			Counters map[string]int `yaml:"counters"`
		}

		require.NoError(t, os.Setenv("TEST_COUNTERS", "a=1,b=invalid"))
		defer func() { _ = os.Unsetenv("TEST_COUNTERS") }()

		var cfg ConfigWithIntMap
		err := Load(&cfg, WithEnv("TEST"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid integer value")
	})

	t.Run("invalid duration", func(t *testing.T) {
		type ConfigWithDuration struct {
			Timeout time.Duration `yaml:"timeout"`
		}

		require.NoError(t, os.Setenv("TEST_TIMEOUT", "invalid-duration"))
		defer func() { _ = os.Unsetenv("TEST_TIMEOUT") }()

		var cfg ConfigWithDuration
		err := Load(&cfg, WithEnv("TEST"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid duration value")
	})
}

func TestEnvSeparator(t *testing.T) {
	t.Run("slice with custom separator - colon", func(t *testing.T) {
		type ConfigWithColonSeparator struct {
			Hosts []string `yaml:"hosts" envSeparator:":"`
		}

		require.NoError(t, os.Setenv("TEST_HOSTS", "host1:host2:host3"))
		defer func() { _ = os.Unsetenv("TEST_HOSTS") }()

		var cfg ConfigWithColonSeparator
		err := Load(&cfg, WithEnv("TEST"))
		require.NoError(t, err)

		assert.Equal(t, []string{"host1", "host2", "host3"}, cfg.Hosts)
	})

	t.Run("slice with custom separator - pipe", func(t *testing.T) {
		type ConfigWithPipeSeparator struct {
			Tags []string `yaml:"tags" envSeparator:"|"`
		}

		require.NoError(t, os.Setenv("TEST_TAGS", "tag1|tag2|tag3"))
		defer func() { _ = os.Unsetenv("TEST_TAGS") }()

		var cfg ConfigWithPipeSeparator
		err := Load(&cfg, WithEnv("TEST"))
		require.NoError(t, err)

		assert.Equal(t, []string{"tag1", "tag2", "tag3"}, cfg.Tags)
	})

	t.Run("slice with custom separator - semicolon", func(t *testing.T) {
		type ConfigWithSemicolonSeparator struct {
			Ports []int `yaml:"ports" envSeparator:";"`
		}

		require.NoError(t, os.Setenv("TEST_PORTS", "8080;8081;8082"))
		defer func() { _ = os.Unsetenv("TEST_PORTS") }()

		var cfg ConfigWithSemicolonSeparator
		err := Load(&cfg, WithEnv("TEST"))
		require.NoError(t, err)

		assert.Equal(t, []int{8080, 8081, 8082}, cfg.Ports)
	})

	t.Run("slice with default separator - comma", func(t *testing.T) {
		type ConfigWithDefaultSeparator struct {
			Hosts []string `yaml:"hosts"`
		}

		require.NoError(t, os.Setenv("TEST_HOSTS", "host1,host2,host3"))
		defer func() { _ = os.Unsetenv("TEST_HOSTS") }()

		var cfg ConfigWithDefaultSeparator
		err := Load(&cfg, WithEnv("TEST"))
		require.NoError(t, err)

		assert.Equal(t, []string{"host1", "host2", "host3"}, cfg.Hosts)
	})

	t.Run("map with custom separator - pipe", func(t *testing.T) {
		type ConfigWithPipeMapSeparator struct {
			Labels map[string]string `yaml:"labels" envSeparator:"|"`
		}

		require.NoError(t, os.Setenv("TEST_LABELS", "env=prod|region=us-east|tier=frontend"))
		defer func() { _ = os.Unsetenv("TEST_LABELS") }()

		var cfg ConfigWithPipeMapSeparator
		err := Load(&cfg, WithEnv("TEST"))
		require.NoError(t, err)

		assert.Equal(t, map[string]string{
			"env":    "prod",
			"region": "us-east",
			"tier":   "frontend",
		}, cfg.Labels)
	})

	t.Run("map with custom separator - semicolon", func(t *testing.T) {
		type ConfigWithSemicolonMapSeparator struct {
			Counters map[string]int `yaml:"counters" envSeparator:";"`
		}

		require.NoError(t, os.Setenv("TEST_COUNTERS", "a=1;b=2;c=3"))
		defer func() { _ = os.Unsetenv("TEST_COUNTERS") }()

		var cfg ConfigWithSemicolonMapSeparator
		err := Load(&cfg, WithEnv("TEST"))
		require.NoError(t, err)

		assert.Equal(t, map[string]int{
			"a": 1,
			"b": 2,
			"c": 3,
		}, cfg.Counters)
	})

	t.Run("map with default separator - comma", func(t *testing.T) {
		type ConfigWithDefaultMapSeparator struct {
			Labels map[string]string `yaml:"labels"`
		}

		require.NoError(t, os.Setenv("TEST_LABELS", "env=prod,region=us-east"))
		defer func() { _ = os.Unsetenv("TEST_LABELS") }()

		var cfg ConfigWithDefaultMapSeparator
		err := Load(&cfg, WithEnv("TEST"))
		require.NoError(t, err)

		assert.Equal(t, map[string]string{
			"env":    "prod",
			"region": "us-east",
		}, cfg.Labels)
	})

	t.Run("multiple fields with different separators", func(t *testing.T) {
		type ConfigWithMixedSeparators struct {
			Hosts    []string          `yaml:"hosts" envSeparator:":"`
			Ports    []int             `yaml:"ports" envSeparator:";"`
			Tags     []string          `yaml:"tags" envSeparator:"|"`
			Labels   map[string]string `yaml:"labels" envSeparator:":"`
			Counters map[string]int    `yaml:"counters"`
		}

		envVars := map[string]string{
			"TEST_HOSTS":    "host1:host2:host3",
			"TEST_PORTS":    "8080;8081;8082",
			"TEST_TAGS":     "tag1|tag2|tag3",
			"TEST_LABELS":   "env=prod:region=us",
			"TEST_COUNTERS": "a=1,b=2",
		}

		for key, value := range envVars {
			require.NoError(t, os.Setenv(key, value))
		}
		defer func() {
			for key := range envVars {
				_ = os.Unsetenv(key)
			}
		}()

		var cfg ConfigWithMixedSeparators
		err := Load(&cfg, WithEnv("TEST"))
		require.NoError(t, err)

		assert.Equal(t, []string{"host1", "host2", "host3"}, cfg.Hosts)
		assert.Equal(t, []int{8080, 8081, 8082}, cfg.Ports)
		assert.Equal(t, []string{"tag1", "tag2", "tag3"}, cfg.Tags)
		assert.Equal(t, map[string]string{"env": "prod", "region": "us"}, cfg.Labels)
		assert.Equal(t, map[string]int{"a": 1, "b": 2}, cfg.Counters)
	})

	t.Run("slice with spaces around custom separator", func(t *testing.T) {
		type ConfigWithSpacedSeparator struct {
			Items []string `yaml:"items" envSeparator:"|"`
		}

		require.NoError(t, os.Setenv("TEST_ITEMS", "item1 | item2 | item3"))
		defer func() { _ = os.Unsetenv("TEST_ITEMS") }()

		var cfg ConfigWithSpacedSeparator
		err := Load(&cfg, WithEnv("TEST"))
		require.NoError(t, err)

		assert.Equal(t, []string{"item1", "item2", "item3"}, cfg.Items)
	})
}

func TestParseSeparated(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		separator string
		expected  []string
	}{
		{"comma separated", "a,b,c", ",", []string{"a", "b", "c"}},
		{"colon separated", "a:b:c", ":", []string{"a", "b", "c"}},
		{"pipe separated", "a|b|c", "|", []string{"a", "b", "c"}},
		{"semicolon separated", "a;b;c", ";", []string{"a", "b", "c"}},
		{"with spaces", " a | b | c ", "|", []string{"a", "b", "c"}},
		{"empty string", "", ",", nil},
		{"single value", "single", ",", []string{"single"}},
		{"trailing separator", "a|b|", "|", []string{"a", "b"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseSeparated(tt.input, tt.separator)
			assert.Equal(t, tt.expected, result)
		})
	}
}
