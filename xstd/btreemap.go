// Package xstd provides ordered collections modeled on Rust's standard
// library: BTreeMap and BTreeSet, both backed by an in-memory B-tree.
package xstd

import "cmp"

// btreeDegree is the minimum degree (t) of the B-tree. Each node holds
// between t-1 and 2t-1 keys. 32 keeps nodes cache-friendly while shallow.
const btreeDegree int = 32

// BTreeMap is an ordered map backed by a B-tree, keyed on cmp.Ordered keys.
// Iteration and range queries visit entries in ascending key order. It is
// not safe for concurrent use.
type BTreeMap[K cmp.Ordered, V any] struct {
	root *btreeNode[K, V]
	size int
}

type btreeNode[K cmp.Ordered, V any] struct {
	keys     []K
	values   []V
	children []*btreeNode[K, V]
}

func (n *btreeNode[K, V]) leaf() bool {
	return len(n.children) == 0
}

// NewBTreeMap creates an empty ordered map.
func NewBTreeMap[K cmp.Ordered, V any]() *BTreeMap[K, V] {
	return &BTreeMap[K, V]{}
}

// Len returns the number of entries in the map.
func (m *BTreeMap[K, V]) Len() int {
	return m.size
}

// IsEmpty reports whether the map contains no entries.
func (m *BTreeMap[K, V]) IsEmpty() bool {
	return m.size == 0
}

// Clear removes all entries from the map.
func (m *BTreeMap[K, V]) Clear() {
	m.root = nil
	m.size = 0
}

// Get returns the value for key and whether it was present.
func (m *BTreeMap[K, V]) Get(key K) (V, bool) {
	n := m.root
	for n != nil {
		i, found := n.search(key)
		if found {
			return n.values[i], true
		}
		if n.leaf() {
			break
		}
		n = n.children[i]
	}
	var zero V
	return zero, false
}

// GetKeyValue returns the stored key and value for key and whether it was
// present, mirroring Rust's BTreeMap::get_key_value.
func (m *BTreeMap[K, V]) GetKeyValue(key K) (K, V, bool) {
	n := m.root
	for n != nil {
		i, found := n.search(key)
		if found {
			return n.keys[i], n.values[i], true
		}
		if n.leaf() {
			break
		}
		n = n.children[i]
	}
	var (
		zeroK K
		zeroV V
	)
	return zeroK, zeroV, false
}

// ContainsKey reports whether key is present.
func (m *BTreeMap[K, V]) ContainsKey(key K) bool {
	_, ok := m.Get(key)
	return ok
}

// search returns the index of key, or the index of the first key greater
// than it, plus whether an exact match was found.
func (n *btreeNode[K, V]) search(key K) (int, bool) {
	lo, hi := 0, len(n.keys)
	for lo < hi {
		mid := (lo + hi) / 2
		switch cmp.Compare(key, n.keys[mid]) {
		case 0:
			return mid, true
		case 1:
			lo = mid + 1
		default:
			hi = mid
		}
	}
	return lo, false
}

// Insert inserts or updates the value for key and returns the previous value
// and whether a previous value existed. This mirrors Rust's BTreeMap::insert,
// which returns the old value as an Option.
func (m *BTreeMap[K, V]) Insert(key K, value V) (V, bool) {
	if m.root == nil {
		m.root = &btreeNode[K, V]{}
	}
	if len(m.root.keys) == 2*btreeDegree-1 {
		newRoot := &btreeNode[K, V]{children: []*btreeNode[K, V]{m.root}}
		newRoot.splitChild(0)
		m.root = newRoot
	}
	old, existed := m.root.insertNonFull(key, value)
	if !existed {
		m.size++
	}
	return old, existed
}

