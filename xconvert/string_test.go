package xconvert

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStringToPointer(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "hello"},
		{"", ""},
		{"123", "123"},
	}

	for _, test := range tests {
		result := StringToPointer(test.input)
		assert.NotNil(t, result)
		assert.Equal(t, test.expected, *result)

		if result == nil || *result != test.expected {
			t.Errorf("StringToPointer(%q) = %v; want %v", test.input, result, test.expected)
		}
	}
}
