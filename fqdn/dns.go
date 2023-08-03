package fqdn

import "strings"

func NormalizeDNSZoneName(name string) string {
	if !strings.HasSuffix(name, ".") {
		name += "."
	}

	return name
}
