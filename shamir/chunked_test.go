package shamir

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSplitBytes(t *testing.T) {
	tests := []struct {
		name       string
		secretSize int
		threshold  int
		total      int
	}{
		{"small secret under chunk size", 16, 2, 3},
		{"exact chunk size", 31, 2, 3},
		{"two chunks", 50, 3, 5},
		{"multiple chunks", 100, 3, 5},
		{"large secret", 1024, 5, 10},
		{"very large secret", 10000, 3, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secret := make([]byte, tt.secretSize)
			_, err := rand.Read(secret)
			require.NoError(t, err)

			shares, err := SplitBytes(secret, tt.threshold, tt.total)
			require.NoError(t, err)
			require.Len(t, shares, tt.total)

			// Verify share properties
			expectedChunks := (tt.secretSize + maxChunkSize - 1) / maxChunkSize
			for i, share := range shares {
				assert.Equal(t, int64(i+1), share.X.Int64())
				assert.Equal(t, tt.threshold, share.Threshold)
				assert.Equal(t, tt.total, share.Total)
				assert.Equal(t, tt.secretSize, share.SecretLen)
				assert.Len(t, share.Ys, expectedChunks)
			}

			// Reconstruct with exact threshold
			recovered, err := CombineBytes(shares[:tt.threshold])
			require.NoError(t, err)
			assert.Equal(t, secret, recovered)

			// Reconstruct with all shares
			recovered, err = CombineBytes(shares)
			require.NoError(t, err)
			assert.Equal(t, secret, recovered)
		})
	}
}

