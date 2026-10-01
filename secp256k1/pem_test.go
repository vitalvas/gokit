package secp256k1

import (
	"encoding/asn1"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPEMRoundTrip(t *testing.T) {
	priv, err := GeneratePrivateKey()
	require.NoError(t, err)

	t.Run("private key PEM round-trip", func(t *testing.T) {
		data, err := MarshalPrivateKeyPEM(priv)
		require.NoError(t, err)
		assert.True(t, strings.HasPrefix(string(data), "-----BEGIN EC PRIVATE KEY-----"))

		back, err := ParsePrivateKeyPEM(data)
		require.NoError(t, err)
		assert.True(t, priv.Equal(back))
		assert.True(t, priv.Pub.Equal(&back.Pub))
	})

	t.Run("public key PEM round-trip", func(t *testing.T) {
		data, err := MarshalPublicKeyPEM(&priv.Pub)
		require.NoError(t, err)
		assert.True(t, strings.HasPrefix(string(data), "-----BEGIN PUBLIC KEY-----"))

		back, err := ParsePublicKeyPEM(data)
		require.NoError(t, err)
		assert.True(t, priv.Pub.Equal(back))
	})

	t.Run("DER private round-trip", func(t *testing.T) {
		der, err := MarshalECPrivateKey(priv)
		require.NoError(t, err)

		back, err := ParseECPrivateKey(der)
		require.NoError(t, err)
		assert.True(t, priv.Equal(back))
	})

	t.Run("DER public round-trip", func(t *testing.T) {
		der, err := MarshalPKIXPublicKey(&priv.Pub)
		require.NoError(t, err)

		back, err := ParsePKIXPublicKey(der)
		require.NoError(t, err)
		assert.True(t, priv.Pub.Equal(back))
	})
}

func TestPEMErrors(t *testing.T) {
	t.Run("no PEM block", func(t *testing.T) {
		_, err := ParsePrivateKeyPEM([]byte("not pem"))
		require.Error(t, err)
		_, err = ParsePublicKeyPEM([]byte("not pem"))
		require.Error(t, err)
	})

	t.Run("garbage DER", func(t *testing.T) {
		_, err := ParseECPrivateKey([]byte{0x30, 0x00, 0xff})
		require.Error(t, err)
		_, err = ParsePKIXPublicKey([]byte{0x30, 0x00, 0xff})
		require.Error(t, err)
	})

	t.Run("wrong curve OID rejected", func(t *testing.T) {
		priv, err := GeneratePrivateKey()
		require.NoError(t, err)

		der, err := asn1.Marshal(ecPrivateKey{
			Version:       1,
			PrivateKey:    priv.Serialize(),
			NamedCurveOID: oidPublicKeyECDSA, // deliberately not secp256k1
		})
		require.NoError(t, err)
		_, err = ParseECPrivateKey(der)
		require.Error(t, err)
	})

	t.Run("non-EC public key rejected", func(t *testing.T) {
		priv, err := GeneratePrivateKey()
		require.NoError(t, err)

		raw := priv.Pub.SerializeUncompressed()
		der, err := asn1.Marshal(pkixPublicKey{
			Algorithm: pkixAlgorithm{Algorithm: oidSecp256k1, Parameters: oidSecp256k1}, // wrong algorithm OID
			PublicKey: asn1.BitString{Bytes: raw, BitLength: 8 * len(raw)},
		})
		require.NoError(t, err)
		_, err = ParsePKIXPublicKey(der)
		require.Error(t, err)
	})

	t.Run("trailing bytes rejected", func(t *testing.T) {
		priv, err := GeneratePrivateKey()
		require.NoError(t, err)

		der, err := MarshalECPrivateKey(priv)
		require.NoError(t, err)
		_, err = ParseECPrivateKey(append(der, 0x00))
		require.Error(t, err)

		pder, err := MarshalPKIXPublicKey(&priv.Pub)
		require.NoError(t, err)
		_, err = ParsePKIXPublicKey(append(pder, 0x00))
		require.Error(t, err)
	})
}

func BenchmarkMarshalECPrivateKey(b *testing.B) {
	priv, err := GeneratePrivateKey()
	require.NoError(b, err)

	b.ResetTimer()
	for b.Loop() {
		_, _ = MarshalECPrivateKey(priv)
	}
}

func BenchmarkParseECPrivateKey(b *testing.B) {
	priv, err := GeneratePrivateKey()
	require.NoError(b, err)
	der, err := MarshalECPrivateKey(priv)
	require.NoError(b, err)

	b.ResetTimer()
	for b.Loop() {
		_, _ = ParseECPrivateKey(der)
	}
}

// FuzzParseECPrivateKey feeds arbitrary DER to the private-key parser; it must
// never panic, and any key it accepts must have an on-curve public key.
func FuzzParseECPrivateKey(f *testing.F) {
	priv, err := GeneratePrivateKey()
	require.NoError(f, err)
	der, err := MarshalECPrivateKey(priv)
	require.NoError(f, err)

	f.Add(der)
	f.Add([]byte{})
	f.Add([]byte{0x30, 0x00})

	f.Fuzz(func(t *testing.T, data []byte) {
		key, err := ParseECPrivateKey(data)
		if err != nil {
			return
		}

		if !key.Pub.IsValid() {
			t.Fatalf("parsed private key has invalid public key")
		}
	})
}

// FuzzParsePKIXPublicKey feeds arbitrary DER to the public-key parser; it must
// never panic, and any key it accepts must lie on the curve.
func FuzzParsePKIXPublicKey(f *testing.F) {
	priv, err := GeneratePrivateKey()
	require.NoError(f, err)
	der, err := MarshalPKIXPublicKey(&priv.Pub)
	require.NoError(f, err)

	f.Add(der)
	f.Add([]byte{})
	f.Add([]byte{0x30, 0x00})

	f.Fuzz(func(t *testing.T, data []byte) {
		pub, err := ParsePKIXPublicKey(data)
		if err != nil {
			return
		}

		if !pub.IsValid() {
			t.Fatalf("parsed public key is not on the curve")
		}
	})
}
