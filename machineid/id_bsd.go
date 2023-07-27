//go:build freebsd || netbsd || openbsd || dragonfly || solaris
// +build freebsd netbsd openbsd dragonfly solaris

package machineid

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
)

const hostidPath = "/etc/hostid"

func machineID() (string, error) {
	id, err := readHostID()
	if err != nil {
		// try fallback
		id, err = readKenv()
		if err != nil {
			return "", err
		}
	}

	return id, nil
}

func readHostID() (string, error) {
	buf, err := os.ReadFile(hostidPath)
	if err != nil {
		return "", err
	}
	return trim(string(buf)), nil
}

func readKenv() (string, error) {
	c := exec.Command("kenv", "-q", "smbios.system.uuid")
	var stdout, stdin bytes.Buffer
	c.Stdin = &stdin
	c.Stdout = &stdout
	c.Stderr = os.Stderr

	if err := c.Run(); err != nil {
		return "", fmt.Errorf("failed to request kenv: %w", err)
	}

	return trim(stdout.String()), nil
}

func trim(s string) string {
	return strings.TrimSpace(strings.Trim(s, "\n"))
}
