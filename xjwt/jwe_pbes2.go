package xjwt

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"hash"
)

// defaultPBES2Count is the iteration count used when encrypting. 600000 matches
// current OWASP guidance for PBKDF2-HMAC-SHA256. It is a variable (not a const)
// solely so tests can lower it; production callers should not change it.
var defaultPBES2Count = 600000

// maxPBES2Count bounds the iteration count accepted on decryption, so a
// malicious token cannot force an unbounded amount of work.
const maxPBES2Count = 10_000_000

func isPBES2Alg(alg string) bool {
	switch alg {
	case PBES2HS256A128KW, PBES2HS384A192KW, PBES2HS512A256KW:
		return true
	default:
		return false
	}
}

// pbes2Params returns the hash constructor and derived KEK length for a PBES2
// algorithm.
func pbes2Params(alg string) (newHash func() hash.Hash, kekLen int, err error) {
	switch alg {
	case PBES2HS256A128KW:
		return sha256.New, 16, nil
	case PBES2HS384A192KW:
		return sha512.New384, 24, nil
	case PBES2HS512A256KW:
		return sha512.New, 32, nil
	default:
		return nil, 0, fmt.Errorf("xjwt: unsupported PBES2 algorithm %q", alg)
	}
}

// pbes2DeriveKEK derives the AES-KW key-encryption key from a password. The
// PBKDF2 salt is the UTF-8 algorithm name, a 0x00 byte, and the salt input p2s
// (RFC 7518 Section 4.8.1.1).
func pbes2DeriveKEK(alg string, password, p2s []byte, p2c int) ([]byte, error) {
	newHash, kekLen, err := pbes2Params(alg)
	if err != nil {
		return nil, err
	}

	salt := make([]byte, 0, len(alg)+1+len(p2s))
	salt = append(salt, []byte(alg)...)
	salt = append(salt, 0x00)
	salt = append(salt, p2s...)

	return pbkdf2.Key(newHash, string(password), salt, p2c, kekLen)
}

// pbes2EncryptCEK generates a CEK, derives the KEK from the password, wraps the
// CEK, and fills p2s/p2c on the header.
func pbes2EncryptCEK(alg, enc string, password []byte, header *jweHeader) (cek, encryptedKey []byte, err error) {
	cekLen, err := cekLength(enc)
	if err != nil {
		return nil, nil, err
	}

	p2s := make([]byte, 16)
	if _, err = rand.Read(p2s); err != nil {
		return nil, nil, err
	}

	kek, err := pbes2DeriveKEK(alg, password, p2s, defaultPBES2Count)
	if err != nil {
		return nil, nil, err
	}

	cek = make([]byte, cekLen)
	if _, err = rand.Read(cek); err != nil {
		return nil, nil, err
	}

	encryptedKey, err = aesKeyWrap(kek, cek)
	if err != nil {
		return nil, nil, err
	}

	header.P2s = base64.RawURLEncoding.EncodeToString(p2s)
	header.P2c = defaultPBES2Count

	return cek, encryptedKey, nil
}

// pbes2DecryptCEK derives the KEK from the password and header p2s/p2c and
// unwraps the CEK.
func pbes2DecryptCEK(alg string, password, encryptedKey []byte, header *jweHeader) ([]byte, error) {
	if header.P2s == "" || header.P2c <= 0 {
		return nil, fmt.Errorf("xjwt: PBES2 header missing p2s/p2c")
	}

	if header.P2c > maxPBES2Count {
		return nil, fmt.Errorf("xjwt: PBES2 iteration count %d exceeds maximum", header.P2c)
	}

	p2s, err := base64.RawURLEncoding.DecodeString(header.P2s)
	if err != nil {
		return nil, err
	}
	if len(p2s) < 8 {
		return nil, fmt.Errorf("xjwt: PBES2 salt must be at least 8 bytes")
	}
	if len(encryptedKey) < 24 || len(encryptedKey)%8 != 0 {
		return nil, fmt.Errorf("xjwt: invalid wrapped CEK length")
	}

	kek, err := pbes2DeriveKEK(alg, password, p2s, header.P2c)
	if err != nil {
		return nil, err
	}

	return aesKeyUnwrap(kek, encryptedKey)
}
