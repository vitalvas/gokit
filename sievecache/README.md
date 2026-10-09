# sievecache

A SIEVE cache in Go: simpler than LRU, faster under concurrency, with strong hit ratios on skewed workloads.

## Features

- **SIEVE eviction**: FIFO queue, one visited bit per entry, sweeping eviction hand
- **Hits never mutate the queue**: a Get takes a shared read lock and sets one atomic bit, so concurrent reads scale across cores
- **Quick demotion**: one-hit wonders are evicted fast and do not pollute the cache
- **Generic**: `Cache[K comparable, V any]`
- **Optional lazy TTL**: per-entry expiry; expired entries miss immediately and are evicted first
- **Zero allocations at steady state**: evicted nodes are recycled for new inserts
- **Eviction callback**: OnEvict for resource cleanup and metrics, invoked outside the lock
- **GetOrSet and Peek**: load-or-store in one call; read without touching eviction state
- **Thread-safe**
- **Zero dependencies**: Only uses Go standard library

## What is SIEVE?

SIEVE keeps entries in insertion order with a visited bit per entry. A hit sets the bit and nothing else. On eviction, a hand sweeps from the oldest entry toward the newest: visited entries get their bit cleared and survive; the first unvisited entry is evicted. The hand keeps its position between evictions, which is what separates SIEVE from CLOCK and gives newly inserted but never re-accessed entries a short life.

Reference: "SIEVE is Simpler than LRU: an Efficient Turn-Key Eviction Algorithm for Web Caches" (Zhang et al., NSDI 2024).

**Use Cases:** read-heavy caches with skewed (Zipfian) key popularity - session lookups, API token caches, per-key config, web object caches.

## Quick Start

```go
package main

import (
    "fmt"
    "time"

    "github.com/vitalvas/gokit/sievecache"
)

func main() {
    c := sievecache.New[string, string](10000)

    c.Set("session:abc", "user-1", time.Hour)

    if v, ok := c.Get("session:abc"); ok {
        fmt.Println(v)
    }
}
```

## API

### New

```go
c := sievecache.New[string, int](10000)
```

Capacity below 1 defaults to 1000.

### Set

```go
c.Set("key", 42, 0)           // never expires
c.Set("key", 42, 5*time.Minute) // expires after 5 minutes
```

Updating an existing key replaces the value, refreshes the TTL, and counts as an access.

### Get

```go
v, ok := c.Get("key")
```

Returns the zero value and false for missing or expired keys.

### GetOrSet

Load-or-store in one call. Returns the existing value (and true) on a hit, or stores and returns the given value (and false) on a miss.

```go
v, loaded := c.GetOrSet("key", expensive, time.Minute)
```

### Peek

Read without setting the visited bit: the entry's eviction priority is unchanged. Useful for monitoring and debugging without distorting the hit pattern.

```go
v, ok := c.Peek("key")
```

### OnEvict

Callback for entries evicted to make room (capacity evictions, including expired entries taken by the sweep). Runs after the lock is released, so it may call back into the cache. Delete and Clear do not trigger it.

```go
c.OnEvict(func(key string, conn *Conn) {
    conn.Close()
})
```

### Delete, Len, Clear

```go
c.Delete("key")
n := c.Len() // includes expired entries not yet evicted
c.Clear()
```

## TTL Semantics

Expiry is lazy: there is no background goroutine (and therefore no Stop method). An expired entry misses on Get, and the eviction sweep removes expired entries first, even if their visited bit is set. Until one of those happens, expired entries still count toward Len and capacity.

## Performance

Apple M4 Pro, Go 1.27:

| Benchmark | ns/op | Allocations |
|-----------|-------|-------------|
| Get hit | 35 | 0 |
| Get hit, 14 goroutines on one key | 143 | 0 |
| Set with eviction | 71 | 0 (evicted node recycled) |
| Mixed 90% Get / 10% Set, parallel | 66 | 0 |

**When to use SIEVE:** reads dominate, key popularity is skewed (Zipfian), and the cheapest possible hit path matters.

**When NOT to use:** scan- or loop-heavy access patterns (sequential backups, full table reads) - SIEVE is not scan-resistant, so an adaptive algorithm fits better there.
