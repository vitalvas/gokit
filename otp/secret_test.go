package otp

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateSecret(t *testing.T) {
	t.Run("minimum length enforced", func(t *testing.T) {
		secret, err := GenerateSecret(0)
		require.NoError(t, err)
		assert.Len(t, secret, 32) // 20 bytes -> 32 base32 chars

		key, err := decodeSecret(secret)
		require.NoError(t, err)
		assert.Len(t, key, 20)
	})

	t.Run("custom length", func(t *testing.T) {
		secret, err := GenerateSecret(32)
		require.NoError(t, err)

		key, err := decodeSecret(secret)
		require.NoError(t, err)
		assert.Len(t, key, 32)
	})

	t.Run("secrets are unique and usable", func(t *testing.T) {
		a, err := GenerateSecret(20)
		require.NoError(t, err)

		b, err := GenerateSecret(20)
		require.NoError(t, err)
		assert.NotEqual(t, a, b)

		_, err = TOTP(a, time.Now(), nil)
		assert.NoError(t, err)
	})
}

func TestDecodeSecret(t *testing.T) {
	want, err := decodeSecret(rfcSecretSHA1)
	require.NoError(t, err)

	t.Run("tolerates lower case, spaces, and padding", func(t *testing.T) {
		for name, variant := range map[string]string{
			"lower case": strings.ToLower(rfcSecretSHA1),
			"spaces":     strings.Join([]string{rfcSecretSHA1[:4], rfcSecretSHA1[4:]}, " "),
			"padded":     strings.Join([]string{rfcSecretSHA1, "======"}, ""),
		} {
			got, err := decodeSecret(variant)
			require.NoError(t, err, name)
			assert.Equal(t, want, got, name)
		}
	})

	t.Run("rejects invalid input", func(t *testing.T) {
		for _, bad := range []string{"", "=", "1!", "ABC18"} {
			_, err := decodeSecret(bad)
			assert.ErrorIs(t, err, ErrInvalidSecret, "input %q", bad)
		}
	})
}
