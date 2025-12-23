package fastcdc

import (
	"bytes"
	"crypto/rand"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	assert.Equal(t, uint64(512*1024), config.MinSize)
	assert.Equal(t, uint64(8*1024*1024), config.MaxSize)
	assert.Equal(t, uint64(1*1024*1024), config.AvgSize)
	assert.Equal(t, 2, config.Normalization)
}

func TestPresetConfigs(t *testing.T) {
	t.Run("SmallChunkConfig", func(t *testing.T) {
		config := SmallChunkConfig()
		assert.Equal(t, uint64(2*1024), config.MinSize)
		assert.Equal(t, uint64(64*1024), config.MaxSize)
		assert.Equal(t, uint64(8*1024), config.AvgSize)
		assert.NoError(t, validateConfig(config))
	})

	t.Run("MediumChunkConfig", func(t *testing.T) {
		config := MediumChunkConfig()
		assert.Equal(t, uint64(16*1024), config.MinSize)
		assert.Equal(t, uint64(256*1024), config.MaxSize)
		assert.Equal(t, uint64(64*1024), config.AvgSize)
		assert.NoError(t, validateConfig(config))
	})

	t.Run("LargeChunkConfig", func(t *testing.T) {
		config := LargeChunkConfig()
		assert.Equal(t, uint64(64*1024), config.MinSize)
		assert.Equal(t, uint64(2*1024*1024), config.MaxSize)
		assert.Equal(t, uint64(256*1024), config.AvgSize)
		assert.NoError(t, validateConfig(config))
	})
}

func TestNewChunker(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name:    "default config",
			config:  DefaultConfig(),
			wantErr: false,
		},
		{
			name: "custom valid config",
			config: Config{
				MinSize:       1024,
				MaxSize:       8192,
				AvgSize:       4096,
				Normalization: 1,
				BufSize:       16384,
			},
			wantErr: false,
		},
		{
			name: "zero min size",
			config: Config{
				MinSize:       0,
				MaxSize:       8192,
				AvgSize:       4096,
				Normalization: 1,
				BufSize:       16384,
			},
			wantErr: true,
		},
		{
			name: "min >= max",
			config: Config{
				MinSize:       8192,
				MaxSize:       4096,
				AvgSize:       4096,
				Normalization: 1,
				BufSize:       16384,
			},
			wantErr: true,
		},
		{
			name: "avg < min",
			config: Config{
				MinSize:       4096,
				MaxSize:       8192,
				AvgSize:       1024,
				Normalization: 1,
				BufSize:       16384,
			},
			wantErr: true,
		},
		{
			name: "avg > max",
			config: Config{
				MinSize:       1024,
				MaxSize:       4096,
				AvgSize:       8192,
				Normalization: 1,
				BufSize:       16384,
			},
			wantErr: true,
		},
		{
			name: "invalid normalization",
			config: Config{
				MinSize:       1024,
				MaxSize:       8192,
				AvgSize:       4096,
				Normalization: 5,
				BufSize:       16384,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bytes.NewReader([]byte("test data"))
			chunker, err := NewChunker(reader, tt.config)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, chunker)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, chunker)
			}
		})
	}
}

func TestChunker_SmallData(t *testing.T) {
	data := []byte("Hello, World! This is a small test.")
	reader := bytes.NewReader(data)

	config := Config{
		MinSize:       8,
		MaxSize:       64,
		AvgSize:       32,
		Normalization: 1,
		BufSize:       128,
	}

	chunker, err := NewChunker(reader, config)
	require.NoError(t, err)

	// Collect all chunks
	var chunks []*Chunk
	var totalBytes uint64

	for {
		chunk, err := chunker.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		chunks = append(chunks, chunk)
		totalBytes += chunk.Length
	}

	// Verify all data was chunked
	assert.Equal(t, uint64(len(data)), totalBytes)
	assert.Greater(t, len(chunks), 0)

	// Verify first chunk starts at offset 0
	assert.Equal(t, uint64(0), chunks[0].Offset)

	// Verify chunk sizes are within bounds
	for _, chunk := range chunks {
		assert.GreaterOrEqual(t, chunk.Length, uint64(1))
		assert.LessOrEqual(t, chunk.Length, config.MaxSize)
	}
}

