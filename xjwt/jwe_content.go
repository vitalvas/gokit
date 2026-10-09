package xjwt

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/binary"
	"fmt"
	"hash"
)

// contentEncrypt encrypts plaintext under cek for the given enc algorithm,
// authenticating aad. It returns the iv, ciphertext, and authentication tag.
func contentEncrypt(enc string, cek, plaintext, aad []byte) (iv, ciphertext, tag []byte, err error) {
	switch enc {
	case A128CBCHS256, A192CBCHS384, A256CBCHS512:
		return cbcHMACEncrypt(enc, cek, plaintext, aad)
	case A128GCM, A192GCM, A256GCM:
		return gcmEncrypt(cek, plaintext, aad)
	default:
		return nil, nil, nil, fmt.Errorf("xjwt: unsupported content encryption algorithm %q", enc)
	}
}

// contentCiphertext bundles the parts of an encrypted JWE payload for the
// decryption helpers (keeps the parameter count within the project limit).
type contentCiphertext struct {
	enc        string
	cek        []byte
	iv         []byte
	ciphertext []byte
	tag        []byte
	aad        []byte
}

// contentDecrypt reverses contentEncrypt.
func contentDecrypt(c contentCiphertext) ([]byte, error) {
	switch c.enc {
	case A128CBCHS256, A192CBCHS384, A256CBCHS512:
		return cbcHMACDecrypt(c)
	case A128GCM, A192GCM, A256GCM:
		return gcmDecrypt(c)
	default:
		return nil, fmt.Errorf("xjwt: unsupported content encryption algorithm %q", c.enc)
	}
}

// cbcHMACParams returns the MAC key length, the AES key length, and the hash
// constructor for an AES-CBC-HMAC algorithm. The CEK is MAC key || ENC key.
func cbcHMACParams(enc string) (macLen, encLen int, h func() hash.Hash, tagLen int) {
	switch enc {
	case A128CBCHS256:
		return 16, 16, sha256.New, 16
	case A192CBCHS384:
		return 24, 24, sha512.New384, 24
	default: // A256CBCHS512
		return 32, 32, sha512.New, 32
	}
}

func cbcHMACEncrypt(enc string, cek, plaintext, aad []byte) (iv, ciphertext, tag []byte, err error) {
	macLen, encLen, newHash, tagLen := cbcHMACParams(enc)
	if len(cek) != macLen+encLen {
		return nil, nil, nil, fmt.Errorf("xjwt: CEK must be %d bytes for %s", macLen+encLen, enc)
	}

	macKey := cek[:macLen]
	encKey := cek[macLen:]

	block, err := aes.NewCipher(encKey)
	if err != nil {
		return nil, nil, nil, err
	}

	iv = make([]byte, aes.BlockSize)
	if _, err = rand.Read(iv); err != nil {
		return nil, nil, nil, err
	}

	padded := pkcs7Pad(plaintext, aes.BlockSize)
	ciphertext = make([]byte, len(padded))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(ciphertext, padded)

	tag = cbcHMACTag(cbcTagInput{
		newHash:    newHash,
		macKey:     macKey,
		aad:        aad,
		iv:         iv,
		ciphertext: ciphertext,
		tagLen:     tagLen,
	})

	return iv, ciphertext, tag, nil
}

