# bloomfilter

A high-performance Bloom filter implementation in Go with optimized bit-level operations and zero-allocation support.

## Features

- **Automatic sizing**: Calculates optimal filter size and hash count from expected elements and false positive rate
- **High performance**: Bit-level operations with uint64 for cache alignment
- **Zero allocations**: Byte slice operations without string conversions
- **Double hashing**: Efficient hash computation using double hashing technique
- **xxHash-style algorithm**: Fast, high-quality hash function
- **Power-of-2 optimization**: Fast modulo operations
- **Export/Import**: Serialize and deserialize filters
- **Estimated count**: Calculate approximate number of elements
- **Type safety**: Compile-time type checking
- **Zero dependencies**: Only uses Go standard library

## What is a Bloom Filter?

A Bloom filter is a space-efficient probabilistic data structure used to test whether an element is a member of a set. It can have false positives but never false negatives:

- **Definite no**: If `Contains` returns `false`, the element is definitely not in the set
- **Probable yes**: If `Contains` returns `true`, the element is probably in the set (may be a false positive)

**Use Cases**: URL deduplication, cache filtering, spell checkers, database query optimization, distributed systems.

## Installation

```bash
go get github.com/vitalvas/gokit/bloomfilter
```

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/vitalvas/gokit/bloomfilter"
)

func main() {
    // Create filter for 10,000 elements with 1% false positive rate
    bf := bloomfilter.NewBloomFilter(10000, 0.01)

    // Add elements
    bf.Add("user123")
    bf.Add("user456")
    bf.Add("user789")

    // Check membership
    if bf.Contains("user123") {
        fmt.Println("user123 might be in the set")
    }

    if !bf.Contains("user999") {
        fmt.Println("user999 is definitely not in the set")
    }
}
```

## Creating a Bloom Filter

### NewBloomFilter

Create a new filter with automatic optimal sizing.

```go
// Expected elements: 1,000,000
// False positive rate: 1%
bf := bloomfilter.NewBloomFilter(1_000_000, 0.01)

fmt.Printf("Filter size: %d bits\n", bf.Size())
fmt.Printf("Hash functions: %d\n", bf.K())
```

**Parameters:**
- `n`: Expected number of elements
- `p`: Desired false positive rate (0.0 to 1.0)

**Returns:**
- Optimally sized Bloom filter

**Automatic Optimizations:**
- Calculates optimal bit count using formula: `m = -n * ln(p) / (ln(2))^2`
- Calculates optimal hash count using formula: `k = (m/n) * ln(2)`
- Rounds size to power of 2 for fast modulo operations
- Ensures size is divisible by 64 for efficient bit operations

### Common False Positive Rates

| Rate | Description | Use Case |
|------|-------------|----------|
| `0.1` (10%) | High FP rate, small size | Non-critical caching |
| `0.01` (1%) | Balanced | General purpose |
| `0.001` (0.1%) | Low FP rate | High accuracy needed |
| `0.0001` (0.01%) | Very low FP, large size | Critical applications |

**Trade-off:** Lower false positive rate requires more memory.

## Adding Elements

### Add

Add a string to the filter.

```go
bf := bloomfilter.NewBloomFilter(1000, 0.01)

bf.Add("hello")
bf.Add("world")
bf.Add("user@example.com")
bf.Add("192.168.1.1")
```

**Performance:** Optimized for speed with minimal allocations.

### AddBytes

Add raw bytes to the filter (zero-allocation).

```go
bf := bloomfilter.NewBloomFilter(1000, 0.01)

// Binary data
data := []byte{0x01, 0x02, 0x03, 0x04}
bf.AddBytes(data)

// Text data
bf.AddBytes([]byte("hello world"))

// Empty data
bf.AddBytes([]byte{})
```

**Advantage:** No string conversion, ideal for processing binary data or when performance is critical.

## Checking Membership

### Contains

Check if a string might be in the filter.

```go
bf := bloomfilter.NewBloomFilter(1000, 0.01)

bf.Add("apple")
bf.Add("banana")

if bf.Contains("apple") {
    fmt.Println("apple is probably in the set")
}

if !bf.Contains("orange") {
    fmt.Println("orange is definitely not in the set")
}
```

**Returns:**
- `true`: Element is probably in the set (may be false positive)
- `false`: Element is definitely not in the set (never false negative)

### ContainsBytes

Check if raw bytes might be in the filter (zero-allocation).

```go
bf := bloomfilter.NewBloomFilter(1000, 0.01)

