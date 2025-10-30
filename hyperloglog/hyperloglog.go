package hyperloglog

import (
	"encoding/gob"
	"math"
	"math/bits"
	"unsafe"
)

// HyperLogLog is a probabilistic data structure for cardinality estimation.
// It can estimate the number of distinct elements in a dataset with minimal memory usage.
//
// Typical error rate: ~1.04 / sqrt(m) where m = 2^precision
// Example: precision=14 gives ~0.81% standard error with 16KB memory
type HyperLogLog struct {
	registers []uint8
	precision uint8
	alpha     float64
	m         uint32
}

// New creates a new HyperLogLog with the specified precision.
// Precision must be between 4 and 18 (inclusive).
//
// Memory usage: 2^precision bytes
// Standard error: ~1.04 / sqrt(2^precision)
//
// Common precision values:
//   - 10: 1KB memory, ~3.2% error
//   - 12: 4KB memory, ~1.6% error
//   - 14: 16KB memory, ~0.81% error (recommended)
//   - 16: 64KB memory, ~0.40% error
func New(precision uint8) *HyperLogLog {
	if precision < 4 || precision > 18 {
		precision = 14 // Default to recommended precision
	}

	m := uint32(1) << precision

	// Calculate alpha constant for bias correction
	var alpha float64
	switch m {
	case 16:
		alpha = 0.673
	case 32:
		alpha = 0.697
	case 64:
		alpha = 0.709
	default:
		alpha = 0.7213 / (1 + 1.079/float64(m))
	}

	return &HyperLogLog{
		registers: make([]uint8, m),
		precision: precision,
		alpha:     alpha,
		m:         m,
	}
}

// Add adds raw bytes to the HyperLogLog.
func (h *HyperLogLog) Add(data []byte) {
	hash := hash64(data)

	// Use first 'precision' bits for register index
	index := hash >> (64 - h.precision)

	// Use remaining bits to count leading zeros
	w := hash<<h.precision | (1 << (h.precision - 1))
	leadingZeros := uint8(bits.LeadingZeros64(w)) + 1

	// Update register with maximum value
	if leadingZeros > h.registers[index] {
		h.registers[index] = leadingZeros
	}
}

// AddString adds a string element to the HyperLogLog.
func (h *HyperLogLog) AddString(s string) {
	h.Add(stringToBytes(s))
}

// Count returns the estimated cardinality (number of distinct elements).
func (h *HyperLogLog) Count() uint64 {
	// Calculate raw estimate using harmonic mean
	var sum float64
	zeros := 0

	for _, val := range h.registers {
		sum += 1.0 / math.Pow(2.0, float64(val))
		if val == 0 {
			zeros++
		}
	}

	estimate := h.alpha * float64(h.m) * float64(h.m) / sum

	// Apply bias correction for different ranges
	switch {
	case estimate <= 2.5*float64(h.m):
		// Small range correction
		if zeros > 0 {
			estimate = float64(h.m) * math.Log(float64(h.m)/float64(zeros))
		}
	case estimate > (1.0/30.0)*math.Pow(2.0, 32):
		// Large range correction
		estimate = -math.Pow(2.0, 32) * math.Log(1.0-estimate/math.Pow(2.0, 32))
	}
	// No correction for medium range

	return uint64(estimate)
}

// Merge combines another HyperLogLog into this one.
// Both HyperLogLogs must have the same precision.
// After merging, this HyperLogLog will estimate the union of both sets.
func (h *HyperLogLog) Merge(other *HyperLogLog) error {
	if h.precision != other.precision {
		return &PrecisionMismatchError{h.precision, other.precision}
	}

	for i := range h.registers {
		if other.registers[i] > h.registers[i] {
			h.registers[i] = other.registers[i]
		}
	}

	return nil
}

// Clone creates a deep copy of the HyperLogLog.
func (h *HyperLogLog) Clone() *HyperLogLog {
	clone := &HyperLogLog{
		registers: make([]uint8, len(h.registers)),
		precision: h.precision,
		alpha:     h.alpha,
		m:         h.m,
	}
	copy(clone.registers, h.registers)
	return clone
}

// Clear resets all registers to zero.
func (h *HyperLogLog) Clear() {
	for i := range h.registers {
		h.registers[i] = 0
	}
}

// Precision returns the precision parameter of this HyperLogLog.
func (h *HyperLogLog) Precision() uint8 {
	return h.precision
}

// Size returns the number of registers (2^precision).
func (h *HyperLogLog) Size() uint32 {
	return h.m
}

