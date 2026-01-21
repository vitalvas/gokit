package shamir

import (
	"bytes"
	"crypto/rand"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSplit(t *testing.T) {
	tests := []struct {
		name      string
		secret    []byte
		threshold int
		total     int
		wantErr   error
	}{
		{
			name:      "valid 2-of-3",
			secret:    []byte("my secret"),
			threshold: 2,
			total:     3,
			wantErr:   nil,
		},
		{
			name:      "valid 3-of-5",
			secret:    []byte("another secret"),
			threshold: 3,
			total:     5,
			wantErr:   nil,
		},
		{
			name:      "valid 5-of-5",
			secret:    []byte("threshold equals total"),
			threshold: 5,
			total:     5,
			wantErr:   nil,
		},
		{
			name:      "empty secret",
			secret:    []byte{},
			threshold: 2,
			total:     3,
			wantErr:   ErrEmptySecret,
		},
		{
			name:      "nil secret",
			secret:    nil,
			threshold: 2,
			total:     3,
			wantErr:   ErrEmptySecret,
		},
		{
			name:      "threshold less than 2",
			secret:    []byte("secret"),
			threshold: 1,
			total:     3,
			wantErr:   ErrInvalidThreshold,
		},
		{
			name:      "total less than threshold",
			secret:    []byte("secret"),
			threshold: 5,
			total:     3,
			wantErr:   ErrInvalidTotal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shares, err := Split(tt.secret, tt.threshold, tt.total)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, shares)
				return
			}

			require.NoError(t, err)
			require.Len(t, shares, tt.total)

			for i, share := range shares {
				assert.Equal(t, int64(i+1), share.X.Int64())
				assert.Equal(t, tt.threshold, share.Threshold)
				assert.Equal(t, tt.total, share.Total)
			}
		})
	}
}

func TestSplitAndCombine(t *testing.T) {
	tests := []struct {
		name      string
		secret    []byte
		threshold int
		total     int
	}{
		{
			name:      "2-of-3 short secret",
			secret:    []byte("hi"),
			threshold: 2,
			total:     3,
		},
		{
			name:      "3-of-5 medium secret",
			secret:    []byte("this is a longer secret message"),
			threshold: 3,
			total:     5,
		},
		{
			name:      "5-of-10 binary secret",
			secret:    []byte{0x00, 0x01, 0x02, 0xff, 0xfe, 0xfd},
			threshold: 5,
			total:     10,
		},
		{
			name:      "2-of-2 minimum",
			secret:    []byte("minimum shares"),
			threshold: 2,
			total:     2,
		},
		{
			name:      "single byte",
			secret:    []byte{0x42},
			threshold: 2,
			total:     3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shares, err := Split(tt.secret, tt.threshold, tt.total)
			require.NoError(t, err)

			// Combine with exact threshold
			recovered, err := Combine(shares[:tt.threshold], len(tt.secret))
			require.NoError(t, err)
			assert.Equal(t, tt.secret, recovered)

			// Combine with all shares
			recovered, err = Combine(shares, len(tt.secret))
			require.NoError(t, err)
			assert.Equal(t, tt.secret, recovered)

			// Combine with more than threshold
			if tt.total > tt.threshold {
				recovered, err = Combine(shares[:tt.threshold+1], len(tt.secret))
				require.NoError(t, err)
				assert.Equal(t, tt.secret, recovered)
			}
		})
	}
}

func TestCombineAuto(t *testing.T) {
	secret := []byte("auto length detection")
	shares, err := Split(secret, 3, 5)
	require.NoError(t, err)

	recovered, err := CombineAuto(shares)
	require.NoError(t, err)
	assert.Equal(t, secret, recovered)
}

func TestCombineWithDifferentShareSubsets(t *testing.T) {
	secret := []byte("test different subsets")
	shares, err := Split(secret, 3, 5)
	require.NoError(t, err)

	subsets := [][]*Share{
		{shares[0], shares[1], shares[2]},
		{shares[0], shares[1], shares[3]},
		{shares[0], shares[2], shares[4]},
		{shares[1], shares[3], shares[4]},
		{shares[2], shares[3], shares[4]},
	}

	for i, subset := range subsets {
		t.Run("subset "+string(rune('A'+i)), func(t *testing.T) {
			recovered, err := Combine(subset, len(secret))
			require.NoError(t, err)
			assert.Equal(t, secret, recovered)
		})
	}
}

