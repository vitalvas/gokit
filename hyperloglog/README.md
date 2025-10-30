# hyperloglog

A high-performance HyperLogLog implementation in Go for cardinality estimation with minimal memory usage.

## Features

- **Cardinality estimation**: Count distinct elements in massive datasets with minimal memory
- **Configurable precision**: Choose between memory usage and accuracy (4-18 bits)
- **Small memory footprint**: Typically 10-100x smaller than exact counting
- **Merge support**: Combine multiple HyperLogLogs with mutating or non-mutating merge
- **Clone support**: Create independent copies for safe concurrent operations
- **Export/Import**: Serialize for storage or network transmission
- **High performance**: ~83ns per add, ~7.6ns for zero-allocation bytes
- **Accurate**: ~0.81% error rate at default precision (14)
- **Rate tracking**: Track unique item rates with automatic decay
- **Zero dependencies**: Only uses Go standard library

## What is HyperLogLog?

HyperLogLog is a probabilistic data structure that estimates the cardinality (number of distinct elements) in a dataset. It provides approximate counts with:

- **Minimal memory**: Uses only 2^precision bytes (typically 16KB)
- **Predictable error**: Standard error ≈ 1.04 / sqrt(2^precision)
- **No false positives/negatives**: Error is in the magnitude, not membership

**Formula:** Standard error = 1.04 / sqrt(m) where m = 2^precision

**Use Cases**: Unique visitor counting, distinct IP tracking, database query optimization, stream processing, distributed systems.

## Installation

```bash
go get github.com/vitalvas/gokit/hyperloglog
```

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/vitalvas/gokit/hyperloglog"
)

func main() {
    // Create HyperLogLog with precision 14 (16KB memory, ~0.81% error)
    hll := hyperloglog.New(14)

    // Add elements
    hll.AddString("user-123")
    hll.AddString("user-456")
    hll.AddString("user-123") // Duplicate

    // Get cardinality estimate
    count := hll.Count()
    fmt.Printf("Distinct elements: %d\n", count)
    // Output: Distinct elements: 2
}
```

## Creating a HyperLogLog

### New

Create a new HyperLogLog with specified precision.

```go
// Precision 14: 16KB memory, ~0.81% error (recommended)
hll := hyperloglog.New(14)

fmt.Printf("Memory usage: %d bytes\n", hll.Size())
fmt.Printf("Precision: %d\n", hll.Precision())
```

**Parameters:**

- `precision`: Must be between 4 and 18 (defaults to 14 if invalid)

**Precision vs Memory vs Error:**

| Precision | Memory | Standard Error | Use Case |
|-----------|--------|----------------|----------|
| 10 | 1 KB | ~3.2% | Low accuracy, memory constrained |
| 12 | 4 KB | ~1.6% | Balanced for small datasets |
| 14 | 16 KB | ~0.81% | **Recommended** for general use |
| 16 | 64 KB | ~0.40% | High accuracy needed |
| 18 | 256 KB | ~0.20% | Very high accuracy |

**Trade-off:** Higher precision = more memory, lower error

## Adding Elements

### Add

Add raw bytes to the HyperLogLog (zero-allocation).

```go
hll := hyperloglog.New(14)

data := []byte("binary-data")
hll.Add(data)
```

**Performance:** `Add` is the fastest method (~7ns) with zero allocations.

### AddString

Add a string element to the HyperLogLog.

```go
hll := hyperloglog.New(14)

hll.AddString("user@example.com")
hll.AddString("192.168.1.1")
hll.AddString("session-abc-123")
```

**Performance:** `AddString` is convenient but slightly slower (~80ns) than `Add` due to string conversion.

## Counting Distinct Elements

### Count

Get the estimated cardinality.

```go
hll := hyperloglog.New(14)

// Add 1 million unique elements
for i := 0; i < 1_000_000; i++ {
    hll.AddString(fmt.Sprintf("element-%d", i))
}

count := hll.Count()
fmt.Printf("Estimated: %d\n", count)
// Output: Estimated: ~1,000,000 (within 0.81% error)
```

**Accuracy:**

- Empty set: Exact (returns 0)
- Small sets (< 1000): Within 5-10% typically
- Medium sets (1K-100K): Within 2-3%
- Large sets (> 100K): Within 1% at precision 14

## Merging HyperLogLogs

### Merge

Merge another HyperLogLog into this one (mutates the receiver).

```go
hll1 := hyperloglog.New(14)
hll2 := hyperloglog.New(14)

