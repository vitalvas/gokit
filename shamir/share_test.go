package shamir

import (
	"crypto/rand"
	"encoding/binary"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShareSerialization(t *testing.T) {
	secret := []byte("serialization test")
	shares, err := Split(secret, 3, 5)
	require.NoError(t, err)

	for _, share := range shares {
		t.Run("binary roundtrip", func(t *testing.T) {
			data := share.Bytes()
			parsed, err := ParseShare(data)
			require.NoError(t, err)
			assert.True(t, share.Equal(parsed))
		})

		t.Run("string roundtrip", func(t *testing.T) {
			str := share.String()
			parsedFromStr, err := ParseShareString(str)
			require.NoError(t, err)
			assert.True(t, share.Equal(parsedFromStr))
		})
	}

	t.Run("reconstruct from serialized", func(t *testing.T) {
		serialized := make([]string, 3)
		for i := range 3 {
			serialized[i] = shares[i].String()
		}

		parsedShares := make([]*Share, 3)
		for i, s := range serialized {
			parsedShares[i], err = ParseShareString(s)
			require.NoError(t, err)
		}

		recovered, err := Combine(parsedShares, len(secret))
		require.NoError(t, err)
		assert.Equal(t, secret, recovered)
	})
}

func TestParseShareErrors(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantErr error
	}{
		{
			name:    "too short",
			data:    []byte{0x01, 0x02},
			wantErr: ErrInvalidShareFormat,
		},
		{
			name:    "wrong version",
			data:    []byte{0x00, 0x00, 0x03, 0x00, 0x05, 0x00, 0x01, 0x00, 0x01, 0x01, 0x02},
			wantErr: ErrUnsupportedVersion,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseShare(tt.data)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}

	t.Run("truncated data", func(t *testing.T) {
		buf := make([]byte, shareHeaderSize+5)
		buf[0] = shareVersion
		binary.BigEndian.PutUint16(buf[1:3], 2)  // threshold
		binary.BigEndian.PutUint16(buf[3:5], 3)  // total
		binary.BigEndian.PutUint16(buf[5:7], 10) // xLen = 10 but not enough data
		binary.BigEndian.PutUint16(buf[7:9], 10) // yLen = 10
		_, err := ParseShare(buf)
		assert.ErrorIs(t, err, ErrInvalidShareFormat)
	})

	t.Run("zero X coordinate", func(t *testing.T) {
		buf := make([]byte, shareHeaderSize+1)
		buf[0] = shareVersion
		binary.BigEndian.PutUint16(buf[1:3], 2) // threshold
		binary.BigEndian.PutUint16(buf[3:5], 3) // total
		binary.BigEndian.PutUint16(buf[5:7], 0) // xLen = 0
		binary.BigEndian.PutUint16(buf[7:9], 1) // yLen = 1
		buf[shareHeaderSize] = 0x42             // y = 0x42
		_, err := ParseShare(buf)
		assert.ErrorIs(t, err, ErrInvalidShareX)
	})
}

func TestParseShareStringError(t *testing.T) {
	_, err := ParseShareString("not-valid-base64!!!")
	assert.ErrorIs(t, err, ErrInvalidShareFormat)
}

func TestShareClone(t *testing.T) {
	original := &Share{
		X:         big.NewInt(42),
		Y:         big.NewInt(1234567890),
		Threshold: 3,
		Total:     5,
	}

	clone := original.Clone()

	assert.True(t, original.Equal(clone))

	// Modify clone and verify original is unchanged
	clone.X.SetInt64(100)
	assert.NotEqual(t, 0, original.X.Cmp(clone.X))
}

func TestShareEqual(t *testing.T) {
	share1 := &Share{
		X:         big.NewInt(1),
		Y:         big.NewInt(100),
		Threshold: 2,
		Total:     3,
	}

	share2 := &Share{
		X:         big.NewInt(1),
		Y:         big.NewInt(100),
		Threshold: 2,
		Total:     3,
	}

	share3 := &Share{
		X:         big.NewInt(2),
		Y:         big.NewInt(100),
		Threshold: 2,
		Total:     3,
	}

	t.Run("equal shares", func(t *testing.T) {
		assert.True(t, share1.Equal(share2))
	})

	t.Run("different X", func(t *testing.T) {
		assert.False(t, share1.Equal(share3))
	})

	t.Run("nil comparison", func(t *testing.T) {
		assert.False(t, share1.Equal(nil))
		assert.True(t, (*Share)(nil).Equal(nil))
	})

	t.Run("different Y", func(t *testing.T) {
		s := share1.Clone()
		s.Y = big.NewInt(999)
		assert.False(t, share1.Equal(s))
	})

	t.Run("different threshold", func(t *testing.T) {
		s := share1.Clone()
		s.Threshold = 5
		assert.False(t, share1.Equal(s))
	})

	t.Run("different total", func(t *testing.T) {
		s := share1.Clone()
		s.Total = 10
		assert.False(t, share1.Equal(s))
	})
}

func BenchmarkShareSerialize(b *testing.B) {
	secret := make([]byte, 32)
	_, _ = rand.Read(secret)
	shares, _ := Split(secret, 3, 5)

	b.ReportAllocs()
	for b.Loop() {
		_ = shares[0].Bytes()
	}
}

func BenchmarkShareParse(b *testing.B) {
	secret := make([]byte, 32)
	_, _ = rand.Read(secret)
	shares, _ := Split(secret, 3, 5)
	data := shares[0].Bytes()

	b.ReportAllocs()
	for b.Loop() {
		_, _ = ParseShare(data)
	}
}
