package secp256k1

import (
	"errors"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type failingReader struct{}

func (failingReader) Read(_ []byte) (int, error) {
	return 0, errors.New("rng failure")
}

func TestKeyEdgeCases(t *testing.T) {
	t.Run("NewPublicKey rejects over-long coordinates", func(t *testing.T) {
		_, err := NewPublicKey(make([]byte, 33), make([]byte, 33))
		require.Error(t, err)
	})

	t.Run("GeneratePrivateKeyFromRand propagates reader error", func(t *testing.T) {
		_, err := GeneratePrivateKeyFromRand(failingReader{})
		require.Error(t, err)
	})
}

func BenchmarkParsePubKey(b *testing.B) {
	priv, err := GeneratePrivateKey()
	require.NoError(b, err)
	ser := priv.Pub.SerializeUncompressed()

	b.ResetTimer()
	for b.Loop() {
		_, _ = ParsePubKey(ser)
	}
}

func BenchmarkParsePubKeyCompressed(b *testing.B) {
	priv, err := GeneratePrivateKey()
	require.NoError(b, err)
	ser := priv.Pub.SerializeCompressed()

	b.ResetTimer()
	for b.Loop() {
		_, _ = ParsePubKey(ser)
	}
}

func BenchmarkGeneratePrivateKey(b *testing.B) {
	for b.Loop() {
		_, _ = GeneratePrivateKey()
	}
}

// FuzzParsePubKey feeds arbitrary bytes to the public-key parser. It must never
// panic; malformed or off-curve input must return an error, and any successful
// parse must round-trip back to the same bytes and land on the curve.
func FuzzParsePubKey(f *testing.F) {
	priv, err := GeneratePrivateKey()
	require.NoError(f, err)

	f.Add(priv.Pub.SerializeUncompressed())
	f.Add(priv.Pub.SerializeCompressed())
	f.Add([]byte{})
	f.Add([]byte{0x04})
	f.Add(make([]byte, 65))

	f.Fuzz(func(t *testing.T, data []byte) {
		pub, err := ParsePubKey(data)
		if err != nil {
			return
		}

		if !isOnCurve(pub.X, pub.Y) {
			t.Fatalf("parsed key is not on curve")
		}

		// Re-serializing in the input's own encoding must reproduce it exactly.
		var got []byte
		if data[0] == prefixUncompressed {
			got = pub.SerializeUncompressed()
		} else {
			got = pub.SerializeCompressed()
		}

		if string(got) != string(data) {
			t.Fatalf("round-trip mismatch: got %x want %x", got, data)
		}
	})
}

// FuzzNewPublicKey feeds arbitrary coordinate bytes to NewPublicKey, which must
// never panic and must reject anything not on the curve.
func FuzzNewPublicKey(f *testing.F) {
	priv, err := GeneratePrivateKey()
	require.NoError(f, err)
	ser := priv.Pub.SerializeUncompressed()

	f.Add(ser[1:33], ser[33:])
	f.Add([]byte{}, []byte{})
	f.Add(make([]byte, 32), make([]byte, 32))

	f.Fuzz(func(t *testing.T, x, y []byte) {
		pub, err := NewPublicKey(x, y)
		if err != nil {
			return
		}

		if !isOnCurve(pub.X, pub.Y) {
			t.Fatalf("accepted off-curve key")
		}
	})
}

func TestKeyParsing(t *testing.T) {
	t.Run("round-trip serialize/parse", func(t *testing.T) {
		priv, err := GeneratePrivateKey()
		require.NoError(t, err)

		ser := priv.Pub.SerializeUncompressed()
		require.Len(t, ser, 65)
		assert.Equal(t, byte(0x04), ser[0])

		parsed, err := ParsePubKey(ser)
		require.NoError(t, err)
		assert.Equal(t, 0, parsed.X.Cmp(priv.Pub.X))
		assert.Equal(t, 0, parsed.Y.Cmp(priv.Pub.Y))
	})

	t.Run("NewPublicKey validates curve", func(t *testing.T) {
		priv, err := GeneratePrivateKey()
		require.NoError(t, err)

		ser := priv.Pub.SerializeUncompressed()
		got, err := NewPublicKey(ser[1:33], ser[33:])
		require.NoError(t, err)
		assert.Equal(t, 0, got.X.Cmp(priv.Pub.X))

		_, err = NewPublicKey([]byte{1}, []byte{1})
		require.Error(t, err)
	})

	t.Run("bad length rejected", func(t *testing.T) {
		_, err := ParsePubKey([]byte{0x04, 0x01})
		require.Error(t, err)
	})

	t.Run("bad prefix rejected", func(t *testing.T) {
		bad := make([]byte, 65)
		bad[0] = 0x02
		_, err := ParsePubKey(bad)
		require.Error(t, err)
	})

	t.Run("off-curve rejected", func(t *testing.T) {
		bad := make([]byte, 65)
		bad[0] = 0x04
		bad[64] = 0x01
		_, err := ParsePubKey(bad)
		require.Error(t, err)
	})

	t.Run("scalar out of range rejected", func(t *testing.T) {
		_, err := PrivKeyFromBytes(make([]byte, 32))
		require.Error(t, err)
	})

	t.Run("PrivKeyFromBytes derives matching public key", func(t *testing.T) {
		seed := mustDecodeHex(t, "c9afa9d845ba75166b5c215767b1d6934e50c3db36e89b127b8a622b120f6721")
		priv, err := PrivKeyFromBytes(seed)
		require.NoError(t, err)
		assert.True(t, isOnCurve(priv.Pub.X, priv.Pub.Y))
	})

	t.Run("compressed round-trip", func(t *testing.T) {
		// Exercise both parities by generating several keys.
		for i := 0; i < 8; i++ {
			priv, err := GeneratePrivateKey()
			require.NoError(t, err)

			ser := priv.Pub.SerializeCompressed()
			require.Len(t, ser, 33)
			assert.Contains(t, []byte{0x02, 0x03}, ser[0])
			assert.Equal(t, priv.Pub.Y.Bit(0) == 1, ser[0] == 0x03)

			parsed, err := ParsePubKey(ser)
			require.NoError(t, err)
			assert.Equal(t, 0, parsed.X.Cmp(priv.Pub.X))
			assert.Equal(t, 0, parsed.Y.Cmp(priv.Pub.Y))
		}
	})

	t.Run("compressed and uncompressed parse to same point", func(t *testing.T) {
		priv, err := GeneratePrivateKey()
		require.NoError(t, err)

		fromC, err := ParsePubKey(priv.Pub.SerializeCompressed())
		require.NoError(t, err)
		fromU, err := ParsePubKey(priv.Pub.SerializeUncompressed())
		require.NoError(t, err)

		assert.Equal(t, 0, fromC.X.Cmp(fromU.X))
		assert.Equal(t, 0, fromC.Y.Cmp(fromU.Y))
	})

	t.Run("compressed bad length rejected", func(t *testing.T) {
		_, err := ParsePubKey([]byte{0x02, 0x01})
		require.Error(t, err)
	})

	t.Run("compressed off-curve x rejected", func(t *testing.T) {
		// x = p-1 has no corresponding curve point (x^3+7 is a non-residue).
		bad := make([]byte, 33)
		bad[0] = 0x02
		copy(bad[1:], P().Bytes())
		bad[32]--
		_, err := ParsePubKey(bad)
		require.Error(t, err)
	})

	t.Run("empty input rejected", func(t *testing.T) {
		_, err := ParsePubKey(nil)
		require.Error(t, err)
	})

	t.Run("unknown prefix rejected", func(t *testing.T) {
		_, err := ParsePubKey([]byte{0x05, 0x01})
		require.Error(t, err)
	})

	t.Run("private key serialize round-trip", func(t *testing.T) {
		priv, err := GeneratePrivateKey()
		require.NoError(t, err)

		ser := priv.Serialize()
		require.Len(t, ser, 32)

		back, err := PrivKeyFromBytes(ser)
		require.NoError(t, err)
		assert.Equal(t, 0, back.D.Cmp(priv.D))
		assert.Equal(t, 0, back.Pub.X.Cmp(priv.Pub.X))
	})

	t.Run("PubKey returns derived public key", func(t *testing.T) {
		priv, err := GeneratePrivateKey()
		require.NoError(t, err)

		assert.Same(t, &priv.Pub, priv.PubKey())
	})

	t.Run("Equal and IsValid", func(t *testing.T) {
		a, err := GeneratePrivateKey()
		require.NoError(t, err)
		b, err := GeneratePrivateKey()
		require.NoError(t, err)

		assert.True(t, a.Pub.IsValid())
		assert.False(t, (&PublicKey{X: big.NewInt(1), Y: big.NewInt(1)}).IsValid())
		assert.False(t, (*PublicKey)(nil).IsValid())

		aCopy := &PublicKey{X: new(big.Int).Set(a.Pub.X), Y: new(big.Int).Set(a.Pub.Y)}
		assert.True(t, a.Pub.Equal(aCopy))
		assert.False(t, a.Pub.Equal(&b.Pub))
		assert.False(t, a.Pub.Equal("not a key"))

		aPrivCopy, err := PrivKeyFromBytes(a.Serialize())
		require.NoError(t, err)
		assert.True(t, a.Equal(aPrivCopy))
		assert.False(t, a.Equal(b))
		assert.False(t, a.Equal("not a key"))
	})
}