// Export serializes the HyperLogLog for storage or transmission.
func (h *HyperLogLog) Export() ([]byte, error) {
	var buf []byte
	enc := gob.NewEncoder(&gobWriter{buf: &buf})

	data := &hllData{
		Precision: h.precision,
		Registers: h.registers,
	}

	if err := enc.Encode(data); err != nil {
		return nil, err
	}

	return buf, nil
}

// Import deserializes a HyperLogLog from exported data.
func Import(data []byte) (*HyperLogLog, error) {
	var hllData hllData
	dec := gob.NewDecoder(&gobReader{buf: data})

	if err := dec.Decode(&hllData); err != nil {
		return nil, err
	}

	hll := New(hllData.Precision)
	hll.registers = hllData.Registers

	return hll, nil
}

// MergeAll creates a new HyperLogLog by merging multiple HyperLogLogs.
// All HyperLogLogs must have the same precision.
// Returns a new HyperLogLog without modifying the input HyperLogLogs.
func MergeAll(hlls ...*HyperLogLog) (*HyperLogLog, error) {
	if len(hlls) == 0 {
		return nil, &MergeError{"no HyperLogLogs provided"}
	}

	// Check all have same precision
	precision := hlls[0].precision
	for i := 1; i < len(hlls); i++ {
		if hlls[i].precision != precision {
			return nil, &PrecisionMismatchError{precision, hlls[i].precision}
		}
	}

	// Create new HLL with same precision
	result := New(precision)

	// Merge all registers
	for _, hll := range hlls {
		for i := range result.registers {
			if hll.registers[i] > result.registers[i] {
				result.registers[i] = hll.registers[i]
			}
		}
	}

	return result, nil
}

// hllData is used for gob encoding/decoding
type hllData struct {
	Precision uint8
	Registers []uint8
}

// PrecisionMismatchError is returned when trying to merge HyperLogLogs with different precisions.
type PrecisionMismatchError struct {
	Precision1 uint8
	Precision2 uint8
}

func (e *PrecisionMismatchError) Error() string {
	return "precision mismatch: cannot merge HyperLogLog with different precisions"
}

// MergeError is returned when a merge operation fails.
type MergeError struct {
	Message string
}

func (e *MergeError) Error() string {
	return e.Message
}

// gobWriter implements io.Writer for gob encoding
type gobWriter struct {
	buf *[]byte
}

func (w *gobWriter) Write(p []byte) (int, error) {
	*w.buf = append(*w.buf, p...)
	return len(p), nil
}

// gobReader implements io.Reader for gob decoding
type gobReader struct {
	buf []byte
	pos int
}

func (r *gobReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.buf) {
		return 0, nil
	}
	n := copy(p, r.buf[r.pos:])
	r.pos += n
	return n, nil
}

// hash64 computes a 64-bit hash using xxHash-inspired algorithm
func hash64(data []byte) uint64 {
	const (
		prime1 = 11400714785074694791
		prime2 = 14029467366897019727
		prime3 = 1609587929392839161
		prime4 = 9650029242287828579
		prime5 = 2870177450012600261
	)

	var h uint64 = prime5

	for len(data) >= 8 {
		k := uint64(data[0]) | uint64(data[1])<<8 | uint64(data[2])<<16 | uint64(data[3])<<24 |
			uint64(data[4])<<32 | uint64(data[5])<<40 | uint64(data[6])<<48 | uint64(data[7])<<56
		k *= prime2
		k = bits.RotateLeft64(k, 31)
		k *= prime1
		h ^= k
		h = bits.RotateLeft64(h, 27)*prime1 + prime4
		data = data[8:]
	}

	if len(data) >= 4 {
		k := uint64(data[0]) | uint64(data[1])<<8 | uint64(data[2])<<16 | uint64(data[3])<<24
		k *= prime1
		h ^= k
		h = bits.RotateLeft64(h, 23)*prime2 + prime3
		data = data[4:]
	}

	for len(data) > 0 {
		k := uint64(data[0])
		k *= prime5
		h ^= k
		h = bits.RotateLeft64(h, 11) * prime1
		data = data[1:]
	}

	h ^= h >> 33
	h *= prime2
	h ^= h >> 29
	h *= prime3
	h ^= h >> 32

	return h
}

// stringToBytes converts string to []byte without allocation
func stringToBytes(s string) []byte {
	return unsafe.Slice(unsafe.StringData(s), len(s))
}
