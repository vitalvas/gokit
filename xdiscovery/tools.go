package xdiscovery

import "fmt"

func SrvJoinHostPort(input []SrvDiscoveredHost) []string {
	hosts := make([]string, 0, len(input))
	for _, row := range input {
		hosts = append(hosts, fmt.Sprintf("%s:%d", row.Target, row.Port))
	}

	return hosts
}
