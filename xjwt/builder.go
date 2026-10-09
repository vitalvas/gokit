package xjwt

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Builder is a fluent constructor for a JWT claim set. Each setter returns the
// builder, so calls can be chained; Build returns the accumulated MapClaims and
// Sign signs them directly.
type Builder struct {
	claims MapClaims
	typ    string
}

// NewBuilder creates an empty claim-set builder.
func NewBuilder() *Builder {
	return &Builder{claims: MapClaims{}, typ: "JWT"}
}

// Issuer sets the iss claim.
func (b *Builder) Issuer(v string) *Builder { return b.set("iss", v) }

// Subject sets the sub claim.
func (b *Builder) Subject(v string) *Builder { return b.set("sub", v) }

// Audience sets the aud claim. A single value is stored as a string; multiple
// values as an array.
func (b *Builder) Audience(v ...string) *Builder {
	if len(v) == 1 {
		return b.set("aud", v[0])
	}

	return b.set("aud", v)
}

// ID sets the jti claim.
func (b *Builder) ID(v string) *Builder { return b.set("jti", v) }

// Expiration sets the exp claim (seconds since the epoch).
func (b *Builder) Expiration(t time.Time) *Builder { return b.setTime("exp", t) }

// NotBefore sets the nbf claim.
func (b *Builder) NotBefore(t time.Time) *Builder { return b.setTime("nbf", t) }

// IssuedAt sets the iat claim.
func (b *Builder) IssuedAt(t time.Time) *Builder { return b.setTime("iat", t) }

// ExpiresIn sets exp to now+d and iat to now, the common OP minting pattern.
func (b *Builder) ExpiresIn(d time.Duration) *Builder {
	now := time.Now()
	b.setTime("iat", now)

	return b.setTime("exp", now.Add(d))
}

// Type sets the JOSE typ header used when the builder signs (default "JWT"); use
// "at+jwt" for RFC 9068 access tokens or "logout+jwt" for back-channel logout.
func (b *Builder) Type(typ string) *Builder {
	b.typ = typ

	return b
}

// Claim sets an arbitrary claim.
func (b *Builder) Claim(name string, v any) *Builder { return b.set(name, v) }

func (b *Builder) set(name string, v any) *Builder {
	b.claims[name] = v

	return b
}

func (b *Builder) setTime(name string, t time.Time) *Builder {
	b.claims[name] = t.Unix()

	return b
}

// Build returns the accumulated claims.
func (b *Builder) Build() MapClaims {
	return b.claims
}

// Sign builds and signs the claims with key under alg, writing kid when set and
// using the builder's typ header (default "JWT").
func (b *Builder) Sign(alg, kid string, key any) (string, error) {
	return SignWithType(alg, b.typ, kid, b.claims, key)
}

// TokenFromRequest extracts a bearer token from an HTTP request: first the
// "Authorization: Bearer <token>" header, then a cookie named by cookieName when
// non-empty. It returns an error if no token is found.
func TokenFromRequest(r *http.Request, cookieName string) (string, error) {
	if auth := r.Header.Get("Authorization"); auth != "" {
		const prefix = "Bearer "
		if len(auth) > len(prefix) && strings.EqualFold(auth[:len(prefix)], prefix) {
			return auth[len(prefix):], nil
		}
	}

	if cookieName != "" {
		if c, err := r.Cookie(cookieName); err == nil && c.Value != "" {
			return c.Value, nil
		}
	}

	return "", fmt.Errorf("xjwt: no bearer token in request")
}

// ParseRequest extracts a bearer token from r (Authorization header, then the
// named cookie) and verifies it, returning the verified token.
func ParseRequest(r *http.Request, cookieName string, resolve KeyResolver, allowedAlgs []string) (*VerifiedToken, error) {
	token, err := TokenFromRequest(r, cookieName)
	if err != nil {
		return nil, err
	}

	return VerifyToken(token, resolve, allowedAlgs)
}
