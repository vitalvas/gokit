package fastcdc

import (
	"crypto/sha256"
	"crypto/sha512"
	"errors"
	"io"
	"sync"
)

const (
	// DefaultMinSize is the minimum chunk size (512 KiB)
	DefaultMinSize = 512 * 1024

	// DefaultMaxSize is the maximum chunk size (8 MiB)
	DefaultMaxSize = 8 * 1024 * 1024

	// DefaultAvgSize is the target average chunk size (1 MiB)
	DefaultAvgSize = 1 * 1024 * 1024

	// DefaultNormalization is the normalization level for chunk size distribution
	DefaultNormalization = 2
)

var (
	// ErrInvalidConfig indicates invalid chunker configuration
	ErrInvalidConfig = errors.New("fastcdc: invalid configuration")

	// chunkPool is a pool for reusing Chunk objects
	chunkPool = sync.Pool{
		New: func() any {
			return &Chunk{}
		},
	}
)

// Chunk represents a single content-defined chunk
type Chunk struct {
	Offset   uint64   // Byte offset in the original data stream
	Length   uint64   // Length of the chunk in bytes
	Data     []byte   // Chunk data (optional, may be nil if not requested)
	Hash     [64]byte // Hash of the chunk data (full size, up to 64 bytes)
	HashSize int      // Actual hash size in bytes (32 for SHA256, 48 for SHA384, 64 for SHA512)
}

// Reset clears the chunk for reuse
func (c *Chunk) Reset() {
	c.Offset = 0
	c.Length = 0
	c.Data = c.Data[:0]
	c.Hash = [64]byte{}
	c.HashSize = 0
}

// GetChunk returns a Chunk from the pool
func GetChunk() *Chunk {
	return chunkPool.Get().(*Chunk)
}

// PutChunk returns a Chunk to the pool
func PutChunk(c *Chunk) {
	c.Reset()
	chunkPool.Put(c)
}

// Config holds the configuration for the FastCDC chunker
type Config struct {
	MinSize       uint64        // Minimum chunk size in bytes
	MaxSize       uint64        // Maximum chunk size in bytes
	AvgSize       uint64        // Target average chunk size in bytes
	Normalization int           // Normalization level (0-3), affects chunk size distribution
	BufSize       int           // Internal buffer size for streaming
	HashAlgorithm HashAlgorithm // Hash algorithm to use (default: SHA256)
}

// DefaultConfig returns the default configuration (1MB average chunks)
func DefaultConfig() Config {
	return Config{
		MinSize:       DefaultMinSize,
		MaxSize:       DefaultMaxSize,
		AvgSize:       DefaultAvgSize,
		Normalization: DefaultNormalization,
		BufSize:       DefaultMaxSize * 2,
		HashAlgorithm: HashSHA256,
	}
}

// SmallChunkConfig returns configuration optimized for small files (8KB average)
func SmallChunkConfig() Config {
	return Config{
		MinSize:       2 * 1024,  // 2 KB
		MaxSize:       64 * 1024, // 64 KB
		AvgSize:       8 * 1024,  // 8 KB
		Normalization: 2,
		BufSize:       128 * 1024,
		HashAlgorithm: HashSHA256,
	}
}

// MediumChunkConfig returns configuration for general purpose use (64KB average)
func MediumChunkConfig() Config {
	return Config{
		MinSize:       16 * 1024,  // 16 KB
		MaxSize:       256 * 1024, // 256 KB
		AvgSize:       64 * 1024,  // 64 KB
		Normalization: 2,
		BufSize:       512 * 1024,
		HashAlgorithm: HashSHA256,
	}
}

// LargeChunkConfig returns configuration optimized for large files (256KB average)
func LargeChunkConfig() Config {
	return Config{
		MinSize:       64 * 1024,       // 64 KB
		MaxSize:       2 * 1024 * 1024, // 2 MB
		AvgSize:       256 * 1024,      // 256 KB
		Normalization: 2,
		BufSize:       4 * 1024 * 1024,
		HashAlgorithm: HashSHA256,
	}
}

// Chunker performs FastCDC content-defined chunking
type Chunker struct {
	config    Config
	maskS     uint64 // Mask for small chunks (before avg size)
	maskL     uint64 // Mask for large chunks (after avg size)
	buf       []byte
	bufStart  int    // Start cursor in buffer
	bufEnd    int    // End cursor in buffer
	offset    uint64 // Global offset in input stream
	reader    io.Reader
	readerEOF bool
	hasher    *Hasher
}

