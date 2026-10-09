package secp256k1

import (
	"encoding/binary"
	"math/bits"
)

// uint256 stores four little-endian 64-bit limbs.
type uint256 [4]uint64

// modulus holds public constants for Montgomery arithmetic with R = 2^256.
// Both moduli are prime, odd, and greater than 2^255.
type modulus struct {
	n        uint256
	n0       uint64  // -n[0]^-1 modulo 2^64
	rr       uint256 // R^2 modulo n, in ordinary representation
	one      uint256 // R modulo n (Montgomery representation of 1)
	exponent uint256 // n-2 for fixed-exponent inversion
}

var (
	fieldModulus = newModulus(uint256{
		0xfffffffefffffc2f, 0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff,
	})
	scalarModulus = newModulus(uint256{
		0xbfd25e8cd0364141, 0xbaaedce6af48a03b, 0xfffffffffffffffe, 0xffffffffffffffff,
	})
)

func newModulus(n uint256) *modulus {
	inverse := uint64(1)
	for range 6 {
		inverse *= 2 - n[0]*inverse
	}
	m := &modulus{
		n:  n,
		n0: 0 - inverse,
		rr: uint256{1},
	}
	for range 512 {
		m.rr = addMod(m.rr, m.rr, m)
	}
	m.one = m.toMontgomery(uint256{1})
	m.exponent, _ = subtractWords(n, uint256{2})
	return m
}

func selectWords(a, b uint256, choice uint64) uint256 {
	mask := uint64(0) - choice // choice is always 0 or 1
	var out uint256
	for i := range out {
		out[i] = (a[i] &^ mask) | (b[i] & mask)
	}
	return out
}

func subtractWords(a, b uint256) (out uint256, borrow uint64) {
	for i := range out {
		out[i], borrow = bits.Sub64(a[i], b[i], borrow)
	}
	return out, borrow
}

func addMod(a, b uint256, m *modulus) uint256 {
	var sum uint256
	var carry uint64
	for i := range sum {
		sum[i], carry = bits.Add64(a[i], b[i], carry)
	}
	reduced, borrow := subtractWords(sum, m.n)
	// Retain the 257th bit: an overflowing sum always needs reduction.
	return selectWords(sum, reduced, carry|(borrow^1))
}

func subtractMod(a, b uint256, m *modulus) uint256 {
	difference, borrow := subtractWords(a, b)
	var corrected uint256
	var carry uint64
	for i := range corrected {
		corrected[i], carry = bits.Add64(difference[i], m.n[i], carry)
	}
	return selectWords(difference, corrected, borrow)
}

// multiplyAdd computes x*y + z + carry as a 128-bit result. Its maximum is
// (2^64-1)^2 + 2*(2^64-1) = 2^128-1, so the high word cannot overflow.
func multiplyAdd(x, y, z, carry uint64) (low, high uint64) {
	high, low = bits.Mul64(x, y)
	var overflow uint64
	low, overflow = bits.Add64(low, z, 0)
	high, _ = bits.Add64(high, 0, overflow)
	low, overflow = bits.Add64(low, carry, 0)
	high, _ = bits.Add64(high, 0, overflow)
	return low, high
}

// accumulate adds x*y*2^(64*offset) to the nine-limb accumulator. Carry
// propagation always visits the remaining limbs, even when carry is zero.
func accumulate(t *[9]uint64, offset int, x uint64, y uint256) {
	var carry uint64
	for j := range y {
		t[offset+j], carry = multiplyAdd(x, y[j], t[offset+j], carry)
	}
	t[offset+4], carry = bits.Add64(t[offset+4], carry, 0)
	for j := offset + 5; j < len(t); j++ {
		t[j], carry = bits.Add64(t[j], 0, carry)
	}
}

// montgomeryMultiply returns a*b/R modulo n. The ninth limb preserves overflow
// during REDC. For reduced inputs, the result before final subtraction is <2n.
func montgomeryMultiply(a, b uint256, m *modulus) uint256 {
	var t [9]uint64
	for i := range a {
		accumulate(&t, i, a[i], b)
	}
	for i := range a {
		accumulate(&t, i, t[i]*m.n0, m.n)
	}
	result := uint256{t[4], t[5], t[6], t[7]}
	reduced, borrow := subtractWords(result, m.n)
	return selectWords(result, reduced, t[8]|(borrow^1))
}

func (m *modulus) toMontgomery(v uint256) uint256 {
	return montgomeryMultiply(v, m.rr, m)
}

func (m *modulus) fromMontgomery(v uint256) uint256 {
	return montgomeryMultiply(v, uint256{1}, m)
}

// inverse computes v^(n-2) with 256 fixed rounds. The exponent is public and
// constant. Zero maps to zero, permitting complete point formulas at infinity.
func (m *modulus) inverse(v uint256) uint256 {
	out := m.one
	for i := 255; i >= 0; i-- {
		out = montgomeryMultiply(out, out, m)
		product := montgomeryMultiply(out, v, m)
		bit := (m.exponent[i/64] >> uint(i%64)) & 1
		out = selectWords(out, product, bit)
	}
	return out
}

func zeroBit(v uint256) uint64 {
	word := v[0] | v[1] | v[2] | v[3]
	return ((word | (0 - word)) >> 63) ^ 1
}

func decodeWords(raw []byte) uint256 {
	var out uint256
	for i := range out {
		out[i] = binary.BigEndian.Uint64(raw[24-8*i : 32-8*i])
	}
	return out
}

func encodeWords(v uint256) []byte {
	var out [32]byte
	for i := range v {
		binary.BigEndian.PutUint64(out[24-8*i:32-8*i], v[i])
	}
	return out[:]
}
