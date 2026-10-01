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

	// d' = priv scalar; if the pubkey has odd Y, negate d so P has even Y.
	d := new(big.Int).Set(priv.D)
	px, py := scalarBaseMult(d)
	if py.Bit(0) == 1 {
		d.Sub(orderN, d)
	}

	pBytes := fixedBytes(px)

	// aux = 32 random bytes; t = d XOR tagged_hash("BIP0340/aux", aux)
	aux := make([]byte, 32)
	if _, err := io.ReadFull(auxRand, aux); err != nil {
		return nil, err
	}

	t := new(big.Int).Xor(d, new(big.Int).SetBytes(taggedHash("BIP0340/aux", aux)))

	// nonce k0 = tagged_hash("BIP0340/nonce", t || P.x || msg) mod n
	k0 := new(big.Int).SetBytes(taggedHash("BIP0340/nonce", fixedBytes(t), pBytes, msg))
	k0.Mod(k0, orderN)
	if k0.Sign() == 0 {
		return nil, errBadKey("schnorr nonce is zero")
	}

	// R = k0*G; if R has odd Y, negate k so R has even Y.
	rx, ry := scalarBaseMult(k0)
	k := k0
	if ry.Bit(0) == 1 {
		k = new(big.Int).Sub(orderN, k0)
	}

	rBytes := fixedBytes(rx)

	// e = tagged_hash("BIP0340/challenge", R.x || P.x || msg) mod n
	e := new(big.Int).SetBytes(taggedHash("BIP0340/challenge", rBytes, pBytes, msg))
	e.Mod(e, orderN)

	// s = (k + e*d) mod n
	s := new(big.Int).Mul(e, d)
	s.Add(s, k)
	s.Mod(s, orderN)

	sig := make([]byte, SchnorrSignatureLen)
	copy(sig[:32], rBytes)
	copy(sig[32:], fixedBytes(s))

	// Self-verify, as recommended by BIP-340, to catch faults.
	if !VerifySchnorr(&PublicKey{X: px, Y: py}, msg, sig) {
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
