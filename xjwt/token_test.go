package xjwt

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVerifyToken(t *testing.T) {
	secret := []byte("test-hmac-secret-material-32byte")
	resolve := func(_ Header) (any, error) { return secret, nil }
	allow := []string{"HS256"}
	now := time.Now()

	sign := func(claims MapClaims) string {
		tok, err := Sign("HS256", "k1", claims, secret)
		require.NoError(t, err)

		return tok
	}

	t.Run("valid token", func(t *testing.T) {
		tok := sign(MapClaims{"sub": "u1", "exp": float64(now.Add(time.Hour).Unix())})

		vt, err := VerifyToken(tok, resolve, allow)
		require.NoError(t, err)
		assert.True(t, vt.Valid)
		assert.Equal(t, "HS256", vt.Header.Alg)
		sub, _ := vt.Claims.GetSubject()
		assert.Equal(t, "u1", sub)
	})

	t.Run("expired token rejected", func(t *testing.T) {
		tok := sign(MapClaims{"sub": "u1", "exp": float64(now.Add(-time.Hour).Unix())})

		_, err := VerifyToken(tok, resolve, allow)
		assert.ErrorIs(t, err, ErrTokenExpired)
	})

	t.Run("expired token tolerated by SkipExpiry", func(t *testing.T) {
		tok := sign(MapClaims{"sub": "u1", "exp": float64(now.Add(-time.Hour).Unix())})

		vt, err := VerifyTokenSkipExpiry(tok, resolve, allow)
		require.NoError(t, err)
		assert.True(t, vt.Valid)
	})

	t.Run("not-yet-valid rejected even by SkipExpiry", func(t *testing.T) {
		tok := sign(MapClaims{
			"exp": float64(now.Add(-time.Hour).Unix()),
			"nbf": float64(now.Add(time.Hour).Unix()),
		})

		_, err := VerifyTokenSkipExpiry(tok, resolve, allow)
		assert.ErrorIs(t, err, ErrTokenNotValidYet)
	})

	t.Run("wrong key rejected", func(t *testing.T) {
		tok := sign(MapClaims{"exp": float64(now.Add(time.Hour).Unix())})
		badResolve := func(_ Header) (any, error) { return []byte("wrong-secret"), nil }

		_, err := VerifyToken(tok, badResolve, allow)
		assert.Error(t, err)
	})

	t.Run("disallowed algorithm rejected", func(t *testing.T) {
		tok := sign(MapClaims{"exp": float64(now.Add(time.Hour).Unix())})

		_, err := VerifyToken(tok, resolve, []string{"RS256"})
		assert.Error(t, err)
	})

	t.Run("leeway tolerates slight expiry", func(t *testing.T) {
		tok := sign(MapClaims{"exp": float64(now.Add(-30 * time.Second).Unix())})

		_, err := VerifyToken(tok, resolve, allow)
		assert.ErrorIs(t, err, ErrTokenExpired)

		vt, err := VerifyTokenWithOptions(tok, resolve, allow, VerifyTokenOptions{Leeway: time.Minute})
		require.NoError(t, err)
		assert.True(t, vt.Valid)
	})

	t.Run("Now overrides reference time", func(t *testing.T) {
		exp := now.Add(time.Hour)
		tok := sign(MapClaims{"exp": float64(exp.Unix())})

		_, err := VerifyTokenWithOptions(tok, resolve, allow, VerifyTokenOptions{Now: exp.Add(time.Hour)})
		assert.ErrorIs(t, err, ErrTokenExpired)
	})

	t.Run("expected issuer enforced", func(t *testing.T) {
		tok := sign(MapClaims{"iss": "right", "exp": float64(now.Add(time.Hour).Unix())})

		_, err := VerifyTokenWithOptions(tok, resolve, allow, VerifyTokenOptions{ExpectedIssuer: "right"})
		require.NoError(t, err)

		_, err = VerifyTokenWithOptions(tok, resolve, allow, VerifyTokenOptions{ExpectedIssuer: "wrong"})
		assert.ErrorIs(t, err, ErrTokenInvalidIssuer)
	})

	t.Run("expected audience enforced (single aud)", func(t *testing.T) {
		tok := sign(MapClaims{"aud": "api", "exp": float64(now.Add(time.Hour).Unix())})

		_, err := VerifyTokenWithOptions(tok, resolve, allow, VerifyTokenOptions{ExpectedAudience: "api"})
		require.NoError(t, err)

		_, err = VerifyTokenWithOptions(tok, resolve, allow, VerifyTokenOptions{ExpectedAudience: "missing"})
		assert.ErrorIs(t, err, ErrTokenInvalidAudience)
	})

	t.Run("multi-audience requires matching azp (OIDC Core 3.1.3.7)", func(t *testing.T) {
		exp := float64(now.Add(time.Hour).Unix())

		// No azp with multiple audiences -> rejected.
		noAzp := sign(MapClaims{"aud": []any{"api", "web"}, "exp": exp})
		_, err := VerifyTokenWithOptions(noAzp, resolve, allow, VerifyTokenOptions{ExpectedAudience: "api"})
		assert.ErrorIs(t, err, ErrTokenInvalidAudience)

		// azp present and equal to the expected audience -> accepted.
		withAzp := sign(MapClaims{"aud": []any{"api", "web"}, "azp": "api", "exp": exp})
		_, err = VerifyTokenWithOptions(withAzp, resolve, allow, VerifyTokenOptions{ExpectedAudience: "api"})
		require.NoError(t, err)

		// azp present but mismatched -> rejected.
		wrongAzp := sign(MapClaims{"aud": []any{"api", "web"}, "azp": "web", "exp": exp})
		_, err = VerifyTokenWithOptions(wrongAzp, resolve, allow, VerifyTokenOptions{ExpectedAudience: "api"})
		assert.ErrorIs(t, err, ErrTokenInvalidAudience)
	})
}

