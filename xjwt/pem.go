package xjwt

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
)

// Sentinel errors for PEM key parsing. Callers can match them with errors.Is.
var (
	// ErrKeyMustBePEMEncoded is returned when the input is not a PEM block.
	ErrKeyMustBePEMEncoded = fmt.Errorf("xjwt: key must be PEM encoded")
	// ErrNotRSAKey is returned when a PEM key is not an RSA key of the expected kind.
	ErrNotRSAKey = fmt.Errorf("xjwt: key is not a valid RSA key")
	// ErrNotECKey is returned when a PEM key is not an ECDSA key of the expected kind.
	ErrNotECKey = fmt.Errorf("xjwt: key is not a valid ECDSA key")
	// ErrNotEdKey is returned when a PEM key is not an Ed25519 key of the expected kind.
	ErrNotEdKey = fmt.Errorf("xjwt: key is not a valid Ed25519 key")
)

// decodePEM extracts the DER bytes from the first PEM block in data.
func decodePEM(data []byte) ([]byte, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, ErrKeyMustBePEMEncoded
	}

	return block.Bytes, nil
}

// parsePKCS8OrLegacyPrivate parses a DER private key in PKCS#8 form, falling
// back to the key-specific legacy encodings (PKCS#1 for RSA, SEC1 for EC).
func parsePKCS8OrLegacyPrivate(der []byte) (crypto.PrivateKey, error) {
	if key, err := x509.ParsePKCS8PrivateKey(der); err == nil {
		return key, nil
	}

	if key, err := x509.ParsePKCS1PrivateKey(der); err == nil {
		return key, nil
	}

	return x509.ParseECPrivateKey(der)
}

// ParseRSAPrivateKeyFromPEM parses a PEM-encoded PKCS#1 or PKCS#8 RSA private key.
func ParseRSAPrivateKeyFromPEM(data []byte) (*rsa.PrivateKey, error) {
	der, err := decodePEM(data)
	if err != nil {
		return nil, err
	}

	key, err := parsePKCS8OrLegacyPrivate(der)
	if err != nil {
		return nil, err
	}

	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, ErrNotRSAKey
	}

	return rsaKey, nil
}

// ParseRSAPublicKeyFromPEM parses a PEM-encoded PKIX or PKCS#1 RSA public key.
func ParseRSAPublicKeyFromPEM(data []byte) (*rsa.PublicKey, error) {
	der, err := decodePEM(data)
	if err != nil {
		return nil, err
	}

	key, err := x509.ParsePKIXPublicKey(der)
	if err != nil {
		if legacy, errLegacy := x509.ParsePKCS1PublicKey(der); errLegacy == nil {
			return legacy, nil
		}

		return nil, ErrNotRSAKey
	}

	rsaKey, ok := key.(*rsa.PublicKey)
	if !ok {
		return nil, ErrNotRSAKey
	}

	return rsaKey, nil
}

// ParseECPrivateKeyFromPEM parses a PEM-encoded SEC1 or PKCS#8 ECDSA private key.
func ParseECPrivateKeyFromPEM(data []byte) (*ecdsa.PrivateKey, error) {
	der, err := decodePEM(data)
	if err != nil {
		return nil, err
	}

	key, err := parsePKCS8OrLegacyPrivate(der)
	if err != nil {
		return nil, err
	}

	ecKey, ok := key.(*ecdsa.PrivateKey)
	if !ok {
		return nil, ErrNotECKey
	}

	return ecKey, nil
}

// ParseECPublicKeyFromPEM parses a PEM-encoded PKIX ECDSA public key.
func ParseECPublicKeyFromPEM(data []byte) (*ecdsa.PublicKey, error) {
	der, err := decodePEM(data)
	if err != nil {
		return nil, err
	}

	key, err := x509.ParsePKIXPublicKey(der)
	if err != nil {
		return nil, ErrNotECKey
	}

	ecKey, ok := key.(*ecdsa.PublicKey)
	if !ok {
		return nil, ErrNotECKey
	}

	return ecKey, nil
}

// ParseEdPrivateKeyFromPEM parses a PEM-encoded PKCS#8 Ed25519 private key.
func ParseEdPrivateKeyFromPEM(data []byte) (ed25519.PrivateKey, error) {
	der, err := decodePEM(data)
	if err != nil {
		return nil, err
	}

	key, err := x509.ParsePKCS8PrivateKey(der)
	if err != nil {
		return nil, ErrNotEdKey
	}

	edKey, ok := key.(ed25519.PrivateKey)
	if !ok {
		return nil, ErrNotEdKey
	}

	return edKey, nil
}

// ParseEdPublicKeyFromPEM parses a PEM-encoded PKIX Ed25519 public key.
func ParseEdPublicKeyFromPEM(data []byte) (ed25519.PublicKey, error) {
	der, err := decodePEM(data)
	if err != nil {
		return nil, err
	}

	key, err := x509.ParsePKIXPublicKey(der)
	if err != nil {
		return nil, ErrNotEdKey
	}

	edKey, ok := key.(ed25519.PublicKey)
	if !ok {
		return nil, ErrNotEdKey
	}

	return edKey, nil
}
