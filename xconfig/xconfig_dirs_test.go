package xconfig

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScanDirectory(t *testing.T) {
	t.Run("directory with config files", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "xconfig-test-*")
		require.NoError(t, err)
		defer func() { _ = os.RemoveAll(tmpDir) }()

		require.NoError(t, os.WriteFile(tmpDir+"/01-base.yaml", []byte("test: value"), 0644))
		require.NoError(t, os.WriteFile(tmpDir+"/02-override.json", []byte("{}"), 0644))
		require.NoError(t, os.WriteFile(tmpDir+"/readme.txt", []byte("ignore"), 0644))

		files, err := scanDirectory(tmpDir)
		require.NoError(t, err)

		assert.Len(t, files, 2)
		assert.Contains(t, files[0], "01-base.yaml")
		assert.Contains(t, files[1], "02-override.json")
	})

	t.Run("nonexistent directory", func(t *testing.T) {
		files, err := scanDirectory("/nonexistent/path")
		require.NoError(t, err)
		assert.Nil(t, files)
	})

	t.Run("empty directory", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "xconfig-test-*")
		require.NoError(t, err)
		defer func() { _ = os.RemoveAll(tmpDir) }()

		files, err := scanDirectory(tmpDir)
		require.NoError(t, err)
		assert.Empty(t, files)
	})
}

func TestLoadFromDirs(t *testing.T) {
	t.Run("single directory", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "xconfig-test-*")
		require.NoError(t, err)
		defer func() { _ = os.RemoveAll(tmpDir) }()

		file1Content := `logger:
  level: "debug"
health:
  address: ":9090"`
		require.NoError(t, os.WriteFile(tmpDir+"/01-base.yaml", []byte(file1Content), 0644))

		file2Content := `logger:
  level: "info"
db:
  host: "dbserver"`
		require.NoError(t, os.WriteFile(tmpDir+"/02-override.yaml", []byte(file2Content), 0644))

		require.NoError(t, os.WriteFile(tmpDir+"/readme.txt", []byte("ignore me"), 0644))

		var cfg TestConfig
		err = Load(&cfg, WithDirs(tmpDir))
		require.NoError(t, err)

		assert.Equal(t, "info", cfg.Logger.Level)
		assert.Equal(t, ":9090", cfg.Health.Address)
		assert.Equal(t, "dbserver", cfg.DB.Host)
	})

	t.Run("directories and files combined", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "xconfig-test-*")
		require.NoError(t, err)
		defer func() { _ = os.RemoveAll(tmpDir) }()

		dirContent := `logger:
  level: "debug"`
		require.NoError(t, os.WriteFile(tmpDir+"/config.yaml", []byte(dirContent), 0644))

		tmpFile, err := os.CreateTemp("", "explicit-config-*.yaml")
		require.NoError(t, err)
		defer func() { _ = os.Remove(tmpFile.Name()) }()

		fileContent := `logger:
  level: "warn"
health:
  address: ":8888"`
		_, err = tmpFile.WriteString(fileContent)
		require.NoError(t, err)
		require.NoError(t, tmpFile.Close())

		var cfg TestConfig
		err = Load(&cfg, WithDirs(tmpDir), WithFiles(tmpFile.Name()))
		require.NoError(t, err)

		assert.Equal(t, "warn", cfg.Logger.Level)
		assert.Equal(t, ":8888", cfg.Health.Address)
	})

	t.Run("nonexistent directory", func(t *testing.T) {
		var cfg TestConfig
		err := Load(&cfg, WithDirs("/nonexistent/path"))
		require.NoError(t, err)
	})
}
