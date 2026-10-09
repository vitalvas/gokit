// Package otp implements HOTP (RFC 4226) and TOTP (RFC 6238) one-time
// passwords using only the standard library, including secret generation
// and otpauth:// provisioning URIs for authenticator apps.
package otp

import (
	"errors"
	"time"
)

// ErrInvalidSecret is returned when a secret is not valid base32.
var ErrInvalidSecret = errors.New("invalid base32 secret")

// Defaults used when an Options field is unset.
const (
	DefaultPeriod = 30
	DefaultDigits = 6
	DefaultSkew   = 1

	minDigits = 6
	maxDigits = 10 // dynamic truncation yields 31 bits, so 10 digits is the ceiling
	maxSkew   = 10
	maxWindow = 1000 // SyncHOTP look-ahead ceiling
	minSecret = 20   // RFC 4226 recommended secret length in bytes
)

// Options configures code generation and verification. A nil *Options means
// all defaults: SHA1, 6 digits, 30-second period, skew of 1 period.
type Options struct {
	// Algorithm is the HMAC hash. Default SHA1.
	Algorithm Algorithm

	// Period is the TOTP time step in seconds. Default 30. Ignored by HOTP.
	Period uint64

	// Digits is the code length, clamped to [6, 10]. Default 6.
	// Note that most authenticator apps only support 6 or 8.
	Digits int

	// Skew is how many periods either side of the current one VerifyTOTP
	// accepts, clamped to at most 10. A non-nil Options with Skew 0 means
	// strict single-period matching. Ignored by HOTP.
	Skew uint64

	// SkewBehind and SkewAhead set an asymmetric verification window and
	// take precedence over Skew when either is non-zero. SkewBehind accepts
	// codes from past periods (user typed the code late), SkewAhead from
	// future periods (client clock runs fast). Both are clamped to at
	// most 10. Ignored by HOTP.
	SkewBehind uint64
	SkewAhead  uint64

	// T0 is the time from which TOTP time steps are counted (RFC 6238 T0).
	// The zero value means the Unix epoch. Ignored by HOTP.
	T0 time.Time
}

func (o *Options) withDefaults() Options {
	out := Options{
		Period:     DefaultPeriod,
		Digits:     DefaultDigits,
		Skew:       DefaultSkew,
		SkewBehind: DefaultSkew,
		SkewAhead:  DefaultSkew,
	}

	if o == nil {
		return out
	}

	out.Algorithm = o.Algorithm
	out.Skew = o.Skew
	out.T0 = o.T0

	if o.Period > 0 {
		out.Period = o.Period
	}

	if o.Digits != 0 {
		out.Digits = o.Digits
	}

	if out.Digits < minDigits {
		out.Digits = minDigits
	}

	if out.Digits > maxDigits {
		out.Digits = maxDigits
	}

	if out.Skew > maxSkew {
		out.Skew = maxSkew
	}

	// Resolve the verification window into SkewBehind/SkewAhead, which is
	// what VerifyTOTP uses; Skew is the symmetric shorthand.
	if o.SkewBehind != 0 || o.SkewAhead != 0 {
		out.SkewBehind = min(o.SkewBehind, maxSkew)
		out.SkewAhead = min(o.SkewAhead, maxSkew)
	} else {
		out.SkewBehind = out.Skew
		out.SkewAhead = out.Skew
	}

	return out
}
