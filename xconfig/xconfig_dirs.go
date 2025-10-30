package xconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

func scanDirectory(dirname string) ([]string, error) {
	var configFiles []string

	entries, err := os.ReadDir(dirname)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // Directory doesn't exist, return empty list
		}
		return nil, fmt.Errorf("failed to read directory %s: %w", dirname, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue // Skip subdirectories
		}

		filename := entry.Name()
		if isConfigFile(filename) {
			fullPath := filepath.Join(dirname, filename)
			configFiles = append(configFiles, fullPath)
		}
	}

	// Sort files for deterministic loading order
	sort.Strings(configFiles)
	return configFiles, nil
}

func loadFromDirs(config interface{}, dirnames []string) error {
	var allFiles []string

	for _, dirname := range dirnames {
		files, err := scanDirectory(dirname)
		if err != nil {
			return fmt.Errorf("failed to scan directory %s: %w", dirname, err)
		}
		allFiles = append(allFiles, files...)
	}

	return loadFromFiles(config, allFiles)
}
