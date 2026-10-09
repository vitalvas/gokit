package xjwt

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vitalvas/gokit/secp256k1"
)

// badB64 is a string that is not valid base64url (contains padding/invalid char).
const badB64 = "!!!not-base64!!!"

func ecJWK(t *testing.T, crv string, curve elliptic.Curve, alg string) (JSONWebKey, any) {
	t.Helper()
	k, err := ecdsa.GenerateKey(curve, rand.Reader)
	require.NoError(t, err)

	xb, yb := ecCoords(t, &k.PublicKey)

	jwk := JSONWebKey{
		Kty: "EC",
		Crv: crv,
		Alg: alg,
		X:   base64.RawURLEncoding.EncodeToString(xb),
		Y:   base64.RawURLEncoding.EncodeToString(yb),
	}

	return jwk, &k.PublicKey
}

func TestJWKPublicKey(t *testing.T) {
	t.Run("RSA", func(t *testing.T) {
		k, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)
		jwk := JSONWebKey{
			Kty: "RSA",
			Alg: "RS256",
			N:   base64.RawURLEncoding.EncodeToString(k.N.Bytes()),
			E:   base64.RawURLEncoding.EncodeToString([]byte{0x01, 0x00, 0x01}),
		}
		pub, err := jwk.PublicKey("RS256")
		require.NoError(t, err)
		assert.IsType(t, &rsa.PublicKey{}, pub)
	})

	t.Run("EC curves", func(t *testing.T) {
		cases := []struct {
			crv   string
			curve elliptic.Curve
			alg   string
		}{
			{"P-256", elliptic.P256(), "ES256"},
			{"P-384", elliptic.P384(), "ES384"},
			{"P-521", elliptic.P521(), "ES512"},
		}
		for _, tc := range cases {
			jwk, _ := ecJWK(t, tc.crv, tc.curve, tc.alg)
			pub, err := jwk.PublicKey(tc.alg)
			require.NoError(t, err, tc.crv)
			assert.IsType(t, &ecdsa.PublicKey{}, pub)
		}
	})

	t.Run("OKP Ed25519", func(t *testing.T) {
		pub, _, err := ed25519.GenerateKey(rand.Reader)
		require.NoError(t, err)
		jwk := JSONWebKey{
			Kty: "OKP",
			Crv: "Ed25519",
			Alg: "EdDSA",
			X:   base64.RawURLEncoding.EncodeToString(pub),
		}
		got, err := jwk.PublicKey("EdDSA")
		require.NoError(t, err)
		assert.IsType(t, ed25519.PublicKey{}, got)
	})

	t.Run("secp256k1", func(t *testing.T) {
		k, err := secp256k1.GeneratePrivateKey()
		require.NoError(t, err)
		ser := k.Pub.SerializeUncompressed()
		jwk := JSONWebKey{
			Kty: "EC",
			Crv: "secp256k1",
			Alg: "ES256K",
			X:   base64.RawURLEncoding.EncodeToString(ser[1:33]),
			Y:   base64.RawURLEncoding.EncodeToString(ser[33:]),
		}
		got, err := jwk.PublicKey("ES256K")
		require.NoError(t, err)
		assert.IsType(t, &secp256k1.PublicKey{}, got)
	})

	t.Run("kty/alg mismatch rejected", func(t *testing.T) {
		jwk := JSONWebKey{
			Kty: "RSA",
			N:   "AAAA",
			E:   "AQAB",
		}
		_, err := jwk.PublicKey("ES256")
		require.Error(t, err)
	})

	t.Run("unsupported curve rejected", func(t *testing.T) {
		jwk := JSONWebKey{
			Kty: "EC",
			Crv: "P-999",
			X:   "AAAA",
			Y:   "AAAA",
		}
		_, err := jwk.PublicKey("ES256")
		require.Error(t, err)
	})

	t.Run("small RSA modulus rejected", func(t *testing.T) {
		jwk := JSONWebKey{
			Kty: "RSA",
			Alg: "RS256",
			N:   "AQAB",
			E:   "AQAB",
		}
		_, err := jwk.PublicKey("RS256")
		require.Error(t, err)
	})
}

