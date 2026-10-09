package xjwt

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vitalvas/gokit/secp256k1"
)

func pemBlock(t *testing.T, typ string, der []byte) []byte {
	t.Helper()

	return pem.EncodeToMemory(&pem.Block{Type: typ, Bytes: der})
}

func TestParseRSAFromPEM(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	t.Run("PKCS8 private", func(t *testing.T) {
		der, err := x509.MarshalPKCS8PrivateKey(key)
		require.NoError(t, err)
		got, err := ParseRSAPrivateKeyFromPEM(pemBlock(t, "PRIVATE KEY", der))
		require.NoError(t, err)
		assert.Equal(t, 0, got.N.Cmp(key.N))
	})

	t.Run("PKCS1 private", func(t *testing.T) {
		der := x509.MarshalPKCS1PrivateKey(key)
		got, err := ParseRSAPrivateKeyFromPEM(pemBlock(t, "RSA PRIVATE KEY", der))
		require.NoError(t, err)
		assert.Equal(t, 0, got.N.Cmp(key.N))
	})

	t.Run("PKIX public", func(t *testing.T) {
		der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
		require.NoError(t, err)
		got, err := ParseRSAPublicKeyFromPEM(pemBlock(t, "PUBLIC KEY", der))
		require.NoError(t, err)
		assert.Equal(t, 0, got.N.Cmp(key.N))
	})

	t.Run("PKCS1 public", func(t *testing.T) {
		der := x509.MarshalPKCS1PublicKey(&key.PublicKey)
		got, err := ParseRSAPublicKeyFromPEM(pemBlock(t, "RSA PUBLIC KEY", der))
		require.NoError(t, err)
		assert.Equal(t, 0, got.N.Cmp(key.N))
	})
}

func TestParseECFromPEM(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	t.Run("SEC1 private", func(t *testing.T) {
		der, err := x509.MarshalECPrivateKey(key)
		require.NoError(t, err)
		got, err := ParseECPrivateKeyFromPEM(pemBlock(t, "EC PRIVATE KEY", der))
		require.NoError(t, err)
		assert.True(t, key.Equal(got))
	})

	t.Run("PKIX public", func(t *testing.T) {
		der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
		require.NoError(t, err)
		got, err := ParseECPublicKeyFromPEM(pemBlock(t, "PUBLIC KEY", der))
		require.NoError(t, err)
		assert.True(t, key.PublicKey.Equal(got))
	})
}

func TestParseEdFromPEM(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	t.Run("PKCS8 private", func(t *testing.T) {
		der, err := x509.MarshalPKCS8PrivateKey(priv)
		require.NoError(t, err)
		got, err := ParseEdPrivateKeyFromPEM(pemBlock(t, "PRIVATE KEY", der))
		require.NoError(t, err)
		assert.Equal(t, priv, got)
	})

	t.Run("PKIX public", func(t *testing.T) {
		der, err := x509.MarshalPKIXPublicKey(pub)
		require.NoError(t, err)
		got, err := ParseEdPublicKeyFromPEM(pemBlock(t, "PUBLIC KEY", der))
		require.NoError(t, err)
		assert.Equal(t, pub, got)
	})
}

func TestParseFromPEMErrors(t *testing.T) {
	t.Run("not PEM", func(t *testing.T) {
		_, err := ParseRSAPrivateKeyFromPEM([]byte("not pem"))
		assert.ErrorIs(t, err, ErrKeyMustBePEMEncoded)
	})

	t.Run("wrong key type rejected", func(t *testing.T) {
		// An EC key fed to the RSA parser must be rejected, not misinterpreted.
		ec, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)
		der, err := x509.MarshalPKCS8PrivateKey(ec)
		require.NoError(t, err)

		_, err = ParseRSAPrivateKeyFromPEM(pemBlock(t, "PRIVATE KEY", der))
		assert.ErrorIs(t, err, ErrNotRSAKey)
	})

	t.Run("public parser rejects wrong type", func(t *testing.T) {
		rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)
		der, err := x509.MarshalPKIXPublicKey(&rsaKey.PublicKey)
		require.NoError(t, err)

		_, err = ParseECPublicKeyFromPEM(pemBlock(t, "PUBLIC KEY", der))
		assert.ErrorIs(t, err, ErrNotECKey)
	})
}

// TestParsePEMEndToEnd proves a PEM-loaded key signs and verifies a token.
func TestParsePEMEndToEnd(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	privDER, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)
	pubDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	require.NoError(t, err)

	priv, err := ParseRSAPrivateKeyFromPEM(pemBlock(t, "PRIVATE KEY", privDER))
	require.NoError(t, err)
	pub, err := ParseRSAPublicKeyFromPEM(pemBlock(t, "PUBLIC KEY", pubDER))
	require.NoError(t, err)

	token, err := Sign("PS256", "k1", MapClaims{"sub": "u1"}, priv)
	require.NoError(t, err)

	_, err = Verify(token, func(_ Header) (any, error) { return pub, nil }, []string{"PS256"})
	require.NoError(t, err)
}

func pemOf(t *testing.T, typ string, der []byte) []byte {
	t.Helper()

	return pem.EncodeToMemory(&pem.Block{Type: typ, Bytes: der})
}

func mustPKCS8(t *testing.T, k any) []byte {
	t.Helper()
	der, err := x509.MarshalPKCS8PrivateKey(k)
	require.NoError(t, err)

	return der
}

