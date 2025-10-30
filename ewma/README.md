# ewma

A high-performance EWMA (Exponentially Weighted Moving Average) implementation in Go for smoothing time series data and calculating rates.

## Features

- **Thread-safe**: Safe for concurrent use with RWMutex
- **Zero allocations**: All operations are allocation-free
- **Standard time windows**: 1-minute, 5-minute, 15-minute helpers
- **Flexible configuration**: Custom alpha and interval support
- **Rate calculation**: Events per second tracking
- **Snapshot support**: Consistent reads of multiple values
- **Lightweight**: ~8ns per operation
- **Zero dependencies**: Only uses Go standard library

## What is EWMA?

Exponentially Weighted Moving Average (EWMA) is a statistical method for smoothing time series data. It gives more weight to recent observations and exponentially less weight to older observations.

**Key Properties:**

- **Adaptive**: Quickly responds to changes
- **Memory efficient**: Constant memory usage
- **Smooth**: Reduces noise in metrics
- **Tunable**: Adjust responsiveness via alpha

**Use Cases:** Request rate monitoring, load balancing, system metrics, rate limiting, response time tracking, resource utilization.

## Installation

```bash
go get github.com/vitalvas/gokit/ewma
```

## Quick Start

```go
package main

import (
    "fmt"
    "time"
    "github.com/vitalvas/gokit/ewma"
)

func main() {
    // Create 1-minute EWMA
    e := ewma.New1MinuteEWMA()

    // Start ticker for updates
    ticker := time.NewTicker(ewma.Interval)
    go func() {
        for range ticker.C {
            e.Tick()
        }
    }()

    // Record events
    for i := 0; i < 100; i++ {
        e.Update()
        time.Sleep(10 * time.Millisecond)
    }

    // Get rate (events per second)
    rate := e.Rate()
    fmt.Printf("Current rate: %.2f events/sec\n", rate)
}
```

## Creating an EWMA

### Standard Time Windows

Use pre-configured windows for common use cases:

```go
// 1-minute window (reacts quickly)
e1 := ewma.New1MinuteEWMA()

// 5-minute window (balanced)
e5 := ewma.New5MinuteEWMA()

// 15-minute window (heavily smoothed)
e15 := ewma.New15MinuteEWMA()
```

**Time Window Characteristics:**

| Window | Alpha | Response Time | Smoothing | Use Case |
|--------|-------|---------------|-----------|----------|
| 1-min | High | Fast | Low | Short-term trends |
| 5-min | Medium | Balanced | Medium | General monitoring |
| 15-min | Low | Slow | High | Long-term trends |

### Custom Alpha

Create EWMA with custom decay rate:

```go
// High alpha (0.9): Quick response, less smoothing
eQuick := ewma.NewWithAlpha(0.9)

// Low alpha (0.1): Slow response, heavy smoothing
eSmooth := ewma.NewWithAlpha(0.1)

// Custom interval
eCustom := ewma.New(0.5, 10*time.Second)
```

**Alpha Guidelines:**

- **0.9-1.0**: Very responsive, minimal smoothing
- **0.5-0.8**: Balanced response and smoothing
- **0.1-0.4**: Heavy smoothing, slow response

## Recording Events

### Update

Record a single event:

```go
e := ewma.New1MinuteEWMA()

// Record event
e.Update()

// Or use Add(1)
e.Add(1)
```

### Add

Record multiple events:

```go
e := ewma.New1MinuteEWMA()

// Record 100 events
e.Add(100)

// Record batch
batch := uint64(1000)
e.Add(batch)
```

### UpdateWithValue

Record weighted values (for metrics like response time):

```go
e := ewma.NewWithAlpha(0.5)

// Record response times
e.UpdateWithValue(45.2)  // 45.2ms
e.UpdateWithValue(52.1)  // 52.1ms
e.UpdateWithValue(48.5)  // 48.5ms

// Get smoothed average
avg := e.Rate()
fmt.Printf("Average response time: %.2fms\n", avg)
```

## Updating the Average

### Tick

Update the moving average at regular intervals:

```go
e := ewma.New1MinuteEWMA()

// Start background ticker
ticker := time.NewTicker(ewma.Interval) // 5 seconds
defer ticker.Stop()

go func() {
    for range ticker.C {
        e.Tick()
    }
}()
```

**Important:** Call `Tick()` at regular intervals (default: every 5 seconds) for accurate rate calculation.

### Manual Ticking

For testing or custom timing:

```go
e := ewma.New(0.5, 1*time.Second)

// Add events
e.Add(100)

// Manually tick
e.Tick()

// Check rate
rate := e.Rate()
// Rate = 100 events / 1 second = 100 events/sec
```

## Querying Metrics

### Rate

Get the current rate (events per second):

```go
e := ewma.New1MinuteEWMA()

// ... record events and tick ...

rate := e.Rate()
fmt.Printf("Rate: %.2f events/sec\n", rate)
```

### Snapshot

Get a consistent view of all values:

