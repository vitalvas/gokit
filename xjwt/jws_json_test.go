package xjwt

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJWSFlattenedJSON(t *testing.T) {
	key, jwk, err := GenerateKey("ES256", "k1")
	require.NoError(t, err)
	pub, err := jwk.PublicJWK().PublicKey("ES256")
	require.NoError(t, err)

	payload := []byte("hello jws")
	resolve := func(_ Header) (any, error) { return pub, nil }

	data, err := SignFlattenedJSON(payload, SignInput{Alg: "ES256", Kid: "k1", Key: key})
	require.NoError(t, err)
	assert.Contains(t, string(data), `"signature"`)

	got, err := VerifyJSON(data, resolve, []string{"ES256"})
	require.NoError(t, err)
	assert.Equal(t, payload, got)

	t.Run("tampered payload rejected", func(t *testing.T) {
		var doc map[string]string
		require.NoError(t, json.Unmarshal(data, &doc))
		// Re-encode a different payload while keeping the original signature.
		doc["payload"] = base64.RawURLEncoding.EncodeToString([]byte("tampered"))
		corrupt, err := json.Marshal(doc)
		require.NoError(t, err)

		_, err = VerifyJSON(corrupt, resolve, []string{"ES256"})
		assert.ErrorIs(t, err, ErrTokenSignatureInvalid)
	})
}

func TestJWSGeneralMultiSignature(t *testing.T) {
	esKey, esJWK, err := GenerateKey("ES256", "es")
	require.NoError(t, err)
	rsKey, rsJWK, err := GenerateKey("RS256", "rs")
	require.NoError(t, err)

	esPub, err := esJWK.PublicJWK().PublicKey("ES256")
	require.NoError(t, err)
	rsPub, err := rsJWK.PublicJWK().PublicKey("RS256")
	require.NoError(t, err)

	payload := []byte("multi-signed")
	data, err := SignJSON(payload,
		SignInput{Alg: "ES256", Kid: "es", Key: esKey},
		SignInput{Alg: "RS256", Kid: "rs", Key: rsKey},
	)
	require.NoError(t, err)

	// Verifier that only knows the RSA key still succeeds via the RSA signature.
	rsOnly := func(h Header) (any, error) {
		if h.Kid == "rs" {
			return rsPub, nil
		}

		return nil, assertNoKey
	}
	got, err := VerifyJSON(data, rsOnly, []string{"ES256", "RS256"})
	require.NoError(t, err)
	assert.Equal(t, payload, got)

	// Verifier that only knows ES also succeeds.
	esOnly := func(h Header) (any, error) {
		if h.Kid == "es" {
			return esPub, nil
		}

		return nil, assertNoKey
	}
	_, err = VerifyJSON(data, esOnly, []string{"ES256", "RS256"})
	require.NoError(t, err)

	t.Run("none of the allowed algs matches", func(t *testing.T) {
		_, err := VerifyJSON(data, rsOnly, []string{"HS256"})
		assert.ErrorIs(t, err, ErrTokenSignatureInvalid)
	})
}

func TestJWSDetached(t *testing.T) {
	key, jwk, err := GenerateKey("EdDSA", "k1")
	require.NoError(t, err)
	pub, err := jwk.PublicJWK().PublicKey("EdDSA")
	require.NoError(t, err)

	payload := []byte("detached payload")
	resolve := func(_ Header) (any, error) { return pub, nil }

	data, err := SignDetachedJSON(payload, SignInput{Alg: "EdDSA", Kid: "k1", Key: key})
	require.NoError(t, err)
	assert.NotContains(t, string(data), `"payload"`)

	got, err := VerifyDetachedJSON(data, payload, resolve, []string{"EdDSA"})
	require.NoError(t, err)
	assert.Equal(t, payload, got)

	t.Run("wrong payload fails", func(t *testing.T) {
		_, err := VerifyDetachedJSON(data, []byte("different"), resolve, []string{"EdDSA"})
		assert.Error(t, err)
	})

	t.Run("embedded document rejected by detached verify", func(t *testing.T) {
		embedded, err := SignFlattenedJSON(payload, SignInput{Alg: "EdDSA", Key: key})
		require.NoError(t, err)
		_, err = VerifyDetachedJSON(embedded, payload, resolve, []string{"EdDSA"})
		assert.Error(t, err)
	})
}