// NewChunker creates a new FastCDC chunker with the given configuration
func NewChunker(r io.Reader, config Config) (*Chunker, error) {
	if err := validateConfig(config); err != nil {
		return nil, err
	}

	maskS, maskL := computeMasks(config.AvgSize, config.Normalization)

	return &Chunker{
		config: config,
		maskS:  maskS,
		maskL:  maskL,
		buf:    make([]byte, config.BufSize),
		reader: r,
		hasher: NewHasher(config.HashAlgorithm),
	}, nil
}

// NewDefaultChunker creates a new FastCDC chunker with default configuration
func NewDefaultChunker(r io.Reader) (*Chunker, error) {
	return NewChunker(r, DefaultConfig())
}

// NewChunkerWithHash creates a new FastCDC chunker with specified hash algorithm
func NewChunkerWithHash(r io.Reader, algorithm HashAlgorithm) (*Chunker, error) {
	config := DefaultConfig()
	config.HashAlgorithm = algorithm
	return NewChunker(r, config)
}

// Next returns the next chunk from the input stream.
// Returns io.EOF when all data has been processed.
func (c *Chunker) Next() (*Chunk, error) {
	if err := c.fillBuffer(); err != nil && err != io.EOF {
		return nil, err
	}

	if c.bufferLen() == 0 {
		return nil, io.EOF
	}

	chunkLen := c.findBoundary()
	if chunkLen == 0 {
		return nil, io.EOF
	}

	chunk := &Chunk{
		Offset: c.offset,
		Length: uint64(chunkLen),
		Data:   make([]byte, chunkLen),
	}

	copy(chunk.Data, c.buf[c.bufStart:c.bufStart+chunkLen])
	chunk.Hash, chunk.HashSize = c.hasher.SumFull(chunk.Data)

	// Advance cursor (no copy needed)
	c.bufStart += chunkLen
	c.offset += uint64(chunkLen)

	return chunk, nil
}

// NextHash returns only the hash and metadata of the next chunk without copying data.
// This is useful when you only need chunk boundaries and hashes for deduplication.
func (c *Chunker) NextHash() (*Chunk, error) {
	if err := c.fillBuffer(); err != nil && err != io.EOF {
		return nil, err
	}

	if c.bufferLen() == 0 {
		return nil, io.EOF
	}

	chunkLen := c.findBoundary()
	if chunkLen == 0 {
		return nil, io.EOF
	}

	hash, hashSize := c.hasher.SumFull(c.buf[c.bufStart : c.bufStart+chunkLen])
	chunk := &Chunk{
		Offset:   c.offset,
		Length:   uint64(chunkLen),
		Hash:     hash,
		HashSize: hashSize,
	}

	// Advance cursor (no copy needed)
	c.bufStart += chunkLen
	c.offset += uint64(chunkLen)

	return chunk, nil
}

// NextInto fills the provided Chunk with the next chunk data.
// This is a zero-allocation method when the chunk's Data slice has sufficient capacity.
// Returns io.EOF when all data has been processed.
func (c *Chunker) NextInto(chunk *Chunk) error {
	if err := c.fillBuffer(); err != nil && err != io.EOF {
		return err
	}

	if c.bufferLen() == 0 {
		return io.EOF
	}

	chunkLen := c.findBoundary()
	if chunkLen == 0 {
		return io.EOF
	}

	chunk.Offset = c.offset
	chunk.Length = uint64(chunkLen)

	// Reuse existing Data slice if capacity is sufficient
	if cap(chunk.Data) >= chunkLen {
		chunk.Data = chunk.Data[:chunkLen]
	} else {
		chunk.Data = make([]byte, chunkLen)
	}

	copy(chunk.Data, c.buf[c.bufStart:c.bufStart+chunkLen])
	chunk.Hash, chunk.HashSize = c.hasher.SumFull(chunk.Data)

	// Advance cursor (no copy needed)
	c.bufStart += chunkLen
	c.offset += uint64(chunkLen)

	return nil
}

// NextHashInto fills the provided Chunk with metadata and hash only (no data copy).
// This is the most efficient zero-allocation method for deduplication workflows.
// Returns io.EOF when all data has been processed.
func (c *Chunker) NextHashInto(chunk *Chunk) error {
	if err := c.fillBuffer(); err != nil && err != io.EOF {
		return err
	}

	if c.bufferLen() == 0 {
		return io.EOF
	}

	chunkLen := c.findBoundary()
	if chunkLen == 0 {
		return io.EOF
	}

	chunk.Offset = c.offset
	chunk.Length = uint64(chunkLen)
	chunk.Data = nil
	chunk.Hash, chunk.HashSize = c.hasher.SumFull(c.buf[c.bufStart : c.bufStart+chunkLen])

	// Advance cursor (no copy needed)
	c.bufStart += chunkLen
	c.offset += uint64(chunkLen)

	return nil
}

