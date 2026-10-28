package xstd

import (
	"cmp"
	"iter"
	"slices"
)

// Iter iterates over every entry in ascending key order, mirroring Rust's
// BTreeMap::iter. Breaking out of the range loop stops iteration early.
func (m *BTreeMap[K, V]) Iter() iter.Seq2[K, V] {
	return m.Range(Unbounded[K](), Unbounded[K]())
}

// Keys iterates over every key in ascending order (Rust BTreeMap::keys).
func (m *BTreeMap[K, V]) Keys() iter.Seq[K] {
	return func(yield func(K) bool) {
		for k := range m.Iter() {
			if !yield(k) {
				return
			}
		}
	}
}

// Values iterates over every value in ascending key order (Rust
// BTreeMap::values).
func (m *BTreeMap[K, V]) Values() iter.Seq[V] {
	return func(yield func(V) bool) {
		for _, v := range m.Iter() {
			if !yield(v) {
				return
			}
		}
	}
}

// IntoKeys drains the map and returns all keys in ascending order, mirroring
// Rust's BTreeMap::into_keys. The map is left empty.
func (m *BTreeMap[K, V]) IntoKeys() []K {
	out := make([]K, 0, m.size)
	for k := range m.Iter() {
		out = append(out, k)
	}
	m.Clear()
	return out
}

// IntoValues drains the map and returns all values in ascending key order,
// mirroring Rust's BTreeMap::into_values. The map is left empty.
func (m *BTreeMap[K, V]) IntoValues() []V {
	out := make([]V, 0, m.size)
	for _, v := range m.Iter() {
		out = append(out, v)
	}
	m.Clear()
	return out
}

// Range iterates over entries whose keys fall within [lower, upper] as
// described by the two bounds, in ascending order. It mirrors Rust's
// BTreeMap::range, which accepts any RangeBounds; use Included/Excluded/
// Unbounded to build the endpoints.
func (m *BTreeMap[K, V]) Range(lower, upper Bound[K]) iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		if m.root != nil {
			m.root.walkBounded(lower, upper, yield)
		}
	}
}

// LowerBound iterates ascending over every entry at or after the given lower
// bound, mirroring forward iteration from Rust's BTreeMap::lower_bound cursor.
func (m *BTreeMap[K, V]) LowerBound(bound Bound[K]) iter.Seq2[K, V] {
	return m.Range(bound, Unbounded[K]())
}

// UpperBound iterates ascending over every entry up to and bounded by the
// given upper bound, mirroring forward iteration from the start to Rust's
// BTreeMap::upper_bound cursor position.
func (m *BTreeMap[K, V]) UpperBound(bound Bound[K]) iter.Seq2[K, V] {
	return m.Range(Unbounded[K](), bound)
}

// afterLower reports whether key satisfies the lower bound.
func afterLower[K cmp.Ordered](key K, b Bound[K]) bool {
	switch b.Kind {
	case BoundIncluded:
		return !cmp.Less(key, b.Value)
	case BoundExcluded:
		return cmp.Less(b.Value, key)
	default:
		return true
	}
}

// beforeUpper reports whether key satisfies the upper bound.
func beforeUpper[K cmp.Ordered](key K, b Bound[K]) bool {
	switch b.Kind {
	case BoundIncluded:
		return !cmp.Less(b.Value, key)
	case BoundExcluded:
		return cmp.Less(key, b.Value)
	default:
		return true
	}
}

func (n *btreeNode[K, V]) walkBounded(lower, upper Bound[K], yield func(K, V) bool) bool {
	start := 0
	if lower.Kind != BoundUnbounded {
		start, _ = n.search(lower.Value)
	}
	for i := start; i < len(n.keys); i++ {
		if !n.leaf() && !n.children[i].walkBounded(lower, upper, yield) {
			return false
		}
		if !afterLower(n.keys[i], lower) {
			continue
		}
		if !beforeUpper(n.keys[i], upper) {
			return true
		}
		if !yield(n.keys[i], n.values[i]) {
			return false
		}
	}
	if !n.leaf() {
		return n.children[len(n.keys)].walkBounded(lower, upper, yield)
	}
	return true
}

// FirstKeyValue returns the entry with the smallest key and whether the map is
// non-empty, mirroring Rust's BTreeMap::first_key_value.
func (m *BTreeMap[K, V]) FirstKeyValue() (K, V, bool) {
	if m.root == nil {
		var (
			zeroK K
			zeroV V
		)
		return zeroK, zeroV, false
	}
	k, v := m.root.min()
	return k, v, true
}

// LastKeyValue returns the entry with the largest key and whether the map is
// non-empty, mirroring Rust's BTreeMap::last_key_value.
func (m *BTreeMap[K, V]) LastKeyValue() (K, V, bool) {
	if m.root == nil {
		var (
			zeroK K
			zeroV V
		)
		return zeroK, zeroV, false
	}
	k, v := m.root.max()
	return k, v, true
}

// PopFirst removes and returns the entry with the smallest key, mirroring
// Rust's BTreeMap::pop_first.
func (m *BTreeMap[K, V]) PopFirst() (K, V, bool) {
	k, v, ok := m.FirstKeyValue()
	if !ok {
		return k, v, false
	}
	m.Remove(k)
	return k, v, true
}

// PopLast removes and returns the entry with the largest key, mirroring Rust's
// BTreeMap::pop_last.
func (m *BTreeMap[K, V]) PopLast() (K, V, bool) {
	k, v, ok := m.LastKeyValue()
	if !ok {
		return k, v, false
	}
	m.Remove(k)
	return k, v, true
}

// Retain keeps only the entries for which f returns true, mirroring Rust's
// BTreeMap::retain.
func (m *BTreeMap[K, V]) Retain(f func(K, V) bool) {
	var drop []K
	for k, v := range m.Iter() {
		if !f(k, v) {
			drop = append(drop, k)
		}
	}
	for _, k := range drop {
		m.Remove(k)
	}
}

// ExtractIf removes and returns every entry for which pred returns true, in
// ascending key order, mirroring Rust's BTreeMap::extract_if. Like Rust, each
// matching entry is removed only as the returned iterator yields it: stopping
// early leaves the not-yet-yielded matches in the map. The predicate is called
// only during iteration and stops being called when iteration stops.
func (m *BTreeMap[K, V]) ExtractIf(pred func(K, V) bool) iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		// Snapshot keys so deletion and rebalancing cannot invalidate traversal.
		keys := slices.Collect(m.Keys())
		for _, k := range keys {
			v, ok := m.Get(k)
			if !ok || !pred(k, v) {
				continue
			}
			m.Remove(k)
			if !yield(k, v) {
				return
			}
		}
	}
}

// Append moves all entries from other into m, leaving other empty. On a key
// present in both, other's value wins, mirroring Rust's BTreeMap::append.
// Appending a map to itself has no effect.
func (m *BTreeMap[K, V]) Append(other *BTreeMap[K, V]) {
	if m == other {
		return
	}
	for k, v := range other.Iter() {
		m.Insert(k, v)
	}
	other.Clear()
}

// SplitOff removes every entry with a key >= key from m and returns them as a
// new map, mirroring Rust's BTreeMap::split_off.
func (m *BTreeMap[K, V]) SplitOff(key K) *BTreeMap[K, V] {
	out := NewBTreeMap[K, V]()
	var move []K
	for k, v := range m.LowerBound(Included(key)) {
		out.Insert(k, v)
		move = append(move, k)
	}
	for _, k := range move {
		m.Remove(k)
	}
	return out
}
