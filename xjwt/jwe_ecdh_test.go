package xjwt

import (
	"crypto/ecdh"
	"crypto/rand"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestECDHCurveByteLenAndName(t *testing.T) {
	// Exercise P-384 and P-521 branches of curveByteLen and ecdhCurveFromKey.
	for _, c := range []struct {
		curve ecdh.Curve
		crv   string
		blen  int
	}{
		{ecdh.P384(), "P-384", 48},
		{ecdh.P521(), "P-521", 66},
	} {
		name, err := ecdhCurveFromKey(c.curve)
		require.NoError(t, err)
		assert.Equal(t, c.crv, name)
		assert.Equal(t, c.blen, curveByteLen(c.crv))
	}
}

func TestEncryptDecryptP384ECDH(t *testing.T) {
	// Drives the P-384 ECDH path end to end (curveByteLen/ecdhCurveFromKey).
	priv, err := ecdh.P384().GenerateKey(rand.Reader)
	require.NoError(t, err)

	token, err := Encrypt(ECDHES, A128GCM, priv.PublicKey(), []byte("p384"), EncryptOptions{})
	require.NoError(t, err)
	got, err := Decrypt(token, priv)
	require.NoError(t, err)
	assert.Equal(t, []byte("p384"), got)
}

func TestToECDHUnsupportedTypes(t *testing.T) {
	_, err := toECDHPublic("not a key")
	require.Error(t, err)
	_, err = toECDHPrivate("not a key")
	require.Error(t, err)
}

func TestECDHCurveUnsupported(t *testing.T) {
	_, err := ecdhCurve("P-999")
	require.Error(t, err)
}

func TestECDHPublicFromJWKBadBase64(t *testing.T) {
	_, err := ecdhPublicFromJWK(&JSONWebKey{Crv: "P-256", X: "!!!", Y: "!!!"})
	require.Error(t, err)
}

func TestDecryptECDHBadAPU(t *testing.T) {
	// Build an ECDH-ES token, then corrupt apu/apv to invalid base64.
	priv, err := ecdhP256Key()
	require.NoError(t, err)
	token, err := Encrypt(ECDHES, A128GCM, priv.PublicKey(), []byte("x"),
		EncryptOptions{APU: []byte("Alice")})
	require.NoError(t, err)

	bad := reheader(t, token, func(h *jweHeader) { h.Apu = "!!!" })
	_, err = Decrypt(bad, priv)
	require.Error(t, err)

	badV := reheader(t, token, func(h *jweHeader) { h.Apv = "!!!" })
	_, err = Decrypt(badV, priv)
	require.Error(t, err)
}

func TestECDHP521RoundTrip(t *testing.T) {
	priv, err := ecdh.P521().GenerateKey(rand.Reader)
	require.NoError(t, err)

	token, err := Encrypt(ECDHESA256, A256GCM, priv.PublicKey(), []byte("p521"), EncryptOptions{})
	require.NoError(t, err)
	got, err := Decrypt(token, priv)
	require.NoError(t, err)
	assert.Equal(t, []byte("p521"), got)
}

func TestSignECWithSignerWrongCurveName(t *testing.T) {
	// ecdhCurveFromKey / curveByteLen already covered for P-384/521; ensure the
	// unsupported-curve path of ecdhCurve is hit via a bad epk crv.
	_, err := ecdhCurve("bogus")
	require.Error(t, err)
}

func TestECDHHelperBranches(t *testing.T) {
	t.Run("toECDHPublic unsupported type", func(t *testing.T) {
		_, err := toECDHPublic("not a key")
		require.Error(t, err)
	})

	t.Run("toECDHPrivate unsupported type", func(t *testing.T) {
		_, err := toECDHPrivate("not a key")
		require.Error(t, err)
	})

	t.Run("toECDHPublic accepts ecdh.PublicKey", func(t *testing.T) {
		k, err := ecdh.P256().GenerateKey(rand.Reader)
		require.NoError(t, err)
		_, err = toECDHPublic(k.PublicKey())
		require.NoError(t, err)
	})

	t.Run("toECDHPrivate accepts ecdh.PrivateKey", func(t *testing.T) {
		k, err := ecdh.P256().GenerateKey(rand.Reader)
		require.NoError(t, err)
		_, err = toECDHPrivate(k)
		require.NoError(t, err)
	})

	t.Run("ecdhCurve unsupported", func(t *testing.T) {
		_, err := ecdhCurve("P-999")
		require.Error(t, err)
	})

	t.Run("ecdhPublicFromJWK bad base64", func(t *testing.T) {
		_, err := ecdhPublicFromJWK(&JSONWebKey{Crv: "P-256", X: badB64, Y: badB64})
		require.Error(t, err)
	})
}

func TestECDHEphemeralKeyValidation(t *testing.T) {
	for _, curve := range []ecdh.Curve{ecdh.P256(), ecdh.P384(), ecdh.P521(), ecdh.X25519()} {
		priv, err := curve.GenerateKey(rand.Reader)
		require.NoError(t, err)
		good, err := ecdhPublicJWK(priv.PublicKey())
		require.NoError(t, err)
		parsed, err := ecdhPublicFromJWK(good)
		require.NoError(t, err)
		assert.True(t, parsed.Equal(priv.PublicKey()))
		for _, coordinate := range []string{"x", "y"} {
			for _, length := range []int{0, 1, curveByteLen(good.Crv) - 1, curveByteLen(good.Crv) + 1, 256} {
				t.Run(fmt.Sprintf("%s/%s/%d", good.Crv, coordinate, length), func(t *testing.T) {
					bad := *good
					if coordinate == "x" {
						bad.X = b64(make([]byte, length))
					} else {
						bad.Y = b64(make([]byte, length))
					}
					if good.Crv == "X25519" && coordinate == "y" && length == 0 {
						return
					}
					require.NotPanics(t, func() { _, err := ecdhPublicFromJWK(&bad); assert.Error(t, err) })
					header := protectedJSON(t, jweHeader{Alg: ECDHES, Enc: A128GCM, Epk: &bad})
					token := strings.Join([]string{header, "", b64(make([]byte, 12)), "", b64(make([]byte, 16))}, ".")
					require.NotPanics(t, func() { _, err := Decrypt(token, priv); assert.Error(t, err) })
				})
			}
		}
		for _, mut := range []func(*JSONWebKey){func(k *JSONWebKey) { k.Kty = "RSA" }, func(k *JSONWebKey) { k.D = "AQ" }} {
			bad := *good
			mut(&bad)
			_, err := ecdhPublicFromJWK(&bad)
			assert.Error(t, err)
		}
	}
	_, err := ecdhPublicFromJWK(nil)
	assert.Error(t, err)
}

func FuzzECDHEphemeralCoordinates(f *testing.F) {
	f.Add("P-256", "EC", make([]byte, 66), make([]byte, 66))
	f.Add("X25519", "OKP", make([]byte, 32), []byte{})
	f.Fuzz(func(_ *testing.T, curve, kind string, x, y []byte) {
		_, _ = ecdhPublicFromJWK(&JSONWebKey{Crv: curve, Kty: kind, X: b64(x), Y: b64(y)})
	})
}
