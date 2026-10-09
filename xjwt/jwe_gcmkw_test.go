package xjwt

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGCMKWKeyLenUnsupported(t *testing.T) {
	_, err := gcmkwKeyLen("A999GCMKW")
	require.Error(t, err)
}

func TestGCMKWDecryptCEKErrors(t *testing.T) {
	key := make([]byte, 16)

	t.Run("wrong key type", func(t *testing.T) {
		_, err := gcmkwDecryptCEK(A128GCMKW, "not-bytes", nil, &jweHeader{})
		require.ErrorIs(t, err, ErrKeyTypeMismatch)
	})

	t.Run("unsupported gcmkw alg", func(t *testing.T) {
		_, err := gcmkwDecryptCEK("A999GCMKW", key, nil, &jweHeader{})
		require.Error(t, err)
	})

	t.Run("wrong key length", func(t *testing.T) {
		_, err := gcmkwDecryptCEK(A256GCMKW, key, nil, &jweHeader{IV: "AA", Tag: "AA"})
		require.Error(t, err)
	})

	t.Run("missing iv/tag", func(t *testing.T) {
		_, err := gcmkwDecryptCEK(A128GCMKW, key, nil, &jweHeader{})
		require.Error(t, err)
	})

	t.Run("bad iv base64", func(t *testing.T) {
		_, err := gcmkwDecryptCEK(A128GCMKW, key, nil, &jweHeader{IV: "!!!", Tag: "AA"})
		require.Error(t, err)
	})

	t.Run("bad tag base64", func(t *testing.T) {
		_, err := gcmkwDecryptCEK(A128GCMKW, key, nil, &jweHeader{IV: "AA", Tag: "!!!"})
		require.Error(t, err)
	})

	t.Run("wrong nonce length", func(t *testing.T) {
		// Valid base64 iv of the wrong length (not 12 bytes).
		_, err := gcmkwDecryptCEK(A128GCMKW, key, nil, &jweHeader{
			IV:  "AAAA", // 3 bytes
			Tag: "AAAAAAAAAAAAAAAAAAAAAA",
		})
		require.Error(t, err)
	})
}

func TestEncryptCEKGCMKWErrors(t *testing.T) {
	t.Run("wrong key type", func(t *testing.T) {
		_, _, err := gcmkwEncryptCEK(A128GCMKW, A128GCM, "not-bytes", &jweHeader{})
		require.ErrorIs(t, err, ErrKeyTypeMismatch)
	})

	t.Run("wrong key length", func(t *testing.T) {
		_, _, err := gcmkwEncryptCEK(A256GCMKW, A128GCM, make([]byte, 16), &jweHeader{})
		require.Error(t, err)
	})

	t.Run("unsupported enc", func(t *testing.T) {
		_, _, err := gcmkwEncryptCEK(A128GCMKW, "BADENC", make([]byte, 16), &jweHeader{})
		require.Error(t, err)
	})
}
