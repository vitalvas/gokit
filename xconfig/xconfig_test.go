package xconfig

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestConfig struct {
	Logger LoggerConfig `yaml:"logger" json:"logger"`
	Health HealthConfig `yaml:"health" json:"health"`
	DB     DBConfig     `yaml:"db" json:"db"`
}

type LoggerConfig struct {
	Level string `yaml:"level" json:"level"`
}

func (c *LoggerConfig) Default() {
	*c = LoggerConfig{
		Level: "info",
	}
}

type HealthConfig struct {
	Address string           `yaml:"address" json:"address"`
	Auth    HealthAuthConfig `yaml:"auth" json:"auth"`
}

func (c *HealthConfig) Default() {
	*c = HealthConfig{
		Address: ":9999",
	}
}

type HealthAuthConfig struct {
	Enabled bool   `yaml:"enabled" json:"enabled"`
	Secret  string `yaml:"secret" json:"secret"`
}

func (c *HealthAuthConfig) Default() {
	*c = HealthAuthConfig{
		Enabled: false,
		Secret:  "",
	}
}

type DBConfig struct {
	Host     string `yaml:"host" json:"host"`
	Port     int    `yaml:"port" json:"port"`
	Username string `yaml:"username" json:"username"`
	SSL      bool   `yaml:"ssl" json:"ssl"`
}

func (c *DBConfig) Default() {
	*c = DBConfig{
		Host:     "localhost",
		Port:     5432,
		Username: "postgres",
		SSL:      false,
	}
}

func assertExpectedTestConfig(t *testing.T, cfg TestConfig) {
	assert.Equal(t, "debug", cfg.Logger.Level)
	assert.Equal(t, ":8080", cfg.Health.Address)
	assert.True(t, cfg.Health.Auth.Enabled)
	assert.Equal(t, "mysecret", cfg.Health.Auth.Secret)
	assert.Equal(t, "remote-host", cfg.DB.Host)
	assert.Equal(t, 3306, cfg.DB.Port)
	assert.Equal(t, "admin", cfg.DB.Username)
	assert.True(t, cfg.DB.SSL)
}

func TestLoad_DefaultsOnly(t *testing.T) {
	var cfg TestConfig

	err := Load(&cfg)
	require.NoError(t, err)

	assert.Equal(t, "info", cfg.Logger.Level)
	assert.Equal(t, ":9999", cfg.Health.Address)
	assert.False(t, cfg.Health.Auth.Enabled)
	assert.Equal(t, "", cfg.Health.Auth.Secret)
	assert.Equal(t, "localhost", cfg.DB.Host)
	assert.Equal(t, 5432, cfg.DB.Port)
	assert.Equal(t, "postgres", cfg.DB.Username)
	assert.False(t, cfg.DB.SSL)
}

func TestLoad_WithFile(t *testing.T) {
	yamlContent := `logger:
  level: debug
health:
  address: ":8080"
  auth:
    enabled: true
    secret: "mysecret"
db:
  host: "remote-host"
  port: 3306
  username: "admin"
  ssl: true
`

	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	_, err = tmpFile.WriteString(yamlContent)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	var cfg TestConfig

	err = Load(&cfg, WithFile(tmpFile.Name()))
	require.NoError(t, err)

	assertExpectedTestConfig(t, cfg)
}

