package xjwt

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/mldsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"math/big"

	"github.com/vitalvas/gokit/secp256k1"
)

// JWKFromKey builds a JSONWebKey from a crypto key. Private keys yield a private
// JWK (with the private components populated); public keys yield a public JWK.
// Supported: *rsa.PrivateKey/PublicKey, *ecdsa.PrivateKey/PublicKey,
// ed25519.PrivateKey/PublicKey, *secp256k1.PrivateKey/PublicKey, and []byte
// (symmetric oct). kid is optional.
func JWKFromKey(key any, kid string) (*JSONWebKey, error) {
	jwk, err := jwkFromKey(key)
	if err != nil {
		return nil, err
	}

	jwk.Kid = kid

	return jwk, nil
}

func jwkFromKey(key any) (*JSONWebKey, error) {
	switch k := key.(type) {
	case *rsa.PrivateKey:
		jwk := rsaPublicJWK(&k.PublicKey)
		jwk.D = base64.RawURLEncoding.EncodeToString(k.D.Bytes())
		jwk.P = base64.RawURLEncoding.EncodeToString(k.Primes[0].Bytes())
		jwk.Q = base64.RawURLEncoding.EncodeToString(k.Primes[1].Bytes())
		jwk.Dp = base64.RawURLEncoding.EncodeToString(k.Precomputed.Dp.Bytes())
		jwk.Dq = base64.RawURLEncoding.EncodeToString(k.Precomputed.Dq.Bytes())
		jwk.Qi = base64.RawURLEncoding.EncodeToString(k.Precomputed.Qinv.Bytes())

		return jwk, nil
	case *rsa.PublicKey:
		return rsaPublicJWK(k), nil
	case *ecdsa.PrivateKey:
		jwk, err := ecPublicJWK(&k.PublicKey)
		if err != nil {
			return nil, err
		}
		dBytes, err := k.Bytes()
		if err != nil {
			return nil, err
		}
		jwk.D = base64.RawURLEncoding.EncodeToString(dBytes)

		return jwk, nil
	case *ecdsa.PublicKey:
		return ecPublicJWK(k)
	case ed25519.PrivateKey:
		pub, ok := k.Public().(ed25519.PublicKey)
		if !ok {
			return nil, fmt.Errorf("xjwt: invalid Ed25519 private key")
		}
		jwk := okpPublicJWK(pub)
		jwk.D = base64.RawURLEncoding.EncodeToString(k.Seed())

		return jwk, nil
	case ed25519.PublicKey:
		return okpPublicJWK(k), nil
	case *secp256k1.PrivateKey:
		jwk := secpPublicJWK(&k.Pub)
		jwk.D = base64.RawURLEncoding.EncodeToString(k.Serialize())

		return jwk, nil
	case *secp256k1.PublicKey:
		return secpPublicJWK(k), nil
	case *mldsa.PrivateKey:
		jwk := mldsaPublicJWK(k.PublicKey())
		jwk.Priv = base64.RawURLEncoding.EncodeToString(k.Bytes())

		return jwk, nil
	case *mldsa.PublicKey:
		return mldsaPublicJWK(k), nil
	case []byte:
		return &JSONWebKey{Kty: "oct", K: base64.RawURLEncoding.EncodeToString(k)}, nil
	default:
		return nil, fmt.Errorf("xjwt: unsupported key type %T", key)
	}
}

func rsaPublicJWK(pub *rsa.PublicKey) *JSONWebKey {
	eBytes := big.NewInt(int64(pub.E)).Bytes()

	return &JSONWebKey{Kty: "RSA", N: base64.RawURLEncoding.EncodeToString(pub.N.Bytes()), E: base64.RawURLEncoding.EncodeToString(eBytes)}
}

func ecPublicJWK(pub *ecdsa.PublicKey) (*JSONWebKey, error) {
	crv, byteLen, err := curveName(pub.Curve)
	if err != nil {
		return nil, err
	}

	raw, err := pub.Bytes() // uncompressed SEC1: 0x04 || X || Y
	if err != nil {
		return nil, err
	}
	x := raw[1 : 1+byteLen]
	y := raw[1+byteLen:]

	return &JSONWebKey{Kty: "EC", Crv: crv, X: base64.RawURLEncoding.EncodeToString(x), Y: base64.RawURLEncoding.EncodeToString(y)}, nil
}

func secpPublicJWK(pub *secp256k1.PublicKey) *JSONWebKey {
	raw := pub.SerializeUncompressed() // 0x04 || X(32) || Y(32)

	return &JSONWebKey{Kty: "EC", Crv: "secp256k1", X: base64.RawURLEncoding.EncodeToString(raw[1:33]), Y: base64.RawURLEncoding.EncodeToString(raw[33:])}
}

func okpPublicJWK(pub ed25519.PublicKey) *JSONWebKey {
	return &JSONWebKey{Kty: "OKP", Crv: "Ed25519", X: base64.RawURLEncoding.EncodeToString(pub)}
}

func curveName(c elliptic.Curve) (name string, byteLen int, err error) {
	switch c {
	case elliptic.P256():
		return "P-256", 32, nil
	case elliptic.P384():
		return "P-384", 48, nil
	case elliptic.P521():
		return "P-521", 66, nil
	default:
		return "", 0, fmt.Errorf("xjwt: unsupported EC curve")
	}
}

