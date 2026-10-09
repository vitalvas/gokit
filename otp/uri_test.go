package otp

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKeyURI(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		uri := KeyURI("Example", "alice@example.com", "ABC234", nil)

		assert.Equal(t,
			"otpauth://totp/Example:alice@example.com?algorithm=SHA1&digits=6&issuer=Example&period=30&secret=ABC234",
			uri)
	})

	t.Run("custom options and escaping", func(t *testing.T) {
		uri := KeyURI("My App", "bob smith", "ABC234", &Options{Algorithm: SHA256, Digits: 8, Period: 60})

		assert.Equal(t,
			"otpauth://totp/My%20App:bob%20smith?algorithm=SHA256&digits=8&issuer=My+App&period=60&secret=ABC234",
			uri)
	})

	t.Run("hotp with counter", func(t *testing.T) {
		uri := HOTPKeyURI("Example", "alice@example.com", "ABC234", 5, nil)

		assert.Equal(t,
			"otpauth://hotp/Example:alice@example.com?algorithm=SHA1&counter=5&digits=6&issuer=Example&secret=ABC234",
			uri)
	})
}

func TestParseKeyURI(t *testing.T) {
	t.Run("totp round trip", func(t *testing.T) {
		opts := &Options{Algorithm: SHA256, Digits: 8, Period: 60}
		uri := KeyURI("My App", "bob smith", rfcSecretSHA1, opts)

		k, err := ParseKeyURI(uri)
		require.NoError(t, err)

		assert.Equal(t, TypeTOTP, k.Type)
		assert.Equal(t, "My App", k.Issuer)
		assert.Equal(t, "bob smith", k.Account)
		assert.Equal(t, rfcSecretSHA1, k.Secret)
		assert.Equal(t, SHA256, k.Options.Algorithm)
		assert.Equal(t, 8, k.Options.Digits)
		assert.Equal(t, uint64(60), k.Options.Period)

		// The parsed key generates the same codes as the original secret.
		now := time.Unix(1234567890, 0)
		want, err := TOTP(rfcSecretSHA1, now, opts)
		require.NoError(t, err)

		got, err := TOTP(k.Secret, now, &k.Options)
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("hotp round trip", func(t *testing.T) {
		uri := HOTPKeyURI("Example", "alice@example.com", rfcSecretSHA1, 42, nil)

		k, err := ParseKeyURI(uri)
		require.NoError(t, err)

		assert.Equal(t, TypeHOTP, k.Type)
		assert.Equal(t, uint64(42), k.Counter)
		assert.Equal(t, SHA1, k.Options.Algorithm)
		assert.Equal(t, 6, k.Options.Digits)
	})

	t.Run("defaults applied", func(t *testing.T) {
		k, err := ParseKeyURI(fmt.Sprintf("otpauth://totp/alice?secret=%s", rfcSecretSHA1))
		require.NoError(t, err)

		assert.Empty(t, k.Issuer)
		assert.Equal(t, "alice", k.Account)
		assert.Equal(t, SHA1, k.Options.Algorithm)
		assert.Equal(t, 6, k.Options.Digits)
		assert.Equal(t, uint64(30), k.Options.Period)
	})

	t.Run("label variants", func(t *testing.T) {
		k, err := ParseKeyURI(fmt.Sprintf("otpauth://totp/Example:%%20alice?secret=%s", rfcSecretSHA1))
		require.NoError(t, err)
		assert.Equal(t, "Example", k.Issuer)
		assert.Equal(t, "alice", k.Account)

		// Query issuer wins over the label prefix.
		k, err = ParseKeyURI(fmt.Sprintf("otpauth://totp/Old:alice?secret=%s&issuer=New", rfcSecretSHA1))
		require.NoError(t, err)
		assert.Equal(t, "New", k.Issuer)
	})

	t.Run("key URI round trip", func(t *testing.T) {
		uri := HOTPKeyURI("Example", "alice", rfcSecretSHA1, 7, nil)

		k, err := ParseKeyURI(uri)
		require.NoError(t, err)
		assert.Equal(t, uri, k.URI())

		// Unknown type falls back to totp.
		loose := Key{Account: "a", Secret: rfcSecretSHA1, Options: (*Options)(nil).withDefaults()}
		parsed, err := ParseKeyURI(loose.URI())
		require.NoError(t, err)
		assert.Equal(t, TypeTOTP, parsed.Type)
	})

	t.Run("lowercase algorithm accepted", func(t *testing.T) {
		k, err := ParseKeyURI(fmt.Sprintf("otpauth://totp/a?secret=%s&algorithm=sha512", rfcSecretSHA1))
		require.NoError(t, err)
		assert.Equal(t, SHA512, k.Options.Algorithm)
	})

	t.Run("invalid URIs", func(t *testing.T) {
		for name, uri := range map[string]string{
			"wrong scheme":     fmt.Sprintf("https://totp/a?secret=%s", rfcSecretSHA1),
			"wrong type":       fmt.Sprintf("otpauth://ocra/a?secret=%s", rfcSecretSHA1),
			"missing secret":   "otpauth://totp/a",
			"bad secret":       "otpauth://totp/a?secret=1!",
			"bad algorithm":    fmt.Sprintf("otpauth://totp/a?secret=%s&algorithm=MD5", rfcSecretSHA1),
			"bad digits":       fmt.Sprintf("otpauth://totp/a?secret=%s&digits=5", rfcSecretSHA1),
			"bad period":       fmt.Sprintf("otpauth://totp/a?secret=%s&period=0", rfcSecretSHA1),
			"missing counter":  fmt.Sprintf("otpauth://hotp/a?secret=%s", rfcSecretSHA1),
			"negative counter": fmt.Sprintf("otpauth://hotp/a?secret=%s&counter=-1", rfcSecretSHA1),
			"not a URI at all": "::",
		} {
			_, err := ParseKeyURI(uri)
			assert.ErrorIs(t, err, ErrInvalidURI, name)
		}
	})
}

func FuzzParseKeyURI(f *testing.F) {
	f.Add("otpauth://totp/Example:alice?secret=GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ")
	f.Add("otpauth://hotp/a?secret=GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ&counter=5")
	f.Add("otpauth://totp/x?secret=AAAA&algorithm=SHA256&digits=8&period=60")
	f.Add("::")

	f.Fuzz(func(t *testing.T, raw string) {
		k, err := ParseKeyURI(raw)
		if err != nil {
			return
		}

		// Regenerate a URI from the parsed key and reparse: the key
		// material and options must survive the round trip.
		uri := k.URI()

		again, err := ParseKeyURI(uri)
		if err != nil {
			t.Fatalf("regenerated URI %q failed to parse: %v", uri, err)
		}

		if again.Secret != k.Secret || again.Type != k.Type || again.Counter != k.Counter ||
			again.Options.Algorithm != k.Options.Algorithm || again.Options.Digits != k.Options.Digits {
			t.Fatalf("round trip mismatch: %+v != %+v", again, k)
		}
	})
}
