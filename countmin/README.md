# countmin

A high-performance Count-min sketch implementation in Go for frequency estimation in data streams.

## Features

- **Frequency estimation**: Estimate item frequencies with bounded error guarantees
- **Configurable accuracy**: Trade memory for accuracy with epsilon and delta parameters
- **Small memory footprint**: O(e/ε × ln(1/δ)) space complexity
- **Error guarantees**: count(x) ≤ true_count(x) + ε × N with probability 1-δ
- **Thread-safe**: Safe for concurrent use with RWMutex
- **Merge support**: Combine multiple sketches from distributed sources
- **Clone support**: Create independent copies for safe operations
- **Export/Import**: Serialize for storage or network transmission
- **High performance**: ~50ns per add/query operation
- **Rate tracking**: Track frequency rates with automatic decay
- **Zero dependencies**: Only uses Go standard library

## What is Count-min Sketch?

Count-min sketch is a probabilistic data structure that estimates the frequency of elements in a data stream. It provides approximate counts with:

- **Bounded error**: Guarantees maximum overestimation
- **Small memory**: Uses far less memory than exact counting
- **Fast updates**: Constant-time add and query operations
- **Never underestimates**: Always returns count ≥ true count

**Error Formula**: count(x) ≤ true_count(x) + ε × N (with probability 1-δ)

where:

- ε (epsilon) = error factor
- δ (delta) = probability of exceeding error bound
- N = total count of all items

**Use Cases**: Request frequency tracking, top-k queries, heavy hitter detection, rate limiting, traffic analysis.

## Installation

```bash
go get github.com/vitalvas/gokit/countmin
```

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/vitalvas/gokit/countmin"
)

func main() {
    // Create Count-min sketch with 0.1% error, 1% failure probability
    cm := countmin.New(0.001, 0.01)

    // Add items
    cm.UpdateString("/api/users")
    cm.UpdateString("/api/posts")
    cm.UpdateString("/api/users") // Duplicate

    // Query frequency
    count := cm.CountString("/api/users")
    fmt.Printf("Frequency: %d\n", count)
    // Output: Frequency: 2 (or slightly higher due to error)

    fmt.Printf("Total requests: %d\n", cm.Total())
    // Output: Total requests: 3
}
```

## Creating a Count-min Sketch

### New

Create a new Count-min sketch with error bounds.

```go
// 0.1% error, 1% failure probability
// Memory: ~106KB (width=2719, depth=5)
cm := countmin.New(0.001, 0.01)

// 1% error, 1% failure probability
// Memory: ~11KB (width=272, depth=5)
cm := countmin.New(0.01, 0.01)

// 0.1% error, 10% failure probability
// Memory: ~64KB (width=2719, depth=3)
cm := countmin.New(0.001, 0.1)
```

**Parameters:**

- `epsilon`: Error factor (typical: 0.001 to 0.01)
- `delta`: Failure probability (typical: 0.01 to 0.1)

**Memory vs Accuracy:**

| Epsilon | Delta | Width | Depth | Memory | Use Case |
|---------|-------|-------|-------|--------|----------|
| 0.001 | 0.01 | 2719 | 5 | ~106 KB | High accuracy |
| 0.01 | 0.01 | 272 | 5 | ~11 KB | **Recommended** |
| 0.01 | 0.1 | 272 | 3 | ~6 KB | Low memory |

### NewWithSize

Create with explicit dimensions (for advanced use).

```go
// 100 columns (width), 5 rows (depth)
cm := countmin.NewWithSize(100, 5)
```

## Adding Elements

### Add

Add raw bytes with a specified count.

```go
cm := countmin.New(0.01, 0.01)

data := []byte("user-123")
cm.Add(data, 5) // Add 5 occurrences
```

### AddString

Add a string with a specified count.

```go
cm := countmin.New(0.01, 0.01)

cm.AddString("/api/users", 10)
cm.AddString("/api/posts", 5)
```

### Update / UpdateString

Add a single occurrence.

```go
cm := countmin.New(0.01, 0.01)

// Add one occurrence
cm.UpdateString("/api/users")

// Equivalent to
cm.AddString("/api/users", 1)
```

## Querying Frequencies

### Count

Get the estimated frequency of an item.

```go
cm := countmin.New(0.01, 0.01)

// Add items
for i := 0; i < 1000; i++ {
    cm.UpdateString("/api/users")
}