func TestVerifyTokenOIDCOptions(t *testing.T) {
	secret := []byte("oidc-opts-secret-key-material-32")
	resolve := func(_ Header) (any, error) { return secret, nil }
	allow := []string{HS256}
	now := time.Now()
	exp := float64(now.Add(time.Hour).Unix())

	sign := func(claims MapClaims) string {
		tok, err := Sign(HS256, "k1", claims, secret)
		require.NoError(t, err)

		return tok
	}

	t.Run("RequireExpiry rejects token without exp", func(t *testing.T) {
		tok := sign(MapClaims{"sub": "u"})
		_, err := VerifyToken(tok, resolve, allow)
		require.NoError(t, err, "default accepts missing exp")

		_, err = VerifyTokenWithOptions(tok, resolve, allow, VerifyTokenOptions{RequireExpiry: true})
		assert.ErrorIs(t, err, ErrTokenMissingExpiry)
	})

	t.Run("iat in the future rejected", func(t *testing.T) {
		tok := sign(MapClaims{"exp": exp, "iat": float64(now.Add(time.Hour).Unix())})
		_, err := VerifyToken(tok, resolve, allow)
		assert.ErrorIs(t, err, ErrTokenUsedBeforeIssued)
	})

	t.Run("nonce enforced", func(t *testing.T) {
		tok := sign(MapClaims{"exp": exp, "nonce": "n-123"})
		_, err := VerifyTokenWithOptions(tok, resolve, allow, VerifyTokenOptions{ExpectedNonce: "n-123"})
		require.NoError(t, err)

		_, err = VerifyTokenWithOptions(tok, resolve, allow, VerifyTokenOptions{ExpectedNonce: "wrong"})
		assert.ErrorIs(t, err, ErrTokenInvalidNonce)
	})

	t.Run("expected issuer sentinel", func(t *testing.T) {
		tok := sign(MapClaims{"exp": exp, "iss": "https://op.example"})
		_, err := VerifyTokenWithOptions(tok, resolve, allow, VerifyTokenOptions{ExpectedIssuer: "https://evil"})
		assert.ErrorIs(t, err, ErrTokenInvalidIssuer)
	})
}

