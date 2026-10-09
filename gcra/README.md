# gcra

A GCRA (Generic Cell Rate Algorithm) rate limiter in Go with per-key state, burst capacity, all-or-nothing batch requests, and retry-after reporting.

## Features

- **GCRA / leaky bucket**: Smooth rate enforcement with no bursts at window boundaries
- **Burst capacity**: Each key may make a configurable number of requests instantly, then sustain the steady rate
- **Batch requests**: `AllowN` consumes capacity for n requests atomically, all-or-nothing
- **Retry-After**: Exact duration until the next request would be allowed, ready for the `Retry-After` HTTP header
- **Export/Import**: Serialize state to survive restarts; offline time is credited on restore
- **Minimal state**: One pointer-free int64 per key, invisible to the garbage collector
- **Monotonic time**: Immune to wall-clock jumps (NTP corrections, DST)
- **Bounded key capacity**: Reclaims drained entries and denies new keys while all tracked state is active
- **Configurable cleanup**: Optional background goroutine for drained entry removal
- **Thread-safe**: All operations protected by mutex
- **O(1) operations**: Map lookup + integer comparison per operation
- **Zero dependencies**: Only uses Go standard library
- **Zero allocations for tracked keys**: Existing-key requests reuse stored state

## What is GCRA?

GCRA is the virtual-scheduling formulation of the leaky bucket algorithm, originally from ATM network traffic shaping. Instead of counting events in windows, it tracks a single timestamp per key: the theoretical arrival time (TAT) of the next conforming request. A request is allowed if it does not arrive too far ahead of schedule, where "too far" is the burst tolerance.

**Key Properties:**

- **O(1) per operation**: Map lookup + integer comparison
- **O(n) memory**: One int64 per tracked key
- **No boundary bursts**: Unlike fixed windows, a key can never double its rate by straddling a window edge
- **Precise retry timing**: The TAT directly yields the exact time the next request becomes conforming

**Use Cases:** Per-client API rate limiting, smooth traffic shaping, request pacing with `Retry-After` responses, protecting downstream services from bursts.

**References:**

- ITU-T I.371 (Traffic control in ATM networks, origin of GCRA)
- RFC 6585 (429 Too Many Requests)
- Commonly used by Redis-backed limiters (e.g. redis-cell) and API gateways

## Quick Start

```go
package main

import (
    "fmt"
    "time"

    "github.com/vitalvas/gokit/gcra"
)

func main() {
    // 100 requests/second sustained, 10 instant burst,
    // max 10000 keys, cleanup every minute
    l := gcra.New(100, time.Second, 10, 10000, time.Minute)
    defer l.Stop()

    if l.Allow("user-123") {
        fmt.Println("Request allowed")
    } else {
        fmt.Printf("Rate limited, retry in %v\n", l.RetryAfter("user-123"))
    }
}
```

## Creating a Limiter

### New

```go
// 100 req/s sustained, burst of 10, max 10000 keys, cleanup every minute
l := gcra.New(100, time.Second, 10, 10000, time.Minute)
defer l.Stop()

// 1000 req/minute, no extra burst (strict pacing), lazy cleanup only
l := gcra.New(1000, time.Minute, 1, 100000, 0)
defer l.Stop()
```

**Parameters:**

- `limit`: Sustained number of requests per period (zero/negative defaults to 1)
- `period`: Duration over which limit applies (zero/negative defaults to 1 second)
- `burst`: Requests a key may make instantly from a cold start (below 1 is treated as 1)
- `maxKeys`: Maximum number of tracked keys (zero/negative defaults to 1000)
- `cleanupInterval`: Background cleanup interval; zero or negative disables background cleanup

The emission interval is `period / limit`: one unit of capacity refills every interval. `burst` sets how many intervals of capacity may be saved up.

**Cleanup Modes:**

| Mode | cleanupInterval | Behavior |
|------|----------------|----------|
| Lazy only | `0` | Drained entries removed when admitting new keys |
| Background | `> 0` | Background goroutine removes drained entries periodically |

A drained entry (a key idle long enough to regain full burst) is indistinguishable from an untracked key, so removal never changes behavior - it only frees memory.

### Stop

Stop the background cleanup goroutine. Safe to call multiple times.

```go
l := gcra.New(100, time.Second, 10, 10000, time.Minute)

l.Stop()
l.Stop() // safe
```

## Checking Rate Limits

### Allow

Check whether a single request conforms, consuming one unit of capacity if it does. Denied requests consume nothing.

```go
if l.Allow("user-123") {
    // Process request
} else {
    // Reject request (429 Too Many Requests)
}
```

### AllowN

Check whether n requests conform as a batch, all-or-nothing. If n exceeds the currently available capacity, nothing is consumed. n less than 1 is always allowed and consumes nothing.

```go
// Charge a bulk operation as 25 requests
if l.AllowN("user-123", 25) {
    // Process batch
} else {
    // Batch does not fit in remaining capacity
}
```

A batch larger than the total burst capacity can never succeed, regardless of how long the key waits.

### RetryAfter

Duration until a single request for the key would be allowed. Returns 0 if a request would be allowed now. Read-only, consumes nothing.

```go
if !l.Allow("user-123") {
    w.Header().Set("Retry-After", strconv.Itoa(int(l.RetryAfter("user-123").Seconds()+1)))
    w.WriteHeader(http.StatusTooManyRequests)
    return
}
```

## Managing State

### Reset

Clear all tracked keys.

```go
l.Reset()
```

### Len

Number of currently tracked keys (may include drained entries not yet cleaned up).

```go
fmt.Printf("Tracked keys: %d\n", l.Len())
```

## Export and Import

Serialize the full limiter state (configuration and per-key capacity debt) for storage or transmission, and restore it later - across restarts or in another process.