func TestSplitBytesErrors(t *testing.T) {
	tests := []struct {
		name      string
		secret    []byte
		threshold int
		total     int
		wantErr   error
	}{
		{"empty secret", []byte{}, 2, 3, ErrEmptySecret},
		{"nil secret", nil, 2, 3, ErrEmptySecret},
		{"threshold less than 2", []byte("secret"), 1, 3, ErrInvalidThreshold},
		{"total less than threshold", []byte("secret"), 5, 3, ErrInvalidTotal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := SplitBytes(tt.secret, tt.threshold, tt.total)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestCombineBytesErrors(t *testing.T) {
	secret := make([]byte, 100)
	_, _ = rand.Read(secret)
	shares, err := SplitBytes(secret, 3, 5)
	require.NoError(t, err)

	t.Run("empty shares", func(t *testing.T) {
		_, err := CombineBytes([]*ChunkedShare{})
		assert.ErrorIs(t, err, ErrInsufficientShares)
	})

	t.Run("insufficient shares", func(t *testing.T) {
		_, err := CombineBytes(shares[:2])
		assert.ErrorIs(t, err, ErrInsufficientShares)
	})

	t.Run("duplicate shares", func(t *testing.T) {
		dupShares := []*ChunkedShare{shares[0], shares[0].Clone(), shares[1]}
		_, err := CombineBytes(dupShares)
		assert.ErrorIs(t, err, ErrDuplicateShares)
	})

	t.Run("inconsistent threshold", func(t *testing.T) {
		badShares := make([]*ChunkedShare, 3)
		for i := range badShares {
			badShares[i] = shares[i].Clone()
		}
		badShares[1].Threshold = 5
		_, err := CombineBytes(badShares)
		assert.ErrorIs(t, err, ErrInconsistentShares)
	})

	t.Run("inconsistent chunk count", func(t *testing.T) {
		badShares := make([]*ChunkedShare, 3)
		for i := range badShares {
			badShares[i] = shares[i].Clone()
		}
		badShares[1].Ys = badShares[1].Ys[:1]
		_, err := CombineBytes(badShares)
		assert.ErrorIs(t, err, ErrInconsistentShares)
	})

	t.Run("inconsistent secret length", func(t *testing.T) {
		badShares := make([]*ChunkedShare, 3)
		for i := range badShares {
			badShares[i] = shares[i].Clone()
		}
		badShares[1].SecretLen = 999
		_, err := CombineBytes(badShares)
		assert.ErrorIs(t, err, ErrInconsistentShares)
	})
}

func TestChunkedShareSerialization(t *testing.T) {
	secret := make([]byte, 100)
	_, _ = rand.Read(secret)
	shares, err := SplitBytes(secret, 3, 5)
	require.NoError(t, err)

	for _, share := range shares {
		t.Run("binary roundtrip", func(t *testing.T) {
			data := share.Bytes()
			parsed, err := ParseChunkedShare(data)
			require.NoError(t, err)
			assert.True(t, share.Equal(parsed))
		})

		t.Run("string roundtrip", func(t *testing.T) {
			str := share.String()
			parsed, err := ParseChunkedShareString(str)
			require.NoError(t, err)
			assert.True(t, share.Equal(parsed))
		})
	}

	t.Run("reconstruct from serialized", func(t *testing.T) {
		serialized := make([]string, 3)
		for i := range 3 {
			serialized[i] = shares[i].String()
		}

		parsedShares := make([]*ChunkedShare, 3)
		for i, s := range serialized {
			parsedShares[i], err = ParseChunkedShareString(s)
			require.NoError(t, err)
		}

		recovered, err := CombineBytes(parsedShares)
		require.NoError(t, err)
		assert.Equal(t, secret, recovered)
	})
}

func TestParseChunkedShareErrors(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantErr error
	}{
		{"too short", []byte{0x02, 0x00}, ErrInvalidShareFormat},
		{"wrong version", make([]byte, chunkedShareHeaderSize), ErrUnsupportedVersion},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseChunkedShare(tt.data)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}

	t.Run("invalid base64", func(t *testing.T) {
		_, err := ParseChunkedShareString("not-valid-base64!!!")
		assert.ErrorIs(t, err, ErrInvalidShareFormat)
	})

	t.Run("truncated x data", func(t *testing.T) {
		buf := make([]byte, chunkedShareHeaderSize+1)
		buf[0] = chunkedShareVersion
		binary.BigEndian.PutUint16(buf[1:3], 3)    // threshold
		binary.BigEndian.PutUint16(buf[3:5], 5)    // total
		binary.BigEndian.PutUint32(buf[5:9], 100)  // secretLen
		binary.BigEndian.PutUint16(buf[9:11], 2)   // numChunks
		binary.BigEndian.PutUint16(buf[11:13], 10) // xLen = 10 but only 1 byte available
		_, err := ParseChunkedShare(buf)
		assert.ErrorIs(t, err, ErrInvalidShareFormat)
	})

	t.Run("zero X coordinate", func(t *testing.T) {
		buf := make([]byte, chunkedShareHeaderSize+1+4)
		buf[0] = chunkedShareVersion
		binary.BigEndian.PutUint16(buf[1:3], 2)   // threshold
		binary.BigEndian.PutUint16(buf[3:5], 3)   // total
		binary.BigEndian.PutUint32(buf[5:9], 10)  // secretLen
		binary.BigEndian.PutUint16(buf[9:11], 1)  // numChunks
		binary.BigEndian.PutUint16(buf[11:13], 0) // xLen = 0, so X will be 0
		_, err := ParseChunkedShare(buf)
		assert.ErrorIs(t, err, ErrInvalidShareX)
	})

	t.Run("truncated y length", func(t *testing.T) {
		buf := make([]byte, chunkedShareHeaderSize+1)
		buf[0] = chunkedShareVersion
		binary.BigEndian.PutUint16(buf[1:3], 2)   // threshold
		binary.BigEndian.PutUint16(buf[3:5], 3)   // total
		binary.BigEndian.PutUint32(buf[5:9], 10)  // secretLen
		binary.BigEndian.PutUint16(buf[9:11], 1)  // numChunks
		binary.BigEndian.PutUint16(buf[11:13], 1) // xLen = 1
		buf[chunkedShareHeaderSize] = 0x01        // x = 1
		_, err := ParseChunkedShare(buf)
		assert.ErrorIs(t, err, ErrInvalidShareFormat)
	})

	t.Run("truncated y data", func(t *testing.T) {
		buf := make([]byte, chunkedShareHeaderSize+1+2)
		buf[0] = chunkedShareVersion
		binary.BigEndian.PutUint16(buf[1:3], 2)                        // threshold
		binary.BigEndian.PutUint16(buf[3:5], 3)                        // total
		binary.BigEndian.PutUint32(buf[5:9], 10)                       // secretLen
		binary.BigEndian.PutUint16(buf[9:11], 1)                       // numChunks
		binary.BigEndian.PutUint16(buf[11:13], 1)                      // xLen = 1
		buf[chunkedShareHeaderSize] = 0x01                             // x = 1
		binary.BigEndian.PutUint16(buf[chunkedShareHeaderSize+1:], 10) // yLen = 10 but no data
		_, err := ParseChunkedShare(buf)
		assert.ErrorIs(t, err, ErrInvalidShareFormat)
	})
}

func TestChunkedShareClone(t *testing.T) {
	secret := make([]byte, 100)
	_, _ = rand.Read(secret)
	shares, err := SplitBytes(secret, 3, 5)
	require.NoError(t, err)

	original := shares[0]
	clone := original.Clone()

	assert.True(t, original.Equal(clone))

	// Modify clone and verify original is unchanged
	clone.X.SetInt64(100)
	clone.Ys[0].SetInt64(999)
	assert.False(t, original.Equal(clone))
}

func TestChunkedShareEqual(t *testing.T) {
	secret := make([]byte, 100)
	_, _ = rand.Read(secret)
	shares, err := SplitBytes(secret, 3, 5)
	require.NoError(t, err)

	t.Run("equal shares", func(t *testing.T) {
		clone := shares[0].Clone()
		assert.True(t, shares[0].Equal(clone))
	})

	t.Run("different X", func(t *testing.T) {
		assert.False(t, shares[0].Equal(shares[1]))
	})

	t.Run("nil comparison", func(t *testing.T) {
		assert.False(t, shares[0].Equal(nil))
		assert.True(t, (*ChunkedShare)(nil).Equal(nil))
	})

	t.Run("different threshold", func(t *testing.T) {
		clone := shares[0].Clone()
		clone.Threshold = 99
		assert.False(t, shares[0].Equal(clone))
	})

	t.Run("different total", func(t *testing.T) {
		clone := shares[0].Clone()
		clone.Total = 99
		assert.False(t, shares[0].Equal(clone))
	})

	t.Run("different secret length", func(t *testing.T) {
		clone := shares[0].Clone()
		clone.SecretLen = 99
		assert.False(t, shares[0].Equal(clone))
	})

	t.Run("different chunk count", func(t *testing.T) {
		clone := shares[0].Clone()
		clone.Ys = clone.Ys[:1]
		assert.False(t, shares[0].Equal(clone))
	})

	t.Run("different Y values", func(t *testing.T) {
		clone := shares[0].Clone()
		clone.Ys[0] = big.NewInt(999999)
		assert.False(t, shares[0].Equal(clone))
	})
}

func TestSplitIntoChunks(t *testing.T) {
	tests := []struct {
		name      string
		dataLen   int
		chunkSize int
		expected  int
	}{
		{"empty", 0, 31, 0},
		{"smaller than chunk", 10, 31, 1},
		{"exact chunk", 31, 31, 1},
		{"two chunks", 32, 31, 2},
		{"multiple chunks", 100, 31, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := make([]byte, tt.dataLen)
			chunks := splitIntoChunks(data, tt.chunkSize)

			if tt.expected == 0 {
				assert.Nil(t, chunks)
			} else {
				assert.Len(t, chunks, tt.expected)
			}
		})
	}
}

func TestCombineBytesWithDifferentSubsets(t *testing.T) {
	secret := make([]byte, 100)
	_, _ = rand.Read(secret)
	shares, err := SplitBytes(secret, 3, 5)
	require.NoError(t, err)

	subsets := [][]*ChunkedShare{
		{shares[0], shares[1], shares[2]},
		{shares[0], shares[1], shares[3]},
		{shares[0], shares[2], shares[4]},
		{shares[1], shares[3], shares[4]},
		{shares[2], shares[3], shares[4]},
	}

	for i, subset := range subsets {
		t.Run("subset "+string(rune('A'+i)), func(t *testing.T) {
			recovered, err := CombineBytes(subset)
			require.NoError(t, err)
			assert.Equal(t, secret, recovered)
		})
	}
}

func BenchmarkSplitBytes(b *testing.B) {
	secret := make([]byte, 1024)
	_, _ = rand.Read(secret)

	b.ReportAllocs()
	for b.Loop() {
		_, _ = SplitBytes(secret, 3, 5)
	}
}

func BenchmarkCombineBytes(b *testing.B) {
	secret := make([]byte, 1024)
	_, _ = rand.Read(secret)
	shares, _ := SplitBytes(secret, 3, 5)

	b.ReportAllocs()
	for b.Loop() {
		_, _ = CombineBytes(shares[:3])
	}
}

func BenchmarkChunkedShareSerialize(b *testing.B) {
	secret := make([]byte, 1024)
	_, _ = rand.Read(secret)
	shares, _ := SplitBytes(secret, 3, 5)

	b.ReportAllocs()
	for b.Loop() {
		_ = shares[0].Bytes()
	}
}

func BenchmarkChunkedShareParse(b *testing.B) {
	secret := make([]byte, 1024)
	_, _ = rand.Read(secret)
	shares, _ := SplitBytes(secret, 3, 5)
	data := shares[0].Bytes()

	b.ReportAllocs()
	for b.Loop() {
		_, _ = ParseChunkedShare(data)
	}
}

func FuzzSplitCombineBytes(f *testing.F) {
	f.Add([]byte("test"), 2, 3)
	f.Add([]byte("longer secret that spans multiple chunks for testing"), 3, 5)
	f.Add(make([]byte, 100), 2, 2)

	f.Fuzz(func(t *testing.T, secret []byte, threshold, total int) {
		if len(secret) == 0 || len(secret) > 500 {
			return
		}
		if threshold < 2 || threshold > 10 {
			return
		}
		if total < threshold || total > 10 {
			return
		}

		shares, err := SplitBytes(secret, threshold, total)
		if err != nil {
			return
		}

		recovered, err := CombineBytes(shares)
		if err != nil {
			t.Fatalf("combine failed: %v", err)
		}

		if !bytes.Equal(secret, recovered) {
			t.Fatalf("recovered secret doesn't match: got %x, want %x", recovered, secret)
		}
	})
}
