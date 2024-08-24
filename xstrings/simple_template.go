package xstrings

import (
	"regexp"
	"strings"
)

func SimpleTemplate(template string, data map[string]string) string {
	re := regexp.MustCompile(`{{\s*(\w+)\s*}}`)

	replace := func(placeholder string) string {
		key := strings.TrimSpace(re.FindStringSubmatch(placeholder)[1])
		if value, exists := data[key]; exists {
			return value
		}
		return placeholder
	}

	return re.ReplaceAllStringFunc(template, replace)
}
