package shamir

import (
	"bytes"
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGF256Add(t *testing.T) {
	tests := []struct {
		a, b, expected uint8
	}{
		{0, 0, 0},
		{1, 0, 1},
		{0, 1, 1},
		{1, 1, 0}, // XOR: 1 ^ 1 = 0
		{0xFF, 0xFF, 0},
		{0xAA, 0x55, 0xFF},
	}

	for _, tt := range tests {
		result := gf256Add(tt.a, tt.b)
		assert.Equal(t, tt.expected, result, "gf256Add(%d, %d)", tt.a, tt.b)
	}
}

func TestGF256Mult(t *testing.T) {
	tests := []struct {
		a, b, expected uint8
	}{
		{0, 0, 0},
		{1, 0, 0},
		{0, 1, 0},
		{1, 1, 1},
		{2, 2, 4},
		{2, 128, 27}, // Tests reduction
	}

	for _, tt := range tests {
		result := gf256Mult(tt.a, tt.b)
		assert.Equal(t, tt.expected, result, "gf256Mult(%d, %d)", tt.a, tt.b)
	}

	// Test commutativity
	for range 100 {
		buf := make([]byte, 2)
		_, _ = rand.Read(buf)
		a, b := buf[0], buf[1]
		assert.Equal(t, gf256Mult(a, b), gf256Mult(b, a), "commutativity for %d, %d", a, b)
	}
}

func TestGF256Inverse(t *testing.T) {
	// Test that a * a^(-1) = 1 for all non-zero a
	for a := 1; a < 256; a++ {
		inv := gf256Inverse(uint8(a))
		product := gf256Mult(uint8(a), inv)
		assert.Equal(t, uint8(1), product, "gf256Inverse(%d) * %d should equal 1", a, a)
	}

	// Zero has no inverse, returns 0
	assert.Equal(t, uint8(0), gf256Inverse(0))
}

func TestGF256Div(t *testing.T) {
	tests := []struct {
		a, b, expected uint8
	}{
		{0, 1, 0},
		{1, 1, 1},
		{4, 2, 2},
		{27, 2, 128}, // Reverse of mult test
	}

	for _, tt := range tests {
		result := gf256Div(tt.a, tt.b)
		assert.Equal(t, tt.expected, result, "gf256Div(%d, %d)", tt.a, tt.b)
	}

	// Test that a / b * b = a
	for range 100 {
		buf := make([]byte, 2)
		_, _ = rand.Read(buf)
		a, b := buf[0], buf[1]
		if b == 0 {
			continue
		}
		result := gf256Mult(gf256Div(a, b), b)
		assert.Equal(t, a, result, "(%d / %d) * %d should equal %d", a, b, b, a)
	}
}

func TestGF256DivByZeroPanics(t *testing.T) {
	assert.Panics(t, func() {
		gf256Div(1, 0)
	})
}

func TestGF256Polynomial(t *testing.T) {
	// f(x) = 5 + 3x + 2x^2 in GF(2^8)
	poly := &gf256Polynomial{
		coefficients: []uint8{5, 3, 2},
	}

	// f(0) should return intercept
	assert.Equal(t, uint8(5), poly.evaluate(0))

	// f(1) = 5 + 3 + 2 = 5 ^ 3 ^ 2 = 4 (XOR addition)
	assert.Equal(t, uint8(4), poly.evaluate(1))
}

func TestSplitGF256(t *testing.T) {
	tests := []struct {
		name      string
		secret    []byte
		threshold int
		total     int
	}{
		{"short secret", []byte("hi"), 2, 3},
		{"medium secret", []byte("hello world"), 3, 5},
		{"32 bytes", make([]byte, 32), 3, 5},
		{"256 bytes", make([]byte, 256), 5, 10},
		{"1KB", make([]byte, 1024), 3, 5},
		{"threshold equals total", []byte("test"), 3, 3},
		{"max shares", []byte("max"), 2, 255},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secret := tt.secret
			if len(secret) > 10 {
				_, _ = rand.Read(secret)
			}

			shares, err := SplitGF256(secret, tt.threshold, tt.total)
			require.NoError(t, err)
			require.Len(t, shares, tt.total)

			// Verify share size
			for _, share := range shares {
				assert.Len(t, share, len(secret)+1, "share size should be len(secret)+1")
			}

			// Reconstruct with exact threshold
			recovered, err := CombineGF256(shares[:tt.threshold])
			require.NoError(t, err)
			assert.Equal(t, secret, recovered)

			// Reconstruct with all shares
			recovered, err = CombineGF256(shares)
			require.NoError(t, err)
			assert.Equal(t, secret, recovered)
		})
	}
}

