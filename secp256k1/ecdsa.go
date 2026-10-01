package secp256k1

import (
	"crypto/hmac"
	"crypto/sha256"
	"math/big"
)

// halfOrder is N/2, used for low-S normalization (BIP-0062 / RFC 8812 require
// canonical low-S signatures).
var halfOrder = new(big.Int).Rsh(orderN, 1)

// Verify reports whether (r, s) is a valid ECDSA signature of hash under pub.
// hash is the already-computed message digest (for ES256K, SHA-256 of the
// signing input). r and s must be in [1, N-1].
func Verify(pub *PublicKey, hash []byte, r, s *big.Int) bool {
	if pub == nil || r == nil || s == nil {
		return false
	}

	if r.Sign() <= 0 || r.Cmp(orderN) >= 0 || s.Sign() <= 0 || s.Cmp(orderN) >= 0 {
		return false
	}

	if !isOnCurve(pub.X, pub.Y) {
		return false
	}

	e := hashToInt(hash)

	// Unreachable for valid input: s is already constrained to [1, n-1] above,
	// and every such value is invertible modulo the prime order n. Kept as a
	// defensive guard.
	wInv := new(big.Int).ModInverse(s, orderN)
	if wInv == nil {
		return false
	}

	u1 := new(big.Int).Mul(e, wInv)
	u1.Mod(u1, orderN)
	u2 := new(big.Int).Mul(r, wInv)
	u2.Mod(u2, orderN)

	// u1*G + u2*Pub
	g := affineToJacobian(gX, gY)
	point := scalarMult(g, u1).add(scalarMult(affineToJacobian(pub.X, pub.Y), u2))

	// Unreachable for valid input: the sum resolves to the identity only for
	// signatures forged against the discrete-log problem. Defensive guard.
	if point.isIdentity() {
		return false
	}

	x, _ := point.toAffine()
	x.Mod(x, orderN)

	return x.Cmp(r) == 0
}

// VerifyStrict is like Verify but additionally requires the signature to be in
// canonical low-S form (s <= N/2), the form Sign always produces. This rejects
// the malleated variant (r, N-s) that Verify accepts, which matters for
// consensus or contexts where a signature's bytes must be unique (e.g. Bitcoin,
// BIP-0062). For JOSE/ES256K, where only validity matters, plain Verify is fine.
func VerifyStrict(pub *PublicKey, hash []byte, r, s *big.Int) bool {
	if s == nil || s.Cmp(halfOrder) > 0 {
		return false
	}

	return Verify(pub, hash, r, s)
}

// Sign produces a deterministic ECDSA signature (RFC 6979) of hash under priv
// and returns the canonical low-S form. hash is the message digest.
//
// Sign is RFC 6979 deterministic (no nonce reuse / RNG risk) but, like the
// rest of this package, is not constant-time; see the package documentation.
// nonceFunc supplies the per-signature nonce. It defaults to RFC 6979 and is a
// package variable only so tests can inject degenerate nonces to exercise the
// (otherwise unreachable) retry branches below.
var nonceFunc = rfc6979Nonce

func Sign(priv *PrivateKey, hash []byte) (r, s *big.Int) {
	r, s, _ = signWithRecovery(priv, hash)

	return r, s
}

// signWithRecovery is the core ECDSA signing loop. In addition to the canonical
// low-S signature it returns the recovery id in [0,3] that lets RecoverPubKey
// reconstruct the public key: bit 0 is the parity of the nonce point's Y
// coordinate, bit 1 is set when that point's X exceeded the group order (and so
// was reduced). Low-S normalization flips the Y parity, so recID tracks that.
func signWithRecovery(priv *PrivateKey, hash []byte) (r, s *big.Int, recID int) {
	e := hashToInt(hash)

	for i := 0; ; i++ {
		k := nonceFunc(priv.D, hash, i)
		if k.Sign() == 0 || k.Cmp(orderN) >= 0 {
			continue
		}

		x, y := scalarBaseMult(k)
		r = new(big.Int).Mod(x, orderN)
		// Effectively unreachable: requires a nonce whose point x-coordinate is
		// a multiple of n. Standard ECDSA retry guard.
		if r.Sign() == 0 {
			continue
		}

		kInv := new(big.Int).ModInverse(k, orderN)
		if kInv == nil {
			continue
		}

		// s = k^-1 * (e + r*d) mod n
		s = new(big.Int).Mul(r, priv.D)
		s.Add(s, e)
		s.Mul(s, kInv)
		s.Mod(s, orderN)
		if s.Sign() == 0 {
			continue
		}

		recID = int(y.Bit(0))
		if x.Cmp(orderN) >= 0 {
			recID |= 2
		}

		// Canonical low-S: if s > n/2, use n - s. Negating s is equivalent to
		// negating the nonce point, which flips its Y parity.
		if s.Cmp(halfOrder) > 0 {
			s.Sub(orderN, s)
			recID ^= 1
		}

		return r, s, recID
	}
}

