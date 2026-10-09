package otp

import (
	"encoding/base32"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func b32(s string) string {
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString([]byte(s))
}

// RFC test secrets: ASCII digit sequences of 20, 32, and 64 bytes.
var (
	rfcSecretSHA1   = b32("12345678901234567890")
	rfcSecretSHA256 = b32("12345678901234567890123456789012")
	rfcSecretSHA512 = b32("1234567890123456789012345678901234567890123456789012345678901234")
)

func TestOptions(t *testing.T) {
	t.Run("nil gives defaults", func(t *testing.T) {
		o := (*Options)(nil).withDefaults()

		assert.Equal(t, SHA1, o.Algorithm)
		assert.Equal(t, uint64(DefaultPeriod), o.Period)
		assert.Equal(t, DefaultDigits, o.Digits)
		assert.Equal(t, uint64(DefaultSkew), o.Skew)
	})

	t.Run("digits clamped to range", func(t *testing.T) {
		assert.Equal(t, 6, (&Options{Digits: 1}).withDefaults().Digits)
		assert.Equal(t, 6, (&Options{Digits: -3}).withDefaults().Digits)
		assert.Equal(t, 10, (&Options{Digits: 100}).withDefaults().Digits)
		assert.Equal(t, 7, (&Options{Digits: 7}).withDefaults().Digits)
		assert.Equal(t, 10, (&Options{Digits: 10}).withDefaults().Digits)
	})

	t.Run("ten digit codes", func(t *testing.T) {
		code, err := HOTP(rfcSecretSHA1, 0, &Options{Digits: 10})
		require.NoError(t, err)
		assert.Len(t, code, 10)
		assert.Equal(t, "1284755224", code) // 31-bit value 1284755224, RFC 4226 appendix D
	})

	t.Run("skew clamped", func(t *testing.T) {
		assert.Equal(t, uint64(maxSkew), (&Options{Skew: 1 << 40}).withDefaults().Skew)
	})
}
