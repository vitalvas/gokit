package otp

import (
	"encoding/base32"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// RFC 6287 Appendix C: PIN (1234) SHA1 hash and the T=132d0b6 timestamp
// (number of minutes since epoch).
var (
	ocraPinHash, _ = hex.DecodeString("7110eda4d09e062aa5e4a390b0a572ac0d2c0220")
	ocraTime       = time.Unix(0x132d0b6*60, 0)
)

func TestParseOCRASuite(t *testing.T) {
	t.Run("full suite", func(t *testing.T) {
		s, err := ParseOCRASuite("OCRA-1:HOTP-SHA256-8:C-QN08-PSHA1-S064-T1M")
		require.NoError(t, err)

		assert.Equal(t, SHA256, s.Hash)
		assert.Equal(t, 8, s.Digits)
		assert.True(t, s.Counter)
		assert.Equal(t, byte('N'), s.QuestionFormat)
		assert.Equal(t, 8, s.QuestionLength)
		assert.True(t, s.HasPassword)
		assert.Equal(t, SHA1, s.PasswordHash)
		assert.Equal(t, 64, s.SessionLength)
		assert.Equal(t, time.Minute, s.TimeStep)
		assert.Equal(t, "OCRA-1:HOTP-SHA256-8:C-QN08-PSHA1-S064-T1M", s.String())
	})

	t.Run("minimal suite", func(t *testing.T) {
		s, err := ParseOCRASuite("OCRA-1:HOTP-SHA1-6:QN08")
		require.NoError(t, err)

		assert.Equal(t, SHA1, s.Hash)
		assert.Equal(t, 6, s.Digits)
		assert.False(t, s.Counter)
		assert.False(t, s.HasPassword)
		assert.Zero(t, s.SessionLength)
		assert.Zero(t, s.TimeStep)
	})

	t.Run("time step units", func(t *testing.T) {
		for suite, want := range map[string]time.Duration{
			"OCRA-1:HOTP-SHA1-6:QN08-T20S": 20 * time.Second,
			"OCRA-1:HOTP-SHA1-6:QN08-T5M":  5 * time.Minute,
			"OCRA-1:HOTP-SHA1-6:QN08-T24H": 24 * time.Hour,
		} {
			s, err := ParseOCRASuite(suite)
			require.NoError(t, err, suite)
			assert.Equal(t, want, s.TimeStep, suite)
		}
	})

	t.Run("no truncation", func(t *testing.T) {
		s, err := ParseOCRASuite("OCRA-1:HOTP-SHA1-0:QN08")
		require.NoError(t, err)
		assert.Zero(t, s.Digits)
	})

	t.Run("invalid suites", func(t *testing.T) {
		for _, bad := range []string{
			"",
			"OCRA-1:HOTP-SHA1-6",
			"OCRA-2:HOTP-SHA1-6:QN08",
			"OCRA-1:TOTP-SHA1-6:QN08",
			"OCRA-1:HOTP-MD5-6:QN08",
			"OCRA-1:HOTP-SHA1-3:QN08",
			"OCRA-1:HOTP-SHA1-11:QN08",
			"OCRA-1:HOTP-SHA1-x:QN08",
			"OCRA-1:HOTP-SHA1-6:C",
			"OCRA-1:HOTP-SHA1-6:QX08",
			"OCRA-1:HOTP-SHA1-6:QN03",
			"OCRA-1:HOTP-SHA1-6:QN65",
			"OCRA-1:HOTP-SHA1-6:QN08-PMD5",
			"OCRA-1:HOTP-SHA1-6:QN08-S64",
			"OCRA-1:HOTP-SHA1-6:QN08-S000",
			"OCRA-1:HOTP-SHA1-6:QN08-T0S",
			"OCRA-1:HOTP-SHA1-6:QN08-T60M",
			"OCRA-1:HOTP-SHA1-6:QN08-T49H",
			"OCRA-1:HOTP-SHA1-6:QN08-T1M-S064",
			"OCRA-1:HOTP-SHA1-6:QN08-X",
			"OCRA-1:HOTP-SHA1-6:PSHA1-QN08",
		} {
			_, err := ParseOCRASuite(bad)
			assert.ErrorIs(t, err, ErrInvalidSuite, bad)
		}
	})
}

func TestOCRA(t *testing.T) {
	t.Run("C.1 one-way SHA1 QN08", func(t *testing.T) {
		codes := []string{
			"237653", "243178", "653583", "740991", "608993",
			"388898", "816933", "224598", "750600", "294470",
		}

		for i, want := range codes {
			q := strings.Repeat(fmt.Sprintf("%d", i), 8)

			code, err := OCRA("OCRA-1:HOTP-SHA1-6:QN08", rfcSecretSHA1, OCRAInput{Question: q})
			require.NoError(t, err)
			assert.Equal(t, want, code, "Q=%s", q)
		}
	})

	t.Run("C.1 counter and PIN SHA256", func(t *testing.T) {
		codes := []string{
			"65347737", "86775851", "78192410", "71565254", "10104329",
			"65983500", "70069104", "91771096", "75011558", "08522129",
		}

		for i, want := range codes {
			in := OCRAInput{Counter: uint64(i), Question: "12345678", Password: ocraPinHash}

			code, err := OCRA("OCRA-1:HOTP-SHA256-8:C-QN08-PSHA1", rfcSecretSHA256, in)
			require.NoError(t, err)
			assert.Equal(t, want, code, "C=%d", i)
		}
	})

	t.Run("C.1 PIN without counter SHA256", func(t *testing.T) {
		codes := []string{"83238735", "01501458", "17957585", "86776967", "86807031"}

		for i, want := range codes {
			in := OCRAInput{Question: strings.Repeat(fmt.Sprintf("%d", i), 8), Password: ocraPinHash}

			code, err := OCRA("OCRA-1:HOTP-SHA256-8:QN08-PSHA1", rfcSecretSHA256, in)
			require.NoError(t, err)
			assert.Equal(t, want, code)
		}
	})

	t.Run("C.1 counter SHA512", func(t *testing.T) {
		codes := []string{
			"07016083", "63947962", "70123924", "25341727", "33203315",
			"34205738", "44343969", "51946085", "20403879", "31409299",
		}

		for i, want := range codes {
			in := OCRAInput{Counter: uint64(i), Question: strings.Repeat(fmt.Sprintf("%d", i), 8)}

			code, err := OCRA("OCRA-1:HOTP-SHA512-8:C-QN08", rfcSecretSHA512, in)
			require.NoError(t, err)
			assert.Equal(t, want, code)
		}
	})

	t.Run("C.1 timestamp SHA512", func(t *testing.T) {
		codes := []string{"95209754", "55907591", "22048402", "24218844", "36209546"}

		for i, want := range codes {
			in := OCRAInput{Question: strings.Repeat(fmt.Sprintf("%d", i), 8), Timestamp: ocraTime}

			code, err := OCRA("OCRA-1:HOTP-SHA512-8:QN08-T1M", rfcSecretSHA512, in)
			require.NoError(t, err)
			assert.Equal(t, want, code)
		}
	})

	t.Run("C.2 mutual SHA256", func(t *testing.T) {
		server := []string{"28247970", "01984843", "65387857", "03351211", "83412541"}
		client := []string{"15510767", "90175646", "33777207", "95285278", "28934924"}

		for i := range server {
			in := OCRAInput{Question: fmt.Sprintf("CLI2222%dSRV1111%d", i, i)}
			code, err := OCRA("OCRA-1:HOTP-SHA256-8:QA08", rfcSecretSHA256, in)
			require.NoError(t, err)
			assert.Equal(t, server[i], code, "server %d", i)

			in = OCRAInput{Question: fmt.Sprintf("SRV1111%dCLI2222%d", i, i)}
			code, err = OCRA("OCRA-1:HOTP-SHA256-8:QA08", rfcSecretSHA256, in)
			require.NoError(t, err)
			assert.Equal(t, client[i], code, "client %d", i)
		}
	})

	t.Run("C.2 mutual SHA512 with PIN", func(t *testing.T) {
		server := []string{"79496648", "76831980", "12250499", "90856481", "12761449"}
		client := []string{"18806276", "70020315", "01600026", "18951020", "32528969"}

		for i := range server {
			in := OCRAInput{Question: fmt.Sprintf("CLI2222%dSRV1111%d", i, i)}
			code, err := OCRA("OCRA-1:HOTP-SHA512-8:QA08", rfcSecretSHA512, in)
			require.NoError(t, err)
			assert.Equal(t, server[i], code, "server %d", i)

			in = OCRAInput{Question: fmt.Sprintf("SRV1111%dCLI2222%d", i, i), Password: ocraPinHash}
			code, err = OCRA("OCRA-1:HOTP-SHA512-8:QA08-PSHA1", rfcSecretSHA512, in)
			require.NoError(t, err)
			assert.Equal(t, client[i], code, "client %d", i)
		}
	})

	t.Run("C.3 plain signature", func(t *testing.T) {
		codes := []string{"53095496", "04110475", "31331128", "76028668", "46554205"}

		for i, want := range codes {
			in := OCRAInput{Question: fmt.Sprintf("SIG1%d000", i)}

			code, err := OCRA("OCRA-1:HOTP-SHA256-8:QA08", rfcSecretSHA256, in)
			require.NoError(t, err)
			assert.Equal(t, want, code)
		}
	})

	t.Run("C.3 signature with timestamp", func(t *testing.T) {
		codes := []string{"77537423", "31970405", "10235557", "95213541", "65360607"}

		for i, want := range codes {
			in := OCRAInput{Question: fmt.Sprintf("SIG1%d00000", i), Timestamp: ocraTime}

			code, err := OCRA("OCRA-1:HOTP-SHA512-8:QA10-T1M", rfcSecretSHA512, in)
			require.NoError(t, err)
			assert.Equal(t, want, code)
		}
	})

	t.Run("hex challenge", func(t *testing.T) {
		code, err := OCRA("OCRA-1:HOTP-SHA1-6:QH08", rfcSecretSHA1, OCRAInput{Question: "ABC1234"})
		require.NoError(t, err)
		assert.Len(t, code, 6)
	})

	t.Run("no truncation returns full HMAC hex", func(t *testing.T) {
		code, err := OCRA("OCRA-1:HOTP-SHA1-0:QN08", rfcSecretSHA1, OCRAInput{Question: "12345678"})
		require.NoError(t, err)
		assert.Len(t, code, 40) // SHA1 = 20 bytes = 40 hex chars

		_, err = hex.DecodeString(code)
		assert.NoError(t, err)
	})

	t.Run("session is left-padded", func(t *testing.T) {
		short, err := OCRA("OCRA-1:HOTP-SHA1-6:QN08-S064", rfcSecretSHA1,
			OCRAInput{Question: "12345678", Session: []byte("abc")})
		require.NoError(t, err)

		padded := make([]byte, 64)
		copy(padded[61:], "abc")
		full, err := OCRA("OCRA-1:HOTP-SHA1-6:QN08-S064", rfcSecretSHA1,
			OCRAInput{Question: "12345678", Session: padded})
		require.NoError(t, err)

		assert.Equal(t, full, short)
	})

	t.Run("invalid secret", func(t *testing.T) {
		_, err := OCRA("OCRA-1:HOTP-SHA1-6:QN08", "!", OCRAInput{Question: "12345678"})
		assert.ErrorIs(t, err, ErrInvalidSecret)
	})
}

func TestOCRAInputValidation(t *testing.T) {
	const suite = "OCRA-1:HOTP-SHA1-6:QN08"

	t.Run("question errors", func(t *testing.T) {
		for name, in := range map[string]OCRAInput{
			"empty":              {},
			"too long":           {Question: strings.Repeat("1", 129)},
			"non-numeric for QN": {Question: "abc12345"},
		} {
			_, err := OCRA(suite, rfcSecretSHA1, in)
			assert.ErrorIs(t, err, ErrInvalidInput, name)
		}

		_, err := OCRA("OCRA-1:HOTP-SHA1-6:QA08", rfcSecretSHA1, OCRAInput{Question: "no spaces"})
		assert.ErrorIs(t, err, ErrInvalidInput)

		_, err = OCRA("OCRA-1:HOTP-SHA1-6:QH08", rfcSecretSHA1, OCRAInput{Question: "XYZ"})
		assert.ErrorIs(t, err, ErrInvalidInput)
	})

	t.Run("password errors", func(t *testing.T) {
		_, err := OCRA("OCRA-1:HOTP-SHA1-6:QN08-PSHA1", rfcSecretSHA1,
			OCRAInput{Question: "12345678", Password: []byte("short")})
		assert.ErrorIs(t, err, ErrInvalidInput)

		_, err = OCRA(suite, rfcSecretSHA1, OCRAInput{Question: "12345678", Password: ocraPinHash})
		assert.ErrorIs(t, err, ErrInvalidInput)
	})

	t.Run("session errors", func(t *testing.T) {
		_, err := OCRA("OCRA-1:HOTP-SHA1-6:QN08-S064", rfcSecretSHA1,
			OCRAInput{Question: "12345678", Session: make([]byte, 65)})
		assert.ErrorIs(t, err, ErrInvalidInput)

		_, err = OCRA(suite, rfcSecretSHA1, OCRAInput{Question: "12345678", Session: []byte("x")})
		assert.ErrorIs(t, err, ErrInvalidInput)
	})
}

func TestVerifyOCRA(t *testing.T) {
	const suite = "OCRA-1:HOTP-SHA1-6:QN08"

	t.Run("matching code", func(t *testing.T) {
		ok, err := VerifyOCRA(suite, rfcSecretSHA1, "237653", OCRAInput{Question: "00000000"})
		require.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("wrong code", func(t *testing.T) {
		ok, err := VerifyOCRA(suite, rfcSecretSHA1, "000000", OCRAInput{Question: "00000000"})
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("invalid suite", func(t *testing.T) {
		_, err := VerifyOCRA("OCRA-9", rfcSecretSHA1, "237653", OCRAInput{Question: "00000000"})
		assert.ErrorIs(t, err, ErrInvalidSuite)
	})
}

func FuzzParseOCRASuite(f *testing.F) {
	f.Add("OCRA-1:HOTP-SHA1-6:QN08")
	f.Add("OCRA-1:HOTP-SHA256-8:C-QN08-PSHA1-S064-T1M")
	f.Add("OCRA-1:HOTP-SHA512-0:QA64-T48H")
	f.Add(":::")
	f.Add("OCRA-1:HOTP-SHA1-6:QN08-")

	f.Fuzz(func(t *testing.T, s string) {
		parsed, err := ParseOCRASuite(s)
		if err != nil {
			return
		}

		if parsed.String() != s {
			t.Fatalf("round trip mismatch: %q != %q", parsed.String(), s)
		}

		if parsed.QuestionFormat != 'A' && parsed.QuestionFormat != 'N' && parsed.QuestionFormat != 'H' {
			t.Fatalf("invalid question format %q accepted", parsed.QuestionFormat)
		}
	})
}

func FuzzOCRA(f *testing.F) {
	f.Add([]byte("12345678901234567890"), uint64(0), "00000000", int64(59), 0)
	f.Add([]byte{0xff}, uint64(1<<40), "99999999", int64(-5), 1)
	f.Add([]byte("key"), uint64(7), "12345678", int64(1<<62), 2)

	suites := []string{
		"OCRA-1:HOTP-SHA1-6:QN08",
		"OCRA-1:HOTP-SHA256-8:C-QN08",
		"OCRA-1:HOTP-SHA512-10:C-QN08-T1M",
	}

	f.Fuzz(func(t *testing.T, rawSecret []byte, counter uint64, question string, unix int64, pick int) {
		if len(rawSecret) == 0 {
			return
		}

		secret := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(rawSecret)
		suite := suites[((pick%len(suites))+len(suites))%len(suites)]
		in := OCRAInput{Counter: counter, Question: question, Timestamp: time.Unix(unix, 0)}

		code, err := OCRA(suite, secret, in)
		if err != nil {
			return // invalid question for the format
		}

		parsed, err := ParseOCRASuite(suite)
		if err != nil || len(code) != parsed.Digits {
			t.Fatalf("code %q has wrong length for %s", code, suite)
		}

		ok, err := VerifyOCRA(suite, secret, code, in)
		if err != nil || !ok {
			t.Fatalf("generated code must verify: ok=%v err=%v", ok, err)
		}
	})
}

func BenchmarkOCRA(b *testing.B) {
	in := OCRAInput{Question: "00000000"}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := OCRA("OCRA-1:HOTP-SHA1-6:QN08", rfcSecretSHA1, in); err != nil {
			b.Fatal(err)
		}
	}
}
