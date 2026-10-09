package xjwt

import (
	"crypto/mldsa"
	"encoding/base64"
	"fmt"
)

// This file holds the ML-DSA (FIPS-204) JWK helpers. The JOSE representation is
// the AKP (Algorithm Key Pair) key type from draft-ietf-jose-pqc: "pub" is the
// public key and "priv" the private seed, both base64url, with "alg" naming the
// parameter set (e.g. "ML-DSA-65"). Sign/verify dispatch lives in alg.go.

// mldsaPublicJWK builds an AKP public JWK from an ML-DSA public key. The caller
// must set Alg to the parameter-set identifier (e.g. "ML-DSA-65"); GenerateKey
// does this automatically.
func mldsaPublicJWK(pub *mldsa.PublicKey) *JSONWebKey {
	return &JSONWebKey{Kty: "AKP", Pub: base64.RawURLEncoding.EncodeToString(pub.Bytes())}
}

// mldsaPrivateKey reconstructs an ML-DSA private key from an AKP private JWK.
func (k JSONWebKey) mldsaPrivateKey() (*mldsa.PrivateKey, error) {
	if k.Priv == "" {
		return nil, fmt.Errorf("xjwt: JWK is not an AKP private key")
	}

	params, err := mldsaParamsForAlg(k.Alg)
	if err != nil {
		return nil, err
	}

	seed, err := base64.RawURLEncoding.DecodeString(k.Priv)
	if err != nil {
		return nil, err
	}

	return mldsa.NewPrivateKey(params, seed)
}

// parseMLDSAPublicKey reconstructs an ML-DSA public key from an AKP public JWK.
func parseMLDSAPublicKey(alg, pubStr string) (*mldsa.PublicKey, error) {
	params, err := mldsaParamsForAlg(alg)
	if err != nil {
		return nil, err
	}

	pubBytes, err := base64.RawURLEncoding.DecodeString(pubStr)
	if err != nil {
		return nil, err
	}

	return mldsa.NewPublicKey(params, pubBytes)
}

// mldsaParamsForAlg maps an AKP JWK's alg to its ML-DSA parameter set.
func mldsaParamsForAlg(alg string) (mldsa.Parameters, error) {
	info, ok := algRegistry[alg]
	if !ok || info.keyType != keyMLDSA {
		return mldsa.Parameters{}, fmt.Errorf("xjwt: AKP JWK has invalid or missing ML-DSA alg %q", alg)
	}

	return info.mldsaParams, nil
}
