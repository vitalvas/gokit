# xentropy

Entropy calculator for measuring randomness and information content in data.

## Features

- **Shannon entropy calculation**: Measure average information content in bits
- **Min-entropy calculation**: Measure worst-case predictability for security
- **Normalized entropy**: Scale entropy to 0-1 range
- **Randomness detection**: Determine if data appears randomly generated
- **Security assessment**: Check if data meets cryptographic quality standards
- **User-friendly metrics**: Convert entropy to 0-100 scale
- **Zero dependencies**: Only uses Go standard library

## Entropy Types

### Shannon Entropy

Shannon entropy measures the **average** amount of information (in bits) produced by a stochastic source of data. It quantifies the uncertainty or randomness in the data:

- **0 bits**: No entropy (all symbols are the same)
- **1 bit**: Binary data with equal distribution
- **8 bits**: Maximum entropy for byte data (all 256 bytes equally distributed)

**Formula:** H(X) = -Σ P(x) * log2(P(x))

**Use Cases**: Data compression analysis, average randomness assessment, information content measurement.

### Min-Entropy

Min-entropy (Rényi entropy with α=∞) measures the **worst-case** predictability by focusing on the most likely outcome. Unlike Shannon entropy which gives an average, min-entropy tells you what an attacker exploiting the most common pattern could achieve.

- **0 bits**: Completely predictable (all symbols are the same)
- **8 bits**: Maximum unpredictability for byte data (uniform distribution)

**Formula:** H_∞(X) = -log2(max(P(x)))

**Use Cases**: Cryptographic key quality, password security, worst-case security analysis.

**Why Both?**

- **Shannon**: Average-case analysis (e.g., "How compressible is this data?")
- **Min-Entropy**: Worst-case analysis (e.g., "How secure is this key against focused attacks?")

## Installation

```bash
go get github.com/vitalvas/gokit/xentropy
```

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/vitalvas/gokit/xentropy"
)

func main() {
    data := []byte("Hello, World!")

    // Calculate Shannon entropy (average case)
    shannon := xentropy.Shannon(data)
    fmt.Printf("Shannon entropy: %.2f bits\n", shannon)

    // Calculate min-entropy (worst case)
    minEnt := xentropy.MinEntropy(data)
    fmt.Printf("Min-entropy: %.2f bits\n", minEnt)

    // Check if data is random (average randomness)
    random := xentropy.IsRandom([]byte("random data"), 0.9)
    fmt.Printf("Is random: %v\n", random)

    // Check if data is cryptographically secure (worst-case)
    secure := xentropy.IsSecure([]byte("crypto key"), 0.8)
    fmt.Printf("Is secure: %v\n", secure)
}
```

## Shannon Entropy Calculation

### Shannon

Calculate Shannon entropy in bits.

```go
data := []byte("hello world")
entropy := xentropy.Shannon(data)
fmt.Printf("Entropy: %.2f bits\n", entropy)
// Output: Entropy: 2.85 bits
```

**Examples:**

| Data | Entropy | Interpretation |
|------|---------|----------------|
| "aaaaaaa" | 0.00 bits | No entropy |
| "aabbccdd" | 2.00 bits | 4 symbols, balanced |
| "password" | 2.75 bits | English text |
| Random bytes | ~8.00 bits | Maximum entropy |

## Min-Entropy Calculation

### MinEntropy

Calculate min-entropy (worst-case predictability) in bits.

```go
data := []byte("aaaaaaaaab")
minEnt := xentropy.MinEntropy(data)
fmt.Printf("Min-Entropy: %.2f bits\n", minEnt)
// Output: Min-Entropy: 0.15 bits (9 'a's, 1 'b' - very predictable)
```

**Comparison with Shannon:**

```go
data := []byte("aaaaaaaaab")
shannon := xentropy.Shannon(data)     // ~0.47 bits (average)
minEnt := xentropy.MinEntropy(data)   // ~0.15 bits (worst-case)

