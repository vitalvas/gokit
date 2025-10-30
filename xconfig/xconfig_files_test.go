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
		err := loadFromFile(&cfg, "/nonexistent/path/config.yaml")
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
		err = loadFromFile(&cfg, tmpFile.Name())
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
