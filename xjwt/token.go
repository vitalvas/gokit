package xjwt

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Sentinel errors returned by the verification path. Callers can match them
// with errors.Is.
var (
	// ErrTokenExpired is returned when the exp claim is in the past.
	ErrTokenExpired = fmt.Errorf("xjwt: token is expired")
	// ErrTokenNotValidYet is returned when the nbf claim is in the future.
	ErrTokenNotValidYet = fmt.Errorf("xjwt: token is not valid yet")
	// ErrTokenInvalidClaims is returned when a claim is malformed.
	ErrTokenInvalidClaims = fmt.Errorf("xjwt: token has invalid claim")
	// ErrTokenUsedBeforeIssued is returned when the iat claim is in the future.
	ErrTokenUsedBeforeIssued = fmt.Errorf("xjwt: token used before issued")
	// ErrTokenMissingExpiry is returned when exp is required but absent.
	ErrTokenMissingExpiry = fmt.Errorf("xjwt: token is missing exp claim")
	// ErrTokenInvalidIssuer is returned when the iss claim does not match.
	ErrTokenInvalidIssuer = fmt.Errorf("xjwt: token has invalid issuer")
	// ErrTokenInvalidAudience is returned when the aud claim does not match.
	ErrTokenInvalidAudience = fmt.Errorf("xjwt: token has invalid audience")
	// ErrTokenInvalidNonce is returned when the nonce claim does not match.
	ErrTokenInvalidNonce = fmt.Errorf("xjwt: token has invalid nonce")
	// ErrTokenInvalidType is returned when the typ header does not match.
	ErrTokenInvalidType = fmt.Errorf("xjwt: token has invalid typ header")
	// ErrTokenMissingClaim is returned when a claim required by
	// RequireAccessTokenClaims (RFC 9068 Section 2.2) is absent.
	ErrTokenMissingClaim = fmt.Errorf("xjwt: token is missing a required claim")
)

// VerifiedToken is the result of a successful verification: the parsed header,
// the claims as a MapClaims, and a Valid flag (always true when returned
// without error; present to preserve the previous call-site contract).
type VerifiedToken struct {
	Header Header
	Claims MapClaims
	Valid  bool
}

// VerifyTokenOptions tunes VerifyTokenWithOptions. The zero value verifies the
// signature and the exp/nbf temporal claims against the current time with no
// clock-skew allowance.
type VerifyTokenOptions struct {
	// SkipExpiry tolerates an expired exp (nbf is still enforced). Useful where
	// an expired token is legitimately accepted, e.g. an OIDC RP-initiated
	// logout id_token_hint.
	SkipExpiry bool
	// RequireExpiry rejects a token that has no exp claim. By default a token
	// without exp is accepted (exp is only enforced when present); set this for
	// RP/resource-server validation where an unbounded token must be refused.
	RequireExpiry bool
	// Leeway is the clock-skew allowance applied to the exp, nbf, and iat checks.
	Leeway time.Duration
	// Now overrides the reference time; the current time is used when zero.
	Now time.Time
	// ExpectedIssuer, when set, requires the iss claim to match exactly.
	ExpectedIssuer string
	// ExpectedAudience, when set, requires the aud claim to contain it. When the
	// token carries multiple audiences, OIDC Core 3.1.3.7 additionally requires
	// an azp claim equal to ExpectedAudience (treated as the client_id).
	ExpectedAudience string
	// VerifyAZP opts into the OIDC Core 3.1.3.7 SHOULD check that, whenever an
	// azp (authorized party) claim is present, it equals ExpectedAudience (the
	// client_id) regardless of how many audiences the token carries. Off by
	// default: the spec clause is a SHOULD, and a single-audience token with a
	// mismatched azp is otherwise accepted. Requires ExpectedAudience to be set.
	VerifyAZP bool
	// ExpectedNonce, when set, requires the nonce claim to match exactly (OIDC
	// replay protection for the authentication request).
	ExpectedNonce string
	// ExpectedType, when set, requires the JOSE typ header to match (e.g.
	// "at+jwt" for RFC 9068 access tokens, "JWT" for ID tokens). Matching is
	// case-insensitive and tolerates the "application/" media-type prefix.
	ExpectedType string
	// RequireAccessTokenClaims opts into enforcing the claims RFC 9068 Section
	// 2.2 mandates for JWT access tokens: iss, exp, aud, sub, client_id, iat,
	// and jti. Off by default; set it on a resource server validating "at+jwt"
	// tokens. This checks presence only; value checks (iss/aud/exp) come from
	// the Expected*/Require* fields above.
	RequireAccessTokenClaims bool
}

