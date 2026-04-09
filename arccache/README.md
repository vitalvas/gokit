# arccache

A generic Adaptive Replacement Cache (ARC) in Go with TTL, byte-size tracking, eviction callbacks, and self-tuning eviction.

## Features

- **Self-tuning**: Automatically adapts to workload (recency vs. frequency)
- **Scan-resistant**: Does not degrade under sequential access patterns
- **Generic**: Works with any comparable key type and any value type
- **TTL support**: Per-entry and default time-to-live
- **Byte-size tracking**: Optional memory-aware eviction via custom size function
- **Eviction callback**: Notified when entries are evicted, deleted, or expired
- **Configurable cleanup**: Optional background goroutine for expired entry removal
- **Thread-safe**: All operations protected by mutex
- **O(1) operations**: Hash map lookups + doubly linked list operations
- **Zero dependencies**: Only uses Go standard library

## What is ARC?

ARC (Adaptive Replacement Cache) is a self-tuning cache replacement algorithm that balances between recency and frequency. It maintains four internal lists:

- **T1**: Recently accessed items (seen once)
- **T2**: Frequently accessed items (seen at least twice)
- **B1**: Ghost entries evicted from T1 (keys only, no values)
- **B2**: Ghost entries evicted from T2 (keys only, no values)

The adaptation parameter **p** (target size for T1) adjusts based on ghost hits:

- Miss in B1 (recently evicted): increase p (favor recency)
- Miss in B2 (frequently evicted): decrease p (favor frequency)

**Reference:** "ARC: A Self-Tuning, Low Overhead Replacement Cache" (Megiddo & Modha, FAST 2003).

**Use Cases:** General-purpose caching, database page caches, CDN caches, application-level caches with mixed workloads.

## Installation

```bash
go get github.com/vitalvas/gokit/arccache
```

## Quick Start

```go
package main

import (
    "fmt"
    "time"

    "github.com/vitalvas/gokit/arccache"
)

func main() {
    // Simple cache with max 1000 items
    c := arccache.New[string, int](1000)
    defer c.Stop()

    c.Set("answer", 42, time.Hour)

    if val, ok := c.Get("answer"); ok {
        fmt.Printf("answer = %d\n", val)
    }
}
```

## Creating a Cache

### New

Create a cache with a maximum number of items.

```go
// Cache up to 10000 items, no TTL, no byte limit
c := arccache.New[string, int](10000)
defer c.Stop()
```

### NewWithOptions

Create a cache with full configuration.

```go
c := arccache.NewWithOptions(arccache.Options[string, []byte]{
    MaxItems: 10000,
    MaxBytes: 64 * 1024 * 1024, // 64 MB
    SizeFunc: func(k string, v []byte) int {
        return len(k) + len(v)
    },
    DefaultTTL:      time.Hour,
    CleanupInterval: 5 * time.Minute,
    OnEvict: func(k string, v []byte) {
        log.Printf("evicted: %s (%d bytes)", k, len(v))
    },
})
defer c.Stop()
```

**Options:**

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `MaxItems` | `int` | 1000 | Maximum number of cached items |
| `MaxBytes` | `int` | 0 (disabled) | Maximum byte size; requires `SizeFunc` |
| `SizeFunc` | `func(K, V) int` | nil | Returns byte size of a key-value pair |
| `DefaultTTL` | `time.Duration` | 0 (no expiry) | Default TTL for entries |
| `OnEvict` | `func(K, V)` | nil | Called when an entry is evicted |
| `CleanupInterval` | `time.Duration` | 0 (lazy only) | Background cleanup interval |

**Notes:**

- `MaxBytes` is ignored if `SizeFunc` is nil
- `MaxItems` and `MaxBytes` are independent limits; either can trigger eviction
- Always call `Stop()` when done to release background goroutines

## Storing and Retrieving

### Set

Add or update a key-value pair.

```go
c := arccache.New[string, int](1000)
defer c.Stop()

// With explicit TTL
c.Set("key1", 42, time.Hour)

// With default TTL (or no expiry if DefaultTTL is 0)
c.Set("key2", 100, 0)

// Update existing key
c.Set("key1", 99, time.Hour)
```

**Behavior:**

- New entries go to T1 (recently accessed)
- Updating an existing entry promotes it to T2 (frequently accessed)
- Re-inserting a previously evicted key (ghost hit) triggers ARC adaptation

### Get

Retrieve a value from the cache.

```go
val, ok := c.Get("key1")
if ok {
    fmt.Printf("Found: %d\n", val)
} else {
    fmt.Println("Not found or expired")
}
```

**Behavior:**

- Items in T1 are promoted to T2 on access (recency -> frequency)
- Items in T2 are moved to the front (most recently used)
- Expired items are lazily removed and return `false`

