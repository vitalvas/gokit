package secp256k1

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestECDH(t *testing.T) {
	alice, err := GeneratePrivateKey()
	require.NoError(t, err)
	bob, err := GeneratePrivateKey()
	require.NoError(t, err)

	t.Run("shared secret agrees both directions", func(t *testing.T) {
		ab, err := alice.ECDH(&bob.Pub)
		require.NoError(t, err)
		ba, err := bob.ECDH(&alice.Pub)
		require.NoError(t, err)

		require.Len(t, ab, 32)
		assert.Equal(t, ab, ba)
	})

	t.Run("different peer yields different secret", func(t *testing.T) {
		carol, err := GeneratePrivateKey()
		require.NoError(t, err)

		ab, err := alice.ECDH(&bob.Pub)
		require.NoError(t, err)
		ac, err := alice.ECDH(&carol.Pub)
		require.NoError(t, err)

		assert.NotEqual(t, ab, ac)
	})

	t.Run("off-curve peer rejected", func(t *testing.T) {
		_, err := alice.ECDH(&PublicKey{X: big.NewInt(1), Y: big.NewInt(1)})
		require.Error(t, err)
	})

	t.Run("nil private key rejected", func(t *testing.T) {
		var p *PrivateKey
		_, err := p.ECDH(&bob.Pub)
		require.Error(t, err)
	})
}

func BenchmarkECDH(b *testing.B) {
	alice, err := GeneratePrivateKey()
	require.NoError(b, err)
	bob, err := GeneratePrivateKey()
	require.NoError(b, err)

	b.ResetTimer()
	for b.Loop() {
		_, _ = alice.ECDH(&bob.Pub)
	}
}
