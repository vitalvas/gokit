package xjwt

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPBES2DefaultCountRoundTrip(t *testing.T) {
	// Exercise the real (production) iteration count for one combo.
	password := []byte("a strong passphrase")
	plaintext := []byte("pbes2 secret")

	token, err := Encrypt(PBES2HS256A128KW, A128CBCHS256, password, plaintext, EncryptOptions{})
	require.NoError(t, err)

	// The protected header must carry p2s and the default p2c.
	var hdr jweHeader
	raw, err := base64.RawURLEncoding.DecodeString(strings.SplitN(token, ".", 2)[0])
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, &hdr))
	assert.NotEmpty(t, hdr.P2s)
	assert.Equal(t, defaultPBES2Count, hdr.P2c)

	got, err := Decrypt(token, password)
	require.NoError(t, err)
	assert.Equal(t, plaintext, got)

	t.Run("wrong password fails", func(t *testing.T) {
		_, err := Decrypt(token, []byte("wrong"))
		assert.Error(t, err)
	})
}

func TestPBES2RejectsExcessiveCount(t *testing.T) {
	restore := defaultPBES2Count
	defaultPBES2Count = 1000
	defer func() { defaultPBES2Count = restore }()

	password := []byte("pw")
	token, err := Encrypt(PBES2HS256A128KW, A128GCM, password, []byte("x"), EncryptOptions{})
	require.NoError(t, err)

	// Rewrite the header with an abusive p2c; the guard must trip before any
	// expensive derivation.
	parts := strings.Split(token, ".")
	raw, err := base64.RawURLEncoding.DecodeString(parts[0])
	require.NoError(t, err)
	var hdr jweHeader
	require.NoError(t, json.Unmarshal(raw, &hdr))
	hdr.P2c = maxPBES2Count + 1
	newHdr, err := json.Marshal(hdr)
	require.NoError(t, err)
	parts[0] = base64.RawURLEncoding.EncodeToString(newHdr)

	_, err = Decrypt(strings.Join(parts, "."), password)
	assert.Error(t, err)
}

func TestPBES2KeyTypeMismatch(t *testing.T) {
	_, err := Encrypt(PBES2HS256A128KW, A128GCM, "not-bytes", []byte("x"), EncryptOptions{})
	assert.ErrorIs(t, err, ErrKeyTypeMismatch)
}

func TestPBES2DecryptCEKErrors(t *testing.T) {
	pw := []byte("pw")

	t.Run("missing p2s/p2c", func(t *testing.T) {
		_, err := pbes2DecryptCEK(PBES2HS256A128KW, pw, nil, &jweHeader{})
		require.Error(t, err)
	})

	t.Run("bad p2s base64", func(t *testing.T) {
		_, err := pbes2DecryptCEK(PBES2HS256A128KW, pw, nil, &jweHeader{P2s: "!!!", P2c: 1000})
		require.Error(t, err)
	})

	t.Run("unsupported pbes2 alg", func(t *testing.T) {
		_, err := pbes2DecryptCEK("PBES2-BAD", pw, nil, &jweHeader{P2s: "AA", P2c: 1000})
		require.Error(t, err)
	})
}

func TestPBES2EncryptCEKUnsupportedEnc(t *testing.T) {
	_, _, err := pbes2EncryptCEK(PBES2HS256A128KW, "BADENC", []byte("pw"), &jweHeader{})
	require.Error(t, err)
}

func TestPBES2ParamsUnsupported(t *testing.T) {
	_, _, err := pbes2Params("PBES2-BAD")
	require.Error(t, err)
}
