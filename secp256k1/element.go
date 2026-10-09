package secp256k1

import "errors"

var errInvalidPoint = errors.New("secp256k1: invalid point")

// point uses homogeneous coordinates: affine (X/Z, Y/Z), with infinity (0,1,0).
type point struct {
	x, y, z uint256
}

type secretPoint struct {
	p point
}

func newSecretPoint() *secretPoint {
	return &secretPoint{p: identityPoint()}
}

func identityPoint() point {
	return point{y: fieldModulus.one}
}

func (e *secretPoint) base() *secretPoint {
	e.p = point{
		x: fieldModulus.toMontgomery(uint256{
			0x59f2815b16f81798, 0x029bfcdb2dce28d9, 0x55a06295ce870b07, 0x79be667ef9dcbbac,
		}),
		y: fieldModulus.toMontgomery(uint256{
			0x9c47d08ffb10d4b8, 0xfd17b448a6855419, 0x5da4fbfc0e1108a8, 0x483ada7726a3c465,
		}),
		z: fieldModulus.one,
	}
	return e
}

func fieldAdd(a, b uint256) uint256 { return addMod(a, b, fieldModulus) }
func fieldSub(a, b uint256) uint256 { return subtractMod(a, b, fieldModulus) }
func fieldMul(a, b uint256) uint256 { return montgomeryMultiply(a, b, fieldModulus) }

// addPoints implements the complete a=0 addition formulas (algorithm 7) of
// Renes, Costello and Batina, https://eprint.iacr.org/2015/1060.
// Here b=7, so the formula's constant 3b is 21.
func addPoints(p, q point) point {
	xx := fieldMul(p.x, q.x)
	yy := fieldMul(p.y, q.y)
	zz := fieldMul(p.z, q.z)
	xy := fieldSub(fieldMul(fieldAdd(p.x, p.y), fieldAdd(q.x, q.y)), fieldAdd(xx, yy))
	yz := fieldSub(fieldMul(fieldAdd(p.y, p.z), fieldAdd(q.y, q.z)), fieldAdd(yy, zz))
	xz := fieldSub(fieldMul(fieldAdd(p.x, p.z), fieldAdd(q.x, q.z)), fieldAdd(xx, zz))
	b3 := fieldModulus.toMontgomery(uint256{21})
	threeXX := fieldAdd(fieldAdd(xx, xx), xx)
	b3ZZ := fieldMul(b3, zz)
	yyMinus := fieldSub(yy, b3ZZ)
	yyPlus := fieldAdd(yy, b3ZZ)
	b3XZ := fieldMul(b3, xz)
	return point{
		x: fieldSub(fieldMul(xy, yyMinus), fieldMul(yz, b3XZ)),
		y: fieldAdd(fieldMul(yyMinus, yyPlus), fieldMul(threeXX, b3XZ)),
		z: fieldAdd(fieldMul(yz, yyPlus), fieldMul(threeXX, xy)),
	}
}

func swapPoints(a, b *point, choice uint64) {
	mask := uint64(0) - choice
	for i := range a.x {
		x := (a.x[i] ^ b.x[i]) & mask
		y := (a.y[i] ^ b.y[i]) & mask
		z := (a.z[i] ^ b.z[i]) & mask
		a.x[i] ^= x
		b.x[i] ^= x
		a.y[i] ^= y
		b.y[i] ^= y
		a.z[i] ^= z
		b.z[i] ^= z
	}
}

// multiply uses a fixed 256-round ladder with masked swaps.
func (e *secretPoint) multiply(s *scalarValue) *secretPoint {
	if s == nil {
		panic(errInvalidScalar)
	}
	k := scalarModulus.fromMontgomery(s.v)
	low, high := identityPoint(), e.p
	for i := 255; i >= 0; i-- {
		bit := (k[i/64] >> uint(i%64)) & 1
		swapPoints(&low, &high, bit)
		high = addPoints(low, high)
		low = addPoints(low, low)
		swapPoints(&low, &high, bit)
	}
	e.p = low
	return e
}

func (e *secretPoint) isIdentity() bool { return zeroBit(e.p.z) == 1 }

func (e *secretPoint) affine() (x, y uint256) {
	inverse := fieldModulus.inverse(e.p.z)
	return fieldModulus.fromMontgomery(fieldMul(e.p.x, inverse)), fieldModulus.fromMontgomery(fieldMul(e.p.y, inverse))
}

// Infinity is encoded with zero coordinates.
func (e *secretPoint) encodeUncompressed() []byte {
	x, y := e.affine()
	out := make([]byte, 65)
	out[0] = 4
	copy(out[1:33], encodeWords(x))
	copy(out[33:], encodeWords(y))
	return out
}

func (e *secretPoint) xCoordinate() []byte {
	x, _ := e.affine()
	return encodeWords(x)
}

func (e *secretPoint) decodeUncompressed(raw []byte) error {
	if len(raw) != 65 || raw[0] != 4 {
		return errInvalidPoint
	}
	x, y := decodeWords(raw[1:33]), decodeWords(raw[33:])
	_, xBorrow := subtractWords(x, fieldModulus.n)
	_, yBorrow := subtractWords(y, fieldModulus.n)
	if xBorrow == 0 || yBorrow == 0 {
		return errInvalidPoint
	}
	x, y = fieldModulus.toMontgomery(x), fieldModulus.toMontgomery(y)
	expected := fieldAdd(fieldMul(fieldMul(x, x), x), fieldModulus.toMontgomery(uint256{7}))
	actual := fieldMul(y, y)
	if zeroBit(fieldSub(actual, expected)) != 1 {
		return errInvalidPoint
	}
	e.p = point{x: x, y: y, z: fieldModulus.one}
	return nil
}
