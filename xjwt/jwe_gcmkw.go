package xjwt

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

func isGCMKWAlg(alg string) bool {
	switch alg {
	case A128GCMKW, A192GCMKW, A256GCMKW:
		return true
	default:
		return false
	}
}

// gcmkwKeyLen returns the required KEK length for an AES-GCMKW algorithm.
func gcmkwKeyLen(alg string) (int, error) {
	switch alg {
	case A128GCMKW:
		return 16, nil
	case A192GCMKW:
		return 24, nil
	case A256GCMKW:
		return 32, nil
	default:
		return 0, fmt.Errorf("xjwt: unsupported GCMKW algorithm %q", alg)
	}
}

// gcmkwEncryptCEK generates a CEK and wraps it with AES-GCM under the KEK,
// writing the GCM iv and tag to the header.
func gcmkwEncryptCEK(alg, enc string, key any, header *jweHeader) (cek, encryptedKey []byte, err error) {
	kek, ok := key.([]byte)
	if !ok {
		return nil, nil, ErrKeyTypeMismatch
	}

	kekLen, err := gcmkwKeyLen(alg)
	if err != nil {
		return nil, nil, err
	}
	if len(kek) != kekLen {
		return nil, nil, fmt.Errorf("xjwt: %s key must be %d bytes", alg, kekLen)
	}

	cekLen, err := cekLength(enc)
	if err != nil {
		return nil, nil, err
	}

	cek = make([]byte, cekLen)
	if _, err = rand.Read(cek); err != nil {
		return nil, nil, err
	}

	gcm, err := newGCM(kek)
	if err != nil {
		return nil, nil, err
	}

	iv := make([]byte, gcm.NonceSize())
	if _, err = rand.Read(iv); err != nil {
		return nil, nil, err
	}

	sealed := gcm.Seal(nil, iv, cek, nil)
	encryptedKey = sealed[:len(sealed)-gcm.Overhead()]
	tag := sealed[len(sealed)-gcm.Overhead():]

	header.IV = base64.RawURLEncoding.EncodeToString(iv)
	header.Tag = base64.RawURLEncoding.EncodeToString(tag)

	return cek, encryptedKey, nil
}

// gcmkwDecryptCEK unwraps the CEK using AES-GCM with the iv/tag from the header.
func gcmkwDecryptCEK(alg string, key any, encryptedKey []byte, header *jweHeader) ([]byte, error) {
	kek, ok := key.([]byte)
	if !ok {
		return nil, ErrKeyTypeMismatch
	}

	kekLen, err := gcmkwKeyLen(alg)
	if err != nil {
		return nil, err
	}
	if len(kek) != kekLen {
		return nil, fmt.Errorf("xjwt: %s key must be %d bytes", alg, kekLen)
	}

	if header.IV == "" || header.Tag == "" {
		return nil, fmt.Errorf("xjwt: GCMKW header missing iv/tag")
	}

	iv, err := base64.RawURLEncoding.DecodeString(header.IV)
	if err != nil {
		return nil, err
	}

	tag, err := base64.RawURLEncoding.DecodeString(header.Tag)
	if err != nil {
		return nil, err
	}

	gcm, err := newGCM(kek)
	if err != nil {
		return nil, err
	}

	if len(iv) != gcm.NonceSize() {
		return nil, fmt.Errorf("xjwt: invalid GCMKW iv length")
	}

	cek, err := gcm.Open(nil, iv, append(encryptedKey, tag...), nil)
	if err != nil {
		return nil, ErrTokenSignatureInvalid
	}

	return cek, nil
}

// newGCM builds an AES-GCM AEAD for the given key.
func newGCM(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	return cipher.NewGCM(block)
}