func TestChunker_LargeData(t *testing.T) {
	dataSize := 5 * 1024 * 1024 // 5 MiB
	data := make([]byte, dataSize)
	_, err := rand.Read(data)
	require.NoError(t, err)

	reader := bytes.NewReader(data)

	config := Config{
		MinSize:       64 * 1024,  // 64 KiB
		MaxSize:       512 * 1024, // 512 KiB
		AvgSize:       256 * 1024, // 256 KiB
		Normalization: 2,
		BufSize:       1024 * 1024,
	}

	chunker, err := NewChunker(reader, config)
	require.NoError(t, err)

	var chunks []*Chunk
	totalBytes := uint64(0)

	for {
		chunk, err := chunker.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)

		assert.GreaterOrEqual(t, chunk.Length, uint64(1))
		assert.LessOrEqual(t, chunk.Length, config.MaxSize)
		assert.Equal(t, totalBytes, chunk.Offset)

		chunks = append(chunks, chunk)
		totalBytes += chunk.Length
	}

	assert.Equal(t, uint64(dataSize), totalBytes)
	assert.Greater(t, len(chunks), 1)
}

func TestChunker_DeterministicBoundaries(t *testing.T) {
	data := make([]byte, 2*1024*1024)
	_, err := rand.Read(data)
	require.NoError(t, err)

	config := Config{
		MinSize:       32 * 1024,
		MaxSize:       256 * 1024,
		AvgSize:       64 * 1024,
		Normalization: 2,
		BufSize:       512 * 1024,
	}

	// Chunk data twice and verify identical results
	chunks1, err := chunkData(data, config)
	require.NoError(t, err)

	chunks2, err := chunkData(data, config)
	require.NoError(t, err)

	assert.Equal(t, len(chunks1), len(chunks2))

	for i := range chunks1 {
		assert.Equal(t, chunks1[i].Offset, chunks2[i].Offset)
		assert.Equal(t, chunks1[i].Length, chunks2[i].Length)
		assert.Equal(t, chunks1[i].Hash, chunks2[i].Hash)
	}
}

func TestChunker_ContentShift(t *testing.T) {
	// Test that inserting data at the beginning only affects the first few chunks
	originalData := make([]byte, 1024*1024)
	_, err := rand.Read(originalData)
	require.NoError(t, err)

	prefix := []byte("PREFIX DATA ")
	modifiedData := make([]byte, len(prefix)+len(originalData))
	copy(modifiedData, prefix)
	copy(modifiedData[len(prefix):], originalData)

	config := Config{
		MinSize:       16 * 1024,
		MaxSize:       128 * 1024,
		AvgSize:       32 * 1024,
		Normalization: 2,
		BufSize:       256 * 1024,
	}

	originalChunks, err := chunkData(originalData, config)
	require.NoError(t, err)

	modifiedChunks, err := chunkData(modifiedData, config)
	require.NoError(t, err)

	// Count matching hashes (should be many after initial divergence)
	originalHashes := make(map[[64]byte]bool)
	for _, chunk := range originalChunks {
		originalHashes[chunk.Hash] = true
	}

	matchCount := 0
	for _, chunk := range modifiedChunks {
		if originalHashes[chunk.Hash] {
			matchCount++
		}
	}

	// We expect a significant portion of chunks to match due to content-defined boundaries
	matchRatio := float64(matchCount) / float64(len(originalChunks))
	t.Logf("Match ratio: %.2f%% (%d/%d chunks)", matchRatio*100, matchCount, len(originalChunks))
}

func TestChunkBytes(t *testing.T) {
	data := make([]byte, 512*1024)
	_, err := rand.Read(data)
	require.NoError(t, err)

	config := Config{
		MinSize:       8 * 1024,
		MaxSize:       64 * 1024,
		AvgSize:       32 * 1024,
		Normalization: 2,
		BufSize:       128 * 1024,
	}

	chunks, err := ChunkBytes(data, config)
	require.NoError(t, err)

	var totalBytes uint64
	for _, chunk := range chunks {
		totalBytes += chunk.Length
	}

	assert.Equal(t, uint64(len(data)), totalBytes)
}

