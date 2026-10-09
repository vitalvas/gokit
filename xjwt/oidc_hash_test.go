package xjwt

import (
	"crypto"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccessTokenHashKnownAnswer(t *testing.T) {
	// OIDC Core 1.0 Section 3.3.2.11 worked example: access_token
	// "jHkWEdUXMU1BwAsC4vtUsZwnNvTIxEl0z9K3vx5KF0Y" with an RS256/ES256 ID token
	// (SHA-256) yields at_hash "77QmUPtjPfzWtF2AnpK9RQ".
	const (
		accessToken = "jHkWEdUXMU1BwAsC4vtUsZwnNvTIxEl0z9K3vx5KF0Y"
		wantAtHash  = "77QmUPtjPfzWtF2AnpK9RQ"
	)

	got, err := AccessTokenHash(RS256, accessToken)
	require.NoError(t, err)
	assert.Equal(t, wantAtHash, got)

	// Same hash family for ES256 (SHA-256).
	gotES, err := AccessTokenHash(ES256, accessToken)
	require.NoError(t, err)
	assert.Equal(t, wantAtHash, gotES)

	assert.True(t, VerifyAccessTokenHash(RS256, accessToken, wantAtHash))
	assert.False(t, VerifyAccessTokenHash(RS256, accessToken, "tampered"))
}

func TestHalfHashPerAlg(t *testing.T) {
	// The digest length tracks the alg's hash: SHA-256 -> 32-byte digest ->
	// 16-byte half -> 22 base64url chars; SHA-512 -> 32-byte half -> 43 chars.
	cases := map[string]int{
		RS256:  22, // SHA-256 half = 16 bytes
		ES384:  32, // SHA-384 half = 24 bytes -> 32 chars
		ES512:  43, // SHA-512 half = 32 bytes -> 43 chars
		EdDSA:  43, // SHA-512
		ES256K: 22,
	}

	for alg, wantLen := range cases {
		t.Run(alg, func(t *testing.T) {
			h, err := AccessTokenHash(alg, "some-access-token")
			require.NoError(t, err)
			assert.Len(t, h, wantLen)
		})
	}
}

func TestCodeAndStateHash(t *testing.T) {
	c, err := CodeHash(ES256, "auth-code-123")
	require.NoError(t, err)
	assert.True(t, VerifyCodeHash(ES256, "auth-code-123", c))
	assert.False(t, VerifyCodeHash(ES256, "auth-code-123", "nope"))

	s, err := StateHash(ES256, "state-xyz")
	require.NoError(t, err)
	assert.NotEmpty(t, s)
}

func TestHashUnknownAlg(t *testing.T) {
	_, err := AccessTokenHash("XX999", "tok")
	require.Error(t, err)
}

func TestOIDCHashHelperErrors(t *testing.T) {
	// Unknown alg in the hash helpers.
	assert.False(t, VerifyAccessTokenHash("BADALG", "tok", "hash"))
	assert.False(t, VerifyCodeHash("BADALG", "code", "hash"))

	_, err := StateHash("BADALG", "state")
	require.Error(t, err)

	// hashForTokenAlg: HS256 has a hash; a bogus alg does not.
	_, err = hashForTokenAlg("BADALG")
	require.Error(t, err)
	h, err := hashForTokenAlg(EdDSA)
	require.NoError(t, err)
	assert.Equal(t, crypto.SHA512, h)
	h, err = hashForTokenAlg(ES256K)
	require.NoError(t, err)
	assert.Equal(t, crypto.SHA256, h)
}