### Delete

Remove an entry from the cache.

```go
c.Delete("key1")

// Safe to call on non-existent keys
c.Delete("missing")
```

## Inspecting State

### Len

Get the number of non-ghost entries.

```go
fmt.Printf("Cached items: %d\n", c.Len())
```

### Bytes

Get current tracked byte usage. Returns 0 if no `SizeFunc` was configured.

```go
fmt.Printf("Memory usage: %d bytes\n", c.Bytes())
```

### Clear

Remove all entries and reset the adaptation parameter.

```go
c.Clear()
```

## TTL Behavior

```go
// Per-entry TTL
c.Set("short-lived", data, 5*time.Minute)
c.Set("long-lived", data, 24*time.Hour)

// Uses DefaultTTL from Options
c.Set("default-ttl", data, 0)

// No expiration (when DefaultTTL is also 0)
c.Set("permanent", data, 0)
```

**Expiry modes:**

| `DefaultTTL` | `Set` TTL | Result |
|---------------|-----------|--------|
| 0 | 0 | No expiration |
| 0 | 5m | Expires in 5 minutes |
| 1h | 0 | Expires in 1 hour |
| 1h | 5m | Expires in 5 minutes |

**Cleanup modes:**

| Mode | `CleanupInterval` | Behavior |
|------|-------------------|----------|
| Lazy only | `0` | Expired entries removed on access |
| Background | `> 0` | Periodic goroutine removes expired entries |

## Byte-Size Tracking

Track and limit memory usage with a custom size function.

```go
c := arccache.NewWithOptions(arccache.Options[string, []byte]{
    MaxItems: 100000,
    MaxBytes: 256 * 1024 * 1024, // 256 MB
    SizeFunc: func(k string, v []byte) int {
        return len(k) + len(v)
    },
})
defer c.Stop()

c.Set("image", largeBlob, time.Hour)

fmt.Printf("Using %d bytes\n", c.Bytes())
```

**Eviction priority:** When byte limit is exceeded, entries are evicted using the same ARC T1/T2 preference as item-count eviction.

## Eviction Callback

Get notified when entries are removed from the cache.

```go
c := arccache.NewWithOptions(arccache.Options[string, *Connection]{
    MaxItems: 1000,
    OnEvict: func(k string, conn *Connection) {
        conn.Close()
    },
})
defer c.Stop()
```

**Triggers:**

- Item evicted to make room for new entries
- Item removed via `Delete`
- Item expired (on access or background cleanup)

**Note:** The callback is called with the mutex held. Do not call back into the cache from the callback.

## Use Cases

### Application Cache

General-purpose cache for expensive computations or database results.

```go
cache := arccache.New[string, *UserProfile](10000)
defer cache.Stop()

func getUser(id string) (*UserProfile, error) {
    if profile, ok := cache.Get(id); ok {
        return profile, nil
    }

    profile, err := db.LoadUser(id)
    if err != nil {
        return nil, err
    }

    cache.Set(id, profile, 15*time.Minute)
    return profile, nil
}
```

### Memory-Bounded Blob Cache

Cache binary data with a hard memory limit.

```go
blobCache := arccache.NewWithOptions(arccache.Options[string, []byte]{
    MaxItems: 100000,
    MaxBytes: 512 * 1024 * 1024, // 512 MB
    SizeFunc: func(k string, v []byte) int {
        return len(k) + len(v)
    },
    DefaultTTL:      time.Hour,
    CleanupInterval: 5 * time.Minute,
})
defer blobCache.Stop()

func getBlob(key string) ([]byte, error) {
    if data, ok := blobCache.Get(key); ok {
        return data, nil
    }

    data, err := storage.Fetch(key)
    if err != nil {
        return nil, err
    }

    blobCache.Set(key, data, 0)
    return data, nil
}
```

### Multi-Tier Cache

Use ARC as an L1 cache backed by Redis.

```go
l1 := arccache.NewWithOptions(arccache.Options[string, []byte]{
    MaxItems:   10000,
    MaxBytes:   64 * 1024 * 1024,
    SizeFunc: func(k string, v []byte) int {
        return len(k) + len(v)
    },
    DefaultTTL: 5 * time.Minute,
})
defer l1.Stop()

func get(key string) ([]byte, error) {
    // L1: in-process ARC cache
    if data, ok := l1.Get(key); ok {
        return data, nil
    }

    // L2: Redis
    data, err := redis.Get(ctx, key).Bytes()
    if err != nil {
        return nil, err
    }

    l1.Set(key, data, 0)
    return data, nil
}
```

### HTTP Response Cache

Cache HTTP responses with size tracking.

