package otp

import (
	"encoding/base32"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHOTP(t *testing.T) {
	t.Run("RFC 4226 appendix D vectors", func(t *testing.T) {
		codes := []string{
			"755224", "287082", "359152", "969429", "338314",
			"254676", "287922", "162583", "399871", "520489",
		}

		for counter, want := range codes {
			code, err := HOTP(rfcSecretSHA1, uint64(counter), nil)
			require.NoError(t, err)
			assert.Equal(t, want, code, "counter %d", counter)
		}
	})

	t.Run("invalid secret", func(t *testing.T) {
		_, err := HOTP("not!base32", 0, nil)
		assert.ErrorIs(t, err, ErrInvalidSecret)
	})
}

func TestVerifyHOTP(t *testing.T) {
	t.Run("matching code", func(t *testing.T) {
		ok, err := VerifyHOTP(rfcSecretSHA1, "755224", 0, nil)
		require.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("wrong code and wrong counter", func(t *testing.T) {
		ok, err := VerifyHOTP(rfcSecretSHA1, "000000", 0, nil)
		require.NoError(t, err)
		assert.False(t, ok)

		ok, err = VerifyHOTP(rfcSecretSHA1, "755224", 1, nil)
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("invalid secret", func(t *testing.T) {
		_, err := VerifyHOTP("!", "755224", 0, nil)
		assert.ErrorIs(t, err, ErrInvalidSecret)
	})
}

func TestSyncHOTP(t *testing.T) {
	t.Run("finds counter ahead within window", func(t *testing.T) {
		code, err := HOTP(rfcSecretSHA1, 7, nil)
		require.NoError(t, err)

		matched, ok, err := SyncHOTP(rfcSecretSHA1, code, 2, 10, nil)
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, uint64(7), matched)
	})

	t.Run("outside window not found", func(t *testing.T) {
		code, err := HOTP(rfcSecretSHA1, 20, nil)
		require.NoError(t, err)

		_, ok, err := SyncHOTP(rfcSecretSHA1, code, 2, 10, nil)
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("window clamped", func(t *testing.T) {
		code, err := HOTP(rfcSecretSHA1, maxWindow+2, nil)
		require.NoError(t, err)

		_, ok, err := SyncHOTP(rfcSecretSHA1, code, 0, 1<<40, nil)
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("counter space exhaustion does not wrap", func(t *testing.T) {
		code, err := HOTP(rfcSecretSHA1, 0, nil)
		require.NoError(t, err)

		_, ok, err := SyncHOTP(rfcSecretSHA1, code, math.MaxUint64-2, 10, nil)
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("invalid secret", func(t *testing.T) {
		_, _, err := SyncHOTP("!", "755224", 0, 10, nil)
		assert.ErrorIs(t, err, ErrInvalidSecret)
	})
}

func FuzzHOTP(f *testing.F) {
	f.Add([]byte("12345678901234567890"), uint64(0), 6)
	f.Add([]byte{0xff}, uint64(1<<63), 8)
	f.Add([]byte("k"), uint64(42), -5)
	f.Add([]byte("secret"), uint64(9), 100)

	f.Fuzz(func(t *testing.T, rawSecret []byte, counter uint64, digits int) {
		if len(rawSecret) == 0 {
			return
		}

		secret := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(rawSecret)
		opts := &Options{Digits: digits}

		code, err := HOTP(secret, counter, opts)
		if err != nil {
			t.Fatalf("HOTP failed on valid secret: %v", err)
		}

		if len(code) != opts.withDefaults().Digits {
			t.Fatalf("code %q has wrong length", code)
		}

		for _, c := range code {
			if c < '0' || c > '9' {
				t.Fatalf("code %q contains non-digit", code)
			}
		}

		ok, err := VerifyHOTP(secret, code, counter, opts)
		if err != nil || !ok {
			t.Fatalf("generated code must verify: ok=%v err=%v", ok, err)
		}
	})
}