// hashToInt converts a digest to an integer per FIPS 186-4 / SEC1: take the
// leftmost N bits (N = bit length of the group order). For SHA-256 and
// secp256k1 both are 256 bits, so this is a plain big-endian decode.
func hashToInt(hash []byte) *big.Int {
	orderBits := orderN.BitLen()
	orderBytes := (orderBits + 7) / 8

	if len(hash) > orderBytes {
		hash = hash[:orderBytes]
	}

	ret := new(big.Int).SetBytes(hash)

	// Dead for secp256k1 specifically (order is exactly 256 bits, so excess is
	// always 0 after the truncation above); retained for curves whose order is
	// not a whole number of bytes.
	if excess := len(hash)*8 - orderBits; excess > 0 {
		ret.Rsh(ret, uint(excess))
	}

	return ret
}

// rfc6979Nonce derives the deterministic nonce k for the given private key and
// message hash, per RFC 6979 Section 3.2 using HMAC-SHA256. attempt selects successive
// candidates when an earlier k was rejected (the spec's K/V update loop).
func rfc6979Nonce(priv *big.Int, hash []byte, attempt int) *big.Int {
	holen := sha256.Size
	rolen := (orderN.BitLen() + 7) / 8

	bx := append(int2octets(priv, rolen), bits2octets(hash, rolen)...)

	// Step b/c: V = 0x01..., K = 0x00...
	v := make([]byte, holen)
	for i := range v {
		v[i] = 0x01
	}

	k := make([]byte, holen)

	// Step d: K = HMAC(K, V || 0x00 || int2octets(x) || bits2octets(h))
	k = mac(k, concat(v, []byte{0x00}, bx))
	// Step e: V = HMAC(K, V)
	v = mac(k, v)
	// Step f: K = HMAC(K, V || 0x01 || int2octets(x) || bits2octets(h))
	k = mac(k, concat(v, []byte{0x01}, bx))
	// Step g: V = HMAC(K, V)
	v = mac(k, v)

	// Step h: generate candidates; skip 'attempt' valid-range candidates so the
	// caller can request the next one after rejecting r==0 etc.
	skip := attempt

	for {
		var t []byte
		for len(t) < rolen {
			v = mac(k, v)
			t = append(t, v...)
		}

		candidate := bits2int(t, orderN.BitLen())
		if candidate.Sign() > 0 && candidate.Cmp(orderN) < 0 {
			if skip == 0 {
				return candidate
			}

			skip--
		}

		// K = HMAC(K, V || 0x00); V = HMAC(K, V)
		k = mac(k, append(v, 0x00))
		v = mac(k, v)
	}
}

func mac(key, msg []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(msg)

	return h.Sum(nil)
}

func concat(parts ...[]byte) []byte {
	var out []byte
	for _, p := range parts {
		out = append(out, p...)
	}

	return out
}

// bits2int (RFC 6979 Section 2.3.2): interpret the leftmost qlen bits of in as an int.
func bits2int(in []byte, qlen int) *big.Int {
	v := new(big.Int).SetBytes(in)
	if excess := len(in)*8 - qlen; excess > 0 {
		v.Rsh(v, uint(excess))
	}

	return v
}

// int2octets (RFC 6979 Section 2.3.3): fixed-length big-endian encoding of v reduced
// to rolen bytes.
func int2octets(v *big.Int, rolen int) []byte {
	out := v.Bytes()
	switch {
	case len(out) < rolen:
		padded := make([]byte, rolen)
		copy(padded[rolen-len(out):], out)

		return padded
	case len(out) > rolen:
		return out[len(out)-rolen:]
	default:
		return out
	}
}

// bits2octets (RFC 6979 Section 2.3.4): bits2int of the hash reduced mod N, then
// int2octets.
func bits2octets(in []byte, rolen int) []byte {
	z1 := bits2int(in, orderN.BitLen())
	z2 := new(big.Int).Sub(z1, orderN)
	if z2.Sign() < 0 {
		return int2octets(z1, rolen)
	}

	return int2octets(z2, rolen)
}
