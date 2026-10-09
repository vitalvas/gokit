package xjwt

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// remoteSigner wraps an in-memory key but is a distinct type, so xjwt cannot
// take the concrete-key fast path and must drive it through crypto.Signer -- the
// same way an HSM/KMS-backed key would be used.
type remoteSigner struct {
	inner crypto.Signer
}

func (r remoteSigner) Public() crypto.PublicKey { return r.inner.Public() }

func (r remoteSigner) Sign(rnd io.Reader, digest []byte, opts crypto.SignerOpts) ([]byte, error) {
	return r.inner.Sign(rnd, digest, opts)
}

func TestSignWithCryptoSigner(t *testing.T) {
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	ec256, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	ec384, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	require.NoError(t, err)
	edPub, edPriv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	cases := []struct {
		alg    string
		signer crypto.Signer
		pub    any
	}{
		{RS256, remoteSigner{rsaKey}, &rsaKey.PublicKey},
		{RS512, remoteSigner{rsaKey}, &rsaKey.PublicKey},
		{PS256, remoteSigner{rsaKey}, &rsaKey.PublicKey},
		{PS384, remoteSigner{rsaKey}, &rsaKey.PublicKey},
		{ES256, remoteSigner{ec256}, &ec256.PublicKey},
		{ES384, remoteSigner{ec384}, &ec384.PublicKey},
		{EdDSA, remoteSigner{edPriv}, edPub},
	}

	for _, tc := range cases {
		t.Run(tc.alg, func(t *testing.T) {
			token, err := Sign(tc.alg, "kid-remote", MapClaims{"sub": "svc"}, tc.signer)
			require.NoError(t, err, "signing via crypto.Signer")

			resolve := func(_ Header) (any, error) { return tc.pub, nil }
			payload, err := Verify(token, resolve, []string{tc.alg})
			require.NoError(t, err, "verifying a crypto.Signer-produced token")
			assert.Contains(t, string(payload), `"sub":"svc"`)
		})
	}
}

// TestSignWithSignerRSAKeyTooSmall confirms the RFC 7518 Section 3.3 2048-bit
// minimum is enforced on the crypto.Signer (HSM/KMS) path, not only on concrete
// in-memory keys.
func TestSignWithSignerRSAKeyTooSmall(t *testing.T) {
	small, err := rsa.GenerateKey(rand.Reader, 1024)
	require.NoError(t, err)

	_, err = Sign(RS256, "kid-remote", MapClaims{"sub": "svc"}, remoteSigner{small})
	require.ErrorIs(t, err, ErrKeyTooSmall)
}

func TestSignWithSignerWrongPublicKeyType(t *testing.T) {
	// A signer whose public key is EC cannot satisfy an RSA alg.
	ec, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	_, err = Sign(RS256, "k", MapClaims{"sub": "x"}, remoteSigner{ec})
	assert.ErrorIs(t, err, ErrKeyTypeMismatch)
}

func TestSignWithSignerCurveMismatch(t *testing.T) {
	// ES512 requires a P-521 key; a P-256 signer must be rejected.
	ec, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	_, err = Sign(ES512, "k", MapClaims{"sub": "x"}, remoteSigner{ec})
	assert.ErrorIs(t, err, ErrKeyTypeMismatch)
}

// wrongPubSigner reports a public key type that does not match the alg.
type wrongPubSigner struct{ pub crypto.PublicKey }

func (w wrongPubSigner) Public() crypto.PublicKey { return w.pub }
func (w wrongPubSigner) Sign(io.Reader, []byte, crypto.SignerOpts) ([]byte, error) {
	return []byte("sig"), nil
}

func TestSignerWrongPublicKeyTypes(t *testing.T) {
	ec, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	// RSA alg, signer exposes an EC public key.
	_, err = Sign(RS256, "k", MapClaims{}, wrongPubSigner{&ec.PublicKey})
	require.ErrorIs(t, err, ErrKeyTypeMismatch)

	// EC alg, signer exposes an RSA public key.
	_, err = Sign(ES256, "k", MapClaims{}, wrongPubSigner{&rsaKey.PublicKey})
	require.ErrorIs(t, err, ErrKeyTypeMismatch)

	// EdDSA alg, signer exposes an RSA public key.
	_, err = Sign(EdDSA, "k", MapClaims{}, wrongPubSigner{&rsaKey.PublicKey})
	require.ErrorIs(t, err, ErrKeyTypeMismatch)
}

func TestSignerMalformedECDSASignature(t *testing.T) {
	ec, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	// A signer with the right EC public key but returning non-DER bytes triggers
	// the ASN.1 unmarshal error path.
	_, err = Sign(ES256, "k", MapClaims{}, wrongPubSigner{&ec.PublicKey})
	require.Error(t, err)
}

// errSigner is a crypto.Signer whose Sign always fails, to drive the signer
// error paths.
type errSigner struct{ pub crypto.PublicKey }

func (e errSigner) Public() crypto.PublicKey { return e.pub }
func (e errSigner) Sign(io.Reader, []byte, crypto.SignerOpts) ([]byte, error) {
	return nil, errors.New("signer failure")
}

func TestSignerErrorPaths(t *testing.T) {
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	ec, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	edPub, _, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	t.Run("RSA signer failure", func(t *testing.T) {
		_, err := Sign(RS256, "k", MapClaims{}, errSigner{&rsaKey.PublicKey})
		require.Error(t, err)
	})
	t.Run("EC signer failure", func(t *testing.T) {
		_, err := Sign(ES256, "k", MapClaims{}, errSigner{&ec.PublicKey})
		require.Error(t, err)
	})
	t.Run("Ed signer failure", func(t *testing.T) {
		_, err := Sign(EdDSA, "k", MapClaims{}, errSigner{edPub})
		require.Error(t, err)
	})
	t.Run("non-signer non-key rejected", func(t *testing.T) {
		_, err := Sign(RS256, "k", MapClaims{}, 12345)
		require.ErrorIs(t, err, ErrKeyTypeMismatch)
	})
}
