package fqdn

import "strings"

func GetDomainFromHostname(name string) *string {
	elems := strings.SplitN(name, ".", 2)
	if len(elems) != 2 {
		return nil
	}

	return &elems[1]
}

func GetDomainNameGuesses(name string) []string {
	var names []string
	var nameSplited string

	names = append(names, name)
	nameSplited = name

	for strings.Contains(nameSplited, ".") {
		result := strings.SplitN(nameSplited, ".", 2)
		nameSplited = result[1]
		names = append(names, result[1])
	}

	return names
}
