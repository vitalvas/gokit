package xjwt

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/mldsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	_ "crypto/sha512" // registers SHA-384/512 for crypto.Hash.New
	"fmt"
	"math/big"

	"github.com/vitalvas/gokit/secp256k1"
)

var (
	// ErrTokenSignatureInvalid is returned when signature verification fails.
	ErrTokenSignatureInvalid = fmt.Errorf("xjwt: signature verification failed")
	// ErrKeyTypeMismatch is returned when the supplied key does not match the
	// algorithm's required key type.
	ErrKeyTypeMismatch = fmt.Errorf("xjwt: key type does not match algorithm")
	// ErrKeyTooSmall is returned when a key is below the minimum size RFC 7518
	// requires: 2048 bits for RSA (Section 3.3/3.5) or the hash output size for
	// HMAC (Section 3.2).
	ErrKeyTooSmall = fmt.Errorf("xjwt: key is smaller than the algorithm minimum")
)

// minRSABits is the minimum RSA modulus size RFC 7518 Section 3.3/3.5 mandates
// ("A key of size 2048 bits or larger MUST be used") for RS*/PS*.
const minRSABits = 2048

// keyType enumerates the public/private key families an algorithm binds to.
type keyType int

const (
	keyRSA keyType = iota
	keyEC
	keyOKP // Ed25519
	keyOctet
	keySecp256k1
	keyMLDSA // ML-DSA (FIPS-204, post-quantum)
)

// algInfo describes one JWS algorithm: the hash it uses, the key family it
// binds to, and (for EC) the expected curve bit size.
type algInfo struct {
	hash        crypto.Hash // 0 for EdDSA (no pre-hash) and ES256K (handled internally)
	keyType     keyType
	ecBits      int              // expected curve size for ES* (0 otherwise)
	pss         bool             // RSA-PSS (PS*) rather than PKCS#1 v1.5 (RS*)
	mldsaParams mldsa.Parameters // ML-DSA parameter set (zero value otherwise)
}

// algRegistry is the complete set of supported algorithms. "none" is
// deliberately absent and never resolvable.
var algRegistry = map[string]algInfo{
	RS256:   {hash: crypto.SHA256, keyType: keyRSA},
	RS384:   {hash: crypto.SHA384, keyType: keyRSA},
	RS512:   {hash: crypto.SHA512, keyType: keyRSA},
	PS256:   {hash: crypto.SHA256, keyType: keyRSA, pss: true},
	PS384:   {hash: crypto.SHA384, keyType: keyRSA, pss: true},
	PS512:   {hash: crypto.SHA512, keyType: keyRSA, pss: true},
	ES256:   {hash: crypto.SHA256, keyType: keyEC, ecBits: 256},
	ES384:   {hash: crypto.SHA384, keyType: keyEC, ecBits: 384},
	ES512:   {hash: crypto.SHA512, keyType: keyEC, ecBits: 521},
	EdDSA:   {keyType: keyOKP},
	HS256:   {hash: crypto.SHA256, keyType: keyOctet},
	HS384:   {hash: crypto.SHA384, keyType: keyOctet},
	HS512:   {hash: crypto.SHA512, keyType: keyOctet},
	ES256K:  {keyType: keySecp256k1},
	MLDSA44: {keyType: keyMLDSA, mldsaParams: mldsa.MLDSA44()},
	MLDSA65: {keyType: keyMLDSA, mldsaParams: mldsa.MLDSA65()},
	MLDSA87: {keyType: keyMLDSA, mldsaParams: mldsa.MLDSA87()},
}

// pssOptions returns the RSA-PSS options for a PS* algorithm: salt length equal
// to the hash size, as required by RFC 7518 Section 3.5.
func pssOptions(h crypto.Hash) *rsa.PSSOptions {
	return &rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthEqualsHash, Hash: h}
}

// IsSupportedAlg reports whether alg is a known signing algorithm. "none" is
// never supported.
func IsSupportedAlg(alg string) bool {
	_, ok := algRegistry[alg]

	return ok
}