// Add different elements to each
for i := 0; i < 1000; i++ {
    hll1.AddString(fmt.Sprintf("set1-%d", i))
}
for i := 0; i < 1000; i++ {
    hll2.AddString(fmt.Sprintf("set2-%d", i))
}

// Merge hll2 into hll1 (hll1 is modified)
err := hll1.Merge(hll2)
if err != nil {
    log.Fatal(err)
}

count := hll1.Count()
// Output: ~2000 (union of both sets)
```

**Note:** The receiver (`hll1`) is modified. Use `Clone()` if you need to preserve the original.

### MergeAll

Create a new HyperLogLog by merging multiple HyperLogLogs (non-mutating).

```go
hll1 := hyperloglog.New(14)
hll2 := hyperloglog.New(14)
hll3 := hyperloglog.New(14)

// Add different elements to each
for i := 0; i < 1000; i++ {
    hll1.AddString(fmt.Sprintf("set1-%d", i))
    hll2.AddString(fmt.Sprintf("set2-%d", i))
    hll3.AddString(fmt.Sprintf("set3-%d", i))
}

// Merge all into a new HLL (originals unchanged)
merged, err := hyperloglog.MergeAll(hll1, hll2, hll3)
if err != nil {
    log.Fatal(err)
}

count := merged.Count()
// Output: ~3000 (union of all three sets)

// Original HLLs are unchanged
fmt.Println(hll1.Count()) // ~1000
fmt.Println(hll2.Count()) // ~1000
fmt.Println(hll3.Count()) // ~1000
```

**Use Cases:**

- Distributed counting across multiple servers
- Combining logs from different sources
- Time-windowed aggregations
- Non-destructive merging of datasets

## Utility Operations

### Clone

Create a deep copy of the HyperLogLog.

```go
hll1 := hyperloglog.New(14)
for i := 0; i < 1000; i++ {
    hll1.AddString(fmt.Sprintf("item-%d", i))
}

// Clone creates independent copy
hll2 := hll1.Clone()

// Modify original without affecting clone
hll1.AddString("new-element")

fmt.Println(hll1.Count()) // ~1001
fmt.Println(hll2.Count()) // ~1000 (unchanged)
```

**Use Case:** Preserve original HLL before merging or modifications.

### Clear

Reset all registers to zero.

```go
hll := hyperloglog.New(14)
hll.AddString("element")

hll.Clear()
count := hll.Count()
// Output: 0
```

### Precision

Get the precision parameter.

```go
hll := hyperloglog.New(14)
p := hll.Precision()
// Output: 14
```

### Size

Get the number of registers (2^precision).

```go
hll := hyperloglog.New(14)
size := hll.Size()
// Output: 16384 (2^14)
```

## Serialization

### Export

Serialize HyperLogLog for storage or transmission.

```go
hll := hyperloglog.New(14)
for i := 0; i < 10000; i++ {
    hll.Add(fmt.Sprintf("element-%d", i))
}

// Export to bytes
data, err := hll.Export()
if err != nil {
    log.Fatal(err)
}

// Save to file
os.WriteFile("hyperloglog.dat", data, 0644)
```

### Import

Deserialize HyperLogLog from exported data.

```go
// Load from file
data, err := os.ReadFile("hyperloglog.dat")
if err != nil {
    log.Fatal(err)
}

// Import
hll, err := hyperloglog.Import(data)
if err != nil {
    log.Fatal(err)
}

count := hll.Count()
fmt.Printf("Restored count: %d\n", count)
```

## Rate: Unique Item Rate Tracking

The `Rate` type tracks the rate at which unique items are added with exponential decay, without requiring manual ticker management.

### When to Use Rate

**Use Rate when:**

- You want to track rate of unique visitors/IPs/sessions
- Events arrive at irregular intervals
- You need automatic decay based on time
- You want to detect bursts of unique items
- You need real-time unique item rate tracking

**Difference from HyperLogLog:**

| Feature | HyperLogLog | Rate |
|---------|-------------|------|
| Purpose | Count distinct items | Track rate of unique items |
| Output | Cardinality estimate | Items per second |
| Time-based | No | Yes (automatic decay) |
| Memory | O(2^precision) | O(1) constant |
| Use case | Total unique count | Real-time rate tracking |

### Creating a Rate

```go
// Create with 60 second half-life
r := hyperloglog.NewRate(60 * time.Second)

