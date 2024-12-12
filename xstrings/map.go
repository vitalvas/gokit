package xstrings

import (
	"sort"
	"strings"
)

func SortMap(data map[string]string) map[string]string {
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	var out = make(map[string]string)

	for _, row := range keys {
		out[row] = data[row]
	}

	return out
}

func ReplaceMap(payload string, data map[string]string) string {
	for k, v := range data {
		payload = strings.ReplaceAll(payload, k, v)
	}

	return payload
}
