package secp256k1

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// bip340Vectors are selected official test vectors from BIP-340
// (https://github.com/bitcoin/bips/blob/master/bip-0340/test-vectors.csv).
var bip340Vectors = []struct {
	name    string
	secKey  string // empty for verify-only vectors
	pubKey  string
	auxRand string
	msg     string
	sig     string
	valid   bool
}{
	{
		name:    "vector 0",
		secKey:  "0000000000000000000000000000000000000000000000000000000000000003",
		pubKey:  "F9308A019258C31049344F85F89D5229B531C845836F99B08601F113BCE036F9",
		auxRand: "0000000000000000000000000000000000000000000000000000000000000000",
		msg:     "0000000000000000000000000000000000000000000000000000000000000000",
		sig:     "E907831F80848D1069A5371B402410364BDF1C5F8307B0084C55F1CE2DCA821525F66A4A85EA8B71E482A74F382D2CE5EBEEE8FDB2172F477DF4900D310536C0",
		valid:   true,
	},
	{
		name:    "vector 1",
		secKey:  "B7E151628AED2A6ABF7158809CF4F3C762E7160F38B4DA56A784D9045190CFEF",
		pubKey:  "DFF1D77F2A671C5F36183726DB2341BE58FEAE1DA2DECED843240F7B502BA659",
		auxRand: "0000000000000000000000000000000000000000000000000000000000000001",
		msg:     "243F6A8885A308D313198A2E03707344A4093822299F31D0082EFA98EC4E6C89",
		sig:     "6896BD60EEAE296DB48A229FF71DFE071BDE413E6D43F917DC8DCF8C78DE33418906D11AC976ABCCB20B091292BFF4EA897EFCB639EA871CFA95F6DE339E4B0A",
		valid:   true,
	},
	{
		name:   "vector 4 verify-only",
		pubKey: "D69C3509BB99E412E68B0FE8544E72837DFA30746D8BE2AA65975F29D22DC7B9",
		msg:    "4DF3C3F68FCC83B27E9D42C90431A72499F17875C81A599B566C9889B9696703",
		sig:    "00000000000000000000003B78CE563F89A0ED9414F5AA28AD0D96D6795F9C6376AFB1548AF603B3EB45C9F8207DEE1060CB71C04E80F593060B07D28308D7F4",
		valid:  true,
	},
	{
		name:   "vector 5 wrong pubkey not on curve",
		pubKey: "EEFDEA4CDB677750A420FEE807EACF21EB9898AE79B9768766E4FAA04A2D4A34",
		msg:    "243F6A8885A308D313198A2E03707344A4093822299F31D0082EFA98EC4E6C89",
		sig:    "6CFF5C3BA86C69EA4B7376F31A9BCB4F74C1976089B2D9963DA2E5543E17776969E89B4C5564D00349106B8497785DD7D1D713A8AE82B32FA79D5F7FC407D39B",
		valid:  false,
	},
	{
		name:   "vector 7 negated message",
		pubKey: "DFF1D77F2A671C5F36183726DB2341BE58FEAE1DA2DECED843240F7B502BA659",
		msg:    "243F6A8885A308D313198A2E03707344A4093822299F31D0082EFA98EC4E6C89",
		sig:    "1FA62E331EDBC21C394792D2AB1100A7B432B013DF3F6FF4F99FCB33E0E1515F28890B3EDB6E7189B630448B515CE4F8622A954CFE545735AAEA5134FCCDB2BD",
		valid:  false,
	},
}

func TestSchnorrVectors(t *testing.T) {
	for _, v := range bip340Vectors {
		t.Run(v.name, func(t *testing.T) {
			pubBytes := mustHexT(t, v.pubKey)
			msg := mustHexT(t, v.msg)
			sig := mustHexT(t, v.sig)

			if v.secKey != "" {
				priv, err := PrivKeyFromBytes(mustHexT(t, v.secKey))
				require.NoError(t, err)

				// Public key must match the x-only vector.
				assert.Equal(t, pubBytes, priv.Pub.SerializeXOnly())

				// Signing with the vector's aux rand must reproduce the signature.
				got, err := signSchnorr(priv, msg, bytes.NewReader(mustHexT(t, v.auxRand)))
				require.NoError(t, err)
				assert.Equal(t, sig, got)
			}

			pub, err := ParseXOnlyPubKey(pubBytes)
			if err != nil {
				// An unparseable key (not on curve) is only acceptable for
				// vectors flagged invalid.
				assert.False(t, v.valid)

				return
			}

			assert.Equal(t, v.valid, VerifySchnorr(pub, msg, sig))
		})
	}
}

