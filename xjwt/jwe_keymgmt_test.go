package xjwt

import (
	"crypto/ecdh"
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecryptCEKErrors(t *testing.T) {
	t.Run("unsupported enc", func(t *testing.T) {
		_, err := decryptCEK(cekDecryptInput{alg: RSAOAEP256, enc: "BADENC"})
		require.Error(t, err)
	})

	t.Run("unsupported alg", func(t *testing.T) {
		_, err := decryptCEK(cekDecryptInput{alg: "BADALG", enc: A128GCM})
		require.Error(t, err)
	})

	t.Run("RSA-OAEP wrong key type", func(t *testing.T) {
		_, err := decryptCEK(cekDecryptInput{alg: RSAOAEP256, enc: A128GCM, key: []byte("not-rsa")})
		require.ErrorIs(t, err, ErrKeyTypeMismatch)
	})

	t.Run("AESKW wrong key type", func(t *testing.T) {
		_, err := decryptCEK(cekDecryptInput{alg: A128KW, enc: A128GCM, key: "not-bytes"})
		require.ErrorIs(t, err, ErrKeyTypeMismatch)
	})

	t.Run("dir wrong key type", func(t *testing.T) {
		_, err := decryptCEK(cekDecryptInput{alg: Dir, enc: A128GCM, key: "not-bytes"})
		require.ErrorIs(t, err, ErrKeyTypeMismatch)
	})

	t.Run("dir wrong key length", func(t *testing.T) {
		_, err := decryptCEK(cekDecryptInput{alg: Dir, enc: A256GCM, key: make([]byte, 16)})
		require.Error(t, err)
	})
}

func TestEncryptCEKErrors(t *testing.T) {
	t.Run("unsupported enc", func(t *testing.T) {
		_, _, _, err := encryptCEK(RSAOAEP256, "BADENC", nil, nil, nil)
		require.Error(t, err)
	})

	t.Run("unsupported alg", func(t *testing.T) {
		_, _, _, err := encryptCEK("BADALG", A128GCM, nil, nil, nil)
		require.Error(t, err)
	})

	t.Run("AESKW wrong key type", func(t *testing.T) {
		_, _, _, err := encryptCEK(A128KW, A128GCM, "not-bytes", nil, nil)
		require.ErrorIs(t, err, ErrKeyTypeMismatch)
	})

	t.Run("dir wrong length", func(t *testing.T) {
		_, _, _, err := encryptCEK(Dir, A256GCM, make([]byte, 8), nil, nil)
		require.Error(t, err)
	})

	t.Run("ECDH wrong key type", func(t *testing.T) {
		_, _, _, err := encryptCEK(ECDHES, A128GCM, "not-ec", nil, nil)
		require.ErrorIs(t, err, ErrKeyTypeMismatch)
	})
}

func TestDecryptCEKECDHErrors(t *testing.T) {
	priv, err := ecdh.P256().GenerateKey(rand.Reader)
	require.NoError(t, err)

	t.Run("missing epk", func(t *testing.T) {
		_, err := decryptCEK(cekDecryptInput{alg: ECDHES, enc: A128GCM, key: priv})
		require.Error(t, err)
	})

	t.Run("wrong key type", func(t *testing.T) {
		epk := &JSONWebKey{Kty: "EC", Crv: "P-256", X: "AA", Y: "AA"}
		_, err := decryptCEK(cekDecryptInput{alg: ECDHES, enc: A128GCM, key: "not-ec", epk: epk})
		require.ErrorIs(t, err, ErrKeyTypeMismatch)
	})

	t.Run("bad epk coords", func(t *testing.T) {
		epk := &JSONWebKey{Kty: "EC", Crv: "P-256", X: "!!!", Y: "!!!"}
		_, err := decryptCEK(cekDecryptInput{alg: ECDHES, enc: A128GCM, key: priv, epk: epk})
		require.Error(t, err)
	})
}

func TestEcdhDerivedSpecDefault(t *testing.T) {
	// A non-ECDH alg falls through to the default branch.
	algID, keyLen, direct := ecdhDerivedSpec("UNKNOWN", A128GCM, 16)
	assert.Equal(t, "UNKNOWN", algID)
	assert.Equal(t, 16, keyLen)
	assert.True(t, direct)
}
