package xconfig

import (
	"fmt"
	"os"
	"strings"
)

func parseDotenvFile(filename string) (map[string]string, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	envVars := make(map[string]string)
	lines := strings.Split(string(data), "\n")

	for lineNum, line := range lines {
		// Trim whitespace
		line = strings.TrimSpace(line)

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Find the first = to split key and value
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid format at line %d: missing '=' in %q", lineNum+1, line)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Validate key (must not be empty and contain only valid characters)
		if key == "" {
			return nil, fmt.Errorf("invalid format at line %d: empty key", lineNum+1)
		}

		// Strip surrounding quotes (single or double)
		value = stripQuotes(value)

		envVars[key] = value
	}

	return envVars, nil
}

func stripQuotes(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

func loadDotenvFiles(filenames []string) error {
	for _, filename := range filenames {
		envVars, err := parseDotenvFile(filename)
		if err != nil {
			return fmt.Errorf("failed to parse dotenv file %s: %w", filename, err)
		}

		// Set environment variables (override existing ones as per requirement)
		for key, value := range envVars {
			if err := os.Setenv(key, value); err != nil {
				return fmt.Errorf("failed to set environment variable %s: %w", key, err)
			}
		}
	}

	return nil
}