data := []byte("test data")
bf.AddBytes(data)

if bf.ContainsBytes(data) {
    fmt.Println("data is probably in the set")
}
```

### String and Bytes Interoperability

String and byte slice with same content are equivalent:

```go
bf := bloomfilter.NewBloomFilter(1000, 0.01)

// Add as string
bf.Add("hello")

// Check as bytes - works!
if bf.ContainsBytes([]byte("hello")) {
    fmt.Println("Found!")
}

// Add as bytes
bf.AddBytes([]byte("world"))

// Check as string - works!
if bf.Contains("world") {
    fmt.Println("Found!")
}
```

## Filter Operations

### Size

Get the number of bits in the filter.

```go
bf := bloomfilter.NewBloomFilter(10000, 0.01)
fmt.Printf("Filter uses %d bits\n", bf.Size())
```

### K

Get the number of hash functions.

```go
bf := bloomfilter.NewBloomFilter(10000, 0.01)
fmt.Printf("Uses %d hash functions\n", bf.K())
```

### EstimatedCount

Get the approximate number of elements added.

```go
bf := bloomfilter.NewBloomFilter(10000, 0.01)

for i := 0; i < 100; i++ {
    bf.Add(fmt.Sprintf("element-%d", i))
}

count := bf.EstimatedCount()
fmt.Printf("Approximately %d elements in filter\n", count)
// Output: Approximately ~100 elements in filter
```

**Formula:** `n ≈ -(m/k) * ln(1 - X/m)` where X is the number of set bits.

**Note:** This is an estimate and becomes less accurate as the filter fills up.

### Clear

Reset all bits in the filter.

```go
bf := bloomfilter.NewBloomFilter(1000, 0.01)

bf.Add("test1")
bf.Add("test2")

fmt.Println(bf.Contains("test1")) // true

bf.Clear()

fmt.Println(bf.Contains("test1")) // false
fmt.Println(bf.EstimatedCount())  // 0
```

**Use Case:** Reuse the same filter without reallocating memory.

## Serialization

### Export

Serialize the filter to bytes.

```go
bf := bloomfilter.NewBloomFilter(1000, 0.01)

bf.Add("user1")
bf.Add("user2")
bf.Add("user3")

data, err := bf.Export()
if err != nil {
    panic(err)
}

// Save to file, database, or send over network
os.WriteFile("filter.dat", data, 0644)
```

### ImportBloomFilter

Deserialize a filter from bytes.

```go
// Load from file, database, or network
data, err := os.ReadFile("filter.dat")
if err != nil {
    panic(err)
}

bf, err := bloomfilter.ImportBloomFilter(data)
if err != nil {
    panic(err)
}

// Use restored filter
if bf.Contains("user1") {
    fmt.Println("user1 found in restored filter")
}
```

**Format:** Uses Go's `encoding/gob` for efficient binary serialization.

## Use Cases

### URL Deduplication

Check if a URL has been seen before without storing all URLs.

```go
// Web crawler example
bf := bloomfilter.NewBloomFilter(10_000_000, 0.001)

func crawl(url string) {
    if bf.Contains(url) {
        // Probably already crawled
        return
    }

    bf.Add(url)
    // Crawl the URL
    processURL(url)
}
```

**Memory Savings:** 10M URLs in Bloom filter uses ~18MB vs ~400MB+ for full URL storage.

### Cache Key Filtering

Avoid cache misses by quickly checking if a key exists.

```go
type Cache struct {
    filter *bloomfilter.BloomFilter
    store  map[string]interface{}
}

func (c *Cache) Get(key string) (interface{}, bool) {
    // Fast negative check
    if !c.filter.Contains(key) {
        return nil, false
    }

    // Only check actual cache if filter says it might exist
    val, ok := c.store[key]
    return val, ok
}

func (c *Cache) Set(key string, value interface{}) {
    c.filter.Add(key)
    c.store[key] = value
}
```

**Benefit:** Avoid expensive cache lookups for keys that definitely don't exist.

### Email Spam Detection

Check if email address is on a blocklist.

```go
bf := bloomfilter.NewBloomFilter(1_000_000, 0.001)

