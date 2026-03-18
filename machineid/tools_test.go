package machineid

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTrim(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", ""},
		{"no whitespace", "abc123", "abc123"},
		{"leading newline", "\nabc123", "abc123"},
		{"trailing newline", "abc123\n", "abc123"},
		{"both newlines", "\nabc123\n", "abc123"},
		{"leading spaces", "  abc123", "abc123"},
		{"trailing spaces", "abc123  ", "abc123"},
		{"mixed whitespace", "\n  abc123  \n", "abc123"},
		{"uuid format", "550e8400-e29b-41d4-a716-446655440000\n", "550e8400-e29b-41d4-a716-446655440000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := trim(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