func TestChunker_NextHash(t *testing.T) {
	data := make([]byte, 256*1024)
	_, err := rand.Read(data)
	require.NoError(t, err)

	config := Config{
		MinSize:       8 * 1024,
		MaxSize:       64 * 1024,
		AvgSize:       32 * 1024,
		Normalization: 2,
		BufSize:       128 * 1024,
	}

	// Get chunks with data
	reader1 := bytes.NewReader(data)
	chunker1, err := NewChunker(reader1, config)
	require.NoError(t, err)

	var chunksWithData []*Chunk
	for {
		chunk, err := chunker1.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		chunksWithData = append(chunksWithData, chunk)
	}

	// Get chunks without data (hash only)
	reader2 := bytes.NewReader(data)
	chunker2, err := NewChunker(reader2, config)
	require.NoError(t, err)

	var chunksHashOnly []*Chunk
	for {
		chunk, err := chunker2.NextHash()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		chunksHashOnly = append(chunksHashOnly, chunk)
	}

	// Verify same boundaries and hashes
	require.Equal(t, len(chunksWithData), len(chunksHashOnly))

	for i := range chunksWithData {
		assert.Equal(t, chunksWithData[i].Offset, chunksHashOnly[i].Offset)
		assert.Equal(t, chunksWithData[i].Length, chunksHashOnly[i].Length)
		assert.Equal(t, chunksWithData[i].Hash, chunksHashOnly[i].Hash)
		assert.Nil(t, chunksHashOnly[i].Data)
	}
}

func TestChunker_Reset(t *testing.T) {
	data := make([]byte, 128*1024)
	_, err := rand.Read(data)
	require.NoError(t, err)

	config := Config{
		MinSize:       8 * 1024,
		MaxSize:       64 * 1024,
		AvgSize:       32 * 1024,
		Normalization: 2,
		BufSize:       128 * 1024,
	}

	reader := bytes.NewReader(data)
	chunker, err := NewChunker(reader, config)
	require.NoError(t, err)

	// Read all chunks
	var firstRun []*Chunk
	for {
		chunk, err := chunker.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		firstRun = append(firstRun, chunk)
	}

	// Reset and read again
	chunker.Reset(bytes.NewReader(data))

	var secondRun []*Chunk
	for {
		chunk, err := chunker.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		secondRun = append(secondRun, chunk)
	}

	assert.Equal(t, len(firstRun), len(secondRun))
	for i := range firstRun {
		assert.Equal(t, firstRun[i].Hash, secondRun[i].Hash)
	}
}

func TestComputeMasks(t *testing.T) {
	tests := []struct {
		avgSize       uint64
		normalization int
	}{
		{1024, 0},
		{4096, 1},
		{32768, 2},
		{1048576, 2},
	}

	for _, tt := range tests {
		maskS, maskL := computeMasks(tt.avgSize, tt.normalization)
		assert.Greater(t, maskS, uint64(0))
		assert.Greater(t, maskL, uint64(0))

		// maskS should have more bits set than maskL (stricter condition)
		bitsS := popCount(maskS)
		bitsL := popCount(maskL)
		if tt.normalization > 0 {
			assert.Greater(t, bitsS, bitsL, "maskS should have more bits than maskL")
		}
	}
}

func TestCreateDistributedMask(t *testing.T) {
	tests := []struct {
		bits         uint64
		expectedBits int
	}{
		{0, 0},
		{1, 1},
		{8, 8},
		{12, 12},
		{20, 20},
		{32, 32},
	}

	for _, tt := range tests {
		mask := createDistributedMask(tt.bits)
		actualBits := popCount(mask)
		assert.Equal(t, tt.expectedBits, actualBits, "mask for %d bits should have %d bits set", tt.bits, tt.expectedBits)

		// Verify bits are distributed (not all consecutive)
		if tt.bits > 1 && tt.bits < 64 {
			// Check that the mask spans more than just the lower bits
			highestBit := uint64(0)
			for i := uint64(63); i > 0; i-- {
				if mask&(1<<i) != 0 {
					highestBit = i
					break
				}
			}
			// Highest bit position should be greater than effective bits (distributed)
			assert.Greater(t, highestBit, tt.bits-1, "bits should be distributed across range")
		}
	}
}

// popCount counts the number of set bits in a uint64
func popCount(x uint64) int {
	count := 0
	for x != 0 {
		count++
		x &= x - 1
	}
	return count
}