func TestValidateRegisteredSentinels(t *testing.T) {
	past := NewNumericDate(time.Now().Add(-time.Hour))
	future := NewNumericDate(time.Now().Add(time.Hour))

	t.Run("missing exp", func(t *testing.T) {
		err := ValidateRegistered(RegisteredClaims{}, ValidateOptions{})
		assert.ErrorIs(t, err, ErrTokenMissingExpiry)
	})

	t.Run("expired matches sentinel", func(t *testing.T) {
		err := ValidateRegistered(RegisteredClaims{ExpiresAt: past}, ValidateOptions{})
		assert.ErrorIs(t, err, ErrTokenExpired)
	})

	t.Run("wrong issuer matches sentinel", func(t *testing.T) {
		err := ValidateRegistered(
			RegisteredClaims{ExpiresAt: future, Issuer: "a"},
			ValidateOptions{ExpectedIssuer: "b"})
		assert.ErrorIs(t, err, ErrTokenInvalidIssuer)
	})

	t.Run("wrong audience matches sentinel", func(t *testing.T) {
		err := ValidateRegistered(
			RegisteredClaims{ExpiresAt: future, Audience: ClaimStrings{"x"}},
			ValidateOptions{ExpectedAudience: "y"})
		assert.ErrorIs(t, err, ErrTokenInvalidAudience)
	})
}

func TestMapClaimsAccessors(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	claims := MapClaims{
		"sub": "user-1",
		"iss": "https://sso.example.com",
		"aud": []any{"app", "api"},
		"exp": float64(now.Add(time.Hour).Unix()),
		"iat": float64(now.Unix()),
	}

	sub, _ := claims.GetSubject()
	assert.Equal(t, "user-1", sub)

	iss, _ := claims.GetIssuer()
	assert.Equal(t, "https://sso.example.com", iss)

	aud, _ := claims.GetAudience()
	assert.Equal(t, []string{"app", "api"}, aud)

	exp, _ := claims.GetExpirationTime()
	assert.Equal(t, now.Add(time.Hour).Unix(), exp.Unix())

	iat, _ := claims.GetIssuedAt()
	assert.Equal(t, now.Unix(), iat.Unix())
}

func TestGetAudienceStringForm(t *testing.T) {
	aud, _ := MapClaims{"aud": "single"}.GetAudience()
	assert.Equal(t, []string{"single"}, aud)

	none, _ := MapClaims{}.GetAudience()
	assert.Empty(t, none)

	_, err := MapClaims{"aud": []any{"app", 42}}.GetAudience()
	assert.Error(t, err)

	_, err = MapClaims{"aud": 42}.GetAudience()
	assert.Error(t, err)
}

