package xjwt

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"encoding/asn1"
	"fmt"
	"math/big"
)

// A crypto.Signer lets callers hold the private key outside the process (HSM,
// KMS, PKCS#11) and still produce JWS signatures. When the key passed to Sign is
// not a concrete in-memory private key, xjwt falls back to these paths, driving
// the key through the standard crypto.Signer interface.

// signerPublic returns key as a crypto.Signer or reports a key-type mismatch.
func asSigner(key any) (crypto.Signer, error) {
	s, ok := key.(crypto.Signer)
	if !ok {
		return nil, ErrKeyTypeMismatch
	}

	return s, nil
}

// signRSAWithSigner signs via a crypto.Signer whose public key is RSA.
func signRSAWithSigner(key any, info algInfo, signingInput string) ([]byte, error) {
	signer, err := asSigner(key)
	if err != nil {
		return nil, err
	}

	pub, ok := signer.Public().(*rsa.PublicKey)
	if !ok {
		return nil, ErrKeyTypeMismatch
	}

	if pub.N.BitLen() < minRSABits {
		return nil, ErrKeyTooSmall
	}

	digest := hashInput(info.hash, signingInput)

	if info.pss {
		return signer.Sign(rand.Reader, digest, pssOptions(info.hash))
	}

	return signer.Sign(rand.Reader, digest, info.hash)
}

// signECWithSigner signs via a crypto.Signer whose public key is ECDSA. The
// signer returns an ASN.1 DER signature, which is converted to the fixed-length
// R||S form JWS requires (RFC 7518 Section 3.4).
func signECWithSigner(key any, info algInfo, signingInput string) ([]byte, error) {
	signer, err := asSigner(key)
	if err != nil {
		return nil, err
	}

	pub, ok := signer.Public().(*ecdsa.PublicKey)
	if !ok {
		return nil, ErrKeyTypeMismatch
	}

	if (pub.Curve.Params().BitSize+7)/8 != (info.ecBits+7)/8 {
		return nil, ErrKeyTypeMismatch
	}

	digest := hashInput(info.hash, signingInput)

	der, err := signer.Sign(rand.Reader, digest, info.hash)
	if err != nil {
		return nil, err
	}

	var sig struct{ R, S *big.Int }
	if _, err := asn1.Unmarshal(der, &sig); err != nil {
		return nil, fmt.Errorf("xjwt: signer returned malformed ECDSA signature: %w", err)
	}

	size := (pub.Curve.Params().BitSize + 7) / 8

	return append(leftPad(sig.R.Bytes(), size), leftPad(sig.S.Bytes(), size)...), nil
}

// signEdWithSigner signs via a crypto.Signer whose public key is Ed25519. EdDSA
// signs the message directly, so the signer is called with crypto.Hash(0).
func signEdWithSigner(key any, signingInput string) ([]byte, error) {
	signer, err := asSigner(key)
	if err != nil {
		return nil, err
	}

	if _, ok := signer.Public().(ed25519.PublicKey); !ok {
		return nil, ErrKeyTypeMismatch
	}

	return signer.Sign(rand.Reader, []byte(signingInput), crypto.Hash(0))
}
