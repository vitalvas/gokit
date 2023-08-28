package xdiscovery

import "testing"

func TestSrvJoinHostPort(t *testing.T) {
	testCases := []struct {
		name     string
		input    []SrvDiscoveredHost
		expected []string
	}{
		{
			name:     "EmptyInput",
			input:    []SrvDiscoveredHost{},
			expected: []string{},
		},
		{
			name: "SingleHost",
			input: []SrvDiscoveredHost{
				{Target: "localhost", Port: 8080},
			},
			expected: []string{"localhost:8080"},
		},
		{
			name: "MultipleHosts",
			input: []SrvDiscoveredHost{
				{Target: "host1", Port: 80},
				{Target: "host2", Port: 443},
			},
			expected: []string{"host1:80", "host2:443"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := SrvJoinHostPort(tc.input)
			if len(result) != len(tc.expected) {
				t.Errorf("Expected %d hosts, but got %d", len(tc.expected), len(result))
			}

			for i := 0; i < len(result); i++ {
				if result[i] != tc.expected[i] {
					t.Errorf("Expected host %s, but got %s", tc.expected[i], result[i])
				}
			}
		})
	}
}
