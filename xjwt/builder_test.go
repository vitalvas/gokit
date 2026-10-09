package xjwt

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuilderSignAndVerify(t *testing.T) {
	secret := []byte("builder-hmac-secret-key-material")
	now := time.Now().Truncate(time.Second)

	token, err := NewBuilder().
		Issuer("https://issuer.example.com").
		Subject("user-1").
		Audience("api").
		ID("jti-1").
		IssuedAt(now).
		Expiration(now.Add(time.Hour)).
		Claim("scope", "openid email").
		Sign("HS256", "k1", secret)
	require.NoError(t, err)

	vt, err := VerifyTokenWithOptions(token, func(_ Header) (any, error) { return secret, nil },
		[]string{"HS256"}, VerifyTokenOptions{ExpectedIssuer: "https://issuer.example.com", ExpectedAudience: "api"})
	require.NoError(t, err)

	sub, _ := vt.Claims.GetSubject()
	assert.Equal(t, "user-1", sub)
	scope, _ := vt.Claims.String("scope")
	assert.Equal(t, "openid email", scope)
}

func TestBuilderMultiAudience(t *testing.T) {
	claims := NewBuilder().Audience("a", "b").Build()
	aud, err := claims.GetAudience()
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b"}, aud)
}

func TestParseRequest(t *testing.T) {
	secret := []byte("request-hmac-secret-key-material")
	resolve := func(_ Header) (any, error) { return secret, nil }
	token, err := NewBuilder().Subject("u1").Expiration(time.Now().Add(time.Hour)).Sign("HS256", "", secret)
	require.NoError(t, err)

	t.Run("from Authorization header", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		vt, err := ParseRequest(r, "", resolve, []string{"HS256"})
		require.NoError(t, err)
		sub, _ := vt.Claims.GetSubject()
		assert.Equal(t, "u1", sub)
	})

	t.Run("from cookie", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.AddCookie(&http.Cookie{Name: "session", Value: token})

		vt, err := ParseRequest(r, "session", resolve, []string{"HS256"})
		require.NoError(t, err)
		sub, _ := vt.Claims.GetSubject()
		assert.Equal(t, "u1", sub)
	})

	t.Run("no token", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		_, err := ParseRequest(r, "session", resolve, []string{"HS256"})
		assert.Error(t, err)
	})

	t.Run("case-insensitive bearer prefix", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set("Authorization", fmt.Sprintf("bearer %s", token))
		_, err := TokenFromRequest(r, "")
		assert.NoError(t, err)
	})
}

func TestBuilderNotBefore(t *testing.T) {
	nbf := time.Now().Add(time.Minute).Truncate(time.Second)
	claims := NewBuilder().NotBefore(nbf).Build()
	got, ok := claims.Int64("nbf")
	require.True(t, ok)
	assert.Equal(t, nbf.Unix(), got)
}

func TestBuilderTypeAndExpiresIn(t *testing.T) {
	secret := []byte("builder-typ-secret-material-32by")
	resolve := func(_ Header) (any, error) { return secret, nil }

	tok, err := NewBuilder().
		Subject("svc").
		Claim("scope", "read write").
		Type("at+jwt").
		ExpiresIn(time.Hour).
		Sign(HS256, "k1", secret)
	require.NoError(t, err)

	hdr, err := DecodeHeader(tok)
	require.NoError(t, err)
	assert.Equal(t, "at+jwt", hdr.Typ)

	vt, err := VerifyTokenWithOptions(tok, resolve, []string{HS256},
		VerifyTokenOptions{ExpectedType: "at+jwt", RequireExpiry: true})
	require.NoError(t, err)

	_, expErr := vt.Claims.GetExpirationTime()
	assert.NoError(t, expErr)

	scopes, ok := vt.Claims.Scopes()
	require.True(t, ok)
	assert.Equal(t, []string{"read", "write"}, scopes)

	// An ID token (typ JWT) is rejected when at+jwt is required.
	idTok, err := Sign(HS256, "k1",
		MapClaims{"sub": "u", "exp": float64(time.Now().Add(time.Hour).Unix())}, secret)
	require.NoError(t, err)
	_, err = VerifyTokenWithOptions(idTok, resolve, []string{HS256}, VerifyTokenOptions{ExpectedType: "at+jwt"})
	assert.ErrorIs(t, err, ErrTokenInvalidType)
}

func TestMapClaimsTypedGetters(t *testing.T) {
	c := MapClaims{
		"roles":     []any{"admin", "user"},
		"group":     "solo",
		"count":     float64(42),
		"auth_time": float64(1700000000),
		"bad_roles": []any{"ok", 5},
	}

	roles, ok := c.StringSlice("roles")
	require.True(t, ok)
	assert.Equal(t, []string{"admin", "user"}, roles)

	solo, ok := c.StringSlice("group")
	require.True(t, ok)
	assert.Equal(t, []string{"solo"}, solo)

	_, ok = c.StringSlice("bad_roles")
	assert.False(t, ok, "non-string element rejected")

	n, ok := c.Int64("count")
	require.True(t, ok)
	assert.Equal(t, int64(42), n)

	at, ok := c.AuthTime()
	require.True(t, ok)
	assert.Equal(t, int64(1700000000), at.Unix())

	_, ok = c.StringSlice("missing")
	assert.False(t, ok)
}

func TestGenerateKeySetsUseSig(t *testing.T) {
	_, jwk, err := GenerateKey(ES256, "k1")
	require.NoError(t, err)
	assert.Equal(t, "sig", jwk.Use)
	assert.Equal(t, ES256, jwk.Alg)
}
