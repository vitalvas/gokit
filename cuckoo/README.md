# cuckoo

A high-performance Cuckoo Filter implementation in Go for approximate set membership testing with deletion support.

## Features

- **Supports deletion**: Unlike Bloom filters, can remove elements
- **Low false positive rate**: ~0.03% with 8-bit fingerprints
- **Fast lookups**: ~19ns per contains operation
- **Space-efficient**: Comparable to Bloom filters, better than hash sets
- **Insert/Delete/Contains**: All operations in O(1) expected time
- **Export/Import**: Serialize for storage or network transmission
- **High load factor**: Supports up to 95% capacity utilization
- **Zero dependencies**: Only uses Go standard library

## What is a Cuckoo Filter?

A Cuckoo Filter is a probabilistic data structure for approximate set membership testing. It provides better performance than Bloom filters with the key advantage of supporting deletions.

**Key Properties:**

- **False positives**: Rare but possible (configurable rate)
- **False negatives**: Never occur
- **Deletions**: Fully supported (unlike Bloom filters)
- **Space efficiency**: Uses ~8-12 bits per element

**Use Cases:** Cache management, duplicate detection, rate limiting, database query optimization, network packet filtering.

## Installation

```bash
go get github.com/vitalvas/gokit/cuckoo
```

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/vitalvas/gokit/cuckoo"
)

func main() {
    // Create filter with expected capacity
    filter := cuckoo.New(1000)

    // Insert elements
    filter.Insert([]byte("user-123"))
    filter.Insert([]byte("user-456"))

    // Check membership
    if filter.Contains([]byte("user-123")) {
        fmt.Println("user-123 might exist")
    }

    // Delete elements
    filter.Delete([]byte("user-123"))

    // Check again
    if !filter.Contains([]byte("user-123")) {
        fmt.Println("user-123 does not exist")
    }
}
```

## Creating a Filter

### New

Create a new cuckoo filter with specified capacity.

```go
// Create filter for 10,000 expected elements
filter := cuckoo.New(10000)

fmt.Printf("Capacity: %.0f elements\n", float64(filter.numBuckets*filter.bucketSize))
fmt.Printf("Load factor: %.2f\n", filter.LoadFactor())
```

**Parameters:**

- `capacity`: Expected number of elements to store

**Configuration:**

- 4 entries per bucket
- 8-bit fingerprints
- Up to 95% load factor support

## Adding Elements

### Insert

Add an element to the filter.

```go
filter := cuckoo.New(1000)

ok := filter.Insert([]byte("element-1"))
if ok {
    fmt.Println("Inserted successfully")
} else {
    fmt.Println("Filter is full")
}
```

**Returns:** `true` if inserted, `false` if filter is full

**Note:** Can insert duplicates (will increase count)

### InsertUnique

Add an element only if it doesn't exist.

```go
filter := cuckoo.New(1000)

// First insert succeeds
ok1 := filter.InsertUnique([]byte("element-1"))
fmt.Println(ok1) // true

// Second insert fails (already exists)
ok2 := filter.InsertUnique([]byte("element-1"))
fmt.Println(ok2) // false
```

**Returns:** `true` if inserted, `false` if already exists or filter is full

**Note:** May rarely reject due to fingerprint collisions (false positives)

## Checking Membership

### Contains

Check if an element might be in the filter.

```go
filter := cuckoo.New(1000)
filter.Insert([]byte("test"))

// Check membership
if filter.Contains([]byte("test")) {
    fmt.Println("Element might exist (could be false positive)")
}

if !filter.Contains([]byte("other")) {
    fmt.Println("Element definitely does not exist")
}
```

**Returns:**

- `true`: Element might exist (with small false positive rate)
- `false`: Element definitely does not exist (no false negatives)

**Performance:** ~19ns per operation

## Deleting Elements

### Delete

Remove an element from the filter.

```go
filter := cuckoo.New(1000)
filter.Insert([]byte("element"))

ok := filter.Delete([]byte("element"))
if ok {
    fmt.Println("Deleted successfully")
} else {
    fmt.Println("Element not found")
}
```

**Returns:** `true` if found and deleted, `false` otherwise

**Note:** If you inserted an element multiple times, you need to delete it the same number of times.

## Utility Operations

### Count

Get the number of elements in the filter.

```go
filter := cuckoo.New(1000)

for i := 0; i < 100; i++ {
    filter.Insert([]byte(fmt.Sprintf("element-%d", i)))
}

fmt.Printf("Count: %d\n", filter.Count())
// Output: Count: 100
```

### LoadFactor

Get the current load factor (0-1 range).

```go
filter := cuckoo.New(1000)