func TestSchnorrRoundTrip(t *testing.T) {
	priv, err := GeneratePrivateKey()
	require.NoError(t, err)

	msg := mustHexT(t, "243F6A8885A308D313198A2E03707344A4093822299F31D0082EFA98EC4E6C89")

	sig, err := SignSchnorr(priv, msg)
	require.NoError(t, err)
	require.Len(t, sig, SchnorrSignatureLen)

	assert.True(t, VerifySchnorr(&priv.Pub, msg, sig))

	t.Run("x-only round-trip", func(t *testing.T) {
		xonly := priv.Pub.SerializeXOnly()
		require.Len(t, xonly, XOnlyPubKeyLen)

		parsed, err := ParseXOnlyPubKey(xonly)
		require.NoError(t, err)
		assert.Equal(t, 0, parsed.X.Cmp(priv.Pub.X))
		assert.True(t, VerifySchnorr(parsed, msg, sig))
	})

	t.Run("tampered message fails", func(t *testing.T) {
		bad := bytes.Clone(msg)
		bad[0] ^= 0xff
		assert.False(t, VerifySchnorr(&priv.Pub, bad, sig))
	})

	t.Run("tampered signature fails", func(t *testing.T) {
		bad := bytes.Clone(sig)
		bad[0] ^= 0xff
		assert.False(t, VerifySchnorr(&priv.Pub, msg, bad))
	})
}

func TestSchnorrInputValidation(t *testing.T) {
	priv, err := GeneratePrivateKey()
	require.NoError(t, err)

	t.Run("sign rejects non-32-byte message", func(t *testing.T) {
		_, err := SignSchnorr(priv, []byte("short"))
		require.Error(t, err)
	})

	t.Run("verify rejects bad lengths", func(t *testing.T) {
		assert.False(t, VerifySchnorr(&priv.Pub, make([]byte, 31), make([]byte, 64)))
		assert.False(t, VerifySchnorr(&priv.Pub, make([]byte, 32), make([]byte, 63)))
		assert.False(t, VerifySchnorr(nil, make([]byte, 32), make([]byte, 64)))
	})

	t.Run("ParseXOnlyPubKey rejects bad length", func(t *testing.T) {
		_, err := ParseXOnlyPubKey(make([]byte, 31))
		require.Error(t, err)
	})

	t.Run("odd-Y key signs and verifies", func(t *testing.T) {
		// Find a key whose public Y is odd to exercise the negation branches.
		for {
			p, err := GeneratePrivateKey()
			require.NoError(t, err)
			if p.Pub.Y.Bit(0) == 1 {
				msg := make([]byte, 32)
				sig, err := SignSchnorr(p, msg)
				require.NoError(t, err)
				assert.True(t, VerifySchnorr(&p.Pub, msg, sig))

				return
			}
		}
	})
}

// FuzzVerifySchnorr feeds arbitrary pubkey/msg/sig bytes to the verifier, which
// must never panic and must reject malformed input.
func FuzzVerifySchnorr(f *testing.F) {
	priv, err := GeneratePrivateKey()
	require.NoError(f, err)
	msg := make([]byte, 32)
	sig, err := SignSchnorr(priv, msg)
	require.NoError(f, err)

	f.Add(priv.Pub.SerializeXOnly(), msg, sig)
	f.Add([]byte{}, []byte{}, []byte{})

	f.Fuzz(func(_ *testing.T, pub, m, s []byte) {
		xonly, err := ParseXOnlyPubKey(pub)
		if err != nil {
			return
		}

		VerifySchnorr(xonly, m, s) // must not panic
	})
}

// FuzzParseXOnlyPubKey feeds arbitrary bytes to the x-only parser; it must never
// panic, and any key it accepts must lie on the curve with even Y.
func FuzzParseXOnlyPubKey(f *testing.F) {
	priv, err := GeneratePrivateKey()
	require.NoError(f, err)

	f.Add(priv.Pub.SerializeXOnly())
	f.Add([]byte{})
	f.Add(make([]byte, 32))

	f.Fuzz(func(t *testing.T, data []byte) {
		pub, err := ParseXOnlyPubKey(data)
		if err != nil {
			return
		}

		if !pub.IsValid() || pub.Y.Bit(0) != 0 {
			t.Fatalf("x-only parse produced invalid or odd-Y point")
		}
	})
}

func BenchmarkSignSchnorr(b *testing.B) {
	priv, err := GeneratePrivateKey()
	require.NoError(b, err)
	msg := make([]byte, 32)

	b.ResetTimer()
	for b.Loop() {
		_, _ = SignSchnorr(priv, msg)
	}
}

func BenchmarkVerifySchnorr(b *testing.B) {
	priv, err := GeneratePrivateKey()
	require.NoError(b, err)
	msg := make([]byte, 32)
	sig, err := SignSchnorr(priv, msg)
	require.NoError(b, err)

	b.ResetTimer()
	for b.Loop() {
		VerifySchnorr(&priv.Pub, msg, sig)
	}
}

func mustHexT(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	require.NoError(t, err)

	return b
}
