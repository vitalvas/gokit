package fastcdc

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHashDataSHA256(t *testing.T) {
	data := []byte("test data")
	hash := HashDataSHA256(data)
	assert.Equal(t, 32, len(hash))

	// Same data should produce same hash
	hash2 := HashDataSHA256(data)
	assert.Equal(t, hash, hash2)
}

func TestHashDataSHA384(t *testing.T) {
	data := []byte("test data")
	hash := HashDataSHA384(data)
	assert.Equal(t, 48, len(hash))

	// Same data should produce same hash
	hash2 := HashDataSHA384(data)
	assert.Equal(t, hash, hash2)
}

func TestHashDataSHA512(t *testing.T) {
	data := []byte("test data")
	hash := HashDataSHA512(data)
	assert.Equal(t, 64, len(hash))

	// Same data should produce same hash
	hash2 := HashDataSHA512(data)
	assert.Equal(t, hash, hash2)
}

func TestComputeHash32(t *testing.T) {
	data := []byte("test data")

	tests := []struct {
		algorithm HashAlgorithm
	}{
		{HashSHA256},
		{HashSHA384},
		{HashSHA512},
		{HashAlgorithm(99)}, // Unknown algorithm defaults to SHA256
	}

	for _, tt := range tests {
		hash := ComputeHash32(data, tt.algorithm)
		assert.Equal(t, 32, len(hash))
		assert.NotEqual(t, [32]byte{}, hash)
	}
}

func TestComputeHashFull(t *testing.T) {
	data := []byte("test data")

	tests := []struct {
		algorithm    HashAlgorithm
		expectedSize int
	}{
		{HashSHA256, 32},
		{HashSHA384, 48},
		{HashSHA512, 64},
		{HashAlgorithm(99), 32}, // Unknown defaults to SHA256
	}

	for _, tt := range tests {
		hash, size := ComputeHashFull(data, tt.algorithm)
		assert.Equal(t, tt.expectedSize, size)
		assert.Equal(t, 64, len(hash)) // Always returns [64]byte
		assert.NotEqual(t, [64]byte{}, hash)
	}
}

func TestComputeHash(t *testing.T) {
	data := []byte("test data")

	tests := []struct {
		algorithm    HashAlgorithm
		expectedSize int
	}{
		{HashSHA256, 32},
		{HashSHA384, 48},
		{HashSHA512, 64},
		{HashAlgorithm(99), 32}, // Unknown defaults to SHA256
	}

	for _, tt := range tests {
		hash := ComputeHash(data, tt.algorithm)
		assert.Equal(t, tt.expectedSize, len(hash))
	}
}

func TestHashAlgorithmSizeDefault(t *testing.T) {
	// Test unknown algorithm defaults to 32
	unknown := HashAlgorithm(99)
	assert.Equal(t, 32, unknown.Size())
}

func TestHasherReset(t *testing.T) {
	// Test that reset works for unknown algorithm (defaults to SHA256)
	hasher := NewHasher(HashAlgorithm(99))
	assert.NotNil(t, hasher)

	hash := hasher.Sum32([]byte("test"))
	assert.Equal(t, 32, len(hash))
	assert.NotEqual(t, [32]byte{}, hash)
}

func TestHashData(t *testing.T) {
	data := []byte("test data for hashing")

	tests := []struct {
		name      string
		algorithm HashAlgorithm
	}{
		{"sha256", HashSHA256},
		{"sha384", HashSHA384},
		{"sha512", HashSHA512},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash1, size1 := HashData(data, tt.algorithm)
			hash2, size2 := HashData(data, tt.algorithm)
			assert.Equal(t, hash1, hash2)
			assert.Equal(t, size1, size2)
			assert.Equal(t, tt.algorithm.Size(), size1)

			// Different data should produce different hash
			differentData := []byte("different test data")
			hash3, _ := HashData(differentData, tt.algorithm)
			assert.NotEqual(t, hash1, hash3)
		})
	}

	// Different algorithms should produce different hashes
	sha256Hash, _ := HashData(data, HashSHA256)
	sha384Hash, _ := HashData(data, HashSHA384)
	sha512Hash, _ := HashData(data, HashSHA512)

	assert.NotEqual(t, sha256Hash, sha384Hash)
	assert.NotEqual(t, sha256Hash, sha512Hash)
	assert.NotEqual(t, sha384Hash, sha512Hash)
}

func TestHashAlgorithm_String(t *testing.T) {
	tests := []struct {
		algorithm HashAlgorithm
		expected  string
	}{
		{HashSHA256, "sha256"},
		{HashSHA384, "sha384"},
		{HashSHA512, "sha512"},
		{HashAlgorithm(99), "unknown"},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, tt.algorithm.String())
	}
}

func TestHashAlgorithm_Size(t *testing.T) {
	tests := []struct {
		algorithm HashAlgorithm
		expected  int
	}{
		{HashSHA256, 32},
		{HashSHA384, 48},
		{HashSHA512, 64},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, tt.algorithm.Size())
	}
}

func TestHasher(t *testing.T) {
	data := []byte("test hasher data")

	tests := []struct {
		name      string
		algorithm HashAlgorithm
	}{
		{"sha256", HashSHA256},
		{"sha384", HashSHA384},
		{"sha512", HashSHA512},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasher := NewHasher(tt.algorithm)

			assert.Equal(t, tt.algorithm, hasher.Algorithm())
			assert.Equal(t, tt.algorithm.Size(), hasher.HashSize())

			// Test Sum
			sum := hasher.Sum(data)
			assert.Equal(t, tt.algorithm.Size(), len(sum))

			// Test Sum32
			_ = hasher.Sum32(data)

			// Test Sum64
			_ = hasher.Sum64(data)

			// Verify consistency
			sum2 := hasher.Sum(data)
			assert.Equal(t, sum, sum2)
		})
	}
}
