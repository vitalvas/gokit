# spacesaving

A high-performance Space-Saving algorithm implementation in Go for finding top-k most frequent items (heavy hitters) in data streams.

## Features

- **Top-k tracking**: Find the most frequent items in bounded memory
- **Guaranteed accuracy**: Error bounds on frequency estimates
- **Constant memory**: O(k) space regardless of stream size
- **Thread-safe**: Safe for concurrent use with RWMutex
- **Heavy hitters detection**: Identify popular items, trending topics, frequent queries
- **Streaming friendly**: Process unlimited data with fixed memory
- **Export/Import**: Serialize for storage or network transmission
- **Zero dependencies**: Only uses Go standard library

## What is Space-Saving?

Space-Saving is a streaming algorithm for finding the top-k most frequent items in a data stream using bounded memory. It maintains exactly k counters and provides approximate frequency counts with guaranteed error bounds.

**Key Properties:**

- **Bounded memory**: Always uses exactly k counters (constant memory)
- **Guaranteed accuracy**: Count errors are bounded and known
- **No false negatives**: All top-k items are always tracked (may include some non-top-k)
- **Deterministic**: Same stream produces same results

**Use Cases:** Network traffic analysis, trending topics detection, popular products tracking, frequent query monitoring, clickstream analysis, log analysis.

## Installation

```bash
go get github.com/vitalvas/gokit/spacesaving
```

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/vitalvas/gokit/spacesaving"
)

func main() {
    // Create Space-Saving tracker for top 10 items
    ss := spacesaving.New(10)

    // Add items from stream
    items := []string{"apple", "banana", "apple", "cherry", "apple", "banana", "apple"}
    for _, item := range items {
        ss.Add(item)
    }

    // Get top 5 items
    top := ss.Top(5)
    for i, item := range top {
        fmt.Printf("%d. %s: %d occurrences\n", i+1, item.Value, item.Count)
    }

    // Check specific item
    count, err := ss.Count("apple")
    fmt.Printf("Apple count: %d (error bound: %d)\n", count, err)
}
```

## Creating a Space-Saving Tracker

### New

Create a new Space-Saving tracker with specified capacity.

```go
// Track top 100 items
ss := spacesaving.New(100)

// Track top 1000 items (more memory, more items tracked)
ss := spacesaving.New(1000)

// Track top 10 items (less memory, fewer items tracked)
ss := spacesaving.New(10)
```

**Capacity Guidelines:**

- **10-100**: Small streams, trending topics
- **100-1000**: Medium streams, popular products
- **1000-10000**: Large streams, network traffic analysis
- **10000+**: Very large streams, comprehensive analysis

| Capacity | Memory | Use Case |
|----------|--------|----------|
| 10 | ~1 KB | Top 10 trending |
| 100 | ~10 KB | Top 100 products |
| 1000 | ~100 KB | Network heavy hitters |
| 10000 | ~1 MB | Comprehensive analysis |

## Adding Items

### Add

Record an occurrence of an item.

```go
ss := spacesaving.New(10)

// Add items
count := ss.Add("apple")
fmt.Printf("Apple has appeared %d times\n", count)

// Add from stream
for _, item := range stream {
    ss.Add(item)
}
```

**Returns:** Approximate count after addition

## Querying Frequencies

### Count

Get the approximate frequency count for an item.

```go
ss := spacesaving.New(100)

// Add items
for i := 0; i < 1000; i++ {
    ss.Add("popular")
}

// Query count
count, err := ss.Count("popular")
fmt.Printf("Count: %d (±%d)\n", count, err)
```

**Returns:**
- `count`: Approximate frequency (may be overestimated)
- `err`: Maximum overestimation (actual count ≥ count - err)

**Error Bounds:**
- Tracked items: Error bound based on eviction history
- Non-tracked items: Error bound is count of minimum tracked item

### Top

Get the top n most frequent items.

```go
ss := spacesaving.New(100)

