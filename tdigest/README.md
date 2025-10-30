# tdigest

A high-performance T-Digest implementation in Go for accurate quantile estimation from streaming or distributed data.

## Features

- **Accurate quantile estimation**: Especially good at extreme quantiles (p99, p99.9, p99.99)
- **Streaming computation**: Process unlimited data with bounded memory
- **Mergeable**: Combine multiple t-digests for distributed systems
- **Small memory footprint**: Configurable compression factor
- **Better than histograms**: More accurate for percentiles with less memory
- **Export/Import**: Serialize for storage or network transmission
- **Zero dependencies**: Only uses Go standard library

## What is T-Digest?

T-Digest is a probabilistic data structure for computing approximate quantiles from streaming data or distributed data. It provides:

**Key Properties:**

- **Adaptive precision**: Higher accuracy at extreme quantiles
- **Bounded memory**: Size controlled by compression factor
- **Mergeable**: Can combine multiple t-digests efficiently
- **Streaming friendly**: Process data as it arrives

**Use Cases:** Monitoring systems, analytics platforms, database query optimization, distributed tracing, performance analysis, SLA tracking.

## Installation

```bash
go get github.com/vitalvas/gokit/tdigest
```

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/vitalvas/gokit/tdigest"
)

func main() {
    // Create t-digest with compression = 100
    td := tdigest.New(100)

    // Add values
    for i := 0; i < 10000; i++ {
        td.Add(float64(i))
    }

    // Query quantiles
    fmt.Printf("Median: %.2f\n", td.Quantile(0.5))
    fmt.Printf("P95: %.2f\n", td.Quantile(0.95))
    fmt.Printf("P99: %.2f\n", td.Quantile(0.99))
    fmt.Printf("P99.9: %.2f\n", td.Quantile(0.999))

    // Get statistics
    fmt.Printf("Count: %.0f\n", td.Count())
    fmt.Printf("Min: %.2f\n", td.Min())
    fmt.Printf("Max: %.2f\n", td.Max())
    fmt.Printf("Mean: %.2f\n", td.Mean())
}
```

## Creating a T-Digest

### New

Create a new t-digest with specified compression factor.

```go
// Default compression (good balance)
td := tdigest.New(100)

// High accuracy (more memory)
td := tdigest.New(200)

// Low memory (less accuracy)
td := tdigest.New(50)
```

**Compression Factor:**

- Controls size-accuracy tradeoff
- Higher values: More accuracy, more memory
- Lower values: Less accuracy, less memory

| Compression | Centroids | Accuracy | Use Case |
|-------------|-----------|----------|----------|
| 50 | 25-50 | Good | Low memory constraints |
| 100 | 50-100 | Excellent | Default (recommended) |
| 200 | 100-200 | Very High | Critical accuracy needs |

## Adding Values

### Add

Add a single value to the t-digest.

```go
td := tdigest.New(100)

// Add values
td.Add(42.0)
td.Add(100.5)
td.Add(15.3)
```

### AddWeighted

Add a value with a specific weight.

```go
td := tdigest.New(100)

// Add value with weight 5
td.AddWeighted(50.0, 5.0)

// Equivalent to adding 50.0 five times
for i := 0; i < 5; i++ {
    td.Add(50.0)
}
```

**Use Case:** Aggregating pre-counted data

## Querying Quantiles

### Quantile

Get the estimated value at a given quantile (0-1).

```go
td := tdigest.New(100)
for i := 0; i < 10000; i++ {
    td.Add(float64(i))
}

// Common quantiles
median := td.Quantile(0.5)    // 50th percentile
p75 := td.Quantile(0.75)      // 75th percentile
p95 := td.Quantile(0.95)      // 95th percentile
p99 := td.Quantile(0.99)      // 99th percentile
p999 := td.Quantile(0.999)    // 99.9th percentile
```

**Performance:** ~6µs per query on 10K values

### CDF

Get the cumulative distribution function value at x.

```go
td := tdigest.New(100)
for i := 0; i < 1000; i++ {
    td.Add(float64(i))
}

// What fraction of values are <= 500?
fraction := td.CDF(500)
fmt.Printf("%.1f%% of values <= 500\n", fraction*100)
// Output: 50.0% of values <= 500
```

**Returns:** Proportion of values ≤ x (0-1 range)

## Statistics

### Count

Get the total number of values.

```go
td := tdigest.New(100)
td.Add(10)
td.Add(20)
td.AddWeighted(30, 5)

count := td.Count()
// Output: 7 (1 + 1 + 5)
```

### Min / Max

Get minimum and maximum values.

```go
td := tdigest.New(100)
td.Add(42)
td.Add(15)
td.Add(99)

fmt.Printf("Range: [%.0f, %.0f]\n", td.Min(), td.Max())
// Output: Range: [15, 99]
```

### Mean

Get the approximate mean of all values.

```go
td := tdigest.New(100)
for i := 1; i <= 100; i++ {
    td.Add(float64(i))
}