count := cm.CountString("/api/users")
// count >= 1000 (may be slightly higher)
```

**Important**: Count-min sketch **never underestimates**, but may overestimate by up to ε × N.

### CountString

Query frequency of a string item.

```go
cm := countmin.New(0.01, 0.01)

cm.AddString("item-a", 100)
cm.AddString("item-b", 200)

count := cm.CountString("item-a")
// count >= 100
```

### Total

Get the total count of all items added.

```go
cm := countmin.New(0.01, 0.01)

cm.AddString("a", 10)
cm.AddString("b", 20)
cm.AddString("c", 30)

total := cm.Total()
// Output: 60
```

## Merging Sketches

### Merge

Merge another sketch into this one (mutates the receiver).

```go
cm1 := countmin.New(0.01, 0.01)
cm2 := countmin.New(0.01, 0.01)

// Add to different sketches
cm1.AddString("item-a", 100)
cm2.AddString("item-b", 200)

// Merge cm2 into cm1 (cm1 is modified)
err := cm1.Merge(cm2)
if err != nil {
    log.Fatal(err)
}

// cm1 now contains both items
fmt.Println(cm1.CountString("item-a")) // >= 100
fmt.Println(cm1.CountString("item-b")) // >= 200
fmt.Println(cm1.Total()) // 300
```

**Note:** Both sketches must have the same dimensions (width and depth).

**Use Cases:**

- Distributed counting across multiple servers
- Combining logs from different sources
- Time-windowed aggregations

## Utility Operations

### Clear

Reset all counters to zero.

```go
cm := countmin.New(0.01, 0.01)
cm.AddString("item", 100)

cm.Clear()

count := cm.CountString("item")
// Output: 0
```

### Clone

Create an independent copy.

```go
cm1 := countmin.New(0.01, 0.01)
cm1.AddString("item", 100)

// Clone creates independent copy
cm2 := cm1.Clone()

// Modify original without affecting clone
cm1.AddString("item", 50)

fmt.Println(cm1.CountString("item")) // >= 150
fmt.Println(cm2.CountString("item")) // >= 100 (unchanged)
```

### EstimatedError

Get the estimated error bound for current load.

```go
cm := countmin.New(0.01, 0.01)

// Add 10,000 items
for i := 0; i < 10000; i++ {
    cm.UpdateString(fmt.Sprintf("item-%d", i))
}

error := cm.EstimatedError()
// error ≈ ε × N = 0.01 × 10000 = 100
```

### Dimensions

Query sketch dimensions.

```go
cm := countmin.New(0.01, 0.01)

width := cm.Width()   // Number of columns
depth := cm.Depth()   // Number of rows
epsilon := cm.Epsilon() // Error factor
delta := cm.Delta()     // Failure probability
```

## Serialization

### Export

Serialize sketch for storage or transmission.

```go
cm := countmin.New(0.01, 0.01)

// Add data
cm.AddString("item-a", 100)
cm.AddString("item-b", 200)

// Export to bytes
data, err := cm.Export()
if err != nil {
    log.Fatal(err)
}

// Save to file
os.WriteFile("countmin.dat", data, 0644)
```

### Import

Deserialize sketch from exported data.

```go
// Load from file
data, err := os.ReadFile("countmin.dat")
if err != nil {
    log.Fatal(err)
}

// Import
cm, err := countmin.Import(data)
if err != nil {
    log.Fatal(err)
}

// Use imported sketch
count := cm.CountString("item-a")
fmt.Printf("Frequency: %d\n", count)
```

## Rate: Frequency Rate Tracking

The `Rate` type tracks the rate at which frequencies change with exponential decay, without requiring manual ticker management.

### When to Use Rate

**Use Rate when:**

- You want to track rate of frequency changes
- Events arrive at irregular intervals
- You need automatic decay based on time
- You want to detect frequency spikes
- You need real-time frequency rate tracking

**Difference from Sketch:**

| Feature | Sketch | Rate |
|---------|--------|------|
| Purpose | Count frequencies | Track frequency rate |
| Output | Frequency estimate | Rate of change |
| Time-based | No | Yes (automatic decay) |
| Memory | O(width × depth) | O(1) constant |
| Use case | Total frequency count | Real-time rate tracking |

### Creating a Rate

```go
// Create with 60 second half-life
r := countmin.NewRate(60 * time.Second)