// Add items...
for _, item := range stream {
    ss.Add(item)
}

// Get top 10
top := ss.Top(10)
for i, item := range top {
    fmt.Printf("%d. %s: %d (±%d)\n",
        i+1, item.Value, item.Count, item.Error)
}
```

**Returns:** Slice of items sorted by frequency (descending)

### All

Get all tracked items sorted by frequency.

```go
ss := spacesaving.New(100)

// Add items...

// Get all tracked items
all := ss.All()
fmt.Printf("Tracking %d items\n", len(all))

for _, item := range all {
    fmt.Printf("%s: %d\n", item.Value, item.Count)
}
```

## Utility Operations

### Size

Get the number of items currently tracked.

```go
ss := spacesaving.New(100)
ss.Add("a")
ss.Add("b")

fmt.Printf("Tracking %d items\n", ss.Size())
// Output: Tracking 2 items
```

### Capacity

Get the maximum number of items that can be tracked.

```go
ss := spacesaving.New(100)
fmt.Printf("Capacity: %d\n", ss.Capacity())
// Output: Capacity: 100
```

### Reset

Clear all tracked items.

```go
ss := spacesaving.New(100)
ss.Add("a")
ss.Add("b")

ss.Reset()

fmt.Printf("Size after reset: %d\n", ss.Size())
// Output: Size after reset: 0
```

## Serialization

### Export

Serialize the tracker for storage or transmission.

```go
ss := spacesaving.New(100)
for i := 0; i < 1000; i++ {
    ss.Add(fmt.Sprintf("item-%d", i%10))
}

// Export
data, err := ss.Export()
if err != nil {
    log.Fatal(err)
}

// Save to file
os.WriteFile("spacesaving.dat", data, 0644)
```

### Import

Deserialize a tracker from exported data.

```go
// Load from file
data, err := os.ReadFile("spacesaving.dat")
if err != nil {
    log.Fatal(err)
}

// Import
ss, err := spacesaving.Import(data)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Loaded tracker with %d items\n", ss.Size())
```

## Use Cases

### Trending Topics Detection

Track trending topics on social media.

```go
trending := spacesaving.New(100)

func recordHashtag(hashtag string) {
    trending.Add(hashtag)
}

func getTrendingTopics(n int) []string {
    top := trending.Top(n)
    topics := make([]string, len(top))
    for i, item := range top {
        topics[i] = item.Value
    }
    return topics
}

func displayTrending() {
    topics := getTrendingTopics(10)
    fmt.Println("Trending Now:")
    for i, topic := range topics {
        fmt.Printf("%d. %s\n", i+1, topic)
    }
}
```

### Network Traffic Analysis

Find heavy hitters in network traffic.

```go
type TrafficMonitor struct {
    sources *spacesaving.SpaceSaving
    dests   *spacesaving.SpaceSaving
}

func NewTrafficMonitor() *TrafficMonitor {
    return &TrafficMonitor{
        sources: spacesaving.New(1000),
        dests:   spacesaving.New(1000),
    }
}

func (tm *TrafficMonitor) RecordPacket(srcIP, dstIP string) {
    tm.sources.Add(srcIP)
    tm.dests.Add(dstIP)
}

func (tm *TrafficMonitor) GetTopSources(n int) []spacesaving.Item {
    return tm.sources.Top(n)
}

func (tm *TrafficMonitor) GetTopDestinations(n int) []spacesaving.Item {
    return tm.dests.Top(n)
}

func (tm *TrafficMonitor) IsHeavyHitter(ip string, threshold uint64) bool {
    count, err := tm.sources.Count(ip)
    // Account for error bound
    return count-err >= threshold
}
```

### Popular Products Tracking

Track popular products in e-commerce.

```go
type ProductTracker struct {
    views     *spacesaving.SpaceSaving
    purchases *spacesaving.SpaceSaving
}

func NewProductTracker() *ProductTracker {
    return &ProductTracker{
        views:     spacesaving.New(500),
        purchases: spacesaving.New(100),
    }
}