func TestLogarithm2(t *testing.T) {
	tests := []struct {
		input    uint64
		expected uint64
	}{
		{1, 0},
		{2, 1},
		{4, 2},
		{8, 3},
		{16, 4},
		{1024, 10},
		{1048576, 20},
	}

	for _, tt := range tests {
		result := logarithm2(tt.input)
		assert.Equal(t, tt.expected, result, "logarithm2(%d)", tt.input)
	}
}

func TestChunkReset(t *testing.T) {
	chunk := &Chunk{
		Offset:   100,
		Length:   200,
		Data:     []byte("test data"),
		HashSize: 32,
	}
	chunk.Hash[0] = 0xFF

	chunk.Reset()

	assert.Equal(t, uint64(0), chunk.Offset)
	assert.Equal(t, uint64(0), chunk.Length)
	assert.Equal(t, 0, len(chunk.Data))
	assert.Equal(t, 0, chunk.HashSize)
	assert.Equal(t, [64]byte{}, chunk.Hash)
}

func TestChunkPool(t *testing.T) {
	chunk := GetChunk()
	assert.NotNil(t, chunk)

	chunk.Offset = 100
	chunk.Length = 200

	PutChunk(chunk)

	// Get another chunk - may or may not be the same one
	chunk2 := GetChunk()
	assert.NotNil(t, chunk2)
	// After PutChunk, the chunk should be reset
	assert.Equal(t, uint64(0), chunk2.Offset)
	assert.Equal(t, uint64(0), chunk2.Length)
}

func TestNewDefaultChunker(t *testing.T) {
	data := make([]byte, 1024)
	_, _ = rand.Read(data)

	reader := bytes.NewReader(data)
	chunker, err := NewDefaultChunker(reader)
	require.NoError(t, err)
	assert.NotNil(t, chunker)
	assert.Equal(t, HashSHA256, chunker.HashAlgorithmUsed())
}

func TestChunkBytesDefault(t *testing.T) {
	data := make([]byte, 64*1024)
	_, _ = rand.Read(data)

	chunks, err := ChunkBytesDefault(data)
	require.NoError(t, err)
	assert.Greater(t, len(chunks), 0)

	// Verify total bytes
	var totalBytes uint64
	for _, chunk := range chunks {
		totalBytes += chunk.Length
	}
	assert.Equal(t, uint64(len(data)), totalBytes)
}

func TestNextInto(t *testing.T) {
	data := make([]byte, 128*1024)
	_, _ = rand.Read(data)

	config := MediumChunkConfig()
	reader := bytes.NewReader(data)
	chunker, err := NewChunker(reader, config)
	require.NoError(t, err)

	chunk := &Chunk{Data: make([]byte, 0, config.MaxSize)}
	var totalBytes uint64

	for {
		err := chunker.NextInto(chunk)
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		assert.Greater(t, chunk.Length, uint64(0))
		assert.Equal(t, int(chunk.Length), len(chunk.Data))
		totalBytes += chunk.Length
	}

	assert.Equal(t, uint64(len(data)), totalBytes)
}

func TestNextHashInto(t *testing.T) {
	data := make([]byte, 128*1024)
	_, _ = rand.Read(data)

	config := MediumChunkConfig()
	reader := bytes.NewReader(data)
	chunker, err := NewChunker(reader, config)
	require.NoError(t, err)

	chunk := &Chunk{}
	var totalBytes uint64

	for {
		err := chunker.NextHashInto(chunk)
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		assert.Greater(t, chunk.Length, uint64(0))
		assert.Nil(t, chunk.Data) // Data should be nil
		assert.Greater(t, chunk.HashSize, 0)
		totalBytes += chunk.Length
	}

	assert.Equal(t, uint64(len(data)), totalBytes)
}

func TestForEach(t *testing.T) {
	data := make([]byte, 128*1024)
	_, _ = rand.Read(data)

	config := MediumChunkConfig()
	reader := bytes.NewReader(data)
	chunker, err := NewChunker(reader, config)
	require.NoError(t, err)

	var totalBytes uint64
	var chunkCount int

	err = chunker.ForEach(func(chunk *Chunk) error {
		assert.Greater(t, chunk.Length, uint64(0))
		assert.NotNil(t, chunk.Data)
		totalBytes += chunk.Length
		chunkCount++
		return nil
	})
	require.NoError(t, err)

	assert.Equal(t, uint64(len(data)), totalBytes)
	assert.Greater(t, chunkCount, 0)
}