func TestValidateTemporal(t *testing.T) {
	now := time.Unix(1700000000, 0)

	t.Run("valid", func(t *testing.T) {
		c := MapClaims{"exp": float64(now.Add(time.Hour).Unix())}
		assert.NoError(t, c.validateTemporal(now, VerifyTokenOptions{}))
	})

	t.Run("expired", func(t *testing.T) {
		c := MapClaims{"exp": float64(now.Add(-time.Hour).Unix())}
		assert.ErrorIs(t, c.validateTemporal(now, VerifyTokenOptions{}), ErrTokenExpired)
	})

	t.Run("expired tolerated when skipExp", func(t *testing.T) {
		c := MapClaims{"exp": float64(now.Add(-time.Hour).Unix())}
		assert.NoError(t, c.validateTemporal(now, VerifyTokenOptions{SkipExpiry: true}))
	})

	t.Run("nbf still enforced when skipExp", func(t *testing.T) {
		c := MapClaims{
			"exp": float64(now.Add(-time.Hour).Unix()),
			"nbf": float64(now.Add(time.Minute).Unix()),
		}
		assert.ErrorIs(t, c.validateTemporal(now, VerifyTokenOptions{SkipExpiry: true}), ErrTokenNotValidYet)
	})

	t.Run("malformed exp still rejected when skipExp", func(t *testing.T) {
		c := MapClaims{"exp": "tomorrow"}
		assert.ErrorIs(t, c.validateTemporal(now, VerifyTokenOptions{SkipExpiry: true}), ErrTokenInvalidClaims)
	})

	t.Run("not yet valid", func(t *testing.T) {
		c := MapClaims{
			"exp": float64(now.Add(time.Hour).Unix()),
			"nbf": float64(now.Add(time.Minute).Unix()),
		}
		assert.ErrorIs(t, c.validateTemporal(now, VerifyTokenOptions{}), ErrTokenNotValidYet)
	})

	t.Run("invalid exp rejected", func(t *testing.T) {
		c := MapClaims{"exp": "tomorrow"}
		assert.ErrorIs(t, c.validateTemporal(now, VerifyTokenOptions{}), ErrTokenInvalidClaims)
	})

	t.Run("out-of-range exp rejected", func(t *testing.T) {
		c := MapClaims{"exp": 1e100}
		assert.ErrorIs(t, c.validateTemporal(now, VerifyTokenOptions{}), ErrTokenInvalidClaims)
	})

	t.Run("invalid nbf rejected", func(t *testing.T) {
		c := MapClaims{
			"exp": float64(now.Add(time.Hour).Unix()),
			"nbf": "tomorrow",
		}
		assert.ErrorIs(t, c.validateTemporal(now, VerifyTokenOptions{}), ErrTokenInvalidClaims)
	})

	t.Run("no exp is allowed at this layer", func(t *testing.T) {
		assert.NoError(t, MapClaims{}.validateTemporal(now, VerifyTokenOptions{}))
	})
}

func TestNumericClaimTypes(t *testing.T) {
	cases := []struct {
		name string
		v    any
		ok   bool
	}{
		{"float64", float64(1700000000), true},
		{"int", int(1700000000), true},
		{"int64", int64(1700000000), true},
		{"json.Number", json.Number("1700000000"), true},
		{"bad json.Number", json.Number("not-a-number"), false},
		{"string", "nope", false},
		{"bool", true, false},
		{"nil", nil, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, ok := numericClaim(c.v)
			assert.Equal(t, c.ok, ok)
		})
	}
}

func TestGetTimeClaims(t *testing.T) {
	t.Run("exp present", func(t *testing.T) {
		got, err := MapClaims{"exp": float64(1700000000)}.GetExpirationTime()
		require.NoError(t, err)
		assert.Equal(t, int64(1700000000), got.Unix())
	})
	t.Run("exp absent", func(t *testing.T) {
		got, err := MapClaims{}.GetExpirationTime()
		require.NoError(t, err)
		assert.True(t, got.IsZero())
	})
	t.Run("exp invalid", func(t *testing.T) {
		_, err := MapClaims{"exp": "bad"}.GetExpirationTime()
		require.Error(t, err)
	})
	t.Run("iat present", func(t *testing.T) {
		got, err := MapClaims{"iat": int64(1700000000)}.GetIssuedAt()
		require.NoError(t, err)
		assert.Equal(t, int64(1700000000), got.Unix())
	})
	t.Run("iat absent", func(t *testing.T) {
		got, err := MapClaims{}.GetIssuedAt()
		require.NoError(t, err)
		assert.True(t, got.IsZero())
	})
	t.Run("iat invalid", func(t *testing.T) {
		_, err := MapClaims{"iat": "bad"}.GetIssuedAt()
		require.Error(t, err)
	})
}

func TestTypedAccessorMisses(t *testing.T) {
	m := MapClaims{"s": "v", "b": true, "n": float64(5), "arr": []any{"a"}}

	_, ok := m.Int64("missing")
	assert.False(t, ok)
	_, ok = m.Int64("s") // present but not numeric
	assert.False(t, ok)

	_, ok = m.StringSlice("missing")
	assert.False(t, ok)

	_, ok = m.Scopes() // no scope claim
	assert.False(t, ok)
	_, ok = MapClaims{"scope": ""}.Scopes() // empty scope
	assert.False(t, ok)
	sc, ok := MapClaims{"scope": "a b c"}.Scopes()
	require.True(t, ok)
	assert.Equal(t, []string{"a", "b", "c"}, sc)

	_, ok = m.AuthTime() // absent
	assert.False(t, ok)
	at, ok := MapClaims{"auth_time": float64(1700000000)}.AuthTime()
	require.True(t, ok)
	assert.Equal(t, int64(1700000000), at.Unix())
}

