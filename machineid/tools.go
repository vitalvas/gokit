package machineid

import "strings"

func trim(s string) string {
	return strings.TrimSpace(strings.Trim(s, "\n"))
}
