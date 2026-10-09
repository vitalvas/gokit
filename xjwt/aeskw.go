package xjwt

import (
	"crypto/aes"
	"crypto/subtle"
	"encoding/binary"
	"fmt"
)

// aesKeyWrap implements the AES Key Wrap algorithm (RFC 3394) with the default
// IV. It wraps plaintext key material (a multiple of 8 bytes, at least 16) under
// a KEK. Used for the A128KW/A192KW/A256KW JWE key-management algorithms.
//
// RFC 3394 is not provided by the standard library, so it is implemented here
// against the spec; the RFC's own test vectors cover it.
var kwDefaultIV = []byte{0xA6, 0xA6, 0xA6, 0xA6, 0xA6, 0xA6, 0xA6, 0xA6}

func aesKeyWrap(kek, plaintext []byte) ([]byte, error) {
	if len(plaintext)%8 != 0 || len(plaintext) < 16 {
		return nil, fmt.Errorf("xjwt: AES-KW plaintext must be a multiple of 8 bytes and at least 16")
	}

	block, err := aes.NewCipher(kek)
	if err != nil {
		return nil, err
	}

	n := len(plaintext) / 8
	r := make([][]byte, n)
	for i := range r {
		r[i] = make([]byte, 8)
		copy(r[i], plaintext[i*8:(i+1)*8])
	}

	a := make([]byte, 8)
	copy(a, kwDefaultIV)

	buf := make([]byte, 16)
	for j := 0; j < 6; j++ {
		for i := 0; i < n; i++ {
			copy(buf[:8], a)
			copy(buf[8:], r[i])
			block.Encrypt(buf, buf)

			copy(a, buf[:8])
			// A = MSB ^ t, where t = n*j + (i+1)
			t := uint64(n*j + i + 1)
			var tb [8]byte
			binary.BigEndian.PutUint64(tb[:], t)
			subtle.XORBytes(a, a, tb[:])

			copy(r[i], buf[8:])
		}
	}

	out := make([]byte, 0, 8+len(plaintext))
	out = append(out, a...)
	for i := 0; i < n; i++ {
		out = append(out, r[i]...)
	}

	return out, nil
}

func aesKeyUnwrap(kek, ciphertext []byte) ([]byte, error) {
	if len(ciphertext)%8 != 0 || len(ciphertext) < 24 {
		return nil, fmt.Errorf("xjwt: AES-KW ciphertext must be a multiple of 8 bytes and at least 24")
	}

	block, err := aes.NewCipher(kek)
	if err != nil {
		return nil, err
	}

	n := len(ciphertext)/8 - 1
	a := make([]byte, 8)
	copy(a, ciphertext[:8])

	r := make([][]byte, n)
	for i := range r {
		r[i] = make([]byte, 8)
		copy(r[i], ciphertext[(i+1)*8:(i+2)*8])
	}

	buf := make([]byte, 16)
	for j := 5; j >= 0; j-- {
		for i := n - 1; i >= 0; i-- {
			// A ^= t, where t = n*j + (i+1)
			t := uint64(n*j + i + 1)
			var tb [8]byte
			binary.BigEndian.PutUint64(tb[:], t)
			subtle.XORBytes(a, a, tb[:])

			copy(buf[:8], a)
			copy(buf[8:], r[i])
			block.Decrypt(buf, buf)

			copy(a, buf[:8])
			copy(r[i], buf[8:])
		}
	}

	// Integrity check against the default IV in constant time.
	if subtle.ConstantTimeCompare(a, kwDefaultIV) != 1 {
		return nil, fmt.Errorf("xjwt: AES-KW integrity check failed")
	}

	out := make([]byte, 0, n*8)
	for i := 0; i < n; i++ {
		out = append(out, r[i]...)
	}

	return out, nil
}
