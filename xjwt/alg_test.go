package xjwt

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vitalvas/gokit/secp256k1"
)

// ecCoords extracts the affine X/Y coordinates of an EC public key as
// fixed-length big-endian bytes, using the non-deprecated crypto/ecdh encoding.
func ecCoords(t *testing.T, pub *ecdsa.PublicKey) (x, y []byte) {
	t.Helper()

	ek, err := pub.ECDH()
	require.NoError(t, err)

	raw := ek.Bytes() // 0x04 || X || Y
	byteLen := (len(raw) - 1) / 2

	return raw[1 : 1+byteLen], raw[1+byteLen:]
}

type testClaims struct {
	RegisteredClaims
	Scope string `json:"scope,omitempty"`
}

// signerFor returns a (private, public) key pair for each supported alg.
func signerFor(t *testing.T, alg string) (priv any, pub any) {
	t.Helper()

	switch alg {
	case "RS256", "RS384", "RS512", "PS256", "PS384", "PS512":
		k, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)

		return k, &k.PublicKey
	case "ES256":
		k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)

		return k, &k.PublicKey
	case "ES384":
		k, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
		require.NoError(t, err)

		return k, &k.PublicKey
	case "ES512":
		k, err := ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
		require.NoError(t, err)

		return k, &k.PublicKey
	case "EdDSA":
		pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
		require.NoError(t, err)

		return privKey, pubKey
	case "HS256", "HS384", "HS512":
		// RFC 7518 Section 3.2: the HMAC key must be at least the hash output
		// size (32/48/64 bytes for HS256/384/512).
		sizes := map[string]int{"HS256": 32, "HS384": 48, "HS512": 64}
		secret := make([]byte, sizes[alg])
		_, err := rand.Read(secret)
		require.NoError(t, err)

		return secret, secret
	case "ES256K":
		k, err := secp256k1.GeneratePrivateKey()
		require.NoError(t, err)

		return k, &k.Pub
	}

	t.Fatalf("unknown alg %s", alg)

	return nil, nil
}

func TestIsSupportedAlg(t *testing.T) {
	for _, alg := range []string{"RS256", "RS384", "RS512", "PS256", "PS384", "PS512", "ES256", "ES384", "ES512", "EdDSA", "HS256", "HS384", "HS512", "ES256K"} {
		assert.True(t, IsSupportedAlg(alg), alg)
	}

	for _, alg := range []string{"none", "", "HS1", "foo"} {
		assert.False(t, IsSupportedAlg(alg), alg)
	}
}

func TestSignVerifyRoundTrip(t *testing.T) {
	algs := []string{"RS256", "RS384", "RS512", "PS256", "PS384", "PS512", "ES256", "ES384", "ES512", "EdDSA", "HS256", "HS384", "HS512", "ES256K"}

	for _, alg := range algs {
		t.Run(alg, func(t *testing.T) {
			priv, pub := signerFor(t, alg)

			claims := testClaims{
				RegisteredClaims: RegisteredClaims{
					Issuer:   "https://sso.example.com",
					Subject:  "user-1",
					Audience: ClaimStrings{"app"},
				},
				Scope: "openid",
			}

			token, err := Sign(alg, "kid-1", claims, priv)
			require.NoError(t, err)

			resolve := func(_ Header) (any, error) { return pub, nil }
			payload, err := Verify(token, resolve, []string{alg})
			require.NoError(t, err)
			assert.Contains(t, string(payload), `"sub":"user-1"`)
		})
	}
}

func TestSignRejectsUnknownAlg(t *testing.T) {
	_, err := Sign("none", "k", testClaims{}, []byte("x"))
	require.Error(t, err)

	_, err = Sign("XX999", "k", testClaims{}, []byte("x"))
	require.Error(t, err)
}

func TestSignKeyTypeMismatch(t *testing.T) {
	_, err := Sign("RS256", "k", testClaims{}, []byte("not-an-rsa-key"))
	require.ErrorIs(t, err, ErrKeyTypeMismatch)
}