// Events after one half-life have 50% weight
// Events after two half-lives have 25% weight
```

**Half-Life Guidelines:**

- **1s-10s**: Very responsive, quick decay
- **30s-60s**: Balanced (default: 60s)
- **5m-15m**: Smooth, slow decay

### Recording Unique Items

No ticker needed - decay is automatic:

```go
r := hyperloglog.NewRate(60 * time.Second)

// Track unique visitors
r.Add(1.0) // One unique visitor
r.Add(5.0) // Five unique visitors

// Current rate with automatic decay
rate := r.Rate()
fmt.Printf("Unique visitor rate: %.2f visitors/sec\n", rate)
```

### With Timestamps

For testing or historical data:

```go
r := hyperloglog.NewRate(60 * time.Second)
now := time.Now()

// Add unique items at specific times
r.AddAt(10.0, now)
r.AddAt(5.0, now.Add(30*time.Second))
r.AddAt(8.0, now.Add(60*time.Second))

// Query rate at specific time
rate := r.RateAt(now.Add(90 * time.Second))
```

### Decay Example

```go
r := hyperloglog.NewRate(60 * time.Second)
r.Add(100.0) // 100 unique items

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

#### Unique Visitor Rate Monitoring

```go
var uniqueVisitorRate = hyperloglog.NewRate(60 * time.Second)
var visitors = hyperloglog.New(14) // For total count

func trackVisitor(userID string) {
    visitors.AddString(userID)

    // Estimate if this is a new unique visitor
    oldCount := visitors.Count()
    testVisitors := visitors.Clone()
    testVisitors.AddString(userID)
    newCount := testVisitors.Count()

    if newCount > oldCount {
        uniqueVisitorRate.Add(1.0)
    }

    // Get current rate
    if rand.Float64() < 0.01 {
        log.Printf("Unique visitor rate: %.2f/sec", uniqueVisitorRate.Rate())
    }
}
```

#### New IP Detection Rate

```go
type IPRateTracker struct {
    seen *hyperloglog.HyperLogLog
    rate *hyperloglog.Rate
}

func NewIPRateTracker() *IPRateTracker {
    return &IPRateTracker{
        seen: hyperloglog.New(14),
        rate: hyperloglog.NewRate(60 * time.Second),
    }
}

func (ipt *IPRateTracker) TrackIP(ip string) {
    oldCount := ipt.seen.Count()
    ipt.seen.AddString(ip)
    newCount := ipt.seen.Count()

    // New unique IP detected
    if newCount > oldCount {
        ipt.rate.Add(1.0)
    }
}

func (ipt *IPRateTracker) NewIPRate() float64 {
    return ipt.rate.Rate()
}

func (ipt *IPRateTracker) TotalUniqueIPs() uint64 {
    return ipt.seen.Count()
}
```

#### Burst Detection

```go
type BurstDetector struct {
    hll       *hyperloglog.HyperLogLog
    rate      *hyperloglog.Rate
    threshold float64
}

func NewBurstDetector(threshold float64) *BurstDetector {
    return &BurstDetector{
        hll:       hyperloglog.New(14),
        rate:      hyperloglog.NewRate(10 * time.Second),
        threshold: threshold,
    }
}

func (bd *BurstDetector) AddItem(item string) bool {
    oldCount := bd.hll.Count()
    bd.hll.AddString(item)
    newCount := bd.hll.Count()

    if newCount > oldCount {
        bd.rate.Add(1.0)
    }

    // Check if we're in a burst of unique items
    if bd.rate.Rate() > bd.threshold {
        return true // Burst detected
    }
    return false
}
```

## Use Cases

### Unique Visitor Counting

Track unique visitors with minimal memory.

```go
hll := hyperloglog.New(14) // 16KB for ~1% error

func trackVisitor(userID string) {
    hll.AddString(userID)
}

func getUniqueVisitors() uint64 {
    return hll.Count()
}

// 1M unique visitors: 16KB vs 40+MB for map[string]bool
```

