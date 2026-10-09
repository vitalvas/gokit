package xjwt

import (
	"bytes"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var jweEncAlgs = []string{
	A128CBCHS256, A192CBCHS384, A256CBCHS512,
	A128GCM, A192GCM, A256GCM,
}

// jweRecipientKey returns a matching (encryptKey, decryptKey) pair for a JWE
// key-management algorithm.
func jweRecipientKey(t *testing.T, alg, enc string) (encKey, decKey any) {
	t.Helper()

	switch alg {
	case RSAOAEP, RSAOAEP256, RSAOAEP384, RSAOAEP512:
		k, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)

		return &k.PublicKey, k
	case A128KW:
		k := symKey(t, 16)

		return k, k
	case A192KW:
		k := symKey(t, 24)

		return k, k
	case A256KW:
		k := symKey(t, 32)

		return k, k
	case A128GCMKW:
		k := symKey(t, 16)

		return k, k
	case A192GCMKW:
		k := symKey(t, 24)

		return k, k
	case A256GCMKW:
		k := symKey(t, 32)

		return k, k
	case Dir:
		n, err := cekLength(enc)
		require.NoError(t, err)
		k := symKey(t, n)

		return k, k
	case ECDHES, ECDHESA128, ECDHESA192, ECDHESA256:
		k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)

		return &k.PublicKey, k
	case PBES2HS256A128KW, PBES2HS384A192KW, PBES2HS512A256KW:
		pw := []byte("correct horse battery staple")

		return pw, pw
	default:
		t.Fatalf("unknown alg %s", alg)

		return nil, nil
	}
}

func symKey(t *testing.T, n int) []byte {
	t.Helper()
	k := make([]byte, n)
	_, err := rand.Read(k)
	require.NoError(t, err)

	return k
}

func TestJWERoundTripMatrix(t *testing.T) {
	// Keep PBKDF2 cheap for the matrix; a dedicated test covers the real count.
	restore := defaultPBES2Count
	defaultPBES2Count = 1000
	defer func() { defaultPBES2Count = restore }()

	keyAlgs := []string{
		RSAOAEP, RSAOAEP256, RSAOAEP384, RSAOAEP512,
		A128KW, A192KW, A256KW,
		A128GCMKW, A192GCMKW, A256GCMKW,
		Dir,
		ECDHES, ECDHESA128, ECDHESA192, ECDHESA256,
		PBES2HS256A128KW, PBES2HS384A192KW, PBES2HS512A256KW,
	}

	plaintext := []byte("the quick brown fox jumps over the lazy dog")

	for _, alg := range keyAlgs {
		for _, enc := range jweEncAlgs {
			// For dir and ECDH-ES direct, the key is tied to the enc key size;
			// the helper handles that.
			t.Run(fmt.Sprintf("%s/%s", alg, enc), func(t *testing.T) {
				encKey, decKey := jweRecipientKey(t, alg, enc)

				token, err := Encrypt(alg, enc, encKey, plaintext, EncryptOptions{Kid: "k1"})
				require.NoError(t, err)

				got, err := Decrypt(token, decKey)
				require.NoError(t, err)
				assert.Equal(t, plaintext, got)
			})
		}
	}
}

func TestJWETamperDetection(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	token, err := Encrypt(RSAOAEP256, A256GCM, &key.PublicKey, []byte("secret"), EncryptOptions{})
	require.NoError(t, err)

	t.Run("tampered ciphertext fails", func(t *testing.T) {
		b := []byte(token)
		b[len(b)-20] ^= 0xff // flip a byte in the ciphertext/tag region
		_, err := Decrypt(string(b), key)
		assert.Error(t, err)
	})

	t.Run("wrong key fails", func(t *testing.T) {
		other, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)
		_, err = Decrypt(token, other)
		assert.Error(t, err)
	})

	t.Run("malformed JWE rejected", func(t *testing.T) {
		_, err := Decrypt("a.b.c", key)
		assert.Error(t, err)
	})
}

func TestJWEDirWrongKeySize(t *testing.T) {
	// dir requires the key to exactly match the enc CEK size.
	_, err := Encrypt(Dir, A256GCM, symKey(t, 16), []byte("x"), EncryptOptions{})
	assert.Error(t, err)
}

func TestJWEKeyTypeMismatch(t *testing.T) {
	_, err := Encrypt(RSAOAEP, A128GCM, []byte("not-rsa"), []byte("x"), EncryptOptions{})
	assert.ErrorIs(t, err, ErrKeyTypeMismatch)
}

