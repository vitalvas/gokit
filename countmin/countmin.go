package countmin

import (
	"encoding/gob"
	"io"
	"math"
	"math/bits"
	"sync"
	"unsafe"
)

// Sketch is a Count-min sketch for frequency estimation.
// It provides approximate frequency counts with bounded error guarantees.
//
// Error bounds:
//   - Overestimation: count(x) ≤ true_count(x) + ε * N (with probability 1-δ)
//   - where ε = e/width, δ = (1/e)^depth, N = total count
//
// Memory usage: depth × width × 8 bytes (uint64)
type Sketch struct {
	mu      sync.RWMutex
	matrix  [][]uint64
	width   uint32
	depth   uint32
	epsilon float64
	delta   float64
	total   uint64
}

// New creates a new Count-min sketch with specified error bounds.
//
// Parameters:
//   - epsilon: error factor (typical: 0.001 to 0.01)
//   - delta: probability of exceeding error bound (typical: 0.01 to 0.1)
//
// Memory usage: O(e/ε * ln(1/δ)) where e ≈ 2.718
//
// Examples:
//   - New(0.001, 0.01): width=2719, depth=5, ~106KB memory
//   - New(0.01, 0.01):  width=272, depth=5, ~11KB memory
//   - New(0.001, 0.1):  width=2719, depth=3, ~64KB memory
func New(epsilon, delta float64) *Sketch {
	if epsilon <= 0 {
		epsilon = 0.001 // Default 0.1% error
	}
	if delta <= 0 || delta >= 1 {
		delta = 0.01 // Default 1% failure probability
	}

	// Calculate dimensions
	width := uint32(math.Ceil(math.E / epsilon))
	depth := uint32(math.Ceil(math.Log(1 / delta)))

	// Allocate matrix
	matrix := make([][]uint64, depth)
	for i := range matrix {
		matrix[i] = make([]uint64, width)
	}

	return &Sketch{
		matrix:  matrix,
		width:   width,
		depth:   depth,
		epsilon: epsilon,
		delta:   delta,
	}
}

// NewWithSize creates a Count-min sketch with explicit dimensions.
// Use this when you know the exact width and depth you need.
func NewWithSize(width, depth uint32) *Sketch {
	if width < 1 {
		width = 272
	}
	if depth < 1 {
		depth = 5
	}

	matrix := make([][]uint64, depth)
	for i := range matrix {
		matrix[i] = make([]uint64, width)
	}

	epsilon := math.E / float64(width)
	delta := math.Exp(-float64(depth))

	return &Sketch{
		matrix:  matrix,
		width:   width,
		depth:   depth,
		epsilon: epsilon,
		delta:   delta,
	}
}

// Add adds an item to the sketch with count n.
func (s *Sketch) Add(data []byte, n uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.total += n

	// Update all rows
	for i := uint32(0); i < s.depth; i++ {
		hash := s.hash(data, i)
		index := hash % uint64(s.width)
		s.matrix[i][index] += n
	}
}

// AddString adds a string item to the sketch with count n.
func (s *Sketch) AddString(str string, n uint64) {
	s.Add(stringToBytes(str), n)
}

// Update adds a single occurrence of an item.
func (s *Sketch) Update(data []byte) {
	s.Add(data, 1)
}

// UpdateString adds a single occurrence of a string item.
func (s *Sketch) UpdateString(str string) {
	s.AddString(str, 1)
}

// Count returns the estimated frequency of an item.
// The estimate is guaranteed to be ≥ true count.
func (s *Sketch) Count(data []byte) uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Query all rows and return minimum
	minCount := uint64(math.MaxUint64)

	for i := uint32(0); i < s.depth; i++ {
		hash := s.hash(data, i)
		index := hash % uint64(s.width)
		count := s.matrix[i][index]

		if count < minCount {
			minCount = count
		}
	}

	return minCount
}

// CountString returns the estimated frequency of a string item.
func (s *Sketch) CountString(str string) uint64 {
	return s.Count(stringToBytes(str))
}

// Total returns the total count of all items added.
func (s *Sketch) Total() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.total
}

// Clear resets all counters to zero.
func (s *Sketch) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.matrix {
		for j := range s.matrix[i] {
			s.matrix[i][j] = 0
		}
	}
	s.total = 0
}