**Memory Savings:** 2500x less memory than exact counting

### Distinct IP Tracking

Monitor unique IP addresses in network logs.

```go
hll := hyperloglog.New(14)

func processLogLine(line string) {
    ip := extractIP(line)
    hll.AddString(ip)
}

func getDistinctIPs() uint64 {
    return hll.Count()
}
```

### Database Query Optimization

Estimate DISTINCT counts without full scans.

```go
// Instead of: SELECT COUNT(DISTINCT user_id) FROM events
// Use HyperLogLog for approximate but fast count

hll := hyperloglog.New(14)
rows, _ := db.Query("SELECT user_id FROM events")

for rows.Next() {
    var userID string
    rows.Scan(&userID)
    hll.AddString(userID)
}

distinctUsers := hll.Count()
fmt.Printf("Approximate distinct users: %d\n", distinctUsers)
```

### Distributed Counting

Aggregate counts across multiple servers.

```go
// Server 1
hll1 := hyperloglog.New(14)
for _, event := range localEvents {
    hll1.AddString(event.UserID)
}
data1, _ := hll1.Export()

// Server 2
hll2 := hyperloglog.New(14)
for _, event := range localEvents {
    hll2.AddString(event.UserID)
}
data2, _ := hll2.Export()

// Aggregator server
hll1Imported, _ := hyperloglog.Import(data1)
hll2Imported, _ := hyperloglog.Import(data2)

hll1Imported.Merge(hll2Imported)
globalDistinct := hll1Imported.Count()
```

### Stream Processing

Count distinct elements in data streams.

```go
hll := hyperloglog.New(14)

func processBatch(events []Event) {
    for _, event := range events {
        hll.AddString(event.ID)
    }

    // Periodic reporting
    if time.Now().Unix()%60 == 0 {
        fmt.Printf("Distinct events (last hour): %d\n", hll.Count())
    }
}
```

## Performance Characteristics

### Time Complexity

| Operation | Complexity | Description |
|-----------|------------|-------------|
| `New` | O(2^p) | Allocate registers |
| `Add` | O(n) | Hash n bytes and update register |
| `AddString` | O(n) | Hash n bytes and update register |
| `Count` | O(2^p) | Iterate all registers |
| `Merge` | O(2^p) | Compare all registers |
| `MergeAll` | O(k * 2^p) | Merge k HLLs |
| `Clone` | O(2^p) | Copy all registers |
| `Clear` | O(2^p) | Reset all registers |

Where p = precision, n = data length, k = number of HLLs

### Space Complexity

**Memory usage:** Exactly 2^precision bytes

**Examples:**

| Precision | Registers | Memory | Typical Cardinality |
|-----------|-----------|--------|---------------------|
| 10 | 1,024 | 1 KB | Up to 10K |
| 12 | 4,096 | 4 KB | Up to 100K |
| 14 | 16,384 | 16 KB | Up to 1M |
| 16 | 65,536 | 64 KB | Up to 10M |
| 18 | 262,144 | 256 KB | Up to 100M+ |

### Benchmarks (Apple M3 Pro)

| Operation | Time | Allocations |
|-----------|------|-------------|
| Add (bytes) | ~7.6 ns | 0 allocs |
| AddString | ~83 ns | 2 allocs |
| Count | ~448 µs | 0 allocs |
| Merge | ~13 µs | 0 allocs |
| Clone | ~1.6 µs | 1 alloc |
| MergeAll (3 HLLs) | ~181 µs | 2 allocs |
| Export | ~4.1 µs | 21 allocs |
| Import | ~14 µs | 162 allocs |

**Performance Characteristics:**

- Add (bytes) is 10x faster than AddString (zero allocations)
- Count scales with precision (O(2^p))
- Merge is very fast (~13µs for 16K registers)
- Clone is efficient (~1.6µs for 16KB copy)
- MergeAll scales linearly with number of HLLs
- Export/Import optimized with gob encoding

## Accuracy

### Error Rate Formula

Standard error = 1.04 / sqrt(2^precision)

**Examples:**

