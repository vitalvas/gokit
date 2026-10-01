package secp256k1

import (
	"bytes"
	"crypto/sha256"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSignVerifyRoundTrip(t *testing.T) {
	priv, err := GeneratePrivateKey()
	require.NoError(t, err)

	digest := sha256.Sum256([]byte("gandalf es256k round trip"))
	r, s := Sign(priv, digest[:])

	t.Run("valid signature verifies", func(t *testing.T) {
		assert.True(t, Verify(&priv.Pub, digest[:], r, s))
	})

	t.Run("low-S canonical", func(t *testing.T) {
		assert.True(t, s.Cmp(halfOrder) <= 0)
	})

	t.Run("tampered digest fails", func(t *testing.T) {
		other := sha256.Sum256([]byte("different message"))
		assert.False(t, Verify(&priv.Pub, other[:], r, s))
	})

	t.Run("deterministic: same input same signature", func(t *testing.T) {
		r2, s2 := Sign(priv, digest[:])
		assert.Equal(t, 0, r.Cmp(r2))
		assert.Equal(t, 0, s.Cmp(s2))
	})

	t.Run("nil arguments rejected", func(t *testing.T) {
		assert.False(t, Verify(nil, digest[:], r, s))
		assert.False(t, Verify(&priv.Pub, digest[:], nil, s))
	})

	t.Run("out-of-range r/s rejected", func(t *testing.T) {
		assert.False(t, Verify(&priv.Pub, digest[:], big.NewInt(0), s))
		assert.False(t, Verify(&priv.Pub, digest[:], r, new(big.Int).Set(orderN)))
	})
}

// TestKnownAnswer pins this implementation to a frozen RFC 6979 vector. The
// expected values were produced by this package and independently validated,
// bit-for-bit, against the reference decred/dcrd secp256k1 library during
// development (deterministic nonce, public key, and low-S signature all
// matched). Freezing them here keeps the package self-contained: any future
// regression in the field, point, RFC 6979, or low-S logic changes one of
// these outputs and fails the test, with no external dependency.
func TestKnownAnswer(t *testing.T) {
	seed := mustDecodeHex(t, "c9afa9d845ba75166b5c215767b1d6934e50c3db36e89b127b8a622b120f6721")

	wantPub := mustDecodeHex(t, "042c8c31fc9f990c6b55e3865a184a4ce50e09481f2eaeb3e60ec1cea13a6ae64564b95e4fdb6948c0386e189b006a29f686769b011704275e4459822dc3328085")
	wantR := mustDecodeHex(t, "adb97a10ac5048cfda0b0b38e1cb547206c07b35305d84c6678c1128fdd220f5")
	wantS := mustDecodeHex(t, "42d4d3ea1a7695f0ca3caca588c879ad3b4d973758fdf1c466657444c27b6391")

	priv, err := PrivKeyFromBytes(seed)
	require.NoError(t, err)

	t.Run("public key matches reference", func(t *testing.T) {
		assert.True(t, bytes.Equal(wantPub, priv.Pub.SerializeUncompressed()))
	})

	digest := sha256.Sum256([]byte("cross-check message"))
	r, s := Sign(priv, digest[:])

	t.Run("signature matches reference vector", func(t *testing.T) {
		assert.True(t, bytes.Equal(wantR, fixedBytes(r)), "r")
		assert.True(t, bytes.Equal(wantS, fixedBytes(s)), "s")
	})

	t.Run("vector verifies", func(t *testing.T) {
		assert.True(t, Verify(&priv.Pub, digest[:],
			new(big.Int).SetBytes(wantR), new(big.Int).SetBytes(wantS)))
	})
}

func TestVerifyEdgeCases(t *testing.T) {
	priv, err := GeneratePrivateKey()
	require.NoError(t, err)

	digest := sha256.Sum256([]byte("edge cases"))
	r, s := Sign(priv, digest[:])

	t.Run("off-curve public key rejected", func(t *testing.T) {
		bad := &PublicKey{X: big.NewInt(1), Y: big.NewInt(1)}
		assert.False(t, Verify(bad, digest[:], r, s))
	})

	t.Run("longer-than-order hash is truncated", func(t *testing.T) {
		// A 64-byte digest exercises the hashToInt/bits2int truncation path.
		long := make([]byte, 64)
		copy(long, digest[:])
		lr, ls := Sign(priv, long)
		assert.True(t, Verify(&priv.Pub, long, lr, ls))
	})
}

func TestVerifyStrictRejectsMalleability(t *testing.T) {
	priv, err := GeneratePrivateKey()
	require.NoError(t, err)

	digest := sha256.Sum256([]byte("malleability"))
	r, s := Sign(priv, digest[:])

	// Sign always returns canonical low-S, so both verifiers accept it.
	require.True(t, s.Cmp(halfOrder) <= 0, "Sign must produce low-S")
	assert.True(t, Verify(&priv.Pub, digest[:], r, s))
	assert.True(t, VerifyStrict(&priv.Pub, digest[:], r, s))

	// The malleated form (r, N-s) is a valid signature that Verify accepts but
	// VerifyStrict must reject.
	highS := new(big.Int).Sub(orderN, s)
	assert.True(t, Verify(&priv.Pub, digest[:], r, highS), "malleated sig is still valid")
	assert.False(t, VerifyStrict(&priv.Pub, digest[:], r, highS), "strict must reject high-S")

	t.Run("nil s rejected", func(t *testing.T) {
		assert.False(t, VerifyStrict(&priv.Pub, digest[:], r, nil))
	})
}

func TestSignRetriesOnBadNonce(t *testing.T) {
	priv, err := GeneratePrivateKey()
	require.NoError(t, err)

	// Inject degenerate first nonces (zero, then out-of-range) so Sign must
	// skip them and fall through to the real RFC 6979 nonce on later attempts.
	orig := nonceFunc
	t.Cleanup(func() { nonceFunc = orig })

	nonceFunc = func(d *big.Int, hash []byte, attempt int) *big.Int {
		switch attempt {
		case 0:
			return big.NewInt(0) // triggers k.Sign()==0 retry
		case 1:
			return new(big.Int).Set(orderN) // triggers k >= n retry
		default:
			return orig(d, hash, attempt)
		}
	}

	digest := sha256.Sum256([]byte("retry path"))
	r, s := Sign(priv, digest[:])
	assert.True(t, Verify(&priv.Pub, digest[:], r, s))
}

func TestSignRetriesOnZeroS(t *testing.T) {
	priv, err := GeneratePrivateKey()
	require.NoError(t, err)

	// Choose a fixed nonce k0 for attempt 0 and derive r0 = (k0*G).x mod n.
	k0 := big.NewInt(12345)
	x0, _ := scalarBaseMult(k0)
	r0 := new(big.Int).Mod(x0, orderN)

	// Craft a hash whose integer value e == (-r0*d) mod n, forcing
	// s = k0^-1 * (e + r0*d) == 0 on attempt 0 so Sign must retry.
	e := new(big.Int).Mul(r0, priv.D)
	e.Neg(e)
	e.Mod(e, orderN)
	hash := fixedBytes(e)

	orig := nonceFunc
	t.Cleanup(func() { nonceFunc = orig })
	nonceFunc = func(d *big.Int, h []byte, attempt int) *big.Int {
		if attempt == 0 {
			return new(big.Int).Set(k0)
		}

		return orig(d, h, attempt)
	}

	r, s := Sign(priv, hash)
	require.NotNil(t, s)
	assert.NotEqual(t, 0, s.Sign(), "final signature must have non-zero s")
	assert.True(t, Verify(&priv.Pub, hash, r, s))
}

func TestRFC6979Octets(t *testing.T) {
	t.Run("int2octets pads short and truncates long", func(t *testing.T) {
		// Short value: 1 byte padded to 32.
		assert.Len(t, int2octets(big.NewInt(1), 32), 32)

		// Long value: a 33-byte number truncated to 32.
		big33 := new(big.Int).Lsh(big.NewInt(1), 8*32) // 2^256, 33 bytes
		assert.Len(t, int2octets(big33, 32), 32)
	})

	t.Run("bits2octets reduces hashes above the order", func(t *testing.T) {
		// All-0xFF input exceeds N, exercising the z2 >= 0 branch.
		in := make([]byte, 32)
		for i := range in {
			in[i] = 0xff
		}
		out := bits2octets(in, 32)
		assert.Len(t, out, 32)
	})

	t.Run("bits2int right-shifts when input has excess bits", func(t *testing.T) {
		// qlen smaller than the input bit length exercises the excess>0 shift.
		v := bits2int([]byte{0xff}, 4)
		assert.Equal(t, int64(0x0f), v.Int64())
	})

	t.Run("int2octets passes through exact-length values", func(t *testing.T) {
		exact := make([]byte, 32)
		exact[31] = 1
		v := new(big.Int).SetBytes(exact)
		assert.Len(t, int2octets(v, 32), 32)
	})
}

func TestHashToIntTruncation(t *testing.T) {
	// A digest longer than the order's byte length is truncated to orderBytes.
	long := make([]byte, 48)
	for i := range long {
		long[i] = 0xab
	}

	got := hashToInt(long)
	want := new(big.Int).SetBytes(long[:32])
	assert.Equal(t, 0, got.Cmp(want))
}

func BenchmarkSign(b *testing.B) {
	priv, err := GeneratePrivateKey()
	require.NoError(b, err)

	digest := sha256.Sum256([]byte("benchmark message"))

	b.ResetTimer()
	for b.Loop() {
		Sign(priv, digest[:])
	}
}

func BenchmarkVerify(b *testing.B) {
	priv, err := GeneratePrivateKey()
	require.NoError(b, err)

	digest := sha256.Sum256([]byte("benchmark message"))
	r, s := Sign(priv, digest[:])

	b.ResetTimer()
	for b.Loop() {
		Verify(&priv.Pub, digest[:], r, s)
	}
}

// FuzzVerify feeds arbitrary signature and digest bytes to the verifier. It
// must never panic and must never accept a forged signature; the only
// acceptable "true" result is the genuine signature seeded below.
func FuzzVerify(f *testing.F) {
	priv, err := GeneratePrivateKey()
	require.NoError(f, err)

	digest := sha256.Sum256([]byte("fuzz seed message"))
	r, s := Sign(priv, digest[:])

	f.Add(digest[:], fixedBytes(r), fixedBytes(s))
	f.Add([]byte{}, []byte{}, []byte{})
	f.Add(digest[:], []byte{0x00}, []byte{0x00})

	f.Fuzz(func(t *testing.T, dgst, rBytes, sBytes []byte) {
		got := Verify(&priv.Pub, dgst, new(big.Int).SetBytes(rBytes), new(big.Int).SetBytes(sBytes))

		// The only input that may verify is the exact seeded triple.
		if got {
			if !bytes.Equal(dgst, digest[:]) ||
				new(big.Int).SetBytes(rBytes).Cmp(r) != 0 ||
				new(big.Int).SetBytes(sBytes).Cmp(s) != 0 {
				t.Fatalf("forged signature accepted")
			}
		}
	})
}