// TestHMACKeyTooSmall enforces RFC 7518 Section 3.2: the HMAC key MUST be at
// least the hash output size. A shorter key is rejected on both sign and verify.
func TestHMACKeyTooSmall(t *testing.T) {
	cases := map[string]int{HS256: 32, HS384: 48, HS512: 64}
	for alg, size := range cases {
		t.Run(alg, func(t *testing.T) {
			short := make([]byte, size-1)

			_, err := Sign(alg, "k", MapClaims{"sub": "u"}, short)
			require.ErrorIs(t, err, ErrKeyTooSmall)

			// A key of exactly the hash size is accepted, and verify also
			// rejects a short key handed to it by a resolver.
			ok := make([]byte, size)
			_, err = rand.Read(ok)
			require.NoError(t, err)
			tok, err := Sign(alg, "k", MapClaims{"sub": "u"}, ok)
			require.NoError(t, err)

			resolve := func(_ Header) (any, error) { return short, nil }
			_, err = Verify(tok, resolve, []string{alg})
			require.ErrorIs(t, err, ErrKeyTooSmall)
		})
	}
}

// TestRSAKeyTooSmall enforces RFC 7518 Section 3.3/3.5: RS*/PS* keys MUST be at
// least 2048 bits. A 1024-bit key is rejected on both sign and verify, and the
// verify guard holds even when the key arrives via a PEM-style resolver.
func TestRSAKeyTooSmall(t *testing.T) {
	small, err := rsa.GenerateKey(rand.Reader, 1024)
	require.NoError(t, err)

	for _, alg := range []string{RS256, PS256} {
		t.Run(alg, func(t *testing.T) {
			_, err := Sign(alg, "k", MapClaims{"sub": "u"}, small)
			require.ErrorIs(t, err, ErrKeyTooSmall)

			// Craft a well-formed token with a compliant key, then verify it
			// against the undersized public key to hit the verify-side guard.
			big, err := rsa.GenerateKey(rand.Reader, 2048)
			require.NoError(t, err)
			tok, err := Sign(alg, "k", MapClaims{"sub": "u"}, big)
			require.NoError(t, err)

			resolve := func(_ Header) (any, error) { return &small.PublicKey, nil }
			_, err = Verify(tok, resolve, []string{alg})
			require.ErrorIs(t, err, ErrKeyTooSmall)
		})
	}
}

func TestSignerNilKey(t *testing.T) {
	// A nil/non-signer value for an RSA/EC/Ed alg hits asSigner's error path.
	_, err := signRSAWithSigner(nil, algRegistry[RS256], "a.b")
	require.ErrorIs(t, err, ErrKeyTypeMismatch)
	_, err = signECWithSigner(nil, algRegistry[ES256], "a.b")
	require.ErrorIs(t, err, ErrKeyTypeMismatch)
	_, err = signEdWithSigner(nil, "a.b")
	require.ErrorIs(t, err, ErrKeyTypeMismatch)
}

// TestVerifyPayloadKeyTypeMismatch hits the per-family type-assertion failures in
// verifyPayload by handing each algorithm a wrong-typed key.
func TestVerifyPayloadKeyTypeMismatch(t *testing.T) {
	notAKey := "wrong"

	cases := []string{RS256, PS256, ES256, EdDSA, HS256, ES256K, MLDSA44}
	for _, alg := range cases {
		t.Run(alg, func(t *testing.T) {
			err := verifyPayload(alg, "a.b", []byte("sig"), notAKey)
			assert.ErrorIs(t, err, ErrKeyTypeMismatch)
		})
	}

	t.Run("unsupported alg", func(t *testing.T) {
		err := verifyPayload("BADALG", "a.b", nil, nil)
		require.Error(t, err)
	})
}

// TestSignPayloadKeyTypeMismatch hits the HMAC and secp256k1 type-assert failures
// (the RSA/EC/EdDSA ones fall through to the crypto.Signer path, covered
// elsewhere).
func TestSignPayloadKeyTypeMismatch(t *testing.T) {
	_, err := signPayload(HS256, "a.b", "not-bytes")
	assert.ErrorIs(t, err, ErrKeyTypeMismatch)

	_, err = signPayload(ES256K, "a.b", "not-secp")
	assert.ErrorIs(t, err, ErrKeyTypeMismatch)

	_, err = signPayload(MLDSA44, "a.b", "not-mldsa")
	assert.ErrorIs(t, err, ErrKeyTypeMismatch)

	_, err = signPayload("BADALG", "a.b", nil)
	require.Error(t, err)
}