for i := 0; i < 500; i++ {
    filter.Insert([]byte(fmt.Sprintf("element-%d", i)))
}

fmt.Printf("Load factor: %.2f%%\n", filter.LoadFactor()*100)
// Output: Load factor: 12.21% (500/4096)
```

**Formula:** `load factor = count / (numBuckets * bucketSize)`

### Reset

Clear all elements from the filter.

```go
filter := cuckoo.New(1000)
filter.Insert([]byte("element"))

filter.Reset()
fmt.Printf("Count: %d\n", filter.Count())
// Output: Count: 0
```

## Serialization

### Export

Serialize the filter for storage or transmission.

```go
filter := cuckoo.New(1000)

for i := 0; i < 500; i++ {
    filter.Insert([]byte(fmt.Sprintf("element-%d", i)))
}

// Export to bytes
data, err := filter.Export()
if err != nil {
    log.Fatal(err)
}

// Save to file
os.WriteFile("filter.dat", data, 0644)
```

### Import

Deserialize a filter from exported data.

```go
// Load from file
data, err := os.ReadFile("filter.dat")
if err != nil {
    log.Fatal(err)
}

// Import
filter, err := cuckoo.Import(data)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Restored filter with %d elements\n", filter.Count())
```

## Use Cases

### Duplicate Detection

Remove duplicates from a data stream.

```go
filter := cuckoo.New(100000)

func processBatch(items []string) []string {
    var unique []string

    for _, item := range items {
        if filter.InsertUnique([]byte(item)) {
            unique = append(unique, item)
        }
    }

    return unique
}
```

### Cache Management

Track which items are in cache.

```go
cacheFilter := cuckoo.New(10000)

func get(key string) (value interface{}, found bool) {
    // Quick check before expensive cache lookup
    if !cacheFilter.Contains([]byte(key)) {
        return nil, false
    }

    // Item might be in cache
    return cache.Get(key)
}

func set(key string, value interface{}) {
    cache.Set(key, value)
    cacheFilter.Insert([]byte(key))
}

func evict(key string) {
    cache.Delete(key)
    cacheFilter.Delete([]byte(key))
}
```

### Rate Limiting

Track request counts per user.

```go
rateLimiter := cuckoo.New(100000)

func checkRateLimit(userID string, maxRequests int) bool {
    key := []byte(userID)

    // Count how many times user appears in filter
    count := 0
    for filter.Contains(key) {
        count++
        if count >= maxRequests {
            return false // Rate limit exceeded
        }
    }

    // Allow request and track it
    filter.Insert(key)
    return true
}
```

### Network Packet Filtering

Filter duplicate packets.

```go
seenPackets := cuckoo.New(1000000)

func processPacket(packet []byte) bool {
    packetID := computePacketID(packet)

    // Check if we've seen this packet before
    if seenPackets.Contains(packetID) {
        return false // Duplicate, drop it
    }

    // New packet, process it
    seenPackets.Insert(packetID)
    return true
}
```

## Performance Characteristics

### Time Complexity

| Operation | Average | Worst Case | Description |
|-----------|---------|------------|-------------|
| `Insert` | O(1) | O(n) | Rare cuckoo kicks |
| `InsertUnique` | O(1) | O(n) | Includes Contains check |
| `Contains` | O(1) | O(1) | Check two buckets |
| `Delete` | O(1) | O(1) | Check two buckets |
| `Reset` | O(n) | O(n) | Clear all buckets |
| `Export` | O(n) | O(n) | Serialize all data |
| `Import` | O(n) | O(n) | Deserialize all data |

Where n = number of buckets

### Space Complexity

**Memory usage:** `(capacity / 4) * (4 entries * 8 bits) = 2 * capacity bytes`

**Examples:**

| Capacity | Buckets | Memory | Load at 95% |
|----------|---------|--------|-------------|
| 1,000 | 256 | 1 KB | 950 elements |
| 10,000 | 4,096 | 16 KB | 9,500 elements |
| 100,000 | 32,768 | 128 KB | 95,000 elements |
| 1,000,000 | 262,144 | 1 MB | 950,000 elements |

### Benchmarks (Apple M3 Pro)

| Operation | Time | Allocations |
|-----------|------|-------------|
| Insert | ~5.0 µs | 0 allocs |
| InsertUnique | ~501 ns | 1 alloc |
| Contains | ~19.5 ns | 0 allocs |
| Delete | ~260 ns | 0 allocs |
| Export | ~43.5 µs | 47 allocs |
| Import | ~128.6 µs | 4289 allocs |

**Performance Notes:**

- Contains is extremely fast (~19ns)
- Insert can be slower due to cuckoo kicks
- Zero allocations for basic operations
- Export/Import optimized with gob encoding

## Accuracy

### False Positive Rate

**Theoretical rate:** ~0.78% with 8-bit fingerprints

**Formula:** `FP rate ≈ 2b / 2^f`

Where:
- b = number of entries per bucket (4)
- f = fingerprint bits (8)

**Example:** `2 * 4 / 2^8 = 8/256 ≈ 0.03125 = 3.125%`

**Measured rate:** ~0.00% in tests (varies with load factor)

### False Negatives

**Never occur.** If `Contains` returns `false`, the element is definitely not in the filter.

## Limitations

### Approximate Membership

Cuckoo filters provide probabilistic results.

```go
filter := cuckoo.New(1000)
filter.Insert([]byte("test"))

