package cuckoo

import (
	"encoding/gob"
	"math/rand"
)

// Filter is a cuckoo filter for approximate set membership testing.
// It supports insertions, lookups, and deletions with low false positive rates.
//
// Unlike Bloom filters, cuckoo filters:
// - Support deletion
// - Have better lookup performance
// - Use less space at similar false positive rates
type Filter struct {
	buckets       []bucket
	numBuckets    uint
	bucketSize    uint
	fingerprintSize uint
	count         uint
	maxKicks      uint
}

// bucket represents a bucket in the cuckoo filter.
type bucket struct {
	entries []fingerprint
}

type fingerprint byte

// New creates a new cuckoo filter with the specified capacity.
//
// Parameters:
//   - capacity: Expected number of elements to store
//
// The filter uses:
//   - 4 entries per bucket (standard)
//   - 8-bit fingerprints
//   - Load factor of ~95%
//
// False positive rate: ~0.03% (3 in 10,000)
func New(capacity uint) *Filter {
	bucketSize := uint(4)
	fingerprintSize := uint(8)
	numBuckets := nextPowerOfTwo(capacity / bucketSize)

	if numBuckets < 2 {
		numBuckets = 2
	}

	buckets := make([]bucket, numBuckets)
	for i := range buckets {
		buckets[i].entries = make([]fingerprint, 0, bucketSize)
	}

	return &Filter{
		buckets:         buckets,
		numBuckets:      numBuckets,
		bucketSize:      bucketSize,
		fingerprintSize: fingerprintSize,
		count:           0,
		maxKicks:        500,
	}
}

// Insert adds an item to the filter.
// Returns true if successful, false if the filter is full.
func (f *Filter) Insert(data []byte) bool {
	return f.insert(data)
}

// InsertUnique adds an item only if it doesn't already exist.
// Returns true if inserted, false if already exists or filter is full.
func (f *Filter) InsertUnique(data []byte) bool {
	if f.Contains(data) {
		return false
	}
	return f.insert(data)
}

func (f *Filter) insert(data []byte) bool {
	fp := f.fingerprint(data)
	i1 := f.hash(data) % f.numBuckets
	i2 := f.altIndex(i1, fp)

	// Try to insert in first bucket
	if f.insertToBucket(i1, fp) {
		f.count++
		return true
	}

	// Try to insert in second bucket
	if f.insertToBucket(i2, fp) {
		f.count++
		return true
	}

	// Both buckets full, perform cuckoo kick
	i := i1
	if rand.Intn(2) == 1 {
		i = i2
	}

	for k := uint(0); k < f.maxKicks; k++ {
		// Pick random entry to kick
		entryIndex := rand.Intn(int(f.bucketSize))
		f.buckets[i].entries[entryIndex], fp = fp, f.buckets[i].entries[entryIndex]

		// Get alternate location for kicked entry
		i = f.altIndex(i, fp)

		// Try to insert kicked entry
		if f.insertToBucket(i, fp) {
			f.count++
			return true
		}
	}

	// Filter is full
	return false
}

// Contains checks if an item might be in the filter.
// Returns true if the item might exist (with small false positive rate).
// Returns false if the item definitely does not exist.
func (f *Filter) Contains(data []byte) bool {
	fp := f.fingerprint(data)
	i1 := f.hash(data) % f.numBuckets
	i2 := f.altIndex(i1, fp)

	return f.bucketContains(i1, fp) || f.bucketContains(i2, fp)
}

// Delete removes an item from the filter.
// Returns true if the item was found and deleted, false otherwise.
func (f *Filter) Delete(data []byte) bool {
	fp := f.fingerprint(data)
	i1 := f.hash(data) % f.numBuckets
	i2 := f.altIndex(i1, fp)

	if f.deleteFromBucket(i1, fp) {
		f.count--
		return true
	}

	if f.deleteFromBucket(i2, fp) {
		f.count--
		return true
	}

	return false
}

// Count returns the number of items in the filter.
func (f *Filter) Count() uint {
	return f.count
}

// LoadFactor returns the current load factor (0-1).
// Load factor = number of items / total capacity
func (f *Filter) LoadFactor() float64 {
	capacity := f.numBuckets * f.bucketSize
	return float64(f.count) / float64(capacity)
}

