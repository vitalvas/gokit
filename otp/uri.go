package otp

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// ErrInvalidURI is returned by ParseKeyURI for malformed otpauth URIs.
var ErrInvalidURI = errors.New("invalid otpauth URI")

// Key types as they appear in otpauth:// URIs.
const (
	TypeTOTP = "totp"
	TypeHOTP = "hotp"
)

// Key is a parsed otpauth:// provisioning URI.
type Key struct {
	// Type is TypeTOTP or TypeHOTP.
	Type string

	// Issuer and Account identify the key. The issuer query parameter
	// takes precedence over the label prefix when both are present.
	Issuer  string
	Account string

	// Secret is the base32 shared secret.
	Secret string

	// Counter is the initial counter for hotp keys.
	Counter uint64

	// Options carries the algorithm, digits, and period from the URI,
	// ready to pass to the generate and verify functions.
	Options Options
}

// KeyURI returns the otpauth://totp provisioning URI for the secret, as
// consumed by authenticator apps (typically rendered as a QR code).
func KeyURI(issuer, account, secret string, opts *Options) string {
	k := Key{
		Type:    TypeTOTP,
		Issuer:  issuer,
		Account: account,
		Secret:  secret,
		Options: opts.withDefaults(),
	}

	return k.URI()
}

// HOTPKeyURI returns the otpauth://hotp provisioning URI for the secret,
// with the initial counter value the token starts from.
func HOTPKeyURI(issuer, account, secret string, counter uint64, opts *Options) string {
	k := Key{
		Type:    TypeHOTP,
		Issuer:  issuer,
		Account: account,
		Secret:  secret,
		Counter: counter,
		Options: opts.withDefaults(),
	}

	return k.URI()
}

// ParseKeyURI parses an otpauth://totp or otpauth://hotp provisioning URI,
// the format produced by KeyURI and HOTPKeyURI and used in enrollment QR
// codes. It validates the secret, algorithm, digits, period, and counter.
func ParseKeyURI(raw string) (*Key, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "otpauth" || (u.Host != TypeTOTP && u.Host != TypeHOTP) {
		return nil, fmt.Errorf("%w: %q", ErrInvalidURI, raw)
	}

	k := &Key{Type: u.Host}

	label := strings.TrimPrefix(u.Path, "/")
	if issuer, account, ok := strings.Cut(label, ":"); ok {
		k.Issuer = issuer
		k.Account = strings.TrimLeft(account, " ")
	} else {
		k.Account = label
	}

	q := u.Query()

	k.Secret = q.Get("secret")
	if _, err := decodeSecret(k.Secret); err != nil {
		return nil, fmt.Errorf("%w: bad secret", ErrInvalidURI)
	}

	if issuer := q.Get("issuer"); issuer != "" {
		k.Issuer = issuer
	}

	switch strings.ToUpper(q.Get("algorithm")) {
	case "", "SHA1":
		k.Options.Algorithm = SHA1
	case "SHA256":
		k.Options.Algorithm = SHA256
	case "SHA512":
		k.Options.Algorithm = SHA512
	default:
		return nil, fmt.Errorf("%w: algorithm %q", ErrInvalidURI, q.Get("algorithm"))
	}

	k.Options.Digits = DefaultDigits
	if d := q.Get("digits"); d != "" {
		n, err := strconv.Atoi(d)
		if err != nil || n < minDigits || n > maxDigits {
			return nil, fmt.Errorf("%w: digits %q", ErrInvalidURI, d)
		}
		k.Options.Digits = n
	}

	switch k.Type {
	case TypeTOTP:
		k.Options.Period = DefaultPeriod
		if p := q.Get("period"); p != "" {
			n, err := strconv.ParseUint(p, 10, 64)
			if err != nil || n == 0 {
				return nil, fmt.Errorf("%w: period %q", ErrInvalidURI, p)
			}
			k.Options.Period = n
		}
	default: // hotp
		c, err := strconv.ParseUint(q.Get("counter"), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("%w: counter %q", ErrInvalidURI, q.Get("counter"))
		}
		k.Counter = c
	}

	return k, nil
}

// URI returns the otpauth:// provisioning URI for the key. A Type other
// than TypeHOTP is treated as TypeTOTP.
func (k *Key) URI() string {
	typ := k.Type
	if typ != TypeHOTP {
		typ = TypeTOTP
	}

	q := url.Values{
		"secret":    {k.Secret},
		"issuer":    {k.Issuer},
		"algorithm": {k.Options.Algorithm.String()},
		"digits":    {strconv.Itoa(k.Options.Digits)},
	}

	if typ == TypeHOTP {
		q.Set("counter", strconv.FormatUint(k.Counter, 10))
	} else {
		q.Set("period", strconv.FormatUint(k.Options.Period, 10))
	}

	u := url.URL{
		Scheme:   "otpauth",
		Host:     typ,
		Path:     fmt.Sprintf("/%s:%s", k.Issuer, k.Account),
		RawQuery: q.Encode(),
	}

	return u.String()
}