var assertNoKey = errNoResolverKey{}

type errNoResolverKey struct{}

func (errNoResolverKey) Error() string { return "no key" }

func TestJWSJSONErrors(t *testing.T) {
	t.Run("SignJSON no inputs", func(t *testing.T) {
		_, err := SignJSON([]byte("p"))
		require.Error(t, err)
	})

	t.Run("signOne unsupported alg", func(t *testing.T) {
		_, err := SignFlattenedJSON([]byte("p"), SignInput{Alg: "BADALG", Key: []byte("k")})
		require.Error(t, err)
	})

	t.Run("VerifyJSON malformed", func(t *testing.T) {
		_, err := VerifyJSON([]byte("not json"), nil, []string{HS256})
		require.Error(t, err)
	})

	t.Run("VerifyJSON no signatures", func(t *testing.T) {
		_, err := VerifyJSON([]byte(`{"payload":"AA"}`), nil, []string{HS256})
		require.Error(t, err)
	})

	t.Run("VerifyJSON empty payload and none supplied", func(t *testing.T) {
		// Flattened doc with a signature but no payload, detached not provided.
		data := []byte(`{"protected":"AA","signature":"AA"}`)
		_, err := VerifyJSON(data, func(Header) (any, error) { return []byte("k"), nil }, []string{HS256})
		require.Error(t, err)
	})
}

func TestVerifyJSONSignatureBranches(t *testing.T) {
	key, jwk, err := GenerateKey(ES256, "k")
	require.NoError(t, err)
	pub, err := jwk.PublicJWK().PublicKey(ES256)
	require.NoError(t, err)
	resolve := func(Header) (any, error) { return pub, nil }
	payload := []byte("p")

	good, err := SignFlattenedJSON(payload, SignInput{Alg: ES256, Kid: "k", Key: key})
	require.NoError(t, err)

	corrupt := func(mut func(m map[string]string)) []byte {
		var m map[string]string
		require.NoError(t, json.Unmarshal(good, &m))
		mut(m)
		b, err := json.Marshal(m)
		require.NoError(t, err)

		return b
	}

	t.Run("bad protected base64", func(t *testing.T) {
		_, err := VerifyJSON(corrupt(func(m map[string]string) { m["protected"] = "!!!" }), resolve, []string{ES256})
		require.Error(t, err)
	})

	t.Run("bad protected json", func(t *testing.T) {
		m := corrupt(func(m map[string]string) {
			m["protected"] = base64.RawURLEncoding.EncodeToString([]byte("not json"))
		})
		_, err := VerifyJSON(m, resolve, []string{ES256})
		require.Error(t, err)
	})

	t.Run("bad signature base64", func(t *testing.T) {
		_, err := VerifyJSON(corrupt(func(m map[string]string) { m["signature"] = "!!!" }), resolve, []string{ES256})
		require.Error(t, err)
	})

	t.Run("resolver error", func(t *testing.T) {
		_, err := VerifyJSON(good, func(Header) (any, error) { return nil, errNoResolverKey{} }, []string{ES256})
		require.Error(t, err)
	})

	t.Run("bad payload base64", func(t *testing.T) {
		_, err := VerifyJSON(corrupt(func(m map[string]string) { m["payload"] = "!!!" }), resolve, []string{ES256})
		require.Error(t, err)
	})
}

// TestJWSJSONUnprotectedAlgRejected: a signature whose protected header carries
// no alg is rejected (verifyJSONSignature empty-alg branch).
func TestJWSJSONUnprotectedAlgRejected(t *testing.T) {
	key := []byte("hmac-secret-key-material-32bytes")
	// Hand-build a flattened JWS whose protected header has no "alg".
	protected := b64([]byte(`{"typ":"JWT"}`))
	payload := b64([]byte(`{"sub":"u"}`))
	sig := b64([]byte("whatever"))
	doc := fmt.Sprintf(`{"payload":"%s","protected":"%s","signature":"%s"}`, payload, protected, sig)

	_, err := VerifyJSON([]byte(doc), func(Header) (any, error) { return key, nil }, []string{HS256})
	require.Error(t, err)
}