// signPayload signs signingInput with key under alg and returns the raw JWS
// signature bytes. key must be the private key matching alg's key type.
func signPayload(alg, signingInput string, key any) ([]byte, error) {
	info, ok := algRegistry[alg]
	if !ok {
		return nil, fmt.Errorf("xjwt: unsupported signing algorithm %q", alg)
	}

	switch info.keyType {
	case keyRSA:
		priv, ok := key.(*rsa.PrivateKey)
		if !ok {
			return signRSAWithSigner(key, info, signingInput)
		}

		if priv.N.BitLen() < minRSABits {
			return nil, ErrKeyTooSmall
		}

		digest := hashInput(info.hash, signingInput)
		if info.pss {
			return rsa.SignPSS(rand.Reader, priv, info.hash, digest, pssOptions(info.hash))
		}

		return rsa.SignPKCS1v15(rand.Reader, priv, info.hash, digest)
	case keyEC:
		priv, ok := key.(*ecdsa.PrivateKey)
		if !ok {
			return signECWithSigner(key, info, signingInput)
		}

		return signECDSA(priv, info, signingInput)
	case keyOKP:
		priv, ok := key.(ed25519.PrivateKey)
		if !ok {
			return signEdWithSigner(key, signingInput)
		}

		return ed25519.Sign(priv, []byte(signingInput)), nil
	case keyOctet:
		secret, ok := key.([]byte)
		if !ok {
			return nil, ErrKeyTypeMismatch
		}

		if len(secret) < info.hash.Size() {
			return nil, ErrKeyTooSmall
		}

		return hmacSign(info.hash, secret, signingInput), nil
	case keySecp256k1:
		priv, ok := key.(*secp256k1.PrivateKey)
		if !ok {
			return nil, ErrKeyTypeMismatch
		}

		digest := sha256.Sum256([]byte(signingInput))
		r, s := secp256k1.Sign(priv, digest[:])
		if r == nil || s == nil {
			return nil, ErrKeyTypeMismatch
		}

		return append(leftPad(r.Bytes(), 32), leftPad(s.Bytes(), 32)...), nil
	case keyMLDSA:
		priv, ok := key.(*mldsa.PrivateKey)
		if !ok || priv == nil || priv.PublicKey().Parameters() != info.mldsaParams {
			return nil, ErrKeyTypeMismatch
		}

		// ML-DSA signs the message directly (crypto.Hash(0)).
		return priv.Sign(rand.Reader, []byte(signingInput), crypto.Hash(0))
	default:
		return nil, fmt.Errorf("xjwt: unsupported signing algorithm %q", alg)
	}
}

