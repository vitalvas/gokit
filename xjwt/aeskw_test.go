package xjwt

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// RFC 3394 Section 4 test vectors.
func TestAESKeyWrapVectors(t *testing.T) {
	cases := []struct {
		name    string
		kek     string
		key     string
		wrapped string
	}{
		{
			name:    "128-bit KEK, 128-bit key",
			kek:     "000102030405060708090A0B0C0D0E0F",
			key:     "00112233445566778899AABBCCDDEEFF",
			wrapped: "1FA68B0A8112B447AEF34BD8FB5A7B829D3E862371D2CFE5",
		},
		{
			name:    "192-bit KEK, 128-bit key",
			kek:     "000102030405060708090A0B0C0D0E0F1011121314151617",
			key:     "00112233445566778899AABBCCDDEEFF",
			wrapped: "96778B25AE6CA435F92B5B97C050AED2468AB8A17AD84E5D",
		},
		{
			name:    "256-bit KEK, 128-bit key",
			kek:     "000102030405060708090A0B0C0D0E0F101112131415161718191A1B1C1D1E1F",
			key:     "00112233445566778899AABBCCDDEEFF",
			wrapped: "64E8C3F9CE0F5BA263E9777905818A2A93C8191E7D6E8AE7",
		},
		{
			name:    "256-bit KEK, 256-bit key",
			kek:     "000102030405060708090A0B0C0D0E0F101112131415161718191A1B1C1D1E1F",
			key:     "00112233445566778899AABBCCDDEEFF000102030405060708090A0B0C0D0E0F",
			wrapped: "28C9F404C4B810F4CBCCB35CFB87F8263F5786E2D80ED326CBC7F0E71A99F43BFB988B9B7A02DD21",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			kek := mustHexDecode(t, tc.kek)
			key := mustHexDecode(t, tc.key)
			want := mustHexDecode(t, tc.wrapped)

			got, err := aesKeyWrap(kek, key)
			require.NoError(t, err)
			assert.Equal(t, want, got, "wrap vector mismatch")

			unwrapped, err := aesKeyUnwrap(kek, want)
			require.NoError(t, err)
			assert.Equal(t, key, unwrapped, "unwrap vector mismatch")
		})
	}
}

func TestAESKeyWrapTamper(t *testing.T) {
	kek := mustHexDecode(t, "000102030405060708090A0B0C0D0E0F")
	key := mustHexDecode(t, "00112233445566778899AABBCCDDEEFF")

	wrapped, err := aesKeyWrap(kek, key)
	require.NoError(t, err)

	wrapped[0] ^= 0xff
	_, err = aesKeyUnwrap(kek, wrapped)
	assert.Error(t, err, "tampered wrap must fail the integrity check")
}

func mustHexDecode(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	require.NoError(t, err)

	return b
}

func TestAESKWBadLengths(t *testing.T) {
	kek := make([]byte, 16)

	// Wrap plaintext not a multiple of 8 / too short.
	_, err := aesKeyWrap(kek, []byte{1, 2, 3})
	require.Error(t, err)

	// Unwrap ciphertext too short.
	_, err = aesKeyUnwrap(kek, []byte{1, 2, 3})
	require.Error(t, err)
}