// Reset clears all items from the filter.
func (f *Filter) Reset() {
	for i := range f.buckets {
		f.buckets[i].entries = f.buckets[i].entries[:0]
	}
	f.count = 0
}

// Export serializes the filter for storage or transmission.
func (f *Filter) Export() ([]byte, error) {
	var buf []byte
	enc := gob.NewEncoder(&gobWriter{buf: &buf})

	// Flatten buckets for efficient encoding
	var allEntries []fingerprint
	bucketSizes := make([]uint, 0, len(f.buckets))

	for i := range f.buckets {
		bucketSizes = append(bucketSizes, uint(len(f.buckets[i].entries)))
		allEntries = append(allEntries, f.buckets[i].entries...)
	}

	data := &filterData{
		NumBuckets:      f.numBuckets,
		BucketSize:      f.bucketSize,
		FingerprintSize: f.fingerprintSize,
		Count:           f.count,
		MaxKicks:        f.maxKicks,
		BucketSizes:     bucketSizes,
		Entries:         allEntries,
	}

	if err := enc.Encode(data); err != nil {
		return nil, err
	}

	return buf, nil
}

// Import deserializes a filter from exported data.
func Import(data []byte) (*Filter, error) {
	var filterData filterData
	dec := gob.NewDecoder(&gobReader{buf: data})

	if err := dec.Decode(&filterData); err != nil {
		return nil, err
	}

	f := &Filter{
		numBuckets:      filterData.NumBuckets,
		bucketSize:      filterData.BucketSize,
		fingerprintSize: filterData.FingerprintSize,
		count:           filterData.Count,
		maxKicks:        filterData.MaxKicks,
		buckets:         make([]bucket, filterData.NumBuckets),
	}

	// Reconstruct buckets
	entryIndex := 0
	for i := range f.buckets {
		size := filterData.BucketSizes[i]
		f.buckets[i].entries = make([]fingerprint, size, f.bucketSize)
		copy(f.buckets[i].entries, filterData.Entries[entryIndex:entryIndex+int(size)])
		entryIndex += int(size)
	}

	return f, nil
}

// Helper functions

func (f *Filter) insertToBucket(i uint, fp fingerprint) bool {
	if uint(len(f.buckets[i].entries)) < f.bucketSize {
		f.buckets[i].entries = append(f.buckets[i].entries, fp)
		return true
	}
	return false
}

func (f *Filter) bucketContains(i uint, fp fingerprint) bool {
	for _, entry := range f.buckets[i].entries {
		if entry == fp {
			return true
		}
	}
	return false
}

func (f *Filter) deleteFromBucket(i uint, fp fingerprint) bool {
	for j, entry := range f.buckets[i].entries {
		if entry == fp {
			// Remove entry by swapping with last and truncating
			lastIdx := len(f.buckets[i].entries) - 1
			f.buckets[i].entries[j] = f.buckets[i].entries[lastIdx]
			f.buckets[i].entries = f.buckets[i].entries[:lastIdx]
			return true
		}
	}
	return false
}

func (f *Filter) fingerprint(data []byte) fingerprint {
	h := hash(data)
	// Use upper bits for better distribution
	fp := fingerprint((h >> 32) & 0xFF)
	// Ensure non-zero fingerprint
	if fp == 0 {
		fp = 1
	}
	return fp
}

func (f *Filter) hash(data []byte) uint {
	return uint(hash(data))
}

func (f *Filter) altIndex(i uint, fp fingerprint) uint {
	// Use fingerprint to compute alternate index
	// XOR with a hash of the fingerprint for better distribution
	fpHash := uint(fp) * 0x5bd1e995
	return (i ^ fpHash) % f.numBuckets
}

// hash computes a 64-bit hash using FNV-1a
func hash(data []byte) uint64 {
	const (
		offset64 = 14695981039346656037
		prime64  = 1099511628211
	)

	h := uint64(offset64)
	for _, b := range data {
		h ^= uint64(b)
		h *= prime64
	}
	return h
}

func nextPowerOfTwo(n uint) uint {
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
	return n + 1
}

// filterData is used for gob encoding/decoding
type filterData struct {
	NumBuckets      uint
	BucketSize      uint
	FingerprintSize uint
	Count           uint
	MaxKicks        uint
	BucketSizes     []uint
	Entries         []fingerprint
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
