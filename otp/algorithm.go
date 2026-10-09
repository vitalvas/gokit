package otp

import (
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"hash"
)

// Algorithm is the HMAC hash algorithm used to derive codes.
// The zero value is SHA1, the only algorithm most authenticator apps support.
type Algorithm int

// Supported HMAC algorithms.
const (
	SHA1 Algorithm = iota
	SHA256
	SHA512
)

// String returns the algorithm name as used in otpauth:// URIs.
func (a Algorithm) String() string {
	switch a {
	case SHA256:
		return "SHA256"
	case SHA512:
		return "SHA512"
	default:
		return "SHA1"
	}
}

func (a Algorithm) hash() func() hash.Hash {
	switch a {
	case SHA256:
		return sha256.New
	case SHA512:
		return sha512.New
	default:
		return sha1.New
	}
}
