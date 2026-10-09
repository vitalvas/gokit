package secp256k1

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestScalarCanonicalDecoding(t *testing.T) {
	s := newScalarValue()
	require.True(t, s.isZero())
	require.NoError(t, s.decode(fixedBytes(big.NewInt(1))))
	for _, raw := range [][]byte{nil, make([]byte, 31), make([]byte, 33), fixedBytes(orderN), bytes.Repeat([]byte{0xff}, 32)} {
		require.Error(t, s.decode(raw))
		require.Equal(t, fixedBytes(big.NewInt(1)), s.encode(), "rejected input must preserve the receiver")
	}
	for _, v := range []*big.Int{big.NewInt(0), big.NewInt(1), new(big.Int).Sub(orderN, big.NewInt(1))} {
		require.NoError(t, s.decode(fixedBytes(v)))
		require.Equal(t, fixedBytes(v), s.encode())
		require.Equal(t, v.Sign() == 0, s.isZero())
	}
	for _, raw := range [][]byte{fixedBytes(orderN), fixedBytes(new(big.Int).Add(orderN, big.NewInt(1))), bytes.Repeat([]byte{0xff}, 32)} {
		require.NoError(t, s.decodeReduced(raw))
		want := new(big.Int).Mod(new(big.Int).SetBytes(raw), orderN)
		require.Equal(t, fixedBytes(want), s.encode())
	}
	for _, raw := range [][]byte{nil, make([]byte, 31), make([]byte, 33)} {
		before := s.encode()
		require.Error(t, s.decodeReduced(raw))
		require.Equal(t, before, s.encode())
	}
}

func TestScalarAliasingAndSelection(t *testing.T) {
	a, b := newScalarValue(), newScalarValue()
	require.NoError(t, a.decode(fixedBytes(big.NewInt(3))))
	require.NoError(t, b.decode(fixedBytes(big.NewInt(5))))
	require.Equal(t, uint64(1), a.lessOrEqual(b))
	require.Equal(t, uint64(0), b.lessOrEqual(a))
	require.Equal(t, uint64(1), a.lessOrEqual(a))
	copyA := a.copy()
	copyA.add(copyA)
	require.Equal(t, fixedBytes(big.NewInt(6)), copyA.encode())
	require.Equal(t, fixedBytes(big.NewInt(3)), a.encode())
	copyA.multiply(copyA)
	require.Equal(t, fixedBytes(big.NewInt(36)), copyA.encode())
	copyA.subtract(copyA)
	require.True(t, copyA.isZero())
	require.True(t, copyA.invert().isZero())
	require.Equal(t, fixedBytes(big.NewInt(1)), a.copy().invert().multiply(a).encode())
	encoded := a.encode()
	encoded[31] = 99
	require.Equal(t, fixedBytes(big.NewInt(3)), a.encode())

	for _, choice := range []uint64{0, 1} {
		u, v := a.copy(), b.copy()
		require.NoError(t, u.selectValue(choice, u, v))
		want := a.encode()
		if choice == 1 {
			want = b.encode()
		}
		require.Equal(t, want, u.encode())
		u, v = a.copy(), b.copy()
		require.NoError(t, v.selectValue(choice, u, v))
		require.Equal(t, want, v.encode())
	}
	require.Error(t, a.selectValue(2, a, b))
	require.Error(t, a.selectValue(0, nil, b))
	require.Error(t, a.selectValue(1, b, nil))
	require.Equal(t, fixedBytes(big.NewInt(3)), a.encode())
}
