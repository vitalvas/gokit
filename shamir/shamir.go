package shamir

import (
	"math/big"
)

// Split divides a secret into n shares, where any k shares can reconstruct the secret.
// The secret is treated as a big-endian byte representation of a field element.
//
// Parameters:
//   - secret: the secret data to split (must be non-empty, at most 65535 bytes, and numerically smaller than the field prime)
//   - threshold: minimum number of shares required for reconstruction (k)
//   - total: total number of shares to generate (n)
//
// Returns a slice of Share objects that can be distributed to participants.
func Split(secret []byte, threshold, total int) ([]*Share, error) {
	if len(secret) > maxShareValue {
		return nil, ErrSecretTooLarge
	}
	if len(secret) == 0 {
		return nil, ErrEmptySecret
	}

	if threshold < 2 || threshold > maxShareValue {
		return nil, ErrInvalidThreshold
	}

	if total < threshold || total > maxShareValue {
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
			SecretLen: len(secret),
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
	if len(secret) > maxShareValue {
		return nil, ErrSecretTooLarge
	}
	if len(secret) == 0 {
		return nil, ErrEmptySecret
	}

	if threshold < 2 || threshold > maxShareValue {
		return nil, ErrInvalidThreshold
	}

	if len(xCoords) < threshold || len(xCoords) > maxShareValue {
		return nil, ErrInvalidTotal
	}

	// Verify all x-coordinates are non-zero and unique
	seen := make(map[string]bool)
	for _, x := range xCoords {
		if x == nil || x.Sign() <= 0 || x.Cmp(prime) >= 0 {
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
			SecretLen: len(secret),
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
	knownLen, err := validateShares(shares)
	if err != nil {
		return nil, err
	}
	if secretLen < 0 || secretLen > maxShareValue || (knownLen > 0 && knownLen != secretLen) {
		return nil, ErrInconsistentShares
	}
	value, err := reconstruct(shares)
	if err != nil {
		return nil, err
	}
	if len(value.Bytes()) > secretLen {
		return nil, ErrInconsistentShares
	}
	return fieldElementToBytes(value, secretLen), nil
}

// CombineAuto reconstructs the exact secret using consistent known lengths.
// Legacy shares with SecretLen == 0 may be mixed with current shares; a known
// length from any share is used. With only legacy shares, leading zeros are lost.
func CombineAuto(shares []*Share) ([]byte, error) {
	secretLen, err := validateShares(shares)
	if err != nil {
		return nil, err
	}
	value, err := reconstruct(shares)
	if err != nil {
		return nil, err
	}
	if secretLen == 0 {
		return value.Bytes(), nil
	}
	if len(value.Bytes()) > secretLen {
		return nil, ErrInconsistentShares
	}
	return fieldElementToBytes(value, secretLen), nil
}

func validateShares(shares []*Share) (int, error) {
	if len(shares) == 0 {
		return 0, ErrInsufficientShares
	}
	if shares[0] == nil {
		return 0, ErrInvalidShareFormat
	}
	threshold, total := shares[0].Threshold, shares[0].Total
	if threshold < 2 || threshold > maxShareValue || total < threshold || total > maxShareValue {
		return 0, ErrInvalidShareFormat
	}
	if len(shares) < threshold {
		return 0, ErrInsufficientShares
	}
	seen := make(map[string]bool, len(shares))
	knownLen := 0
	for _, share := range shares {
		if share == nil || share.X == nil || share.Y == nil {
			return 0, ErrInvalidShareFormat
		}
		if share.Threshold != threshold || share.Total != total || share.SecretLen < 0 || share.SecretLen > maxShareValue {
			return 0, ErrInconsistentShares
		}
		if share.X.Sign() <= 0 || share.X.Cmp(prime) >= 0 {
			return 0, ErrInvalidShareX
		}
		if share.Y.Sign() < 0 || share.Y.Cmp(prime) >= 0 {
			return 0, ErrInvalidShareFormat
		}
		if share.SecretLen > 0 {
			if knownLen != 0 && knownLen != share.SecretLen {
				return 0, ErrInconsistentShares
			}
			knownLen = share.SecretLen
		}
		key := share.X.String()
		if seen[key] {
			return 0, ErrDuplicateShares
		}
		seen[key] = true
	}
	return knownLen, nil
}

func reconstruct(shares []*Share) (*big.Int, error) {
	threshold := shares[0].Threshold
	xs, ys := make([]*big.Int, threshold), make([]*big.Int, threshold)
	for i, share := range shares[:threshold] {
		xs[i], ys[i] = share.X, share.Y
	}
	value := lagrangeInterpolate(xs, ys)
	if value == nil {
		return nil, ErrVerificationFailed
	}
	return value, nil
}
