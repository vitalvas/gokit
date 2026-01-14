package xconfig

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDotenv(t *testing.T) {
	t.Run("default dotenv file", func(t *testing.T) {
		originalDir, err := os.Getwd()
		require.NoError(t, err)

		tmpDir, err := os.MkdirTemp("", "dotenv-test-*")
		require.NoError(t, err)
		defer func() { _ = os.RemoveAll(tmpDir) }()

		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(originalDir) }()

		content := `APP_DB_HOST=default_host
APP_DB_PORT=5432`

		err = os.WriteFile(".env", []byte(content), 0o600)
		require.NoError(t, err)

		defer func() {
			_ = os.Unsetenv("APP_DB_HOST")
			_ = os.Unsetenv("APP_DB_PORT")
		}()

		var cfg TestConfig
		err = Load(&cfg, WithDotenv(), WithEnv("APP"))
		require.NoError(t, err)

		assert.Equal(t, "default_host", cfg.DB.Host)
		assert.Equal(t, 5432, cfg.DB.Port)
	})

	t.Run("basic dotenv loading", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "test-*.env")
		require.NoError(t, err)
		defer func() { _ = os.Remove(tmpFile.Name()) }()

		content := `APP_DB_HOST=localhost
APP_DB_PORT=5432
APP_DB_USERNAME=testuser
APP_DB_PASSWORD=testpass`

		_, err = tmpFile.WriteString(content)
		require.NoError(t, err)
		require.NoError(t, tmpFile.Close())

		defer func() {
			_ = os.Unsetenv("APP_DB_HOST")
			_ = os.Unsetenv("APP_DB_PORT")
			_ = os.Unsetenv("APP_DB_USERNAME")
			_ = os.Unsetenv("APP_DB_PASSWORD")
		}()

		var cfg TestConfig
		err = Load(&cfg, WithDotenv(tmpFile.Name()), WithEnv("APP"))
		require.NoError(t, err)

		assert.Equal(t, "localhost", cfg.DB.Host)
		assert.Equal(t, 5432, cfg.DB.Port)
		assert.Equal(t, "testuser", cfg.DB.Username)
	})

	t.Run("dotenv with comments and empty lines", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "test-*.env")
		require.NoError(t, err)
		defer func() { _ = os.Remove(tmpFile.Name()) }()

		content := `# Database configuration
APP_DB_HOST=dbserver

# Port number
APP_DB_PORT=3306

# Empty lines above and below are ignored

APP_DB_USERNAME=admin
# Inline comment after value not supported in basic format`

		_, err = tmpFile.WriteString(content)
		require.NoError(t, err)
		require.NoError(t, tmpFile.Close())

		defer func() {
			_ = os.Unsetenv("APP_DB_HOST")
			_ = os.Unsetenv("APP_DB_PORT")
			_ = os.Unsetenv("APP_DB_USERNAME")
		}()

		var cfg TestConfig
		err = Load(&cfg, WithDotenv(tmpFile.Name()), WithEnv("APP"))
		require.NoError(t, err)

		assert.Equal(t, "dbserver", cfg.DB.Host)
		assert.Equal(t, 3306, cfg.DB.Port)
		assert.Equal(t, "admin", cfg.DB.Username)
	})

	t.Run("dotenv overrides existing environment variables", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "test-*.env")
		require.NoError(t, err)
		defer func() { _ = os.Remove(tmpFile.Name()) }()

		content := `TEST_VAR=dotenv_value`

		_, err = tmpFile.WriteString(content)
		require.NoError(t, err)
		require.NoError(t, tmpFile.Close())

		require.NoError(t, os.Setenv("TEST_VAR", "original_value"))
		defer func() { _ = os.Unsetenv("TEST_VAR") }()

		err = loadDotenvFiles([]string{tmpFile.Name()})
		require.NoError(t, err)

		assert.Equal(t, "dotenv_value", os.Getenv("TEST_VAR"))
	})

	t.Run("multiple dotenv files", func(t *testing.T) {
		tmpFile1, err := os.CreateTemp("", "test1-*.env")
		require.NoError(t, err)
		defer func() { _ = os.Remove(tmpFile1.Name()) }()

		content1 := `APP_DB_HOST=localhost
APP_DB_PORT=5432`

		_, err = tmpFile1.WriteString(content1)
		require.NoError(t, err)
		require.NoError(t, tmpFile1.Close())

		tmpFile2, err := os.CreateTemp("", "test2-*.env")
		require.NoError(t, err)
		defer func() { _ = os.Remove(tmpFile2.Name()) }()

		content2 := `APP_DB_PORT=3306
APP_DB_USERNAME=admin`

		_, err = tmpFile2.WriteString(content2)
		require.NoError(t, err)
		require.NoError(t, tmpFile2.Close())

		defer func() {
			_ = os.Unsetenv("APP_DB_HOST")
			_ = os.Unsetenv("APP_DB_PORT")
			_ = os.Unsetenv("APP_DB_USERNAME")
		}()

		var cfg TestConfig
		err = Load(&cfg, WithDotenv(tmpFile1.Name(), tmpFile2.Name()), WithEnv("APP"))
		require.NoError(t, err)

		assert.Equal(t, "localhost", cfg.DB.Host)
		assert.Equal(t, 3306, cfg.DB.Port)
		assert.Equal(t, "admin", cfg.DB.Username)
	})

	t.Run("dotenv priority: dotenv < files < env", func(t *testing.T) {
		dotenvFile, err := os.CreateTemp("", "test-*.env")
		require.NoError(t, err)
		defer func() { _ = os.Remove(dotenvFile.Name()) }()

		dotenvContent := `APP_LOGGER_LEVEL=dotenv_level
APP_HEALTH_ADDRESS=:7777
APP_DB_HOST=dotenv_host`

		_, err = dotenvFile.WriteString(dotenvContent)
		require.NoError(t, err)
		require.NoError(t, dotenvFile.Close())

		configFile, err := os.CreateTemp("", "config-*.yaml")
		require.NoError(t, err)
		defer func() { _ = os.Remove(configFile.Name()) }()

		configContent := `logger:
  level: "file_level"
db:
  host: "file_host"`

		_, err = configFile.WriteString(configContent)
		require.NoError(t, err)
		require.NoError(t, configFile.Close())

		require.NoError(t, os.Setenv("APP_LOGGER_LEVEL", "env_level"))
		defer func() {
			_ = os.Unsetenv("APP_LOGGER_LEVEL")
			_ = os.Unsetenv("APP_HEALTH_ADDRESS")
			_ = os.Unsetenv("APP_DB_HOST")
		}()

		var cfg TestConfig
		err = Load(&cfg,
			WithDotenv(dotenvFile.Name()),
			WithFiles(configFile.Name()),
			WithEnv("APP"))
		require.NoError(t, err)

		assert.Equal(t, "dotenv_level", cfg.Logger.Level)
		assert.Equal(t, "dotenv_host", cfg.DB.Host)
		assert.Equal(t, ":7777", cfg.Health.Address)
	})

	t.Run("invalid dotenv format: missing equals", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "test-*.env")
		require.NoError(t, err)
		defer func() { _ = os.Remove(tmpFile.Name()) }()

		content := `VALID_KEY=valid_value
INVALID_LINE_NO_EQUALS
ANOTHER_VALID=value`

		_, err = tmpFile.WriteString(content)
		require.NoError(t, err)
		require.NoError(t, tmpFile.Close())

		var cfg TestConfig
		err = Load(&cfg, WithDotenv(tmpFile.Name()))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid format")
		assert.Contains(t, err.Error(), "missing '='")
	})

	t.Run("invalid dotenv format: empty key", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "test-*.env")
		require.NoError(t, err)
		defer func() { _ = os.Remove(tmpFile.Name()) }()

		content := `=value_without_key`

		_, err = tmpFile.WriteString(content)
		require.NoError(t, err)
		require.NoError(t, tmpFile.Close())

		var cfg TestConfig
		err = Load(&cfg, WithDotenv(tmpFile.Name()))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty key")
	})

	t.Run("dotenv file does not exist", func(t *testing.T) {
		var cfg TestConfig
		err := Load(&cfg, WithDotenv("/nonexistent/path/to/.env"))
		require.NoError(t, err)
	})

	t.Run("dotenv with whitespace handling", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "test-*.env")
		require.NoError(t, err)
		defer func() { _ = os.Remove(tmpFile.Name()) }()

		content := `  SPACED_KEY  =  spaced_value
NORMAL_KEY=normal_value`

		_, err = tmpFile.WriteString(content)
		require.NoError(t, err)
		require.NoError(t, tmpFile.Close())

		defer func() {
			_ = os.Unsetenv("SPACED_KEY")
			_ = os.Unsetenv("NORMAL_KEY")
		}()

		err = loadDotenvFiles([]string{tmpFile.Name()})
		require.NoError(t, err)

		assert.Equal(t, "spaced_value", os.Getenv("SPACED_KEY"))
		assert.Equal(t, "normal_value", os.Getenv("NORMAL_KEY"))
	})

	t.Run("dotenv with complex values", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "test-*.env")
		require.NoError(t, err)
		defer func() { _ = os.Remove(tmpFile.Name()) }()

		content := `DATABASE_URL=postgres://user:pass@localhost:5432/db
API_KEY=abc-123-def-456
NUMBERS=123
BOOLEAN=true
SPECIAL_CHARS=!@#$%^&*()_+-=[]{}|;:,.<>?`

		_, err = tmpFile.WriteString(content)
		require.NoError(t, err)
		require.NoError(t, tmpFile.Close())

		defer func() {
			_ = os.Unsetenv("DATABASE_URL")
			_ = os.Unsetenv("API_KEY")
			_ = os.Unsetenv("NUMBERS")
			_ = os.Unsetenv("BOOLEAN")
			_ = os.Unsetenv("SPECIAL_CHARS")
		}()

		err = loadDotenvFiles([]string{tmpFile.Name()})
		require.NoError(t, err)

		assert.Equal(t, "postgres://user:pass@localhost:5432/db", os.Getenv("DATABASE_URL"))
		assert.Equal(t, "abc-123-def-456", os.Getenv("API_KEY"))
		assert.Equal(t, "123", os.Getenv("NUMBERS"))
		assert.Equal(t, "true", os.Getenv("BOOLEAN"))
		assert.Equal(t, "!@#$%^&*()_+-=[]{}|;:,.<>?", os.Getenv("SPECIAL_CHARS"))
	})

	t.Run("dotenv with empty value", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "test-*.env")
		require.NoError(t, err)
		defer func() { _ = os.Remove(tmpFile.Name()) }()

		content := `EMPTY_KEY=
NORMAL_KEY=normal_value`

		_, err = tmpFile.WriteString(content)
		require.NoError(t, err)
		require.NoError(t, tmpFile.Close())

		defer func() {
			_ = os.Unsetenv("EMPTY_KEY")
			_ = os.Unsetenv("NORMAL_KEY")
		}()

		err = loadDotenvFiles([]string{tmpFile.Name()})
		require.NoError(t, err)

		assert.Equal(t, "", os.Getenv("EMPTY_KEY"))
		assert.Equal(t, "normal_value", os.Getenv("NORMAL_KEY"))
	})
}
