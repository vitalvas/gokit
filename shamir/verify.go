package shamir

import (
	"math/big"
)

// VerifyShare checks if a share is consistent with other shares.
// This works by checking if all provided shares lie on the same polynomial.
// Returns true if the share is valid, false otherwise.
func VerifyShare(share *Share, otherShares []*Share) bool {
	if share == nil || len(otherShares) < share.Threshold-1 {
		return false
	}

	// We need at least threshold shares to verify
	// The share being verified plus (threshold-1) other shares
	allShares := make([]*Share, 0, len(otherShares)+1)
	allShares = append(allShares, share)
	allShares = append(allShares, otherShares...)

	if len(allShares) < share.Threshold {
		return false
	}

	// Verify all shares have consistent threshold
	for _, s := range allShares {
		if s.Threshold != share.Threshold {
			return false
		}
	}

	// Check for duplicate x-coordinates
	seen := make(map[string]bool)
	for _, s := range allShares {
		key := s.X.String()
		if seen[key] {
			return false
		}
		seen[key] = true
	}

	// Take threshold shares and interpolate
	// Then verify that all other shares lie on the same polynomial
	baseShares := allShares[:share.Threshold]

	xs := make([]*big.Int, share.Threshold)
	ys := make([]*big.Int, share.Threshold)
	for i, s := range baseShares {
		xs[i] = s.X
		ys[i] = s.Y
	}

	// For each remaining share, verify it lies on the polynomial
	for i := share.Threshold; i < len(allShares); i++ {
		extraShare := allShares[i]

		// Compute what y should be at extraShare.X using Lagrange interpolation
		expectedY := lagrangeEvaluate(xs, ys, extraShare.X)
		if expectedY == nil {
			return false
		}

		if expectedY.Cmp(extraShare.Y) != 0 {
			return false
		}
	}

	return true
}

// VerifyAllShares verifies that all provided shares are mutually consistent.
// This checks that all shares lie on the same polynomial of the expected degree.
func VerifyAllShares(shares []*Share) bool {
	if len(shares) == 0 {
		return false
	}

	threshold := shares[0].Threshold
	if len(shares) < threshold {
		return false
	}

	// Verify all shares have consistent parameters
	for _, share := range shares {
		if share.Threshold != threshold {
			return false
		}
	}

	// Check for duplicate x-coordinates
	seen := make(map[string]bool)
	for _, s := range shares {
		key := s.X.String()
		if seen[key] {
			return false
		}
		seen[key] = true
	}

	// Use threshold shares to define the polynomial
	baseShares := shares[:threshold]

	xs := make([]*big.Int, threshold)
	ys := make([]*big.Int, threshold)
	for i, s := range baseShares {
		xs[i] = s.X
		ys[i] = s.Y
	}

	// Verify all remaining shares lie on the polynomial
	for i := threshold; i < len(shares); i++ {
		extraShare := shares[i]

		expectedY := lagrangeEvaluate(xs, ys, extraShare.X)
		if expectedY == nil {
			return false
		}

		if expectedY.Cmp(extraShare.Y) != 0 {
			return false
		}
	}

	return true
}

// lagrangeEvaluate performs Lagrange interpolation to find f(x) at a given point.
func lagrangeEvaluate(xs, ys []*big.Int, x *big.Int) *big.Int {
	if len(xs) != len(ys) || len(xs) == 0 {
		return nil
	}

	result := big.NewInt(0)

	for i := range xs {
		// Calculate the Lagrange basis polynomial L_i(x)
		numerator := big.NewInt(1)
		denominator := big.NewInt(1)

		for j := range xs {
			if i == j {
				continue
			}

			// numerator *= (x - x_j)
			numerator = fieldMul(numerator, fieldSub(x, xs[j]))

			// denominator *= (x_i - x_j)
			denominator = fieldMul(denominator, fieldSub(xs[i], xs[j]))
		}

		// L_i(x) = numerator / denominator
		basis := fieldDiv(numerator, denominator)
		if basis == nil {
			return nil
		}

		// result += y_i * L_i(x)
		term := fieldMul(ys[i], basis)
		result = fieldAdd(result, term)
	}

	return result
}
