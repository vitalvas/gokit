//go:build darwin
// +build darwin

package machineid

import (
	"bytes"
	"errors"
	"os"
	"strings"
)

func machineID() (string, error) {
	buf := &bytes.Buffer{}
	err := run(buf, os.Stderr, "ioreg", "-rd1", "-c", "IOPlatformExpertDevice")
	if err != nil {
		return "", err
	}

	id, err := extractID(buf.String())
	if err != nil {
		return "", err
	}

	return trim(id), nil
}

func extractID(lines string) (string, error) {
	for _, line := range strings.Split(lines, "\n") {
		if strings.Contains(line, "IOPlatformUUID") {
			parts := strings.SplitAfter(line, `" = "`)
			if len(parts) == 2 {
				return strings.TrimRight(parts[1], `"`), nil
			}
		}
	}

	return "", errors.New("failed to extract 'IOPlatformUUID' value from `ioreg` output")
}