mean := td.Mean()
// Output: ~50.5
```

## Merging T-Digests

### Merge

Combine multiple t-digests for distributed computation.

```go
// Server 1
td1 := tdigest.New(100)
for i := 0; i < 5000; i++ {
    td1.Add(float64(i))
}

// Server 2
td2 := tdigest.New(100)
for i := 5000; i < 10000; i++ {
    td2.Add(float64(i))
}

// Merge on aggregator
td1.Merge(td2)

// Now td1 contains data from both servers
median := td1.Quantile(0.5)
// Output: ~5000
```

**Note:** Both t-digests should have the same compression factor for best results.

## Utility Operations

### Reset

Clear all data from the t-digest.

```go
td := tdigest.New(100)
td.Add(10)
td.Add(20)

td.Reset()

fmt.Printf("Count: %.0f\n", td.Count())
// Output: Count: 0
```

### Compression

Get the compression factor.

```go
td := tdigest.New(150)
fmt.Printf("Compression: %.0f\n", td.Compression())
// Output: Compression: 150
```

## Serialization

### Export

Serialize the t-digest for storage or transmission.

```go
td := tdigest.New(100)
for i := 0; i < 10000; i++ {
    td.Add(float64(i))
}

// Export to bytes
data, err := td.Export()
if err != nil {
    log.Fatal(err)
}

// Save to file
os.WriteFile("tdigest.dat", data, 0644)
```

### Import

Deserialize a t-digest from exported data.

```go
// Load from file
data, err := os.ReadFile("tdigest.dat")
if err != nil {
    log.Fatal(err)
}

// Import
td, err := tdigest.Import(data)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Loaded digest with %.0f values\n", td.Count())
```

## Use Cases

### Response Time Monitoring

Track API response time percentiles.

```go
responseTimes := tdigest.New(100)

func recordRequest(duration float64) {
    responseTimes.Add(duration)
}

func getMetrics() map[string]float64 {
    return map[string]float64{
        "p50":  responseTimes.Quantile(0.50),
        "p95":  responseTimes.Quantile(0.95),
        "p99":  responseTimes.Quantile(0.99),
        "p999": responseTimes.Quantile(0.999),
        "max":  responseTimes.Max(),
    }
}
```

### SLA Tracking

Monitor SLA compliance.

```go
func checkSLA(td *tdigest.TDigest, slaMs float64) {
    p99 := td.Quantile(0.99)

    if p99 > slaMs {
        fmt.Printf("SLA breach: P99=%.2fms exceeds %2fms\n", p99, slaMs)
    } else {
        fmt.Printf("SLA OK: P99=%.2fms under %.2fms\n", p99, slaMs)
    }
}
```

### Distributed Percentiles

Aggregate percentiles across multiple servers.

```go
// Each server maintains local t-digest
func serverMetrics() *tdigest.TDigest {
    td := tdigest.New(100)

    for _, request := range requests {
        td.Add(request.Duration)
    }

    return td
}

// Aggregator combines all servers
func globalMetrics(servers []string) *tdigest.TDigest {
    global := tdigest.New(100)

    for _, server := range servers {
        local := fetchDigestFromServer(server)
        global.Merge(local)
    }

    return global
}
```

### Time-Series Aggregation

Compute rolling percentiles over time windows.

```go
type RollingDigest struct {
    current *tdigest.TDigest
    window  time.Duration
    lastReset time.Time
}

func (r *RollingDigest) Add(value float64) {
    now := time.Now()

    if now.Sub(r.lastReset) > r.window {
        r.current.Reset()
        r.lastReset = now
    }

    r.current.Add(value)
}

func (r *RollingDigest) Quantile(q float64) float64 {
    return r.current.Quantile(q)
}
```

### Database Query Latency

Track query performance.

```go
queryLatency := tdigest.New(100)

func executeQuery(query string) (result interface{}, err error) {
    start := time.Now()
    defer func() {
        duration := time.Since(start).Milliseconds()
        queryLatency.Add(float64(duration))
    }()

    return db.Query(query)
}

