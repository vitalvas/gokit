package bloomfilter

import (
	"bytes"
	"encoding/gob"
	"math"
	"unsafe"
)

// BloomFilter represents a high-performance Bloom filter using bit-level operations
type BloomFilter struct {
	bitset []uint64 // Use uint64 for better cache alignment and SIMD potential
	m      uint64   // Number of bits (must be divisible by 64)
	k      uint32   // Number of hash functions
	mask   uint64   // Bitmask for fast modulo operation (m-1, when m is power of 2)
}

// NewBloomFilter creates a new Bloom filter optimized for performance
func NewBloomFilter(n uint, p float64) *BloomFilter {
	optM := optimalM(n, p)
	k := optimalK(n, optM)

	// Round up to next power of 2 for faster modulo operations
	m := nextPowerOf2(optM)

	// Ensure m is divisible by 64 for efficient bit operations
	if m%64 != 0 {
		m = ((m / 64) + 1) * 64
	}

	return &BloomFilter{
		bitset: make([]uint64, m/64),
		m:      m,
		k:      k,
		mask:   m - 1, // For power-of-2 modulo optimization
	}
}

// Add element to the bloom filter using double hashing technique
func (bf *BloomFilter) Add(element string) {
	h1, h2 := bf.hash(element)

	for i := uint32(0); i < bf.k; i++ {
		// Double hashing: hash = h1 + i*h2
		hash := (h1 + uint64(i)*h2) & bf.mask

		// Calculate word and bit position
		wordIdx := hash >> 6 // Divide by 64
		bitIdx := hash & 63  // Modulo 64

		// Set bit using atomic OR operation
		bf.bitset[wordIdx] |= 1 << bitIdx
	}
}

// Contains checks if element might be in the set
func (bf *BloomFilter) Contains(element string) bool {
	h1, h2 := bf.hash(element)

	for i := uint32(0); i < bf.k; i++ {
		// Double hashing: hash = h1 + i*h2
		hash := (h1 + uint64(i)*h2) & bf.mask

		// Calculate word and bit position
		wordIdx := hash >> 6 // Divide by 64
		bitIdx := hash & 63  // Modulo 64

		// Check bit
		if (bf.bitset[wordIdx] & (1 << bitIdx)) == 0 {
			return false
		}
	}
	return true
}

// AddBytes adds raw bytes to the bloom filter (zero-allocation version)
func (bf *BloomFilter) AddBytes(data []byte) {
	h1, h2 := bf.hashBytes(data)

	for i := uint32(0); i < bf.k; i++ {
		hash := (h1 + uint64(i)*h2) & bf.mask
		wordIdx := hash >> 6
		bitIdx := hash & 63
		bf.bitset[wordIdx] |= 1 << bitIdx
	}
}

// ContainsBytes checks if raw bytes might be in the set (zero-allocation version)
func (bf *BloomFilter) ContainsBytes(data []byte) bool {
	h1, h2 := bf.hashBytes(data)

	for i := uint32(0); i < bf.k; i++ {
		hash := (h1 + uint64(i)*h2) & bf.mask
		wordIdx := hash >> 6
		bitIdx := hash & 63
		if (bf.bitset[wordIdx] & (1 << bitIdx)) == 0 {
			return false
		}
	}
	return true
}

// hash computes two independent hash values using optimized xxHash-style algorithm
func (bf *BloomFilter) hash(s string) (uint64, uint64) {
	// Convert string to []byte without allocation using unsafe
	data := unsafe.Slice(unsafe.StringData(s), len(s))
	return bf.hashBytes(data)
}

