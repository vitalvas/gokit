package otp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAlgorithm(t *testing.T) {
	t.Run("names", func(t *testing.T) {
		assert.Equal(t, "SHA1", SHA1.String())
		assert.Equal(t, "SHA256", SHA256.String())
		assert.Equal(t, "SHA512", SHA512.String())
		assert.Equal(t, "SHA1", Algorithm(42).String())
	})

	t.Run("hash sizes", func(t *testing.T) {
		assert.Equal(t, 20, SHA1.hash()().Size())
		assert.Equal(t, 32, SHA256.hash()().Size())
		assert.Equal(t, 64, SHA512.hash()().Size())
		assert.Equal(t, 20, Algorithm(42).hash()().Size())
	})
}
