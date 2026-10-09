package otp

import (
	"encoding/base32"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTOTP(t *testing.T) {
	t.Run("RFC 6238 appendix B vectors", func(t *testing.T) {
		times := []int64{59, 1111111109, 1111111111, 1234567890, 2000000000, 20000000000}

		vectors := map[Algorithm]struct {
			secret string
			codes  []string
		}{
			SHA1:   {rfcSecretSHA1, []string{"94287082", "07081804", "14050471", "89005924", "69279037", "65353130"}},
			SHA256: {rfcSecretSHA256, []string{"46119246", "68084774", "67062674", "91819424", "90698825", "77737706"}},
			SHA512: {rfcSecretSHA512, []string{"90693936", "25091201", "99943326", "93441116", "38618901", "47863826"}},
		}

		for alg, v := range vectors {
			opts := &Options{Algorithm: alg, Digits: 8}

			for i, ts := range times {
				code, err := TOTP(v.secret, time.Unix(ts, 0), opts)
				require.NoError(t, err)
				assert.Equal(t, v.codes[i], code, "%s at t=%d", alg, ts)
			}
		}
	})

	t.Run("pre-epoch time maps to counter zero", func(t *testing.T) {
		code, err := TOTP(rfcSecretSHA1, time.Unix(-1000, 0), nil)
		require.NoError(t, err)

		atZero, err := HOTP(rfcSecretSHA1, 0, nil)
		require.NoError(t, err)
		assert.Equal(t, atZero, code)
	})

	t.Run("custom T0 shifts the counter", func(t *testing.T) {
		t0 := time.Unix(1000000000, 0)
		opts := &Options{Digits: 8, T0: t0}

		// With T0 = t0, the code at t0+59s must equal the RFC vector at t=59.
		code, err := TOTP(rfcSecretSHA1, t0.Add(59*time.Second), opts)
		require.NoError(t, err)
		assert.Equal(t, "94287082", code)

		// Times before T0 map to counter 0.
		before, err := TOTP(rfcSecretSHA1, t0.Add(-time.Hour), opts)
		require.NoError(t, err)

		atZero, err := TOTP(rfcSecretSHA1, t0, opts)
		require.NoError(t, err)
		assert.Equal(t, atZero, before)
	})

	t.Run("invalid secret", func(t *testing.T) {
		_, err := TOTP("", time.Now(), nil)
		assert.ErrorIs(t, err, ErrInvalidSecret)
	})
}

func TestVerifyTOTP(t *testing.T) {
	now := time.Unix(1234567890, 0)

	t.Run("current and adjacent periods within default skew", func(t *testing.T) {
		for _, offset := range []time.Duration{0, -30 * time.Second, 30 * time.Second} {
			code, err := TOTP(rfcSecretSHA1, now.Add(offset), nil)
			require.NoError(t, err)

			ok, err := VerifyTOTP(rfcSecretSHA1, code, now, nil)
			require.NoError(t, err)
			assert.True(t, ok, "offset %v", offset)
		}
	})

	t.Run("outside skew window rejected", func(t *testing.T) {
		code, err := TOTP(rfcSecretSHA1, now.Add(2*30*time.Second), nil)
		require.NoError(t, err)

		ok, err := VerifyTOTP(rfcSecretSHA1, code, now, nil)
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("zero skew is strict", func(t *testing.T) {
		opts := &Options{Skew: 0}

		code, err := TOTP(rfcSecretSHA1, now.Add(-30*time.Second), opts)
		require.NoError(t, err)

		ok, err := VerifyTOTP(rfcSecretSHA1, code, now, opts)
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("skew window near epoch does not underflow", func(t *testing.T) {
		code, err := TOTP(rfcSecretSHA1, time.Unix(0, 0), nil)
		require.NoError(t, err)

		ok, err := VerifyTOTP(rfcSecretSHA1, code, time.Unix(0, 0), nil)
		require.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("asymmetric skew accepts behind only", func(t *testing.T) {
		opts := &Options{SkewBehind: 1}

		behind, err := TOTP(rfcSecretSHA1, now.Add(-30*time.Second), opts)
		require.NoError(t, err)

		ok, err := VerifyTOTP(rfcSecretSHA1, behind, now, opts)
		require.NoError(t, err)
		assert.True(t, ok)

		ahead, err := TOTP(rfcSecretSHA1, now.Add(30*time.Second), opts)
		require.NoError(t, err)

		ok, err = VerifyTOTP(rfcSecretSHA1, ahead, now, opts)
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("asymmetric skew clamped", func(t *testing.T) {
		o := (&Options{SkewBehind: 1 << 40, SkewAhead: 3}).withDefaults()
		assert.Equal(t, uint64(maxSkew), o.SkewBehind)
		assert.Equal(t, uint64(3), o.SkewAhead)
	})

	t.Run("wrong code rejected", func(t *testing.T) {
		ok, err := VerifyTOTP(rfcSecretSHA1, "00000000", now, nil)
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("invalid secret", func(t *testing.T) {
		_, err := VerifyTOTP("====", "123456", now, nil)
		assert.ErrorIs(t, err, ErrInvalidSecret)
	})
}

func TestVerifyTOTPCounter(t *testing.T) {
	now := time.Unix(1234567890, 0)

	t.Run("returns matched counter for replay protection", func(t *testing.T) {
		code, err := TOTP(rfcSecretSHA1, now, nil)
		require.NoError(t, err)

		counter, ok, err := VerifyTOTPCounter(rfcSecretSHA1, code, now, nil)
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, uint64(1234567890/30), counter)

		// Replay guard: the same code matches the same counter, so a
		// caller persisting the watermark rejects the second use.
		again, ok, err := VerifyTOTPCounter(rfcSecretSHA1, code, now.Add(10*time.Second), nil)
		require.NoError(t, err)
		assert.True(t, ok)
		assert.False(t, again > counter)
	})

	t.Run("counter reflects the period the code came from", func(t *testing.T) {
		behind, err := TOTP(rfcSecretSHA1, now.Add(-30*time.Second), nil)
		require.NoError(t, err)

		counter, ok, err := VerifyTOTPCounter(rfcSecretSHA1, behind, now, nil)
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, uint64(1234567890/30-1), counter)
	})

	t.Run("no match", func(t *testing.T) {
		_, ok, err := VerifyTOTPCounter(rfcSecretSHA1, "000000", now, nil)
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("invalid secret", func(t *testing.T) {
		_, _, err := VerifyTOTPCounter("!", "123456", now, nil)
		assert.ErrorIs(t, err, ErrInvalidSecret)
	})
}

func TestTimeRemaining(t *testing.T) {
	t.Run("counts down within the period", func(t *testing.T) {
		assert.Equal(t, time.Unix(60, 0), ExpiresAt(time.Unix(59, 0), nil))
		assert.Equal(t, time.Second, TimeRemaining(time.Unix(59, 0), nil))
		assert.Equal(t, 30*time.Second, TimeRemaining(time.Unix(30, 0), nil))
		assert.Equal(t, 500*time.Millisecond, TimeRemaining(time.Unix(59, int64(500*time.Millisecond)), nil))
	})

	t.Run("respects period and T0", func(t *testing.T) {
		t0 := time.Unix(1000, 0)
		opts := &Options{Period: 60, T0: t0}

		assert.Equal(t, t0.Add(time.Minute), ExpiresAt(t0.Add(59*time.Second), opts))
		assert.Equal(t, time.Second, TimeRemaining(t0.Add(59*time.Second), opts))
	})
}

func FuzzVerifyTOTP(f *testing.F) {
	f.Add([]byte("12345678901234567890"), int64(59), uint64(30), uint64(1), 0)
	f.Add([]byte{1, 2, 3}, int64(-100), uint64(0), uint64(1<<50), 2)
	f.Add([]byte("k"), int64(1<<62), uint64(1), uint64(0), 99)

	f.Fuzz(func(t *testing.T, rawSecret []byte, unix int64, period, skew uint64, alg int) {
		if len(rawSecret) == 0 {
			return
		}

		secret := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(rawSecret)
		opts := &Options{Algorithm: Algorithm(alg), Period: period, Skew: skew}
		at := time.Unix(unix, 0)

		code, err := TOTP(secret, at, opts)
		if err != nil {
			t.Fatalf("TOTP failed on valid secret: %v", err)
		}

		ok, err := VerifyTOTP(secret, code, at, opts)
		if err != nil || !ok {
			t.Fatalf("code generated at t must verify at t: ok=%v err=%v", ok, err)
		}
	})
}

func BenchmarkTOTP(b *testing.B) {
	now := time.Now()

	b.Run("generate", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if _, err := TOTP(rfcSecretSHA1, now, nil); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("verify", func(b *testing.B) {
		code, err := TOTP(rfcSecretSHA1, now, nil)
		if err != nil {
			b.Fatal(err)
		}

		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if _, err := VerifyTOTP(rfcSecretSHA1, code, now, nil); err != nil {
				b.Fatal(err)
			}
		}
	})
}
