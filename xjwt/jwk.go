package xjwt

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/vitalvas/gokit/secp256k1"
)

// JSONWebKey is a JSON Web Key (RFC 7517), modelling RSA, EC (including
// secp256k1), OKP (Ed25519), and symmetric (oct) keys. Private-key components
// (D, P, Q, ... / K for oct) are populated only for private keys.
type JSONWebKey struct {
	Kty    string   `json:"kty"`
	Use    string   `json:"use,omitempty"`
	Kid    string   `json:"kid,omitempty"`
	Alg    string   `json:"alg,omitempty"`
	KeyOps []string `json:"key_ops,omitempty"`
	// X.509 metadata (RFC 7517 Section 4.6-4.9)
	X5u     string   `json:"x5u,omitempty"`
	X5c     []string `json:"x5c,omitempty"`
	X5t     string   `json:"x5t,omitempty"`
	X5tS256 string   `json:"x5t#S256,omitempty"`
	// RSA public
	N string `json:"n,omitempty"`
	E string `json:"e,omitempty"`
	// RSA private
	D  string `json:"d,omitempty"`
	P  string `json:"p,omitempty"`
	Q  string `json:"q,omitempty"`
	Dp string `json:"dp,omitempty"`
	Dq string `json:"dq,omitempty"`
	Qi string `json:"qi,omitempty"`
	// EC/OKP
	Crv string `json:"crv,omitempty"`
	X   string `json:"x,omitempty"`
	Y   string `json:"y,omitempty"`
	// oct (symmetric)
	K string `json:"k,omitempty"`
	// AKP (Algorithm Key Pair, e.g. ML-DSA): pub is the public key, priv the
	// private seed, both base64url (draft-ietf-jose-pqc).
	Pub  string `json:"pub,omitempty"`
	Priv string `json:"priv,omitempty"`
}

// JWKS is a JSON Web Key Set.
type JWKS struct {
	Keys []JSONWebKey `json:"keys"`
}

// Len returns the number of keys in the set.
func (s *JWKS) Len() int {
	return len(s.Keys)
}

// LookupKeyID returns the first key with the given kid and whether one was found.
func (s *JWKS) LookupKeyID(kid string) (JSONWebKey, bool) {
	for _, k := range s.Keys {
		if k.Kid == kid {
			return k, true
		}
	}

	return JSONWebKey{}, false
}

// AddKey appends a key to the set.
func (s *JWKS) AddKey(key JSONWebKey) {
	s.Keys = append(s.Keys, key)
}

// RemoveKey removes all keys with the given kid, reporting whether any were
// removed.
func (s *JWKS) RemoveKey(kid string) bool {
	kept := s.Keys[:0]
	removed := false
	for _, k := range s.Keys {
		if k.Kid == kid {
			removed = true

			continue
		}
		kept = append(kept, k)
	}
	s.Keys = kept

	return removed
}

// ParseJWKS decodes a JWKS document into its keys. Individual key validity is
// checked at resolution time, not here.
func ParseJWKS(data []byte) (*JWKS, error) {
	var set JWKS
	if err := json.Unmarshal(data, &set); err != nil {
		return nil, fmt.Errorf("xjwt: parsing JWKS: %w", err)
	}

	return &set, nil
}

// PublicKey converts a JWK to its crypto public key, validating that the key
// type and curve are consistent with the requested algorithm. The returned key
// is one of *rsa.PublicKey, *ecdsa.PublicKey, ed25519.PublicKey, or
// *secp256k1.PublicKey.
func (k JSONWebKey) PublicKey(alg string) (any, error) {
	if !matchesAlg(k.Kty, k.Crv, alg) {
		return nil, fmt.Errorf("xjwt: key type %q/%q does not match alg %q", k.Kty, k.Crv, alg)
	}

	switch k.Kty {
	case "RSA":
		return parseRSAPublicKey(k.N, k.E)
	case "EC":
		if k.Crv == "secp256k1" {
			return parseSecp256k1PublicKey(k.X, k.Y)
		}

		return parseECPublicKey(k.Crv, k.X, k.Y)
	case "OKP":
		return parseOKPPublicKey(k.Crv, k.X)
	case "AKP":
		return parseMLDSAPublicKey(alg, k.Pub)
	default:
		return nil, fmt.Errorf("xjwt: unsupported key type %q", k.Kty)
	}
}

// matchesAlg reports whether a JWK's kty/crv can carry the given algorithm.
func matchesAlg(kty, crv, alg string) bool {
	switch {
	case alg == EdDSA:
		return kty == "OKP" && crv == "Ed25519"
	case alg == ES256K:
		return kty == "EC" && crv == "secp256k1"
	case alg == ES256:
		return kty == "EC" && crv == "P-256"
	case alg == ES384:
		return kty == "EC" && crv == "P-384"
	case alg == ES512:
		return kty == "EC" && crv == "P-521"
	case alg == MLDSA44 || alg == MLDSA65 || alg == MLDSA87:
		return kty == "AKP"
	case len(alg) >= 2 && (alg[:2] == "RS" || alg[:2] == "PS"):
		return kty == "RSA"
	default:
		return false
	}
}