func TestJWEX25519(t *testing.T) {
	priv, err := ecdh.X25519().GenerateKey(rand.Reader)
	require.NoError(t, err)

	plaintext := []byte("x25519 payload")

	for _, alg := range []string{ECDHES, ECDHESA128, ECDHESA192, ECDHESA256} {
		for _, enc := range []string{A128GCM, A256CBCHS512} {
			t.Run(fmt.Sprintf("%s/%s", alg, enc), func(t *testing.T) {
				token, err := Encrypt(alg, enc, priv.PublicKey(), plaintext, EncryptOptions{})
				require.NoError(t, err)

				got, err := Decrypt(token, priv)
				require.NoError(t, err)
				assert.Equal(t, plaintext, got)
			})
		}
	}
}

func TestJWEECDHWithAPUAPV(t *testing.T) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	plaintext := []byte("apu/apv payload")

	token, err := Encrypt(ECDHESA128, A128GCM, &priv.PublicKey, plaintext,
		EncryptOptions{APU: []byte("Alice"), APV: []byte("Bob")})
	require.NoError(t, err)

	// apu/apv must be written to the protected header.
	raw, err := base64.RawURLEncoding.DecodeString(strings.Split(token, ".")[0])
	require.NoError(t, err)
	assert.Contains(t, string(raw), `"apu"`)
	assert.Contains(t, string(raw), `"apv"`)

	// Round-trip: the header-carried apu/apv feed the KDF on both sides.
	got, err := Decrypt(token, priv)
	require.NoError(t, err)
	assert.Equal(t, plaintext, got)

	t.Run("tampered apv breaks key agreement", func(t *testing.T) {
		var hdr jweHeader
		require.NoError(t, json.Unmarshal(raw, &hdr))
		hdr.Apv = base64.RawURLEncoding.EncodeToString([]byte("Eve"))
		newHdr, err := json.Marshal(hdr)
		require.NoError(t, err)

		parts := strings.Split(token, ".")
		parts[0] = base64.RawURLEncoding.EncodeToString(newHdr)
		_, err = Decrypt(strings.Join(parts, "."), priv)
		assert.Error(t, err, "changed apv must derive a different key and fail")
	})
}

func TestJWEDecryptAllowlist(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	token, err := Encrypt(RSAOAEP256, A256GCM, &key.PublicKey, []byte("secret"), EncryptOptions{})
	require.NoError(t, err)

	t.Run("allowed alg and enc", func(t *testing.T) {
		got, err := DecryptWithOptions(token, key, DecryptOptions{
			AllowedAlgs: []string{RSAOAEP256},
			AllowedEnc:  []string{A256GCM},
		})
		require.NoError(t, err)
		assert.Equal(t, "secret", string(got))
	})

	t.Run("disallowed alg rejected", func(t *testing.T) {
		_, err := DecryptWithOptions(token, key, DecryptOptions{AllowedAlgs: []string{RSAOAEP}})
		require.Error(t, err)
	})

	t.Run("disallowed enc rejected", func(t *testing.T) {
		_, err := DecryptWithOptions(token, key, DecryptOptions{AllowedEnc: []string{A128GCM}})
		require.Error(t, err)
	})

	t.Run("empty allowlist accepts any", func(t *testing.T) {
		got, err := DecryptWithOptions(token, key, DecryptOptions{})
		require.NoError(t, err)
		assert.Equal(t, "secret", string(got))
	})
}

func TestJWECompression(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	// A highly compressible payload.
	plaintext := bytes.Repeat([]byte("compress me "), 500)

	token, err := Encrypt(RSAOAEP256, A256GCM, &key.PublicKey, plaintext,
		EncryptOptions{Compress: true})
	require.NoError(t, err)

	// The protected header must declare zip=DEF.
	raw, err := base64.RawURLEncoding.DecodeString(strings.Split(token, ".")[0])
	require.NoError(t, err)
	assert.Contains(t, string(raw), `"zip":"DEF"`)

	got, err := Decrypt(token, key)
	require.NoError(t, err)
	assert.Equal(t, plaintext, got)
}

// TestJWEDecryptRFC7516 decrypts the A128KW/A128CBC-HS256 example from RFC 7516
// Appendix A.3, proving wire-format interoperability with other implementations.
func TestJWEDecryptRFC7516(t *testing.T) {
	key, err := base64.RawURLEncoding.DecodeString("GawgguFyGrWKav7AX4VKUg")
	require.NoError(t, err)

	const jwe = `eyJhbGciOiJBMTI4S1ciLCJlbmMiOiJBMTI4Q0JDLUhTMjU2In0.6KB707dM9YTIgHtLvtgWQ8mKwboJW3of9locizkDTHzBC2IlrT1oOQ.AxY8DCtDaGlsbGljb3RoZQ.KDlTtXchhZTGufMYmOYGS4HffxPSUrfmqCHXaI9wOGY.U0m_YmjN04DJvceFICbCVQ`

	got, err := Decrypt(jwe, key)
	require.NoError(t, err)
	assert.Equal(t, "Live long and prosper.", string(got))
}