// VerifyToken verifies a compact JWS with the resolver and allowlist, decodes
// its payload into a MapClaims, and rejects tokens whose exp is in the past or
// whose nbf is in the future. It is the high-level entry point for parsing and
// validating a JWT.
func VerifyToken(tokenString string, resolve KeyResolver, allowedAlgs []string) (*VerifiedToken, error) {
	return VerifyTokenWithOptions(tokenString, resolve, allowedAlgs, VerifyTokenOptions{})
}

// VerifyTokenSkipExpiry is like VerifyToken but tolerates an expired exp, still
// enforcing the signature and nbf. This is useful where an expired token is
// legitimately accepted, e.g. an OIDC RP-initiated logout id_token_hint
// (OpenID Connect RP-Initiated Logout 1.0, Section 2).
func VerifyTokenSkipExpiry(tokenString string, resolve KeyResolver, allowedAlgs []string) (*VerifiedToken, error) {
	return VerifyTokenWithOptions(tokenString, resolve, allowedAlgs, VerifyTokenOptions{SkipExpiry: true})
}

// VerifyTokenWithOptions is VerifyToken with explicit control over clock skew,
// the reference time, and optional issuer/audience checks.
func VerifyTokenWithOptions(tokenString string, resolve KeyResolver, allowedAlgs []string, opts VerifyTokenOptions) (*VerifiedToken, error) {
	payload, err := Verify(tokenString, resolve, allowedAlgs)
	if err != nil {
		return nil, err
	}

	header, err := DecodeHeader(tokenString)
	if err != nil {
		return nil, err
	}

	if opts.ExpectedType != "" && !typHeaderMatches(header.Typ, opts.ExpectedType) {
		return nil, fmt.Errorf("%w: %q", ErrTokenInvalidType, header.Typ)
	}

	var claims MapClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, err
	}

	now := opts.Now
	if now.IsZero() {
		now = time.Now()
	}

	// Match golang-jwt's default parser: reject expired or not-yet-valid
	// tokens after the signature is verified.
	if err := claims.validateTemporal(now, opts); err != nil {
		return nil, err
	}

	if err := claims.validateIdentity(opts); err != nil {
		return nil, err
	}

	if opts.RequireAccessTokenClaims {
		if err := claims.requireAccessTokenClaims(); err != nil {
			return nil, err
		}
	}

	return &VerifiedToken{Header: header, Claims: claims, Valid: true}, nil
}

// requireAccessTokenClaims enforces the presence of every claim RFC 9068
// Section 2.2 mandates for a JWT access token. It checks presence only; the
// temporal and identity values are validated elsewhere.
func (m MapClaims) requireAccessTokenClaims() error {
	for _, name := range []string{"iss", "exp", "aud", "sub", "client_id", "iat", "jti"} {
		if _, ok := m[name]; !ok {
			return fmt.Errorf("%w: %s", ErrTokenMissingClaim, name)
		}
	}

	return nil
}

// typHeaderMatches compares a JOSE typ header against an expected value,
// case-insensitively and tolerating the optional "application/" media-type
// prefix (RFC 7519 Section 5.1 / RFC 9068).
func typHeaderMatches(got, want string) bool {
	norm := func(s string) string {
		s = strings.ToLower(s)

		return strings.TrimPrefix(s, "application/")
	}

	return norm(got) == norm(want)
}

// validateTemporal rejects tokens whose exp is in the past, whose nbf is in the
// future, or whose iat is in the future, applying leeway for clock skew. When
// opts.SkipExpiry is true the exp claim is not enforced (but a malformed exp is
// still rejected). A missing exp is an error only when opts.RequireExpiry is set.
func (m MapClaims) validateTemporal(now time.Time, opts VerifyTokenOptions) error {
	leeway := opts.Leeway

	if _, ok := m["exp"]; ok {
		exp, ok := numericClaim(m["exp"])
		if !ok {
			return fmt.Errorf("%w: exp", ErrTokenInvalidClaims)
		}
		if !opts.SkipExpiry && now.After(exp.Add(leeway)) {
			return ErrTokenExpired
		}
	} else if opts.RequireExpiry {
		return ErrTokenMissingExpiry
	}

	if _, ok := m["nbf"]; ok {
		nbf, ok := numericClaim(m["nbf"])
		if !ok {
			return fmt.Errorf("%w: nbf", ErrTokenInvalidClaims)
		}
		if now.Add(leeway).Before(nbf) {
			return ErrTokenNotValidYet
		}
	}

	if _, ok := m["iat"]; ok {
		iat, ok := numericClaim(m["iat"])
		if !ok {
			return fmt.Errorf("%w: iat", ErrTokenInvalidClaims)
		}
		if iat.After(now.Add(leeway)) {
			return ErrTokenUsedBeforeIssued
		}
	}

	return nil
}

