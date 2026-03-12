# radixtree

A high-performance, concurrent-safe, generic radix tree (compressed trie) implementation in Go.

## Features

- **Generic types**: Type-safe values via Go generics (`Tree[V any]`)
- **Concurrent-safe**: Read-write mutex for safe concurrent access
- **Zero-allocation reads**: Get, LongestPrefix, and Delete allocate nothing
- **Minimal-allocation bulk operations**: PrefixSearch and WalkPrefix use a single contiguous buffer for all keys
- **Both string and []byte keys**: Zero-copy byte slice support via `unsafe.String`
- **Prefix operations**: Search and iterate by prefix, find longest matching prefix
- **Full iteration**: Walk all entries with early-stop support
- **Automatic compaction**: Nodes are merged after deletion to keep the tree compact
- **Zero dependencies**: Only uses Go standard library

## What is a Radix Tree?

A radix tree (also called a compressed trie or Patricia tree) is a space-optimized trie where nodes with a single child are merged with their parent. This makes it efficient for storing keys that share common prefixes.

**Key properties:**

- Lookup, insert, and delete in O(k) time where k is key length
- Memory-efficient for keys with shared prefixes (URLs, file paths, IP prefixes, etc.)
- Supports prefix-based operations natively

**Use cases:** HTTP routing, IP/BGP prefix lookup, autocomplete, file system paths, configuration key storage.

## Installation

```bash
go get github.com/vitalvas/gokit/radixtree
```

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/vitalvas/gokit/radixtree"
)

func main() {
    tree := radixtree.New[string]()

    // Insert key-value pairs
    tree.Insert("/api/v1/users", "users-handler")
    tree.Insert("/api/v1/users/admin", "admin-handler")
    tree.Insert("/api/v1/posts", "posts-handler")

    // Exact lookup
    if val, ok := tree.Get("/api/v1/users"); ok {
        fmt.Println(val) // users-handler
    }

    // Longest prefix match
    key, val, ok := tree.LongestPrefix("/api/v1/users/123/profile")
    if ok {
        fmt.Printf("%s -> %s\n", key, val) // /api/v1/users -> users-handler
    }

    // Find all entries under a prefix
    results := tree.PrefixSearch("/api/v1/users")
    for k, v := range results {
        fmt.Printf("%s -> %s\n", k, v)
    }
}
```

## API

### Creating a Tree

```go
tree := radixtree.New[int]()
```

### Insert

Add or update a key-value pair. Returns `true` if the key is new, `false` if updated.

```go
tree.Insert("key", 42)
tree.InsertBytes([]byte("key"), 42)
```

### Get

Retrieve a value by exact key. Returns the value and whether it was found.

```go
val, ok := tree.Get("key")
val, ok = tree.GetBytes([]byte("key"))
```

### Contains

Check if a key exists without retrieving the value.

```go
exists := tree.Contains("key")
exists = tree.ContainsBytes([]byte("key"))
```

### Delete

Remove a key. Returns `true` if the key existed and was removed.

```go
deleted := tree.Delete("key")
deleted = tree.DeleteBytes([]byte("key"))
```

### ShortestPrefix

Find the entry with the shortest key that is a prefix of the given key.

```go
tree.Insert("/api", 1)
tree.Insert("/api/v1", 2)
tree.Insert("/api/v1/users", 3)

key, val, ok := tree.ShortestPrefix("/api/v1/users/123")
// key="/api", val=1, ok=true

key, val, ok = tree.ShortestPrefixBytes([]byte("/api/v1/users/123"))
```

### LongestPrefix

Find the entry with the longest key that is a prefix of the given key.

```go
tree.Insert("/api", 1)
tree.Insert("/api/v1", 2)
tree.Insert("/api/v1/users", 3)

key, val, ok := tree.LongestPrefix("/api/v1/users/123")
// key="/api/v1/users", val=3, ok=true

key, val, ok = tree.LongestPrefixBytes([]byte("/api/v1/users/123"))
```

### PrefixSearch

Return all entries where the key starts with the given prefix.

```go
tree.Insert("apple", 1)
tree.Insert("app", 2)
tree.Insert("application", 3)
tree.Insert("banana", 4)

results := tree.PrefixSearch("app")
// {"apple": 1, "app": 2, "application": 3}

results = tree.PrefixSearchBytes([]byte("app"))
```

### WalkPrefix

Iterate over all entries matching a prefix with a callback. More efficient than PrefixSearch when you don't need to collect all results into a map.

```go
tree.WalkPrefix("/api/v1", func(key string, value int) bool {
    fmt.Printf("%s -> %d\n", key, value)
    return true // return false to stop early
})

tree.WalkPrefixBytes([]byte("/api/v1"), func(key string, value int) bool {
    return true
})
```

### HasPrefix

Check if any key in the tree starts with the given prefix. More efficient than PrefixSearch when you only need to know if matches exist.

```go
exists := tree.HasPrefix("/api/v1")
exists = tree.HasPrefixBytes([]byte("/api/v1"))
```

### DeletePrefix

Remove all keys that start with the given prefix. Returns the number of deleted entries.

```go
count := tree.DeletePrefix("/api/v1")
count = tree.DeletePrefixBytes([]byte("/api/v1"))
```

### Merge

Add all entries from another tree into this tree. Existing keys are overwritten.

```go
tree1 := radixtree.New[int]()
tree1.Insert("a", 1)

