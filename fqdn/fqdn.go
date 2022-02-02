package fqdn

import "strings"

func GetDomainFromHostname(name string) *string {
	elems := strings.SplitN(name, ".", 2)
	if len(elems) != 2 {
		return nil
	}

	return &elems[1]
}