// Merge combines another sketch into this one.
// Both sketches must have the same dimensions.
func (s *Sketch) Merge(other *Sketch) error {
	if s.width != other.width || s.depth != other.depth {
		return &DimensionMismatchError{s.width, s.depth, other.width, other.depth}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	other.mu.RLock()
	defer other.mu.RUnlock()

	for i := range s.matrix {
		for j := range s.matrix[i] {
			s.matrix[i][j] += other.matrix[i][j]
		}
	}
	s.total += other.total

	return nil
}

// Clone creates a deep copy of the sketch.
func (s *Sketch) Clone() *Sketch {
	s.mu.RLock()
	defer s.mu.RUnlock()

	matrix := make([][]uint64, s.depth)
	for i := range s.matrix {
		matrix[i] = make([]uint64, s.width)
		copy(matrix[i], s.matrix[i])
	}

	return &Sketch{
		matrix:  matrix,
		width:   s.width,
		depth:   s.depth,
		epsilon: s.epsilon,
		delta:   s.delta,
		total:   s.total,
	}
}

// Width returns the width of the sketch.
func (s *Sketch) Width() uint32 {
	return s.width
}

// Depth returns the depth of the sketch.
func (s *Sketch) Depth() uint32 {
	return s.depth
}

// Epsilon returns the error factor.
func (s *Sketch) Epsilon() float64 {
	return s.epsilon
}

// Delta returns the failure probability.
func (s *Sketch) Delta() float64 {
	return s.delta
}

// EstimatedError returns the estimated error bound for current load.
// Error ≈ ε × N, where N is total count.
func (s *Sketch) EstimatedError() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return uint64(s.epsilon * float64(s.total))
}

// Export serializes the sketch for storage or transmission.
func (s *Sketch) Export() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var buf []byte
	enc := gob.NewEncoder(&gobWriter{buf: &buf})

	data := &sketchData{
		Matrix:  s.matrix,
		Width:   s.width,
		Depth:   s.depth,
		Epsilon: s.epsilon,
		Delta:   s.delta,
		Total:   s.total,
	}

	if err := enc.Encode(data); err != nil {
		return nil, err
	}

	return buf, nil
}

// Import deserializes a sketch from exported data.
func Import(data []byte) (*Sketch, error) {
	var sketchData sketchData
	dec := gob.NewDecoder(&gobReader{buf: data})

	if err := dec.Decode(&sketchData); err != nil {
		return nil, err
	}

	return &Sketch{
		matrix:  sketchData.Matrix,
		width:   sketchData.Width,
		depth:   sketchData.Depth,
		epsilon: sketchData.Epsilon,
		delta:   sketchData.Delta,
		total:   sketchData.Total,
	}, nil
}

// hash computes hash value for data with seed based on row.
func (s *Sketch) hash(data []byte, row uint32) uint64 {
	// Use different hash seed per row
	seed := uint64(row)*0x9e3779b97f4a7c15 + 0x517cc1b727220a95
	return hash64(data, seed)
}

// hash64 computes a 64-bit hash using xxHash-inspired algorithm with seed.
func hash64(data []byte, seed uint64) uint64 {
	const (
		prime1 = 11400714785074694791
		prime2 = 14029467366897019727
		prime3 = 1609587929392839161
		prime4 = 9650029242287828579
		prime5 = 2870177450012600261
	)

	h := seed + prime5

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

// stringToBytes converts string to []byte without allocation.
func stringToBytes(s string) []byte {
	return unsafe.Slice(unsafe.StringData(s), len(s))
}

// sketchData is used for gob encoding/decoding.
type sketchData struct {
	Matrix  [][]uint64
	Width   uint32
	Depth   uint32
	Epsilon float64
	Delta   float64
	Total   uint64
}

// DimensionMismatchError is returned when trying to merge sketches with different dimensions.
type DimensionMismatchError struct {
	Width1 uint32
	Depth1 uint32
	Width2 uint32
	Depth2 uint32
}

func (e *DimensionMismatchError) Error() string {
	return "dimension mismatch: cannot merge Count-min sketches with different dimensions"
}

// gobWriter implements io.Writer for gob encoding.
type gobWriter struct {
	buf *[]byte
}

func (w *gobWriter) Write(p []byte) (int, error) {
	*w.buf = append(*w.buf, p...)
	return len(p), nil
}

// gobReader implements io.Reader for gob decoding.
type gobReader struct {
	buf []byte
	pos int
}

func (r *gobReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.buf) {
		return 0, io.EOF
	}
	n := copy(p, r.buf[r.pos:])
	r.pos += n
	return n, nil
}