func (pt *ProductTracker) RecordView(productID string) {
    pt.views.Add(productID)
}

func (pt *ProductTracker) RecordPurchase(productID string) {
    pt.purchases.Add(productID)
}

func (pt *ProductTracker) GetTrendingProducts(n int) []spacesaving.Item {
    return pt.views.Top(n)
}

func (pt *ProductTracker) GetBestSellers(n int) []spacesaving.Item {
    return pt.purchases.Top(n)
}

func (pt *ProductTracker) GetProductMetrics(productID string) map[string]uint64 {
    views, viewsErr := pt.views.Count(productID)
    purchases, purchasesErr := pt.purchases.Count(productID)

    return map[string]uint64{
        "views":         views,
        "views_err":     viewsErr,
        "purchases":     purchases,
        "purchases_err": purchasesErr,
    }
}
```

### Database Query Monitoring

Track frequent database queries.

```go
type QueryMonitor struct {
    queries *spacesaving.SpaceSaving
    mu      sync.Mutex
}

func NewQueryMonitor() *QueryMonitor {
    return &QueryMonitor{
        queries: spacesaving.New(200),
    }
}

func (qm *QueryMonitor) RecordQuery(query string) {
    // Normalize query (remove parameters)
    normalized := normalizeQuery(query)
    qm.queries.Add(normalized)
}

func (qm *QueryMonitor) GetFrequentQueries(n int) []spacesaving.Item {
    return qm.queries.Top(n)
}

func (qm *QueryMonitor) ShouldOptimize(query string, threshold uint64) bool {
    normalized := normalizeQuery(query)
    count, err := qm.queries.Count(normalized)

    // Conservative estimate (account for error)
    return count-err >= threshold
}

func normalizeQuery(query string) string {
    // Remove parameter values, keep structure
    // Example: "SELECT * FROM users WHERE id = 123"
    //       -> "SELECT * FROM users WHERE id = ?"
    // Implementation details omitted
    return query
}
```

### Clickstream Analysis

Analyze user clickstream patterns.

```go
type ClickstreamAnalyzer struct {
    pages   *spacesaving.SpaceSaving
    domains *spacesaving.SpaceSaving
}

func NewClickstreamAnalyzer() *ClickstreamAnalyzer {
    return &ClickstreamAnalyzer{
        pages:   spacesaving.New(1000),
        domains: spacesaving.New(100),
    }
}

func (ca *ClickstreamAnalyzer) RecordClick(url string) {
    ca.pages.Add(url)

    domain := extractDomain(url)
    ca.domains.Add(domain)
}

func (ca *ClickstreamAnalyzer) GetPopularPages(n int) []spacesaving.Item {
    return ca.pages.Top(n)
}

func (ca *ClickstreamAnalyzer) GetPopularDomains(n int) []spacesaving.Item {
    return ca.domains.Top(n)
}

func (ca *ClickstreamAnalyzer) GetStats() map[string]interface{} {
    return map[string]interface{}{
        "tracked_pages":   ca.pages.Size(),
        "tracked_domains": ca.domains.Size(),
        "top_page":        ca.pages.Top(1)[0].Value,
    }
}

func extractDomain(url string) string {
    // Extract domain from URL
    // Implementation details omitted
    return ""
}
```

### DNS Query Monitoring

Track DNS query rates from network interface using pcap.

```go
package main

import (
    "fmt"
    "time"

    "github.com/google/gopacket"
    "github.com/google/gopacket/layers"
    "github.com/google/gopacket/pcap"
    "github.com/vitalvas/gokit/spacesaving"
)