| Precision | Standard Error | 95% Confidence |
|-----------|----------------|----------------|
| 10 | ~3.2% | ±6.4% |
| 12 | ~1.6% | ±3.2% |
| 14 | ~0.81% | ±1.6% |
| 16 | ~0.40% | ±0.8% |

### Real-World Accuracy

Tested with actual data:

| True Count | Precision | Estimated | Error Rate |
|-----------|-----------|-----------|------------|
| 1,000 | 10 | ~1,020 | 2.0% |
| 10,000 | 12 | ~9,760 | 2.4% |
| 100,000 | 14 | ~99,920 | 0.08% |
| 1,000,000 | 16 | ~996,400 | 0.36% |

## Limitations

### Approximate Counts

HyperLogLog provides estimates, not exact counts.

```go
hll := hyperloglog.New(14)

// Add exactly 10,000 unique elements
for i := 0; i < 10000; i++ {
    hll.AddString(fmt.Sprintf("element-%d", i))
}

count := hll.Count()
// Might return: 9,876 or 10,123 (within ~0.81% error)
```

**Mitigation:** Use higher precision for more accuracy (more memory).

### No Element Removal

Cannot remove individual elements.

```go
hll.AddString("element")
// No hll.Remove("element") - not supported
```

**Workaround:** Use counting Bloom filter or create new HyperLogLog.

### Fixed Precision

Cannot change precision after creation.

```go
hll := hyperloglog.New(12)
// Cannot resize to precision 14
// Must create new HyperLogLog and re-add elements
```

### Merge Requires Same Precision

```go
hll1 := hyperloglog.New(14)
hll2 := hyperloglog.New(12)

err := hll1.Merge(hll2)
// Error: precision mismatch
```

## Best Practices

### Choose Appropriate Precision

```go
// For < 10K distinct elements
hll := hyperloglog.New(10) // 1KB, ~3% error

// For 10K-100K distinct elements
hll := hyperloglog.New(12) // 4KB, ~1.6% error

// For 100K-1M distinct elements (recommended default)
hll := hyperloglog.New(14) // 16KB, ~0.81% error

// For > 1M distinct elements
hll := hyperloglog.New(16) // 64KB, ~0.4% error
```

### Use Bytes for Performance

```go
// Slower: string wrapper
hll.AddString(string(data))

// Faster: direct bytes (zero-allocation)
hll.Add(data)
```

### Persistence for Long-Running Processes

```go
// Startup: try to load existing HLL
data, err := os.ReadFile("hll.dat")
var hll *hyperloglog.HyperLogLog

if err == nil {
    hll, _ = hyperloglog.Import(data)
} else {
    hll = hyperloglog.New(14)
}

// Periodic save (every hour)
go func() {
    ticker := time.NewTicker(1 * time.Hour)
    for range ticker.C {
        data, _ := hll.Export()
        os.WriteFile("hll.dat", data, 0644)
    }
}()
```

### Distributed Aggregation

```go
// Each worker maintains local HLL
func worker(id int, events <-chan Event) {
    hll := hyperloglog.New(14)

    for event := range events {
        hll.AddString(event.UserID)
    }

    // Send to aggregator
    data, _ := hll.Export()
    sendToAggregator(data)
}

// Aggregator merges all HLLs
func aggregator(dataStream <-chan []byte) {
    hll := hyperloglog.New(14)

    for data := range dataStream {
        workerHLL, _ := hyperloglog.Import(data)
        hll.Merge(workerHLL)
    }

    fmt.Printf("Global distinct: %d\n", hll.Count())
}
```

## Comparison with Other Approaches

| Approach | Memory (1M elements) | Accuracy | Add Time |
|----------|---------------------|----------|----------|
| map[string]bool | ~40-80 MB | Exact | ~100 ns |
| Bloom Filter | ~1.4 MB | Membership only | ~100 ns |
| **HyperLogLog** | **16 KB** | **~0.81% error** | **~80 ns** |
| Exact COUNT DISTINCT | Full scan | Exact | Slow |

**HyperLogLog Advantages:**

- **2500x less memory** than exact counting with map
- **Mergeable** for distributed systems
- **Predictable error** bounds
- **Fast operations** (constant time add)

## License

This project is part of the [gokit](https://github.com/vitalvas/gokit) library.