// For uniform data, they're equal:
uniform := []byte("aabb")
shannonUniform := xentropy.Shannon(uniform)       // 1.00 bits
minEntUniform := xentropy.MinEntropy(uniform)     // 1.00 bits
```

**When to Use:**

- Use **Shannon** for data compression, average randomness
- Use **MinEntropy** for cryptographic keys, security-critical applications

## Normalized Entropy

### Normalized

Calculate normalized entropy (0-1 scale).

```go
data := []byte("aabb")
norm := xentropy.Normalized(data)
fmt.Printf("Normalized: %.2f\n", norm)
// Output: Normalized: 1.00 (perfect entropy for 2 symbols)
```

**Interpretation:**

- **0.0**: No entropy (all same characters)
- **0.5**: Moderate entropy
- **1.0**: Maximum entropy (perfectly balanced distribution)

### MinNormalized

Normalized min-entropy for worst-case analysis.

```go
data := []byte("aaab")
norm := xentropy.MinNormalized(data)
fmt.Printf("Min-Normalized: %.2f\n", norm)
// max_count=3, total=4, min_entropy=-log2(3/4)=0.415
// unique=2, max_min_entropy=log2(2)=1.0
// normalized=0.415/1.0=0.415
```

## Randomness and Security Detection

### IsRandom

Check if data appears randomly generated.

```go
// Truly random data
randomData := make([]byte, 256)
rand.Read(randomData)
isRandom := xentropy.IsRandom(randomData, 0.9)
fmt.Printf("Is random: %v\n", isRandom)
// Output: Is random: true

// Predictable data
pattern := []byte("ababababab")
isRandom = xentropy.IsRandom(pattern, 0.9)
fmt.Printf("Is random: %v\n", isRandom)
// Output: Is random: false
```

**Parameters:**

- `data`: Byte slice to analyze
- `threshold`: Normalized entropy threshold (default 0.9)
  - Higher threshold = stricter randomness requirement
  - Typical values: 0.85-0.95

**Use Cases:**

- Verify cryptographic key quality
- Test random number generators
- Detect non-random patterns in data
- Quality control for entropy sources

### IsSecure

Check if data has sufficient min-entropy for cryptographic use.

```go
// Good cryptographic key
goodKey := make([]byte, 32)
rand.Read(goodKey)
isSecure := xentropy.IsSecure(goodKey, 0.8)
fmt.Printf("Is secure: %v\n", isSecure)
// Output: Is secure: true

// Weak key with repeated patterns
weakKey := []byte("abcdabcdabcd")
isSecure = xentropy.IsSecure(weakKey, 0.8)
fmt.Printf("Is secure: %v\n", isSecure)
// Output: Is secure: false
```

**Parameters:**

- `data`: Byte slice to analyze
- `threshold`: Normalized min-entropy threshold (default 0.8)
  - Higher threshold = stricter security requirement
  - Typical values: 0.7-0.9 for cryptographic use

**Use Cases:**

- Validate cryptographic keys before use
- Ensure password/passphrase quality
- Verify random number generator output
- Security-critical entropy sources

**IsRandom vs IsSecure:**

- `IsRandom`: Uses Shannon entropy (average case) - threshold 0.9
- `IsSecure`: Uses min-entropy (worst case) - threshold 0.8
- Use `IsSecure` for security-critical applications

## Metrics and User-Friendly Output

### Metric

Convert entropy to 0-100 scale.

```go
metric := xentropy.Metric([]byte("P@ssw0rd!"))
fmt.Printf("Entropy metric: %.1f/100\n", metric)
```

**Scale:**

- **0-25**: Low entropy (predictable)
- **25-50**: Moderate entropy
- **50-75**: Good entropy
- **75-100**: Excellent entropy (highly random)

### MinMetric

Convert min-entropy to 0-100 scale for worst-case assessment.

```go
metric := xentropy.MinMetric([]byte("aaab"))
fmt.Printf("Min-entropy metric: %.1f/100\n", metric)
// Output: 41.5/100 (moderate worst-case unpredictability)
```

## Use Cases

### Cryptographic Key Quality

```go
key := generateKey()

// Check worst-case security with min-entropy
if !xentropy.IsSecure(key, 0.8) {
    return errors.New("insufficient key entropy (worst-case)")
}

// Also check average randomness
if !xentropy.IsRandom(key, 0.9) {
    return errors.New("insufficient key entropy (average)")
}