func TestParseJWKS(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		set, err := ParseJWKS([]byte(`{"keys":[{"kty":"EC","crv":"P-256","x":"AA","y":"BB"}]}`))
		require.NoError(t, err)
		assert.Len(t, set.Keys, 1)
	})

	t.Run("invalid json", func(t *testing.T) {
		_, err := ParseJWKS([]byte(`not json`))
		require.Error(t, err)
	})
}

func TestMatchesAlg(t *testing.T) {
	cases := []struct {
		kty  string
		crv  string
		alg  string
		want bool
	}{
		{"RSA", "", "RS256", true},
		{"RSA", "", "PS256", true},
		{"RSA", "", "PS512", true},
		{"EC", "P-256", "ES256", true},
		{"EC", "P-256", "ES384", false},
		{"EC", "P-384", "ES256", false},
		{"EC", "secp256k1", "ES256K", true},
		{"EC", "secp256k1", "ES256", false},
		{"OKP", "Ed25519", "EdDSA", true},
		{"OKP", "X25519", "EdDSA", false},
		{"RSA", "", "ES256", false},
		{"EC", "P-256", "EdDSA", false},
		{"EC", "", "ES256K", false},
		{"oct", "", "HS256", false},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, matchesAlg(tc.kty, tc.crv, tc.alg), "%s/%s/%s", tc.kty, tc.crv, tc.alg)
	}
}

func TestParseRSAPublicKeyExponentValidation(t *testing.T) {
	bigN := b64(new(big.Int).Lsh(big.NewInt(1), 2100).Bytes())

	t.Run("exponent zero", func(t *testing.T) {
		_, err := parseRSAPublicKey(bigN, b64([]byte{0}))
		require.Error(t, err)
	})
	t.Run("exponent even", func(t *testing.T) {
		_, err := parseRSAPublicKey(bigN, b64([]byte{4}))
		require.Error(t, err)
	})
	t.Run("exponent one", func(t *testing.T) {
		_, err := parseRSAPublicKey(bigN, b64([]byte{1}))
		require.Error(t, err)
	})
	t.Run("exponent too large", func(t *testing.T) {
		_, err := parseRSAPublicKey(bigN, b64(make([]byte, 5))) // > 31 bits
		require.Error(t, err)
	})
}

func TestParseECPublicKeyOffCurve(t *testing.T) {
	// Valid-length but off-curve coordinates (all 0x01) must be rejected.
	x := b64(append([]byte{1}, make([]byte, 31)...))
	_, err := parseECPublicKey("P-256", x, x)
	require.Error(t, err)
}

func TestPublicKeyUnsupportedKtyDirect(t *testing.T) {
	// matchesAlg passes (bogus alg path) is impossible, so call via a kty that
	// matchesAlg would accept for an RSA-shaped alg but with an unknown kty is not
	// reachable; instead exercise the PublicKey switch default through an AKP-like
	// mismatch is covered elsewhere. Here cover the EC secp256k1 dispatch.
	priv, _, err := GenerateKey(ES256K, "k")
	require.NoError(t, err)
	jwk, err := JWKFromKey(priv, "k")
	require.NoError(t, err)
	_, err = jwk.PublicJWK().PublicKey(ES256K)
	require.NoError(t, err)
}

func TestResolverFromJWKSAlgMismatch(t *testing.T) {
	// A JWK that declares a different alg than the token header must be skipped
	// (resolver kid matches, alg does not) -> no matching key.
	key, jwk, err := GenerateKey(ES256, "k1")
	require.NoError(t, err)
	token, err := Sign(ES256, "k1", MapClaims{"sub": "u"}, key)
	require.NoError(t, err)

	pub := jwk.PublicJWK()
	pub.Alg = ES384 // deliberately wrong declared alg
	set := &JWKS{Keys: []JSONWebKey{pub}}

	_, err = VerifyWithJWKS(token, set, []string{ES256})
	require.Error(t, err)
}