// TryInsert inserts value only if key is absent. It returns the value now
// stored for key and whether the insert happened: on an existing key it keeps
// the current value and returns (current, false), mirroring the error path of
// Rust's BTreeMap::try_insert.
func (m *BTreeMap[K, V]) TryInsert(key K, value V) (V, bool) {
	if existing, ok := m.Get(key); ok {
		return existing, false
	}
	m.Insert(key, value)
	return value, true
}

// GetOrInsertWith returns the value for key, inserting the result of f when key
// is absent, mirroring the common Rust entry().or_insert_with(..) pattern.
func (m *BTreeMap[K, V]) GetOrInsertWith(key K, f func() V) V {
	if existing, ok := m.Get(key); ok {
		return existing
	}
	v := f()
	m.Insert(key, v)
	return v
}

func (n *btreeNode[K, V]) insertNonFull(key K, value V) (V, bool) {
	i, found := n.search(key)
	if found {
		old := n.values[i]
		n.values[i] = value
		return old, true
	}
	if n.leaf() {
		n.keys = append(n.keys, key)
		n.values = append(n.values, value)
		copy(n.keys[i+1:], n.keys[i:])
		copy(n.values[i+1:], n.values[i:])
		n.keys[i] = key
		n.values[i] = value
		var zero V
		return zero, false
	}
	if len(n.children[i].keys) == 2*btreeDegree-1 {
		n.splitChild(i)
		switch cmp.Compare(key, n.keys[i]) {
		case 0:
			old := n.values[i]
			n.values[i] = value
			return old, true
		case 1:
			i++
		}
	}
	return n.children[i].insertNonFull(key, value)
}

// splitChild splits the full child at index i around its median key, which
// is pulled up into n.
func (n *btreeNode[K, V]) splitChild(i int) {
	child := n.children[i]
	mid := btreeDegree - 1

	right := &btreeNode[K, V]{
		keys:   append([]K(nil), child.keys[mid+1:]...),
		values: append([]V(nil), child.values[mid+1:]...),
	}
	if !child.leaf() {
		right.children = append([]*btreeNode[K, V](nil), child.children[mid+1:]...)
	}

	midKey, midVal := child.keys[mid], child.values[mid]
	child.keys = child.keys[:mid]
	child.values = child.values[:mid]
	if !child.leaf() {
		child.children = child.children[:mid+1]
	}

	n.children = append(n.children, nil)
	copy(n.children[i+2:], n.children[i+1:])
	n.children[i+1] = right

	n.keys = append(n.keys, midKey)
	n.values = append(n.values, midVal)
	copy(n.keys[i+1:], n.keys[i:])
	copy(n.values[i+1:], n.values[i:])
	n.keys[i] = midKey
	n.values[i] = midVal
}

// Remove removes key and returns its value and whether it was present,
// mirroring Rust's BTreeMap::remove.
func (m *BTreeMap[K, V]) Remove(key K) (V, bool) {
	var zero V
	if m.root == nil {
		return zero, false
	}
	old, found := m.root.delete(key)
	if !found {
		return zero, false
	}
	m.size--
	if len(m.root.keys) == 0 && !m.root.leaf() {
		m.root = m.root.children[0]
	}
	return old, true
}

// RemoveEntry removes key and returns the stored key and value and whether it
// was present, mirroring Rust's BTreeMap::remove_entry.
func (m *BTreeMap[K, V]) RemoveEntry(key K) (K, V, bool) {
	v, ok := m.Remove(key)
	if !ok {
		var zeroK K
		return zeroK, v, false
	}
	return key, v, true
}

// delete removes key from the subtree rooted at n, which is assumed to hold
// at least btreeDegree keys (or be the root). It returns the deleted value.
func (n *btreeNode[K, V]) delete(key K) (V, bool) {
	i, found := n.search(key)
	if n.leaf() {
		if !found {
			var zero V
			return zero, false
		}
		old := n.values[i]
		n.keys = append(n.keys[:i], n.keys[i+1:]...)
		n.values = append(n.values[:i], n.values[i+1:]...)
		return old, true
	}
	if found {
		return n.deleteInternal(i), true
	}
	return n.deleteFromChild(key, i)
}

