package xjwt

import (
	"crypto"
	_ "crypto/sha256" // register SHA-256 for crypto.Hash.New
	_ "crypto/sha512" // register SHA-384/512
	"encoding/base64"
	"encoding/json"
	"fmt"
)

// JWKThumbprint computes the RFC 7638 JWK Thumbprint (SHA-256) of a JWK document.
func JWKThumbprint(jwkJSON json.RawMessage) (string, error) {
	var jwk JSONWebKey
	if err := json.Unmarshal(jwkJSON, &jwk); err != nil {
		return "", err
	}

	tp, err := jwk.Thumbprint(crypto.SHA256)
	if err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(tp), nil
}

// Thumbprint computes the RFC 7638 JWK Thumbprint of the key using the given
// hash. The canonical form contains only the required members in lexicographic
// order (guaranteed by json.Marshal of a map).
func (k JSONWebKey) Thumbprint(h crypto.Hash) ([]byte, error) {
	canonical, err := k.canonicalThumbprintJSON()
	if err != nil {
		return nil, err
	}

	hasher := h.New()
	hasher.Write(canonical)

	return hasher.Sum(nil), nil
}

func (k JSONWebKey) canonicalThumbprintJSON() ([]byte, error) {
	switch k.Kty {
	case "EC":
		if k.Crv == "" || k.X == "" || k.Y == "" {
			return nil, fmt.Errorf("xjwt: EC JWK missing required thumbprint member")
		}

		return json.Marshal(map[string]string{"crv": k.Crv, "kty": k.Kty, "x": k.X, "y": k.Y})
	case "RSA":
		if k.E == "" || k.N == "" {
			return nil, fmt.Errorf("xjwt: RSA JWK missing required thumbprint member")
		}

		return json.Marshal(map[string]string{"e": k.E, "kty": k.Kty, "n": k.N})
	case "OKP":
		if k.Crv == "" || k.X == "" {
			return nil, fmt.Errorf("xjwt: OKP JWK missing required thumbprint member")
		}

		return json.Marshal(map[string]string{"crv": k.Crv, "kty": k.Kty, "x": k.X})
	case "oct":
		if k.K == "" {
			return nil, fmt.Errorf("xjwt: oct JWK missing required thumbprint member")
		}

		return json.Marshal(map[string]string{"k": k.K, "kty": k.Kty})
	default:
		return nil, fmt.Errorf("xjwt: unsupported key type for thumbprint: %s", k.Kty)
	}
}

// AssignKeyID sets the key's Kid to its RFC 7638 SHA-256 thumbprint.
func (k *JSONWebKey) AssignKeyID() error {
	tp, err := k.Thumbprint(crypto.SHA256)
	if err != nil {
		return err
	}

	k.Kid = base64.RawURLEncoding.EncodeToString(tp)

	return nil
}