// validateIdentity enforces the expected issuer, audience (with the OIDC azp
// rule), and nonce when set.
func (m MapClaims) validateIdentity(opts VerifyTokenOptions) error {
	if opts.ExpectedIssuer != "" {
		iss, _ := m.GetIssuer()
		if iss != opts.ExpectedIssuer {
			return fmt.Errorf("%w: %q", ErrTokenInvalidIssuer, iss)
		}
	}

	if opts.ExpectedAudience != "" {
		if err := m.validateAudience(opts.ExpectedAudience, opts.VerifyAZP); err != nil {
			return err
		}
	}

	if opts.ExpectedNonce != "" {
		nonce, _ := m.String("nonce")
		if nonce != opts.ExpectedNonce {
			return ErrTokenInvalidNonce
		}
	}

	return nil
}

// validateAudience enforces that aud contains want and applies the OIDC Core
// 3.1.3.7 azp rule: when the token has more than one audience, an azp claim must
// be present and equal to want (the relying party's client_id). When verifyAZP
// is set, a present azp must also equal want even for a single-audience token
// (OIDC Core 3.1.3.7 SHOULD).
func (m MapClaims) validateAudience(want string, verifyAZP bool) error {
	aud, err := m.GetAudience()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrTokenInvalidAudience, err)
	}

	found := false
	for _, a := range aud {
		if a == want {
			found = true

			break
		}
	}

	if !found {
		return fmt.Errorf("%w: missing %q", ErrTokenInvalidAudience, want)
	}

	if len(aud) > 1 {
		azp, _ := m.String("azp")
		if azp == "" {
			return fmt.Errorf("%w: multiple audiences require an azp claim", ErrTokenInvalidAudience)
		}
		if azp != want {
			return fmt.Errorf("%w: azp %q does not match", ErrTokenInvalidAudience, azp)
		}
	} else if verifyAZP {
		if azp, ok := m.String("azp"); ok && azp != want {
			return fmt.Errorf("%w: azp %q does not match", ErrTokenInvalidAudience, azp)
		}
	}

	return nil
}

func numericClaim(v any) (time.Time, bool) {
	switch n := v.(type) {
	case float64:
		return numericDateFromFloat(n)
	case int:
		return time.Unix(int64(n), 0), true
	case int64:
		return time.Unix(n, 0), true
	case json.Number:
		i, err := n.Int64()
		if err != nil {
			return time.Time{}, false
		}

		return time.Unix(i, 0), true
	default:
		return time.Time{}, false
	}
}

// GetSubject returns the sub claim.
func (m MapClaims) GetSubject() (string, error) {
	s, _ := m["sub"].(string)

	return s, nil
}

// GetIssuer returns the iss claim.
func (m MapClaims) GetIssuer() (string, error) {
	s, _ := m["iss"].(string)

	return s, nil
}

// GetAudience returns the aud claim as a slice, accepting both the string and
// array JSON encodings.
func (m MapClaims) GetAudience() ([]string, error) {
	v, ok := m["aud"]
	if !ok {
		return nil, nil
	}

	switch aud := v.(type) {
	case string:
		return []string{aud}, nil
	case []string:
		return aud, nil
	case []any:
		out := make([]string, 0, len(aud))
		for i, item := range aud {
			s, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("xjwt: invalid aud claim item %d", i)
			}
			out = append(out, s)
		}

		return out, nil
	default:
		return nil, fmt.Errorf("xjwt: invalid aud claim")
	}
}

// GetExpirationTime returns the exp claim as a time, or the zero time if
// absent.
func (m MapClaims) GetExpirationTime() (time.Time, error) {
	v, ok := m["exp"]
	if !ok {
		return time.Time{}, nil
	}

	exp, ok := numericClaim(v)
	if !ok {
		return time.Time{}, fmt.Errorf("xjwt: invalid exp claim")
	}

	return exp, nil
}

// GetIssuedAt returns the iat claim as a time, or the zero time if absent.
func (m MapClaims) GetIssuedAt() (time.Time, error) {
	v, ok := m["iat"]
	if !ok {
		return time.Time{}, nil
	}

	iat, ok := numericClaim(v)
	if !ok {
		return time.Time{}, fmt.Errorf("xjwt: invalid iat claim")
	}

	return iat, nil
}