```go
e := ewma.New1MinuteEWMA()
e.Add(100)

snapshot := e.Snapshot()
fmt.Printf("Rate: %.2f\n", snapshot.Rate)
fmt.Printf("Uncounted: %d\n", snapshot.Uncounted)
fmt.Printf("Initialized: %v\n", snapshot.Initialized)
```

**Use Case:** When you need multiple values that must be consistent with each other.

## Utility Operations

### Reset

Clear all data and reset to initial state:

```go
e := ewma.New1MinuteEWMA()
e.Add(1000)

e.Reset()

fmt.Printf("Rate: %.2f\n", e.Rate())
// Output: Rate: 0.00
```

### Set

Directly set the rate (useful for initialization):

```go
e := ewma.New1MinuteEWMA()

// Initialize with baseline
e.Set(100.0)

fmt.Printf("Rate: %.2f\n", e.Rate())
// Output: Rate: 100.00
```

## MovingAverage

Convenient wrapper for tracking multiple time windows:

```go
ma := ewma.NewMovingAverage()

// Start ticker
ticker := time.NewTicker(ewma.Interval)
go func() {
    for range ticker.C {
        ma.Tick()
    }
}()

// Record events
ma.Update()  // Records to all three windows

// Get rates
m1, m5, m15 := ma.Rates()
fmt.Printf("1m: %.2f, 5m: %.2f, 15m: %.2f\n", m1, m5, m15)

// Or individually
fmt.Printf("1-min rate: %.2f\n", ma.Rate1())
fmt.Printf("5-min rate: %.2f\n", ma.Rate5())
fmt.Printf("15-min rate: %.2f\n", ma.Rate15())
```

## Use Cases

### Request Rate Monitoring

Track HTTP request rates:

```go
var requestRate = ewma.New1MinuteEWMA()

func init() {
    // Start ticker
    ticker := time.NewTicker(ewma.Interval)
    go func() {
        for range ticker.C {
            requestRate.Tick()
        }
    }()
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
    requestRate.Update()

    // ... handle request ...

    // Log current rate
    if rand.Float64() < 0.01 { // Sample 1%
        log.Printf("Request rate: %.2f req/sec", requestRate.Rate())
    }
}
```

### Response Time Tracking

Monitor average response times:

```go
var responseTime = ewma.NewWithAlpha(0.5)

func handleRequest(w http.ResponseWriter, r *http.Request) {
    start := time.Now()
    defer func() {
        duration := time.Since(start).Milliseconds()
        responseTime.UpdateWithValue(float64(duration))
    }()

    // ... handle request ...
}

func getMetrics() map[string]float64 {
    return map[string]float64{
        "avg_response_ms": responseTime.Rate(),
    }
}
```

### Load Balancing

Choose backend based on smoothed load:

```go
type Backend struct {
    Name string
    Load *ewma.EWMA
}

func (b *Backend) RecordRequest() {
    b.Load.Update()
}

func selectBackend(backends []*Backend) *Backend {
    // Choose backend with lowest smoothed load
    minLoad := math.MaxFloat64
    var selected *Backend

    for _, b := range backends {
        load := b.Load.Rate()
        if load < minLoad {
            minLoad = load
            selected = b
        }
    }

    return selected
}
```

### Rate Limiting

Smooth rate calculation for rate limiting:

```go
type RateLimiter struct {
    rate  *ewma.EWMA
    limit float64
}

func NewRateLimiter(limit float64) *RateLimiter {
    rl := &RateLimiter{
        rate:  ewma.New1MinuteEWMA(),
        limit: limit,
    }

    // Start ticker
    ticker := time.NewTicker(ewma.Interval)
    go func() {
        for range ticker.C {
            rl.rate.Tick()
        }
    }()

    return rl
}

func (rl *RateLimiter) Allow() bool {
    currentRate := rl.rate.Rate()

    if currentRate >= rl.limit {
        return false
    }

    rl.rate.Update()
    return true
}
```

### System Metrics

Track CPU or memory usage:

```go
var cpuUsage = ewma.New5MinuteEWMA()

func collectMetrics() {
    ticker := time.NewTicker(ewma.Interval)
    for range ticker.C {
        // Get current CPU usage
        usage := getCurrentCPUUsage()
        cpuUsage.UpdateWithValue(usage)
        cpuUsage.Tick()
    }
}

func getSmoothedCPU() float64 {
    return cpuUsage.Rate()
}
```

### Multi-Window Monitoring

Track rates across multiple time windows:

```go
var requestMetrics = ewma.NewMovingAverage()

func init() {
    ticker := time.NewTicker(ewma.Interval)
    go func() {
        for range ticker.C {
            requestMetrics.Tick()
        }
    }()
}

func recordRequest() {
    requestMetrics.Update()
}

func printMetrics() {
    m1, m5, m15 := requestMetrics.Rates()
    log.Printf("Request rate - 1m: %.2f, 5m: %.2f, 15m: %.2f",
        m1, m5, m15)
}
```