func TestLoad_WithEnv(t *testing.T) {
	envVars := map[string]string{
		"TEST_LOGGER_LEVEL":        "error",
		"TEST_HEALTH_ADDRESS":      ":7777",
		"TEST_HEALTH_AUTH_ENABLED": "true",
		"TEST_HEALTH_AUTH_SECRET":  "envsecret",
		"TEST_DB_HOST":             "env-host",
		"TEST_DB_PORT":             "1234",
		"TEST_DB_USERNAME":         "envuser",
		"TEST_DB_SSL":              "true",
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
	assert.Equal(t, ":7777", cfg.Health.Address)
	assert.True(t, cfg.Health.Auth.Enabled)
	assert.Equal(t, "envsecret", cfg.Health.Auth.Secret)
	assert.Equal(t, "env-host", cfg.DB.Host)
	assert.Equal(t, 1234, cfg.DB.Port)
	assert.Equal(t, "envuser", cfg.DB.Username)
	assert.True(t, cfg.DB.SSL)
}

func TestLoad_FileAndEnv(t *testing.T) {
	yamlContent := `logger:
  level: debug
health:
  address: ":8080"
db:
  host: "file-host"
  port: 3306
`

	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	_, err = tmpFile.WriteString(yamlContent)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	envVars := map[string]string{
		"TEST_LOGGER_LEVEL":        "warn",
		"TEST_HEALTH_AUTH_ENABLED": "true",
		"TEST_HEALTH_AUTH_SECRET":  "envsecret",
		"TEST_DB_USERNAME":         "envuser",
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

	err = Load(&cfg, WithFile(tmpFile.Name()), WithEnv("TEST"))
	require.NoError(t, err)

	assert.Equal(t, "warn", cfg.Logger.Level)
	assert.Equal(t, ":8080", cfg.Health.Address)
	assert.True(t, cfg.Health.Auth.Enabled)
	assert.Equal(t, "envsecret", cfg.Health.Auth.Secret)
	assert.Equal(t, "file-host", cfg.DB.Host)
	assert.Equal(t, 3306, cfg.DB.Port)
	assert.Equal(t, "envuser", cfg.DB.Username)
	assert.False(t, cfg.DB.SSL)
}

func TestLoad_NonExistentFile(t *testing.T) {
	var cfg TestConfig

	err := Load(&cfg, WithFile("non-existent-file.yaml"))
	require.NoError(t, err)

	assert.Equal(t, "info", cfg.Logger.Level)
	assert.Equal(t, ":9999", cfg.Health.Address)
}

func TestLoad_InvalidConfig(t *testing.T) {
	err := Load(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config must be a non-nil pointer")

	var cfg TestConfig
	err = Load(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config must be a non-nil pointer")
}

func TestLoad_InvalidYAML(t *testing.T) {
	invalidYAML := `logger:
  level: debug
health:
  - invalid yaml structure
`

	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	_, err = tmpFile.WriteString(invalidYAML)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	var cfg TestConfig

	err = Load(&cfg, WithFile(tmpFile.Name()))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load from file")
}

func TestLoad_InvalidEnvValues(t *testing.T) {
	envVars := map[string]string{
		"TEST_DB_PORT":             "invalid-port",
		"TEST_HEALTH_AUTH_ENABLED": "invalid-bool",
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
	assert.Error(t, err)
}

func TestLoad_ConcurrentAccess(t *testing.T) {
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()

			var cfg TestConfig
			err := Load(&cfg)
			assert.NoError(t, err)
			assert.Equal(t, "info", cfg.Logger.Level)
			assert.Equal(t, ":9999", cfg.Health.Address)
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestLoad_PartialConfig(t *testing.T) {
	yamlContent := `logger:
  level: debug
`

	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	_, err = tmpFile.WriteString(yamlContent)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	var cfg TestConfig

	err = Load(&cfg, WithFile(tmpFile.Name()))
	require.NoError(t, err)

	assert.Equal(t, "debug", cfg.Logger.Level)
	assert.Equal(t, ":9999", cfg.Health.Address)
	assert.False(t, cfg.Health.Auth.Enabled)
	assert.Equal(t, "localhost", cfg.DB.Host)
}

type ConfigWithoutDefaults struct {
	Name  string `yaml:"name"`
	Value int    `yaml:"value"`
}

func TestLoad_NoDefaultMethods(t *testing.T) {
	var cfg ConfigWithoutDefaults

	err := Load(&cfg)
	require.NoError(t, err)

	assert.Equal(t, "", cfg.Name)
	assert.Equal(t, 0, cfg.Value)
}

func TestLoad_EnvCaseSensitivity(t *testing.T) {
	envVars := map[string]string{
		"test_logger_level": "debug",
		"TEST_LOGGER_LEVEL": "info",
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

	assert.Equal(t, "info", cfg.Logger.Level)
}

func TestLoad_WithJSONFile(t *testing.T) {
	jsonContent := `{
  "logger": {
    "level": "debug"
  },
  "health": {
    "address": ":8080",
    "auth": {
      "enabled": true,
      "secret": "mysecret"
    }
  },
  "db": {
    "host": "remote-host",
    "port": 3306,
    "username": "admin",
    "ssl": true
  }
}`

	tmpFile, err := os.CreateTemp("", "config-*.json")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	_, err = tmpFile.WriteString(jsonContent)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	var cfg TestConfig

	err = Load(&cfg, WithFile(tmpFile.Name()))
	require.NoError(t, err)

	assertExpectedTestConfig(t, cfg)
}

func TestLoad_InvalidJSONFile(t *testing.T) {
	invalidJSON := `{
  "logger": {
    "level": "debug"
  },
  "health": {
    "address": ":8080",
    invalid json structure
  }
}`

	tmpFile, err := os.CreateTemp("", "config-*.json")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	_, err = tmpFile.WriteString(invalidJSON)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	var cfg TestConfig

	err = Load(&cfg, WithFile(tmpFile.Name()))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load from file")
}

func TestLoad_JSONTagsOnly(t *testing.T) {
	type JSONOnlyConfig struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	jsonContent := `{
  "name": "test",
  "age": 25
}`

	tmpFile, err := os.CreateTemp("", "config-*.json")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	_, err = tmpFile.WriteString(jsonContent)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	var cfg JSONOnlyConfig

	err = Load(&cfg, WithFile(tmpFile.Name()))
	require.NoError(t, err)

	assert.Equal(t, "test", cfg.Name)
	assert.Equal(t, 25, cfg.Age)
}

func TestLoad_JSONEnvVars(t *testing.T) {
	type JSONOnlyConfig struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	envVars := map[string]string{
		"TEST_NAME": "envtest",
		"TEST_AGE":  "30",
	}

	for key, value := range envVars {
		require.NoError(t, os.Setenv(key, value))
	}
	defer func() {
		for key := range envVars {
			_ = os.Unsetenv(key)
		}
	}()

	var cfg JSONOnlyConfig

	err := Load(&cfg, WithEnv("TEST"))
	require.NoError(t, err)

	assert.Equal(t, "envtest", cfg.Name)
	assert.Equal(t, 30, cfg.Age)
}

func TestLoad_FileExtensionDetection(t *testing.T) {
	testCases := []struct {
		name      string
		ext       string
		content   string
		expectErr bool
	}{
		{
			name: "json extension",
			ext:  ".json",
			content: `{
  "logger": {
    "level": "debug"
  }
}`,
			expectErr: false,
		},
		{
			name: "yaml extension",
			ext:  ".yaml",
			content: `logger:
  level: debug`,
			expectErr: false,
		},
		{
			name: "yml extension",
			ext:  ".yml",
			content: `logger:
  level: debug`,
			expectErr: false,
		},
		{
			name: "unknown extension defaults to yaml",
			ext:  ".txt",
			content: `logger:
  level: debug`,
			expectErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpFile, err := os.CreateTemp("", "config-*"+tc.ext)
			require.NoError(t, err)
			defer func() { _ = os.Remove(tmpFile.Name()) }()

			_, err = tmpFile.WriteString(tc.content)
			require.NoError(t, err)
			require.NoError(t, tmpFile.Close())

			var cfg TestConfig

			err = Load(&cfg, WithFile(tmpFile.Name()))
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, "debug", cfg.Logger.Level)
			}
		})
	}
}

func TestLoad_MultipleFiles(t *testing.T) {
	baseYAML := `logger:
  level: info
health:
  address: ":9999"
db:
  host: "localhost"
  port: 5432`

	overrideJSON := `{
  "logger": {
    "level": "debug"
  },
  "health": {
    "address": ":8080",
    "auth": {
      "enabled": true,
      "secret": "override-secret"
    }
  }
}`

	baseFile, err := os.CreateTemp("", "base-*.yaml")
	require.NoError(t, err)
	defer func() { _ = os.Remove(baseFile.Name()) }()

	_, err = baseFile.WriteString(baseYAML)
	require.NoError(t, err)
	require.NoError(t, baseFile.Close())

	overrideFile, err := os.CreateTemp("", "override-*.json")
	require.NoError(t, err)
	defer func() { _ = os.Remove(overrideFile.Name()) }()

	_, err = overrideFile.WriteString(overrideJSON)
	require.NoError(t, err)
	require.NoError(t, overrideFile.Close())

	var cfg TestConfig

	err = Load(&cfg, WithFile(baseFile.Name()), WithFile(overrideFile.Name()))
	require.NoError(t, err)

	assert.Equal(t, "debug", cfg.Logger.Level)
	assert.Equal(t, ":8080", cfg.Health.Address)
	assert.True(t, cfg.Health.Auth.Enabled)
	assert.Equal(t, "override-secret", cfg.Health.Auth.Secret)
	assert.Equal(t, "localhost", cfg.DB.Host)
	assert.Equal(t, 5432, cfg.DB.Port)
	assert.Equal(t, "postgres", cfg.DB.Username)
	assert.False(t, cfg.DB.SSL)
}

func TestLoad_WithFilesOption(t *testing.T) {
	file1Content := `logger:
  level: info
db:
  host: "file1-host"`

	file2Content := `{
  "logger": {
    "level": "warn"
  },
  "health": {
    "address": ":7777"
  }
}`

	file3Content := `logger:
  level: error
health:
  auth:
    enabled: true`

	file1, err := os.CreateTemp("", "config1-*.yaml")
	require.NoError(t, err)
	defer func() { _ = os.Remove(file1.Name()) }()

	file2, err := os.CreateTemp("", "config2-*.json")
	require.NoError(t, err)
	defer func() { _ = os.Remove(file2.Name()) }()

	file3, err := os.CreateTemp("", "config3-*.yml")
	require.NoError(t, err)
	defer func() { _ = os.Remove(file3.Name()) }()

	_, err = file1.WriteString(file1Content)
	require.NoError(t, err)
	require.NoError(t, file1.Close())

	_, err = file2.WriteString(file2Content)
	require.NoError(t, err)
	require.NoError(t, file2.Close())

	_, err = file3.WriteString(file3Content)
	require.NoError(t, err)
	require.NoError(t, file3.Close())

	var cfg TestConfig

	err = Load(&cfg, WithFiles(file1.Name(), file2.Name(), file3.Name()))
	require.NoError(t, err)

	assert.Equal(t, "error", cfg.Logger.Level)
	assert.Equal(t, ":7777", cfg.Health.Address)
	assert.True(t, cfg.Health.Auth.Enabled)
	assert.Equal(t, "", cfg.Health.Auth.Secret)
	assert.Equal(t, "file1-host", cfg.DB.Host)
	assert.Equal(t, 5432, cfg.DB.Port)
}

func TestLoad_MultipleFilesWithEnv(t *testing.T) {
	baseConfig := `logger:
  level: info
health:
  address: ":9999"`

	overrideConfig := `{
  "health": {
    "address": ":8080"
  },
  "db": {
    "host": "json-host"
  }
}`

	baseFile, err := os.CreateTemp("", "base-*.yaml")
	require.NoError(t, err)
	defer func() { _ = os.Remove(baseFile.Name()) }()

	overrideFile, err := os.CreateTemp("", "override-*.json")
	require.NoError(t, err)
	defer func() { _ = os.Remove(overrideFile.Name()) }()

	_, err = baseFile.WriteString(baseConfig)
	require.NoError(t, err)
	require.NoError(t, baseFile.Close())

	_, err = overrideFile.WriteString(overrideConfig)
	require.NoError(t, err)
	require.NoError(t, overrideFile.Close())

	envVars := map[string]string{
		"TEST_LOGGER_LEVEL":        "debug",
		"TEST_HEALTH_AUTH_ENABLED": "true",
		"TEST_HEALTH_AUTH_SECRET":  "env-secret",
		"TEST_DB_PORT":             "3306",
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

	err = Load(&cfg, WithFiles(baseFile.Name(), overrideFile.Name()), WithEnv("TEST"))
	require.NoError(t, err)

	assert.Equal(t, "debug", cfg.Logger.Level)
	assert.Equal(t, ":8080", cfg.Health.Address)
	assert.True(t, cfg.Health.Auth.Enabled)
	assert.Equal(t, "env-secret", cfg.Health.Auth.Secret)
	assert.Equal(t, "json-host", cfg.DB.Host)
	assert.Equal(t, 3306, cfg.DB.Port)
}

func TestLoad_MixedFileFormats(t *testing.T) {
	yamlFile := `logger:
  level: yaml-level
health:
  address: ":9000"`

	jsonFile := `{
  "logger": {
    "level": "json-level"
  },
  "db": {
    "host": "json-host",
    "ssl": true
  }
}`

	ymlFile := `health:
  address: ":7000"
  auth:
    enabled: true
db:
  port: 1234`

	tmpYaml, err := os.CreateTemp("", "config-*.yaml")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpYaml.Name()) }()

	tmpJSON, err := os.CreateTemp("", "config-*.json")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpJSON.Name()) }()

	tmpYml, err := os.CreateTemp("", "config-*.yml")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpYml.Name()) }()

	_, err = tmpYaml.WriteString(yamlFile)
	require.NoError(t, err)
	require.NoError(t, tmpYaml.Close())

	_, err = tmpJSON.WriteString(jsonFile)
	require.NoError(t, err)
	require.NoError(t, tmpJSON.Close())

	_, err = tmpYml.WriteString(ymlFile)
	require.NoError(t, err)
	require.NoError(t, tmpYml.Close())

	var cfg TestConfig

	err = Load(&cfg, WithFiles(tmpYaml.Name(), tmpJSON.Name(), tmpYml.Name()))
	require.NoError(t, err)

	assert.Equal(t, "json-level", cfg.Logger.Level)
	assert.Equal(t, ":7000", cfg.Health.Address)
	assert.True(t, cfg.Health.Auth.Enabled)
	assert.Equal(t, "", cfg.Health.Auth.Secret)
	assert.Equal(t, "json-host", cfg.DB.Host)
	assert.Equal(t, 1234, cfg.DB.Port)
	assert.Equal(t, "postgres", cfg.DB.Username)
	assert.True(t, cfg.DB.SSL)
}

func TestLoad_NonExistentFileInMultiple(t *testing.T) {
	validConfig := `logger:
  level: debug`

	validFile, err := os.CreateTemp("", "valid-*.yaml")
	require.NoError(t, err)
	defer func() { _ = os.Remove(validFile.Name()) }()

	_, err = validFile.WriteString(validConfig)
	require.NoError(t, err)
	require.NoError(t, validFile.Close())

	var cfg TestConfig

	err = Load(&cfg, WithFiles(validFile.Name(), "non-existent.yaml"))
	require.NoError(t, err)

	assert.Equal(t, "debug", cfg.Logger.Level)
	assert.Equal(t, ":9999", cfg.Health.Address)
}