// Compare both measures
shannon := xentropy.Shannon(key)
minEnt := xentropy.MinEntropy(key)
fmt.Printf("Key quality - Shannon: %.2f bits, Min: %.2f bits\n", shannon, minEnt)
```

### Data Compression Analysis

```go
data := loadFile("data.txt")
entropy := xentropy.Shannon(data)
maxCompression := 8.0 / entropy
fmt.Printf("Theoretical max compression: %.2fx\n", maxCompression)
```

### Random Number Generator Testing

```go
func testRNG(rng func() []byte) {
    samples := 1000
    passed := 0

    for i := 0; i < samples; i++ {
        data := rng()
        if xentropy.IsRandom(data, 0.9) {
            passed++
        }
    }

    fmt.Printf("RNG quality: %.1f%%\n", float64(passed)/float64(samples)*100)
}
```

### Text vs Binary Detection

```go
func detectDataType(data []byte) string {
    entropy := xentropy.Shannon(data)

    switch {
    case entropy < 3.0:
        return "text (low entropy)"
    case entropy < 6.0:
        return "compressed or encoded"
    default:
        return "binary or encrypted (high entropy)"
    }
}
```

## Performance

### Benchmarks (Apple M3 Pro)

| Operation | Input Size | Time | Memory |
|-----------|-----------|------|--------|
| Shannon | 100 B | ~3.1 µs | ~9.5 KB |
| Shannon | 500 B | ~8.0 µs | ~9.5 KB |
| Shannon | 5 KB | ~61 µs | ~9.5 KB |
| Normalized | 500 B | ~12.7 µs | ~11 KB |
| IsRandom | 256 B | ~11.9 µs | ~11 KB |
| MinEntropy | 100 B | ~2.9 µs | ~9.5 KB |
| MinEntropy | 500 B | ~7.7 µs | ~9.5 KB |
| MinEntropy | 5 KB | ~60 µs | ~9.5 KB |
| MinNormalized | 500 B | ~12.3 µs | ~11 KB |
| IsSecure | 256 B | ~11.3 µs | ~11 KB |

**Performance Characteristics:**

- Linear time complexity: O(n) where n is data length
- Constant space complexity: O(1) - uses fixed-size frequency map (256 entries)
- Scales well with data size
- Microsecond range for kilobyte-sized data

## Examples

### File Entropy Analyzer

```go
package main

import (
    "fmt"
    "os"
    "github.com/vitalvas/gokit/xentropy"
)

func analyzeFile(filename string) {
    data, err := os.ReadFile(filename)
    if err != nil {
        panic(err)
    }

    entropy := xentropy.Shannon(data)
    normalized := xentropy.Normalized(data)
    metric := xentropy.Metric(data)

    fmt.Printf("File: %s\n", filename)
    fmt.Printf("Size: %d bytes\n", len(data))
    fmt.Printf("Shannon entropy: %.2f bits\n", entropy)
    fmt.Printf("Normalized entropy: %.2f\n", normalized)
    fmt.Printf("Entropy metric: %.1f/100\n", metric)

    // Classify file type by entropy
    switch {
    case entropy < 1.0:
        fmt.Println("Type: Very low entropy (highly repetitive)")
    case entropy < 3.0:
        fmt.Println("Type: Low entropy (text file)")
    case entropy < 5.0:
        fmt.Println("Type: Medium entropy (code, markup)")
    case entropy < 7.0:
        fmt.Println("Type: High entropy (compressed, encoded)")
    default:
        fmt.Println("Type: Very high entropy (encrypted, random)")
    }
}
```

## Best Practices

### 1. Choose Appropriate Thresholds

```go
// Cryptographic keys: high threshold
if !xentropy.IsRandom(cryptoKey, 0.95) {
    return errors.New("insufficient entropy")
}

// General randomness: moderate threshold
if !xentropy.IsRandom(sessionID, 0.85) {
    return errors.New("not random enough")
}
```

### 2. Consider Data Length

```go
// Short data may have misleading entropy
if len(data) < 10 {
    fmt.Println("Warning: sample too small for reliable entropy measurement")
}
```

### 3. Use Total Entropy for Security Assessment

```go
// Per-character entropy alone is insufficient for security
entropyPerChar := xentropy.Shannon([]byte(password))
totalEntropy := entropyPerChar * float64(len(password))

// For strong security, aim for at least 80 bits total
if totalEntropy < 80 {
    fmt.Println("Warning: insufficient entropy for strong security")
}
```

## Limitations

### False Positives in Randomness Detection

Perfectly balanced non-random patterns may pass randomness tests:

```go
// This is NOT random, but has perfect normalized entropy
pattern := []byte("aabb")
isRandom := xentropy.IsRandom(pattern, 0.99)
// Returns: true (perfect entropy for 2 symbols)
```

**Solution:** Combine entropy analysis with other tests (runs test, autocorrelation).

### Small Sample Sizes

Entropy measurements are less reliable for very small data:

```go
// Unreliable: too few bytes
entropy := xentropy.Shannon([]byte("ab"))

// Better: adequate sample size
entropy := xentropy.Shannon([]byte(strings.Repeat("ab", 50)))
```

**Recommendation:** Use at least 100 bytes for reliable measurements.

### Language-Specific Entropy

Natural language text has characteristic entropy ranges:

- English text: ~1-5 bits per character
- Code: ~4-6 bits per character
- Compressed data: ~7-8 bits per byte

These patterns don't indicate poor quality, just predictable structure.

## License

This project is part of the [gokit](https://github.com/vitalvas/gokit) library.