```go
// Export
data, err := l.Export()
if err != nil {
    return err
}
os.WriteFile("limiter.state", data, 0o600)

// Import
data, _ := os.ReadFile("limiter.state")
l, err := gcra.Import(data)
if err != nil {
    return err
}
defer l.Stop()
```

**Behavior:**

- Exported TATs are wall-clock timestamps, so time spent offline counts: keys that fully drained between export and import are dropped, partially drained keys keep only their remaining debt
- Configuration (rate, burst, maxKeys, cleanup interval) travels with the state; the background cleanup goroutine is restarted if it was configured
- `Import` returns `ErrInvalidData` for structurally valid but inconsistent data (corrupted configuration)

## Capacity Behavior

When the number of tracked keys reaches `maxKeys`, adding a new key checks capacity:

1. First, remove all drained entries (keys that regained full burst capacity)
2. If still at capacity, deny the new key until an existing key fully drains.
   `RetryAfter` reports the earliest capacity release time for an untracked key.

**Sizing guidelines:**

| Use Case | Suggested maxKeys |
|----------|-------------------|
| Per-user API limits | Number of active users |
| Per-IP rate limiting | Number of unique client IPs |
| Per-endpoint limiting | Number of endpoints |

Set `maxKeys` to accommodate the expected active key population. Active limits
are never reset by key churn; capacity exhaustion denies new keys.

## Use Cases

### HTTP API Rate Limiting

```go
limiter := gcra.New(100, time.Second, 20, 100000, time.Minute)
defer limiter.Stop()

func handler(w http.ResponseWriter, r *http.Request) {
    key := clientKey(r)

    if !limiter.Allow(key) {
        seconds := int(limiter.RetryAfter(key).Seconds() + 1)
        w.Header().Set("Retry-After", strconv.Itoa(seconds))
        http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
        return
    }

    // Process request...
}
```

### Weighted Requests

Charge expensive operations more than cheap ones.

```go
limiter := gcra.New(1000, time.Second, 100, 100000, time.Minute)
defer limiter.Stop()

func search(userID string, query Query) error {
    cost := 1
    if query.Deep {
        cost = 10
    }

    if !limiter.AllowN(userID, cost) {
        return fmt.Errorf("rate limit exceeded")
    }

    return runSearch(query)
}
```

### Outbound Request Pacing

Smooth calls to a downstream service that enforces its own limits.

```go
// Downstream allows 50 req/s; pace with no burst to stay smooth
pacer := gcra.New(50, time.Second, 1, 10, 0)
defer pacer.Stop()

func callDownstream(req Request) (Response, error) {
    for !pacer.Allow("downstream") {
        time.Sleep(pacer.RetryAfter("downstream"))
    }
    return client.Do(req)
}
```

### Multi-Tier Limits

```go
perSecond := gcra.New(10, time.Second, 10, 100000, time.Minute)
perHour := gcra.New(1000, time.Hour, 50, 100000, 10*time.Minute)
defer perSecond.Stop()
defer perHour.Stop()

func allow(userID string) bool {
    // Most restrictive wins; note: a deny in the second limiter
    // still consumes from the first
    return perSecond.Allow(userID) && perHour.Allow(userID)
}
```

## Performance Characteristics

### Time Complexity

| Operation | Complexity | Description |
|-----------|------------|-------------|
| `Allow` / `AllowN` | O(1) | Map lookup + integer comparison |
| `RetryAfter` | O(1) | Map lookup |
| `Reset` | O(1) | Replace map |
| `Len` | O(1) | Map length |
| Eviction | O(n) | Scan all entries (only on capacity) |

### Benchmarks

Apple M4 Pro, Go 1.27:

| Benchmark | ns/op | Allocations |
|-----------|-------|-------------|
| Allow, single key | 20 | 0 |
| Allow, 1000 keys | 27 | 0 |
| Allow, parallel contended key | 140 | 0 |

### Space Complexity

**Memory usage:** O(n) where n = number of tracked keys

**Per-key overhead:** ~56 bytes (string key header + int64 TAT + map overhead); entry values contain no pointers, so the map is not scanned by the garbage collector.

| Keys | Memory |
|------|--------|
| 10,000 | ~560 KB |
| 100,000 | ~5.6 MB |
| 1,000,000 | ~56 MB |

## Thread Safety

All operations are thread-safe and protected by a single mutex.

```go
l := gcra.New(1000, time.Second, 100, 100000, time.Minute)
defer l.Stop()

// Safe to call from multiple goroutines
go func() { l.Allow("user-1") }()
go func() { l.Allow("user-2") }()
go func() { fmt.Println(l.RetryAfter("user-1")) }()
```

## Comparison with Other Rate Limiting Algorithms

| Algorithm | Burst Handling | Memory per key | Smoothness | Boundary burst |
|-----------|---------------|----------------|------------|----------------|
| GCRA (this package) | Configurable burst | 1 int64 | Very high | None |
| Fixed Window | Allows burst at boundary | counter + flag + time | Low | Up to 2x limit |
| Sliding Window Log | No boundary burst | O(rate) timestamps | High | None |
| Token Bucket | Controlled burst | tokens + time | High | None |

GCRA is mathematically equivalent to a token bucket, but stores one timestamp instead of a token count plus a refill timestamp, and yields the exact retry time for free.

**When to Use GCRA:**

- Need smooth per-key rate limiting without window-edge bursts
- Want precise `Retry-After` values for clients
- Memory per key matters (one int64)
- Request weights vary (`AllowN`)

**When NOT to Use:**

- Need hard lockout semantics after abuse (use `fixedwindow`)
- Need usage counters/quota reporting per window (use `fixedwindow`)
- Need distributed limiting across processes (needs shared storage, e.g. redis-cell)