// ChunkFunc is a callback function for ForEach iteration
type ChunkFunc func(chunk *Chunk) error

// ForEach iterates over all chunks calling the provided function.
// This is the most efficient way to process chunks as it reuses a single Chunk object.
// The chunk passed to fn is reused between calls - do not retain references to it.
func (c *Chunker) ForEach(fn ChunkFunc) error {
	chunk := GetChunk()
	defer PutChunk(chunk)

	// Pre-allocate data buffer to max size
	chunk.Data = make([]byte, 0, c.config.MaxSize)

	for {
		err := c.NextInto(chunk)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		if err := fn(chunk); err != nil {
			return err
		}
	}
}

// ForEachHash iterates over all chunks calling the provided function with hash only.
// This is the most efficient way to process chunk hashes for deduplication.
// The chunk passed to fn is reused between calls - do not retain references to it.
func (c *Chunker) ForEachHash(fn ChunkFunc) error {
	chunk := GetChunk()
	defer PutChunk(chunk)

	for {
		err := c.NextHashInto(chunk)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		if err := fn(chunk); err != nil {
			return err
		}
	}
}

// ChunkBytes chunks the provided byte slice and returns all chunks
func ChunkBytes(data []byte, config Config) ([]*Chunk, error) {
	if err := validateConfig(config); err != nil {
		return nil, err
	}

	maskS, maskL := computeMasks(config.AvgSize, config.Normalization)
	hasher := NewHasher(config.HashAlgorithm)

	// Pre-allocate estimated number of chunks
	estimatedChunks := len(data)/int(config.AvgSize) + 1
	chunks := make([]*Chunk, 0, estimatedChunks)

	offset := uint64(0)
	remaining := len(data)

	for remaining > 0 {
		chunkLen := findBoundaryInSlice(data[offset:], config.MinSize, config.MaxSize, config.AvgSize, maskS, maskL)

		chunk := &Chunk{
			Offset: offset,
			Length: uint64(chunkLen),
			Data:   data[offset : offset+uint64(chunkLen)], // Reference original data
		}
		chunk.Hash, chunk.HashSize = hasher.SumFull(chunk.Data)

		chunks = append(chunks, chunk)
		offset += uint64(chunkLen)
		remaining -= chunkLen
	}

	return chunks, nil
}

// ChunkBytesDefault chunks the provided byte slice using default configuration
func ChunkBytesDefault(data []byte) ([]*Chunk, error) {
	return ChunkBytes(data, DefaultConfig())
}

// ChunkBytesWithHash chunks the provided byte slice with specified hash algorithm
func ChunkBytesWithHash(data []byte, algorithm HashAlgorithm) ([]*Chunk, error) {
	config := DefaultConfig()
	config.HashAlgorithm = algorithm
	return ChunkBytes(data, config)
}

// fillBuffer reads more data from the reader into the buffer
func (c *Chunker) fillBuffer() error {
	if c.readerEOF {
		return io.EOF
	}

	// Calculate available data in buffer
	available := c.bufEnd - c.bufStart

	// Compact buffer if we need more space and have consumed data
	if c.bufStart > 0 && len(c.buf)-c.bufEnd < int(c.config.MaxSize) {
		copy(c.buf, c.buf[c.bufStart:c.bufEnd])
		c.bufStart = 0
		c.bufEnd = available
	}

	// Ensure we have at least MaxSize bytes if possible
	for available < int(c.config.MaxSize) {
		n, err := c.reader.Read(c.buf[c.bufEnd:])
		c.bufEnd += n
		available += n

		if err == io.EOF {
			c.readerEOF = true
			return io.EOF
		}

		if err != nil {
			return err
		}
	}

	return nil
}

// findBoundary finds the next chunk boundary using FastCDC algorithm
func (c *Chunker) findBoundary() int {
	return findBoundaryInSlice(c.buf[c.bufStart:c.bufEnd], c.config.MinSize, c.config.MaxSize, c.config.AvgSize, c.maskS, c.maskL)
}

// bufferLen returns the available data length in the buffer
func (c *Chunker) bufferLen() int {
	return c.bufEnd - c.bufStart
}

