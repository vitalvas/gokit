package otp

import (
	"crypto/hmac"
	"crypto/subtle"
	"encoding/binary"
	"fmt"
)

// HOTP returns the RFC 4226 code for the given counter.
func HOTP(secret string, counter uint64, opts *Options) (string, error) {
	o := opts.withDefaults()

	key, err := decodeSecret(secret)
	if err != nil {
		return "", err
	}

	return hotp(key, counter, o), nil
}

// VerifyHOTP reports whether code matches the counter value.
// The comparison is constant-time.
func VerifyHOTP(secret, code string, counter uint64, opts *Options) (bool, error) {
	o := opts.withDefaults()

	key, err := decodeSecret(secret)
	if err != nil {
		return false, err
	}

	return equal(hotp(key, counter, o), code), nil
}

// SyncHOTP implements the RFC 4226 section 7.4 look-ahead window for
// counter resynchronization: it checks code against the counters in
// [counter, counter+window] (window clamped to 1000) and returns the
// counter that matched. On success the caller should persist the returned
// counter plus one as the next expected value. The comparison is
// constant-time and always scans the full window.
func SyncHOTP(secret, code string, counter, window uint64, opts *Options) (uint64, bool, error) {
	o := opts.withDefaults()

	key, err := decodeSecret(secret)
	if err != nil {
		return 0, false, err
	}

	if window > maxWindow {
		window = maxWindow
	}

	var matched uint64
	found := false

	for i := uint64(0); i <= window; i++ {
		c := counter + i
		if c < counter {
			break // counter space exhausted
		}

		if equal(hotp(key, c, o), code) && !found {
			matched, found = c, true
		}
	}

	return matched, found, nil
}

// hotp computes HMAC(key, counter) and truncates it per RFC 4226.
func hotp(key []byte, counter uint64, o Options) string {
	mac := hmac.New(o.Algorithm.hash(), key)

	var msg [8]byte
	binary.BigEndian.PutUint64(msg[:], counter)
	mac.Write(msg[:])

	return truncate(mac.Sum(nil), o.Digits)
}

// truncate implements the RFC 4226 dynamic truncation of an HMAC value to
// a decimal code of the given length.
func truncate(sum []byte, digits int) string {
	offset := sum[len(sum)-1] & 0x0f
	code := binary.BigEndian.Uint32(sum[offset:offset+4]) & 0x7fffffff

	return fmt.Sprintf("%0*d", digits, uint64(code)%pow10(digits))
}

func equal(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func pow10(n int) uint64 {
	p := uint64(1)
	for range n {
		p *= 10
	}

	return p
}