// Events after one half-life have 50% weight
// Events after two half-lives have 25% weight
```

**Half-Life Guidelines:**

- **1s-10s**: Very responsive, quick decay
- **30s-60s**: Balanced (default: 60s)
- **5m-15m**: Smooth, slow decay

### Recording Frequencies

No ticker needed - decay is automatic:

```go
r := countmin.NewRate(60 * time.Second)

// Track frequency counts
r.Add(10.0)  // 10 requests
r.Add(5.0)   // 5 requests

// Current rate with automatic decay
rate := r.Rate()
fmt.Printf("Frequency rate: %.2f/sec\n", rate)
```

### With Timestamps

For testing or historical data:

```go
r := countmin.NewRate(60 * time.Second)
now := time.Now()

// Add frequency counts at specific times
r.AddAt(100.0, now)
r.AddAt(50.0, now.Add(30*time.Second))
r.AddAt(80.0, now.Add(60*time.Second))

// Query rate at specific time
rate := r.RateAt(now.Add(90 * time.Second))
```

### Decay Example

```go
r := countmin.NewRate(60 * time.Second)
r.Add(100.0) // 100 requests

// Immediately: rate = 100
fmt.Printf("Now: %.2f\n", r.Rate())

// After 60 seconds (one half-life): rate ≈ 50
time.Sleep(60 * time.Second)
fmt.Printf("After 60s: %.2f\n", r.Rate())

// After 120 seconds (two half-lives): rate ≈ 25
time.Sleep(60 * time.Second)
fmt.Printf("After 120s: %.2f\n", r.Rate())
```

### Rate Implementation Examples

#### Endpoint Frequency Rate Monitoring

```go
var endpointRates = make(map[string]*countmin.Rate)
var endpointSketch = countmin.New(0.01, 0.01)

func trackEndpoint(endpoint string) {
    // Update sketch
    endpointSketch.UpdateString(endpoint)

    // Get current frequency
    freq := endpointSketch.CountString(endpoint)

    // Update rate tracker
    if endpointRates[endpoint] == nil {
        endpointRates[endpoint] = countmin.NewRate(60 * time.Second)
    }
    endpointRates[endpoint].Add(float64(freq))

    // Get current rate
    if rand.Float64() < 0.01 {
        rate := endpointRates[endpoint].Rate()
        log.Printf("Endpoint %s rate: %.2f req/sec", endpoint, rate)
    }
}
```

#### Traffic Spike Detection

```go
type SpikeDetector struct {
    sketch    *countmin.Sketch
    rate      *countmin.Rate
    threshold float64
}

func NewSpikeDetector(threshold float64) *SpikeDetector {
    return &SpikeDetector{
        sketch:    countmin.New(0.01, 0.01),
        rate:      countmin.NewRate(10 * time.Second),
        threshold: threshold,
    }
}

func (sd *SpikeDetector) AddRequest(item string) bool {
    oldFreq := sd.sketch.CountString(item)
    sd.sketch.UpdateString(item)
    newFreq := sd.sketch.CountString(item)

    if newFreq > oldFreq {
        sd.rate.Add(float64(newFreq - oldFreq))
    }

    // Check if we're in a spike
    if sd.rate.Rate() > sd.threshold {
        return true // Spike detected
    }
    return false
}
```

#### Adaptive Frequency Tracking

```go
type AdaptiveTracker struct {
    sketch *countmin.Sketch
    rate   *countmin.Rate
}

func NewAdaptiveTracker() *AdaptiveTracker {
    return &AdaptiveTracker{
        sketch: countmin.New(0.01, 0.01),
        rate:   countmin.NewRate(30 * time.Second),
    }
}

func (at *AdaptiveTracker) Track(item string) {
    at.sketch.UpdateString(item)
    freq := at.sketch.CountString(item)
    at.rate.Add(float64(freq))
}

func (at *AdaptiveTracker) GetFrequency(item string) uint64 {
    return at.sketch.CountString(item)
}

func (at *AdaptiveTracker) GetRate() float64 {
    return at.rate.Rate()
}
```

## Use Cases

### Request Frequency Tracking

Track API endpoint frequencies.

```go
cm := countmin.New(0.01, 0.01) // 1% error

