package xstrings

import (
	"reflect"
	"testing"
)

func TestChunks(t *testing.T) {
	tests := []struct {
		name      string
		inputList []string
		chunkSize int
		expected  [][]string
	}{
		{
			name:      "Test Case 1",
			inputList: []string{"a", "b", "c", "d", "e"},
			chunkSize: 2,
			expected:  [][]string{{"a", "b"}, {"c", "d"}, {"e"}},
		},
		{
			name:      "Test Case 2",
			inputList: []string{"apple", "banana", "cherry", "date", "elderberry"},
			chunkSize: 3,
			expected:  [][]string{{"apple", "banana", "cherry"}, {"date", "elderberry"}},
		},
		{
			name:      "Test Case 3",
			inputList: []string{"one", "two", "three"},
			chunkSize: 1,
			expected:  [][]string{{"one"}, {"two"}, {"three"}},
		},
		{
			name:      "Test Case 4",
			inputList: []string{"x", "y", "z"},
			chunkSize: 5,
			expected:  [][]string{{"x", "y", "z"}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := Chunks(test.inputList, test.chunkSize)
			if !reflect.DeepEqual(result, test.expected) {
				t.Errorf("Expected %v, but got %v", test.expected, result)
			}
		})
	}
}