func TestDecodeHeaderErrors(t *testing.T) {
	_, err := DecodeHeader("a.b") // wrong segment count
	require.Error(t, err)
	_, err = DecodeHeader("!!!.b.c") // bad base64 header
	require.Error(t, err)
	_, err = DecodeHeader(fmt.Sprintf("%s.b.c", base64Seg(`not json`))) // bad json header
	require.Error(t, err)
}

// base64Seg encodes s as a raw-url base64 JWS segment.
func base64Seg(s string) string {
	return b64(([]byte)(s))
}

func TestValidateTemporalMalformedIat(t *testing.T) {
	err := MapClaims{"iat": "not-numeric"}.validateTemporal(time.Now(), VerifyTokenOptions{})
	require.ErrorIs(t, err, ErrTokenInvalidClaims)
}

func TestValidateAudienceMalformed(t *testing.T) {
	// aud present but not a valid audience type.
	err := MapClaims{"aud": 42}.validateIdentity(VerifyTokenOptions{ExpectedAudience: "x"})
	require.ErrorIs(t, err, ErrTokenInvalidAudience)
}

func TestInt64AllNumericTypes(t *testing.T) {
	assert.Equal(t, int64(1), mustInt64(t, MapClaims{"v": int64(1)}))
	assert.Equal(t, int64(2), mustInt64(t, MapClaims{"v": int(2)}))
	assert.Equal(t, int64(3), mustInt64(t, MapClaims{"v": float64(3)}))
}

func mustInt64(t *testing.T, m MapClaims) int64 {
	t.Helper()
	n, ok := m.Int64("v")
	require.True(t, ok)

	return n
}

func TestStringSliceTypes(t *testing.T) {
	// native []string
	got, ok := MapClaims{"v": []string{"a", "b"}}.StringSlice("v")
	require.True(t, ok)
	assert.Equal(t, []string{"a", "b"}, got)

	// nil value
	_, ok = MapClaims{"v": nil}.StringSlice("v")
	assert.False(t, ok)

	// []any of strings
	got, ok = MapClaims{"v": []any{"x"}}.StringSlice("v")
	require.True(t, ok)
	assert.Equal(t, []string{"x"}, got)
}

func TestVerifyTokenBadHeaderAndPayload(t *testing.T) {
	secret := []byte("verify-token-secret-material-32b")
	resolve := func(Header) (any, error) { return secret, nil }

	t.Run("malformed token header", func(t *testing.T) {
		_, err := VerifyToken("!!!.body.sig", resolve, []string{HS256})
		require.Error(t, err)
	})

	t.Run("payload not JSON", func(t *testing.T) {
		// Craft a validly-signed token whose payload is not a JSON object.
		enc := base64.RawURLEncoding
		header := enc.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
		payload := enc.EncodeToString([]byte(`not-json`))
		signingInput := fmt.Sprintf("%s.%s", header, payload)
		sig, err := signPayload(HS256, signingInput, secret)
		require.NoError(t, err)
		token := fmt.Sprintf("%s.%s.%s", header, payload, enc.EncodeToString(sig))

		_, err = VerifyToken(token, resolve, []string{HS256})
		require.Error(t, err)
	})
}

func TestTokenKidMalformed(t *testing.T) {
	// tokenKid returns "" for a malformed token (DecodeHeader fails).
	assert.Equal(t, "", tokenKid("not-a-token"))
}

func TestNumericDateUnmarshalBadJSON(t *testing.T) {
	var d NumericDate
	assert.Error(t, d.UnmarshalJSON([]byte(`"not-a-number"`)))
	assert.Error(t, d.UnmarshalJSON([]byte(`1e999`))) // out of range
}

