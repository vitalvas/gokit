package xentropy

import "math"

// MinEntropy calculates the min-entropy of the input data in bits.
// Min-entropy (Rényi entropy with α=∞) measures the worst-case predictability
// by focusing on the most likely outcome. It's the most conservative entropy measure
// and is critical for cryptographic applications.
//
// Formula: H_∞(X) = -log2(max(P(x)))
//
// Unlike Shannon entropy which measures average unpredictability, min-entropy
// measures worst-case unpredictability - what an attacker exploiting the most
// common pattern could achieve.
//
// Returns a value between 0 (completely predictable) and log2(n) where n is the
// number of unique symbols. For byte data, maximum min-entropy is 8 bits.
func MinEntropy(data []byte) float64 {
	if len(data) == 0 {
		return 0
	}

	// Count frequency of each byte
	freq := make(map[byte]int, 256)
	for _, b := range data {
		freq[b]++
	}

	// Find maximum frequency (most common symbol)
	maxCount := 0
	for _, count := range freq {
		if count > maxCount {
			maxCount = count
		}
	}

	// Calculate min-entropy
	maxProb := float64(maxCount) / float64(len(data))
	return -math.Log2(maxProb)
}

// MinNormalized calculates the normalized min-entropy (0-1 scale).
// This divides the min-entropy by the maximum possible min-entropy
// for the given number of unique symbols.
//
// Returns:
//   - 0: Completely predictable (all symbols are the same)
//   - 1: Maximum min-entropy (perfectly uniform distribution)
func MinNormalized(data []byte) float64 {
	if len(data) == 0 {
		return 0
	}

	entropy := MinEntropy(data)

	// Count unique symbols
	unique := make(map[byte]struct{}, 256)
	for _, b := range data {
		unique[b] = struct{}{}
	}

	// Maximum min-entropy is log2(unique symbols)
	maxEntropy := math.Log2(float64(len(unique)))
	if maxEntropy == 0 {
		return 0
	}

	return entropy / maxEntropy
}

// MinMetric returns a min-entropy metric between 0-100.
// This is a user-friendly representation where:
//   - 0-25: Low min-entropy (highly predictable)
//   - 25-50: Moderate min-entropy
//   - 50-75: Good min-entropy
//   - 75-100: Excellent min-entropy (highly unpredictable)
func MinMetric(data []byte) float64 {
	return MinNormalized(data) * 100
}

// IsSecure checks if data has sufficient min-entropy for security purposes.
// Returns true if the min-entropy meets the threshold (default 0.8).
// This is more conservative than IsRandom and better suited for cryptographic applications.
func IsSecure(data []byte, threshold float64) bool {
	if threshold <= 0 {
		threshold = 0.8
	}
	return MinNormalized(data) >= threshold
}