func TestSplitGF256Errors(t *testing.T) {
	tests := []struct {
		name      string
		secret    []byte
		threshold int
		total     int
		wantErr   error
	}{
		{"empty secret", []byte{}, 2, 3, ErrEmptySecret},
		{"nil secret", nil, 2, 3, ErrEmptySecret},
		{"threshold < 2", []byte("test"), 1, 3, ErrInvalidThreshold},
		{"threshold > 255", []byte("test"), 256, 300, ErrInvalidThreshold},
		{"total < threshold", []byte("test"), 5, 3, ErrInvalidTotal},
		{"total > 255", []byte("test"), 3, 256, ErrInvalidTotal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := SplitGF256(tt.secret, tt.threshold, tt.total)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestCombineGF256Errors(t *testing.T) {
	secret := []byte("test secret")
	shares, err := SplitGF256(secret, 3, 5)
	require.NoError(t, err)

	t.Run("insufficient shares", func(t *testing.T) {
		_, err := CombineGF256(shares[:1])
		assert.ErrorIs(t, err, ErrInsufficientShares)
	})

	t.Run("empty shares", func(t *testing.T) {
		_, err := CombineGF256([]GF256Share{})
		assert.ErrorIs(t, err, ErrInsufficientShares)
	})

	t.Run("inconsistent share lengths", func(t *testing.T) {
		badShares := []GF256Share{
			shares[0],
			shares[1][:5],
		}
		_, err := CombineGF256(badShares)
		assert.ErrorIs(t, err, ErrInconsistentShares)
	})

	t.Run("duplicate X coordinates", func(t *testing.T) {
		dupShares := []GF256Share{
			shares[0],
			shares[0].Clone(),
			shares[1],
		}
		_, err := CombineGF256(dupShares)
		assert.ErrorIs(t, err, ErrDuplicateShares)
	})

	t.Run("share too short", func(t *testing.T) {
		_, err := CombineGF256([]GF256Share{{0x01}})
		assert.ErrorIs(t, err, ErrInsufficientShares)
	})

	t.Run("zero X coordinate", func(t *testing.T) {
		badShare := shares[0].Clone()
		badShare[len(badShare)-1] = 0
		_, err := CombineGF256([]GF256Share{badShare, shares[1], shares[2]})
		assert.ErrorIs(t, err, ErrInvalidShareX)
	})
}

func TestCombineGF256WithDifferentSubsets(t *testing.T) {
	secret := []byte("test different subsets with gf256")
	shares, err := SplitGF256(secret, 3, 5)
	require.NoError(t, err)

	subsets := [][]GF256Share{
		{shares[0], shares[1], shares[2]},
		{shares[0], shares[1], shares[3]},
		{shares[0], shares[2], shares[4]},
		{shares[1], shares[3], shares[4]},
		{shares[2], shares[3], shares[4]},
	}

	for i, subset := range subsets {
		t.Run("subset "+string(rune('A'+i)), func(t *testing.T) {
			recovered, err := CombineGF256(subset)
			require.NoError(t, err)
			assert.Equal(t, secret, recovered)
		})
	}
}

func TestGF256ShareMethods(t *testing.T) {
	secret := []byte("test")
	shares, err := SplitGF256(secret, 2, 3)
	require.NoError(t, err)

	share := shares[0]

	t.Run("X", func(t *testing.T) {
		x := share.X()
		assert.NotEqual(t, uint8(0), x)
	})

	t.Run("Y", func(t *testing.T) {
		y := share.Y()
		assert.Len(t, y, len(secret))
	})

	t.Run("Clone", func(t *testing.T) {
		clone := share.Clone()
		assert.True(t, share.Equal(clone))

		// Modify clone, original unchanged
		clone[0] ^= 0xFF
		assert.False(t, share.Equal(clone))
	})

	t.Run("Equal", func(t *testing.T) {
		clone := share.Clone()
		assert.True(t, share.Equal(clone))
		assert.False(t, share.Equal(shares[1]))
		assert.False(t, share.Equal(nil))
		assert.False(t, share.Equal(GF256Share{}))
	})

	t.Run("String", func(t *testing.T) {
		str := share.String()
		assert.NotEmpty(t, str)

		parsed, err := ParseGF256ShareString(str)
		require.NoError(t, err)
		assert.True(t, share.Equal(parsed))
	})
}

func TestGF256ShareEmptyMethods(t *testing.T) {
	var empty GF256Share

	assert.Equal(t, uint8(0), empty.X())
	assert.Nil(t, empty.Y())
	assert.Equal(t, "", empty.String())

	short := GF256Share{0x01}
	assert.Nil(t, short.Y())
}

func TestGF256InterpolateEdgeCases(t *testing.T) {
	t.Run("empty inputs", func(t *testing.T) {
		result := gf256Interpolate([]uint8{}, []uint8{})
		assert.Equal(t, uint8(0), result)
	})

	t.Run("mismatched lengths", func(t *testing.T) {
		result := gf256Interpolate([]uint8{1, 2}, []uint8{1})
		assert.Equal(t, uint8(0), result)
	})
}

func TestParseGF256ShareStringErrors(t *testing.T) {
	t.Run("invalid base64", func(t *testing.T) {
		_, err := ParseGF256ShareString("not-valid-base64!!!")
		assert.ErrorIs(t, err, ErrInvalidShareFormat)
	})

	t.Run("too short", func(t *testing.T) {
		_, err := ParseGF256ShareString("AQ==") // single byte
		assert.ErrorIs(t, err, ErrInvalidShareFormat)
	})

	t.Run("zero X coordinate", func(t *testing.T) {
		_, err := ParseGF256ShareString("AQA=") // [0x01, 0x00] - X is 0
		assert.ErrorIs(t, err, ErrInvalidShareX)
	})

	t.Run("empty base64", func(t *testing.T) {
		_, err := ParseGF256ShareString("")
		assert.ErrorIs(t, err, ErrInvalidShareFormat)
	})
}

func TestCombineGF256ShareTooShort(t *testing.T) {
	shortShares := []GF256Share{{0x01}, {0x02}}
	_, err := CombineGF256(shortShares)
	assert.ErrorIs(t, err, ErrInvalidShareFormat)
}

func TestGF256ShareStringSerialization(t *testing.T) {
	secret := []byte("test serialization roundtrip")
	shares, err := SplitGF256(secret, 3, 5)
	require.NoError(t, err)

	// Serialize all shares to strings
	serialized := make([]string, len(shares))
	for i, share := range shares {
		serialized[i] = share.String()
	}

	// Parse back and reconstruct
	parsedShares := make([]GF256Share, 3)
	for i := range 3 {
		parsedShares[i], err = ParseGF256ShareString(serialized[i])
		require.NoError(t, err)
	}

	recovered, err := CombineGF256(parsedShares)
	require.NoError(t, err)
	assert.Equal(t, secret, recovered)
}

func TestGF256ShareSizeComparison(t *testing.T) {
	secret := make([]byte, 32)
	_, _ = rand.Read(secret)

	// GF256 shares
	gf256Shares, err := SplitGF256(secret, 3, 5)
	require.NoError(t, err)

	// Prime field shares
	primeShares, err := Split(secret[:31], 3, 5)
	require.NoError(t, err)

	gf256Size := len(gf256Shares[0])
	primeSize := len(primeShares[0].Bytes())

	t.Logf("Secret size: %d bytes", len(secret))
	t.Logf("GF256 share size: %d bytes", gf256Size)
	t.Logf("Prime field share size: %d bytes", primeSize)

	assert.Equal(t, len(secret)+1, gf256Size, "GF256 share should be secret+1 bytes")
	assert.Less(t, gf256Size, primeSize, "GF256 shares should be smaller")
}

func BenchmarkSplitGF256(b *testing.B) {
	secret := make([]byte, 32)
	_, _ = rand.Read(secret)

	b.ReportAllocs()
	for b.Loop() {
		_, _ = SplitGF256(secret, 3, 5)
	}
}

func BenchmarkCombineGF256(b *testing.B) {
	secret := make([]byte, 32)
	_, _ = rand.Read(secret)
	shares, _ := SplitGF256(secret, 3, 5)

	b.ReportAllocs()
	for b.Loop() {
		_, _ = CombineGF256(shares[:3])
	}
}

func BenchmarkSplitGF256Large(b *testing.B) {
	secret := make([]byte, 1024)
	_, _ = rand.Read(secret)

	b.ReportAllocs()
	for b.Loop() {
		_, _ = SplitGF256(secret, 3, 5)
	}
}

func BenchmarkCombineGF256Large(b *testing.B) {
	secret := make([]byte, 1024)
	_, _ = rand.Read(secret)
	shares, _ := SplitGF256(secret, 3, 5)

	b.ReportAllocs()
	for b.Loop() {
		_, _ = CombineGF256(shares[:3])
	}
}

func BenchmarkGF256Mult(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		gf256Mult(0x53, 0xCA)
	}
}

func BenchmarkGF256Inverse(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		gf256Inverse(0x53)
	}
}

func FuzzSplitCombineGF256(f *testing.F) {
	f.Add([]byte("test"), 2, 3)
	f.Add([]byte("longer secret for testing"), 3, 5)
	f.Add(make([]byte, 100), 5, 10)

	f.Fuzz(func(t *testing.T, secret []byte, threshold, total int) {
		if len(secret) == 0 || len(secret) > 500 {
			return
		}
		if threshold < 2 || threshold > 50 {
			return
		}
		if total < threshold || total > 50 {
			return
		}

		shares, err := SplitGF256(secret, threshold, total)
		if err != nil {
			return
		}

		recovered, err := CombineGF256(shares[:threshold])
		if err != nil {
			t.Fatalf("combine failed: %v", err)
		}

		if !bytes.Equal(secret, recovered) {
			t.Fatalf("recovered secret doesn't match")
		}
	})
}
