package xstrings

import "testing"

func TestSimpleTemplate(t *testing.T) {
	tests := []struct {
		name     string
		template string
		data     map[string]string
		expected string
	}{
		{
			name:     "basic replacement",
			template: "Hello, {{ name }}!",
			data:     map[string]string{"name": "John"},
			expected: "Hello, John!",
		},
		{
			name:     "multiple replacements",
			template: "Welcome to {{ location }} in {{ year }}.",
			data:     map[string]string{"location": "Golang World", "year": "2024"},
			expected: "Welcome to Golang World in 2024.",
		},
		{
			name:     "no placeholders",
			template: "No placeholders here.",
			data:     map[string]string{"key": "value"},
			expected: "No placeholders here.",
		},
		{
			name:     "placeholder not found",
			template: "Hello, {{ name }}!",
			data:     map[string]string{"location": "Golang World"},
			expected: "Hello, {{ name }}!",
		},
		{
			name:     "placeholders with spaces",
			template: "Hello, {{  name  }}! Welcome to {{            location }}.",
			data:     map[string]string{"name": "John", "location": "Golang World"},
			expected: "Hello, John! Welcome to Golang World.",
		},
		{
			name:     "empty data",
			template: "Hello, {{ name }}!",
			data:     map[string]string{},
			expected: "Hello, {{ name }}!",
		},
		{
			name:     "placeholder with special characters",
			template: "Hello, {{ name }}! Welcome to {{ location }}.",
			data:     map[string]string{"name": "John", "location": "Golang World & Go Playground"},
			expected: "Hello, John! Welcome to Golang World & Go Playground.",
		},
		{
			name:     "placeholder without spaces on the left",
			template: "Hello, {{name }}! Welcome to {{ location }}.",
			data:     map[string]string{"name": "John", "location": "Golang World"},
			expected: "Hello, John! Welcome to Golang World.",
		},
		{
			name:     "placeholder without spaces on the right",
			template: "Hello, {{ name}}! Welcome to {{ location }}.",
			data:     map[string]string{"name": "John", "location": "Golang World"},
			expected: "Hello, John! Welcome to Golang World.",
		},
		{
			name:     "placeholder without spaces on both sides",
			template: "Hello, {{name}}! Welcome to {{ location }}.",
			data:     map[string]string{"name": "John", "location": "Golang World"},
			expected: "Hello, John! Welcome to Golang World.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SimpleTemplate(tt.template, tt.data)
			if result != tt.expected {
				t.Errorf("SimpleTemplate() = %v, want %v", result, tt.expected)
			}
		})
	}
}