tree2 := radixtree.New[int]()
tree2.Insert("b", 2)

tree1.Merge(tree2) // tree1 now contains both "a" and "b"
```

### Keys

Return all keys in the tree.

```go
keys := tree.Keys() // []string{"a", "b", "c"}
```

### Values

Return all values in the tree.

```go
values := tree.Values() // []int{1, 2, 3}
```

### Walk

Iterate over all entries in the tree.

```go
tree.Walk(func(key string, value int) bool {
    fmt.Printf("%s -> %d\n", key, value)
    return true // return false to stop early
})
```

### Clear

Remove all entries from the tree.

```go
tree.Clear()
```

### Len

Return the number of entries.

```go
count := tree.Len()
```

## Use Cases

### HTTP Route Matching

```go
tree := radixtree.New[http.HandlerFunc]()
tree.Insert("/api/v1/users", usersHandler)
tree.Insert("/api/v1/posts", postsHandler)
tree.Insert("/api/v2/users", usersV2Handler)

// Find the best matching route
_, handler, ok := tree.LongestPrefix(request.URL.Path)
if ok {
    handler(w, r)
}
```

### BGP/IP Prefix Lookup

```go
tree := radixtree.New[string]()
tree.Insert("10.", "class-a")
tree.Insert("10.0.", "datacenter-1")
tree.Insert("10.0.1.", "subnet-1")
tree.Insert("192.168.", "private")

_, location, ok := tree.LongestPrefix("10.0.1.50")
// location="subnet-1"
```

### Configuration Key Store

```go
tree := radixtree.New[string]()
tree.Insert("app.database.host", "localhost")
tree.Insert("app.database.port", "5432")
tree.Insert("app.cache.host", "redis")

// Get all database config
dbConfig := tree.PrefixSearch("app.database.")
```

### Autocomplete

```go
tree := radixtree.New[bool]()
tree.Insert("hello", true)
tree.Insert("help", true)
tree.Insert("helicopter", true)
tree.Insert("world", true)

suggestions := tree.PrefixSearch("hel")
// {"hello": true, "help": true, "helicopter": true}
```

## Concurrency

All operations are concurrent-safe. Reads use a shared read lock, writes use an exclusive lock.

```go
tree := radixtree.New[int]()

// Safe to call from multiple goroutines
go func() { tree.Insert("key1", 1) }()
go func() { tree.Get("key1") }()
go func() { tree.PrefixSearch("key") }()
```

## Performance Characteristics

### Time Complexity

| Operation | Complexity | Description |
|-----------|------------|-------------|
| Insert | O(k) | k = key length |
| Get / Contains | O(k) | k = key length |
| Delete | O(k) | k = key length |
| ShortestPrefix | O(k) | k = key length, returns on first match |
| LongestPrefix | O(k) | k = key length |
| HasPrefix | O(k) | k = prefix length |
| PrefixSearch | O(k + n) | k = prefix length, n = matching entries |
| WalkPrefix | O(k + n) | k = prefix length, n = matching entries |
| DeletePrefix | O(k + n) | k = prefix length, n = matching entries |
| Walk / Keys / Values | O(N) | N = total entries |
| Merge | O(M) | M = entries in source tree |
| Clear | O(1) | Resets root node |

### Benchmarks (Apple M3 Pro)

| Operation | Time | Memory | Allocations |
|-----------|------|--------|-------------|
| Insert | ~135 ns/op | 88 B/op | 1 alloc |
| Get | ~58 ns/op | 0 B/op | 0 allocs |
| Delete | ~93 ns/op | 0 B/op | 0 allocs |
| ShortestPrefix | ~7 ns/op | 0 B/op | 0 allocs |
| LongestPrefix | ~22 ns/op | 0 B/op | 0 allocs |
| PrefixSearch (1111 results) | ~41 us/op | 79 KB/op | 7 allocs |
| WalkPrefix (1111 results) | ~18 us/op | 25 KB/op | 1 alloc |

**Allocation details:**

- **Get, Contains, Delete, LongestPrefix, HasPrefix**: Zero allocations
- **Insert**: 1 allocation for the new tree node
- **PrefixSearch**: Pre-calculated contiguous key buffer + pre-sized map (7 allocs regardless of result count)
- **WalkPrefix**: Single contiguous key buffer (1 alloc regardless of result count)

### Memory Layout

Keys with shared prefixes share tree nodes, reducing memory usage compared to a flat map:

```
Insert: "test", "testing", "team", "tea"

Tree structure:
  root
   |
  "te"
  / \
"st"  "a"
 |     |
"ing" "m"
```

## License

This project is part of the [gokit](https://github.com/vitalvas/gokit) library.