func parseRSAPublicKey(nStr, eStr string) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(nStr)
	if err != nil {
		return nil, err
	}

	eBytes, err := base64.RawURLEncoding.DecodeString(eStr)
	if err != nil {
		return nil, err
	}

	n := new(big.Int).SetBytes(nBytes)
	if n.BitLen() < 2048 {
		return nil, fmt.Errorf("xjwt: RSA modulus is too small")
	}

	eBig := new(big.Int).SetBytes(eBytes)
	if !eBig.IsInt64() || eBig.Sign() <= 0 || eBig.BitLen() > 31 {
		return nil, fmt.Errorf("xjwt: invalid RSA exponent")
	}

	e := int(eBig.Int64())
	if e < 3 || e%2 == 0 {
		return nil, fmt.Errorf("xjwt: invalid RSA exponent")
	}

	return &rsa.PublicKey{N: n, E: e}, nil
}

func parseECPublicKey(crv, xStr, yStr string) (*ecdsa.PublicKey, error) {
	xBytes, err := base64.RawURLEncoding.DecodeString(xStr)
	if err != nil {
		return nil, err
	}

	yBytes, err := base64.RawURLEncoding.DecodeString(yStr)
	if err != nil {
		return nil, err
	}

	var curve elliptic.Curve

	switch crv {
	case "P-256":
		curve = elliptic.P256()
	case "P-384":
		curve = elliptic.P384()
	case "P-521":
		curve = elliptic.P521()
	default:
		return nil, fmt.Errorf("xjwt: unsupported EC curve %q", crv)
	}

	byteLen := (curve.Params().BitSize + 7) / 8
	if len(xBytes) > byteLen || len(yBytes) > byteLen {
		return nil, fmt.Errorf("xjwt: EC coordinate too large")
	}

	// Build the uncompressed SEC1 point; ParseUncompressedPublicKey both decodes
	// it into an *ecdsa.PublicKey and validates it lies on the curve.
	encoded := make([]byte, 1+2*byteLen)
	encoded[0] = 0x04
	copy(encoded[1+byteLen-len(xBytes):1+byteLen], xBytes)
	copy(encoded[1+2*byteLen-len(yBytes):], yBytes)

	pub, err := ecdsa.ParseUncompressedPublicKey(curve, encoded)
	if err != nil {
		return nil, fmt.Errorf("xjwt: EC point is not on curve")
	}

	return pub, nil
}

func parseSecp256k1PublicKey(xStr, yStr string) (*secp256k1.PublicKey, error) {
	xBytes, err := base64.RawURLEncoding.DecodeString(xStr)
	if err != nil {
		return nil, err
	}

	yBytes, err := base64.RawURLEncoding.DecodeString(yStr)
	if err != nil {
		return nil, err
	}

	return secp256k1.NewPublicKey(xBytes, yBytes)
}

func parseOKPPublicKey(crv, xStr string) (ed25519.PublicKey, error) {
	if crv != "Ed25519" {
		return nil, fmt.Errorf("xjwt: unsupported OKP curve %q", crv)
	}

	xBytes, err := base64.RawURLEncoding.DecodeString(xStr)
	if err != nil {
		return nil, err
	}

	if len(xBytes) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("xjwt: invalid Ed25519 key length %d", len(xBytes))
	}

	return ed25519.PublicKey(xBytes), nil
}

// resolverFromJWKS builds a KeyResolver that selects a key from set by kid (when
// present) and validates it against the header algorithm.
//
// Key selection binds to the kid when the token supplies one: if the header has
// a kid, only a key with that exact kid is considered. A token without a kid is
// matched against any key of a compatible type/alg. This prevents a token that
// names a specific kid from being verified by an unrelated key in the set.
type verificationKeys []any

func resolverFromJWKS(set *JWKS) KeyResolver {
	return func(h Header) (any, error) {
		if set == nil {
			return nil, fmt.Errorf("xjwt: missing JWKS")
		}
		var keys verificationKeys
		for i := range set.Keys {
			k := set.Keys[i]

			// A token that names a kid must be matched by that exact key id.
			if h.Kid != "" && k.Kid != h.Kid {
				continue
			}

			// When the JWK declares an algorithm it must match the token's.
			if k.Alg != "" && k.Alg != h.Alg {
				continue
			}

			if k.Use != "" && k.Use != "sig" {
				continue
			}
			if k.KeyOps != nil && !algAllowed("verify", k.KeyOps) {
				continue
			}

			if !matchesAlg(k.Kty, k.Crv, h.Alg) {
				continue
			}

			if key, err := k.PublicKey(h.Alg); err == nil {
				keys = append(keys, key)
			}
		}
		if len(keys) == 1 {
			return keys[0], nil
		}
		if len(keys) > 1 {
			return keys, nil
		}

		return nil, fmt.Errorf("xjwt: no matching key for kid %q alg %q", h.Kid, h.Alg)
	}
}

// VerifyWithJWKS verifies a token against a parsed JWKS, returning the payload.
func VerifyWithJWKS(token string, set *JWKS, allowedAlgs []string) ([]byte, error) {
	return Verify(token, resolverFromJWKS(set), allowedAlgs)
}
