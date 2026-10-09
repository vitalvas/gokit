package xjwt

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// b64 is shorthand for raw-url base64 used to hand-craft tokens.
func b64(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

// dotJoin assembles compact-JWS segments without string '+'.
func dotJoin(parts ...string) string { return strings.Join(parts, ".") }

func TestDecodeHeaderAndPayload(t *testing.T) {
	priv, _ := signerFor(t, "ES256")
	token, err := Sign("ES256", "k1", testClaims{Scope: "openid"}, priv)
	require.NoError(t, err)

	h, err := DecodeHeader(token)
	require.NoError(t, err)
	assert.Equal(t, "ES256", h.Alg)
	assert.Equal(t, "k1", h.Kid)

	payload, err := DecodePayloadUnverified(token)
	require.NoError(t, err)
	assert.Contains(t, string(payload), "openid")

	_, err = DecodePayloadUnverified("only.two")
	require.Error(t, err)
}

// --- Adversarial suite: these must error, never panic, never accept. ---

func TestRejectAlgNone(t *testing.T) {
	header := b64([]byte(`{"alg":"none","typ":"JWT"}`))
	payload := b64([]byte(`{"sub":"attacker"}`))
	token := dotJoin(header, payload, "")

	resolve := func(_ Header) (any, error) { return nil, nil }

	for _, allow := range [][]string{{"none"}, {"RS256"}, {}} {
		_, err := Verify(token, resolve, allow)
		require.Error(t, err, "alg:none must be rejected for allowlist %v", allow)
	}
}

func TestRejectUnsupportedJOSEHeaders(t *testing.T) {
	priv, pub := signerFor(t, "ES256")
	resolve := func(_ Header) (any, error) { return pub, nil }
	payload := b64([]byte(`{"sub":"user-1"}`))

	cases := map[string]map[string]any{
		"critical header": {
			"alg":  "ES256",
			"typ":  "JWT",
			"crit": []string{"b64"},
			"b64":  true,
		},
		"unencoded payload header": {
			"alg": "ES256",
			"typ": "JWT",
			"b64": false,
		},
	}

	for name, header := range cases {
		t.Run(name, func(t *testing.T) {
			headerJSON, err := json.Marshal(header)
			require.NoError(t, err)

			signingInput := dotJoin(b64(headerJSON), payload)
			sig, err := signPayload("ES256", signingInput, priv)
			require.NoError(t, err)

			_, err = Verify(dotJoin(signingInput, b64(sig)), resolve, []string{"ES256"})
			require.Error(t, err)
		})
	}
}

func TestRejectAlgNotInAllowlist(t *testing.T) {
	priv, pub := signerFor(t, "ES256")
	token, err := Sign("ES256", "k1", testClaims{Scope: "x"}, priv)
	require.NoError(t, err)

	resolve := func(_ Header) (any, error) { return pub, nil }

	_, err = Verify(token, resolve, []string{"RS256"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not in the allowed set")
}

func TestRejectRSHSConfusion(t *testing.T) {
	// Attacker signs HS256 using the RSA public key bytes as the HMAC secret,
	// hoping the verifier treats the public key as a shared secret. The
	// allowlist and key-type binding must both block this.
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	pubDER, err := x509.MarshalPKIXPublicKey(&rsaKey.PublicKey)
	require.NoError(t, err)

	header := b64([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payload := b64([]byte(`{"sub":"attacker"}`))
	signingInput := dotJoin(header, payload)
	mac := hmac.New(sha256.New, pubDER)
	mac.Write([]byte(signingInput))
	forged := dotJoin(signingInput, b64(mac.Sum(nil)))

	resolve := func(_ Header) (any, error) { return &rsaKey.PublicKey, nil }

	// RS256-only allowlist rejects the HS256 token outright.
	_, err = Verify(forged, resolve, []string{"RS256"})
	require.Error(t, err)

	// Even if HS256 is allowed, the resolved *rsa.PublicKey is not a []byte
	// secret, so key-type binding rejects it.
	_, err = Verify(forged, resolve, []string{"HS256"})
	require.Error(t, err)
}

func TestRejectTamperedPayload(t *testing.T) {
	priv, pub := signerFor(t, "ES256")
	token, err := Sign("ES256", "k1", testClaims{
		RegisteredClaims: RegisteredClaims{Subject: "user-1"},
	}, priv)
	require.NoError(t, err)

	parts := strings.Split(token, ".")
	parts[1] = b64([]byte(`{"sub":"admin"}`))
	tampered := strings.Join(parts, ".")

	resolve := func(_ Header) (any, error) { return pub, nil }
	_, err = Verify(tampered, resolve, []string{"ES256"})
	require.Error(t, err)
}

func TestRejectTamperedSignature(t *testing.T) {
	priv, pub := signerFor(t, "ES256")
	token, err := Sign("ES256", "k1", testClaims{Scope: "x"}, priv)
	require.NoError(t, err)

	parts := strings.Split(token, ".")
	sig, _ := base64.RawURLEncoding.DecodeString(parts[2])
	sig[0] ^= 0xff
	parts[2] = b64(sig)
	tampered := strings.Join(parts, ".")

	resolve := func(_ Header) (any, error) { return pub, nil }
	_, err = Verify(tampered, resolve, []string{"ES256"})
	require.Error(t, err)
}

func TestRejectMalformedTokens(t *testing.T) {
	_, pub := signerFor(t, "ES256")
	resolve := func(_ Header) (any, error) { return pub, nil }

	cases := map[string]string{
		"two segments":      "aaa.bbb",
		"four segments":     "a.b.c.d",
		"empty":             "",
		"bad base64 header": dotJoin("!!!", b64([]byte("{}")), "sig"),
		"non-json header":   dotJoin(b64([]byte("not-json")), b64([]byte("{}")), "sig"),
		"bad base64 sig":    dotJoin(b64([]byte(`{"alg":"ES256"}`)), b64([]byte("{}")), "!!!"),
	}

	for name, token := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := Verify(token, resolve, []string{"ES256"})
			require.Error(t, err)
		})
	}
}

func TestRejectWrongKeyType(t *testing.T) {
	priv, _ := signerFor(t, "ES256")
	token, err := Sign("ES256", "k1", testClaims{Scope: "x"}, priv)
	require.NoError(t, err)

	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	resolve := func(_ Header) (any, error) { return &rsaKey.PublicKey, nil }

	_, err = Verify(token, resolve, []string{"ES256"})
	require.Error(t, err)
}

func TestRejectWrongCurve(t *testing.T) {
	priv, _ := signerFor(t, "ES256")
	token, err := Sign("ES256", "k1", testClaims{Scope: "x"}, priv)
	require.NoError(t, err)

	wrong, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	require.NoError(t, err)
	resolve := func(_ Header) (any, error) { return &wrong.PublicKey, nil }

	_, err = Verify(token, resolve, []string{"ES256"})
	require.Error(t, err)
}

func TestVerifyNeverPanics(t *testing.T) {
	_, pub := signerFor(t, "ES256")
	resolve := func(_ Header) (any, error) { return pub, nil }

	garbage := []string{
		"", ".", "..", "...", "a.b.c",
		strings.Repeat("A", 5000),
		dotJoin(b64([]byte(`{"alg":"ES256"}`)), "", ""),
		dotJoin(b64([]byte(`{}`)), b64([]byte(`{}`)), b64([]byte("short"))),
	}

	for _, g := range garbage {
		assert.NotPanics(t, func() {
			_, _ = Verify(g, resolve, []string{"ES256"})
		})
	}
}

func TestResolverErrorRejects(t *testing.T) {
	priv, _ := signerFor(t, "ES256")
	token, err := Sign("ES256", "k1", testClaims{Scope: "x"}, priv)
	require.NoError(t, err)

	resolve := func(_ Header) (any, error) {
		return nil, errTestResolver
	}
	_, err = Verify(token, resolve, []string{"ES256"})
	require.ErrorIs(t, err, errTestResolver)
}

var errTestResolver = &resolverError{}

type resolverError struct{}

func (*resolverError) Error() string { return "no key" }

func TestVerifyWithJWKS(t *testing.T) {
	priv, pub := signerFor(t, "ES256")
	ecPub := pub.(*ecdsa.PublicKey)

	token, err := Sign("ES256", "kid-9", testClaims{
		RegisteredClaims: RegisteredClaims{Subject: "u"},
	}, priv)
	require.NoError(t, err)

	xb, yb := ecCoords(t, ecPub)

	set := &JWKS{Keys: []JSONWebKey{{
		Kty: "EC",
		Crv: "P-256",
		Kid: "kid-9",
		Alg: "ES256",
		X:   b64(xb),
		Y:   b64(yb),
	}}}

	payload, err := VerifyWithJWKS(token, set, []string{"ES256"})
	require.NoError(t, err)

	var claims map[string]any
	require.NoError(t, json.Unmarshal(payload, &claims))
	assert.Equal(t, "u", claims["sub"])
}

// TestVerifyWithJWKSPS256 is a regression test: an RSA JWK must resolve for a
// PS* signature, not only RS* (matchesAlg previously matched only the "RS"
// prefix).
func TestVerifyWithJWKSPS256(t *testing.T) {
	key, jwk, err := GenerateKey(PS256, "kid-ps")
	require.NoError(t, err)

	token, err := Sign(PS256, "kid-ps", MapClaims{"sub": "ps-user"}, key)
	require.NoError(t, err)

	pubJWK := jwk.PublicJWK()
	pubJWK.Alg = PS256
	set := &JWKS{Keys: []JSONWebKey{pubJWK}}

	payload, err := VerifyWithJWKS(token, set, []string{PS256})
	require.NoError(t, err)
	assert.Contains(t, string(payload), `"sub":"ps-user"`)
}

// TestVerifyWithJWKSKidBinding is a regression test: a token that names a kid
// must only be verified by a key with that exact kid, never by a keyless or
// differently-identified key in the set.
func TestVerifyWithJWKSKidBinding(t *testing.T) {
	key, jwk, err := GenerateKey(ES256, "")
	require.NoError(t, err)
	token, err := Sign(ES256, "wanted-kid", MapClaims{"sub": "u"}, key)
	require.NoError(t, err)

	t.Run("keyless key does not satisfy a kid'd token", func(t *testing.T) {
		set := &JWKS{Keys: []JSONWebKey{jwk.PublicJWK()}} // kid == ""
		_, err := VerifyWithJWKS(token, set, []string{ES256})
		require.Error(t, err)
	})

	t.Run("wrong-kid key does not satisfy", func(t *testing.T) {
		wrong := jwk.PublicJWK()
		wrong.Kid = "other-kid"
		set := &JWKS{Keys: []JSONWebKey{wrong}}
		_, err := VerifyWithJWKS(token, set, []string{ES256})
		require.Error(t, err)
	})

	t.Run("matching-kid key verifies", func(t *testing.T) {
		match := jwk.PublicJWK()
		match.Kid = "wanted-kid"
		set := &JWKS{Keys: []JSONWebKey{match}}
		_, err := VerifyWithJWKS(token, set, []string{ES256})
		require.NoError(t, err)
	})

	t.Run("kid-less token still matches a compatible key", func(t *testing.T) {
		tok, err := Sign(ES256, "", MapClaims{"sub": "u"}, key)
		require.NoError(t, err)
		set := &JWKS{Keys: []JSONWebKey{jwk.PublicJWK()}}
		_, err = VerifyWithJWKS(tok, set, []string{ES256})
		require.NoError(t, err)
	})
}

func TestVerifyWithJWKSSkipsEncryptionKeys(t *testing.T) {
	priv, pub := signerFor(t, "ES256")
	ecPub := pub.(*ecdsa.PublicKey)

	token, err := Sign("ES256", "kid-9", testClaims{
		RegisteredClaims: RegisteredClaims{Subject: "u"},
	}, priv)
	require.NoError(t, err)

	xb, yb := ecCoords(t, ecPub)

	set := &JWKS{Keys: []JSONWebKey{
		{
			Kty: "EC",
			Use: "enc",
			Crv: "P-256",
			Kid: "kid-9",
			Alg: "ES256",
			X:   b64(xb),
			Y:   b64(yb),
		},
		{
			Kty: "EC",
			Use: "sig",
			Crv: "P-256",
			Kid: "kid-9",
			Alg: "ES256",
			X:   b64(xb),
			Y:   b64(yb),
		},
	}}

	_, err = VerifyWithJWKS(token, set, []string{"ES256"})
	require.NoError(t, err)

	set.Keys = set.Keys[:1]
	_, err = VerifyWithJWKS(token, set, []string{"ES256"})
	require.Error(t, err)
}

func TestVerifyWithJWKSSkipsWrongCurveKeys(t *testing.T) {
	priv, pub := signerFor(t, "ES256")
	ecPub := pub.(*ecdsa.PublicKey)

	token, err := Sign("ES256", "", testClaims{
		RegisteredClaims: RegisteredClaims{Subject: "u"},
	}, priv)
	require.NoError(t, err)

	wrongPriv, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	require.NoError(t, err)
	wrongX, wrongY := ecCoords(t, &wrongPriv.PublicKey)
	xb, yb := ecCoords(t, ecPub)

	set := &JWKS{Keys: []JSONWebKey{
		{
			Kty: "EC",
			Crv: "P-384",
			X:   b64(wrongX),
			Y:   b64(wrongY),
		},
		{
			Kty: "EC",
			Crv: "P-256",
			X:   b64(xb),
			Y:   b64(yb),
		},
	}}

	_, err = VerifyWithJWKS(token, set, []string{"ES256"})
	require.NoError(t, err)
}

func TestSplitCompact(t *testing.T) {
	t.Run("exact count splits", func(t *testing.T) {
		parts, ok := splitCompact("a.b.c", 3)
		require.True(t, ok)
		assert.Equal(t, []string{"a", "b", "c"}, parts)

		five, ok := splitCompact("a.b.c.d.e", 5)
		require.True(t, ok)
		assert.Equal(t, []string{"a", "b", "c", "d", "e"}, five)
	})

	t.Run("empty segments preserved", func(t *testing.T) {
		parts, ok := splitCompact("a..c", 3)
		require.True(t, ok)
		assert.Equal(t, []string{"a", "", "c"}, parts)
	})

	t.Run("wrong count rejected", func(t *testing.T) {
		_, ok := splitCompact("a.b", 3)
		assert.False(t, ok)
		_, ok = splitCompact("a.b.c.d", 3)
		assert.False(t, ok)
		_, ok = splitCompact("abc", 3)
		assert.False(t, ok)
	})
}

// TestSplitCompactDotFlood is the regression test for the golang-jwt
// GHSA-mh63-6h87-95cp memory-amplification class: a token padded with a flood of
// dots must be rejected without allocating a slice proportional to its length.
func TestSplitCompactDotFlood(t *testing.T) {
	flood := fmt.Sprintf("x%s", strings.Repeat(".", 2_000_000))

	// Rejected at every public entry point.
	_, err := Verify(flood, func(Header) (any, error) { return []byte("k"), nil }, []string{HS256})
	require.Error(t, err)
	_, err = DecodePayloadUnverified(flood)
	require.Error(t, err)
	_, err = DecryptWithOptions(flood, []byte("k"), DecryptOptions{})
	require.Error(t, err)

	// The rejection must not allocate an O(n) slice: splitCompact returns before
	// materialising segments when the separator count is wrong.
	allocs := testing.AllocsPerRun(10, func() {
		_, _ = splitCompact(flood, 3)
	})
	assert.LessOrEqual(t, allocs, 1.0, "dot-flood rejection must not allocate per-segment")
}

func TestVerifyWrongKeyForAlg(t *testing.T) {
	// Verify returns ErrKeyTypeMismatch when the resolver returns an
	// incompatible key for the header alg.
	ec, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	tok, err := Sign(ES256, "k", MapClaims{"sub": "u"}, ec)
	require.NoError(t, err)

	_, err = Verify(tok, func(Header) (any, error) { return []byte("hmac-secret"), nil }, []string{ES256})
	require.ErrorIs(t, err, ErrKeyTypeMismatch)
}

func TestVerifyBadSignatureES256KAndEdDSA(t *testing.T) {
	for _, alg := range []string{ES256K, EdDSA, ES256, RS256} {
		t.Run(alg, func(t *testing.T) {
			key, jwk, err := GenerateKey(alg, "k")
			require.NoError(t, err)
			token, err := Sign(alg, "k", MapClaims{"sub": "u"}, key)
			require.NoError(t, err)

			// Corrupt the signature segment.
			bad := fmt.Sprintf("%sAAAA", token[:len(token)-4])
			pub, err := jwk.PublicJWK().PublicKey(alg)
			require.NoError(t, err)
			_, err = Verify(bad, func(Header) (any, error) { return pub, nil }, []string{alg})
			require.ErrorIs(t, err, ErrTokenSignatureInvalid)
		})
	}
}

func TestECDSASignCurveMismatch(t *testing.T) {
	// signECDSA rejects a key whose curve size does not match the alg.
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	_, err = Sign(ES384, "k", MapClaims{"sub": "x"}, k)
	require.ErrorIs(t, err, ErrKeyTypeMismatch)
}