// Load known spam addresses
for _, email := range spamAddresses {
    bf.Add(email)
}

func isSpam(email string) bool {
    if bf.Contains(email) {
        // Probably spam, do additional verification
        return verifySpam(email)
    }
    // Definitely not in blocklist
    return false
}
```

### Distributed Systems

Synchronize set membership across nodes efficiently.

```go
// Node 1: Export filter
bf1 := bloomfilter.NewBloomFilter(100000, 0.01)
for _, item := range localItems {
    bf1.Add(item)
}
data, _ := bf1.Export()

// Send data to Node 2 (small size!)
sendToNode2(data)

// Node 2: Import filter
bf2, _ := bloomfilter.ImportBloomFilter(data)

// Check if item exists on Node 1 without querying
if bf2.Contains("item123") {
    // Probably exists on Node 1
}
```

### Database Query Optimization

Avoid database queries for non-existent records.

```go
bf := bloomfilter.NewBloomFilter(1_000_000, 0.01)

// Populate filter with all user IDs from database
rows, _ := db.Query("SELECT user_id FROM users")
for rows.Next() {
    var userID string
    rows.Scan(&userID)
    bf.Add(userID)
}

func getUserByID(id string) (*User, error) {
    // Fast check before database query
    if !bf.Contains(id) {
        return nil, ErrUserNotFound
    }

    // Only query database if user might exist
    return db.QueryUser(id)
}
```

## Performance Characteristics

### Time Complexity

| Operation | Complexity | Description |
|-----------|------------|-------------|
| `NewBloomFilter` | O(m/64) | Allocate bitset array |
| `Add` | O(k) | k hash computations |
| `Contains` | O(k) | k hash computations |
| `EstimatedCount` | O(m/64) | Count set bits |
| `Clear` | O(m/64) | Reset all bits |
| `Export` | O(m/64) | Serialize bitset |
| `Import` | O(m/64) | Deserialize bitset |

Where:
- `m` = number of bits
- `k` = number of hash functions (typically 7-10)

### Space Complexity

**Memory usage:** Approximately `1.44 * n * log2(1/p)` bits

**Examples:**

| Elements | FP Rate | Memory | Memory/Element |
|----------|---------|--------|----------------|
| 1,000 | 1% | ~1.4 KB | 11.5 bits |
| 10,000 | 1% | ~14 KB | 11.5 bits |
| 100,000 | 1% | ~140 KB | 11.5 bits |
| 1,000,000 | 1% | ~1.4 MB | 11.5 bits |
| 1,000,000 | 0.1% | ~2.1 MB | 17.2 bits |

**Comparison with map[string]bool:**

| Data Structure | 1M Elements | Memory/Element |
|----------------|-------------|----------------|
| Bloom Filter (1% FP) | ~1.4 MB | 11.5 bits |
| Bloom Filter (0.1% FP) | ~2.1 MB | 17.2 bits |
| map[string]bool | ~40-80 MB | 320-640 bits |

**Savings:** 20-60x less memory than native Go map.

### Benchmarks (Apple M3 Pro)

| Operation | Time | Allocations |
|-----------|------|-------------|
| Add (1M elements) | ~94 ns | 2 allocs |
| Add (10M elements) | ~273 ns | 1 alloc |
| Add (50M elements) | ~442 ns | 1 alloc |
| AddBytes | ~16 ns | 0 allocs |
| Contains (1M elements) | ~90 ns | 1 alloc |
| Contains (10M elements) | ~309 ns | 1 alloc |
| ContainsBytes | ~17 ns | 0 allocs |
| Hash (string) | ~15 ns | 0 allocs |
| Hash (bytes) | ~15 ns | 0 allocs |

**Performance Characteristics:**

- Zero-allocation operations with byte methods
- Consistent ~15ns hashing for all inputs
- Scales linearly with filter size
- ~90-450ns per operation for typical use cases

### Optimizations

The implementation includes several performance optimizations:

1. **Power-of-2 sizing**: Enables fast modulo using bitwise AND
2. **uint64 bitset**: Better cache alignment and SIMD potential
3. **Double hashing**: Only 2 hash computations for k hashes
4. **xxHash-style algorithm**: Fast, high-quality hashing
5. **Zero-allocation bytes**: No string conversions for byte operations
6. **Unsafe string-to-bytes**: No allocation when hashing strings
7. **64-bit alignment**: Ensures size is divisible by 64

## Limitations

### False Positives

Bloom filters can have false positives but never false negatives.

```go
bf := bloomfilter.NewBloomFilter(100, 0.1) // 10% false positive rate

