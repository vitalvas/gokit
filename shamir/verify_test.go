package shamir

import (
	"crypto/rand"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVerifyShare(t *testing.T) {
	secret := []byte("verify test")
	shares, err := Split(secret, 3, 5)
	require.NoError(t, err)

	t.Run("valid shares", func(t *testing.T) {
		// Verify each share against others (need more than threshold for meaningful verification)
		for i, share := range shares {
			otherShares := make([]*Share, 0, len(shares)-1)
			for j, s := range shares {
				if i != j {
					otherShares = append(otherShares, s)
				}
			}
			// Use all other shares for verification
			assert.True(t, VerifyShare(share, otherShares), "share %d should be valid", i)
		}
	})

	t.Run("fake share", func(t *testing.T) {
		// Create a fake share - needs to be verified against more than threshold shares
		// to detect it's invalid (with exactly threshold, any point is valid)
		fakeShare := &Share{
			X:         big.NewInt(100),
			Y:         big.NewInt(999999),
			Threshold: 3,
			Total:     5,
		}
		// Use 3 real shares to verify the fake one (threshold=3, so 3+1=4 total points)
		assert.False(t, VerifyShare(fakeShare, shares[:3]))
	})

	t.Run("nil share", func(t *testing.T) {
		assert.False(t, VerifyShare(nil, shares[:2]))
	})

	t.Run("insufficient other shares", func(t *testing.T) {
		assert.False(t, VerifyShare(shares[0], shares[1:2]))
	})

	t.Run("inconsistent threshold", func(t *testing.T) {
		otherShares := make([]*Share, 2)
		for i := range otherShares {
			otherShares[i] = shares[i+1].Clone()
		}
		otherShares[0].Threshold = 5
		assert.False(t, VerifyShare(shares[0], otherShares))
	})

	t.Run("duplicate x coordinates", func(t *testing.T) {
		otherShares := []*Share{shares[1], shares[1].Clone()}
		assert.False(t, VerifyShare(shares[0], otherShares))
	})
}

func TestVerifyAllShares(t *testing.T) {
	secret := []byte("verify all test")
	shares, err := Split(secret, 3, 5)
	require.NoError(t, err)

	t.Run("valid shares", func(t *testing.T) {
		assert.True(t, VerifyAllShares(shares))
	})

	t.Run("corrupted share", func(t *testing.T) {
		corruptedShares := make([]*Share, len(shares))
		for i, s := range shares {
			corruptedShares[i] = s.Clone()
		}
		corruptedShares[2].Y.Add(corruptedShares[2].Y, big.NewInt(1))
		assert.False(t, VerifyAllShares(corruptedShares))
	})

	t.Run("empty shares", func(t *testing.T) {
		assert.False(t, VerifyAllShares([]*Share{}))
	})

	t.Run("insufficient shares", func(t *testing.T) {
		assert.False(t, VerifyAllShares(shares[:2]))
	})

	t.Run("inconsistent threshold", func(t *testing.T) {
		badShares := make([]*Share, len(shares))
		for i, s := range shares {
			badShares[i] = s.Clone()
		}
		badShares[1].Threshold = 5
		assert.False(t, VerifyAllShares(badShares))
	})

	t.Run("duplicate x coordinates", func(t *testing.T) {
		badShares := make([]*Share, 3)
		badShares[0] = shares[0].Clone()
		badShares[1] = shares[0].Clone()
		badShares[2] = shares[2].Clone()
		assert.False(t, VerifyAllShares(badShares))
	})
}

func BenchmarkVerifyShare(b *testing.B) {
	secret := make([]byte, 32)
	_, _ = rand.Read(secret)
	shares, _ := Split(secret, 3, 5)

	b.ReportAllocs()
	for b.Loop() {
		VerifyShare(shares[0], shares[1:3])
	}
}

func BenchmarkVerifyAllShares(b *testing.B) {
	secret := make([]byte, 32)
	_, _ = rand.Read(secret)
	shares, _ := Split(secret, 3, 5)

	b.ReportAllocs()
	for b.Loop() {
		VerifyAllShares(shares)
	}
}

func TestLagrangeEvaluate(t *testing.T) {
	t.Run("empty slices", func(t *testing.T) {
		result := lagrangeEvaluate([]*big.Int{}, []*big.Int{}, big.NewInt(5))
		assert.Nil(t, result)
	})

	t.Run("mismatched lengths", func(t *testing.T) {
		xs := []*big.Int{big.NewInt(1), big.NewInt(2)}
		ys := []*big.Int{big.NewInt(1)}
		result := lagrangeEvaluate(xs, ys, big.NewInt(5))
		assert.Nil(t, result)
	})

	t.Run("duplicate x coordinates", func(t *testing.T) {
		xs := []*big.Int{big.NewInt(1), big.NewInt(1), big.NewInt(3)}
		ys := []*big.Int{big.NewInt(10), big.NewInt(20), big.NewInt(30)}
		result := lagrangeEvaluate(xs, ys, big.NewInt(5))
		assert.Nil(t, result)
	})
}
