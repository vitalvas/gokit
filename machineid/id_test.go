package machineid

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestID(t *testing.T) {
	t.Run("returns non-empty string", func(t *testing.T) {
		id, err := ID()
		require.NoError(t, err)
		assert.NotEmpty(t, id)
	})

	t.Run("returns consistent value", func(t *testing.T) {
		id1, err1 := ID()
		require.NoError(t, err1)

		id2, err2 := ID()
		require.NoError(t, err2)

		assert.Equal(t, id1, id2)
	})
}

func TestIDOnce(t *testing.T) {
	t.Run("returns non-empty string", func(t *testing.T) {
		id, err := IDOnce()
		require.NoError(t, err)
		assert.NotEmpty(t, id)
	})

	t.Run("returns same value on multiple calls", func(t *testing.T) {
		id1, err1 := IDOnce()
		require.NoError(t, err1)

		id2, err2 := IDOnce()
		require.NoError(t, err2)

		assert.Equal(t, id1, id2)
	})

	t.Run("matches ID function", func(t *testing.T) {
		idOnce, err1 := IDOnce()
		require.NoError(t, err1)

		id, err2 := ID()
		require.NoError(t, err2)

		assert.Equal(t, idOnce, id)
	})
}

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

func BenchmarkID(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_, _ = ID()
	}
}

func BenchmarkIDOnce(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_, _ = IDOnce()
	}
}