// TestVerifyAZPFlag covers the OIDC Core 3.1.3.7 SHOULD check behind VerifyAZP:
// a single-audience token with a mismatched azp is accepted by default but
// rejected when the flag is set.
func TestVerifyAZPFlag(t *testing.T) {
	secret := []byte("azp-flag-hmac-secret-material-32")
	resolve := func(_ Header) (any, error) { return secret, nil }
	allow := []string{HS256}
	exp := float64(time.Now().Add(time.Hour).Unix())

	sign := func(claims MapClaims) string {
		tok, err := Sign(HS256, "k", claims, secret)
		require.NoError(t, err)

		return tok
	}

	t.Run("single aud mismatched azp accepted by default", func(t *testing.T) {
		tok := sign(MapClaims{"aud": "client-A", "azp": "client-B", "exp": exp})
		_, err := VerifyTokenWithOptions(tok, resolve, allow, VerifyTokenOptions{ExpectedAudience: "client-A"})
		require.NoError(t, err)
	})

	t.Run("single aud mismatched azp rejected with flag", func(t *testing.T) {
		tok := sign(MapClaims{"aud": "client-A", "azp": "client-B", "exp": exp})
		_, err := VerifyTokenWithOptions(tok, resolve, allow, VerifyTokenOptions{
			ExpectedAudience: "client-A",
			VerifyAZP:        true,
		})
		require.ErrorIs(t, err, ErrTokenInvalidAudience)
	})

	t.Run("matching azp passes with flag", func(t *testing.T) {
		tok := sign(MapClaims{"aud": "client-A", "azp": "client-A", "exp": exp})
		_, err := VerifyTokenWithOptions(tok, resolve, allow, VerifyTokenOptions{
			ExpectedAudience: "client-A",
			VerifyAZP:        true,
		})
		require.NoError(t, err)
	})

	t.Run("absent azp passes with flag", func(t *testing.T) {
		tok := sign(MapClaims{"aud": "client-A", "exp": exp})
		_, err := VerifyTokenWithOptions(tok, resolve, allow, VerifyTokenOptions{
			ExpectedAudience: "client-A",
			VerifyAZP:        true,
		})
		require.NoError(t, err)
	})
}

// TestRequireAccessTokenClaims covers the RFC 9068 Section 2.2 required-claim
// enforcement behind RequireAccessTokenClaims.
func TestRequireAccessTokenClaims(t *testing.T) {
	secret := []byte("at-claims-hmac-secret-material-32")
	resolve := func(_ Header) (any, error) { return secret, nil }
	allow := []string{HS256}
	exp := float64(time.Now().Add(time.Hour).Unix())
	iat := float64(time.Now().Add(-time.Minute).Unix())

	full := MapClaims{
		"iss":       "https://issuer.example.com",
		"exp":       exp,
		"aud":       "resource",
		"sub":       "user-1",
		"client_id": "client-1",
		"iat":       iat,
		"jti":       "token-id-1",
	}
	sign := func(claims MapClaims) string {
		tok, err := Sign(HS256, "k", claims, secret)
		require.NoError(t, err)

		return tok
	}

	t.Run("complete access token passes", func(t *testing.T) {
		_, err := VerifyTokenWithOptions(sign(full), resolve, allow, VerifyTokenOptions{
			RequireAccessTokenClaims: true,
		})
		require.NoError(t, err)
	})

	for _, missing := range []string{"iss", "exp", "aud", "sub", "client_id", "iat", "jti"} {
		t.Run(fmt.Sprintf("missing %s rejected", missing), func(t *testing.T) {
			claims := MapClaims{}
			for k, v := range full {
				if k != missing {
					claims[k] = v
				}
			}
			_, err := VerifyTokenWithOptions(sign(claims), resolve, allow, VerifyTokenOptions{
				RequireAccessTokenClaims: true,
			})
			require.ErrorIs(t, err, ErrTokenMissingClaim)
		})
	}
}