// deleteInternal removes the key at index i, an internal separator, by
// replacing it with its in-order predecessor or successor.
func (n *btreeNode[K, V]) deleteInternal(i int) V {
	old := n.values[i]
	switch {
	case len(n.children[i].keys) >= btreeDegree:
		k, v := n.children[i].max()
		n.keys[i], n.values[i] = k, v
		n.children[i].delete(k)
	case len(n.children[i+1].keys) >= btreeDegree:
		k, v := n.children[i+1].min()
		n.keys[i], n.values[i] = k, v
		n.children[i+1].delete(k)
	default:
		n.mergeChild(i)
		n.children[i].delete(n.children[i].keys[btreeDegree-1])
	}
	return old
}

// deleteFromChild descends into child i to delete key, ensuring the child
// has at least btreeDegree keys before recursing.
func (n *btreeNode[K, V]) deleteFromChild(key K, i int) (V, bool) {
	if len(n.children[i].keys) < btreeDegree {
		i = n.fill(i)
	}
	return n.children[i].delete(key)
}

func (n *btreeNode[K, V]) min() (K, V) {
	for !n.leaf() {
		n = n.children[0]
	}
	return n.keys[0], n.values[0]
}

func (n *btreeNode[K, V]) max() (K, V) {
	for !n.leaf() {
		n = n.children[len(n.children)-1]
	}
	last := len(n.keys) - 1
	return n.keys[last], n.values[last]
}

// fill ensures child i has at least btreeDegree keys by borrowing from a
// sibling or merging, and returns the index the child now lives at.
func (n *btreeNode[K, V]) fill(i int) int {
	if i > 0 && len(n.children[i-1].keys) >= btreeDegree {
		n.borrowFromPrev(i)
		return i
	}
	if i < len(n.children)-1 && len(n.children[i+1].keys) >= btreeDegree {
		n.borrowFromNext(i)
		return i
	}
	if i < len(n.children)-1 {
		n.mergeChild(i)
		return i
	}
	n.mergeChild(i - 1)
	return i - 1
}

func (n *btreeNode[K, V]) borrowFromPrev(i int) {
	child, left := n.children[i], n.children[i-1]

	child.keys = append([]K{n.keys[i-1]}, child.keys...)
	child.values = append([]V{n.values[i-1]}, child.values...)

	lastK := len(left.keys) - 1
	n.keys[i-1], n.values[i-1] = left.keys[lastK], left.values[lastK]
	left.keys = left.keys[:lastK]
	left.values = left.values[:lastK]

	if !child.leaf() {
		lastC := len(left.children) - 1
		child.children = append([]*btreeNode[K, V]{left.children[lastC]}, child.children...)
		left.children = left.children[:lastC]
	}
}

func (n *btreeNode[K, V]) borrowFromNext(i int) {
	child, right := n.children[i], n.children[i+1]

	child.keys = append(child.keys, n.keys[i])
	child.values = append(child.values, n.values[i])

	n.keys[i], n.values[i] = right.keys[0], right.values[0]
	right.keys = right.keys[1:]
	right.values = right.values[1:]

	if !child.leaf() {
		child.children = append(child.children, right.children[0])
		right.children = right.children[1:]
	}
}

// mergeChild merges child i+1 into child i with n.keys[i] as the separator.
func (n *btreeNode[K, V]) mergeChild(i int) {
	left, right := n.children[i], n.children[i+1]

	left.keys = append(left.keys, n.keys[i])
	left.values = append(left.values, n.values[i])
	left.keys = append(left.keys, right.keys...)
	left.values = append(left.values, right.values...)
	left.children = append(left.children, right.children...)

	n.keys = append(n.keys[:i], n.keys[i+1:]...)
	n.values = append(n.values[:i], n.values[i+1:]...)
	n.children = append(n.children[:i+1], n.children[i+2:]...)
}