func TestLoad_InvalidFileInMultiple(t *testing.T) {
	validConfig := `logger:
  level: debug`

	invalidConfig := `logger:
  level: debug
health:
  - invalid yaml`

	validFile, err := os.CreateTemp("", "valid-*.yaml")
	require.NoError(t, err)
	defer func() { _ = os.Remove(validFile.Name()) }()

	invalidFile, err := os.CreateTemp("", "invalid-*.yaml")
	require.NoError(t, err)
	defer func() { _ = os.Remove(invalidFile.Name()) }()

	_, err = validFile.WriteString(validConfig)
	require.NoError(t, err)
	require.NoError(t, validFile.Close())

	_, err = invalidFile.WriteString(invalidConfig)
	require.NoError(t, err)
	require.NoError(t, invalidFile.Close())

	var cfg TestConfig

	err = Load(&cfg, WithFiles(validFile.Name(), invalidFile.Name()))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load from files")
}

type SliceConfig struct {
	Hosts   []string  `yaml:"hosts" json:"hosts"`
	Ports   []int     `yaml:"ports" json:"ports"`
	Enabled []bool    `yaml:"enabled" json:"enabled"`
	Weights []float64 `yaml:"weights" json:"weights"`
	Tags    []string  `yaml:"tags" json:"tags"`
}

func (c *SliceConfig) Default() {
	*c = SliceConfig{
		Hosts:   []string{"localhost"},
		Ports:   []int{8080},
		Enabled: []bool{true},
		Weights: []float64{1.0},
		Tags:    []string{"default"},
	}
}

func TestLoad_SlicesFromEnv(t *testing.T) {
	envVars := map[string]string{
		"TEST_HOSTS":   "host1,host2,host3",
		"TEST_PORTS":   "8080,9090,3000",
		"TEST_ENABLED": "true,false,true",
		"TEST_WEIGHTS": "1.0,2.5,0.8",
		"TEST_TAGS":    "web,api,cache",
	}

	for key, value := range envVars {
		require.NoError(t, os.Setenv(key, value))
	}
	defer func() {
		for key := range envVars {
			_ = os.Unsetenv(key)
		}
	}()

	var cfg SliceConfig

	err := Load(&cfg, WithEnv("TEST"))
	require.NoError(t, err)

	assert.Equal(t, []string{"host1", "host2", "host3"}, cfg.Hosts)
	assert.Equal(t, []int{8080, 9090, 3000}, cfg.Ports)
	assert.Equal(t, []bool{true, false, true}, cfg.Enabled)
	assert.Equal(t, []float64{1.0, 2.5, 0.8}, cfg.Weights)
	assert.Equal(t, []string{"web", "api", "cache"}, cfg.Tags)
}

func TestLoad_SlicesWithSpaces(t *testing.T) {
	envVars := map[string]string{
		"TEST_HOSTS": " host1 , host2 , host3 ",
		"TEST_PORTS": "8080, 9090 ,3000",
		"TEST_TAGS":  "web , api, cache ",
	}

	for key, value := range envVars {
		require.NoError(t, os.Setenv(key, value))
	}
	defer func() {
		for key := range envVars {
			_ = os.Unsetenv(key)
		}
	}()

	var cfg SliceConfig

	err := Load(&cfg, WithEnv("TEST"))
	require.NoError(t, err)

	assert.Equal(t, []string{"host1", "host2", "host3"}, cfg.Hosts)
	assert.Equal(t, []int{8080, 9090, 3000}, cfg.Ports)
	assert.Equal(t, []string{"web", "api", "cache"}, cfg.Tags)
}

func TestLoad_EmptySliceEnv(t *testing.T) {
	envVars := map[string]string{
		"TEST_HOSTS": "",
		"TEST_PORTS": ",,",
		"TEST_TAGS":  " , , ",
	}

	for key, value := range envVars {
		require.NoError(t, os.Setenv(key, value))
	}
	defer func() {
		for key := range envVars {
			_ = os.Unsetenv(key)
		}
	}()

	var cfg SliceConfig

	err := Load(&cfg, WithEnv("TEST"))
	require.NoError(t, err)

	assert.Equal(t, []string{"localhost"}, cfg.Hosts)
	assert.Equal(t, []int{}, cfg.Ports)
	assert.Equal(t, []string{}, cfg.Tags)
}

func TestLoad_SlicesFromYAML(t *testing.T) {
	yamlContent := `hosts:
  - "server1.example.com"
  - "server2.example.com"
  - "server3.example.com"
ports:
  - 8080
  - 9090
  - 3000
enabled:
  - true
  - false
  - true
weights:
  - 1.0
  - 2.5
  - 0.8
tags:
  - "production"
  - "api"
  - "cache"`

	tmpFile, err := os.CreateTemp("", "slice-config-*.yaml")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	_, err = tmpFile.WriteString(yamlContent)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	var cfg SliceConfig

	err = Load(&cfg, WithFile(tmpFile.Name()))
	require.NoError(t, err)

	assert.Equal(t, []string{"server1.example.com", "server2.example.com", "server3.example.com"}, cfg.Hosts)
	assert.Equal(t, []int{8080, 9090, 3000}, cfg.Ports)
	assert.Equal(t, []bool{true, false, true}, cfg.Enabled)
	assert.Equal(t, []float64{1.0, 2.5, 0.8}, cfg.Weights)
	assert.Equal(t, []string{"production", "api", "cache"}, cfg.Tags)
}

func TestLoad_SlicesFromJSON(t *testing.T) {
	jsonContent := `{
  "hosts": ["api.example.com", "db.example.com"],
  "ports": [443, 5432],
  "enabled": [true, true],
  "weights": [1.5, 3.0],
  "tags": ["secure", "database"]
}`

	tmpFile, err := os.CreateTemp("", "slice-config-*.json")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	_, err = tmpFile.WriteString(jsonContent)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	var cfg SliceConfig

	err = Load(&cfg, WithFile(tmpFile.Name()))
	require.NoError(t, err)

	assert.Equal(t, []string{"api.example.com", "db.example.com"}, cfg.Hosts)
	assert.Equal(t, []int{443, 5432}, cfg.Ports)
	assert.Equal(t, []bool{true, true}, cfg.Enabled)
	assert.Equal(t, []float64{1.5, 3.0}, cfg.Weights)
	assert.Equal(t, []string{"secure", "database"}, cfg.Tags)
}

func TestLoad_SlicesFileAndEnv(t *testing.T) {
	yamlContent := `hosts:
  - "file-host1"
  - "file-host2"
ports:
  - 8080
  - 9090`

	tmpFile, err := os.CreateTemp("", "slice-config-*.yaml")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	_, err = tmpFile.WriteString(yamlContent)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	envVars := map[string]string{
		"TEST_HOSTS": "env-host1,env-host2,env-host3",
		"TEST_TAGS":  "env-tag1,env-tag2",
	}

	for key, value := range envVars {
		require.NoError(t, os.Setenv(key, value))
	}
	defer func() {
		for key := range envVars {
			_ = os.Unsetenv(key)
		}
	}()

	var cfg SliceConfig

	err = Load(&cfg, WithFile(tmpFile.Name()), WithEnv("TEST"))
	require.NoError(t, err)

	assert.Equal(t, []string{"env-host1", "env-host2", "env-host3"}, cfg.Hosts)
	assert.Equal(t, []int{8080, 9090}, cfg.Ports)
	assert.Equal(t, []string{"env-tag1", "env-tag2"}, cfg.Tags)
}

func TestLoad_InvalidSliceValues(t *testing.T) {
	testCases := []struct {
		name   string
		envVar string
		value  string
	}{
		{"invalid int", "TEST_PORTS", "8080,invalid,9090"},
		{"invalid bool", "TEST_ENABLED", "true,invalid,false"},
		{"invalid float", "TEST_WEIGHTS", "1.0,invalid,2.0"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.NoError(t, os.Setenv(tc.envVar, tc.value))
			defer func() { _ = os.Unsetenv(tc.envVar) }()

			var cfg SliceConfig

			err := Load(&cfg, WithEnv("TEST"))
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid")
		})
	}
}

func TestLoad_SliceDefaults(t *testing.T) {
	var cfg SliceConfig

	err := Load(&cfg)
	require.NoError(t, err)

	assert.Equal(t, []string{"localhost"}, cfg.Hosts)
	assert.Equal(t, []int{8080}, cfg.Ports)
	assert.Equal(t, []bool{true}, cfg.Enabled)
	assert.Equal(t, []float64{1.0}, cfg.Weights)
	assert.Equal(t, []string{"default"}, cfg.Tags)
}

type NestedSliceConfig struct {
	Database DatabaseSliceConfig `yaml:"database" json:"database"`
	Servers  []ServerConfig      `yaml:"servers" json:"servers"`
}

type DatabaseSliceConfig struct {
	Hosts []string `yaml:"hosts" json:"hosts"`
	Ports []int    `yaml:"ports" json:"ports"`
}

func (c *DatabaseSliceConfig) Default() {
	*c = DatabaseSliceConfig{
		Hosts: []string{"localhost"},
		Ports: []int{5432},
	}
}

type ServerConfig struct {
	Name string `yaml:"name" json:"name"`
	Port int    `yaml:"port" json:"port"`
}

