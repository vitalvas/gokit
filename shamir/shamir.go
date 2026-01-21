package shamir

import (
	"math/big"
)

// Split divides a secret into n shares, where any k shares can reconstruct the secret.
// The secret is treated as a big-endian byte representation of a field element.
//
// Parameters:
//   - secret: the secret data to split (must be non-empty and smaller than the field prime)
//   - threshold: minimum number of shares required for reconstruction (k)
//   - total: total number of shares to generate (n)
//
// Returns a slice of Share objects that can be distributed to participants.
func Split(secret []byte, threshold, total int) ([]*Share, error) {
	if len(secret) == 0 {
		return nil, ErrEmptySecret
	}

	if threshold < 2 {
		return nil, ErrInvalidThreshold
	}

	if total < threshold {
		return nil, ErrInvalidTotal
	}

	// Convert secret to field element
	secretInt := bytesToFieldElement(secret)

	// Verify secret is within field bounds
	if secretInt.Cmp(prime) >= 0 {
		return nil, ErrSecretTooLarge
	}

	// Create random polynomial with secret as constant term
	poly, err := newRandomPolynomial(secretInt, threshold)
	if err != nil {
		return nil, err
	}

	// Generate shares
	shares := make([]*Share, total)
	for i := range total {
		// x-coordinates are 1, 2, 3, ... (never 0)
		x := big.NewInt(int64(i + 1))
		y := poly.evaluate(x)

		shares[i] = &Share{
			X:         x,
			Y:         y,
			Threshold: threshold,
			Total:     total,
		}
	}

	return shares, nil
}

// SplitWithCustomX divides a secret using custom x-coordinates for shares.
// This is useful when you need deterministic or specific share indices.
//
// Parameters:
//   - secret: the secret data to split
//   - threshold: minimum number of shares required for reconstruction
//   - xCoords: x-coordinates for each share (must all be non-zero and unique)
func SplitWithCustomX(secret []byte, threshold int, xCoords []*big.Int) ([]*Share, error) {
	if len(secret) == 0 {
		return nil, ErrEmptySecret
	}

	if threshold < 2 {
		return nil, ErrInvalidThreshold
	}

	if len(xCoords) < threshold {
		return nil, ErrInvalidTotal
	}

	// Verify all x-coordinates are non-zero and unique
	seen := make(map[string]bool)
	for _, x := range xCoords {
		if x.Sign() == 0 {
			return nil, ErrInvalidShareX
		}
		key := x.String()
		if seen[key] {
			return nil, ErrDuplicateShares
		}
		seen[key] = true
	}

	// Convert secret to field element
	secretInt := bytesToFieldElement(secret)

	if secretInt.Cmp(prime) >= 0 {
		return nil, ErrSecretTooLarge
	}

	// Create random polynomial with secret as constant term
	poly, err := newRandomPolynomial(secretInt, threshold)
	if err != nil {
		return nil, err
	}

	// Generate shares
	shares := make([]*Share, len(xCoords))
	for i, x := range xCoords {
		y := poly.evaluate(x)

		shares[i] = &Share{
			X:         new(big.Int).Set(x),
			Y:         y,
			Threshold: threshold,
			Total:     len(xCoords),
		}
	}

	return shares, nil
}

// Combine reconstructs the secret from the given shares using Lagrange interpolation.
// At least threshold shares are required.
//
// Parameters:
//   - shares: slice of shares to combine
//   - secretLen: expected length of the reconstructed secret in bytes
//
// Returns the reconstructed secret.
func Combine(shares []*Share, secretLen int) ([]byte, error) {
	if len(shares) == 0 {
		return nil, ErrInsufficientShares
	}

	threshold := shares[0].Threshold
	if len(shares) < threshold {
		return nil, ErrInsufficientShares
	}

	// Verify shares have consistent parameters and unique x-coordinates
	seen := make(map[string]bool)
	for _, share := range shares {
		if share.Threshold != threshold {
			return nil, ErrInconsistentShares
		}
		key := share.X.String()
		if seen[key] {
			return nil, ErrDuplicateShares
		}
		seen[key] = true
	}

	// Use only the required number of shares
	usedShares := shares[:threshold]

	// Extract x and y coordinates
	xs := make([]*big.Int, threshold)
	ys := make([]*big.Int, threshold)
	for i, share := range usedShares {
		xs[i] = share.X
		ys[i] = share.Y
	}

	// Perform Lagrange interpolation to find f(0) = secret
	secretInt := lagrangeInterpolate(xs, ys)
	if secretInt == nil {
		return nil, ErrVerificationFailed
	}

	return fieldElementToBytes(secretInt, secretLen), nil
}

// CombineAuto reconstructs the secret from shares, automatically determining the secret length.
// This uses the minimum bytes needed to represent the secret value.
func CombineAuto(shares []*Share) ([]byte, error) {
	if len(shares) == 0 {
		return nil, ErrInsufficientShares
	}

	threshold := shares[0].Threshold
	if len(shares) < threshold {
		return nil, ErrInsufficientShares
	}

	// Verify shares have consistent parameters and unique x-coordinates
	seen := make(map[string]bool)
	for _, share := range shares {
		if share.Threshold != threshold {
			return nil, ErrInconsistentShares
		}
		key := share.X.String()
		if seen[key] {
			return nil, ErrDuplicateShares
		}
		seen[key] = true
	}

	// Use only the required number of shares
	usedShares := shares[:threshold]

	// Extract x and y coordinates
	xs := make([]*big.Int, threshold)
	ys := make([]*big.Int, threshold)
	for i, share := range usedShares {
		xs[i] = share.X
		ys[i] = share.Y
	}

	// Perform Lagrange interpolation to find f(0) = secret
	secretInt := lagrangeInterpolate(xs, ys)
	if secretInt == nil {
		return nil, ErrVerificationFailed
	}

	return secretInt.Bytes(), nil
}
