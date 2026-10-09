package xjwt

import (
	"strings"
	"time"
)

// OpenID Connect standard-claim accessors (OIDC Core 1.0 Section 5.1). Each
// returns the claim as a string and whether it was present and a string.

// String returns the named claim as a string and whether it was present and of
// string type.
func (m MapClaims) String(name string) (string, bool) {
	v, ok := m[name].(string)

	return v, ok
}

// Bool returns the named claim as a bool and whether it was present and a bool.
func (m MapClaims) Bool(name string) (bool, bool) {
	v, ok := m[name].(bool)

	return v, ok
}

// Int64 returns the named claim as an int64 and whether it was present and
// numeric. JSON numbers decode as float64 (or json.Number); both are accepted.
func (m MapClaims) Int64(name string) (int64, bool) {
	switch n := m[name].(type) {
	case float64:
		return int64(n), true
	case int64:
		return n, true
	case int:
		return int64(n), true
	default:
		return 0, false
	}
}

// StringSlice returns the named claim as a []string and whether it was present
// and array-shaped. A single string value is returned as a one-element slice,
// matching how IdPs encode claims like "roles" or "groups" either way.
func (m MapClaims) StringSlice(name string) ([]string, bool) {
	switch v := m[name].(type) {
	case nil:
		return nil, false
	case string:
		return []string{v}, true
	case []string:
		return v, true
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			s, ok := item.(string)
			if !ok {
				return nil, false
			}
			out = append(out, s)
		}

		return out, true
	default:
		return nil, false
	}
}

// Scopes returns the space-delimited "scope" claim as a slice (OAuth2 / RFC 9068
// access tokens). Returns nil, false when the claim is absent or not a string.
func (m MapClaims) Scopes() ([]string, bool) {
	s, ok := m.String("scope")
	if !ok || s == "" {
		return nil, false
	}

	return strings.Fields(s), true
}

// AuthTime returns the "auth_time" claim as a time and whether it was present.
func (m MapClaims) AuthTime() (time.Time, bool) {
	sec, ok := m.Int64("auth_time")
	if !ok {
		return time.Time{}, false
	}

	return time.Unix(sec, 0), true
}

// Name returns the "name" claim.
func (m MapClaims) Name() (string, bool) { return m.String("name") }

// GivenName returns the "given_name" claim.
func (m MapClaims) GivenName() (string, bool) { return m.String("given_name") }

// FamilyName returns the "family_name" claim.
func (m MapClaims) FamilyName() (string, bool) { return m.String("family_name") }

// Email returns the "email" claim.
func (m MapClaims) Email() (string, bool) { return m.String("email") }

// EmailVerified returns the "email_verified" claim.
func (m MapClaims) EmailVerified() (bool, bool) { return m.Bool("email_verified") }

// PreferredUsername returns the "preferred_username" claim.
func (m MapClaims) PreferredUsername() (string, bool) { return m.String("preferred_username") }

// Nonce returns the "nonce" claim (OIDC authentication request binding).
func (m MapClaims) Nonce() (string, bool) { return m.String("nonce") }

// AuthorizedParty returns the "azp" claim.
func (m MapClaims) AuthorizedParty() (string, bool) { return m.String("azp") }
