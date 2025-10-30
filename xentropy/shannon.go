package xentropy

import "math"

// Shannon calculates the Shannon entropy of the input data in bits.
// Shannon entropy measures the average amount of information (in bits)
// produced by a stochastic source of data.
//
// Formula: H(X) = -Σ P(x) * log2(P(x))
//
// Returns a value between 0 (no entropy) and log2(n) where n is the
// number of unique symbols. For byte data, maximum entropy is 8 bits.
func Shannon(data []byte) float64 {
	if len(data) == 0 {
		return 0
	}

	// Count frequency of each byte
	freq := make(map[byte]int, 256)
	for _, b := range data {
		freq[b]++
	}

	// Calculate entropy
	var entropy float64
	length := float64(len(data))

	for _, count := range freq {
		if count > 0 {
			probability := float64(count) / length
			entropy -= probability * math.Log2(probability)
		}
	}

	return entropy
}

// Normalized calculates the normalized Shannon entropy (0-1 scale).
// This divides the Shannon entropy by the maximum possible entropy
// for the given number of unique symbols.
//
// Returns:
//   - 0: No entropy (all symbols are the same)
//   - 1: Maximum entropy (perfectly random distribution)
func Normalized(data []byte) float64 {
	if len(data) == 0 {
		return 0
	}

	entropy := Shannon(data)

	// Count unique symbols
	unique := make(map[byte]struct{}, 256)
	for _, b := range data {
		unique[b] = struct{}{}
	}

	// Maximum entropy is log2(unique symbols)
	maxEntropy := math.Log2(float64(len(unique)))
	if maxEntropy == 0 {
		return 0
	}

	return entropy / maxEntropy
}

// Metric returns an entropy metric between 0-100.
// This is a user-friendly representation where:
//   - 0-25: Low entropy (predictable)
//   - 25-50: Moderate entropy
//   - 50-75: Good entropy
//   - 75-100: Excellent entropy (highly random)
func Metric(data []byte) float64 {
	return Normalized(data) * 100
}

// IsRandom checks if data appears to be randomly generated.
// Returns true if the normalized entropy is above the threshold (default 0.9).
func IsRandom(data []byte, threshold float64) bool {
	if threshold <= 0 {
		threshold = 0.9
	}
	return Normalized(data) >= threshold
}
