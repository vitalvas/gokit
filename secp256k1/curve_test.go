package secp256k1

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustDecodeHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	require.NoError(t, err)

	return b
}

func TestCurveBasics(t *testing.T) {
	t.Run("generator is on curve", func(t *testing.T) {
		assert.True(t, isOnCurve(gX, gY))
	})

	t.Run("identity scalar mult", func(t *testing.T) {
		x, y := scalarBaseMult(big.NewInt(0))
		assert.Equal(t, 0, x.Sign())
		assert.Equal(t, 0, y.Sign())
	})

	t.Run("1*G equals G", func(t *testing.T) {
		x, y := scalarBaseMult(big.NewInt(1))
		assert.Equal(t, gX, x)
		assert.Equal(t, gY, y)
	})

	t.Run("2*G is on curve", func(t *testing.T) {
		x, y := scalarBaseMult(big.NewInt(2))
		assert.True(t, isOnCurve(x, y))
	})

	t.Run("off-curve point rejected", func(t *testing.T) {
		assert.False(t, isOnCurve(big.NewInt(1), big.NewInt(1)))
	})

	t.Run("exported params", func(t *testing.T) {
		assert.Equal(t, 0, P().Cmp(primeP))
		assert.Equal(t, 0, N().Cmp(orderN))
	})
}

func TestCurveLaw(t *testing.T) {
	g := affineToJacobian(gX, gY)

	t.Run("P + identity = P", func(t *testing.T) {
		sum := g.add(newJacobianIdentity())
		sx, sy := sum.toAffine()
		assert.Equal(t, 0, sx.Cmp(gX))
		assert.Equal(t, 0, sy.Cmp(gY))
	})

	t.Run("identity + P = P", func(t *testing.T) {
		sum := newJacobianIdentity().add(g)
		sx, sy := sum.toAffine()
		assert.Equal(t, 0, sx.Cmp(gX))
		assert.Equal(t, 0, sy.Cmp(gY))
	})

	t.Run("P + P = 2P (add falls through to double)", func(t *testing.T) {
		viaAdd := g.add(g)
		viaDouble := g.double()
		ax, ay := viaAdd.toAffine()
		dx, dy := viaDouble.toAffine()
		assert.Equal(t, 0, ax.Cmp(dx))
		assert.Equal(t, 0, ay.Cmp(dy))
	})

	t.Run("P + (-P) = identity", func(t *testing.T) {
		negY := new(big.Int).Sub(primeP, gY)
		negG := affineToJacobian(gX, negY)
		sum := g.add(negG)
		assert.True(t, sum.isIdentity())
	})

	t.Run("double of y=0 point is identity", func(t *testing.T) {
		// A point with Y == 0 doubles to the identity; construct one directly.
		p := &jacobianPoint{x: big.NewInt(5), y: big.NewInt(0), z: big.NewInt(1)}
		assert.True(t, p.double().isIdentity())
	})
}

func TestMustHexPanicsOnBadConstant(t *testing.T) {
	assert.Panics(t, func() {
		mustHex("not-hex-zz")
	})
}

func TestIsOnCurveBounds(t *testing.T) {
	t.Run("negative coordinate rejected", func(t *testing.T) {
		assert.False(t, isOnCurve(big.NewInt(-1), gY))
	})

	t.Run("coordinate >= p rejected", func(t *testing.T) {
		assert.False(t, isOnCurve(new(big.Int).Set(primeP), gY))
	})
}

func BenchmarkScalarBaseMult(b *testing.B) {
	k := mustHex("c9afa9d845ba75166b5c215767b1d6934e50c3db36e89b127b8a622b120f6721")

	b.ResetTimer()
	for b.Loop() {
		scalarBaseMult(k)
	}
}