func TestJWKPublicKeyErrors(t *testing.T) {
	t.Run("kty mismatch for alg", func(t *testing.T) {
		_, err := JSONWebKey{Kty: "RSA"}.PublicKey(ES256)
		require.Error(t, err)
	})

	t.Run("bad RSA modulus base64", func(t *testing.T) {
		_, err := JSONWebKey{Kty: "RSA", N: badB64, E: "AQAB"}.PublicKey(RS256)
		require.Error(t, err)
	})

	t.Run("bad RSA exponent base64", func(t *testing.T) {
		_, err := JSONWebKey{Kty: "RSA", N: base64.RawURLEncoding.EncodeToString(make([]byte, 300)), E: badB64}.PublicKey(RS256)
		require.Error(t, err)
	})

	t.Run("RSA modulus too small", func(t *testing.T) {
		_, err := JSONWebKey{Kty: "RSA", N: base64.RawURLEncoding.EncodeToString([]byte{1, 2, 3}), E: "AQAB"}.PublicKey(RS256)
		require.Error(t, err)
	})

	t.Run("EC bad coord base64", func(t *testing.T) {
		_, err := JSONWebKey{Kty: "EC", Crv: "P-256", X: badB64, Y: badB64}.PublicKey(ES256)
		require.Error(t, err)
	})

	t.Run("EC unsupported curve", func(t *testing.T) {
		_, err := parseECPublicKey("P-999", "AA", "AA")
		require.Error(t, err)
	})

	t.Run("EC coordinate too large", func(t *testing.T) {
		big33 := base64.RawURLEncoding.EncodeToString(make([]byte, 40))
		_, err := parseECPublicKey("P-256", big33, big33)
		require.Error(t, err)
	})

	t.Run("OKP wrong curve", func(t *testing.T) {
		_, err := parseOKPPublicKey("X25519", "AA")
		require.Error(t, err)
	})

	t.Run("OKP bad base64", func(t *testing.T) {
		_, err := parseOKPPublicKey("Ed25519", badB64)
		require.Error(t, err)
	})

	t.Run("OKP wrong length", func(t *testing.T) {
		_, err := parseOKPPublicKey("Ed25519", base64.RawURLEncoding.EncodeToString([]byte{1, 2, 3}))
		require.Error(t, err)
	})

	t.Run("secp256k1 bad base64", func(t *testing.T) {
		_, err := parseSecp256k1PublicKey(badB64, badB64)
		require.Error(t, err)
	})

	t.Run("AKP missing alg", func(t *testing.T) {
		_, err := parseMLDSAPublicKey("not-an-alg", "AA")
		require.Error(t, err)
	})

	t.Run("AKP bad base64", func(t *testing.T) {
		_, err := parseMLDSAPublicKey(MLDSA44, badB64)
		require.Error(t, err)
	})

	t.Run("unsupported kty", func(t *testing.T) {
		_, err := JSONWebKey{Kty: "WHAT"}.PublicKey("WHAT")
		require.Error(t, err)
	})
}

