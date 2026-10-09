package xjwt

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConcatKDF(t *testing.T) {
	z := []byte("shared-secret-z")
	algID := []byte("A128GCM")
	partyU := []byte("Alice")
	partyV := []byte("Bob")

	t.Run("deterministic", func(t *testing.T) {
		a := concatKDF(z, 16, algID, partyU, partyV)
		b := concatKDF(z, 16, algID, partyU, partyV)
		assert.Equal(t, a, b)
	})

	t.Run("output length matches request", func(t *testing.T) {
		// 16 fits in a single SHA-256 block; 48 crosses the 32-byte boundary.
		for _, keyLen := range []int{16, 48} {
			out := concatKDF(z, keyLen, algID, partyU, partyV)
			require.Len(t, out, keyLen)
		}
	})

	t.Run("different algID yields different output", func(t *testing.T) {
		a := concatKDF(z, 16, []byte("A128GCM"), partyU, partyV)
		b := concatKDF(z, 16, []byte("A256GCM"), partyU, partyV)
		assert.NotEqual(t, a, b)
	})
}