func main() {
    // Open interface
    handle, _ := pcap.OpenLive("eth0", 1600, true, pcap.BlockForever)
    defer handle.Close()
    handle.SetBPFFilter("udp port 53")

    // Create rate tracker with 60 second half-life
    tracker := spacesaving.NewRateTracker(50, 60*time.Second)

    // Capture DNS queries
    go func() {
        packets := gopacket.NewPacketSource(handle, handle.LinkType())
        for packet := range packets.Packets() {
            if dnsLayer := packet.Layer(layers.LayerTypeDNS); dnsLayer != nil {
                dns, _ := dnsLayer.(*layers.DNS)
                if !dns.QR { // Query only
                    for _, q := range dns.Questions {
                        tracker.Touch(string(q.Name))
                    }
                }
            }
        }
    }()

    // Display top 10 every 5 seconds
    ticker := time.NewTicker(5 * time.Second)
    for range ticker.C {
        fmt.Println("Top DNS Query Rates:")
        for i, item := range tracker.Top(10) {
            fmt.Printf("%d. %s: %.2f queries/sec\n", i+1, item.Value, item.Rate)
        }
    }
}
```

### Time-Windowed Tracking

Track top items over time windows.

```go
type WindowedTracker struct {
    current  *spacesaving.SpaceSaving
    previous *spacesaving.SpaceSaving
    window   time.Duration
    lastReset time.Time
    mu       sync.Mutex
}

func NewWindowedTracker(capacity int, window time.Duration) *WindowedTracker {
    return &WindowedTracker{
        current:   spacesaving.New(capacity),
        previous:  spacesaving.New(capacity),
        window:    window,
        lastReset: time.Now(),
    }
}

func (wt *WindowedTracker) Add(item string) {
    wt.mu.Lock()
    defer wt.mu.Unlock()

    // Check if window expired
    if time.Since(wt.lastReset) > wt.window {
        wt.previous = wt.current
        wt.current = spacesaving.New(wt.current.Capacity())
        wt.lastReset = time.Now()
    }

    wt.current.Add(item)
}

func (wt *WindowedTracker) GetCurrentTop(n int) []spacesaving.Item {
    wt.mu.Lock()
    defer wt.mu.Unlock()
    return wt.current.Top(n)
}

func (wt *WindowedTracker) GetPreviousTop(n int) []spacesaving.Item {
    wt.mu.Lock()
    defer wt.mu.Unlock()
    return wt.previous.Top(n)
}
```

## Performance Characteristics

### Time Complexity

| Operation | Average | Worst | Description |
|-----------|---------|-------|-------------|
| `Add` | O(log k) | O(log k) | Heap operation |
| `Count` | O(1) | O(1) | Map lookup |
| `Top` | O(k log k) | O(k log k) | Sort tracked items |
| `All` | O(k log k) | O(k log k) | Sort all items |
| `Size` | O(1) | O(1) | Return counter |
| `Reset` | O(1) | O(1) | Clear map |

Where k = capacity (number of counters)

### Space Complexity

**Memory usage:** O(k) where k is capacity

**Per-item overhead:** ~80 bytes (counter struct + heap entry + map entry)

**Total memory:** capacity × 80 bytes

| Capacity | Memory | Items Tracked |
|----------|--------|---------------|
| 10 | ~1 KB | 10 |
| 100 | ~8 KB | 100 |
| 1000 | ~80 KB | 1000 |
| 10000 | ~800 KB | 10000 |

### Benchmarks (Apple M3 Pro)

| Operation | Time | Allocations |
|-----------|------|-------------|
| Add | ~320 ns | 0 allocs |
| Count | ~25 ns | 0 allocs |
| Top(10) | ~8.5 µs | 1 alloc |
| Export | ~110 µs | 40 allocs |
| Import | ~105 µs | 200 allocs |
| Concurrent Add | ~850 ns | 0 allocs |

**Performance Notes:**

- Add operation includes heap maintenance
- Count is fast map lookup (no allocations)
- Top requires sorting (one allocation for slice)
- Concurrent operations ~3x slower due to locking
- Suitable for high-frequency streams (millions/sec)

## Algorithm Details

### How Space-Saving Works

1. **Initialization**: Create k counters (all empty)
2. **Item Arrival**:
   - If item already tracked: Increment its counter
   - If space available: Create new counter with count = 1
   - If no space: Replace minimum counter, inherit its count + 1
3. **Error Tracking**: Each counter tracks maximum overestimation

### Error Guarantees

**Theorem:** For any item with true frequency f:

- If tracked: estimated count ≤ f + error
- If not tracked: true count < (smallest tracked count)

**Example:**

```go
ss := spacesaving.New(2)

