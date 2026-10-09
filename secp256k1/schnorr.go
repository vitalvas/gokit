package secp256k1

import (
	"crypto/rand"
	"crypto/sha256"
	"io"
	"math/big"
)

// SchnorrSignatureLen is the fixed length of a BIP-340 Schnorr signature
// (R.x || s), and XOnlyPubKeyLen is the length of an x-only public key.
const (
	SchnorrSignatureLen = 64
	XOnlyPubKeyLen      = 32
)

// taggedHash implements BIP-340's tagged hashing:
// SHA256(SHA256(tag) || SHA256(tag) || msg).
func taggedHash(tag string, parts ...[]byte) []byte {
	tagSum := sha256.Sum256([]byte(tag))

	h := sha256.New()
	h.Write(tagSum[:])
	h.Write(tagSum[:])

	for _, p := range parts {
		h.Write(p)
	}

	return h.Sum(nil)
}

// SerializeXOnly returns the 32-byte BIP-340 x-only encoding of the public key
// (the X coordinate; Y parity is dropped and assumed even on parse).
func (p *PublicKey) SerializeXOnly() []byte {
	return fixedBytes(p.X)
}

// ParseXOnlyPubKey decodes a 32-byte BIP-340 x-only public key, lifting it to
// the point with even Y as required by the spec.
func ParseXOnlyPubKey(serialized []byte) (*PublicKey, error) {
	if len(serialized) != XOnlyPubKeyLen {
		return nil, errBadKey("x-only public key must be %d bytes, got %d", XOnlyPubKeyLen, len(serialized))
	}

	x := new(big.Int).SetBytes(serialized)

	// BIP-340: the implied point is the one with even Y.
	y, err := decompressY(x, false)
	if err != nil {
		return nil, err
	}

	return &PublicKey{X: x, Y: y}, nil
}

// SignSchnorr produces a BIP-340 Schnorr signature over the 32-byte message msg
// under priv, drawing auxiliary randomness from crypto/rand. The returned
// signature is 64 bytes (R.x || s).
//
// BIP-340 is defined for a 32-byte message (typically a hash of the real
// message); msg must be exactly 32 bytes.
func SignSchnorr(priv *PrivateKey, msg []byte) ([]byte, error) {
	return signSchnorr(priv, msg, rand.Reader)
}

func signSchnorr(priv *PrivateKey, msg []byte, auxRand io.Reader) ([]byte, error) {
	if len(msg) != 32 {
		return nil, errBadKey("schnorr message must be 32 bytes, got %d", len(msg))
	}

	d, err := priv.secretScalar()
	if err != nil {
		return nil, err
	}
	pub := newSecretPoint().base().multiply(d).encodeUncompressed()
	negD := newScalarValue().subtract(d)
	_ = d.selectValue(uint64(pub[64]&1), d, negD)
	pBytes := pub[1:33]

	aux := make([]byte, 32)
	if _, err := io.ReadFull(auxRand, aux); err != nil {
		return nil, err
	}
	t := d.encode()
	auxHash := taggedHash("BIP0340/aux", aux)
	for i := range t {
		t[i] ^= auxHash[i]
	}
	k := reducedScalar(taggedHash("BIP0340/nonce", t, pBytes, msg))
	if k.isZero() {
		return nil, errBadKey("schnorr nonce is zero")
	}
	point := newSecretPoint().base().multiply(k).encodeUncompressed()
	negK := newScalarValue().subtract(k)
	_ = k.selectValue(uint64(point[64]&1), k, negK)
	rBytes := point[1:33]
	e := reducedScalar(taggedHash("BIP0340/challenge", rBytes, pBytes, msg))
	s := e.multiply(d).add(k)
	sig := append(append([]byte(nil), rBytes...), s.encode()...)

	// Self-verify, as recommended by BIP-340, to catch faults.
	if !VerifySchnorr(priv.PubKey(), msg, sig) {
		return nil, errBadKey("schnorr self-verification failed")
	}

	return sig, nil
}

// VerifySchnorr reports whether sig is a valid BIP-340 Schnorr signature of the
// 32-byte msg under pub. pub is treated as an x-only key (its X is used; Y is
// re-derived as even), matching the BIP-340 verification algorithm.
func VerifySchnorr(pub *PublicKey, msg, sig []byte) bool {
	if pub == nil || pub.X == nil || len(msg) != 32 || len(sig) != SchnorrSignatureLen {
		return false
	}

	// Lift P to even Y per BIP-340.
	px := new(big.Int).Set(pub.X)
	py, err := decompressY(px, false)
	if err != nil {
		return false
	}

	r := new(big.Int).SetBytes(sig[:32])
	s := new(big.Int).SetBytes(sig[32:])

	// Reject out-of-range r (>= p) and s (>= n).
	if r.Cmp(primeP) >= 0 || s.Cmp(orderN) >= 0 {
		return false
	}

	// e = tagged_hash("BIP0340/challenge", r || P.x || msg) mod n
	e := new(big.Int).SetBytes(taggedHash("BIP0340/challenge", fixedBytes(r), fixedBytes(px), msg))
	e.Mod(e, orderN)

	// R = s*G - e*P
	sG := scalarMult(affineToJacobian(gX, gY), s)
	negE := new(big.Int).Sub(orderN, e)
	negEP := scalarMult(affineToJacobian(px, py), negE)
	rPoint := sG.add(negEP)

	if rPoint.isIdentity() {
		return false
	}

	rx, ry := rPoint.toAffine()

	// Valid iff R has even Y and R.x == r.
	return ry.Bit(0) == 0 && rx.Cmp(r) == 0
}