// hashBytes computes two independent hash values for raw bytes
func (bf *BloomFilter) hashBytes(data []byte) (uint64, uint64) {
	const (
		prime1 = 0x9E3779B185EBCA87
		prime2 = 0xC2B2AE3D27D4EB4F
		prime3 = 0x165667B19E3779F9
		prime4 = 0x85EBCA77C2B2AE63
	)

	h1 := uint64(len(data)) * prime1
	h2 := uint64(len(data)) * prime2

	// Process 8-byte chunks
	for len(data) >= 8 {
		k := *(*uint64)(unsafe.Pointer(&data[0]))
		h1 ^= (k * prime3)
		h1 = rotateLeft64(h1, 27) * prime1

		h2 ^= (k * prime4)
		h2 = rotateLeft64(h2, 31) * prime2

		data = data[8:]
	}

	// Process remaining bytes
	for len(data) > 0 {
		h1 ^= uint64(data[0]) * prime1
		h1 = rotateLeft64(h1, 11) * prime2

		h2 ^= uint64(data[0]) * prime2
		h2 = rotateLeft64(h2, 13) * prime1

		data = data[1:]
	}

	// Final mixing
	h1 ^= h1 >> 33
	h1 *= prime3
	h1 ^= h1 >> 33

	h2 ^= h2 >> 33
	h2 *= prime4
	h2 ^= h2 >> 33

	return h1, h2
}

// rotateLeft64 rotates a 64-bit integer left by n bits
func rotateLeft64(x uint64, n int) uint64 {
	return (x << n) | (x >> (64 - n))
}

// nextPowerOf2 returns the next power of 2 greater than or equal to n
func nextPowerOf2(n uint) uint64 {
	if n == 0 {
		return 1
	}
	n--
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	n |= n >> 32
	return uint64(n + 1)
}

// Export serializes the bloom filter
func (bf *BloomFilter) Export() ([]byte, error) {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)

	exportData := struct {
		Bitset []uint64
		M      uint64
		K      uint32
	}{
		Bitset: bf.bitset,
		M:      bf.m,
		K:      bf.k,
	}

	if err := encoder.Encode(exportData); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ImportBloomFilter deserializes a bloom filter
func ImportBloomFilter(data []byte) (*BloomFilter, error) {
	var exportData struct {
		Bitset []uint64
		M      uint64
		K      uint32
	}

	buf := bytes.NewBuffer(data)
	decoder := gob.NewDecoder(buf)
	if err := decoder.Decode(&exportData); err != nil {
		return nil, err
	}

	return &BloomFilter{
		bitset: exportData.Bitset,
		m:      exportData.M,
		k:      exportData.K,
		mask:   exportData.M - 1,
	}, nil
}

// Size returns the number of bits in the filter
func (bf *BloomFilter) Size() uint64 {
	return bf.m
}

// K returns the number of hash functions
func (bf *BloomFilter) K() uint32 {
	return bf.k
}

// EstimatedCount returns the estimated number of elements added to the filter
func (bf *BloomFilter) EstimatedCount() uint64 {
	setBits := uint64(0)
	for _, word := range bf.bitset {
		setBits += uint64(popcount(word))
	}

	if setBits == 0 {
		return 0
	}

	// Using the formula: n ≈ -(m/k) * ln(1 - X/m) where X is number of set bits
	ratio := float64(setBits) / float64(bf.m)
	if ratio >= 1.0 {
		return math.MaxUint64 // Filter is full
	}

	estimated := -float64(bf.m) / float64(bf.k) * math.Log(1.0-ratio)
	if estimated < 0 {
		return 0
	}

	return uint64(estimated)
}

// popcount counts the number of set bits in a uint64
func popcount(x uint64) int {
	// Use compiler intrinsic for bit counting
	count := 0
	for x != 0 {
		count++
		x &= x - 1 // Clear the lowest set bit
	}
	return count
}

// Clear resets all bits in the filter
func (bf *BloomFilter) Clear() {
	for i := range bf.bitset {
		bf.bitset[i] = 0
	}
}

// optimalM calculates the optimal number of bits
func optimalM(n uint, p float64) uint {
	return uint(math.Ceil(float64(n) * math.Abs(math.Log(p)) / (math.Log(2) * math.Log(2))))
}

// optimalK calculates the optimal number of hash functions
func optimalK(n, m uint) uint32 {
	k := float64(m) / float64(n) * math.Log(2)
	return uint32(math.Round(k))
}
