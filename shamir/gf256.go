package shamir

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
)

// GF256Share represents a lightweight share using GF(2^8) field.
// Format: [y_0, y_1, ..., y_n, x] where x is the last byte.
// Share size is always len(secret) + 1.
type GF256Share []byte

// gf256Add adds two numbers in GF(2^8). Also used for subtraction (symmetric).
func gf256Add(a, b uint8) uint8 {
	return a ^ b
}

// gf256Mult multiplies two numbers in GF(2^8) using Russian peasant multiplication.
// Uses the irreducible polynomial x^8 + x^4 + x^3 + x + 1 (0x11B).
func gf256Mult(a, b uint8) uint8 {
	var r uint8
	for range 8 {
		if b&1 == 1 {
			r ^= a
		}
		hiBit := a & 0x80
		a <<= 1
		if hiBit == 0x80 {
			a ^= 0x1B // Reduction polynomial (x^8 + x^4 + x^3 + x + 1) & 0xFF
		}
		b >>= 1
	}
	return r
}

// gf256Inverse calculates the multiplicative inverse in GF(2^8) using exponentiation.
// a^(-1) = a^254 in GF(2^8)
func gf256Inverse(a uint8) uint8 {
	if a == 0 {
		return 0
	}

	// Compute a^254 using square-and-multiply
	// 254 = 11111110 in binary
	b := gf256Mult(a, a)   // a^2
	c := gf256Mult(a, b)   // a^3
	b = gf256Mult(c, c)    // a^6
	b = gf256Mult(b, b)    // a^12
	c = gf256Mult(b, c)    // a^15
	b = gf256Mult(b, b)    // a^24
	b = gf256Mult(b, b)    // a^48
	b = gf256Mult(b, c)    // a^63
	b = gf256Mult(b, b)    // a^126
	b = gf256Mult(a, b)    // a^127
	return gf256Mult(b, b) // a^254
}

// gf256Div divides two numbers in GF(2^8).
func gf256Div(a, b uint8) uint8 {
	if b == 0 {
		panic("shamir: division by zero in GF(2^8)")
	}
	if a == 0 {
		return 0
	}
	return gf256Mult(a, gf256Inverse(b))
}

// gf256Polynomial represents a polynomial over GF(2^8).
type gf256Polynomial struct {
	coefficients []uint8
}

// newGF256Polynomial creates a random polynomial with given intercept (secret byte).
func newGF256Polynomial(intercept uint8, degree int) (*gf256Polynomial, error) {
	coefficients := make([]uint8, degree+1)
	coefficients[0] = intercept

	if degree > 0 {
		if _, err := rand.Read(coefficients[1:]); err != nil {
			return nil, err
		}
	}

	return &gf256Polynomial{coefficients: coefficients}, nil
}

// evaluate evaluates the polynomial at x using Horner's method.
func (p *gf256Polynomial) evaluate(x uint8) uint8 {
	if x == 0 {
		return p.coefficients[0]
	}

	degree := len(p.coefficients) - 1
	result := p.coefficients[degree]

	for i := degree - 1; i >= 0; i-- {
		result = gf256Add(gf256Mult(result, x), p.coefficients[i])
	}

	return result
}

// gf256Interpolate performs Lagrange interpolation at x=0 to recover the secret.
func gf256Interpolate(xSamples, ySamples []uint8) uint8 {
	if len(xSamples) != len(ySamples) || len(xSamples) == 0 {
		return 0
	}

	var result uint8
	for i := range xSamples {
		var basis uint8 = 1
		for j := range xSamples {
			if i == j {
				continue
			}
			// basis *= (0 - x_j) / (x_i - x_j)
			// In GF(2^8): 0 - x_j = x_j (subtraction is XOR)
			num := xSamples[j]
			denom := gf256Add(xSamples[i], xSamples[j])
			basis = gf256Mult(basis, gf256Div(num, denom))
		}
		result = gf256Add(result, gf256Mult(ySamples[i], basis))
	}

	return result
}