```go
type CachedResponse struct {
    StatusCode int
    Headers    map[string]string
    Body       []byte
}

responseCache := arccache.NewWithOptions(arccache.Options[string, *CachedResponse]{
    MaxItems: 50000,
    MaxBytes: 256 * 1024 * 1024,
    SizeFunc: func(url string, resp *CachedResponse) int {
        size := len(url) + len(resp.Body)
        for k, v := range resp.Headers {
            size += len(k) + len(v)
        }
        return size
    },
    DefaultTTL:      10 * time.Minute,
    CleanupInterval: time.Minute,
})
defer responseCache.Stop()
```

## Performance Characteristics

### Time Complexity

| Operation | Complexity | Description |
|-----------|------------|-------------|
| `Get` | O(1) | Map lookup + list move |
| `Set` (existing) | O(1) | Map lookup + list move |
| `Set` (new) | O(1) | Map insert + possible eviction |
| `Delete` | O(1) | Map delete + list remove |
| `Len` | O(1) | Sum of list lengths |
| `Bytes` | O(1) | Return tracked value |
| `Clear` | O(1) | Reset all structures |

### Space Complexity

**Memory usage:** O(2c) where c = MaxItems (cache entries + ghost entries)

**Per-entry overhead:** ~120 bytes (element struct + map entry + list pointers)

| Items | Overhead |
|-------|----------|
| 1,000 | ~240 KB |
| 10,000 | ~2.4 MB |
| 100,000 | ~24 MB |
| 1,000,000 | ~240 MB |

### Benchmarks (Apple M3 Pro)

| Operation | Time | Allocations |
|-----------|------|-------------|
| Get (hot key) | ~14 ns | 0 allocs |
| Set (existing key) | ~15 ns | 0 allocs |
| Set (new key) | ~235 ns | 3 allocs |
| Set (with eviction) | ~305 ns | 3 allocs |
| Concurrent Get/Set | ~200 ns | 1 alloc |

**Performance Notes:**

- Get and Set on existing keys are zero-allocation
- New key insertion allocates the entry struct
- Concurrent performance ~15x slower due to mutex contention
- Suitable for high-throughput caching (millions of ops/sec for hot keys)

## ARC vs. LRU

| Property | ARC | LRU |
|----------|-----|-----|
| Adaptation | Self-tuning | None |
| Scan resistance | Yes | No |
| Memory overhead | 2x (ghost lists) | 1x |
| Implementation | More complex | Simple |
| Workload sensitivity | Adapts automatically | Fixed recency bias |

**When to use ARC:**

- Mixed workloads (some keys accessed recently, others frequently)
- Scan-heavy workloads (large sequential scans that would flush an LRU)
- Workloads where the access pattern changes over time

**When LRU is sufficient:**

- Purely recency-based access patterns
- Memory is very constrained (ARC uses 2x ghost overhead)
- Simple caching needs without scan resistance

## Thread Safety

All operations are thread-safe:

```go
c := arccache.New[string, int](10000)
defer c.Stop()

// Safe from multiple goroutines
go func() {
    for _, item := range batch1 {
        c.Set(item.Key, item.Value, time.Hour)
    }
}()

go func() {
    for _, key := range queries {
        if val, ok := c.Get(key); ok {
            process(val)
        }
    }
}()
```

## Best Practices

### Size MaxItems for Your Workload

```go
// Too small: frequent eviction, poor hit rate
c := arccache.New[string, int](10)

// Good: sized for working set
c := arccache.New[string, int](10000)
```

### Use Background Cleanup for Long TTLs

```go
// Long TTL: use background cleanup to free memory
c := arccache.NewWithOptions(arccache.Options[string, int]{
    MaxItems:        10000,
    DefaultTTL:      24 * time.Hour,
    CleanupInterval: time.Hour,
})
defer c.Stop()

// Short TTL: lazy cleanup is sufficient
c := arccache.NewWithOptions(arccache.Options[string, int]{
    MaxItems:   10000,
    DefaultTTL: time.Minute,
})
defer c.Stop()
```

### Always Call Stop

```go
c := arccache.NewWithOptions(arccache.Options[string, int]{
    MaxItems:        10000,
    CleanupInterval: 5 * time.Minute,
})
defer c.Stop() // Prevents goroutine leak
```

### Use SizeFunc for Variable-Size Values

```go
// For []byte values
arccache.Options[string, []byte]{
    SizeFunc: func(k string, v []byte) int {
        return len(k) + len(v)
    },
}

// For string values
arccache.Options[string, string]{
    SizeFunc: func(k string, v string) int {
        return len(k) + len(v)
    },
}

// For struct values (approximate)
arccache.Options[string, *User]{
    SizeFunc: func(k string, u *User) int {
        return len(k) + len(u.Name) + len(u.Email) + 64 // base struct overhead
    },
}
```

## License

This project is part of the [gokit](https://github.com/vitalvas/gokit) library.
