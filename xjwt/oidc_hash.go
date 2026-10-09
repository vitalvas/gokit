package xjwt

import (
	"crypto"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
)

// hashForTokenAlg returns the hash function paired with a JWS signing algorithm,
// used for the OIDC at_hash/c_hash/s_hash computations (the hash whose output
// size matches the alg, e.g. ES512 -> SHA-512). EdDSA (Ed25519) uses SHA-512 per
// OIDC guidance; ES256K uses SHA-256.
func hashForTokenAlg(alg string) (crypto.Hash, error) {
	info, ok := algRegistry[alg]
	if !ok {
		return 0, fmt.Errorf("xjwt: unknown signing algorithm %q", alg)
	}

	if info.hash != 0 {
		return info.hash, nil
	}

	// Algorithms with no pre-hash in the registry (EdDSA, ES256K) still have a
	// defined hash for the OIDC *_hash claims.
	switch alg {
	case EdDSA:
		return crypto.SHA512, nil
	case ES256K:
		return crypto.SHA256, nil
	default:
		return 0, fmt.Errorf("xjwt: no OIDC hash defined for algorithm %q", alg)
	}
}

// oidcHalfHash computes the OIDC "left-most half" hash claim (at_hash, c_hash,
// s_hash): hash the ASCII value with the hash matching alg, take the left-most
// half of the digest, and base64url-encode it (OIDC Core 1.0 Section 3.3.2.11).
func oidcHalfHash(alg, value string) (string, error) {
	h, err := hashForTokenAlg(alg)
	if err != nil {
		return "", err
	}

	hasher := h.New()
	hasher.Write([]byte(value))
	sum := hasher.Sum(nil)

	return base64.RawURLEncoding.EncodeToString(sum[:len(sum)/2]), nil
}

// AccessTokenHash computes the at_hash claim for an access token, bound to the ID
// token's signing algorithm alg (OIDC Core 1.0 Section 3.3.2.11). An OP sets this on the
// ID token in the hybrid/implicit flow; an RP recomputes it to validate.
func AccessTokenHash(alg, accessToken string) (string, error) {
	return oidcHalfHash(alg, accessToken)
}

// CodeHash computes the c_hash claim for an authorization code, bound to alg
// (OIDC Core 1.0 Section 3.3.2.11).
func CodeHash(alg, code string) (string, error) {
	return oidcHalfHash(alg, code)
}

// StateHash computes the s_hash claim for an OAuth state value, bound to alg
// (OpenID Connect Financial-grade API).
func StateHash(alg, state string) (string, error) {
	return oidcHalfHash(alg, state)
}

// VerifyAccessTokenHash reports whether atHash is the correct at_hash for the
// access token under alg, comparing in constant time.
func VerifyAccessTokenHash(alg, accessToken, atHash string) bool {
	want, err := AccessTokenHash(alg, accessToken)
	if err != nil {
		return false
	}

	return subtle.ConstantTimeCompare([]byte(want), []byte(atHash)) == 1
}

// VerifyCodeHash reports whether cHash is the correct c_hash for code under alg.
func VerifyCodeHash(alg, code, cHash string) bool {
	want, err := CodeHash(alg, code)
	if err != nil {
		return false
	}

	return subtle.ConstantTimeCompare([]byte(want), []byte(cHash)) == 1
}
