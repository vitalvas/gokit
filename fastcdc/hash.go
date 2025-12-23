package fastcdc

import (
	"crypto/sha256"
	"crypto/sha512"
	"hash"
)

// HashAlgorithm represents the available hash algorithms
type HashAlgorithm uint8

const (
	// HashSHA256 uses SHA-256 (default)
	HashSHA256 HashAlgorithm = iota

	// HashSHA384 uses SHA-384
	HashSHA384

	// HashSHA512 uses SHA-512
	HashSHA512
)

// String returns the string representation of the hash algorithm
func (h HashAlgorithm) String() string {
	switch h {
	case HashSHA256:
		return "sha256"
	case HashSHA384:
		return "sha384"
	case HashSHA512:
		return "sha512"
	default:
		return "unknown"
	}
}

// Size returns the hash output size in bytes
func (h HashAlgorithm) Size() int {
	switch h {
	case HashSHA256:
		return 32
	case HashSHA384:
		return 48
	case HashSHA512:
		return 64
	default:
		return 32
	}
}

// Hasher provides hash computation for chunk data
type Hasher struct {
	algorithm HashAlgorithm
	hash      hash.Hash
}

// NewHasher creates a new hasher with the specified algorithm
func NewHasher(algorithm HashAlgorithm) *Hasher {
	h := &Hasher{algorithm: algorithm}
	h.reset()
	return h
}

// reset initializes or resets the internal hash state
func (h *Hasher) reset() {
	switch h.algorithm {
	case HashSHA256:
		h.hash = sha256.New()
	case HashSHA384:
		h.hash = sha512.New384()
	case HashSHA512:
		h.hash = sha512.New()
	default:
		h.hash = sha256.New()
	}
}

// Sum computes the hash of the given data and returns it as a byte slice
func (h *Hasher) Sum(data []byte) []byte {
	h.hash.Reset()
	h.hash.Write(data)
	return h.hash.Sum(nil)
}

// Sum32 computes a 32-byte hash (truncates SHA-512 to 32 bytes)
func (h *Hasher) Sum32(data []byte) [32]byte {
	h.hash.Reset()
	h.hash.Write(data)
	sum := h.hash.Sum(nil)

	var result [32]byte
	copy(result[:], sum[:32])
	return result
}

// Sum64 computes a 64-byte hash (pads shorter hashes with zeros)
func (h *Hasher) Sum64(data []byte) [64]byte {
	h.hash.Reset()
	h.hash.Write(data)
	sum := h.hash.Sum(nil)

	var result [64]byte
	copy(result[:], sum)
	return result
}

// SumFull computes the hash and returns both the result and actual hash size
func (h *Hasher) SumFull(data []byte) ([64]byte, int) {
	h.hash.Reset()
	h.hash.Write(data)
	sum := h.hash.Sum(nil)

	var result [64]byte
	copy(result[:], sum)
	return result, len(sum)
}

// Algorithm returns the hash algorithm being used
func (h *Hasher) Algorithm() HashAlgorithm {
	return h.algorithm
}

// HashSize returns the native output size of the hash algorithm
func (h *Hasher) HashSize() int {
	return h.algorithm.Size()
}

// ComputeHash is a convenience function for one-shot hashing
func ComputeHash(data []byte, algorithm HashAlgorithm) []byte {
	switch algorithm {
	case HashSHA256:
		sum := sha256.Sum256(data)
		return sum[:]
	case HashSHA384:
		sum := sha512.Sum384(data)
		return sum[:]
	case HashSHA512:
		sum := sha512.Sum512(data)
		return sum[:]
	default:
		sum := sha256.Sum256(data)
		return sum[:]
	}
}

// ComputeHash32 computes a 32-byte hash using the specified algorithm
func ComputeHash32(data []byte, algorithm HashAlgorithm) [32]byte {
	switch algorithm {
	case HashSHA256:
		return sha256.Sum256(data)
	case HashSHA384:
		sum := sha512.Sum384(data)
		var result [32]byte
		copy(result[:], sum[:32])
		return result
	case HashSHA512:
		sum := sha512.Sum512(data)
		var result [32]byte
		copy(result[:], sum[:32])
		return result
	default:
		return sha256.Sum256(data)
	}
}

// ComputeHashFull computes a full hash and returns both the result and actual size
func ComputeHashFull(data []byte, algorithm HashAlgorithm) ([64]byte, int) {
	var result [64]byte
	switch algorithm {
	case HashSHA256:
		sum := sha256.Sum256(data)
		copy(result[:], sum[:])
		return result, 32
	case HashSHA384:
		sum := sha512.Sum384(data)
		copy(result[:], sum[:])
		return result, 48
	case HashSHA512:
		sum := sha512.Sum512(data)
		copy(result[:], sum[:])
		return result, 64
	default:
		sum := sha256.Sum256(data)
		copy(result[:], sum[:])
		return result, 32
	}
}
