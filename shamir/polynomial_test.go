package shamir

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPolynomialEvaluate(t *testing.T) {
	// f(x) = 5 + 3x + 2x^2
	coeffs := []*big.Int{
		big.NewInt(5), // constant term
		big.NewInt(3), // x coefficient
		big.NewInt(2), // x^2 coefficient
	}
	poly := newPolynomial(coeffs)

	tests := []struct {
		x        int64
		expected int64
	}{
		{0, 5},  // f(0) = 5
		{1, 10}, // f(1) = 5 + 3 + 2 = 10
		{2, 19}, // f(2) = 5 + 6 + 8 = 19
		{3, 32}, // f(3) = 5 + 9 + 18 = 32
	}

	for _, tt := range tests {
		result := poly.evaluate(big.NewInt(tt.x))
		assert.Equal(t, tt.expected, result.Int64(), "f(%d)", tt.x)
	}
}

func TestPolynomialEvaluateEmpty(t *testing.T) {
	poly := newPolynomial([]*big.Int{})
	result := poly.evaluate(big.NewInt(5))
	assert.Equal(t, int64(0), result.Int64())
}

func TestNewRandomPolynomial(t *testing.T) {
	secret := big.NewInt(42)

	t.Run("valid polynomial", func(t *testing.T) {
		poly, err := newRandomPolynomial(secret, 3)
		require.NoError(t, err)
		assert.Len(t, poly.coefficients, 3)
		assert.Equal(t, int64(42), poly.coefficients[0].Int64())
	})

	t.Run("invalid threshold", func(t *testing.T) {
		_, err := newRandomPolynomial(secret, 0)
		assert.ErrorIs(t, err, ErrInvalidThreshold)
	})
}

func TestLagrangeInterpolate(t *testing.T) {
	t.Run("quadratic polynomial", func(t *testing.T) {
		// Points from f(x) = 5 + 3x + 2x^2
		// f(1) = 10, f(2) = 19, f(3) = 32
		xs := []*big.Int{big.NewInt(1), big.NewInt(2), big.NewInt(3)}
		ys := []*big.Int{big.NewInt(10), big.NewInt(19), big.NewInt(32)}

		result := lagrangeInterpolate(xs, ys)
		assert.Equal(t, int64(5), result.Int64()) // f(0) = 5
	})

	t.Run("linear polynomial", func(t *testing.T) {
		// Points from f(x) = 3 + 2x
		// f(1) = 5, f(2) = 7
		xs := []*big.Int{big.NewInt(1), big.NewInt(2)}
		ys := []*big.Int{big.NewInt(5), big.NewInt(7)}

		result := lagrangeInterpolate(xs, ys)
		assert.Equal(t, int64(3), result.Int64()) // f(0) = 3
	})

	t.Run("empty input", func(t *testing.T) {
		result := lagrangeInterpolate([]*big.Int{}, []*big.Int{})
		assert.Nil(t, result)
	})

	t.Run("mismatched lengths", func(t *testing.T) {
		xs := []*big.Int{big.NewInt(1), big.NewInt(2)}
		ys := []*big.Int{big.NewInt(5)}
		result := lagrangeInterpolate(xs, ys)
		assert.Nil(t, result)
	})

	t.Run("duplicate x coordinates", func(t *testing.T) {
		xs := []*big.Int{big.NewInt(1), big.NewInt(1), big.NewInt(3)}
		ys := []*big.Int{big.NewInt(10), big.NewInt(20), big.NewInt(30)}
		result := lagrangeInterpolate(xs, ys)
		assert.Nil(t, result)
	})
}
