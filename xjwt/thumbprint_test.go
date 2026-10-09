package xjwt

import (
	"crypto"
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJWKThumbprint(t *testing.T) {
	t.Run("EC ignores extra members and is deterministic", func(t *testing.T) {
		jwk := json.RawMessage(`{"kty":"EC","crv":"P-256","x":"AA","y":"BB","extra":"ignored"}`)
		tp, err := JWKThumbprint(jwk)
		require.NoError(t, err)
		assert.NotEmpty(t, tp)

		tp2, _ := JWKThumbprint(jwk)
		assert.Equal(t, tp, tp2)
	})

	t.Run("RSA", func(t *testing.T) {
		jwk := json.RawMessage(`{"kty":"RSA","n":"AA","e":"AQAB"}`)
		tp, err := JWKThumbprint(jwk)
		require.NoError(t, err)
		assert.NotEmpty(t, tp)
	})

	t.Run("missing member rejected", func(t *testing.T) {
		_, err := JWKThumbprint(json.RawMessage(`{"kty":"EC","crv":"P-256","x":"AA"}`))
		require.Error(t, err)

		_, err = JWKThumbprint(json.RawMessage(`{"kty":"RSA","n":"AA"}`))
		require.Error(t, err)
	})

	t.Run("oct is supported", func(t *testing.T) {
		_, err := JWKThumbprint(json.RawMessage(`{"kty":"oct","k":"AA"}`))
		require.NoError(t, err)
	})

	t.Run("unknown kty rejected", func(t *testing.T) {
		_, err := JWKThumbprint(json.RawMessage(`{"kty":"XYZ"}`))
		require.Error(t, err)
	})

	t.Run("invalid json rejected", func(t *testing.T) {
		_, err := JWKThumbprint(json.RawMessage(`nope`))
		require.Error(t, err)
	})
}

func TestThumbprintMethodAndAssignKeyID(t *testing.T) {
	jwk := JSONWebKey{Kty: "RSA", N: "abc", E: "AQAB"}

	tp256, err := jwk.Thumbprint(crypto.SHA256)
	require.NoError(t, err)
	assert.Len(t, tp256, 32)

	tp512, err := jwk.Thumbprint(crypto.SHA512)
	require.NoError(t, err)
	assert.Len(t, tp512, 64)

	// Deterministic.
	tp256b, err := jwk.Thumbprint(crypto.SHA256)
	require.NoError(t, err)
	assert.Equal(t, tp256, tp256b)

	t.Run("AssignKeyID sets kid to the SHA-256 thumbprint", func(t *testing.T) {
		k := JSONWebKey{Kty: "RSA", N: "abc", E: "AQAB"}
		require.NoError(t, k.AssignKeyID())
		assert.Equal(t, base64.RawURLEncoding.EncodeToString(tp256), k.Kid)
	})

	t.Run("OKP thumbprint supported", func(t *testing.T) {
		okp := JSONWebKey{Kty: "OKP", Crv: "Ed25519", X: "abc"}
		tp, err := okp.Thumbprint(crypto.SHA256)
		require.NoError(t, err)
		assert.Len(t, tp, 32)
	})
}

func TestThumbprintErrors(t *testing.T) {
	// Missing required members per kty.
	for _, k := range []JSONWebKey{
		{Kty: "EC", Crv: "P-256"},    // missing x/y
		{Kty: "RSA"},                 // missing n/e
		{Kty: "OKP", Crv: "Ed25519"}, // missing x
		{Kty: "oct"},                 // missing k
		{Kty: "WHAT"},                // unsupported
	} {
		_, err := k.Thumbprint(crypto.SHA256)
		require.Error(t, err, k.Kty)
	}

	// AssignKeyID propagates the error for an incomplete key.
	bad := &JSONWebKey{Kty: "RSA"}
	require.Error(t, bad.AssignKeyID())
}
