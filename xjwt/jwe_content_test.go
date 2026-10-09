package xjwt

import (
	"crypto/aes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPKCS7Unpad(t *testing.T) {
	bs := aes.BlockSize

	t.Run("valid", func(t *testing.T) {
		padded := pkcs7Pad([]byte("hello"), bs)
		out, err := pkcs7Unpad(padded, bs)
		require.NoError(t, err)
		assert.Equal(t, []byte("hello"), out)
	})

	t.Run("empty", func(t *testing.T) {
		_, err := pkcs7Unpad(nil, bs)
		require.Error(t, err)
	})

	t.Run("not block aligned", func(t *testing.T) {
		_, err := pkcs7Unpad(make([]byte, bs+1), bs)
		require.Error(t, err)
	})

	t.Run("pad byte zero", func(t *testing.T) {
		b := make([]byte, bs) // last byte 0x00
		_, err := pkcs7Unpad(b, bs)
		require.Error(t, err)
	})

	t.Run("pad byte too large", func(t *testing.T) {
		b := make([]byte, bs)
		b[bs-1] = byte(bs + 1)
		_, err := pkcs7Unpad(b, bs)
		require.Error(t, err)
	})

	t.Run("inconsistent padding bytes", func(t *testing.T) {
		b := make([]byte, bs)
		b[bs-1] = 0x03
		b[bs-2] = 0x03
		b[bs-3] = 0x02 // should be 0x03
		_, err := pkcs7Unpad(b, bs)
		require.Error(t, err)
	})
}

func TestCBCHMACDecryptErrors(t *testing.T) {
	t.Run("wrong CEK length", func(t *testing.T) {
		_, err := cbcHMACDecrypt(contentCiphertext{enc: A128CBCHS256, cek: make([]byte, 10)})
		require.Error(t, err)
	})

	t.Run("bad ciphertext length passes MAC but fails block check", func(t *testing.T) {
		cek := make([]byte, 32) // A128CBC-HS256: 16 mac + 16 enc
		aad := []byte("aad")
		iv := make([]byte, aes.BlockSize)
		ct := []byte{1, 2, 3} // not a block multiple
		// Compute a valid tag (same params the function uses) so we reach the
		// ciphertext-length check rather than failing on the MAC.
		_, _, newHash, tagLen := cbcHMACParams(A128CBCHS256)
		tag := cbcHMACTag(cbcTagInput{
			newHash:    newHash,
			macKey:     cek[:16],
			aad:        aad,
			iv:         iv,
			ciphertext: ct,
			tagLen:     tagLen,
		})
		_, err := cbcHMACDecrypt(contentCiphertext{
			enc:        A128CBCHS256,
			cek:        cek,
			iv:         iv,
			ciphertext: ct,
			tag:        tag,
			aad:        aad,
		})
		require.Error(t, err)
	})
}

func TestGCMDecryptBadNonce(t *testing.T) {
	_, err := gcmDecrypt(contentCiphertext{cek: make([]byte, 16), iv: make([]byte, 7), ciphertext: []byte("ct"), tag: make([]byte, 16)})
	require.Error(t, err)
}

func TestContentEncryptUnsupported(t *testing.T) {
	iv, _, _, err := contentEncrypt("BADENC", nil, nil, nil)
	require.Error(t, err)
	require.Nil(t, iv)
	_, err = contentDecrypt(contentCiphertext{enc: "BADENC"})
	require.Error(t, err)
}