// Stream: A, B, C, C, C
ss.Add("A")  // A:1
ss.Add("B")  // A:1, B:1
ss.Add("C")  // A evicted, C:2 (inherited A's count + 1)
ss.Add("C")  // C:3
ss.Add("C")  // C:4

// Query results:
count, err := ss.Count("C")
// count=4, err=1 (actual: 4, estimate: 4, error from inheriting A's count)
```

## Accuracy

### Frequency Estimation Accuracy

**Guaranteed Properties:**

1. **No false negatives**: All true top-k items are tracked
2. **Bounded error**: Overestimation is limited and known
3. **Monotonicity**: Counts only increase (never decrease)

### Comparison with True Counts

**Test scenario:** 10K items, Zipf distribution, capacity=100

| Rank | True Count | Estimated | Error | Relative Error |
|------|-----------|-----------|-------|----------------|
| 1 | 1000 | 1000 | 0 | 0.0% |
| 10 | 100 | 100 | 0 | 0.0% |
| 50 | 20 | 20-22 | 0-2 | 0-10% |
| 100 | 10 | 10-12 | 0-2 | 0-20% |

**Accuracy characteristics:**

- **Heavy hitters** (top items): Near-perfect accuracy
- **Medium frequency**: Small overestimation
- **Rare items**: May not be tracked

## Thread Safety

All operations are thread-safe and protected by RWMutex:

```go
ss := spacesaving.New(100)

// Safe to call from multiple goroutines
go func() {
    for item := range stream1 {
        ss.Add(item)
    }
}()

go func() {
    for item := range stream2 {
        ss.Add(item)
    }
}()

go func() {
    ticker := time.NewTicker(time.Second)
    for range ticker.C {
        top := ss.Top(10)
        displayTopItems(top)
    }
}()
```

**Locking:**

- Read operations (Count, Top, All): Use RLock (concurrent readers allowed)
- Write operations (Add, Reset): Use Lock (exclusive access)
- No deadlocks: No nested locks

## Best Practices

### Choose Appropriate Capacity

```go
// Small capacity: Less memory, tracks fewer items
ss := spacesaving.New(10)

// Medium capacity: Balanced (recommended for most cases)
ss := spacesaving.New(100)

// Large capacity: More memory, tracks more items
ss := spacesaving.New(1000)
```

**Guidelines:**

- Start with capacity = 2 × expected top-k
- Increase if you need more accurate counts for lower-ranked items
- Decrease if memory is constrained

### Monitor Error Bounds

```go
// Check error bound when accuracy matters
count, err := ss.Count(item)
if err > threshold {
    log.Printf("High uncertainty: count=%d ±%d", count, err)
}

// Use conservative estimates
minCount := count - err
if minCount >= threshold {
    // Definitely frequent
}
```

### Combine with Exact Counting

```go
// Track exact counts for true top-k, use Space-Saving for candidates
type HybridTracker struct {
    candidates *spacesaving.SpaceSaving
    exact      map[string]uint64
    threshold  uint64
}