func cbcHMACDecrypt(c contentCiphertext) ([]byte, error) {
	macLen, encLen, newHash, tagLen := cbcHMACParams(c.enc)
	if len(c.cek) != macLen+encLen {
		return nil, fmt.Errorf("xjwt: CEK must be %d bytes for %s", macLen+encLen, c.enc)
	}

	macKey := c.cek[:macLen]
	encKey := c.cek[macLen:]

	expectedTag := cbcHMACTag(cbcTagInput{
		newHash:    newHash,
		macKey:     macKey,
		aad:        c.aad,
		iv:         c.iv,
		ciphertext: c.ciphertext,
		tagLen:     tagLen,
	})
	if subtle.ConstantTimeCompare(expectedTag, c.tag) != 1 {
		return nil, ErrTokenSignatureInvalid
	}

	block, err := aes.NewCipher(encKey)
	if err != nil {
		return nil, err
	}

	if len(c.ciphertext) == 0 || len(c.ciphertext)%aes.BlockSize != 0 || len(c.iv) != aes.BlockSize {
		return nil, fmt.Errorf("xjwt: invalid JWE ciphertext")
	}

	plaintext := make([]byte, len(c.ciphertext))
	cipher.NewCBCDecrypter(block, c.iv).CryptBlocks(plaintext, c.ciphertext)

	return pkcs7Unpad(plaintext, aes.BlockSize)
}

// cbcTagInput bundles the inputs to the AES-CBC-HMAC authentication tag.
type cbcTagInput struct {
	newHash    func() hash.Hash
	macKey     []byte
	aad        []byte
	iv         []byte
	ciphertext []byte
	tagLen     int
}

// cbcHMACTag computes the authentication tag: HMAC over
// aad || iv || ciphertext || AL, truncated to tagLen (RFC 7518 Section 5.2.2).
func cbcHMACTag(in cbcTagInput) []byte {
	mac := hmac.New(in.newHash, in.macKey)
	mac.Write(in.aad)
	mac.Write(in.iv)
	mac.Write(in.ciphertext)

	var al [8]byte
	binary.BigEndian.PutUint64(al[:], uint64(len(in.aad))*8)
	mac.Write(al[:])

	return mac.Sum(nil)[:in.tagLen]
}

func gcmEncrypt(cek, plaintext, aad []byte) (iv, ciphertext, tag []byte, err error) {
	block, err := aes.NewCipher(cek)
	if err != nil {
		return nil, nil, nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, nil, err
	}

	iv = make([]byte, gcm.NonceSize())
	if _, err = rand.Read(iv); err != nil {
		return nil, nil, nil, err
	}

	sealed := gcm.Seal(nil, iv, plaintext, aad)
	// GCM appends the 16-byte tag; JWE carries it separately.
	ciphertext = sealed[:len(sealed)-gcm.Overhead()]
	tag = sealed[len(sealed)-gcm.Overhead():]

	return iv, ciphertext, tag, nil
}

func gcmDecrypt(c contentCiphertext) ([]byte, error) {
	block, err := aes.NewCipher(c.cek)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	if len(c.iv) != gcm.NonceSize() {
		return nil, fmt.Errorf("xjwt: invalid GCM nonce length")
	}

	plaintext, err := gcm.Open(nil, c.iv, append(c.ciphertext, c.tag...), c.aad)
	if err != nil {
		return nil, ErrTokenSignatureInvalid
	}

	return plaintext, nil
}

// pkcs7Pad appends PKCS#7 padding to a multiple of blockSize.
func pkcs7Pad(data []byte, blockSize int) []byte {
	pad := blockSize - len(data)%blockSize
	out := make([]byte, len(data)+pad)
	copy(out, data)
	for i := len(data); i < len(out); i++ {
		out[i] = byte(pad)
	}

	return out
}

// pkcs7Unpad removes PKCS#7 padding, validating it.
func pkcs7Unpad(data []byte, blockSize int) ([]byte, error) {
	if len(data) == 0 || len(data)%blockSize != 0 {
		return nil, fmt.Errorf("xjwt: invalid padding")
	}

	pad := int(data[len(data)-1])
	if pad == 0 || pad > blockSize || pad > len(data) {
		return nil, fmt.Errorf("xjwt: invalid padding")
	}

	// Constant-time check of all padding bytes.
	var bad byte
	for i := len(data) - pad; i < len(data); i++ {
		bad |= data[i] ^ byte(pad)
	}
	if bad != 0 {
		return nil, fmt.Errorf("xjwt: invalid padding")
	}

	return data[:len(data)-pad], nil
}
