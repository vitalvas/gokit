package xstrings

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetConsumerTag(t *testing.T) {
	t.Run("returns valid tag", func(t *testing.T) {
		tag, err := GetConsumerTag()
		require.NoError(t, err)
		assert.NotEmpty(t, tag)
		assert.Contains(t, tag, "-")
	})

	t.Run("returns unique tags", func(t *testing.T) {
		tag1, err := GetConsumerTag()
		require.NoError(t, err)
		tag2, err := GetConsumerTag()
		require.NoError(t, err)
		assert.NotEqual(t, tag1, tag2)
	})
}
