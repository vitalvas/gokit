//go:build darwin
// +build darwin

package machineid

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func machineID() (string, error) {
	c := exec.Command("ioreg", "-rd1", "-c", "IOPlatformExpertDevice")
	var stdout, stdin bytes.Buffer
	c.Stdin = &stdin
	c.Stdout = &stdout
	c.Stderr = os.Stderr

	if err := c.Run(); err != nil {
		return "", fmt.Errorf("failed to request ioreg: %w", err)
	}

	id, err := extractID(stdout.String())
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

func trim(s string) string {
	return strings.TrimSpace(strings.Trim(s, "\n"))
}
