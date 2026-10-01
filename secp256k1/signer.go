package secp256k1

import (
	"crypto"
	"encoding/asn1"
	"io"
	"math/big"
)

// ecdsaSignature is the ASN.1 DER structure for an ECDSA signature
// (SEQUENCE { r INTEGER, s INTEGER }), per SEC1 / RFC 3279.
type ecdsaSignature struct {
	R, S *big.Int
}

// SignDER signs hash under priv and returns the signature in ASN.1 DER form
// (the encoding used by X.509, TLS, and Bitcoin). The signature is canonical
// low-S, matching Sign. hash is the message digest.
func SignDER(priv *PrivateKey, hash []byte) ([]byte, error) {
	r, s := Sign(priv, hash)

	return asn1.Marshal(ecdsaSignature{R: r, S: s})
}

// VerifyDER reports whether sig, an ASN.1 DER ECDSA signature, is valid for
// hash under pub. Trailing bytes after the DER structure are rejected.
func VerifyDER(pub *PublicKey, hash, sig []byte) bool {
	var parsed ecdsaSignature

	rest, err := asn1.Unmarshal(sig, &parsed)
	if err != nil || len(rest) != 0 {
		return false
	}

	return Verify(pub, hash, parsed.R, parsed.S)
}

// VerifyDERStrict is like VerifyDER but additionally requires canonical low-S
// form, rejecting malleated signatures. See VerifyStrict.
func VerifyDERStrict(pub *PublicKey, hash, sig []byte) bool {
	var parsed ecdsaSignature

	rest, err := asn1.Unmarshal(sig, &parsed)
	if err != nil || len(rest) != 0 {
		return false
	}

	return VerifyStrict(pub, hash, parsed.R, parsed.S)
}

// Public returns the public key as a crypto.PublicKey, satisfying crypto.Signer.
func (p *PrivateKey) Public() crypto.PublicKey {
	return &p.Pub
}

// Sign implements crypto.Signer. digest is the already-hashed message; opts is
// accepted for interface compatibility but ignored (the digest is signed
// directly, as the caller is responsible for choosing the hash). The returned
// signature is ASN.1 DER, the convention for crypto.Signer implementations
// (crypto/ecdsa, crypto/rsa). For raw R||S output (e.g. JOSE ES256K) call the
// package-level Sign function instead.
func (p *PrivateKey) Sign(_ io.Reader, digest []byte, _ crypto.SignerOpts) ([]byte, error) {
	return SignDER(p, digest)
}