func TestJWKPrivateKeyErrors(t *testing.T) {
	t.Run("unsupported kty", func(t *testing.T) {
		_, err := JSONWebKey{Kty: "WHAT"}.PrivateKey()
		require.Error(t, err)
	})

	t.Run("RSA not a private key", func(t *testing.T) {
		_, err := JSONWebKey{Kty: "RSA", N: "a", E: "AQAB"}.PrivateKey()
		require.Error(t, err)
	})

	t.Run("RSA bad private base64", func(t *testing.T) {
		n := base64.RawURLEncoding.EncodeToString(make([]byte, 300))
		jwk := JSONWebKey{
			Kty: "RSA",
			N:   n,
			E:   "AQAB",
			D:   badB64,
			P:   "AA",
			Q:   "AA",
		}
		_, err := jwk.PrivateKey()
		require.Error(t, err)
	})

	t.Run("EC not a private key", func(t *testing.T) {
		_, err := JSONWebKey{Kty: "EC", Crv: "P-256"}.PrivateKey()
		require.Error(t, err)
	})

	t.Run("EC bad private base64", func(t *testing.T) {
		_, err := JSONWebKey{Kty: "EC", Crv: "P-256", D: badB64}.PrivateKey()
		require.Error(t, err)
	})

	t.Run("EC unsupported curve", func(t *testing.T) {
		_, err := JSONWebKey{Kty: "EC", Crv: "P-999", D: "AA"}.PrivateKey()
		require.Error(t, err)
	})

	t.Run("OKP not a private key", func(t *testing.T) {
		_, err := JSONWebKey{Kty: "OKP", Crv: "Ed25519"}.PrivateKey()
		require.Error(t, err)
	})

	t.Run("OKP bad base64", func(t *testing.T) {
		_, err := JSONWebKey{Kty: "OKP", Crv: "Ed25519", D: badB64}.PrivateKey()
		require.Error(t, err)
	})

	t.Run("OKP wrong seed length", func(t *testing.T) {
		_, err := JSONWebKey{Kty: "OKP", Crv: "Ed25519", D: base64.RawURLEncoding.EncodeToString([]byte{1})}.PrivateKey()
		require.Error(t, err)
	})

	t.Run("AKP not a private key", func(t *testing.T) {
		_, err := JSONWebKey{Kty: "AKP", Alg: MLDSA44}.PrivateKey()
		require.Error(t, err)
	})

	t.Run("AKP missing alg", func(t *testing.T) {
		_, err := JSONWebKey{Kty: "AKP", Priv: "AA"}.PrivateKey()
		require.Error(t, err)
	})

	t.Run("AKP bad base64", func(t *testing.T) {
		_, err := JSONWebKey{Kty: "AKP", Alg: MLDSA44, Priv: badB64}.PrivateKey()
		require.Error(t, err)
	})

	t.Run("oct", func(t *testing.T) {
		k, err := JSONWebKey{Kty: "oct", K: base64.RawURLEncoding.EncodeToString([]byte("secret"))}.PrivateKey()
		require.NoError(t, err)
		assert.Equal(t, []byte("secret"), k)
	})
}

func TestJWKFromKeyUnsupported(t *testing.T) {
	_, err := JWKFromKey(12345, "k")
	require.Error(t, err)
}

func TestGenerateKeyUnsupportedAlg(t *testing.T) {
	_, _, err := GenerateKey("XX999", "k")
	require.Error(t, err)
}

func TestSecp256k1PrivateKeyFromJWK(t *testing.T) {
	// Exercises the secp256k1 branch of ecPrivateKey.
	priv, jwk, err := GenerateKey(ES256K, "k")
	require.NoError(t, err)
	back, err := jwk.PrivateKey()
	require.NoError(t, err)
	assert.IsType(t, priv, back)
}

func TestJWKSVerificationKeySelection(t *testing.T) {
	_, first, err := GenerateKey(ES256, "first")
	require.NoError(t, err)
	priv, second, err := GenerateKey(ES256, "second")
	require.NoError(t, err)
	set := &JWKS{Keys: []JSONWebKey{first.PublicJWK(), second.PublicJWK()}}
	for _, kid := range []string{"", "first", "second", "missing"} {
		token, err := Sign(ES256, kid, MapClaims{"sub": "u"}, priv)
		require.NoError(t, err)
		_, err = VerifyWithJWKS(token, set, []string{ES256})
		if kid == "" || kid == "second" {
			require.NoError(t, err)
		} else {
			require.Error(t, err)
		}
	}
	token, err := Sign(ES256, "second", MapClaims{}, priv)
	require.NoError(t, err)
	for _, ops := range [][]string{nil, {}, {"verify"}, {"sign"}, {"encrypt"}, {"decrypt", "verify"}} {
		key := second.PublicJWK()
		key.Use = ""
		key.KeyOps = ops
		_, err := VerifyWithJWKS(token, &JWKS{Keys: []JSONWebKey{key}}, []string{ES256})
		if ops == nil || algAllowed("verify", ops) {
			require.NoError(t, err)
		} else {
			require.Error(t, err)
		}
	}
}
