package secp256k1

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

func checkModArithmetic(t testing.TB, m *modulus, a, b *big.Int) {
	t.Helper()
	n := new(big.Int).SetBytes(encodeWords(m.n))
	r := new(big.Int).Lsh(big.NewInt(1), 256)
	rInverse := new(big.Int).ModInverse(r, n)
	aw, bw := decodeWords(fixedBytes(a)), decodeWords(fixedBytes(b))
	am, bm := m.toMontgomery(aw), m.toMontgomery(bw)
	want := new(big.Int).Mod(new(big.Int).Mul(a, r), n)
	require.Equal(t, fixedBytes(want), encodeWords(am), "Montgomery conversion")
	require.Equal(t, fixedBytes(a), encodeWords(m.fromMontgomery(am)), "round trip")
	want.Mod(new(big.Int).Mul(new(big.Int).Mul(a, b), rInverse), n)
	require.Equal(t, fixedBytes(want), encodeWords(montgomeryMultiply(aw, bw, m)), "raw Montgomery product")

	for _, operation := range []struct {
		name string
		got  uint256
		want *big.Int
	}{
		{name: "add", got: addMod(am, bm, m), want: new(big.Int).Add(a, b)},
		{name: "subtract", got: subtractMod(am, bm, m), want: new(big.Int).Sub(a, b)},
		{name: "multiply", got: montgomeryMultiply(am, bm, m), want: new(big.Int).Mul(a, b)},
	} {
		operation.want.Mod(operation.want, n)
		require.Equal(t, fixedBytes(operation.want), encodeWords(m.fromMontgomery(operation.got)), operation.name)
		_, borrow := subtractWords(operation.got, m.n)
		require.Equal(t, uint64(1), borrow, "%s must produce a reduced value", operation.name)
	}
}

func TestMontgomeryArithmeticAgainstBigInt(t *testing.T) {
	for _, m := range []*modulus{fieldModulus, scalarModulus} {
		n := new(big.Int).SetBytes(encodeWords(m.n))
		require.Equal(t, uint64(0xffffffffffffffff), m.n[0]*m.n0)
		values := []*big.Int{
			big.NewInt(0), big.NewInt(1), big.NewInt(2),
			new(big.Int).Sub(n, big.NewInt(1)), new(big.Int).Sub(n, big.NewInt(2)),
			new(big.Int).Lsh(big.NewInt(1), 255),
			new(big.Int).SetBytes(bytes.Repeat([]byte{0xaa}, 32)),
			new(big.Int).SetBytes(bytes.Repeat([]byte{0x55}, 32)),
		}
		for _, a := range values {
			for _, b := range values {
				checkModArithmetic(t, m, a, b)
			}
			inverse := m.inverse(m.toMontgomery(decodeWords(fixedBytes(a))))
			want := new(big.Int).ModInverse(a, n)
			if want == nil {
				want = new(big.Int)
			}
			require.Equal(t, fixedBytes(want), encodeWords(m.fromMontgomery(inverse)))
		}
		for i := range 256 {
			aBytes := sha256.Sum256(fmt.Appendf(nil, "arithmetic/a/%d", i))
			bBytes := sha256.Sum256(fmt.Appendf(nil, "arithmetic/b/%d", i))
			a := new(big.Int).Mod(new(big.Int).SetBytes(aBytes[:]), n)
			b := new(big.Int).Mod(new(big.Int).SetBytes(bBytes[:]), n)
			checkModArithmetic(t, m, a, b)
		}
	}
}

func TestFixedWidthSelectionAndCarry(t *testing.T) {
	zero := uint256{}
	allOnes := uint256{^uint64(0), ^uint64(0), ^uint64(0), ^uint64(0)}
	got, borrow := subtractWords(zero, uint256{1})
	require.Equal(t, uint64(1), borrow)
	require.Equal(t, allOnes, got)
	require.Equal(t, zero, selectWords(zero, allOnes, 0))
	require.Equal(t, allOnes, selectWords(zero, allOnes, 1))
	require.Equal(t, uint64(1), zeroBit(zero))
	for i := range 4 {
		for bit := range 64 {
			var one uint256
			one[i] = uint64(1) << bit
			require.Equal(t, uint64(0), zeroBit(one))
			require.Equal(t, one, decodeWords(encodeWords(one)))
		}
	}
	low, high := multiplyAdd(^uint64(0), ^uint64(0), ^uint64(0), ^uint64(0))
	require.Equal(t, ^uint64(0), low)
	require.Equal(t, ^uint64(0), high)
}

func FuzzMontgomeryArithmetic(f *testing.F) {
	f.Add(make([]byte, 64))
	f.Add(bytes.Repeat([]byte{0xff}, 64))
	f.Add(append(fixedBytes(new(big.Int).Sub(primeP, big.NewInt(1))), fixedBytes(new(big.Int).Sub(orderN, big.NewInt(1)))...))
	f.Fuzz(func(t *testing.T, raw []byte) {
		if len(raw) != 64 {
			return
		}
		for _, m := range []*modulus{fieldModulus, scalarModulus} {
			n := new(big.Int).SetBytes(encodeWords(m.n))
			a := new(big.Int).Mod(new(big.Int).SetBytes(raw[:32]), n)
			b := new(big.Int).Mod(new(big.Int).SetBytes(raw[32:]), n)
			checkModArithmetic(t, m, a, b)
		}
	})
}
