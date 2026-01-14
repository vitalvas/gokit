package xconfig

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadFromFile(t *testing.T) {
	t.Run("YAML file", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "config-*.yaml")
		require.NoError(t, err)
		defer func() { _ = os.Remove(tmpFile.Name()) }()

		content := `logger:
  level: "debug"
health:
  address: ":9090"
db:
  host: "testhost"
  port: 3306`

		_, err = tmpFile.WriteString(content)
		require.NoError(t, err)
		require.NoError(t, tmpFile.Close())

		var cfg TestConfig
		err = Load(&cfg, WithFiles(tmpFile.Name()))
		require.NoError(t, err)

		assert.Equal(t, "debug", cfg.Logger.Level)
		assert.Equal(t, ":9090", cfg.Health.Address)
		assert.Equal(t, "testhost", cfg.DB.Host)
		assert.Equal(t, 3306, cfg.DB.Port)
	})

	t.Run("JSON file", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "config-*.json")
		require.NoError(t, err)
		defer func() { _ = os.Remove(tmpFile.Name()) }()

		content := `{
  "logger": {
    "level": "error"
  },
  "health": {
    "address": ":3000"
  },
  "db": {
    "host": "jsonhost",
    "port": 5433
  }
}`

		_, err = tmpFile.WriteString(content)
		require.NoError(t, err)
		require.NoError(t, tmpFile.Close())

		var cfg TestConfig
		err = Load(&cfg, WithFiles(tmpFile.Name()))
		require.NoError(t, err)

		assert.Equal(t, "error", cfg.Logger.Level)
		assert.Equal(t, ":3000", cfg.Health.Address)
		assert.Equal(t, "jsonhost", cfg.DB.Host)
		assert.Equal(t, 5433, cfg.DB.Port)
	})

	t.Run("nonexistent file", func(t *testing.T) {
		var cfg TestConfig
		err := loadFromFile(&cfg, "/nonexistent/path/config.yaml", false)
		require.NoError(t, err)
	})

	t.Run("unsupported file extension", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "config-*.txt")
		require.NoError(t, err)
		defer func() { _ = os.Remove(tmpFile.Name()) }()

		_, err = tmpFile.WriteString("some content")
		require.NoError(t, err)
		require.NoError(t, tmpFile.Close())

		var cfg TestConfig
		err = loadFromFile(&cfg, tmpFile.Name(), false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported file extension")
	})
}

func TestLoadFromFiles(t *testing.T) {
	t.Run("multiple files", func(t *testing.T) {
		tmpFile1, err := os.CreateTemp("", "config1-*.yaml")
		require.NoError(t, err)
		defer func() { _ = os.Remove(tmpFile1.Name()) }()

		content1 := `logger:
  level: "debug"
health:
  address: ":9090"`

		_, err = tmpFile1.WriteString(content1)
		require.NoError(t, err)
		require.NoError(t, tmpFile1.Close())

		tmpFile2, err := os.CreateTemp("", "config2-*.yaml")
		require.NoError(t, err)
		defer func() { _ = os.Remove(tmpFile2.Name()) }()

		content2 := `logger:
  level: "error"
db:
  host: "testhost"
  port: 3306`

		_, err = tmpFile2.WriteString(content2)
		require.NoError(t, err)
		require.NoError(t, tmpFile2.Close())

		var cfg TestConfig
		err = Load(&cfg, WithFiles(tmpFile1.Name(), tmpFile2.Name()))
		require.NoError(t, err)

		assert.Equal(t, "error", cfg.Logger.Level)
		assert.Equal(t, ":9090", cfg.Health.Address)
		assert.Equal(t, "testhost", cfg.DB.Host)
		assert.Equal(t, 3306, cfg.DB.Port)
	})
}