func TestLoad_NestedSlicesFromEnv(t *testing.T) {
	envVars := map[string]string{
		"TEST_DATABASE_HOSTS": "db1,db2,db3",
		"TEST_DATABASE_PORTS": "5432,5433,5434",
	}

	for key, value := range envVars {
		require.NoError(t, os.Setenv(key, value))
	}
	defer func() {
		for key := range envVars {
			_ = os.Unsetenv(key)
		}
	}()

	var cfg NestedSliceConfig

	err := Load(&cfg, WithEnv("TEST"))
	require.NoError(t, err)

	assert.Equal(t, []string{"db1", "db2", "db3"}, cfg.Database.Hosts)
	assert.Equal(t, []int{5432, 5433, 5434}, cfg.Database.Ports)
}

type MapConfig struct {
	Labels   map[string]string  `yaml:"labels" json:"labels"`
	Ports    map[string]int     `yaml:"ports" json:"ports"`
	Features map[string]bool    `yaml:"features" json:"features"`
	Weights  map[string]float64 `yaml:"weights" json:"weights"`
	Metadata map[string]string  `yaml:"metadata" json:"metadata"`
}

func (c *MapConfig) Default() {
	*c = MapConfig{
		Labels:   map[string]string{"env": "dev"},
		Ports:    map[string]int{"http": 8080},
		Features: map[string]bool{"auth": true},
		Weights:  map[string]float64{"cpu": 1.0},
		Metadata: map[string]string{"version": "1.0"},
	}
}

func TestLoad_MapsFromEnv(t *testing.T) {
	envVars := map[string]string{
		"TEST_LABELS":   "env=prod,region=us-east,tier=web",
		"TEST_PORTS":    "http=80,https=443,ssh=22",
		"TEST_FEATURES": "auth=true,cache=false,ssl=true",
		"TEST_WEIGHTS":  "cpu=2.5,memory=1.8,disk=0.9",
		"TEST_METADATA": "version=2.0,build=abc123",
	}

	for key, value := range envVars {
		require.NoError(t, os.Setenv(key, value))
	}
	defer func() {
		for key := range envVars {
			_ = os.Unsetenv(key)
		}
	}()

	var cfg MapConfig

	err := Load(&cfg, WithEnv("TEST"))
	require.NoError(t, err)

	expectedLabels := map[string]string{"env": "prod", "region": "us-east", "tier": "web"}
	expectedPorts := map[string]int{"http": 80, "https": 443, "ssh": 22}
	expectedFeatures := map[string]bool{"auth": true, "cache": false, "ssl": true}
	expectedWeights := map[string]float64{"cpu": 2.5, "memory": 1.8, "disk": 0.9}
	expectedMetadata := map[string]string{"version": "2.0", "build": "abc123"}

	assert.Equal(t, expectedLabels, cfg.Labels)
	assert.Equal(t, expectedPorts, cfg.Ports)
	assert.Equal(t, expectedFeatures, cfg.Features)
	assert.Equal(t, expectedWeights, cfg.Weights)
	assert.Equal(t, expectedMetadata, cfg.Metadata)
}

func TestLoad_MapsWithSpaces(t *testing.T) {
	envVars := map[string]string{
		"TEST_LABELS": " env=prod , region=us-east , tier=web ",
		"TEST_PORTS":  "http = 80, https= 443 ,ssh =22",
	}

	for key, value := range envVars {
		require.NoError(t, os.Setenv(key, value))
	}
	defer func() {
		for key := range envVars {
			_ = os.Unsetenv(key)
		}
	}()

	var cfg MapConfig

	err := Load(&cfg, WithEnv("TEST"))
	require.NoError(t, err)

	expectedLabels := map[string]string{"env": "prod", "region": "us-east", "tier": "web"}
	expectedPorts := map[string]int{"http": 80, "https": 443, "ssh": 22}

	assert.Equal(t, expectedLabels, cfg.Labels)
	assert.Equal(t, expectedPorts, cfg.Ports)
}

func TestLoad_EmptyMapEnv(t *testing.T) {
	envVars := map[string]string{
		"TEST_LABELS":   "",
		"TEST_PORTS":    ",,",
		"TEST_METADATA": " , , ",
	}

	for key, value := range envVars {
		require.NoError(t, os.Setenv(key, value))
	}
	defer func() {
		for key := range envVars {
			_ = os.Unsetenv(key)
		}
	}()

	var cfg MapConfig

	err := Load(&cfg, WithEnv("TEST"))
	require.NoError(t, err)

	assert.Equal(t, map[string]string{"env": "dev"}, cfg.Labels)
	assert.Equal(t, map[string]int{}, cfg.Ports)
	assert.Equal(t, map[string]string{}, cfg.Metadata)
}

func TestLoad_MapsFromYAML(t *testing.T) {
	yamlContent := `labels:
  env: "production"
  region: "us-west"
  service: "api"
ports:
  http: 8080
  https: 8443
  metrics: 9090
features:
  auth: true
  cache: false
  logging: true
weights:
  cpu: 2.0
  memory: 1.5
  network: 0.8
metadata:
  version: "3.0"
  commit: "def456"`

	tmpFile, err := os.CreateTemp("", "map-config-*.yaml")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	_, err = tmpFile.WriteString(yamlContent)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	var cfg MapConfig

	err = Load(&cfg, WithFile(tmpFile.Name()))
	require.NoError(t, err)

	expectedLabels := map[string]string{"env": "production", "region": "us-west", "service": "api"}
	expectedPorts := map[string]int{"http": 8080, "https": 8443, "metrics": 9090}
	expectedFeatures := map[string]bool{"auth": true, "cache": false, "logging": true}
	expectedWeights := map[string]float64{"cpu": 2.0, "memory": 1.5, "network": 0.8}
	expectedMetadata := map[string]string{"version": "3.0", "commit": "def456"}

	assert.Equal(t, expectedLabels, cfg.Labels)
	assert.Equal(t, expectedPorts, cfg.Ports)
	assert.Equal(t, expectedFeatures, cfg.Features)
	assert.Equal(t, expectedWeights, cfg.Weights)
	assert.Equal(t, expectedMetadata, cfg.Metadata)
}

func TestLoad_MapsFromJSON(t *testing.T) {
	jsonContent := `{
  "labels": {
    "env": "staging",
    "team": "backend"
  },
  "ports": {
    "api": 3000,
    "health": 3001
  },
  "features": {
    "debug": true,
    "profiling": false
  },
  "weights": {
    "priority": 0.7,
    "load": 2.3
  },
  "metadata": {
    "deploy": "manual",
    "owner": "devops"
  }
}`

	tmpFile, err := os.CreateTemp("", "map-config-*.json")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	_, err = tmpFile.WriteString(jsonContent)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	var cfg MapConfig

	err = Load(&cfg, WithFile(tmpFile.Name()))
	require.NoError(t, err)

	expectedLabels := map[string]string{"env": "staging", "team": "backend"}
	expectedPorts := map[string]int{"api": 3000, "health": 3001, "http": 8080}
	expectedFeatures := map[string]bool{"debug": true, "profiling": false, "auth": true}
	expectedWeights := map[string]float64{"priority": 0.7, "load": 2.3, "cpu": 1.0}
	expectedMetadata := map[string]string{"deploy": "manual", "owner": "devops", "version": "1.0"}

	assert.Equal(t, expectedLabels, cfg.Labels)
	assert.Equal(t, expectedPorts, cfg.Ports)
	assert.Equal(t, expectedFeatures, cfg.Features)
	assert.Equal(t, expectedWeights, cfg.Weights)
	assert.Equal(t, expectedMetadata, cfg.Metadata)
}

func TestLoad_MapsFileAndEnv(t *testing.T) {
	yamlContent := `labels:
  env: "file-env"
  service: "file-service"
ports:
  http: 8080
  metrics: 9090`

	tmpFile, err := os.CreateTemp("", "map-config-*.yaml")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	_, err = tmpFile.WriteString(yamlContent)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	envVars := map[string]string{
		"TEST_LABELS":   "env=env-override,region=us-west",
		"TEST_METADATA": "source=env,priority=high",
	}

	for key, value := range envVars {
		require.NoError(t, os.Setenv(key, value))
	}
	defer func() {
		for key := range envVars {
			_ = os.Unsetenv(key)
		}
	}()

	var cfg MapConfig

	err = Load(&cfg, WithFile(tmpFile.Name()), WithEnv("TEST"))
	require.NoError(t, err)

	expectedLabels := map[string]string{"env": "env-override", "region": "us-west"}
	expectedPorts := map[string]int{"http": 8080, "metrics": 9090}
	expectedMetadata := map[string]string{"source": "env", "priority": "high"}

	assert.Equal(t, expectedLabels, cfg.Labels)
	assert.Equal(t, expectedPorts, cfg.Ports)
	assert.Equal(t, expectedMetadata, cfg.Metadata)
}

func TestLoad_InvalidMapValues(t *testing.T) {
	testCases := []struct {
		name   string
		envVar string
		value  string
	}{
		{"invalid format missing equals", "TEST_LABELS", "envprod,region=us-east"},
		{"invalid int value", "TEST_PORTS", "http=80,https=invalid"},
		{"invalid bool value", "TEST_FEATURES", "auth=true,cache=invalid"},
		{"invalid float value", "TEST_WEIGHTS", "cpu=2.5,memory=invalid"},
		{"empty key", "TEST_LABELS", "=value,key=value2"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.NoError(t, os.Setenv(tc.envVar, tc.value))
			defer func() { _ = os.Unsetenv(tc.envVar) }()

			var cfg MapConfig

			err := Load(&cfg, WithEnv("TEST"))
			assert.Error(t, err)
		})
	}
}

