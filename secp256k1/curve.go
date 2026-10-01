package secp256k1

import (
	"fmt"
	"math/big"
)

// Curve parameters for secp256k1 (SEC 2, y^2 = x^3 + 7 over F_p).
var (
	primeP = mustHex("FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEFFFFFC2F")
	orderN = mustHex("FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364141")
	coeffB = big.NewInt(7)
	gX     = mustHex("79BE667EF9DCBBAC55A06295CE870B07029BFCDB2DCE28D959F2815B16F81798")
	gY     = mustHex("483ADA7726A3C4655DA4FBFC0E1108A8FD17B448A68554199C47D08FFB10D4B8")
)

// coordinateLen is the byte length of an secp256k1 field element / coordinate.
const coordinateLen = 32

func mustHex(s string) *big.Int {
	n, ok := new(big.Int).SetString(s, 16)
	if !ok {
		panic(fmt.Sprintf("secp256k1: invalid constant %s", s))
	}

	return n
}

// P returns the field prime.
func P() *big.Int { return new(big.Int).Set(primeP) }

// N returns the group order.
func N() *big.Int { return new(big.Int).Set(orderN) }

// jacobianPoint is a point in Jacobian coordinates (X, Y, Z) representing the
// affine point (X/Z^2, Y/Z^3). The identity is any point with Z == 0.
type jacobianPoint struct {
	x, y, z *big.Int
}

func newJacobianIdentity() *jacobianPoint {
	return &jacobianPoint{x: big.NewInt(1), y: big.NewInt(1), z: big.NewInt(0)}
}

func (p *jacobianPoint) isIdentity() bool {
	return p.z.Sign() == 0
}

// set copies q into p.
func (p *jacobianPoint) set(q *jacobianPoint) {
	p.x.Set(q.x)
	p.y.Set(q.y)
	p.z.Set(q.z)
}

// setIdentity resets p to the point at infinity (Z = 0).
func (p *jacobianPoint) setIdentity() {
	p.x.SetInt64(1)
	p.y.SetInt64(1)
	p.z.SetInt64(0)
}

// affineToJacobian lifts an affine (x, y) into Jacobian coordinates (Z = 1).
func affineToJacobian(x, y *big.Int) *jacobianPoint {
	return &jacobianPoint{
		x: new(big.Int).Set(x),
		y: new(big.Int).Set(y),
		z: big.NewInt(1),
	}
}

// toAffine converts back to affine coordinates, returning the curve identity as
// (0, 0).
func (p *jacobianPoint) toAffine() (*big.Int, *big.Int) {
	if p.isIdentity() {
		return big.NewInt(0), big.NewInt(0)
	}

	zInv := new(big.Int).ModInverse(p.z, primeP)
	zInv2 := new(big.Int).Mul(zInv, zInv)
	zInv2.Mod(zInv2, primeP)
	zInv3 := new(big.Int).Mul(zInv2, zInv)
	zInv3.Mod(zInv3, primeP)

	x := new(big.Int).Mul(p.x, zInv2)
	x.Mod(x, primeP)
	y := new(big.Int).Mul(p.y, zInv3)
	y.Mod(y, primeP)

	return x, y
}

// scratch holds reusable big.Int temporaries for the point arithmetic so that
// scalarMult does not allocate fresh intermediates on every doubling/addition.
// The field operations below reduce mod p in place into these scratch values; a
// single scratch is threaded through one scalarMult call and never shared across
// goroutines.
type scratch struct {
	a, b, c, d, e, f, t0, t1 *big.Int
}

func newScratch() *scratch {
	return &scratch{
		a:  new(big.Int),
		b:  new(big.Int),
		c:  new(big.Int),
		d:  new(big.Int),
		e:  new(big.Int),
		f:  new(big.Int),
		t0: new(big.Int),
		t1: new(big.Int),
	}
}