bf.Add("apple")

// Will return true (correct positive)
bf.Contains("apple")

// Might return true even though we never added "banana"
bf.Contains("banana") // Could be false positive
```

**Mitigation:** Use lower false positive rate (increases memory usage).

### No Deletion

Standard Bloom filters don't support element removal.

```go
bf := bloomfilter.NewBloomFilter(1000, 0.01)

bf.Add("item")
// No bf.Remove("item") - not supported
```

**Workarounds:**
- Use counting Bloom filter (different data structure)
- Create a new filter and re-add remaining elements
- Use `Clear()` to reset entire filter

### Growing Filter

Cannot resize filter after creation.

```go
bf := bloomfilter.NewBloomFilter(1000, 0.01)

// Add 1000 elements - OK
// Add 2000 elements - false positive rate increases
// Cannot resize!
```

**Solution:** Create new larger filter and re-add elements.

## Best Practices

### Choose Appropriate Size

```go
// Estimate maximum elements
maxElements := 1_000_000

// Choose acceptable false positive rate
falsePositiveRate := 0.01 // 1%

// Create filter
bf := bloomfilter.NewBloomFilter(maxElements, falsePositiveRate)
```

### Use Bytes for Performance

```go
// When processing binary data or performance is critical
data := []byte("binary data")
bf.AddBytes(data)
if bf.ContainsBytes(data) {
    // Process
}

// When working with strings naturally
bf.Add("user@example.com")
if bf.Contains("user@example.com") {
    // Process
}
```

### Monitor Fill Rate

```go
bf := bloomfilter.NewBloomFilter(10000, 0.01)

// Add elements
for _, item := range items {
    bf.Add(item)
}

// Check estimated count
count := bf.EstimatedCount()
capacity := 10000

if float64(count)/float64(capacity) > 0.8 {
    fmt.Println("Warning: Filter is 80% full, consider creating larger filter")
}
```

### Combine with Exact Check

For critical applications, verify positive results:

```go
func userExists(userID string) bool {
    // Quick negative check
    if !bf.Contains(userID) {
        return false // Definitely doesn't exist
    }

    // Verify with exact check (database, map, etc.)
    return db.CheckUserExists(userID)
}
```

### Persistence

Save and restore filters to avoid recomputation:

```go
// Startup: try to load existing filter
data, err := os.ReadFile("filter.dat")
var bf *bloomfilter.BloomFilter

if err == nil {
    bf, _ = bloomfilter.ImportBloomFilter(data)
} else {
    bf = bloomfilter.NewBloomFilter(1_000_000, 0.01)
    // Populate filter
}

// Shutdown: save filter
data, _ := bf.Export()
os.WriteFile("filter.dat", data, 0644)
```

## Examples

### Duplicate Detection in Stream

```go
bf := bloomfilter.NewBloomFilter(1_000_000, 0.001)

func processBatch(events []Event) {
    for _, event := range events {
        if bf.Contains(event.ID) {
            // Likely duplicate, skip
            continue
        }

        bf.Add(event.ID)
        processEvent(event)
    }
}
```

### Weak Password Checker

```go
// Load common passwords into filter
bf := bloomfilter.NewBloomFilter(10_000_000, 0.001)

passwords, _ := os.ReadFile("common-passwords.txt")
for _, pwd := range strings.Split(string(passwords), "\n") {
    bf.Add(pwd)
}

func isWeakPassword(password string) bool {
    if bf.Contains(password) {
        return true // Probably in common password list
    }
    return false // Not in list
}
```

### IP Blocklist

```go
bf := bloomfilter.NewBloomFilter(100_000, 0.01)

// Load blocked IPs
for _, ip := range blockedIPs {
    bf.Add(ip)
}

func isBlocked(ip string) bool {
    return bf.Contains(ip)
}

// Use in middleware
func middleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        clientIP := r.RemoteAddr
        if isBlocked(clientIP) {
            http.Error(w, "Forbidden", http.StatusForbidden)
            return
        }
        next.ServeHTTP(w, r)
    })
}
```

## License

This project is part of the [gokit](https://github.com/vitalvas/gokit) library.