func TestCombineErrors(t *testing.T) {
	secret := []byte("error test")
	shares, err := Split(secret, 3, 5)
	require.NoError(t, err)

	tests := []struct {
		name    string
		shares  []*Share
		wantErr error
	}{
		{
			name:    "empty shares",
			shares:  []*Share{},
			wantErr: ErrInsufficientShares,
		},
		{
			name:    "nil shares",
			shares:  nil,
			wantErr: ErrInsufficientShares,
		},
		{
			name:    "insufficient shares",
			shares:  shares[:2],
			wantErr: ErrInsufficientShares,
		},
		{
			name: "duplicate shares",
			shares: []*Share{
				shares[0],
				shares[0].Clone(),
				shares[1],
			},
			wantErr: ErrDuplicateShares,
		},
		{
			name: "inconsistent threshold",
			shares: func() []*Share {
				s := make([]*Share, 3)
				for i := range s {
					s[i] = shares[i].Clone()
				}
				s[1].Threshold = 5
				return s
			}(),
			wantErr: ErrInconsistentShares,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Combine(tt.shares, len(secret))
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestCombineAutoErrors(t *testing.T) {
	secret := []byte("error test")
	shares, err := Split(secret, 3, 5)
	require.NoError(t, err)

	t.Run("empty shares", func(t *testing.T) {
		_, err := CombineAuto([]*Share{})
		assert.ErrorIs(t, err, ErrInsufficientShares)
	})

	t.Run("insufficient shares", func(t *testing.T) {
		_, err := CombineAuto(shares[:2])
		assert.ErrorIs(t, err, ErrInsufficientShares)
	})

	t.Run("duplicate shares", func(t *testing.T) {
		dupShares := []*Share{shares[0], shares[0].Clone(), shares[1]}
		_, err := CombineAuto(dupShares)
		assert.ErrorIs(t, err, ErrDuplicateShares)
	})
}

func TestSplitWithCustomX(t *testing.T) {
	secret := []byte("custom x coordinates")

	xCoords := []*big.Int{
		big.NewInt(100),
		big.NewInt(200),
		big.NewInt(300),
		big.NewInt(400),
		big.NewInt(500),
	}

	shares, err := SplitWithCustomX(secret, 3, xCoords)
	require.NoError(t, err)
	require.Len(t, shares, 5)

	for i, share := range shares {
		assert.Equal(t, 0, share.X.Cmp(xCoords[i]))
	}

	// Reconstruct with any 3 shares
	recovered, err := Combine(shares[:3], len(secret))
	require.NoError(t, err)
	assert.Equal(t, secret, recovered)
}

func TestSplitWithCustomXErrors(t *testing.T) {
	secret := []byte("test")

	tests := []struct {
		name    string
		xCoords []*big.Int
		wantErr error
	}{
		{
			name: "zero x-coordinate",
			xCoords: []*big.Int{
				big.NewInt(0),
				big.NewInt(1),
				big.NewInt(2),
			},
			wantErr: ErrInvalidShareX,
		},
		{
			name: "duplicate x-coordinates",
			xCoords: []*big.Int{
				big.NewInt(1),
				big.NewInt(1),
				big.NewInt(2),
			},
			wantErr: ErrDuplicateShares,
		},
		{
			name: "insufficient x-coordinates",
			xCoords: []*big.Int{
				big.NewInt(1),
			},
			wantErr: ErrInvalidTotal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := SplitWithCustomX(secret, 2, tt.xCoords)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}

	t.Run("empty secret", func(t *testing.T) {
		_, err := SplitWithCustomX([]byte{}, 2, []*big.Int{big.NewInt(1), big.NewInt(2)})
		assert.ErrorIs(t, err, ErrEmptySecret)
	})

	t.Run("invalid threshold", func(t *testing.T) {
		_, err := SplitWithCustomX(secret, 1, []*big.Int{big.NewInt(1), big.NewInt(2)})
		assert.ErrorIs(t, err, ErrInvalidThreshold)
	})
}

func TestLargeSecret(t *testing.T) {
	// Test with a 32-byte secret (like an AES key)
	secret := make([]byte, 32)
	_, err := rand.Read(secret)
	require.NoError(t, err)

	shares, err := Split(secret, 3, 5)
	require.NoError(t, err)

	recovered, err := Combine(shares, 32)
	require.NoError(t, err)
	assert.Equal(t, secret, recovered)
}

func TestSecretWithLeadingZeros(t *testing.T) {
	// Secret with leading zeros
	secret := []byte{0x00, 0x00, 0x01, 0x02, 0x03}

	shares, err := Split(secret, 2, 3)
	require.NoError(t, err)

	// Must specify length to preserve leading zeros
	recovered, err := Combine(shares, len(secret))
	require.NoError(t, err)
	assert.Equal(t, secret, recovered)
}

func TestSplitSecretTooLarge(t *testing.T) {
	secret := make([]byte, 33)
	for i := range secret {
		secret[i] = 0xFF
	}
	_, err := Split(secret, 2, 3)
	assert.ErrorIs(t, err, ErrSecretTooLarge)
}

func TestSplitWithCustomXSecretTooLarge(t *testing.T) {
	secret := make([]byte, 33)
	for i := range secret {
		secret[i] = 0xFF
	}
	xCoords := []*big.Int{big.NewInt(1), big.NewInt(2), big.NewInt(3)}
	_, err := SplitWithCustomX(secret, 2, xCoords)
	assert.ErrorIs(t, err, ErrSecretTooLarge)
}

func BenchmarkSplit(b *testing.B) {
	secret := make([]byte, 32)
	_, _ = rand.Read(secret)

	b.ReportAllocs()
	for b.Loop() {
		_, _ = Split(secret, 3, 5)
	}
}

func BenchmarkCombine(b *testing.B) {
	secret := make([]byte, 32)
	_, _ = rand.Read(secret)
	shares, _ := Split(secret, 3, 5)

	b.ReportAllocs()
	for b.Loop() {
		_, _ = Combine(shares[:3], 32)
	}
}

func FuzzSplitCombine(f *testing.F) {
	f.Add([]byte("test"), 2, 3)
	f.Add([]byte("longer secret"), 3, 5)
	f.Add([]byte{0x00, 0x01}, 2, 2)

	f.Fuzz(func(t *testing.T, secret []byte, threshold, total int) {
		if len(secret) == 0 || len(secret) > 31 {
			return
		}
		if threshold < 2 || threshold > 10 {
			return
		}
		if total < threshold || total > 10 {
			return
		}

		shares, err := Split(secret, threshold, total)
		if err != nil {
			return
		}

		recovered, err := Combine(shares, len(secret))
		if err != nil {
			t.Fatalf("combine failed: %v", err)
		}

		if !bytes.Equal(secret, recovered) {
			t.Fatalf("recovered secret doesn't match: got %x, want %x", recovered, secret)
		}
	})
}