func TestLoad_MapDefaults(t *testing.T) {
	var cfg MapConfig

	err := Load(&cfg)
	require.NoError(t, err)

	assert.Equal(t, map[string]string{"env": "dev"}, cfg.Labels)
	assert.Equal(t, map[string]int{"http": 8080}, cfg.Ports)
	assert.Equal(t, map[string]bool{"auth": true}, cfg.Features)
	assert.Equal(t, map[string]float64{"cpu": 1.0}, cfg.Weights)
	assert.Equal(t, map[string]string{"version": "1.0"}, cfg.Metadata)
}

type NestedMapConfig struct {
	Database DatabaseMapConfig      `yaml:"database" json:"database"`
	Services map[string]ServiceInfo `yaml:"services" json:"services"`
}

type DatabaseMapConfig struct {
	Config map[string]string `yaml:"config" json:"config"`
	Ports  map[string]int    `yaml:"ports" json:"ports"`
}

func (c *DatabaseMapConfig) Default() {
	*c = DatabaseMapConfig{
		Config: map[string]string{"driver": "postgres"},
		Ports:  map[string]int{"main": 5432},
	}
}

type ServiceInfo struct {
	Port    int  `yaml:"port" json:"port"`
	Enabled bool `yaml:"enabled" json:"enabled"`
}

func TestLoad_NestedMapsFromEnv(t *testing.T) {
	envVars := map[string]string{
		"TEST_DATABASE_CONFIG": "host=localhost,ssl=require",
		"TEST_DATABASE_PORTS":  "main=5432,replica=5433",
	}

	for key, value := range envVars {
		require.NoError(t, os.Setenv(key, value))
	}
	defer func() {
		for key := range envVars {
			_ = os.Unsetenv(key)
		}
	}()

	var cfg NestedMapConfig

	err := Load(&cfg, WithEnv("TEST"))
	require.NoError(t, err)

	expectedConfig := map[string]string{"host": "localhost", "ssl": "require"}
	expectedPorts := map[string]int{"main": 5432, "replica": 5433}

	assert.Equal(t, expectedConfig, cfg.Database.Config)
	assert.Equal(t, expectedPorts, cfg.Database.Ports)
}