// Might return true for non-inserted elements (rare)
if filter.Contains([]byte("other")) {
    // Could be false positive
}
```

**Mitigation:** Use higher capacity for lower false positive rate.

### Filter Can Fill Up

Insertions can fail when load factor is too high.

```go
filter := cuckoo.New(10)

// Eventually returns false
for i := 0; ; i++ {
    if !filter.Insert([]byte(fmt.Sprintf("element-%d", i))) {
        fmt.Printf("Filter full after %d insertions\n", i)
        break
    }
}
```

**Mitigation:** Create filter with sufficient capacity (aim for 90-95% load).

### Deletion Can Cause False Negatives

Deleting an element that was never inserted might corrupt the filter.

```go
filter := cuckoo.New(100)
filter.Insert([]byte("test1"))

// Dangerous: deleting non-existent element
filter.Delete([]byte("test2"))

// Might now return false for test1 if fingerprints collide
```

**Mitigation:** Only delete elements you know were inserted.

## Best Practices

### Size Filter Appropriately

```go
// Estimate expected elements
expectedElements := 100000

// Add 10% buffer for safety
capacity := uint(float64(expectedElements) * 1.1)

filter := cuckoo.New(capacity)
```

### Check Insertion Success

```go
if !filter.Insert(data) {
    // Filter is full, handle error
    log.Println("Warning: Filter is full")

    // Option 1: Create larger filter
    newFilter := cuckoo.New(filter.Count() * 2)

    // Option 2: Reset and start over
    filter.Reset()
}
```

### Track Deletions Carefully

```go
// Good: Track what you inserted
inserted := make(map[string]bool)

func add(item string) {
    data := []byte(item)
    filter.Insert(data)
    inserted[item] = true
}

func remove(item string) {
    if inserted[item] {
        data := []byte(item)
        filter.Delete(data)
        delete(inserted, item)
    }
}
```

### Monitor Load Factor

```go
func insertWithCheck(filter *cuckoo.Filter, data []byte) bool {
    // Check load factor before inserting
    if filter.LoadFactor() > 0.95 {
        log.Println("Warning: Filter is nearly full")
        return false
    }

    return filter.Insert(data)
}
```

### Handle False Positives

```go
func mightContain(filter *cuckoo.Filter, item string) bool {
    data := []byte(item)

    // Quick filter check
    if !filter.Contains(data) {
        return false // Definitely not present
    }

    // Might be false positive, verify with authoritative source
    return database.Exists(item)
}
```

## Comparison with Other Approaches

| Approach | Insert | Contains | Delete | Space | False Positives |
|----------|--------|----------|--------|-------|-----------------|
| Hash Set | O(1) | O(1) | O(1) | High | None |
| Bloom Filter | O(1) | O(1) | ❌ | Low | ~1% |
| **Cuckoo Filter** | **O(1)** | **O(1)** | **O(1)** | **Low** | **~0.03%** |
| Sorted Array | O(n) | O(log n) | O(n) | Medium | None |

**Cuckoo Filter Advantages:**

- **Supports deletion** (unlike Bloom filters)
- **Faster lookups** than Bloom filters (~2x)
- **Lower false positive rate** at similar space usage
- **Better cache locality** (fewer memory accesses)

**When to Use:**

- ✅ Need deletion support
- ✅ Can tolerate rare false positives
- ✅ Want fast membership tests
- ✅ Limited memory budget
- ❌ Need 100% accuracy (use hash set)
- ❌ Extremely low false positive requirement (use hash set)

## License

This project is part of the [gokit](https://github.com/vitalvas/gokit) library.