func printQueryStats() {
    fmt.Printf("Query Latency Stats:\n")
    fmt.Printf("  Median: %.2fms\n", queryLatency.Quantile(0.5))
    fmt.Printf("  P95: %.2fms\n", queryLatency.Quantile(0.95))
    fmt.Printf("  P99: %.2fms\n", queryLatency.Quantile(0.99))
}
```

## Performance Characteristics

### Time Complexity

| Operation | Average | Description |
|-----------|---------|-------------|
| `Add` | O(log n) | Insert with compression |
| `AddWeighted` | O(log n) | Insert weighted value |
| `Quantile` | O(n) | Query percentile (n = centroids) |
| `CDF` | O(n) | Cumulative distribution |
| `Merge` | O(n + m) | Combine digests |
| `Mean` | O(n) | Calculate average |

Where n = number of centroids (~compression factor)

### Space Complexity

**Memory usage:** O(compression)

**Typical sizes:**

| Compression | Centroids | Memory | Values Represented |
|-------------|-----------|--------|-------------------|
| 50 | ~50 | ~1 KB | Millions |
| 100 | ~100 | ~2 KB | Billions |
| 200 | ~200 | ~4 KB | Unlimited |

### Benchmarks (Apple M3 Pro)

| Operation | Time | Allocations |
|-----------|------|-------------|
| Add | ~112 µs | 4 allocs |
| Quantile | ~6.1 µs | 0 allocs |
| CDF | ~2.8 µs | 0 allocs |
| Export | ~230.6 µs | 40 allocs |
| Import | ~220.6 µs | 202 allocs |

**Performance Notes:**

- Quantile queries are fast (~6µs)
- CDF is very fast (~3µs)
- Add includes compression overhead
- Zero allocations for read operations

## Accuracy

### Quantile Accuracy

**Theoretical error:** < 1% for most quantiles

**Measured accuracy (10K uniform values):**

| Quantile | Expected | Actual | Error |
|----------|----------|--------|-------|
| 0.50 | 5000 | 5001 | 0.02% |
| 0.95 | 9500 | 9505 | 0.05% |
| 0.99 | 9900 | 9898 | 0.02% |
| 0.999 | 9990 | 9989 | 0.01% |

**Key Insight:** Accuracy is better at extreme quantiles (the most important percentiles for monitoring).

### Compression Factor Impact

Higher compression = better accuracy, more memory:

| Compression | P99 Error | Memory |
|-------------|-----------|--------|
| 50 | ~0.5% | 1 KB |
| 100 | ~0.2% | 2 KB |
| 200 | ~0.1% | 4 KB |

## Limitations

### Approximate Results

T-digest provides estimates, not exact values.

```go
td := tdigest.New(100)
for i := 0; i < 10000; i++ {
    td.Add(float64(i))
}

// Exact median would be 5000
// T-digest might return 4998 or 5002
median := td.Quantile(0.5)
```

**Mitigation:** Use higher compression for better accuracy.

### Streaming Only (No Deletion)

Cannot remove individual values.

```go
td := tdigest.New(100)
td.Add(42)
// No td.Remove(42) method
```

**Workaround:** Use time-windowed digests or reset periodically.

### Compression Overhead

Adding values triggers periodic compression.

```go
// Compression happens automatically
// Cost: O(n log n) where n = number of centroids
for i := 0; i < 1000000; i++ {
    td.Add(float64(i)) // Occasionally slower due to compression
}
```

**Mitigation:** Use batch inserts when possible.

## Best Practices

### Choose Appropriate Compression

```go
// Low memory, good accuracy
td := tdigest.New(50)

// Default (recommended)
td := tdigest.New(100)

// High accuracy, more memory
td := tdigest.New(200)
```

### Batch Processing

```go
// Good: Process in batches
func processBatch(values []float64) {
    td := tdigest.New(100)
    for _, v := range values {
        td.Add(v)
    }
    return td
}
```

### Time Windows

```go
// Reset periodically for rolling metrics
ticker := time.NewTicker(5 * time.Minute)
go func() {
    for range ticker.C {
        td.Reset()
    }
}()
```

### Distributed Aggregation

```go
// Each worker maintains local t-digest
workers := make([]*tdigest.TDigest, numWorkers)
for i := range workers {
    workers[i] = tdigest.New(100)
}

// Aggregate on coordinator
coordinator := tdigest.New(100)
for _, worker := range workers {
    coordinator.Merge(worker)
}
```

## Comparison with Other Approaches

| Approach | Memory | Accuracy | Merge | Stream | Quantiles |
|----------|--------|----------|-------|--------|-----------|
| Sorted Array | O(n) | Exact | ❌ | ❌ | O(1) |
| Histogram | O(buckets) | Poor | ✅ | ✅ | O(1) |
| **T-Digest** | **O(δ)** | **Excellent** | **✅** | **✅** | **O(δ)** |
| Reservoir | O(k) | Good | ❌ | ✅ | O(k log k) |

**T-Digest Advantages:**

- **Mergeable** (unlike sorted arrays or reservoirs)
- **Adaptive precision** (better at extremes than histograms)
- **Bounded memory** (unlike sorted arrays)
- **Streaming friendly** (unlike offline algorithms)

**When to Use:**

- ✅ Need accurate extreme quantiles (p99+)
- ✅ Distributed or streaming data
- ✅ Limited memory budget
- ✅ Need to merge across time or servers
- ❌ Need exact results (use sorting)
- ❌ Need very fast adds (use histogram)

## License

This project is part of the [gokit](https://github.com/vitalvas/gokit) library.