func (ht *HybridTracker) Add(item string) {
    ht.candidates.Add(item)

    count, _ := ht.candidates.Count(item)
    if count >= ht.threshold {
        // Promote to exact tracking
        if _, exists := ht.exact[item]; !exists {
            ht.exact[item] = 0
        }
        ht.exact[item]++
    }
}
```

### Use Time Windows for Trending

```go
// Reset periodically to detect trending (not just all-time popular)
ticker := time.NewTicker(1 * time.Hour)
go func() {
    for range ticker.C {
        ss.Reset()
    }
}()
```

## RateTracker: Space-Saving with Exponential Decay

The `RateTracker` is a variant of Space-Saving that tracks **rates** (events per second) instead of raw counts, using exponential decay to give more weight to recent events.

### When to Use RateTracker

Use `RateTracker` instead of basic `SpaceSaving` when:

- ✅ Recent events matter more than old events
- ✅ You need to track events/sec, requests/sec, bytes/sec
- ✅ You want trending items (not all-time popular)
- ✅ Burst detection and adaptive monitoring

Use basic `SpaceSaving` when:

- ✅ You need all-time frequency counts
- ✅ Every event has equal weight regardless of time

### Creating a RateTracker

```go
// Track top 100 items with 60-second half-life
rt := spacesaving.NewRateTracker(100, 60*time.Second)

// Short half-life for real-time trending (10 seconds)
rt := spacesaving.NewRateTracker(100, 10*time.Second)

// Long half-life for stable rates (5 minutes)
rt := spacesaving.NewRateTracker(100, 5*time.Minute)
```

**Half-Life Parameter:**

The half-life determines how quickly old events decay. After one half-life period, an event's contribution is reduced to 50%.

| Half-Life | Use Case | Decay Speed |
|-----------|----------|-------------|
| 10s | Real-time trending | Very fast |
| 60s | Request rate monitoring | Fast |
| 5m | Network traffic analysis | Medium |
| 15m | Long-term rate tracking | Slow |

### Recording Events

```go
rt := spacesaving.NewRateTracker(100, 60*time.Second)

// Record event at current time
rate := rt.Touch("user-123")
fmt.Printf("Current rate: %.2f events/sec\n", rate)

// Record event at specific time (for historical data)
rate = rt.TouchAt("user-123", specificTime)
```

### Querying Rates

```go
rt := spacesaving.NewRateTracker(100, 60*time.Second)

// Get current rate for an item
rate, errorRate := rt.Rate("user-123")
fmt.Printf("Rate: %.2f ±%.2f events/sec\n", rate, errorRate)

// Get rate at specific time
rate, errorRate = rt.RateAt("user-123", specificTime)

// Get top 10 items by rate
top := rt.Top(10)
for i, item := range top {
    fmt.Printf("%d. %s: %.2f events/sec\n", i+1, item.Value, item.Rate)
}

// Get all tracked items
all := rt.All()
```

### Exponential Decay Explained

With exponential decay, the contribution of an event decreases over time:

```go
rt := spacesaving.NewRateTracker(100, 60*time.Second)
now := time.Now()

// Touch item at t=0
rt.TouchAt("item", now)

// Check rate at different times:
// At t=0s:  rate ≈ 1.0
// At t=60s: rate ≈ 0.5 (one half-life)
// At t=120s: rate ≈ 0.25 (two half-lives)
// At t=180s: rate ≈ 0.125 (three half-lives)
```

Formula: `rate(t) = rate(0) * exp(-ln(2) * t / halfLife)`

### RateTracker Use Cases

#### Real-Time Trending Topics

```go
type TrendingTracker struct {
    topics *spacesaving.RateTracker
}

func NewTrendingTracker() *TrendingTracker {
    return &TrendingTracker{
        // 10-second half-life for fast-moving trends
        topics: spacesaving.NewRateTracker(100, 10*time.Second),
    }
}

func (tt *TrendingTracker) RecordMention(topic string) {
    tt.topics.Touch(topic)
}

func (tt *TrendingTracker) GetTrending(n int) []spacesaving.RateItem {
    return tt.topics.Top(n)
}

// Example
tracker := NewTrendingTracker()
tracker.RecordMention("#golang")
tracker.RecordMention("#ai")
tracker.RecordMention("#golang")

trending := tracker.GetTrending(10)
// Shows topics with highest recent mention rate
```

#### Request Rate Monitoring

```go
type RequestMonitor struct {
    endpoints *spacesaving.RateTracker
}

