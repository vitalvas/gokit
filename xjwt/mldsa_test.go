package xjwt

import (
	"crypto/mldsa"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMLDSASignVerify(t *testing.T) {
	for _, alg := range []string{MLDSA44, MLDSA65, MLDSA87} {
		t.Run(alg, func(t *testing.T) {
			key, jwk, err := GenerateKey(alg, "pq-1")
			require.NoError(t, err)
			assert.Equal(t, "AKP", jwk.Kty)
			assert.Equal(t, alg, jwk.Alg)
			assert.Equal(t, "sig", jwk.Use)
			assert.NotEmpty(t, jwk.Pub)
			assert.NotEmpty(t, jwk.Priv)

			token, err := Sign(alg, "pq-1", MapClaims{"sub": "pq-user"}, key)
			require.NoError(t, err)

			// Verify with the public key recovered from the published JWK.
			pub, err := jwk.PublicJWK().PublicKey(alg)
			require.NoError(t, err)
			_, ok := pub.(*mldsa.PublicKey)
			require.True(t, ok)

			payload, err := Verify(token, func(Header) (any, error) { return pub, nil }, []string{alg})
			require.NoError(t, err)
			assert.Contains(t, string(payload), `"sub":"pq-user"`)
		})
	}
}

func TestMLDSAJWKRoundTrip(t *testing.T) {
	key, jwk, err := GenerateKey(MLDSA65, "k")
	require.NoError(t, err)

	// Private JWK -> key -> sign; public JWK strips priv.
	recovered, err := jwk.PrivateKey()
	require.NoError(t, err)
	priv, ok := recovered.(*mldsa.PrivateKey)
	require.True(t, ok)
	assert.True(t, priv.Equal(key))

	pubJWK := jwk.PublicJWK()
	assert.Empty(t, pubJWK.Priv, "public JWK must not carry the private seed")
	assert.NotEmpty(t, pubJWK.Pub)
}

func TestMLDSAVerifyViaJWKS(t *testing.T) {
	key, jwk, err := GenerateKey(MLDSA87, "pq-kid")
	require.NoError(t, err)

	token, err := Sign(MLDSA87, "pq-kid", MapClaims{"sub": "u"}, key)
	require.NoError(t, err)

	set := &JWKS{Keys: []JSONWebKey{jwk.PublicJWK()}}
	payload, err := VerifyWithJWKS(token, set, []string{MLDSA87})
	require.NoError(t, err)
	assert.Contains(t, string(payload), `"sub":"u"`)
}

func TestMLDSAWrongKeyRejected(t *testing.T) {
	key, _, err := GenerateKey(MLDSA44, "a")
	require.NoError(t, err)
	_, other, err := GenerateKey(MLDSA44, "b")
	require.NoError(t, err)

	token, err := Sign(MLDSA44, "a", MapClaims{"sub": "u"}, key)
	require.NoError(t, err)

	otherPub, err := other.PublicJWK().PublicKey(MLDSA44)
	require.NoError(t, err)
	_, err = Verify(token, func(Header) (any, error) { return otherPub, nil }, []string{MLDSA44})
	assert.ErrorIs(t, err, ErrTokenSignatureInvalid)
}

func TestMLDSAKeyTypeMismatch(t *testing.T) {
	_, err := Sign(MLDSA65, "k", MapClaims{"sub": "u"}, []byte("not-mldsa"))
	assert.ErrorIs(t, err, ErrKeyTypeMismatch)
}