func TestIsConfigFile(t *testing.T) {
	tests := []struct {
		filename string
		expected bool
	}{
		{"config.yaml", true},
		{"config.yml", true},
		{"config.json", true},
		{"config.YAML", true},
		{"config.JSON", true},
		{"config.txt", false},
		{"config.xml", false},
		{"config", false},
		{"readme.md", false},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := isConfigFile(tt.filename)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestUnmarshalFormats(t *testing.T) {
	type unmarshalFunc func([]byte, interface{}, bool) error
	type formatTestData struct {
		valid         string
		invalid       string
		unknownField  string
		nestedUnknown string
		empty         string
		expectedLevel string
		expectedHost  string
		expectedPort  int
	}

	formats := []struct {
		name      string
		unmarshal unmarshalFunc
		data      formatTestData
	}{
		{
			name:      "YAML",
			unmarshal: unmarshalYAML,
			data: formatTestData{
				valid:         "logger:\n  level: \"info\"\ndb:\n  host: \"localhost\"\n  port: 5432",
				invalid:       "invalid: yaml: syntax:",
				unknownField:  "logger:\n  level: \"info\"\nunknown_field: \"value\"",
				nestedUnknown: "logger:\n  level: \"info\"\n  unknown_nested: \"value\"",
				empty:         "{}",
				expectedLevel: "info",
				expectedHost:  "localhost",
				expectedPort:  5432,
			},
		},
		{
			name:      "JSON",
			unmarshal: unmarshalJSON,
			data: formatTestData{
				valid:         `{"logger":{"level":"info"},"db":{"host":"localhost","port":5432}}`,
				invalid:       `{invalid json}`,
				unknownField:  `{"logger":{"level":"info"},"unknown_field":"value"}`,
				nestedUnknown: `{"logger":{"level":"info","unknown_nested":"value"}}`,
				empty:         `{}`,
				expectedLevel: "info",
				expectedHost:  "localhost",
				expectedPort:  5432,
			},
		},
	}

	for _, f := range formats {
		t.Run(f.name, func(t *testing.T) {
			t.Run("valid", func(t *testing.T) {
				var cfg TestConfig
				err := f.unmarshal([]byte(f.data.valid), &cfg, false)
				require.NoError(t, err)
				assert.Equal(t, f.data.expectedLevel, cfg.Logger.Level)
				assert.Equal(t, f.data.expectedHost, cfg.DB.Host)
				assert.Equal(t, f.data.expectedPort, cfg.DB.Port)
			})

			t.Run("invalid syntax", func(t *testing.T) {
				var cfg TestConfig
				err := f.unmarshal([]byte(f.data.invalid), &cfg, false)
				require.Error(t, err)
			})

			t.Run("strict mode with unknown field", func(t *testing.T) {
				var cfg TestConfig
				err := f.unmarshal([]byte(f.data.unknownField), &cfg, true)
				require.Error(t, err)
				assert.Contains(t, err.Error(), "unknown")
			})

			t.Run("non-strict mode with unknown field", func(t *testing.T) {
				var cfg TestConfig
				err := f.unmarshal([]byte(f.data.unknownField), &cfg, false)
				require.NoError(t, err)
				assert.Equal(t, f.data.expectedLevel, cfg.Logger.Level)
			})

			t.Run("nested struct unknown field in strict mode", func(t *testing.T) {
				var cfg TestConfig
				err := f.unmarshal([]byte(f.data.nestedUnknown), &cfg, true)
				require.Error(t, err)
			})

			t.Run("empty document", func(t *testing.T) {
				var cfg TestConfig
				err := f.unmarshal([]byte(f.data.empty), &cfg, false)
				require.NoError(t, err)
			})
		})
	}
}

func TestStrictMode(t *testing.T) {
	t.Run("YAML strict mode rejects unknown fields", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "config-*.yaml")
		require.NoError(t, err)
		defer func() { _ = os.Remove(tmpFile.Name()) }()

		content := `logger:
  level: "debug"
unknown_field: "should fail"`

		_, err = tmpFile.WriteString(content)
		require.NoError(t, err)
		require.NoError(t, tmpFile.Close())

		var cfg TestConfig
		err = Load(&cfg, WithFiles(tmpFile.Name()), WithStrict(true))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown")
	})

	t.Run("YAML non-strict mode ignores unknown fields", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "config-*.yaml")
		require.NoError(t, err)
		defer func() { _ = os.Remove(tmpFile.Name()) }()

		content := `logger:
  level: "debug"
unknown_field: "should be ignored"`

		_, err = tmpFile.WriteString(content)
		require.NoError(t, err)
		require.NoError(t, tmpFile.Close())

		var cfg TestConfig
		err = Load(&cfg, WithFiles(tmpFile.Name()), WithStrict(false))
		require.NoError(t, err)
		assert.Equal(t, "debug", cfg.Logger.Level)
	})

	t.Run("JSON strict mode rejects unknown fields", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "config-*.json")
		require.NoError(t, err)
		defer func() { _ = os.Remove(tmpFile.Name()) }()

		content := `{
  "logger": {"level": "error"},
  "unknown_field": "should fail"
}`

		_, err = tmpFile.WriteString(content)
		require.NoError(t, err)
		require.NoError(t, tmpFile.Close())

		var cfg TestConfig
		err = Load(&cfg, WithFiles(tmpFile.Name()), WithStrict(true))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown")
	})

	t.Run("JSON non-strict mode ignores unknown fields", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "config-*.json")
		require.NoError(t, err)
		defer func() { _ = os.Remove(tmpFile.Name()) }()

		content := `{
  "logger": {"level": "error"},
  "unknown_field": "should be ignored"
}`

		_, err = tmpFile.WriteString(content)
		require.NoError(t, err)
		require.NoError(t, tmpFile.Close())

		var cfg TestConfig
		err = Load(&cfg, WithFiles(tmpFile.Name()), WithStrict(false))
		require.NoError(t, err)
		assert.Equal(t, "error", cfg.Logger.Level)
	})
}
