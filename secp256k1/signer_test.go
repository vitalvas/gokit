package secp256k1

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/sha256"
	"encoding/asn1"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSignDERRoundTrip(t *testing.T) {
	priv, err := GeneratePrivateKey()
	require.NoError(t, err)

	digest := sha256.Sum256([]byte("message to sign"))

	der, err := SignDER(priv, digest[:])
	require.NoError(t, err)
	assert.True(t, VerifyDER(&priv.Pub, digest[:], der))

	t.Run("wrong digest fails", func(t *testing.T) {
		other := sha256.Sum256([]byte("different"))
		assert.False(t, VerifyDER(&priv.Pub, other[:], der))
	})

	t.Run("garbage DER fails", func(t *testing.T) {
		assert.False(t, VerifyDER(&priv.Pub, digest[:], []byte{0x30, 0x00}))
		assert.False(t, VerifyDER(&priv.Pub, digest[:], nil))
	})

	t.Run("trailing bytes rejected", func(t *testing.T) {
		assert.False(t, VerifyDER(&priv.Pub, digest[:], append(der, 0x00)))
	})

	t.Run("strict rejects malleated DER", func(t *testing.T) {
		r, s := Sign(priv, digest[:])
		highS := new(big.Int).Sub(orderN, s)
		malleated, err := asn1.Marshal(ecdsaSignature{R: r, S: highS})
		require.NoError(t, err)

		assert.True(t, VerifyDER(&priv.Pub, digest[:], malleated))
		assert.False(t, VerifyDERStrict(&priv.Pub, digest[:], malleated))
		assert.True(t, VerifyDERStrict(&priv.Pub, digest[:], der))
		assert.False(t, VerifyDERStrict(&priv.Pub, digest[:], []byte{0x30, 0x00}))
	})
}

func TestCryptoSigner(t *testing.T) {
	priv, err := GeneratePrivateKey()
	require.NoError(t, err)

	// PrivateKey must satisfy the standard crypto.Signer interface.
	var signer crypto.Signer = priv

	pub, ok := signer.Public().(*PublicKey)
	require.True(t, ok)
	assert.Equal(t, 0, pub.X.Cmp(priv.Pub.X))

	digest := sha256.Sum256([]byte("via crypto.Signer"))

	sig, err := signer.Sign(rand.Reader, digest[:], crypto.SHA256)
	require.NoError(t, err)

	// The signer emits DER, verifiable via VerifyDER.
	assert.True(t, VerifyDER(pub, digest[:], sig))
}

func BenchmarkSignDER(b *testing.B) {
	priv, err := GeneratePrivateKey()
	require.NoError(b, err)
	digest := sha256.Sum256([]byte("benchmark message"))

	b.ResetTimer()
	for b.Loop() {
		_, _ = SignDER(priv, digest[:])
	}
}

func BenchmarkVerifyDER(b *testing.B) {
	priv, err := GeneratePrivateKey()
	require.NoError(b, err)
	digest := sha256.Sum256([]byte("benchmark message"))
	sig, err := SignDER(priv, digest[:])
	require.NoError(b, err)

	b.ResetTimer()
	for b.Loop() {
		VerifyDER(&priv.Pub, digest[:], sig)
	}
}

// FuzzVerifyDER feeds arbitrary DER signature bytes to the verifier; it must
// never panic and must reject anything but the genuine seeded signature.
func FuzzVerifyDER(f *testing.F) {
	priv, err := GeneratePrivateKey()
	require.NoError(f, err)
	digest := sha256.Sum256([]byte("der fuzz seed"))
	sig, err := SignDER(priv, digest[:])
	require.NoError(f, err)

	f.Add(digest[:], sig)
	f.Add([]byte{}, []byte{})
	f.Add(digest[:], []byte{0x30, 0x00})

	f.Fuzz(func(t *testing.T, dgst, s []byte) {
		if VerifyDER(&priv.Pub, dgst, s) {
			if !bytes.Equal(dgst, digest[:]) || !bytes.Equal(s, sig) {
				t.Fatalf("forged DER signature accepted")
			}
		}
	})
}
