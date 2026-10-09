package xjwt

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vitalvas/gokit/secp256k1"
)

func TestGenerateKeyAndRoundTrip(t *testing.T) {
	algs := []string{"RS256", "PS256", "ES256", "ES384", "ES512", "ES256K", "EdDSA", "HS256", "HS384", "HS512"}

	for _, alg := range algs {
		t.Run(alg, func(t *testing.T) {
			key, jwk, err := GenerateKey(alg, "kid-1")
			require.NoError(t, err)
			require.NotNil(t, key)
			assert.Equal(t, "kid-1", jwk.Kid)
			assert.Equal(t, alg, jwk.Alg)

			// The generated key must sign and verify a token.
			token, err := Sign(alg, "kid-1", MapClaims{"sub": "u1"}, key)
			require.NoError(t, err)

			// Recover the key from the private JWK and derive the verifier.
			back, err := jwk.PrivateKey()
			require.NoError(t, err)

			pub := publicOf(t, back, alg)
			_, err = Verify(token, func(_ Header) (any, error) { return pub, nil }, []string{alg})
			require.NoError(t, err, "key recovered from JWK must verify")
		})
	}
}

// publicOf returns the verification key matching a recovered private key.
func publicOf(t *testing.T, priv any, _ string) any {
	t.Helper()

	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &k.PublicKey
	case *ecdsa.PrivateKey:
		return &k.PublicKey
	case ed25519.PrivateKey:
		return k.Public()
	case *secp256k1.PrivateKey:
		return &k.Pub
	case []byte:
		return k // HMAC uses the same secret
	default:
		t.Fatalf("unexpected key type %T", priv)

		return nil
	}
}

func TestJWKFromKeyPublicAndPrivate(t *testing.T) {
	t.Run("RSA private carries all CRT params", func(t *testing.T) {
		priv, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)

		jwk, err := JWKFromKey(priv, "k")
		require.NoError(t, err)
		assert.Equal(t, "RSA", jwk.Kty)
		assert.NotEmpty(t, jwk.D)
		assert.NotEmpty(t, jwk.P)
		assert.NotEmpty(t, jwk.Q)
		assert.NotEmpty(t, jwk.Dp)

		recovered, err := jwk.PrivateKey()
		require.NoError(t, err)
		rp, ok := recovered.(*rsa.PrivateKey)
		require.True(t, ok)
		assert.True(t, priv.Equal(rp))
	})

	t.Run("public JWK strips private material", func(t *testing.T) {
		priv, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)
		jwk, err := JWKFromKey(priv, "k")
		require.NoError(t, err)

		pub := jwk.PublicJWK()
		assert.Empty(t, pub.D)
		assert.Empty(t, pub.P)
		assert.NotEmpty(t, pub.N) // public material retained
	})

	t.Run("oct symmetric", func(t *testing.T) {
		secret := []byte("0123456789abcdef0123456789abcdef")
		jwk, err := JWKFromKey(secret, "k")
		require.NoError(t, err)
		assert.Equal(t, "oct", jwk.Kty)

		back, err := jwk.PrivateKey()
		require.NoError(t, err)
		assert.Equal(t, secret, back)
	})

	t.Run("unsupported key type", func(t *testing.T) {
		_, err := JWKFromKey("not a key", "k")
		require.Error(t, err)
	})
}

func TestJWKSSetOperations(t *testing.T) {
	set := &JWKS{}
	assert.Equal(t, 0, set.Len())

	set.AddKey(JSONWebKey{Kty: "RSA", Kid: "a", N: "n", E: "AQAB"})
	set.AddKey(JSONWebKey{Kty: "RSA", Kid: "b", N: "n", E: "AQAB"})
	assert.Equal(t, 2, set.Len())

	got, ok := set.LookupKeyID("b")
	require.True(t, ok)
	assert.Equal(t, "b", got.Kid)

	_, ok = set.LookupKeyID("missing")
	assert.False(t, ok)

	assert.True(t, set.RemoveKey("a"))
	assert.Equal(t, 1, set.Len())
	assert.False(t, set.RemoveKey("a"), "removing an absent kid reports false")

	_, ok = set.LookupKeyID("a")
	assert.False(t, ok)
}

func TestJWKMetadataFields(t *testing.T) {
	jwk := JSONWebKey{
		Kty:     "RSA",
		Kid:     "k1",
		Use:     "sig",
		KeyOps:  []string{"verify"},
		X5u:     "https://example.com/certs",
		X5c:     []string{"MIIBcert"},
		X5t:     "thumb160",
		X5tS256: "thumb256",
		N:       "abc",
		E:       "AQAB",
	}

	data, err := json.Marshal(jwk)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"key_ops":["verify"]`)
	assert.Contains(t, string(data), `"x5t#S256":"thumb256"`)
	assert.Contains(t, string(data), `"x5c":["MIIBcert"]`)

	var back JSONWebKey
	require.NoError(t, json.Unmarshal(data, &back))
	assert.Equal(t, jwk.X5tS256, back.X5tS256)
	assert.Equal(t, jwk.KeyOps, back.KeyOps)
	assert.Equal(t, jwk.X5c, back.X5c)
}

func TestMarshalKeyToPEM(t *testing.T) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	privPEM, err := MarshalPrivateKeyToPEM(priv)
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(string(privPEM), "-----BEGIN PRIVATE KEY-----"))

	pubPEM, err := MarshalPublicKeyToPEM(&priv.PublicKey)
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(string(pubPEM), "-----BEGIN PUBLIC KEY-----"))

	// Round-trip through the PEM parsers.
	gotPriv, err := ParseECPrivateKeyFromPEM(privPEM)
	require.NoError(t, err)
	assert.True(t, priv.Equal(gotPriv))

	gotPub, err := ParseECPublicKeyFromPEM(pubPEM)
	require.NoError(t, err)
	assert.True(t, priv.PublicKey.Equal(gotPub))
}

// TestJWKFromPublicKeys exercises the public-key branches of jwkFromKey.
func TestJWKFromPublicKeys(t *testing.T) {
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	ecKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	edPub, _, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	secpKey, err := secp256k1.GeneratePrivateKey()
	require.NoError(t, err)

	t.Run("RSA public", func(t *testing.T) {
		jwk, err := JWKFromKey(&rsaKey.PublicKey, "k")
		require.NoError(t, err)
		assert.Equal(t, "RSA", jwk.Kty)
		assert.Empty(t, jwk.D)
	})

	t.Run("EC public", func(t *testing.T) {
		jwk, err := JWKFromKey(&ecKey.PublicKey, "k")
		require.NoError(t, err)
		assert.Equal(t, "EC", jwk.Kty)
	})

	t.Run("Ed25519 public", func(t *testing.T) {
		jwk, err := JWKFromKey(edPub, "k")
		require.NoError(t, err)
		assert.Equal(t, "OKP", jwk.Kty)
	})

	t.Run("secp256k1 public", func(t *testing.T) {
		jwk, err := JWKFromKey(&secpKey.Pub, "k")
		require.NoError(t, err)
		assert.Equal(t, "secp256k1", jwk.Crv)
	})

	t.Run("oct", func(t *testing.T) {
		jwk, err := JWKFromKey([]byte("secret-bytes"), "k")
		require.NoError(t, err)
		assert.Equal(t, "oct", jwk.Kty)
	})
}