// doubleInto sets out = 2*p using the standard Jacobian doubling for a = 0
// curves, using sc for intermediates. out must not alias p's fields.
func (sc *scratch) doubleInto(out, p *jacobianPoint) {
	if p.isIdentity() || p.y.Sign() == 0 {
		out.setIdentity()

		return
	}

	a, b, c, d, e, f := sc.a, sc.b, sc.c, sc.d, sc.e, sc.f

	// A = X^2, B = Y^2, C = B^2
	modMul(a, p.x, p.x)
	modMul(b, p.y, p.y)
	modMul(c, b, b)

	// D = 2*((X+B)^2 - A - C)
	d.Add(p.x, b)
	modMul(d, d, d)
	d.Sub(d, a)
	d.Sub(d, c)
	d.Lsh(d, 1)
	mod(d)

	// E = 3*A, F = E^2  (a = 0, so no a*Z^4 term)
	e.Lsh(a, 1)
	e.Add(e, a)
	mod(e)
	modMul(f, e, e)

	// Z3 = 2*Y*Z  (compute before overwriting out, in case out aliases p)
	t := sc.t0
	modMul(t, p.y, p.z)
	t.Lsh(t, 1)
	mod(t)

	// X3 = F - 2*D
	x3 := sc.t1
	x3.Lsh(d, 1)
	x3.Sub(f, x3)
	mod(x3)

	// Y3 = E*(D - X3) - 8*C
	out.y.Sub(d, x3)
	modMul(out.y, out.y, e)
	c.Lsh(c, 3)
	out.y.Sub(out.y, c)
	mod(out.y)

	out.x.Set(x3)
	out.z.Set(t)
}

// addInto sets out = p + q using Jacobian addition, using sc for intermediates.
// out must not alias p or q.
func (sc *scratch) addInto(out, p, q *jacobianPoint) {
	if p.isIdentity() {
		out.set(q)

		return
	}

	if q.isIdentity() {
		out.set(p)

		return
	}

	z1z1, z2z2, u1, u2, s1, s2 := sc.a, sc.b, sc.c, sc.d, sc.e, sc.f

	modMul(z1z1, p.z, p.z)
	modMul(z2z2, q.z, q.z)

	modMul(u1, p.x, z2z2)
	modMul(u2, q.x, z1z1)

	modMul(s1, p.y, q.z)
	modMul(s1, s1, z2z2)
	modMul(s2, q.y, p.z)
	modMul(s2, s2, z1z1)

	if u1.Cmp(u2) == 0 {
		if s1.Cmp(s2) != 0 {
			out.setIdentity()

			return
		}

		sc.doubleInto(out, p)

		return
	}

	h, i, j, r, v := sc.t0, sc.t1, new(big.Int), new(big.Int), new(big.Int)

	h.Sub(u2, u1)
	mod(h)
	i.Lsh(h, 1)
	modMul(i, i, i)
	modMul(j, h, i)
	r.Sub(s2, s1)
	r.Lsh(r, 1)
	mod(r)
	modMul(v, u1, i)

	// Z3 = ((Z1+Z2)^2 - Z1Z1 - Z2Z2) * H  (before overwriting out)
	z3 := new(big.Int)
	z3.Add(p.z, q.z)
	modMul(z3, z3, z3)
	z3.Sub(z3, z1z1)
	z3.Sub(z3, z2z2)
	modMul(z3, z3, h)

	// X3 = R^2 - J - 2*V
	x3 := new(big.Int)
	modMul(x3, r, r)
	x3.Sub(x3, j)
	v2 := new(big.Int).Lsh(v, 1)
	x3.Sub(x3, v2)
	mod(x3)

	// Y3 = R*(V - X3) - 2*S1*J
	out.y.Sub(v, x3)
	modMul(out.y, out.y, r)
	modMul(s1, s1, j)
	s1.Lsh(s1, 1)
	out.y.Sub(out.y, s1)
	mod(out.y)

	out.x.Set(x3)
	out.z.Set(z3)
}

// scalarMult returns k*p via double-and-add over the big-endian bits of k.
func scalarMult(p *jacobianPoint, k *big.Int) *jacobianPoint {
	result := newJacobianIdentity()
	if k.Sign() == 0 {
		return result
	}

	sc := newScratch()
	addend := &jacobianPoint{x: new(big.Int).Set(p.x), y: new(big.Int).Set(p.y), z: new(big.Int).Set(p.z)}
	tmp := newJacobianIdentity()

	for i := 0; i < k.BitLen(); i++ {
		if k.Bit(i) == 1 {
			sc.addInto(tmp, result, addend)
			result, tmp = tmp, result
		}

		sc.doubleInto(tmp, addend)
		addend, tmp = tmp, addend
	}

	return result
}