func TestPEMParsersErrors(t *testing.T) {
	ec, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	edPub, edPriv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	ecPKCS8, _ := x509.MarshalPKCS8PrivateKey(ec)
	ecPKIX, _ := x509.MarshalPKIXPublicKey(&ec.PublicKey)
	rsaPKIX, _ := x509.MarshalPKIXPublicKey(&rsaKey.PublicKey)
	edPKCS8, _ := x509.MarshalPKCS8PrivateKey(edPriv)
	edPKIX, _ := x509.MarshalPKIXPublicKey(edPub)

	notPEM := []byte("definitely not pem")
	garbageDER := pemOf(t, "PRIVATE KEY", []byte{0x01, 0x02, 0x03})

	t.Run("not PEM rejected by every parser", func(t *testing.T) {
		_, e1 := ParseRSAPrivateKeyFromPEM(notPEM)
		_, e2 := ParseRSAPublicKeyFromPEM(notPEM)
		_, e3 := ParseECPrivateKeyFromPEM(notPEM)
		_, e4 := ParseECPublicKeyFromPEM(notPEM)
		_, e5 := ParseEdPrivateKeyFromPEM(notPEM)
		_, e6 := ParseEdPublicKeyFromPEM(notPEM)
		for _, e := range []error{e1, e2, e3, e4, e5, e6} {
			require.ErrorIs(t, e, ErrKeyMustBePEMEncoded)
		}
	})

	t.Run("wrong key type rejected", func(t *testing.T) {
		// EC key handed to RSA parsers, and vice versa.
		_, err := ParseRSAPrivateKeyFromPEM(pemOf(t, "PRIVATE KEY", ecPKCS8))
		require.ErrorIs(t, err, ErrNotRSAKey)
		_, err = ParseRSAPublicKeyFromPEM(pemOf(t, "PUBLIC KEY", ecPKIX))
		require.ErrorIs(t, err, ErrNotRSAKey)
		_, err = ParseECPrivateKeyFromPEM(pemOf(t, "PRIVATE KEY", mustPKCS8(t, rsaKey)))
		require.ErrorIs(t, err, ErrNotECKey)
		_, err = ParseECPublicKeyFromPEM(pemOf(t, "PUBLIC KEY", rsaPKIX))
		require.ErrorIs(t, err, ErrNotECKey)
		_, err = ParseEdPrivateKeyFromPEM(pemOf(t, "PRIVATE KEY", ecPKCS8))
		require.ErrorIs(t, err, ErrNotEdKey)
		_, err = ParseEdPublicKeyFromPEM(pemOf(t, "PUBLIC KEY", ecPKIX))
		require.ErrorIs(t, err, ErrNotEdKey)
	})

	t.Run("garbage DER rejected", func(t *testing.T) {
		_, e1 := ParseRSAPrivateKeyFromPEM(garbageDER)
		_, e2 := ParseECPrivateKeyFromPEM(garbageDER)
		_, e3 := ParseEdPrivateKeyFromPEM(garbageDER)
		_, e4 := ParseECPublicKeyFromPEM(garbageDER)
		for _, e := range []error{e1, e2, e3, e4} {
			require.Error(t, e)
		}
	})

	t.Run("valid Ed25519 round-trips", func(t *testing.T) {
		priv, err := ParseEdPrivateKeyFromPEM(pemOf(t, "PRIVATE KEY", edPKCS8))
		require.NoError(t, err)
		assert.Equal(t, edPriv, priv)
		pub, err := ParseEdPublicKeyFromPEM(pemOf(t, "PUBLIC KEY", edPKIX))
		require.NoError(t, err)
		assert.Equal(t, edPub, pub)
	})

	t.Run("RSA PKCS1 public round-trips", func(t *testing.T) {
		der := x509.MarshalPKCS1PublicKey(&rsaKey.PublicKey)
		got, err := ParseRSAPublicKeyFromPEM(pemOf(t, "RSA PUBLIC KEY", der))
		require.NoError(t, err)
		assert.Equal(t, 0, got.N.Cmp(rsaKey.N))
	})
}

func TestParseEdPublicKeyGarbage(t *testing.T) {
	// A PEM block whose DER is not a valid PKIX key -> ErrNotEdKey.
	block := "-----BEGIN PUBLIC KEY-----\nAAAA\n-----END PUBLIC KEY-----\n"
	_, err := ParseEdPublicKeyFromPEM([]byte(block))
	require.Error(t, err)
}

func TestMarshalPublicKeyToPEMValid(t *testing.T) {
	ecKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	pem, err := MarshalPublicKeyToPEM(&ecKey.PublicKey)
	require.NoError(t, err)
	assert.Contains(t, string(pem), "BEGIN PUBLIC KEY")
}

func TestMarshalPublicKeyToPEMError(t *testing.T) {
	// secp256k1 public keys are not representable via crypto/x509.
	k, err := secp256k1.GeneratePrivateKey()
	require.NoError(t, err)
	_, err = MarshalPublicKeyToPEM(&k.Pub)
	require.Error(t, err)
}

func TestMarshalKeyToPEMErrors(t *testing.T) {
	// secp256k1 keys are not representable via crypto/x509.
	priv, _, err := GenerateKey(ES256K, "k")
	require.NoError(t, err)
	_, perr := MarshalPrivateKeyToPEM(priv)
	require.Error(t, perr)
}