## Performance Characteristics

### Time Complexity

| Operation | Complexity | Description |
|-----------|------------|-------------|
| `Add` | O(1) | Update counter |
| `Update` | O(1) | Add one event |
| `UpdateWithValue` | O(1) | Update with value |
| `Tick` | O(1) | Calculate new rate |
| `Rate` | O(1) | Read current rate |
| `Snapshot` | O(1) | Copy current state |
| `Reset` | O(1) | Clear state |

All operations are constant time.

### Space Complexity

**Memory usage:** O(1) - constant memory per EWMA

**Typical size:** ~80 bytes per EWMA instance

### Benchmarks (Apple M3 Pro)

| Operation | Time | Allocations |
|-----------|------|-------------|
| Add | **7.8 ns** | 0 allocs |
| Update | **7.7 ns** | 0 allocs |
| Tick | **8.9 ns** | 0 allocs |
| Rate | **5.5 ns** | 0 allocs |
| Snapshot | **6.2 ns** | 0 allocs |
| MovingAverage Add | **22.5 ns** | 0 allocs |
| MovingAverage Tick | **27.0 ns** | 0 allocs |
| Concurrent Add | **93.5 ns** | 0 allocs |
| Concurrent Rate | **106.0 ns** | 0 allocs |

**Performance Notes:**

- All operations are allocation-free
- Single EWMA operations: <10ns
- MovingAverage (3 EWMAs): <30ns
- Concurrent operations: ~100ns
- Suitable for high-frequency updates (millions/sec)

## Thread Safety

All EWMA operations are thread-safe:

```go
e := ewma.New1MinuteEWMA()

// Safe to call from multiple goroutines
go func() {
    for {
        e.Update()
        time.Sleep(time.Millisecond)
    }
}()

go func() {
    ticker := time.NewTicker(time.Second)
    for range ticker.C {
        rate := e.Rate()
        fmt.Printf("Rate: %.2f\n", rate)
    }
}()
```

**Locking:**

- Read operations use `RLock` (multiple readers allowed)
- Write operations use `Lock` (exclusive access)
- No lock contention for read-heavy workloads

## Best Practices

### Choose Appropriate Time Window

```go
// Short-term monitoring (reacts quickly to changes)
shortTerm := ewma.New1MinuteEWMA()

// General monitoring (balanced)
general := ewma.New5MinuteEWMA()

// Long-term trends (heavily smoothed)
longTerm := ewma.New15MinuteEWMA()
```

### Always Use Ticker

```go
// Good: Use ticker for regular updates
ticker := time.NewTicker(ewma.Interval)
go func() {
    for range ticker.C {
        myEWMA.Tick()
    }
}()

// Bad: Manual/irregular ticking
// (results in inaccurate rates)
time.Sleep(6 * time.Second)
myEWMA.Tick()
```

### Use Snapshot for Consistent Reads

```go
// Good: Single snapshot for consistency
snapshot := e.Snapshot()
if snapshot.Initialized && snapshot.Rate > threshold {
    // ... take action ...
}

// Less good: Multiple reads (may be inconsistent)
if e.Rate() > threshold {
    // Rate might change between calls
    log.Printf("Rate: %.2f", e.Rate())
}
```

### Batch Updates When Possible

```go
// Good: Batch update
e.Add(batchSize)

// Less efficient: Individual updates
for i := 0; i < batchSize; i++ {
    e.Update()
}
```

### Use MovingAverage for Multiple Windows

```go
// Good: Single MovingAverage
ma := ewma.NewMovingAverage()
ma.Update()  // Updates all three windows

// Less efficient: Separate EWMAs
e1 := ewma.New1MinuteEWMA()
e5 := ewma.New5MinuteEWMA()
e15 := ewma.New15MinuteEWMA()
e1.Update()
e5.Update()
e15.Update()
```

## Comparison with Other Approaches

| Approach | Memory | Accuracy | Latency | Thread-Safe |
|----------|--------|----------|---------|-------------|
| Simple Counter | O(1) | Exact | O(1) | Requires sync |
| Sliding Window | O(n) | Exact | O(n) | Complex |
| **EWMA** | **O(1)** | **Approximate** | **O(1)** | **✅ Built-in** |
| Ring Buffer | O(n) | Exact | O(1) amortized | Requires sync |

**EWMA Advantages:**

- **Constant memory** (unlike sliding windows)
- **Built-in thread safety** (unlike simple counters)
- **Smooth metrics** (reduces noise)
- **Fast** (~8ns per operation)
- **Simple API** (easy to use correctly)

**When to Use:**

- ✅ Real-time monitoring
- ✅ Rate limiting with smoothing
- ✅ Load balancing decisions
- ✅ Response time tracking
- ✅ Resource utilization metrics
- ❌ Need exact counts (use counter)
- ❌ Need precise percentiles (use histogram)

## License

This project is part of the [gokit](https://github.com/vitalvas/gokit) library.