// add returns p + q as a fresh point. Used for the single point addition at the
// end of verification/recovery (not in a hot loop), so it allocates rather than
// sharing scalarMult's scratch reuse.
func (p *jacobianPoint) add(q *jacobianPoint) *jacobianPoint {
	out := newJacobianIdentity()
	newScratch().addInto(out, p, q)

	return out
}

// double returns 2*p as a fresh point (non-hot path; see add).
func (p *jacobianPoint) double() *jacobianPoint {
	out := newJacobianIdentity()
	newScratch().doubleInto(out, p)

	return out
}

// scalarBaseMult returns k*G.
func scalarBaseMult(k *big.Int) (*big.Int, *big.Int) {
	g := affineToJacobian(gX, gY)

	return scalarMult(g, k).toAffine()
}

// mod reduces v modulo the field prime, keeping the result non-negative.
func mod(v *big.Int) *big.Int {
	v.Mod(v, primeP)

	return v
}

// modMul sets out = (x * y) mod p. out may alias x or y.
func modMul(out, x, y *big.Int) {
	out.Mul(x, y)
	out.Mod(out, primeP)
}

// isOnCurve reports whether the affine point (x, y) satisfies the curve
// equation and lies within the field.
func isOnCurve(x, y *big.Int) bool {
	if x.Sign() < 0 || x.Cmp(primeP) >= 0 || y.Sign() < 0 || y.Cmp(primeP) >= 0 {
		return false
	}

	// y^2 mod p
	y2 := new(big.Int).Mul(y, y)
	y2.Mod(y2, primeP)

	// x^3 + 7 mod p
	x3 := new(big.Int).Mul(x, x)
	x3.Mul(x3, x)
	x3.Add(x3, coeffB)
	x3.Mod(x3, primeP)

	return y2.Cmp(x3) == 0
}

// decompressY recovers the Y coordinate for a given X and the parity requested
// by yOdd (the compressed-point prefix: 0x02 -> even, 0x03 -> odd). It returns
// an error if x is out of range or x^3+7 has no square root (x is not a valid
// curve point's abscissa).
//
// Because secp256k1's field prime satisfies p = 3 (mod 4), the square root of a
// quadratic residue a is a^((p+1)/4) mod p; squaring the result confirms a was
// a residue in the first place.
func decompressY(x *big.Int, yOdd bool) (*big.Int, error) {
	if x.Sign() < 0 || x.Cmp(primeP) >= 0 {
		return nil, errBadKey("x coordinate out of field range")
	}

	// alpha = x^3 + 7 mod p
	alpha := new(big.Int).Mul(x, x)
	alpha.Mul(alpha, x)
	alpha.Add(alpha, coeffB)
	alpha.Mod(alpha, primeP)

	// y = alpha^((p+1)/4) mod p
	exp := new(big.Int).Add(primeP, big.NewInt(1))
	exp.Rsh(exp, 2)
	y := new(big.Int).Exp(alpha, exp, primeP)

	// Confirm y^2 == alpha; otherwise alpha was a non-residue and x is not on
	// the curve.
	check := new(big.Int).Mul(y, y)
	check.Mod(check, primeP)
	if check.Cmp(alpha) != 0 {
		return nil, errBadKey("compressed point is not on the curve")
	}

	// Pick the root with the requested parity (y and p-y have opposite parity
	// since p is odd).
	if y.Bit(0) == 1 != yOdd {
		y.Sub(primeP, y)
	}

	return y, nil
}

// fixedBytes returns the 32-byte big-endian encoding of v, left-padded.
func fixedBytes(v *big.Int) []byte {
	out := make([]byte, coordinateLen)
	v.FillBytes(out)

	return out
}

func errBadKey(format string, args ...any) error {
	return fmt.Errorf("secp256k1: %s", fmt.Sprintf(format, args...))
}
