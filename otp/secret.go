package otp

import (
	"crypto/rand"
	"encoding/base32"
	"strings"
)

// GenerateSecret returns a new random secret of n bytes, base32-encoded
// without padding. Values of n below 20 (the RFC 4226 recommended minimum)
// are raised to 20.
func GenerateSecret(n int) (string, error) {
	if n < minSecret {
		n = minSecret
	}

	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}

	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(buf), nil
}

// decodeSecret decodes a base32 secret, tolerating lower case, spaces, and
// trailing padding, as commonly produced by provisioning tools.
func decodeSecret(secret string) ([]byte, error) {
	s := strings.ToUpper(strings.ReplaceAll(secret, " ", ""))
	s = strings.TrimRight(s, "=")

	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(s)
	if err != nil || len(key) == 0 {
		return nil, ErrInvalidSecret
	}

	return key, nil
}