func TestLoad_UnsupportedMapKeyType(t *testing.T) {
	type UnsupportedMapConfig struct {
		IntKeyMap map[int]string `yaml:"int_key_map" json:"int_key_map"`
	}

	require.NoError(t, os.Setenv("TEST_INT_KEY_MAP", "1=value1,2=value2"))
	defer func() { _ = os.Unsetenv("TEST_INT_KEY_MAP") }()

	var cfg UnsupportedMapConfig

	err := Load(&cfg, WithEnv("TEST"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported map key type")
}

func TestLoad_WithDefault(t *testing.T) {
	customDefault := TestConfig{
		Logger: LoggerConfig{Level: "trace"},
		Health: HealthConfig{
			Address: ":7777",
			Auth: HealthAuthConfig{
				Enabled: true,
				Secret:  "custom-secret",
			},
		},
		DB: DBConfig{
			Host:     "custom-host",
			Port:     3306,
			Username: "custom-user",
			SSL:      true,
		},
	}

	var cfg TestConfig

	err := Load(&cfg, WithDefault(customDefault))
	require.NoError(t, err)

	assert.Equal(t, "trace", cfg.Logger.Level)
	assert.Equal(t, ":7777", cfg.Health.Address)
	assert.True(t, cfg.Health.Auth.Enabled)
	assert.Equal(t, "custom-secret", cfg.Health.Auth.Secret)
	assert.Equal(t, "custom-host", cfg.DB.Host)
	assert.Equal(t, 3306, cfg.DB.Port)
	assert.Equal(t, "custom-user", cfg.DB.Username)
	assert.True(t, cfg.DB.SSL)
}

func TestLoad_WithDefaultOverridesStructDefaults(t *testing.T) {
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
}

func TestLoad_WithDefaultAndFile(t *testing.T) {
	customDefault := TestConfig{
		Logger: LoggerConfig{Level: "trace"},
		Health: HealthConfig{Address: ":7777"},
	}

	yamlContent := `logger:
  level: debug
db:
  host: "file-host"
  port: 5432`

	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	_, err = tmpFile.WriteString(yamlContent)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	var cfg TestConfig

	err = Load(&cfg, WithDefault(customDefault), WithFile(tmpFile.Name()))
	require.NoError(t, err)

	assert.Equal(t, "debug", cfg.Logger.Level)
	assert.Equal(t, ":7777", cfg.Health.Address)
	assert.Equal(t, "file-host", cfg.DB.Host)
	assert.Equal(t, 5432, cfg.DB.Port)
}

func TestLoad_WithDefaultAndEnv(t *testing.T) {
	customDefault := TestConfig{
		Logger: LoggerConfig{Level: "trace"},
		Health: HealthConfig{Address: ":7777"},
		DB:     DBConfig{Host: "custom-host"},
	}

	envVars := map[string]string{
		"TEST_LOGGER_LEVEL": "warn",
		"TEST_DB_PORT":      "3306",
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

	err := Load(&cfg, WithDefault(customDefault), WithEnv("TEST"))
	require.NoError(t, err)

	assert.Equal(t, "warn", cfg.Logger.Level)
	assert.Equal(t, ":7777", cfg.Health.Address)
	assert.Equal(t, "custom-host", cfg.DB.Host)
	assert.Equal(t, 3306, cfg.DB.Port)
}

func TestLoad_WithDefaultFileAndEnv(t *testing.T) {
	customDefault := TestConfig{
		Logger: LoggerConfig{Level: "trace"},
		Health: HealthConfig{
			Address: ":7777",
			Auth:    HealthAuthConfig{Enabled: true, Secret: "custom"},
		},
		DB: DBConfig{Host: "custom-host", Port: 1234},
	}

	yamlContent := `logger:
  level: debug
health:
  address: ":8888"
db:
  port: 5432`

	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	_, err = tmpFile.WriteString(yamlContent)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	envVars := map[string]string{
		"TEST_LOGGER_LEVEL":       "error",
		"TEST_HEALTH_AUTH_SECRET": "env-secret",
		"TEST_DB_HOST":            "env-host",
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

	err = Load(&cfg, WithDefault(customDefault), WithFile(tmpFile.Name()), WithEnv("TEST"))
	require.NoError(t, err)

	assert.Equal(t, "error", cfg.Logger.Level)
	assert.Equal(t, ":8888", cfg.Health.Address)
	assert.True(t, cfg.Health.Auth.Enabled)
	assert.Equal(t, "env-secret", cfg.Health.Auth.Secret)
	assert.Equal(t, "env-host", cfg.DB.Host)
	assert.Equal(t, 5432, cfg.DB.Port)
}

func TestLoad_WithDefaultSlices(t *testing.T) {
	customDefault := SliceConfig{
		Hosts:   []string{"custom1", "custom2"},
		Ports:   []int{9000, 9001},
		Enabled: []bool{false, true},
		Weights: []float64{2.5, 3.0},
		Tags:    []string{"custom"},
	}

	var cfg SliceConfig

	err := Load(&cfg, WithDefault(customDefault))
	require.NoError(t, err)

	assert.Equal(t, []string{"custom1", "custom2"}, cfg.Hosts)
	assert.Equal(t, []int{9000, 9001}, cfg.Ports)
	assert.Equal(t, []bool{false, true}, cfg.Enabled)
	assert.Equal(t, []float64{2.5, 3.0}, cfg.Weights)
	assert.Equal(t, []string{"custom"}, cfg.Tags)
}

func TestLoad_WithDefaultMaps(t *testing.T) {
	customDefault := MapConfig{
		Labels:   map[string]string{"env": "custom", "region": "custom-region"},
		Ports:    map[string]int{"api": 9000, "health": 9001},
		Features: map[string]bool{"debug": true, "profiling": true},
		Weights:  map[string]float64{"cpu": 3.0, "memory": 2.0},
		Metadata: map[string]string{"version": "custom", "build": "custom-build"},
	}

	var cfg MapConfig

	err := Load(&cfg, WithDefault(customDefault))
	require.NoError(t, err)

	expectedLabels := map[string]string{"env": "custom", "region": "custom-region"}
	expectedPorts := map[string]int{"api": 9000, "health": 9001}
	expectedFeatures := map[string]bool{"debug": true, "profiling": true}
	expectedWeights := map[string]float64{"cpu": 3.0, "memory": 2.0}
	expectedMetadata := map[string]string{"version": "custom", "build": "custom-build"}

	assert.Equal(t, expectedLabels, cfg.Labels)
	assert.Equal(t, expectedPorts, cfg.Ports)
	assert.Equal(t, expectedFeatures, cfg.Features)
	assert.Equal(t, expectedWeights, cfg.Weights)
	assert.Equal(t, expectedMetadata, cfg.Metadata)
}

func TestLoad_WithDefaultTypeMismatch(t *testing.T) {
	type DifferentConfig struct {
		Name string `yaml:"name"`
		Age  int    `yaml:"age"`
	}

	customDefault := DifferentConfig{Name: "test", Age: 25}

	var cfg TestConfig

	err := Load(&cfg, WithDefault(customDefault))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not match config type")
}

func TestLoad_WithDefaultNilConfig(t *testing.T) {
	customDefault := TestConfig{Logger: LoggerConfig{Level: "trace"}}

	err := Load(nil, WithDefault(customDefault))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config must be a non-nil pointer")
}

func TestLoad_WithDefaultNonPointerConfig(t *testing.T) {
	customDefault := TestConfig{Logger: LoggerConfig{Level: "trace"}}
	var cfg TestConfig

	err := Load(cfg, WithDefault(customDefault))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config must be a non-nil pointer")
}

type ConfigWithPointers struct {
	Logger *LoggerConfig `yaml:"logger" json:"logger"`
	Health *HealthConfig `yaml:"health" json:"health"`
}

func TestLoad_WithDefaultPointers(t *testing.T) {
	customDefault := ConfigWithPointers{
		Logger: &LoggerConfig{Level: "trace"},
		Health: &HealthConfig{
			Address: ":7777",
			Auth: HealthAuthConfig{
				Enabled: true,
				Secret:  "custom-secret",
			},
		},
	}

	var cfg ConfigWithPointers

	err := Load(&cfg, WithDefault(customDefault))
	require.NoError(t, err)

	require.NotNil(t, cfg.Logger)
	assert.Equal(t, "trace", cfg.Logger.Level)
	require.NotNil(t, cfg.Health)
	assert.Equal(t, ":7777", cfg.Health.Address)
	assert.True(t, cfg.Health.Auth.Enabled)
	assert.Equal(t, "custom-secret", cfg.Health.Auth.Secret)
}

// Additional comprehensive tests

func TestLoad_EdgeCaseEnvironmentVariables(t *testing.T) {
	type EdgeCaseConfig struct {
		EmptyString   string            `yaml:"empty_string"`
		ZeroInt       int               `yaml:"zero_int"`
		NegativeInt   int               `yaml:"negative_int"`
		LargeInt      int64             `yaml:"large_int"`
		SmallFloat    float32           `yaml:"small_float"`
		LargeFloat    float64           `yaml:"large_float"`
		UnicodeString string            `yaml:"unicode_string"`
		SpecialChars  string            `yaml:"special_chars"`
		BoolVariants  []bool            `yaml:"bool_variants"`
		MixedTypes    map[string]string `yaml:"mixed_types"`
	}

	envVars := map[string]string{
		"TEST_EMPTY_STRING":   "",
		"TEST_ZERO_INT":       "0",
		"TEST_NEGATIVE_INT":   "-42",
		"TEST_LARGE_INT":      "9223372036854775807",
		"TEST_SMALL_FLOAT":    "0.000001",
		"TEST_LARGE_FLOAT":    "1.7976931348623157e+308",
		"TEST_UNICODE_STRING": "Hello 世界 🌍",
		"TEST_SPECIAL_CHARS":  "!@#$%^&*()_+-=[]{}|;:,.<>?",
		"TEST_BOOL_VARIANTS":  "true,false,1,0,True,False,TRUE,FALSE",
		"TEST_MIXED_TYPES":    "url=https://example.com,path=/tmp/test,empty=",
	}

	for key, value := range envVars {
		require.NoError(t, os.Setenv(key, value))
	}
	defer func() {
		for key := range envVars {
			_ = os.Unsetenv(key)
		}
	}()

	var cfg EdgeCaseConfig

	err := Load(&cfg, WithEnv("TEST"))
	require.NoError(t, err)

	assert.Equal(t, "", cfg.EmptyString)
	assert.Equal(t, 0, cfg.ZeroInt)
	assert.Equal(t, -42, cfg.NegativeInt)
	assert.Equal(t, int64(9223372036854775807), cfg.LargeInt)
	assert.Equal(t, float32(0.000001), cfg.SmallFloat)
	assert.Equal(t, 1.7976931348623157e+308, cfg.LargeFloat)
	assert.Equal(t, "Hello 世界 🌍", cfg.UnicodeString)
	assert.Equal(t, "!@#$%^&*()_+-=[]{}|;:,.<>?", cfg.SpecialChars)

	expectedBools := []bool{true, false, true, false, true, false, true, false}
	assert.Equal(t, expectedBools, cfg.BoolVariants)

	expectedMap := map[string]string{"url": "https://example.com", "path": "/tmp/test", "empty": ""}
	assert.Equal(t, expectedMap, cfg.MixedTypes)
}

func TestLoad_ComplexNestedStructures(t *testing.T) {
	type DatabaseConnection struct {
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
		SSL      bool   `yaml:"ssl"`
	}

	type CacheConfig struct {
		Redis DatabaseConnection `yaml:"redis"`
		TTL   int                `yaml:"ttl"`
	}

	type ServiceConfig struct {
		Name      string            `yaml:"name"`
		Endpoints []string          `yaml:"endpoints"`
		Headers   map[string]string `yaml:"headers"`
		Timeouts  map[string]int    `yaml:"timeouts"`
		Features  map[string]bool   `yaml:"features"`
	}

	type ComplexConfig struct {
		App      ServiceConfig      `yaml:"app"`
		Database DatabaseConnection `yaml:"database"`
		Cache    CacheConfig        `yaml:"cache"`
		Services []ServiceConfig    `yaml:"services"`
	}

	yamlContent := `app:
  name: "main-service"
  endpoints:
    - "http://api.example.com"
    - "http://backup.example.com"
  headers:
    content-type: "application/json"
    user-agent: "test-client"
  timeouts:
    connect: 30
    read: 60
  features:
    auth: true
    logging: false
database:
  host: "db.example.com"
  port: 5432
  username: "admin"
  password: "secret"
  ssl: true
cache:
  redis:
    host: "redis.example.com"
    port: 6379
    username: ""
    password: "redis-secret"
    ssl: false
  ttl: 3600
services:
  - name: "auth-service"
    endpoints:
      - "http://auth.example.com"
    headers:
      authorization: "Bearer token"
    timeouts:
      connect: 10
    features:
      rate_limit: true
  - name: "notification-service"
    endpoints:
      - "http://notify1.example.com"
      - "http://notify2.example.com"
    headers:
      api-key: "notify-key"
    timeouts:
      connect: 5
      read: 30
    features:
      retry: true
      circuit_breaker: false`

	tmpFile, err := os.CreateTemp("", "complex-config-*.yaml")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	_, err = tmpFile.WriteString(yamlContent)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	envVars := map[string]string{
		"TEST_APP_NAME":          "overridden-service",
		"TEST_DATABASE_PASSWORD": "env-password",
		"TEST_CACHE_TTL":         "7200",
		"TEST_CACHE_REDIS_HOST":  "env-redis.example.com",
		"TEST_APP_ENDPOINTS":     "http://env1.example.com,http://env2.example.com",
		"TEST_APP_HEADERS":       "authorization=Bearer env-token,custom=env-value",
		"TEST_APP_TIMEOUTS":      "connect=15,read=45",
		"TEST_APP_FEATURES":      "auth=false,logging=true,debug=true",
	}

	for key, value := range envVars {
		require.NoError(t, os.Setenv(key, value))
	}
	defer func() {
		for key := range envVars {
			_ = os.Unsetenv(key)
		}
	}()

	var cfg ComplexConfig

	err = Load(&cfg, WithFile(tmpFile.Name()), WithEnv("TEST"))
	require.NoError(t, err)

	// Test overridden values
	assert.Equal(t, "overridden-service", cfg.App.Name)
	assert.Equal(t, []string{"http://env1.example.com", "http://env2.example.com"}, cfg.App.Endpoints)
	assert.Equal(t, "env-password", cfg.Database.Password)
	assert.Equal(t, "env-redis.example.com", cfg.Cache.Redis.Host)
	assert.Equal(t, 7200, cfg.Cache.TTL)

	// Test maps from env
	expectedHeaders := map[string]string{"authorization": "Bearer env-token", "custom": "env-value"}
	assert.Equal(t, expectedHeaders, cfg.App.Headers)

	expectedTimeouts := map[string]int{"connect": 15, "read": 45}
	assert.Equal(t, expectedTimeouts, cfg.App.Timeouts)

	expectedFeatures := map[string]bool{"auth": false, "logging": true, "debug": true}
	assert.Equal(t, expectedFeatures, cfg.App.Features)

	// Test values from file (not overridden)
	assert.Equal(t, "db.example.com", cfg.Database.Host)
	assert.Equal(t, 5432, cfg.Database.Port)
	assert.Equal(t, "admin", cfg.Database.Username)
	assert.True(t, cfg.Database.SSL)

	// Test slice from file
	assert.Len(t, cfg.Services, 2)
	assert.Equal(t, "auth-service", cfg.Services[0].Name)
	assert.Equal(t, "notification-service", cfg.Services[1].Name)
}

func TestLoad_ErrorConditions(t *testing.T) {
	testCases := []struct {
		name          string
		config        interface{}
		options       []Option
		expectedError string
	}{
		{
			name:          "nil config",
			config:        nil,
			options:       []Option{},
			expectedError: "config must be a non-nil pointer",
		},
		{
			name:          "non-pointer config",
			config:        TestConfig{},
			options:       []Option{},
			expectedError: "config must be a non-nil pointer",
		},
		{
			name:          "directory not file",
			config:        &TestConfig{},
			options:       []Option{WithFile("/tmp")},
			expectedError: "failed to load from files",
		},
		{
			name:          "custom default type mismatch",
			config:        &TestConfig{},
			options:       []Option{WithDefault("invalid type")},
			expectedError: "does not match config type",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := Load(tc.config, tc.options...)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tc.expectedError)
		})
	}
}

func TestLoad_TagPriority(t *testing.T) {
	type TagTestConfig struct {
		YamlOnly   string `yaml:"yaml_field"`
		JSONOnly   string `json:"json_field"`
		BothTags   string `yaml:"yaml_name" json:"json_name"`
		NoTags     string
		YamlIgnore string `yaml:"-" json:"json_not_ignored"`
		JSONIgnore string `yaml:"yaml_not_ignored" json:"-"`
	}

	envVars := map[string]string{
		"TEST_YAML_FIELD":       "yaml-value",
		"TEST_JSON_FIELD":       "json-value",
		"TEST_YAML_NAME":        "yaml-priority-value",
		"TEST_NOTAGS":           "field-name-value",
		"TEST_JSON_NOT_IGNORED": "json-not-ignored-value",
		"TEST_YAML_NOT_IGNORED": "yaml-not-ignored-value",
	}

	for key, value := range envVars {
		require.NoError(t, os.Setenv(key, value))
	}
	defer func() {
		for key := range envVars {
			_ = os.Unsetenv(key)
		}
	}()

	var cfg TagTestConfig

	err := Load(&cfg, WithEnv("TEST"))
	require.NoError(t, err)

	assert.Equal(t, "yaml-value", cfg.YamlOnly)
	assert.Equal(t, "json-value", cfg.JSONOnly)
	assert.Equal(t, "yaml-priority-value", cfg.BothTags)      // yaml takes priority
	assert.Equal(t, "field-name-value", cfg.NoTags)           // Uses lowercase field name
	assert.Equal(t, "json-not-ignored-value", cfg.YamlIgnore) // yaml:"-" ignores yaml tag, uses json tag
	assert.Equal(t, "yaml-not-ignored-value", cfg.JSONIgnore) // yaml takes priority over json:"-"
}

func TestLoad_UnsignedIntegerTypes(t *testing.T) {
	type UintConfig struct {
		Uint8Val  uint8  `yaml:"uint8_val"`
		Uint16Val uint16 `yaml:"uint16_val"`
		Uint32Val uint32 `yaml:"uint32_val"`
		Uint64Val uint64 `yaml:"uint64_val"`
		UintVal   uint   `yaml:"uint_val"`
	}

	envVars := map[string]string{
		"TEST_UINT8_VAL":  "255",
		"TEST_UINT16_VAL": "65535",
		"TEST_UINT32_VAL": "4294967295",
		"TEST_UINT64_VAL": "18446744073709551615",
		"TEST_UINT_VAL":   "123456789",
	}

	for key, value := range envVars {
		require.NoError(t, os.Setenv(key, value))
	}
	defer func() {
		for key := range envVars {
			_ = os.Unsetenv(key)
		}
	}()

	var cfg UintConfig

	err := Load(&cfg, WithEnv("TEST"))
	require.NoError(t, err)

	assert.Equal(t, uint8(255), cfg.Uint8Val)
	assert.Equal(t, uint16(65535), cfg.Uint16Val)
	assert.Equal(t, uint32(4294967295), cfg.Uint32Val)
	assert.Equal(t, uint64(18446744073709551615), cfg.Uint64Val)
	assert.Equal(t, uint(123456789), cfg.UintVal)
}

func TestLoad_IntegerOverflow(t *testing.T) {
	type OverflowConfig struct {
		SmallInt int8 `yaml:"small_int"`
	}

	require.NoError(t, os.Setenv("TEST_SMALL_INT", "999"))
	defer func() { _ = os.Unsetenv("TEST_SMALL_INT") }()

	var cfg OverflowConfig

	err := Load(&cfg, WithEnv("TEST"))
	require.NoError(t, err)

	// Go's strconv.ParseInt with int8 will parse 999 successfully but SetInt will overflow
	// The actual behavior depends on Go's implementation
	// This test ensures we don't panic on overflow
}

func TestLoad_EmptySlicesAndMaps(t *testing.T) {
	type EmptyConfig struct {
		EmptySlice []string          `yaml:"empty_slice"`
		EmptyMap   map[string]string `yaml:"empty_map"`
		NilSlice   []int             `yaml:"nil_slice"`
		NilMap     map[string]int    `yaml:"nil_map"`
	}

	envVars := map[string]string{
		"TEST_EMPTY_SLICE": "",
		"TEST_EMPTY_MAP":   "",
		// NilSlice and NilMap are not set
	}

	for key, value := range envVars {
		require.NoError(t, os.Setenv(key, value))
	}
	defer func() {
		for key := range envVars {
			_ = os.Unsetenv(key)
		}
	}()

	var cfg EmptyConfig

	err := Load(&cfg, WithEnv("TEST"))
	require.NoError(t, err)

	assert.Nil(t, cfg.EmptySlice) // empty string env var doesn't create slice
	assert.Nil(t, cfg.EmptyMap)   // empty string env var doesn't create map
	assert.Nil(t, cfg.NilSlice)
	assert.Nil(t, cfg.NilMap)
}

func TestLoad_PointerFields(t *testing.T) {
	type PointerConfig struct {
		StringPtr *string       `yaml:"string_ptr"`
		IntPtr    *int          `yaml:"int_ptr"`
		BoolPtr   *bool         `yaml:"bool_ptr"`
		NestedPtr *LoggerConfig `yaml:"nested_ptr"`
	}

	envVars := map[string]string{
		"TEST_STRING_PTR":       "pointer-value",
		"TEST_INT_PTR":          "42",
		"TEST_BOOL_PTR":         "true",
		"TEST_NESTED_PTR_LEVEL": "trace",
	}

	for key, value := range envVars {
		require.NoError(t, os.Setenv(key, value))
	}
	defer func() {
		for key := range envVars {
			_ = os.Unsetenv(key)
		}
	}()

	var cfg PointerConfig

	err := Load(&cfg, WithEnv("TEST"))
	require.NoError(t, err)

	require.NotNil(t, cfg.StringPtr)
	assert.Equal(t, "pointer-value", *cfg.StringPtr)

	require.NotNil(t, cfg.IntPtr)
	assert.Equal(t, 42, *cfg.IntPtr)

	require.NotNil(t, cfg.BoolPtr)
	assert.True(t, *cfg.BoolPtr)

	require.NotNil(t, cfg.NestedPtr)
	assert.Equal(t, "trace", cfg.NestedPtr.Level)
}

func TestLoad_ConcurrentLoadingSafety(t *testing.T) {
	const numGoroutines = 20
	const numIterations = 10

	done := make(chan bool, numGoroutines)
	errors := make(chan error, numGoroutines*numIterations)

	// Set up some environment variables
	envVars := map[string]string{
		"CONCURRENT_LOGGER_LEVEL":   "debug",
		"CONCURRENT_HEALTH_ADDRESS": ":8080",
	}

	for key, value := range envVars {
		require.NoError(t, os.Setenv(key, value))
	}
	defer func() {
		for key := range envVars {
			_ = os.Unsetenv(key)
		}
	}()

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer func() { done <- true }()

			for j := 0; j < numIterations; j++ {
				var cfg TestConfig
				err := Load(&cfg, WithEnv("CONCURRENT"))
				if err != nil {
					errors <- err
					return
				}

				if cfg.Logger.Level != "debug" || cfg.Health.Address != ":8080" {
					errors <- fmt.Errorf("unexpected config values: level=%s, address=%s",
						cfg.Logger.Level, cfg.Health.Address)
					return
				}
			}
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Check for errors
	close(errors)
	for err := range errors {
		t.Errorf("Concurrent loading error: %v", err)
	}
}

func TestLoad_LargeConfigurationFile(t *testing.T) {
	// Generate a large configuration with many fields
	type LargeSubConfig struct {
		Field1  string `yaml:"field1"`
		Field2  string `yaml:"field2"`
		Field3  string `yaml:"field3"`
		Field4  string `yaml:"field4"`
		Field5  string `yaml:"field5"`
		Field6  string `yaml:"field6"`
		Field7  string `yaml:"field7"`
		Field8  string `yaml:"field8"`
		Field9  string `yaml:"field9"`
		Field10 string `yaml:"field10"`
	}

	type LargeConfig struct {
		Section1  LargeSubConfig `yaml:"section1"`
		Section2  LargeSubConfig `yaml:"section2"`
		Section3  LargeSubConfig `yaml:"section3"`
		Section4  LargeSubConfig `yaml:"section4"`
		Section5  LargeSubConfig `yaml:"section5"`
		Section6  LargeSubConfig `yaml:"section6"`
		Section7  LargeSubConfig `yaml:"section7"`
		Section8  LargeSubConfig `yaml:"section8"`
		Section9  LargeSubConfig `yaml:"section9"`
		Section10 LargeSubConfig `yaml:"section10"`
	}

	// Build large YAML content
	var yamlBuilder strings.Builder
	for section := 1; section <= 10; section++ {
		yamlBuilder.WriteString(fmt.Sprintf("section%d:\n", section))
		for field := 1; field <= 10; field++ {
			yamlBuilder.WriteString(fmt.Sprintf("  field%d: \"value-s%d-f%d\"\n", field, section, field))
		}
	}

	tmpFile, err := os.CreateTemp("", "large-config-*.yaml")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	_, err = tmpFile.WriteString(yamlBuilder.String())
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	var cfg LargeConfig

	start := time.Now()
	err = Load(&cfg, WithFile(tmpFile.Name()))
	duration := time.Since(start)

	require.NoError(t, err)

	// Verify some values
	assert.Equal(t, "value-s1-f1", cfg.Section1.Field1)
	assert.Equal(t, "value-s5-f5", cfg.Section5.Field5)
	assert.Equal(t, "value-s10-f10", cfg.Section10.Field10)

	// Performance check - should load quickly
	assert.Less(t, duration, time.Second, "Large config should load in under 1 second")
}

type MacroTestConfig struct {
	BasicString   string            `yaml:"basic_string" json:"basic_string"`
	DatabaseURL   string            `yaml:"database_url" json:"database_url"`
	NestedConfig  MacroNestedConfig `yaml:"nested" json:"nested"`
	StringSlice   []string          `yaml:"string_slice" json:"string_slice"`
	StringMap     map[string]string `yaml:"string_map" json:"string_map"`
	NoMacroString string            `yaml:"no_macro" json:"no_macro"`
}

type MacroNestedConfig struct {
	Host string `yaml:"host" json:"host"`
	Port string `yaml:"port" json:"port"`
}

func TestExpandMacros_BasicString(t *testing.T) {
	// Set environment variables for testing
	require.NoError(t, os.Setenv("TEST_HOST", "localhost"))
	require.NoError(t, os.Setenv("TEST_PORT", "5432"))
	defer func() {
		_ = os.Unsetenv("TEST_HOST")
		_ = os.Unsetenv("TEST_PORT")
	}()

	yamlContent := `basic_string: "Server running on ${env:TEST_HOST}:${env:TEST_PORT}"
database_url: "postgres://user:pass@${env:TEST_HOST}:${env:TEST_PORT}/db"
no_macro: "plain string without macros"`

	tmpFile, err := os.CreateTemp("", "macro-config-*.yaml")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	_, err = tmpFile.WriteString(yamlContent)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	var cfg MacroTestConfig
	err = Load(&cfg, WithFile(tmpFile.Name()))
	require.NoError(t, err)

	assert.Equal(t, "Server running on localhost:5432", cfg.BasicString)
	assert.Equal(t, "postgres://user:pass@localhost:5432/db", cfg.DatabaseURL)
	assert.Equal(t, "plain string without macros", cfg.NoMacroString)
}

func TestExpandMacros_NestedStructs(t *testing.T) {
	require.NoError(t, os.Setenv("DB_HOST", "db.example.com"))
	require.NoError(t, os.Setenv("DB_PORT", "3306"))
	defer func() {
		_ = os.Unsetenv("DB_HOST")
		_ = os.Unsetenv("DB_PORT")
	}()

	yamlContent := `nested:
  host: "${env:DB_HOST}"
  port: "${env:DB_PORT}"`

	tmpFile, err := os.CreateTemp("", "nested-macro-*.yaml")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	_, err = tmpFile.WriteString(yamlContent)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	var cfg MacroTestConfig
	err = Load(&cfg, WithFile(tmpFile.Name()))
	require.NoError(t, err)

	assert.Equal(t, "db.example.com", cfg.NestedConfig.Host)
	assert.Equal(t, "3306", cfg.NestedConfig.Port)
}

func TestExpandMacros_StringSlices(t *testing.T) {
	require.NoError(t, os.Setenv("SERVER1", "web1.example.com"))
	require.NoError(t, os.Setenv("SERVER2", "web2.example.com"))
	defer func() {
		_ = os.Unsetenv("SERVER1")
		_ = os.Unsetenv("SERVER2")
	}()

	yamlContent := `string_slice:
  - "${env:SERVER1}"
  - "${env:SERVER2}"
  - "static.example.com"`

	tmpFile, err := os.CreateTemp("", "slice-macro-*.yaml")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	_, err = tmpFile.WriteString(yamlContent)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	var cfg MacroTestConfig
	err = Load(&cfg, WithFile(tmpFile.Name()))
	require.NoError(t, err)

	expected := []string{"web1.example.com", "web2.example.com", "static.example.com"}
	assert.Equal(t, expected, cfg.StringSlice)
}

func TestExpandMacros_StringMaps(t *testing.T) {
	require.NoError(t, os.Setenv("APP_ENV", "production"))
	require.NoError(t, os.Setenv("APP_VERSION", "1.2.3"))
	defer func() {
		_ = os.Unsetenv("APP_ENV")
		_ = os.Unsetenv("APP_VERSION")
	}()

	yamlContent := `string_map:
  environment: "${env:APP_ENV}"
  version: "${env:APP_VERSION}"
  static_key: "static_value"`

	tmpFile, err := os.CreateTemp("", "map-macro-*.yaml")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	_, err = tmpFile.WriteString(yamlContent)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	var cfg MacroTestConfig
	err = Load(&cfg, WithFile(tmpFile.Name()))
	require.NoError(t, err)

	expected := map[string]string{
		"environment": "production",
		"version":     "1.2.3",
		"static_key":  "static_value",
	}
	assert.Equal(t, expected, cfg.StringMap)
}

func TestExpandMacros_UndefinedEnvVar(t *testing.T) {
	// Ensure the environment variable is not set
	_ = os.Unsetenv("UNDEFINED_VAR")

	yamlContent := `basic_string: "Value with ${env:UNDEFINED_VAR} should remain unchanged"`

	tmpFile, err := os.CreateTemp("", "undefined-macro-*.yaml")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	_, err = tmpFile.WriteString(yamlContent)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	var cfg MacroTestConfig
	err = Load(&cfg, WithFile(tmpFile.Name()))
	require.NoError(t, err)

	// Should remain unchanged when env var is not set
	assert.Equal(t, "Value with ${env:UNDEFINED_VAR} should remain unchanged", cfg.BasicString)
}

func TestExpandMacros_EmptyEnvVar(t *testing.T) {
	require.NoError(t, os.Setenv("EMPTY_VAR", ""))
	defer func() { _ = os.Unsetenv("EMPTY_VAR") }()

	yamlContent := `basic_string: "Value with ${env:EMPTY_VAR} should remain unchanged"`

	tmpFile, err := os.CreateTemp("", "empty-macro-*.yaml")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	_, err = tmpFile.WriteString(yamlContent)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	var cfg MacroTestConfig
	err = Load(&cfg, WithFile(tmpFile.Name()))
	require.NoError(t, err)

	// Should remain unchanged when env var is empty
	assert.Equal(t, "Value with ${env:EMPTY_VAR} should remain unchanged", cfg.BasicString)
}

func TestExpandMacros_MultipleMacrosInOneString(t *testing.T) {
	require.NoError(t, os.Setenv("HOST", "example.com"))
	require.NoError(t, os.Setenv("PORT", "8080"))
	require.NoError(t, os.Setenv("PROTOCOL", "https"))
	defer func() {
		_ = os.Unsetenv("HOST")
		_ = os.Unsetenv("PORT")
		_ = os.Unsetenv("PROTOCOL")
	}()

	yamlContent := `basic_string: "${env:PROTOCOL}://${env:HOST}:${env:PORT}/api/v1"`

	tmpFile, err := os.CreateTemp("", "multiple-macro-*.yaml")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	_, err = tmpFile.WriteString(yamlContent)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	var cfg MacroTestConfig
	err = Load(&cfg, WithFile(tmpFile.Name()))
	require.NoError(t, err)

	assert.Equal(t, "https://example.com:8080/api/v1", cfg.BasicString)
}

func TestExpandMacros_JSONFormat(t *testing.T) {
	require.NoError(t, os.Setenv("JSON_HOST", "json.example.com"))
	require.NoError(t, os.Setenv("JSON_PORT", "9000"))
	defer func() {
		_ = os.Unsetenv("JSON_HOST")
		_ = os.Unsetenv("JSON_PORT")
	}()

	jsonContent := `{
  "basic_string": "Server at ${env:JSON_HOST}:${env:JSON_PORT}",
  "nested": {
    "host": "${env:JSON_HOST}",
    "port": "${env:JSON_PORT}"
  }
}`

	tmpFile, err := os.CreateTemp("", "macro-config-*.json")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	_, err = tmpFile.WriteString(jsonContent)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	var cfg MacroTestConfig
	err = Load(&cfg, WithFile(tmpFile.Name()))
	require.NoError(t, err)

	assert.Equal(t, "Server at json.example.com:9000", cfg.BasicString)
	assert.Equal(t, "json.example.com", cfg.NestedConfig.Host)
	assert.Equal(t, "9000", cfg.NestedConfig.Port)
}

func TestExpandMacros_MacroWithEnvOverride(t *testing.T) {
	// Set up environment variables for both macro expansion and env override
	require.NoError(t, os.Setenv("MACRO_HOST", "file-host.com"))
	require.NoError(t, os.Setenv("TEST_BASIC_STRING", "env-override-value"))
	defer func() {
		_ = os.Unsetenv("MACRO_HOST")
		_ = os.Unsetenv("TEST_BASIC_STRING")
	}()

	yamlContent := `basic_string: "host is ${env:MACRO_HOST}"`

	tmpFile, err := os.CreateTemp("", "macro-env-*.yaml")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	_, err = tmpFile.WriteString(yamlContent)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	var cfg MacroTestConfig
	err = Load(&cfg, WithFile(tmpFile.Name()), WithEnv("TEST"))
	require.NoError(t, err)

	// Environment variable should override the macro-expanded value
	assert.Equal(t, "env-override-value", cfg.BasicString)
}

func TestExpandMacros_InvalidMacroSyntax(t *testing.T) {
	yamlContent := `basic_string: "Invalid syntax ${env:} and ${env:MISSING_CLOSING"`

	tmpFile, err := os.CreateTemp("", "invalid-macro-*.yaml")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	_, err = tmpFile.WriteString(yamlContent)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	var cfg MacroTestConfig
	err = Load(&cfg, WithFile(tmpFile.Name()))
	require.NoError(t, err)

	// Invalid syntax should remain unchanged
	assert.Equal(t, "Invalid syntax ${env:} and ${env:MISSING_CLOSING", cfg.BasicString)
}