// SplitGF256 splits a secret into shares using GF(2^8) field arithmetic.
// Each share is len(secret)+1 bytes, with the X coordinate as the last byte.
//
// Parameters:
//   - secret: the secret data to split (any length)
//   - threshold: minimum shares required for reconstruction (2-255)
//   - total: total shares to generate (threshold-255)
//
// Returns shares where each share[i] = [y_0, y_1, ..., y_n, x]
func SplitGF256(secret []byte, threshold, total int) ([]GF256Share, error) {
	if len(secret) == 0 {
		return nil, ErrEmptySecret
	}

	if threshold < 2 || threshold > 255 {
		return nil, ErrInvalidThreshold
	}

	if total < threshold || total > 255 {
		return nil, ErrInvalidTotal
	}

	// Generate unique random X coordinates
	xCoords, err := generateUniqueXCoords(total)
	if err != nil {
		return nil, err
	}

	// Allocate output shares
	shares := make([]GF256Share, total)
	for i := range shares {
		shares[i] = make([]byte, len(secret)+1)
		shares[i][len(secret)] = xCoords[i] // X coordinate is last byte
	}

	// For each byte of the secret, create a polynomial and evaluate at each X
	for byteIdx, secretByte := range secret {
		poly, err := newGF256Polynomial(secretByte, threshold-1)
		if err != nil {
			return nil, err
		}

		for shareIdx := range total {
			x := xCoords[shareIdx]
			y := poly.evaluate(x)
			shares[shareIdx][byteIdx] = y
		}
	}

	return shares, nil
}

// CombineGF256 reconstructs the secret from shares.
// Requires at least threshold shares (threshold is inferred from share count).
func CombineGF256(shares []GF256Share) ([]byte, error) {
	if len(shares) < 2 {
		return nil, ErrInsufficientShares
	}

	// Verify all shares have the same length
	shareLen := len(shares[0])
	if shareLen < 2 {
		return nil, ErrInvalidShareFormat
	}

	for _, share := range shares[1:] {
		if len(share) != shareLen {
			return nil, ErrInconsistentShares
		}
	}

	// Extract X coordinates and check for duplicates
	xCoords := make([]uint8, len(shares))
	seen := make(map[uint8]bool)
	for i, share := range shares {
		x := share[shareLen-1]
		if x == 0 {
			return nil, ErrInvalidShareX
		}
		if seen[x] {
			return nil, ErrDuplicateShares
		}
		seen[x] = true
		xCoords[i] = x
	}

	// Reconstruct each byte of the secret
	secretLen := shareLen - 1
	secret := make([]byte, secretLen)

	for byteIdx := range secretLen {
		ySamples := make([]uint8, len(shares))
		for shareIdx, share := range shares {
			ySamples[shareIdx] = share[byteIdx]
		}
		secret[byteIdx] = gf256Interpolate(xCoords, ySamples)
	}

	return secret, nil
}

// generateUniqueXCoords generates n unique random non-zero bytes.
func generateUniqueXCoords(n int) ([]uint8, error) {
	if n > 255 {
		return nil, ErrInvalidTotal
	}

	result := make([]uint8, n)
	seen := make(map[uint8]bool)
	buf := make([]byte, 1)

	for i := 0; i < n; {
		if _, err := rand.Read(buf); err != nil {
			return nil, err
		}
		x := buf[0]
		if x == 0 || seen[x] {
			continue
		}
		seen[x] = true
		result[i] = x
		i++
	}

	return result, nil
}

// X returns the X coordinate of the share.
func (s GF256Share) X() uint8 {
	if len(s) == 0 {
		return 0
	}
	return s[len(s)-1]
}

// Y returns the Y values (all bytes except the last).
func (s GF256Share) Y() []byte {
	if len(s) < 2 {
		return nil
	}
	return s[:len(s)-1]
}

// Clone creates a copy of the share.
func (s GF256Share) Clone() GF256Share {
	clone := make([]byte, len(s))
	copy(clone, s)
	return clone
}

// Equal checks if two shares are equal in constant time.
func (s GF256Share) Equal(other GF256Share) bool {
	if len(s) != len(other) {
		return false
	}
	return subtle.ConstantTimeCompare(s, other) == 1
}

// String returns the share as a base64-encoded string.
func (s GF256Share) String() string {
	return base64.StdEncoding.EncodeToString(s)
}

// ParseGF256ShareString deserializes a GF256 share from a base64-encoded string.
func ParseGF256ShareString(str string) (GF256Share, error) {
	data, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		return nil, errors.Join(ErrInvalidShareFormat, err)
	}
	if len(data) < 2 {
		return nil, ErrInvalidShareFormat
	}
	if data[len(data)-1] == 0 {
		return nil, ErrInvalidShareX
	}
	return GF256Share(data), nil
}
