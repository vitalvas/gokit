package machineid

import (
	"io"
	"os"
	"os/exec"
	"strings"
)

func run(stdout, stderr io.Writer, cmd string, args ...string) error {
	c := exec.Command(cmd, args...)
	c.Stdin = os.Stdin
	c.Stdout = stdout
	c.Stderr = stderr
	return c.Run()
}

func trim(s string) string {
	return strings.TrimSpace(strings.Trim(s, "\n"))
}