func handleRequest(endpoint string) {
    cm.UpdateString(endpoint)

    // Check frequency
    count := cm.CountString(endpoint)
    if count > 1000 {
        log.Printf("High traffic on %s: %d requests", endpoint, count)
    }
}
```

### Heavy Hitter Detection

Find top frequent items.

```go
cm := countmin.New(0.001, 0.01)

func processLog(ip string) {
    cm.UpdateString(ip)

    // Check if heavy hitter (> 1% of total traffic)
    count := cm.CountString(ip)
    if count > cm.Total()/100 {
        log.Printf("Heavy hitter: %s (%d requests)", ip, count)
    }
}
```

### Rate Limiting

Track request rates per user.

```go
type RateLimiter struct {
    sketch *countmin.Sketch
    limit  uint64
}

func NewRateLimiter(limit uint64) *RateLimiter {
    return &RateLimiter{
        sketch: countmin.New(0.01, 0.01),
        limit:  limit,
    }
}

func (rl *RateLimiter) Allow(userID string) bool {
    count := rl.sketch.CountString(userID)

    if count >= rl.limit {
        return false // Rate limit exceeded
    }

    rl.sketch.UpdateString(userID)
    return true
}
```

### Distributed Counting

Aggregate counts across multiple servers.

```go
// Server 1
cm1 := countmin.New(0.01, 0.01)
for _, event := range localEvents {
    cm1.UpdateString(event.Type)
}
data1, _ := cm1.Export()

// Server 2
cm2 := countmin.New(0.01, 0.01)
for _, event := range localEvents {
    cm2.UpdateString(event.Type)
}
data2, _ := cm2.Export()

// Aggregator
cm1Imported, _ := countmin.Import(data1)
cm2Imported, _ := countmin.Import(data2)

cm1Imported.Merge(cm2Imported)
globalCount := cm1Imported.CountString("click")
```

### Traffic Analysis

Monitor network traffic patterns.

```go
cm := countmin.New(0.001, 0.01)

func analyzePacket(srcIP, dstIP string) {
    flow := srcIP + "->" + dstIP
    cm.UpdateString(flow)

    // Detect traffic anomalies
    count := cm.CountString(flow)
    avgTraffic := cm.Total() / uint64(estimatedFlows)

    if count > avgTraffic*10 {
        log.Printf("Anomaly detected: %s (%d packets)", flow, count)
    }
}
```

## Performance Characteristics

### Time Complexity

| Operation | Complexity | Description |
|-----------|------------|-------------|
| `New` | O(width × depth) | Allocate matrix |
| `Add` | O(depth) | Update depth counters |
| `Count` | O(depth) | Query depth counters |
| `Clear` | O(width × depth) | Reset all counters |
| `Merge` | O(width × depth) | Add all counters |
| `Clone` | O(width × depth) | Copy all counters |

All update and query operations are **O(depth)**, typically 3-7 operations.

### Space Complexity

**Memory usage**: width × depth × 8 bytes (uint64)

**Formula**: (e/ε) × ln(1/δ) × 8 bytes

**Examples:**

| Epsilon | Delta | Width | Depth | Memory |
|---------|-------|-------|-------|--------|
| 0.001 | 0.01 | 2719 | 5 | ~106 KB |
| 0.01 | 0.01 | 272 | 5 | ~11 KB |
| 0.01 | 0.1 | 272 | 3 | ~6 KB |

### Benchmarks (Apple M3 Pro)

| Operation | Time | Allocations |
|-----------|------|-------------|
| Add | ~50 ns | 0 allocs |
| AddString | ~55 ns | 0 allocs |
| Count | ~45 ns | 0 allocs |
| Clone | ~1.2 µs | 1 alloc |
| Concurrent Add | ~180 ns | 0 allocs |
| Concurrent Count | ~160 ns | 0 allocs |
| Export | ~3.5 µs | 21 allocs |
| Import | ~12 µs | 155 allocs |

**Performance Notes:**

- All operations are allocation-free except Clone and Export/Import
- Add and Count operations are ~50ns (very fast)
- Thread-safe concurrent operations: ~160-180ns
- Suitable for high-frequency updates (millions/sec)

## Accuracy

### Error Bounds

**Guarantee**: count(x) ≤ true_count(x) + ε × N (with probability 1-δ)

**Examples:**

| Total (N) | Epsilon | Max Error (ε × N) | Example Count |
|-----------|---------|-------------------|---------------|
| 1,000 | 0.01 | 10 | 50 ± 10 |
| 10,000 | 0.01 | 100 | 500 ± 100 |
| 100,000 | 0.001 | 100 | 5000 ± 100 |
| 1,000,000 | 0.001 | 1000 | 50000 ± 1000 |

### Real-World Accuracy

With uniform distribution (ε=0.001, δ=0.01):

| True Count | Estimated Count | Error | Error % |
|------------|-----------------|-------|---------|
| 10 | 10-12 | 0-2 | 0-20% |
| 100 | 100-105 | 0-5 | 0-5% |
| 1,000 | 1,000-1,020 | 0-20 | 0-2% |
| 10,000 | 10,000-10,100 | 0-100 | 0-1% |

**Note**: Actual error is typically much smaller than the bound ε × N.

## Limitations

### Approximate Counts

Count-min sketch provides estimates, not exact counts.

```go
cm := countmin.New(0.01, 0.01)