// verifyPayload checks sig over signingInput with key under alg. key must be
// the public key matching alg's key type.
func verifyPayload(alg, signingInput string, sig []byte, key any) error {
	if keys, ok := key.(verificationKeys); ok {
		for _, candidate := range keys {
			if verifyPayload(alg, signingInput, sig, candidate) == nil {
				return nil
			}
		}
		return ErrTokenSignatureInvalid
	}
	info, ok := algRegistry[alg]
	if !ok {
		return fmt.Errorf("xjwt: unsupported signing algorithm %q", alg)
	}

	switch info.keyType {
	case keyRSA:
		pub, ok := key.(*rsa.PublicKey)
		if !ok {
			return ErrKeyTypeMismatch
		}

		if pub.N.BitLen() < minRSABits {
			return ErrKeyTooSmall
		}

		digest := hashInput(info.hash, signingInput)
		if info.pss {
			if err := rsa.VerifyPSS(pub, info.hash, digest, sig, pssOptions(info.hash)); err != nil {
				return ErrTokenSignatureInvalid
			}

			return nil
		}

		if err := rsa.VerifyPKCS1v15(pub, info.hash, digest, sig); err != nil {
			return ErrTokenSignatureInvalid
		}

		return nil
	case keyEC:
		pub, ok := key.(*ecdsa.PublicKey)
		if !ok {
			return ErrKeyTypeMismatch
		}

		return verifyECDSA(pub, info, signingInput, sig)
	case keyOKP:
		pub, ok := key.(ed25519.PublicKey)
		if !ok {
			return ErrKeyTypeMismatch
		}

		if !ed25519.Verify(pub, []byte(signingInput), sig) {
			return ErrTokenSignatureInvalid
		}

		return nil
	case keyOctet:
		secret, ok := key.([]byte)
		if !ok {
			return ErrKeyTypeMismatch
		}

		if len(secret) < info.hash.Size() {
			return ErrKeyTooSmall
		}

		expected := hmacSign(info.hash, secret, signingInput)
		if !hmac.Equal(expected, sig) {
			return ErrTokenSignatureInvalid
		}

		return nil
	case keySecp256k1:
		pub, ok := key.(*secp256k1.PublicKey)
		if !ok {
			return ErrKeyTypeMismatch
		}

		if len(sig) != 64 {
			return ErrTokenSignatureInvalid
		}

		digest := sha256.Sum256([]byte(signingInput))
		r := new(big.Int).SetBytes(sig[:32])
		s := new(big.Int).SetBytes(sig[32:])
		if !secp256k1.Verify(pub, digest[:], r, s) {
			return ErrTokenSignatureInvalid
		}

		return nil
	case keyMLDSA:
		pub, ok := key.(*mldsa.PublicKey)
		if !ok || pub == nil || pub.Parameters() != info.mldsaParams {
			return ErrKeyTypeMismatch
		}

		if err := mldsa.Verify(pub, []byte(signingInput), sig, nil); err != nil {
			return ErrTokenSignatureInvalid
		}

		return nil
	default:
		return fmt.Errorf("xjwt: unsupported signing algorithm %q", alg)
	}
}

func signECDSA(priv *ecdsa.PrivateKey, info algInfo, signingInput string) ([]byte, error) {
	if (priv.Curve.Params().BitSize+7)/8 != (info.ecBits+7)/8 {
		return nil, ErrKeyTypeMismatch
	}

	digest := hashInput(info.hash, signingInput)

	r, s, err := ecdsa.Sign(rand.Reader, priv, digest)
	if err != nil {
		return nil, err
	}

	size := (priv.Curve.Params().BitSize + 7) / 8

	return append(leftPad(r.Bytes(), size), leftPad(s.Bytes(), size)...), nil
}

func verifyECDSA(pub *ecdsa.PublicKey, info algInfo, signingInput string, sig []byte) error {
	// Bind the alg to the exact curve so ES256 cannot be verified with a P-384
	// key, etc.
	if (pub.Curve.Params().BitSize+7)/8 != (info.ecBits+7)/8 {
		return ErrKeyTypeMismatch
	}

	size := (pub.Curve.Params().BitSize + 7) / 8

	// RFC 7518 Section 3.4: JWS ECDSA signatures are the fixed-length raw R || S, not
	// ASN.1 DER.
	if len(sig) != 2*size {
		return ErrTokenSignatureInvalid
	}

	digest := hashInput(info.hash, signingInput)
	r := new(big.Int).SetBytes(sig[:size])
	s := new(big.Int).SetBytes(sig[size:])

	if !ecdsa.Verify(pub, digest, r, s) {
		return ErrTokenSignatureInvalid
	}

	return nil
}

func hashInput(h crypto.Hash, signingInput string) []byte {
	hasher := h.New()
	hasher.Write([]byte(signingInput))

	return hasher.Sum(nil)
}

func hmacSign(h crypto.Hash, secret []byte, signingInput string) []byte {
	mac := hmac.New(h.New, secret)
	mac.Write([]byte(signingInput))

	return mac.Sum(nil)
}

func leftPad(b []byte, size int) []byte {
	if len(b) >= size {
		return b
	}

	out := make([]byte, size)
	copy(out[size-len(b):], b)

	return out
}
