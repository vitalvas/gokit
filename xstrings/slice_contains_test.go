package xstrings

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSliceContain(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		assert.True(t, SliceContain([]string{"a", "b", "c"}, "b"))
	})

	t.Run("not found", func(t *testing.T) {
		assert.False(t, SliceContain([]string{"a", "b", "c"}, "d"))
	})

	t.Run("empty slice", func(t *testing.T) {
		assert.False(t, SliceContain([]string{}, "a"))
	})

	t.Run("nil slice", func(t *testing.T) {
		assert.False(t, SliceContain(nil, "a"))
	})

	t.Run("empty key", func(t *testing.T) {
		assert.False(t, SliceContain([]string{"a", "b"}, ""))
	})

	t.Run("empty key in slice with empty", func(t *testing.T) {
		assert.True(t, SliceContain([]string{"a", "", "b"}, ""))
	})
}
