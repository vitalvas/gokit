# xstd

Ordered generic collections: `BTreeMap` and `BTreeSet`, both backed by an
in-memory B-tree.

Keys are kept in ascending order, so iteration and range queries are ordered.
Keys must satisfy `cmp.Ordered` (any comparable primitive). Neither type is safe
for concurrent use; guard with your own lock if shared across goroutines.

Methods that may not find a value return `(T, bool)`; those that return a
key/value pair return `(K, V, bool)`.

## Bounds

Range queries take two `Bound[K]` endpoints:

```go
xstd.Included(v)    // inclusive endpoint
xstd.Excluded(v)    // exclusive endpoint
xstd.Unbounded[K]() // open-ended endpoint
```

## BTreeMap

```go
m := xstd.NewBTreeMap[string, int]()

old, existed := m.Insert("b", 2) // existed == false
m.Insert("a", 1)
m.Insert("c", 3)

v, ok := m.Get("b")   // 2, true
m.ContainsKey("z")    // false
m.Len()               // 3

for k, v := range m.Iter() { // a, b, c in order
    fmt.Println(k, v)
}

for k, v := range m.Range(xstd.Included("a"), xstd.Excluded("c")) { // a, b
    fmt.Println(k, v)
}

old, ok = m.Remove("b") // 2, true
```

| Method | Description |
|--------|-------------|
| `Insert(k, v) (V, bool)` | Insert/update; returns the previous value. |
| `TryInsert(k, v) (V, bool)` | Insert only if absent; returns the current value + whether inserted. |
| `GetOrInsertWith(k, func() V) V` | Get, inserting a computed value if absent. |
| `Get(k) (V, bool)` | Value and whether present. |
| `GetKeyValue(k) (K, V, bool)` | Stored key and value. |
| `ContainsKey(k) bool` | Presence check. |
| `Remove(k) (V, bool)` | Remove; returns the removed value. |
| `RemoveEntry(k) (K, V, bool)` | Remove; returns the removed key and value. |
| `Len() int` | Entry count. |
| `IsEmpty() bool` | Emptiness check. |
| `Clear()` | Remove all entries. |
| `FirstKeyValue() (K, V, bool)` | Smallest entry. |
| `LastKeyValue() (K, V, bool)` | Largest entry. |
| `PopFirst() (K, V, bool)` | Remove and return the smallest entry. |
| `PopLast() (K, V, bool)` | Remove and return the largest entry. |
| `Iter() iter.Seq2[K, V]` | Ascending iteration. |
| `Keys() iter.Seq[K]` | Ascending keys. |
| `Values() iter.Seq[V]` | Values in ascending key order. |
| `IntoKeys() []K` | Drain to a sorted key slice. |
| `IntoValues() []V` | Drain to a value slice. |
| `Range(lo, hi Bound[K]) iter.Seq2[K, V]` | Entries within the bounds. |
| `LowerBound(Bound[K]) iter.Seq2[K, V]` | Entries from the lower bound to the end. |
| `UpperBound(Bound[K]) iter.Seq2[K, V]` | Entries from the start to the upper bound. |
| `Retain(func(K, V) bool)` | Keep only matching entries. |
| `ExtractIf(func(K, V) bool) iter.Seq2[K, V]` | Lazily remove and yield matching entries. |
| `Append(*BTreeMap)` | Move all entries from another map. |
| `SplitOff(k) *BTreeMap` | Split off entries with key >= k. |

## BTreeSet

A `BTreeSet[T]` is a `BTreeMap[T, struct{}]`.

```go
s := xstd.NewBTreeSet[int]()

s.Insert(3) // true  (newly added)
s.Insert(3) // false (already present)
s.Insert(1)

for v := range s.Iter() {                                   // 1, 3
    fmt.Println(v)
}
for v := range s.Range(xstd.Included(1), xstd.Included(3)) { // 1, 3
    fmt.Println(v)
}

a := xstd.NewBTreeSet[int]()
a.Insert(1); a.Insert(2); a.Insert(3)
b := xstd.NewBTreeSet[int]()
b.Insert(3); b.Insert(4)

for v := range a.Union(b)        { fmt.Println(v) } // 1 2 3 4
for v := range a.Intersection(b) { fmt.Println(v) } // 3
for v := range a.Difference(b)   { fmt.Println(v) } // 1 2
```

| Method | Description |
|--------|-------------|
| `Insert(v) bool` | Add; `true` if newly added. |
| `Replace(v) (T, bool)` | Insert; return the replaced element. |
| `Get(v) (T, bool)` | Stored element equal to v. |
| `Contains(v) bool` | Presence check. |
| `Remove(v) bool` | Remove; `true` if present. |
| `Take(v) (T, bool)` | Remove and return the stored element. |
| `GetOrInsert(v) T` | Get, inserting v if absent. |
| `GetOrInsertWith(v, func(T) T) T` | Get, inserting a computed element if absent. |
| `First() (T, bool)` | Smallest element. |
| `Last() (T, bool)` | Largest element. |
| `PopFirst() (T, bool)` | Remove and return the smallest. |
| `PopLast() (T, bool)` | Remove and return the largest. |
| `Len() int` | Element count. |
| `IsEmpty() bool` | Emptiness check. |
| `Clear()` | Remove all elements. |
| `Retain(func(T) bool)` | Keep only matching elements. |
| `Iter() iter.Seq[T]` | Ascending iteration. |
| `Range(lo, hi Bound[T]) iter.Seq[T]` | Elements within the bounds. |
| `ExtractIf(func(T) bool) iter.Seq[T]` | Lazily remove and yield matching elements. |
| `Append(*BTreeSet)` | Move all elements from another set. |
| `SplitOff(v) *BTreeSet` | Split off elements >= v. |
| `Union(*BTreeSet) iter.Seq[T]` | Elements in either set. |
| `Intersection(*BTreeSet) iter.Seq[T]` | Elements in both sets. |
| `Difference(*BTreeSet) iter.Seq[T]` | Elements only in the receiver. |
| `SymmetricDifference(*BTreeSet) iter.Seq[T]` | Elements in exactly one set. |
| `IsDisjoint(*BTreeSet) bool` | No shared elements. |
| `IsSubset(*BTreeSet) bool` | Every element is in other. |
| `IsSuperset(*BTreeSet) bool` | Other is a subset. |

## Performance

B-tree operations are O(log n). Benchmarks on Apple M3 Pro (100k entries):

| Operation | Time | Allocations |
|-----------|------|-------------|
| `Insert` | ~40 ns/op | 0 allocs/op |
| `Get` | ~35 ns/op | 0 allocs/op |
