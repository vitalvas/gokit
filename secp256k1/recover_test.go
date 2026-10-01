package secp256k1

import (
	"crypto/sha256"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecoverPubKey(t *testing.T) {
	// Exercise several keys so both nonce-point parities are hit.
	for i := 0; i < 8; i++ {
		priv, err := GeneratePrivateKey()
		require.NoError(t, err)

		digest := sha256.Sum256([]byte{byte(i), 'r', 'e', 'c'})

		r, s, recID := SignRecoverable(priv, digest[:])
		require.True(t, Verify(&priv.Pub, digest[:], r, s))

		recovered, err := RecoverPubKey(digest[:], r, s, recID)
		require.NoError(t, err)
		assert.True(t, priv.Pub.Equal(recovered), "recovered key must match signer")
	}
}

func TestRecoverPubKeyEdgeCases(t *testing.T) {
	priv, err := GeneratePrivateKey()
	require.NoError(t, err)
	digest := sha256.Sum256([]byte("edge"))
	r, s, recID := SignRecoverable(priv, digest[:])

	t.Run("wrong recovery id recovers a different key", func(t *testing.T) {
		other, err := RecoverPubKey(digest[:], r, s, recID^1)
		// Either it recovers a different (valid) point or fails; it must not
		// return the signer's key.
		if err == nil {
			assert.False(t, priv.Pub.Equal(other))
		}
	})

	t.Run("out-of-range recovery id rejected", func(t *testing.T) {
		_, err := RecoverPubKey(digest[:], r, s, 4)
		require.Error(t, err)
		_, err = RecoverPubKey(digest[:], r, s, -1)
		require.Error(t, err)
	})

	t.Run("out-of-range signature values rejected", func(t *testing.T) {
		_, err := RecoverPubKey(digest[:], big.NewInt(0), s, recID)
		require.Error(t, err)
		_, err = RecoverPubKey(digest[:], r, new(big.Int).Set(N()), recID)
		require.Error(t, err)
		_, err = RecoverPubKey(digest[:], nil, s, recID)
		require.Error(t, err)
	})
}

func BenchmarkSignRecoverable(b *testing.B) {
	priv, err := GeneratePrivateKey()
	require.NoError(b, err)
	digest := sha256.Sum256([]byte("benchmark message"))

	b.ResetTimer()
	for b.Loop() {
		SignRecoverable(priv, digest[:])
	}
}

func BenchmarkRecoverPubKey(b *testing.B) {
	priv, err := GeneratePrivateKey()
	require.NoError(b, err)
	digest := sha256.Sum256([]byte("benchmark message"))
	r, s, recID := SignRecoverable(priv, digest[:])

	b.ResetTimer()
	for b.Loop() {
		_, _ = RecoverPubKey(digest[:], r, s, recID)
	}
}

// FuzzRecoverPubKey feeds arbitrary r/s/recID to the recovery routine, which
// must never panic. Any key it returns must lie on the curve.
func FuzzRecoverPubKey(f *testing.F) {
	priv, err := GeneratePrivateKey()
	require.NoError(f, err)
	digest := sha256.Sum256([]byte("recover fuzz seed"))
	r, s, recID := SignRecoverable(priv, digest[:])

	f.Add(digest[:], fixedBytes(r), fixedBytes(s), recID)
	f.Add([]byte{}, []byte{}, []byte{}, 0)
	f.Add(digest[:], []byte{0x01}, []byte{0x01}, 5)

	f.Fuzz(func(t *testing.T, dgst, rBytes, sBytes []byte, id int) {
		pub, err := RecoverPubKey(dgst, new(big.Int).SetBytes(rBytes), new(big.Int).SetBytes(sBytes), id)
		if err != nil {
			return
		}

		if !pub.IsValid() {
			t.Fatalf("recovered key is not on the curve")
		}
	})
}