// PrivateKey converts a private JWK to its crypto private key: *rsa.PrivateKey,
// *ecdsa.PrivateKey, ed25519.PrivateKey, *secp256k1.PrivateKey, *mldsa.PrivateKey
// (AKP), or []byte (oct).
func (k JSONWebKey) PrivateKey() (any, error) {
	switch k.Kty {
	case "RSA":
		return k.rsaPrivateKey()
	case "EC":
		return k.ecPrivateKey()
	case "OKP":
		return k.okpPrivateKey()
	case "AKP":
		return k.mldsaPrivateKey()
	case "oct":
		return base64.RawURLEncoding.DecodeString(k.K)
	default:
		return nil, fmt.Errorf("xjwt: unsupported key type %q", k.Kty)
	}
}

func (k JSONWebKey) rsaPrivateKey() (*rsa.PrivateKey, error) {
	if k.D == "" {
		return nil, fmt.Errorf("xjwt: JWK is not an RSA private key")
	}

	pub, err := parseRSAPublicKey(k.N, k.E)
	if err != nil {
		return nil, err
	}

	d, err := base64.RawURLEncoding.DecodeString(k.D)
	if err != nil {
		return nil, err
	}

	p, err := base64.RawURLEncoding.DecodeString(k.P)
	if err != nil {
		return nil, err
	}

	q, err := base64.RawURLEncoding.DecodeString(k.Q)
	if err != nil {
		return nil, err
	}

	priv := &rsa.PrivateKey{
		PublicKey: *pub,
		D:         new(big.Int).SetBytes(d),
		Primes:    []*big.Int{new(big.Int).SetBytes(p), new(big.Int).SetBytes(q)},
	}
	priv.Precompute()

	if err := priv.Validate(); err != nil {
		return nil, fmt.Errorf("xjwt: invalid RSA private key: %w", err)
	}

	return priv, nil
}

func (k JSONWebKey) ecPrivateKey() (any, error) {
	if k.D == "" {
		return nil, fmt.Errorf("xjwt: JWK is not an EC private key")
	}

	d, err := base64.RawURLEncoding.DecodeString(k.D)
	if err != nil {
		return nil, err
	}

	if k.Crv == "secp256k1" {
		return secp256k1.PrivKeyFromBytes(d)
	}

	var curve elliptic.Curve
	switch k.Crv {
	case "P-256":
		curve = elliptic.P256()
	case "P-384":
		curve = elliptic.P384()
	case "P-521":
		curve = elliptic.P521()
	default:
		return nil, fmt.Errorf("xjwt: unsupported EC curve %q", k.Crv)
	}

	// ParseRawPrivateKey builds and validates the key (deriving the public point)
	// without the deprecated direct-field assignment.
	return ecdsa.ParseRawPrivateKey(curve, d)
}

func (k JSONWebKey) okpPrivateKey() (ed25519.PrivateKey, error) {
	if k.D == "" {
		return nil, fmt.Errorf("xjwt: JWK is not an OKP private key")
	}

	seed, err := base64.RawURLEncoding.DecodeString(k.D)
	if err != nil {
		return nil, err
	}

	if len(seed) != ed25519.SeedSize {
		return nil, fmt.Errorf("xjwt: invalid Ed25519 seed length %d", len(seed))
	}

	return ed25519.NewKeyFromSeed(seed), nil
}

// PublicJWK returns a copy of k with all private components stripped, suitable
// for publishing in a JWKS.
func (k JSONWebKey) PublicJWK() JSONWebKey {
	k.D, k.P, k.Q, k.Dp, k.Dq, k.Qi, k.K, k.Priv = "", "", "", "", "", "", "", ""

	return k
}

// GenerateKey generates a new private key for the given JWS algorithm and
// returns it both as a crypto key and as a private JSONWebKey. For HS* a random
// secret of the hash's output size is generated.
func GenerateKey(alg, kid string) (key any, jwk *JSONWebKey, err error) {
	switch alg {
	case RS256, RS384, RS512, PS256, PS384, PS512:
		key, err = rsa.GenerateKey(rand.Reader, 2048)
	case ES256:
		key, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	case ES384:
		key, err = ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	case ES512:
		key, err = ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
	case ES256K:
		key, err = secp256k1.GeneratePrivateKey()
	case EdDSA:
		_, key, err = ed25519.GenerateKey(rand.Reader)
	case MLDSA44, MLDSA65, MLDSA87:
		key, err = mldsa.GenerateKey(algRegistry[alg].mldsaParams)
	case HS256, HS384, HS512:
		secret := make([]byte, hmacSecretLen(alg))
		if _, err = rand.Read(secret); err != nil {
			return nil, nil, err
		}
		key = secret
	default:
		return nil, nil, fmt.Errorf("xjwt: cannot generate key for alg %q", alg)
	}

	if err != nil {
		return nil, nil, err
	}

	jwk, err = JWKFromKey(key, kid)
	if err != nil {
		return nil, nil, err
	}

	jwk.Alg = alg
	jwk.Use = "sig"

	return key, jwk, nil
}

func hmacSecretLen(alg string) int {
	switch alg {
	case HS384:
		return 48
	case HS512:
		return 64
	default:
		return 32
	}
}

// MarshalPublicKeyToPEM encodes a public key as a PKIX "PUBLIC KEY" PEM block.
// secp256k1 public keys are not representable in PKIX via crypto/x509; use the
// secp256k1 package's own encoders for those.
func MarshalPublicKeyToPEM(pub any) ([]byte, error) {
	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return nil, fmt.Errorf("xjwt: marshaling public key: %w", err)
	}

	return pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}), nil
}

// MarshalPrivateKeyToPEM encodes a private key as a PKCS#8 "PRIVATE KEY" PEM
// block. secp256k1 private keys are not representable via crypto/x509; use the
// secp256k1 package's own encoders for those.
func MarshalPrivateKeyToPEM(priv any) ([]byte, error) {
	der, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return nil, fmt.Errorf("xjwt: marshaling private key: %w", err)
	}

	return pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}), nil
}