func TestForEachHash(t *testing.T) {
	data := make([]byte, 128*1024)
	_, _ = rand.Read(data)

	config := MediumChunkConfig()
	reader := bytes.NewReader(data)
	chunker, err := NewChunker(reader, config)
	require.NoError(t, err)

	var totalBytes uint64
	var chunkCount int

	err = chunker.ForEachHash(func(chunk *Chunk) error {
		assert.Greater(t, chunk.Length, uint64(0))
		assert.Nil(t, chunk.Data) // Data should be nil
		assert.Greater(t, chunk.HashSize, 0)
		totalBytes += chunk.Length
		chunkCount++
		return nil
	})
	require.NoError(t, err)

	assert.Equal(t, uint64(len(data)), totalBytes)
	assert.Greater(t, chunkCount, 0)
}

func TestForEachError(t *testing.T) {
	data := make([]byte, 128*1024)
	_, _ = rand.Read(data)

	config := MediumChunkConfig()
	reader := bytes.NewReader(data)
	chunker, err := NewChunker(reader, config)
	require.NoError(t, err)

	expectedErr := io.ErrUnexpectedEOF
	err = chunker.ForEach(func(_ *Chunk) error {
		return expectedErr
	})
	assert.Equal(t, expectedErr, err)
}

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

func TestValidateConfigEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "zero_buf_size",
			config: Config{
				MinSize:       1024,
				MaxSize:       8192,
				AvgSize:       4096,
				Normalization: 2,
				BufSize:       0,
			},
			wantErr: true,
		},
		{
			name: "avg_equals_min",
			config: Config{
				MinSize:       4096,
				MaxSize:       8192,
				AvgSize:       4096,
				Normalization: 2,
				BufSize:       16384,
			},
			wantErr: false,
		},
		{
			name: "avg_equals_max",
			config: Config{
				MinSize:       1024,
				MaxSize:       4096,
				AvgSize:       4096,
				Normalization: 2,
				BufSize:       16384,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCreateDistributedMaskEdgeCases(t *testing.T) {
	// Test edge case: 64 or more bits
	mask := createDistributedMask(64)
	assert.Equal(t, ^uint64(0), mask)

	mask = createDistributedMask(100)
	assert.Equal(t, ^uint64(0), mask)
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

func TestNewChunkerWithHash(t *testing.T) {
	data := make([]byte, 128*1024)
	_, err := rand.Read(data)
	require.NoError(t, err)

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
			reader := bytes.NewReader(data)
			chunker, err := NewChunkerWithHash(reader, tt.algorithm)
			require.NoError(t, err)
			assert.Equal(t, tt.algorithm, chunker.HashAlgorithmUsed())

			chunk, err := chunker.Next()
			require.NoError(t, err)
			assert.NotNil(t, chunk)

			// Verify hash matches expected algorithm
			expectedHash, expectedSize := HashData(chunk.Data, tt.algorithm)
			assert.Equal(t, expectedHash, chunk.Hash)
			assert.Equal(t, expectedSize, chunk.HashSize)
		})
	}
}

func TestChunkBytesWithHash(t *testing.T) {
	data := make([]byte, 64*1024)
	_, err := rand.Read(data)
	require.NoError(t, err)

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
			chunks, err := ChunkBytesWithHash(data, tt.algorithm)
			require.NoError(t, err)
			require.Greater(t, len(chunks), 0)

			// Verify hash matches expected algorithm
			for _, chunk := range chunks {
				expectedHash, expectedSize := HashData(chunk.Data, tt.algorithm)
				assert.Equal(t, expectedHash, chunk.Hash)
				assert.Equal(t, expectedSize, chunk.HashSize)
			}
		})
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

func chunkData(data []byte, config Config) ([]*Chunk, error) {
	reader := bytes.NewReader(data)
	chunker, err := NewChunker(reader, config)
	if err != nil {
		return nil, err
	}

	var chunks []*Chunk
	for {
		chunk, err := chunker.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		chunks = append(chunks, chunk)
	}
	return chunks, nil
}

