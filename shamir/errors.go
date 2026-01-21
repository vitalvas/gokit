package shamir

import "errors"

var (
	// ErrInvalidThreshold is returned when threshold is less than 2.
	ErrInvalidThreshold = errors.New("shamir: threshold must be at least 2")

	// ErrInvalidTotal is returned when total shares is less than threshold.
	ErrInvalidTotal = errors.New("shamir: total shares must be at least equal to threshold")

	// ErrSecretTooLarge is returned when the secret exceeds the field size.
	ErrSecretTooLarge = errors.New("shamir: secret is too large for the field")

	// ErrInsufficientShares is returned when not enough shares are provided for reconstruction.
	ErrInsufficientShares = errors.New("shamir: insufficient shares for reconstruction")

	// ErrDuplicateShares is returned when duplicate share indices are provided.
	ErrDuplicateShares = errors.New("shamir: duplicate share indices detected")

	// ErrInvalidShareFormat is returned when share data is malformed.
	ErrInvalidShareFormat = errors.New("shamir: invalid share format")

	// ErrUnsupportedVersion is returned when share version is not supported.
	ErrUnsupportedVersion = errors.New("shamir: unsupported share version")

	// ErrInvalidShareX is returned when share X coordinate is zero.
	ErrInvalidShareX = errors.New("shamir: share X coordinate must be non-zero")

	// ErrEmptySecret is returned when trying to split an empty secret.
	ErrEmptySecret = errors.New("shamir: secret cannot be empty")

	// ErrInconsistentShares is returned when shares have inconsistent parameters.
	ErrInconsistentShares = errors.New("shamir: shares have inconsistent threshold or total values")

	// ErrVerificationFailed is returned when share verification fails.
	ErrVerificationFailed = errors.New("shamir: share verification failed")
)
