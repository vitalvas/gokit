package secp256k1

import (
	"crypto/sha256"
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompletePointAddition(t *testing.T) {
	g := newSecretPoint().base()
	identity := newSecretPoint()
	require.True(t, identity.isIdentity())
	for _, p := range []point{addPoints(g.p, identity.p), addPoints(identity.p, g.p)} {
		require.Equal(t, g.encodeUncompressed(), (&secretPoint{p: p}).encodeUncompressed())
	}
	require.True(t, (&secretPoint{p: addPoints(identity.p, identity.p)}).isIdentity())
	negative := point{x: g.p.x, y: fieldSub(uint256{}, g.p.y), z: g.p.z}
	require.True(t, (&secretPoint{p: addPoints(g.p, negative)}).isIdentity())
	x, y := scalarBaseMult(big.NewInt(2))
	doubled := &secretPoint{p: addPoints(g.p, g.p)}
	require.Equal(t, (&PublicKey{X: x, Y: y}).SerializeUncompressed(), doubled.encodeUncompressed())
	scale := fieldModulus.toMontgomery(uint256{17})
	rescaled := point{x: fieldMul(g.p.x, scale), y: fieldMul(g.p.y, scale), z: scale}
	require.Equal(t, doubled.encodeUncompressed(), (&secretPoint{p: addPoints(g.p, rescaled)}).encodeUncompressed())
	a, b := g.p, identity.p
	swapPoints(&a, &b, 0)
	require.Equal(t, g.p, a)
	require.Equal(t, identity.p, b)
	swapPoints(&a, &b, 1)
	require.Equal(t, identity.p, a)
	require.Equal(t, g.p, b)
}

func TestSecretPointMultiplicationAgainstReference(t *testing.T) {
	peerX, peerY := scalarBaseMult(big.NewInt(7))
	peerBytes := (&PublicKey{X: peerX, Y: peerY}).SerializeUncompressed()
	values := make([]*big.Int, 0, 20)
	values = append(values, big.NewInt(0), big.NewInt(1), big.NewInt(2), new(big.Int).Sub(orderN, big.NewInt(1)))
	for i := range 16 {
		raw := sha256.Sum256(fmt.Appendf(nil, "point multiplication/%d", i))
		values = append(values, new(big.Int).Mod(new(big.Int).SetBytes(raw[:]), orderN))
	}
	for _, v := range values {
		s := newScalarValue()
		require.NoError(t, s.decode(fixedBytes(v)))
		for _, raw := range [][]byte{newSecretPoint().base().encodeUncompressed(), peerBytes} {
			p := newSecretPoint()
			require.NoError(t, p.decodeUncompressed(raw))
			ref := scalarMult(affineToJacobian(new(big.Int).SetBytes(raw[1:33]), new(big.Int).SetBytes(raw[33:])), v)
			p.multiply(s)
			require.Equal(t, v.Sign() == 0, p.isIdentity())
			x, y := ref.toAffine()
			require.Equal(t, (&PublicKey{X: x, Y: y}).SerializeUncompressed(), p.encodeUncompressed())
			require.Equal(t, fixedBytes(x), p.xCoordinate())
		}
		require.True(t, newSecretPoint().multiply(s).isIdentity())
	}
	require.Panics(t, func() { newSecretPoint().multiply(nil) })
}

func TestSecretPointDecodeRejectsInvalidInput(t *testing.T) {
	p := newSecretPoint().base()
	valid := p.encodeUncompressed()
	xOutOfRange, yOutOfRange := append([]byte(nil), valid...), append([]byte(nil), valid...)
	copy(xOutOfRange[1:33], fixedBytes(primeP))
	copy(yOutOfRange[33:], fixedBytes(primeP))
	wrongPrefix := append([]byte(nil), valid...)
	wrongPrefix[0] = 2
	for _, raw := range [][]byte{nil, valid[:64], append(append([]byte(nil), valid...), 0), wrongPrefix, xOutOfRange, yOutOfRange, newSecretPoint().encodeUncompressed()} {
		require.Error(t, p.decodeUncompressed(raw))
		require.Equal(t, valid, p.encodeUncompressed(), "invalid input must preserve the receiver")
	}
	require.NoError(t, p.decodeUncompressed(valid))
	require.Equal(t, valid, p.encodeUncompressed())
}