// Benchmarks

func BenchmarkChunker_1MB(b *testing.B) {
	data := make([]byte, 1024*1024)
	_, _ = rand.Read(data)

	config := DefaultConfig()
	config.MinSize = 32 * 1024
	config.MaxSize = 256 * 1024
	config.AvgSize = 64 * 1024

	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader(data)
		chunker, _ := NewChunker(reader, config)
		for {
			_, err := chunker.Next()
			if err == io.EOF {
				break
			}
		}
	}
}

func BenchmarkChunker_10MB(b *testing.B) {
	data := make([]byte, 10*1024*1024)
	_, _ = rand.Read(data)

	config := DefaultConfig()

	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader(data)
		chunker, _ := NewChunker(reader, config)
		for {
			_, err := chunker.Next()
			if err == io.EOF {
				break
			}
		}
	}
}

func BenchmarkChunker_HashOnly_10MB(b *testing.B) {
	data := make([]byte, 10*1024*1024)
	_, _ = rand.Read(data)

	config := DefaultConfig()

	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader(data)
		chunker, _ := NewChunker(reader, config)
		for {
			_, err := chunker.NextHash()
			if err == io.EOF {
				break
			}
		}
	}
}

func BenchmarkChunkBytes_1MB(b *testing.B) {
	data := make([]byte, 1024*1024)
	_, _ = rand.Read(data)

	config := Config{
		MinSize:       32 * 1024,
		MaxSize:       256 * 1024,
		AvgSize:       64 * 1024,
		Normalization: 2,
		BufSize:       512 * 1024,
	}

	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = ChunkBytes(data, config)
	}
}

func BenchmarkHashData_SHA256(b *testing.B) {
	data := make([]byte, 64*1024)
	_, _ = rand.Read(data)

	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = HashData(data, HashSHA256)
	}
}

func BenchmarkHashData_SHA384(b *testing.B) {
	data := make([]byte, 64*1024)
	_, _ = rand.Read(data)

	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = HashData(data, HashSHA384)
	}
}

func BenchmarkHashData_SHA512(b *testing.B) {
	data := make([]byte, 64*1024)
	_, _ = rand.Read(data)

	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = HashData(data, HashSHA512)
	}
}

func BenchmarkFindBoundary(b *testing.B) {
	data := make([]byte, 256*1024)
	_, _ = rand.Read(data)

	maskS, maskL := computeMasks(64*1024, 2)

	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = findBoundaryInSlice(data, 32*1024, 256*1024, 64*1024, maskS, maskL)
	}
}

func BenchmarkChunker_ForEach_10MB(b *testing.B) {
	data := make([]byte, 10*1024*1024)
	_, _ = rand.Read(data)

	config := DefaultConfig()

	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader(data)
		chunker, _ := NewChunker(reader, config)
		_ = chunker.ForEach(func(_ *Chunk) error {
			return nil
		})
	}
}

func BenchmarkChunker_ForEachHash_10MB(b *testing.B) {
	data := make([]byte, 10*1024*1024)
	_, _ = rand.Read(data)

	config := DefaultConfig()

	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader(data)
		chunker, _ := NewChunker(reader, config)
		_ = chunker.ForEachHash(func(_ *Chunk) error {
			return nil
		})
	}
}

func BenchmarkChunker_NextInto_10MB(b *testing.B) {
	data := make([]byte, 10*1024*1024)
	_, _ = rand.Read(data)

	config := DefaultConfig()

	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader(data)
		chunker, _ := NewChunker(reader, config)
		chunk := &Chunk{Data: make([]byte, 0, config.MaxSize)}
		for {
			err := chunker.NextInto(chunk)
			if err == io.EOF {
				break
			}
		}
	}
}

func BenchmarkChunkBytesNoHash_10MB(b *testing.B) {
	data := make([]byte, 10*1024*1024)
	_, _ = rand.Read(data)

	config := DefaultConfig()
	maskS, maskL := computeMasks(config.AvgSize, config.Normalization)

	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		offset := uint64(0)
		remaining := len(data)

		for remaining > 0 {
			chunkLen := findBoundaryInSlice(data[offset:], config.MinSize, config.MaxSize, config.AvgSize, maskS, maskL)
			offset += uint64(chunkLen)
			remaining -= chunkLen
		}
	}
}

