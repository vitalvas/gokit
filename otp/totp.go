package otp

import "time"

// TOTP returns the RFC 6238 code for the given time.
func TOTP(secret string, t time.Time, opts *Options) (string, error) {
	o := opts.withDefaults()

	key, err := decodeSecret(secret)
	if err != nil {
		return "", err
	}

	return hotp(key, timeCounter(t, o.Period, o.T0), o), nil
}

// VerifyTOTP reports whether code is valid at time t, accepting codes from
// up to SkewBehind periods before and SkewAhead periods after the current
// one to tolerate clock drift. The comparison is constant-time and always
// checks the full window.
func VerifyTOTP(secret, code string, t time.Time, opts *Options) (bool, error) {
	_, ok, err := VerifyTOTPCounter(secret, code, t, opts)

	return ok, err
}

// VerifyTOTPCounter is VerifyTOTP returning the time-step counter the code
// matched. Persisting the counter of the last accepted code enables replay
// protection: reject any code whose counter is not greater than it.
//
//	counter, ok, err := otp.VerifyTOTPCounter(secret, code, time.Now(), nil)
//	if ok && counter > lastAccepted {
//		lastAccepted = counter // code accepted exactly once
//	}
func VerifyTOTPCounter(secret, code string, t time.Time, opts *Options) (uint64, bool, error) {
	o := opts.withDefaults()

	key, err := decodeSecret(secret)
	if err != nil {
		return 0, false, err
	}

	center := timeCounter(t, o.Period, o.T0)

	var matched uint64
	found := false

	for c := center - min(center, o.SkewBehind); c <= center+o.SkewAhead; c++ {
		if equal(hotp(key, c, o), code) && !found {
			matched, found = c, true
		}
	}

	return matched, found, nil
}

// ExpiresAt returns the end of the TOTP time step containing t, i.e., the
// moment the code valid at t stops being the current code.
func ExpiresAt(t time.Time, opts *Options) time.Time {
	o := opts.withDefaults()

	start := int64(0)
	if !o.T0.IsZero() {
		start = o.T0.Unix()
	}

	next := timeCounter(t, o.Period, o.T0) + 1

	return time.Unix(start+int64(next*o.Period), 0)
}

// TimeRemaining returns how long the TOTP code at time t remains the
// current code. Useful for countdown displays.
func TimeRemaining(t time.Time, opts *Options) time.Duration {
	return ExpiresAt(t, opts).Sub(t)
}

// timeCounter returns the time-step counter for t: floor((t - t0) / period).
// A zero t0 means the Unix epoch. Times before t0 map to counter 0.
func timeCounter(t time.Time, period uint64, t0 time.Time) uint64 {
	ts := t.Unix()
	if !t0.IsZero() {
		ts -= t0.Unix()
	}

	if ts < 0 {
		return 0
	}

	return uint64(ts) / period
}
