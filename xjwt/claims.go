package xjwt

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"time"
)

// NumericDate is an RFC 7519 numeric date: seconds since the Unix epoch,
// marshalled as an integer.
type NumericDate struct {
	time.Time
}

// NewNumericDate wraps a time.Time as a NumericDate (second precision).
func NewNumericDate(t time.Time) *NumericDate {
	if t.IsZero() {
		return nil
	}

	return &NumericDate{t.Truncate(time.Second)}
}

// MarshalJSON renders the date as integer seconds since the epoch.
func (d NumericDate) MarshalJSON() ([]byte, error) {
	return []byte(strconv.FormatInt(d.Unix(), 10)), nil
}

// UnmarshalJSON accepts integer or floating-point epoch seconds.
func (d *NumericDate) UnmarshalJSON(b []byte) error {
	var f float64
	if err := json.Unmarshal(b, &f); err != nil {
		return err
	}

	t, ok := numericDateFromFloat(f)
	if !ok {
		return fmt.Errorf("xjwt: numeric date out of range")
	}

	d.Time = t

	return nil
}

func numericDateFromFloat(f float64) (time.Time, bool) {
	const (
		minInt64Float          = -9223372036854775808.0
		maxInt64ExclusiveFloat = 9223372036854775808.0
	)

	if math.IsInf(f, 0) || math.IsNaN(f) || f < minInt64Float || f >= maxInt64ExclusiveFloat {
		return time.Time{}, false
	}

	return time.Unix(int64(f), 0), true
}

// ClaimStrings is a JWT claim that is either a single string or an array of
// strings (e.g. aud).
type ClaimStrings []string

// MarshalJSON always renders the value as a JSON array, so a single-element
// audience is emitted as an array for a stable wire format.
func (s ClaimStrings) MarshalJSON() ([]byte, error) {
	return json.Marshal([]string(s))
}

// UnmarshalJSON accepts a string or an array of strings.
func (s *ClaimStrings) UnmarshalJSON(b []byte) error {
	var single string
	if err := json.Unmarshal(b, &single); err == nil {
		*s = ClaimStrings{single}

		return nil
	}

	var many []string
	if err := json.Unmarshal(b, &many); err != nil {
		return err
	}

	*s = many

	return nil
}

// RegisteredClaims holds the IANA-registered JWT claims (RFC 7519 Section 4.1).
// It is intended to be embedded in concrete token claim structs.
type RegisteredClaims struct {
	Issuer    string       `json:"iss,omitempty"`
	Subject   string       `json:"sub,omitempty"`
	Audience  ClaimStrings `json:"aud,omitempty"`
	ExpiresAt *NumericDate `json:"exp,omitempty"`
	NotBefore *NumericDate `json:"nbf,omitempty"`
	IssuedAt  *NumericDate `json:"iat,omitempty"`
	ID        string       `json:"jti,omitempty"`
}

// MapClaims is an open claim set used where a fixed struct is inconvenient
// (e.g. signed userinfo responses, opaque-token conversion).
type MapClaims map[string]any

// ValidateOptions parametrise claim validation. Empty fields are not checked.
type ValidateOptions struct {
	ExpectedIssuer   string
	ExpectedAudience string
	Leeway           time.Duration
	Now              time.Time
}

// ValidateRegistered checks the temporal and identity claims. exp is required;
// iss/aud are checked only when expected values are supplied. A small leeway
// absorbs clock skew.
func ValidateRegistered(c RegisteredClaims, opts ValidateOptions) error {
	now := opts.Now
	if now.IsZero() {
		now = time.Now()
	}

	if c.ExpiresAt == nil {
		return ErrTokenMissingExpiry
	}

	if now.After(c.ExpiresAt.Add(opts.Leeway)) {
		return ErrTokenExpired
	}

	if c.NotBefore != nil && now.Add(opts.Leeway).Before(c.NotBefore.Time) {
		return ErrTokenNotValidYet
	}

	if c.IssuedAt != nil && c.IssuedAt.After(now.Add(opts.Leeway)) {
		return ErrTokenUsedBeforeIssued
	}

	if opts.ExpectedIssuer != "" && c.Issuer != opts.ExpectedIssuer {
		return fmt.Errorf("%w: %q", ErrTokenInvalidIssuer, c.Issuer)
	}

	if opts.ExpectedAudience != "" && !audienceContains(c.Audience, opts.ExpectedAudience) {
		return fmt.Errorf("%w: missing %q", ErrTokenInvalidAudience, opts.ExpectedAudience)
	}

	return nil
}

// audienceContains reports whether aud includes target.
func audienceContains(aud ClaimStrings, target string) bool {
	for _, a := range aud {
		if a == target {
			return true
		}
	}

	return false
}