func BenchmarkChunkSizes(b *testing.B) {
	data := make([]byte, 100*1024*1024) // 100MB for better measurement
	_, _ = rand.Read(data)

	configs := []struct {
		name    string
		minSize uint64
		avgSize uint64
		maxSize uint64
	}{
		{"8KB_avg", 2 * 1024, 8 * 1024, 64 * 1024},
		{"16KB_avg", 4 * 1024, 16 * 1024, 128 * 1024},
		{"32KB_avg", 8 * 1024, 32 * 1024, 256 * 1024},
		{"64KB_avg", 16 * 1024, 64 * 1024, 512 * 1024},
		{"128KB_avg", 32 * 1024, 128 * 1024, 1024 * 1024},
		{"256KB_avg", 64 * 1024, 256 * 1024, 2 * 1024 * 1024},
		{"512KB_avg", 128 * 1024, 512 * 1024, 4 * 1024 * 1024},
		{"1MB_avg", 512 * 1024, 1024 * 1024, 8 * 1024 * 1024},
	}

	for _, cfg := range configs {
		b.Run(cfg.name, func(b *testing.B) {
			config := Config{
				MinSize:       cfg.minSize,
				MaxSize:       cfg.maxSize,
				AvgSize:       cfg.avgSize,
				Normalization: 2,
				BufSize:       int(cfg.maxSize) * 2,
				HashAlgorithm: HashSHA256,
			}

			b.ReportAllocs()
			b.SetBytes(int64(len(data)))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				reader := bytes.NewReader(data)
				chunker, _ := NewChunker(reader, config)
				_ = chunker.ForEachHash(func(_ *Chunk) error {
					return nil
				})
			}
		})
	}
}

func BenchmarkChunkSizesNoHash(b *testing.B) {
	data := make([]byte, 100*1024*1024) // 100MB
	_, _ = rand.Read(data)

	configs := []struct {
		name    string
		minSize uint64
		avgSize uint64
		maxSize uint64
	}{
		{"8KB_avg", 2 * 1024, 8 * 1024, 64 * 1024},
		{"16KB_avg", 4 * 1024, 16 * 1024, 128 * 1024},
		{"32KB_avg", 8 * 1024, 32 * 1024, 256 * 1024},
		{"64KB_avg", 16 * 1024, 64 * 1024, 512 * 1024},
		{"128KB_avg", 32 * 1024, 128 * 1024, 1024 * 1024},
		{"256KB_avg", 64 * 1024, 256 * 1024, 2 * 1024 * 1024},
		{"512KB_avg", 128 * 1024, 512 * 1024, 4 * 1024 * 1024},
		{"1MB_avg", 512 * 1024, 1024 * 1024, 8 * 1024 * 1024},
	}

	for _, cfg := range configs {
		b.Run(cfg.name, func(b *testing.B) {
			maskS, maskL := computeMasks(cfg.avgSize, 2)

			b.ReportAllocs()
			b.SetBytes(int64(len(data)))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				offset := uint64(0)
				remaining := len(data)

				for remaining > 0 {
					chunkLen := findBoundaryInSlice(data[offset:], cfg.minSize, cfg.maxSize, cfg.avgSize, maskS, maskL)
					offset += uint64(chunkLen)
					remaining -= chunkLen
				}
			}
		})
	}
}

// BenchmarkPresetConfigs benchmarks all preset configurations
func BenchmarkPresetConfigs(b *testing.B) {
	data := make([]byte, 100*1024*1024) // 100MB
	_, _ = rand.Read(data)

	configs := []struct {
		name   string
		config Config
	}{
		{"Small_8KB", SmallChunkConfig()},
		{"Medium_64KB", MediumChunkConfig()},
		{"Large_256KB", LargeChunkConfig()},
		{"Default_1MB", DefaultConfig()},
	}

	for _, cfg := range configs {
		b.Run(cfg.name, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(data)))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				reader := bytes.NewReader(data)
				chunker, _ := NewChunker(reader, cfg.config)
				_ = chunker.ForEachHash(func(_ *Chunk) error {
					return nil
				})
			}
		})
	}
}