func NewRequestMonitor() *RequestMonitor {
    return &RequestMonitor{
        // 60-second half-life for request rate
        endpoints: spacesaving.NewRateTracker(200, 60*time.Second),
    }
}

func (rm *RequestMonitor) RecordRequest(endpoint string) {
    rm.endpoints.Touch(endpoint)
}

func (rm *RequestMonitor) GetTopEndpoints(n int) []spacesaving.RateItem {
    return rm.endpoints.Top(n)
}

func (rm *RequestMonitor) IsHighRate(endpoint string, threshold float64) bool {
    rate, errorRate := rm.endpoints.Rate(endpoint)
    // Conservative estimate
    return rate-errorRate >= threshold
}

// Example
monitor := NewRequestMonitor()

// In HTTP handler
func handleRequest(w http.ResponseWriter, r *http.Request) {
    monitor.RecordRequest(r.URL.Path)

    // Check if rate limit exceeded
    if monitor.IsHighRate(r.URL.Path, 100.0) {
        http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
        return
    }

    // Handle request...
}
```

#### Network Traffic Analysis

```go
type BandwidthMonitor struct {
    sources *spacesaving.RateTracker
}

func NewBandwidthMonitor() *BandwidthMonitor {
    return &BandwidthMonitor{
        // 5-minute half-life for bandwidth tracking
        sources: spacesaving.NewRateTracker(1000, 5*time.Minute),
    }
}

func (bm *BandwidthMonitor) RecordTraffic(srcIP string, bytes int) {
    // Convert bytes to events (1 event = 1KB)
    bm.sources.TouchAt(srcIP, time.Now())
}

func (bm *BandwidthMonitor) GetTopSources(n int) []spacesaving.RateItem {
    return bm.sources.Top(n)
}

func (bm *BandwidthMonitor) DetectAnomalies(threshold float64) []string {
    anomalies := []string{}
    top := bm.sources.Top(100)

    for _, item := range top {
        if item.Rate > threshold {
            anomalies = append(anomalies, item.Value)
        }
    }

    return anomalies
}
```

#### Adaptive Load Balancing

```go
type LoadBalancer struct {
    servers map[string]*spacesaving.RateTracker
}

func NewLoadBalancer(servers []string) *LoadBalancer {
    lb := &LoadBalancer{
        servers: make(map[string]*spacesaving.RateTracker),
    }

    for _, server := range servers {
        // Short half-life for responsive load balancing
        lb.servers[server] = spacesaving.NewRateTracker(100, 30*time.Second)
    }

    return lb
}

func (lb *LoadBalancer) RecordRequest(server, clientIP string) {
    if rt, exists := lb.servers[server]; exists {
        rt.Touch(clientIP)
    }
}

func (lb *LoadBalancer) SelectServer() string {
    // Select server with lowest current rate
    minRate := math.MaxFloat64
    var selected string

    for server, rt := range lb.servers {
        all := rt.All()
        totalRate := 0.0
        for _, item := range all {
            totalRate += item.Rate
        }

        if totalRate < minRate {
            minRate = totalRate
            selected = server
        }
    }

    return selected
}
```

### Comparison: SpaceSaving vs RateTracker

| Feature | SpaceSaving | RateTracker |
|---------|-------------|-------------|
| Tracks | Frequency counts | Rates (events/sec) |
| Time weighting | Equal (all time) | Exponential decay |
| Use for | All-time popular | Recent trending |
| Memory | O(k) | O(k) |
| Performance | ~91 ns/op | ~200 ns/op |
| Best for | Historical data | Real-time monitoring |

**Example comparison:**

```go
// Scenario: Track popular items over 24 hours

// SpaceSaving: All events have equal weight
ss := spacesaving.New(100)
// Item popular 20 hours ago has same weight as item popular now

// RateTracker: Recent events weighted more
rt := spacesaving.NewRateTracker(100, 1*time.Hour)
// Item popular now has much higher weight than 20 hours ago
```

### RateTracker Serialization

```go
rt := spacesaving.NewRateTracker(100, 60*time.Second)

