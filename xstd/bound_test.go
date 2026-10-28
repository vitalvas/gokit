package xstd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBound(t *testing.T) {
	inc := Included(5)
	assert.Equal(t, BoundIncluded, inc.Kind)
	assert.Equal(t, 5, inc.Value)

	exc := Excluded("x")
	assert.Equal(t, BoundExcluded, exc.Kind)
	assert.Equal(t, "x", exc.Value)

	unb := Unbounded[int]()
	assert.Equal(t, BoundUnbounded, unb.Kind)
	assert.Zero(t, unb.Value)
}

func TestBoundPredicates(t *testing.T) {
	t.Run("lower", func(t *testing.T) {
		assert.True(t, afterLower(5, Unbounded[int]()))
		assert.True(t, afterLower(5, Included(5)))
		assert.False(t, afterLower(4, Included(5)))
		assert.False(t, afterLower(5, Excluded(5)))
		assert.True(t, afterLower(6, Excluded(5)))
	})
	t.Run("upper", func(t *testing.T) {
		assert.True(t, beforeUpper(5, Unbounded[int]()))
		assert.True(t, beforeUpper(5, Included(5)))
		assert.False(t, beforeUpper(6, Included(5)))
		assert.False(t, beforeUpper(5, Excluded(5)))
		assert.True(t, beforeUpper(4, Excluded(5)))
	})
}