// Add exactly 1000 items
for i := 0; i < 1000; i++ {
    cm.UpdateString("item")
}

count := cm.CountString("item")
// Might return: 1005 or 1010 (slightly higher)
```

**Mitigation**: Use smaller epsilon for more accuracy (more memory).

### No Element Removal

Cannot remove or decrement counts.

```go
cm.UpdateString("item")
// No cm.Remove("item") - not supported
```

**Workaround**: Create new sketch or use different data structure.

### Fixed Dimensions

Cannot change dimensions after creation.

```go
cm := countmin.New(0.01, 0.01)
// Cannot resize - must create new sketch
```

### Merge Requires Same Dimensions

```go
cm1 := countmin.NewWithSize(100, 5)
cm2 := countmin.NewWithSize(200, 5)

err := cm1.Merge(cm2)
// Error: dimension mismatch
```

## Best Practices

### Choose Appropriate Error Bounds

```go
// For high accuracy (< 0.1% error)
cm := countmin.New(0.001, 0.01) // ~106KB

// For general use (~ 1% error) - RECOMMENDED
cm := countmin.New(0.01, 0.01) // ~11KB

// For low memory (~ 1% error, higher failure rate)
cm := countmin.New(0.01, 0.1) // ~6KB
```

### Use Bytes for Performance

```go
// Slower: string wrapper
cm.UpdateString(string(data))

// Faster: direct bytes
cm.Update(data)
```

### Periodic Reset for Time Windows

```go
// Reset every hour for hourly statistics
go func() {
    ticker := time.NewTicker(1 * time.Hour)
    for range ticker.C {
        cm.Clear()
    }
}()
```

### Distributed Aggregation

```go
// Each worker maintains local sketch
func worker(id int, events <-chan Event) {
    cm := countmin.New(0.01, 0.01)

    for event := range events {
        cm.UpdateString(event.ID)
    }

    // Send to aggregator
    data, _ := cm.Export()
    sendToAggregator(data)
}

// Aggregator merges all sketches
func aggregator(dataStream <-chan []byte) {
    cm := countmin.New(0.01, 0.01)

    for data := range dataStream {
        workerSketch, _ := countmin.Import(data)
        cm.Merge(workerSketch)
    }

    fmt.Printf("Global frequency: %d\n", cm.Total())
}
```

## Comparison with Other Approaches

| Approach | Memory (1M items) | Accuracy | Query Time | Updates |
|----------|-------------------|----------|------------|---------|
| map[string]int | ~40-80 MB | Exact | O(1) | Slow delete |
| **Count-min** | **~11 KB** | **~1% error** | **O(1)** | **Fast** |
| Count sketch | ~11 KB | ~1% error | O(1) | Allows delete |
| Bloom filter | ~1.4 MB | Membership only | O(1) | No counts |

**Count-min Advantages:**

- **3600x less memory** than exact counting with map
- **Fast operations** (constant time)
- **Mergeable** for distributed systems
- **Predictable error** bounds
- **Simple implementation**

**When to Use:**

- Frequency estimation in streams
- Heavy hitter detection
- Rate limiting with bounded memory
- Traffic analysis
- Distributed aggregation
- Top-k queries (with heap)

**When NOT to Use:**

- Need exact counts (use map)
- Need to remove elements (use Count sketch)
- Only need membership (use Bloom filter)
- Very small datasets (use map)

## License

This project is part of the [gokit](https://github.com/vitalvas/gokit) library.