// Add data...
for i := 0; i < 1000; i++ {
    rt.Touch(fmt.Sprintf("item-%d", i%10))
}

// Export (rates are normalized to current time)
data, err := rt.Export()
if err != nil {
    log.Fatal(err)
}

// Save to file
os.WriteFile("rates.dat", data, 0644)

// Load from file
data, err = os.ReadFile("rates.dat")
if err != nil {
    log.Fatal(err)
}

// Import
imported, err := spacesaving.ImportRateTracker(data)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Loaded %d items\n", imported.Size())
```

### RateTracker Performance

**RateTracker Performance (Apple M3 Pro):**

| Operation | Time | Allocations |
|-----------|------|-------------|
| Touch | ~200 ns | 1 alloc |
| Rate | ~40 ns | 0 allocs |
| Top(10) | ~15 µs | 4 allocs |
| Concurrent Touch | ~450 ns | 1 alloc |
| Export | ~20 µs | 28 allocs |
| Import | ~50 µs | 399 allocs |

**Notes:**

- Touch is ~2x slower than basic SpaceSaving (due to decay calculation)
- Rate query is fast with zero allocations
- Still suitable for high-frequency streams (millions/sec)

### RateTracker Thread Safety

All RateTracker operations are thread-safe:

```go
rt := spacesaving.NewRateTracker(100, 60*time.Second)

// Safe from multiple goroutines
go func() {
    for event := range stream1 {
        rt.Touch(event)
    }
}()

go func() {
    for event := range stream2 {
        rt.Touch(event)
    }
}()

go func() {
    ticker := time.NewTicker(time.Second)
    for range ticker.C {
        top := rt.Top(10)
        displayTopItems(top)
    }
}()
```

### RateTracker Best Practices

#### Choose Appropriate Half-Life

```go
// Too short: Noisy, unstable results
rt := spacesaving.NewRateTracker(100, 1*time.Second)

// Too long: Slow to adapt, similar to basic SpaceSaving
rt := spacesaving.NewRateTracker(100, 1*time.Hour)

// Good balance for most real-time use cases
rt := spacesaving.NewRateTracker(100, 60*time.Second)
```

**Rule of thumb:** Half-life should be 3-5x your monitoring interval.

#### Account for Error Bounds

```go
rate, errorRate := rt.Rate("item")

// Conservative threshold check
if rate - errorRate >= threshold {
    // Definitely above threshold
}

// Optimistic threshold check
if rate + errorRate >= threshold {
    // Possibly above threshold
}
```

#### Periodic Reset for Discrete Windows

```go
// Reset every hour for hourly trending
ticker := time.NewTicker(1 * time.Hour)
go func() {
    for range ticker.C {
        rt.Reset()
    }
}()
```

## Comparison with Other Approaches

| Approach | Memory | Accuracy | Top-K | Stream | Counts |
|----------|--------|----------|-------|--------|--------|
| Exact Count | O(n) | Exact | ✅ | ✅ | Exact |
| Space-Saving | O(k) | Approximate | ✅ | ✅ | Bounded error |
| Count-Min Sketch | O(w×d) | Approximate | ❌ | ✅ | Overestimate |
| Lossy Counting | O(1/ε) | Approximate | ✅ | ✅ | Underestimate |

**Space-Saving Advantages:**

- **Bounded memory**: Always exactly k counters
- **Guaranteed top-k**: Never misses truly frequent items
- **Known error bounds**: Can assess result quality
- **Simple**: Easy to implement and understand

**When to Use:**

- ✅ Need to find top-k items
- ✅ Memory is constrained
- ✅ Stream is unbounded
- ✅ Can tolerate small count errors
- ❌ Need exact counts (use exact counting)
- ❌ Need arbitrary item queries (use Count-Min Sketch)

## License

This project is part of the [gokit](https://github.com/vitalvas/gokit) library.
