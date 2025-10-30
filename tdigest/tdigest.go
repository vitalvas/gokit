package tdigest

import (
	"encoding/gob"
	"math"
	"sort"
)

// TDigest is a data structure for accurate estimation of quantiles.
// It provides streaming computation of quantiles with:
// - High accuracy for extreme quantiles (p99, p99.9, p99.99)
// - Small memory footprint (configurable)
// - Ability to merge multiple t-digests
// - Better accuracy than histograms
//
// Use cases: Monitoring, analytics, distributed systems, percentile calculation
type TDigest struct {
	compression float64
	centroids   []centroid
	count       float64
	min         float64
	max         float64
}

// centroid represents a cluster of values with a mean and weight
type centroid struct {
	Mean   float64
	Weight float64
}

// New creates a new t-digest with the specified compression factor.
//
// Compression controls the size-accuracy tradeoff:
//   - Higher values: More accuracy, more memory
//   - Lower values: Less accuracy, less memory
//
// Recommended values:
//   - 100: Default, good balance (50-100 centroids)
//   - 200: High accuracy (100-200 centroids)
//   - 50: Low memory (25-50 centroids)
func New(compression float64) *TDigest {
	if compression <= 0 {
		compression = 100
	}

	return &TDigest{
		compression: compression,
		centroids:   make([]centroid, 0, int(compression)*2),
		count:       0,
		min:         math.Inf(1),
		max:         math.Inf(-1),
	}
}

// Add inserts a value into the t-digest.
func (td *TDigest) Add(value float64) {
	td.AddWeighted(value, 1)
}

// AddWeighted inserts a value with a specific weight.
func (td *TDigest) AddWeighted(value, weight float64) {
	if weight <= 0 {
		return
	}

	// Update min/max
	if value < td.min {
		td.min = value
	}
	if value > td.max {
		td.max = value
	}

	// Add as new centroid
	td.centroids = append(td.centroids, centroid{
		Mean:   value,
		Weight: weight,
	})
	td.count += weight

	// Compress if needed
	if len(td.centroids) > int(td.compression)*2 {
		td.compress()
	}
}

// Quantile returns the estimated value at the given quantile (0-1).
//
// Examples:
//   - Quantile(0.5) returns median
//   - Quantile(0.95) returns 95th percentile
//   - Quantile(0.99) returns 99th percentile
func (td *TDigest) Quantile(q float64) float64 {
	if len(td.centroids) == 0 {
		return math.NaN()
	}

	if q < 0 {
		q = 0
	}
	if q > 1 {
		q = 1
	}

	// Sort centroids if needed
	if !td.isSorted() {
		td.sort()
	}

	if len(td.centroids) == 1 {
		return td.centroids[0].Mean
	}

	// Handle edge cases
	if q == 0 {
		return td.min
	}
	if q == 1 {
		return td.max
	}

	// Find quantile using linear interpolation
	index := q * td.count
	weightSum := float64(0)

	for i := 0; i < len(td.centroids); i++ {
		c := &td.centroids[i]
		weightSum += c.Weight

		if weightSum >= index {
			// Interpolate between this centroid and previous
			if i == 0 {
				return c.Mean
			}

			prev := &td.centroids[i-1]
			prevWeightSum := weightSum - c.Weight

			// Linear interpolation
			if c.Weight > 1 {
				// Interpolate within centroid
				fraction := (index - prevWeightSum) / c.Weight
				return prev.Mean + fraction*(c.Mean-prev.Mean)
			}

			return c.Mean
		}
	}

	return td.max
}

// CDF returns the cumulative distribution function value at x.
// Returns the proportion of values <= x.
func (td *TDigest) CDF(x float64) float64 {
	if len(td.centroids) == 0 {
		return math.NaN()
	}

	if !td.isSorted() {
		td.sort()
	}

	if x < td.min {
		return 0
	}
	if x > td.max {
		return 1
	}

	if len(td.centroids) == 1 {
		if x < td.centroids[0].Mean {
			return 0
		}
		return 1
	}

	weightSum := float64(0)

	for i := 0; i < len(td.centroids); i++ {
		c := &td.centroids[i]

		if x < c.Mean {
			// Interpolate between previous and current centroid
			if i == 0 {
				return 0
			}

			prev := &td.centroids[i-1]
			prevWeightSum := weightSum

			// Linear interpolation
			fraction := (x - prev.Mean) / (c.Mean - prev.Mean)
			return (prevWeightSum + fraction*c.Weight) / td.count
		}

		weightSum += c.Weight
	}

	return 1
}