// findBoundaryInSlice implements the core FastCDC boundary detection
func findBoundaryInSlice(data []byte, minSize, maxSize, avgSize uint64, maskS, maskL uint64) int {
	dataLen := uint64(len(data))

	if dataLen <= minSize {
		return int(dataLen)
	}

	// Limit search to maxSize
	searchLen := min(dataLen, maxSize)

	var fingerprint uint64

	// Skip minSize bytes (gear hash doesn't affect boundary before minSize)
	i := minSize

	// Phase 1: Before average size, use stricter mask (maskS)
	// This makes it harder to find a boundary, resulting in larger chunks
	normalPoint := min(avgSize, searchLen)

	for ; i < normalPoint; i++ {
		fingerprint = (fingerprint << 1) + gearTable[data[i]]
		if (fingerprint & maskS) == 0 {
			return int(i + 1)
		}
	}

	// Phase 2: After average size, use looser mask (maskL)
	// This makes it easier to find a boundary
	for ; i < searchLen; i++ {
		fingerprint = (fingerprint << 1) + gearTable[data[i]]
		if (fingerprint & maskL) == 0 {
			return int(i + 1)
		}
	}

	// No boundary found, return maxSize or remaining data
	return int(searchLen)
}

// computeMasks calculates the masks for FastCDC based on average size and normalization
func computeMasks(avgSize uint64, normalization int) (maskS, maskL uint64) {
	bits := logarithm2(avgSize)

	// Normalization adjusts the masks to control chunk size distribution
	// Higher normalization = more uniform chunk sizes
	bitsS := bits + uint64(normalization)
	bitsL := bits - uint64(normalization)

	// Ensure bits don't go negative or too high
	bitsL = max(bitsL, 1)
	bitsS = min(bitsS, 63)

	// Use distributed masks for better deduplication
	maskS = createDistributedMask(bitsS)
	maskL = createDistributedMask(bitsL)

	return maskS, maskL
}

// createDistributedMask creates a mask with bits evenly distributed across 64 bits.
// This improves chunk boundary uniformity and deduplication ratio compared to
// compact masks where all bits are consecutive.
//
// For example, with 12 effective bits:
//   - Compact:     0x0FFF (bits 0-11 set)
//   - Distributed: bits spread across positions 0, 5, 10, 15, 20, 25, 30, 35, 40, 45, 50, 55
func createDistributedMask(effectiveBits uint64) uint64 {
	if effectiveBits == 0 {
		return 0
	}
	if effectiveBits >= 64 {
		return ^uint64(0)
	}

	// Spread effectiveBits across 64-bit range
	var mask uint64
	spacing := 64 / effectiveBits

	for i := uint64(0); i < effectiveBits; i++ {
		pos := i * spacing
		if pos < 64 {
			mask |= 1 << pos
		}
	}

	return mask
}

// logarithm2 returns floor(log2(n))
func logarithm2(n uint64) uint64 {
	var bits uint64
	for n > 1 {
		n >>= 1
		bits++
	}
	return bits
}

// validateConfig validates the chunker configuration
func validateConfig(config Config) error {
	if config.MinSize == 0 {
		return ErrInvalidConfig
	}

	if config.MaxSize == 0 {
		return ErrInvalidConfig
	}

	if config.AvgSize == 0 {
		return ErrInvalidConfig
	}

	if config.MinSize >= config.MaxSize {
		return ErrInvalidConfig
	}

	if config.AvgSize < config.MinSize || config.AvgSize > config.MaxSize {
		return ErrInvalidConfig
	}

	if config.Normalization < 0 || config.Normalization > 3 {
		return ErrInvalidConfig
	}

	if config.BufSize == 0 {
		return ErrInvalidConfig
	}

	return nil
}

// Reset resets the chunker state for reuse with a new reader
func (c *Chunker) Reset(r io.Reader) {
	c.reader = r
	c.bufStart = 0
	c.bufEnd = 0
	c.offset = 0
	c.readerEOF = false
}

// HashAlgorithmUsed returns the hash algorithm used by the chunker
func (c *Chunker) HashAlgorithmUsed() HashAlgorithm {
	return c.config.HashAlgorithm
}

// HashData computes a hash of the given data using the specified algorithm
func HashData(data []byte, algorithm HashAlgorithm) ([64]byte, int) {
	return ComputeHashFull(data, algorithm)
}

// HashDataSHA256 computes the SHA-256 hash of the given data (32 bytes)
func HashDataSHA256(data []byte) [32]byte {
	sum := sha256.Sum256(data)
	return sum
}

// HashDataSHA384 computes the SHA-384 hash of the given data (48 bytes)
func HashDataSHA384(data []byte) [48]byte {
	sum := sha512.Sum384(data)
	return sum
}

// HashDataSHA512 computes the SHA-512 hash of the given data (64 bytes)
func HashDataSHA512(data []byte) [64]byte {
	sum := sha512.Sum512(data)
	return sum
}
