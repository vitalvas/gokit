package xjwt

import (
	"crypto/ecdh"
	"crypto/ecdsa"
	"encoding/base64"
	"fmt"
)

// cekLength returns the content-encryption key length in bytes for a JWE "enc"
// algorithm.
func cekLength(enc string) (int, error) {
	switch enc {
	case A128CBCHS256:
		return 32, nil
	case A192CBCHS384:
		return 48, nil
	case A256CBCHS512:
		return 64, nil
	case A128GCM:
		return 16, nil
	case A192GCM:
		return 24, nil
	case A256GCM:
		return 32, nil
	default:
		return 0, fmt.Errorf("xjwt: unsupported content encryption algorithm %q", enc)
	}
}

// toECDHPublic converts a supported EC public key to a *ecdh.PublicKey. It
// accepts *ecdh.PublicKey and *ecdsa.PublicKey (NIST curves).
func toECDHPublic(key any) (*ecdh.PublicKey, error) {
	switch k := key.(type) {
	case *ecdh.PublicKey:
		return k, nil
	case *ecdsa.PublicKey:
		return k.ECDH()
	default:
		return nil, fmt.Errorf("xjwt: unsupported ECDH public key type %T", key)
	}
}

// toECDHPrivate converts a supported EC private key to a *ecdh.PrivateKey.
func toECDHPrivate(key any) (*ecdh.PrivateKey, error) {
	switch k := key.(type) {
	case *ecdh.PrivateKey:
		return k, nil
	case *ecdsa.PrivateKey:
		return k.ECDH()
	default:
		return nil, fmt.Errorf("xjwt: unsupported ECDH private key type %T", key)
	}
}

// ecdhCurve maps a JWK "crv" to its ecdh.Curve.
func ecdhCurve(crv string) (ecdh.Curve, error) {
	switch crv {
	case "P-256":
		return ecdh.P256(), nil
	case "P-384":
		return ecdh.P384(), nil
	case "P-521":
		return ecdh.P521(), nil
	case "X25519":
		return ecdh.X25519(), nil
	default:
		return nil, fmt.Errorf("xjwt: unsupported ECDH curve %q", crv)
	}
}

// ecdhPublicJWK builds an "epk" JWK from an ephemeral ECDH public key. NIST
// curves use kty=EC with x and y; X25519 uses kty=OKP with the raw key as x.
func ecdhPublicJWK(pub *ecdh.PublicKey) (*JSONWebKey, error) {
	crv, err := ecdhCurveFromKey(pub.Curve())
	if err != nil {
		return nil, err
	}

	raw := pub.Bytes()

	if crv == "X25519" {
		return &JSONWebKey{Kty: "OKP", Crv: "X25519", X: base64.RawURLEncoding.EncodeToString(raw)}, nil
	}

	// NIST: uncompressed 0x04 || X || Y.
	byteLen := (len(raw) - 1) / 2

	return &JSONWebKey{
		Kty: "EC",
		Crv: crv,
		X:   base64.RawURLEncoding.EncodeToString(raw[1 : 1+byteLen]),
		Y:   base64.RawURLEncoding.EncodeToString(raw[1+byteLen:]),
	}, nil
}

// ecdhPublicFromJWK rebuilds an ECDH public key from an "epk" JWK.
func ecdhPublicFromJWK(jwk *JSONWebKey) (*ecdh.PublicKey, error) {
	curve, err := ecdhCurve(jwk.Crv)
	if err != nil {
		return nil, err
	}

	x, err := base64.RawURLEncoding.DecodeString(jwk.X)
	if err != nil {
		return nil, err
	}

	if jwk.Crv == "X25519" {
		return curve.NewPublicKey(x) // raw 32-byte key
	}

	y, err := base64.RawURLEncoding.DecodeString(jwk.Y)
	if err != nil {
		return nil, err
	}

	byteLen := curveByteLen(jwk.Crv)
	encoded := make([]byte, 1+2*byteLen)
	encoded[0] = 0x04
	copy(encoded[1+byteLen-len(x):1+byteLen], x)
	copy(encoded[1+2*byteLen-len(y):], y)

	return curve.NewPublicKey(encoded)
}

func ecdhCurveFromKey(c ecdh.Curve) (string, error) {
	switch c {
	case ecdh.P256():
		return "P-256", nil
	case ecdh.P384():
		return "P-384", nil
	case ecdh.P521():
		return "P-521", nil
	case ecdh.X25519():
		return "X25519", nil
	default:
		return "", fmt.Errorf("xjwt: unsupported ECDH curve")
	}
}

func curveByteLen(crv string) int {
	switch crv {
	case "P-384":
		return 48
	case "P-521":
		return 66
	default:
		return 32
	}
}