// Count returns the total number of values added to the t-digest.
func (td *TDigest) Count() float64 {
	return td.count
}

// Min returns the minimum value seen.
func (td *TDigest) Min() float64 {
	if len(td.centroids) == 0 {
		return math.NaN()
	}
	return td.min
}

// Max returns the maximum value seen.
func (td *TDigest) Max() float64 {
	if len(td.centroids) == 0 {
		return math.NaN()
	}
	return td.max
}

// Mean returns the approximate mean of all values.
func (td *TDigest) Mean() float64 {
	if len(td.centroids) == 0 || td.count == 0 {
		return math.NaN()
	}

	sum := float64(0)
	for i := range td.centroids {
		sum += td.centroids[i].Mean * td.centroids[i].Weight
	}

	return sum / td.count
}

// Merge combines another t-digest into this one.
// Both t-digests should have the same compression factor for best results.
func (td *TDigest) Merge(other *TDigest) {
	if other == nil || len(other.centroids) == 0 {
		return
	}

	// Update min/max
	if other.min < td.min {
		td.min = other.min
	}
	if other.max > td.max {
		td.max = other.max
	}

	// Add all centroids
	td.centroids = append(td.centroids, other.centroids...)
	td.count += other.count

	// Compress the combined result
	td.compress()
}

// Reset clears all data from the t-digest.
func (td *TDigest) Reset() {
	td.centroids = td.centroids[:0]
	td.count = 0
	td.min = math.Inf(1)
	td.max = math.Inf(-1)
}

// Compression returns the compression factor.
func (td *TDigest) Compression() float64 {
	return td.compression
}

// Export serializes the t-digest for storage or transmission.
func (td *TDigest) Export() ([]byte, error) {
	var buf []byte
	enc := gob.NewEncoder(&gobWriter{buf: &buf})

	data := &tdigestData{
		Compression: td.compression,
		Centroids:   td.centroids,
		Count:       td.count,
		Min:         td.min,
		Max:         td.max,
	}

	if err := enc.Encode(data); err != nil {
		return nil, err
	}

	return buf, nil
}

// Import deserializes a t-digest from exported data.
func Import(data []byte) (*TDigest, error) {
	var tdigestData tdigestData
	dec := gob.NewDecoder(&gobReader{buf: data})

	if err := dec.Decode(&tdigestData); err != nil {
		return nil, err
	}

	td := &TDigest{
		compression: tdigestData.Compression,
		centroids:   tdigestData.Centroids,
		count:       tdigestData.Count,
		min:         tdigestData.Min,
		max:         tdigestData.Max,
	}

	return td, nil
}

// Internal methods

func (td *TDigest) compress() {
	if len(td.centroids) <= 1 {
		return
	}

	// Sort centroids by mean
	td.sort()

	// Merge centroids based on t-digest algorithm
	newCentroids := make([]centroid, 0, len(td.centroids))
	current := td.centroids[0]

	weightSum := float64(0)

	for i := 1; i < len(td.centroids); i++ {
		c := td.centroids[i]
		weightSum += current.Weight

		// Check if we should merge this centroid
		q := weightSum / td.count
		k := td.scaleFunction(q)

		if current.Weight+c.Weight <= td.maxWeight(k) {
			// Merge centroids
			totalWeight := current.Weight + c.Weight
			current.Mean = (current.Mean*current.Weight + c.Mean*c.Weight) / totalWeight
			current.Weight = totalWeight
		} else {
			// Start new centroid
			newCentroids = append(newCentroids, current)
			current = c
		}
	}

	// Add last centroid
	newCentroids = append(newCentroids, current)

	td.centroids = newCentroids
}

func (td *TDigest) scaleFunction(q float64) float64 {
	// Using k_2 scale function (better for extreme quantiles)
	return (td.compression / (2 * math.Pi)) * math.Asin(2*q-1)
}

func (td *TDigest) maxWeight(k float64) float64 {
	return (4 * td.count * k * (1 - k)) / td.compression
}

func (td *TDigest) sort() {
	sort.Slice(td.centroids, func(i, j int) bool {
		return td.centroids[i].Mean < td.centroids[j].Mean
	})
}

func (td *TDigest) isSorted() bool {
	for i := 1; i < len(td.centroids); i++ {
		if td.centroids[i].Mean < td.centroids[i-1].Mean {
			return false
		}
	}
	return true
}

// tdigestData is used for gob encoding/decoding
type tdigestData struct {
	Compression float64
	Centroids   []centroid
	Count       float64
	Min         float64
	Max         float64
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