func ecdhP256Key() (*ecdh.PrivateKey, error) {
	return ecdh.P256().GenerateKey(rand.Reader)
}

// reheader rebuilds a compact JWE from a header struct and the other four
// (already-encoded) segments, for crafting malformed-header test cases.
func reheader(t *testing.T, token string, mutate func(*jweHeader)) string {
	t.Helper()
	parts := strings.Split(token, ".")
	raw, err := base64.RawURLEncoding.DecodeString(parts[0])
	require.NoError(t, err)
	var h jweHeader
	require.NoError(t, json.Unmarshal(raw, &h))
	mutate(&h)
	nh, err := json.Marshal(h)
	require.NoError(t, err)
	parts[0] = base64.RawURLEncoding.EncodeToString(nh)

	return strings.Join(parts, ".")
}

func TestDecryptErrorPaths(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	token, err := Encrypt(RSAOAEP256, A256GCM, &key.PublicKey, []byte("secret"), EncryptOptions{})
	require.NoError(t, err)

	t.Run("bad header base64", func(t *testing.T) {
		parts := strings.Split(token, ".")
		parts[0] = "!!!"
		_, err := Decrypt(strings.Join(parts, "."), key)
		require.Error(t, err)
	})

	t.Run("bad header json", func(t *testing.T) {
		parts := strings.Split(token, ".")
		parts[0] = base64.RawURLEncoding.EncodeToString([]byte("not json"))
		_, err := Decrypt(strings.Join(parts, "."), key)
		require.Error(t, err)
	})

	t.Run("bad encrypted-key base64", func(t *testing.T) {
		parts := strings.Split(token, ".")
		parts[1] = "!!!"
		_, err := Decrypt(strings.Join(parts, "."), key)
		require.Error(t, err)
	})

	t.Run("bad iv base64", func(t *testing.T) {
		parts := strings.Split(token, ".")
		parts[2] = "!!!"
		_, err := Decrypt(strings.Join(parts, "."), key)
		require.Error(t, err)
	})

	t.Run("bad ciphertext base64", func(t *testing.T) {
		parts := strings.Split(token, ".")
		parts[3] = "!!!"
		_, err := Decrypt(strings.Join(parts, "."), key)
		require.Error(t, err)
	})

	t.Run("bad tag base64", func(t *testing.T) {
		parts := strings.Split(token, ".")
		parts[4] = "!!!"
		_, err := Decrypt(strings.Join(parts, "."), key)
		require.Error(t, err)
	})

	t.Run("unknown zip", func(t *testing.T) {
		bad := reheader(t, token, func(h *jweHeader) { h.Zip = "BOGUS" })
		_, err := Decrypt(bad, key)
		require.Error(t, err)
	})
}

func TestCekLengthUnsupported(t *testing.T) {
	_, err := cekLength("A999GCM")
	require.Error(t, err)
}

func TestEncryptDecryptErrors(t *testing.T) {
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	t.Run("unsupported key alg on encrypt", func(t *testing.T) {
		_, err := Encrypt("BADALG", A128GCM, &rsaKey.PublicKey, []byte("x"), EncryptOptions{})
		require.Error(t, err)
	})

	t.Run("unsupported enc on encrypt", func(t *testing.T) {
		_, err := Encrypt(RSAOAEP256, "BADENC", &rsaKey.PublicKey, []byte("x"), EncryptOptions{})
		require.Error(t, err)
	})

	t.Run("wrong segment count", func(t *testing.T) {
		_, err := Decrypt("a.b.c", rsaKey)
		require.Error(t, err)
	})

	t.Run("bad base64 segments", func(t *testing.T) {
		_, err := Decrypt(strings.Join([]string{badB64, badB64, badB64, badB64, badB64}, "."), rsaKey)
		require.Error(t, err)
	})

	t.Run("unsupported alg in header", func(t *testing.T) {
		hdr := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"BAD","enc":"A128GCM"}`))
		_, err := Decrypt(fmt.Sprintf("%s.AA.AA.AA.AA", hdr), rsaKey)
		require.Error(t, err)
	})
}
