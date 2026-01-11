package xstrings

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSimpleTemplate(t *testing.T) {
	t.Run("basic replacement", func(t *testing.T) {
		result := SimpleTemplate("Hello, {{ name }}!", map[string]string{"name": "John"})
		assert.Equal(t, "Hello, John!", result)
	})

	t.Run("multiple replacements", func(t *testing.T) {
		result := SimpleTemplate("Welcome to {{ location }} in {{ year }}.", map[string]string{"location": "Golang World", "year": "2024"})
		assert.Equal(t, "Welcome to Golang World in 2024.", result)
	})

	t.Run("no placeholders", func(t *testing.T) {
		result := SimpleTemplate("No placeholders here.", map[string]string{"key": "value"})
		assert.Equal(t, "No placeholders here.", result)
	})

	t.Run("placeholder not found", func(t *testing.T) {
		result := SimpleTemplate("Hello, {{ name }}!", map[string]string{"location": "Golang World"})
		assert.Equal(t, "Hello, {{ name }}!", result)
	})

	t.Run("empty data", func(t *testing.T) {
		result := SimpleTemplate("Hello, {{ name }}!", map[string]string{})
		assert.Equal(t, "Hello, {{ name }}!", result)
	})

	t.Run("placeholder without spaces", func(t *testing.T) {
		result := SimpleTemplate("Hello, {{name}}!", map[string]string{"name": "John"})
		assert.Equal(t, "Hello, John!", result)
	})
}

func BenchmarkSimpleTemplate(b *testing.B) {
	template := "Hello, {{ name }}! Welcome to {{ location }}."
	data := map[string]string{"name": "John", "location": "Golang World"}
	b.ReportAllocs()
	for b.Loop() {
		_ = SimpleTemplate(template, data)
	}
}

func FuzzSimpleTemplate(f *testing.F) {
	f.Add("Hello, {{ name }}!", "name", "John")
	f.Add("No placeholders", "key", "value")
	f.Add("{{ a }} and {{ b }}", "a", "X")

	f.Fuzz(func(_ *testing.T, template, key, value string) {
		data := map[string]string{key: value}
		_ = SimpleTemplate(template, data)
	})
}
