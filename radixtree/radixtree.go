package radixtree

import (
	"sync"
	"unsafe"
)

// Tree is a concurrent-safe radix tree (compressed trie) that supports
// generic value types. It provides efficient storage and retrieval of
// key-value pairs where keys share common prefixes.
type Tree[V any] struct {
	mu   sync.RWMutex
	root node[V]
	size int
}

type node[V any] struct {
	prefix   string
	value    V
	hasValue bool
	children []*node[V]
}

// New creates a new empty radix tree.
func New[V any]() *Tree[V] {
	return &Tree[V]{}
}

// Len returns the number of entries in the tree.
func (t *Tree[V]) Len() int {
	t.mu.RLock()
	n := t.size
	t.mu.RUnlock()

	return n
}

// Insert adds or updates a key-value pair in the tree.
// Returns true if a new key was inserted, false if an existing key was updated.
func (t *Tree[V]) Insert(key string, value V) bool {
	t.mu.Lock()
	result := t.insert(key, value)
	t.mu.Unlock()

	return result
}

// InsertBytes adds or updates a key-value pair using a byte slice key.
// Returns true if a new key was inserted, false if an existing key was updated.
func (t *Tree[V]) InsertBytes(key []byte, value V) bool {
	return t.Insert(bytesToString(key), value)
}

func (t *Tree[V]) insert(key string, value V) bool {
	n := &t.root

	for {
		if len(key) == 0 {
			isNew := !n.hasValue
			n.value = value
			n.hasValue = true

			if isNew {
				t.size++
			}

			return isNew
		}

		child := n.findChild(key[0])
		if child == nil {
			n.children = append(n.children, &node[V]{
				prefix:   key,
				value:    value,
				hasValue: true,
			})

			t.size++

			return true
		}

		commonLen := longestCommonPrefix(key, child.prefix)

		if commonLen == len(child.prefix) {
			key = key[commonLen:]
			n = child

			continue
		}

		// Split the existing node. Replace in parent before mutating
		// child.prefix, since replaceChild finds by first byte.
		splitNode := &node[V]{
			prefix:   child.prefix[:commonLen],
			children: make([]*node[V], 0, 2),
		}

		n.replaceChild(key[0], splitNode)

		child.prefix = child.prefix[commonLen:]
		splitNode.children = append(splitNode.children, child)

		if commonLen == len(key) {
			splitNode.value = value
			splitNode.hasValue = true
		} else {
			splitNode.children = append(splitNode.children, &node[V]{
				prefix:   key[commonLen:],
				value:    value,
				hasValue: true,
			})
		}

		t.size++

		return true
	}
}

// Contains returns true if the key exists in the tree.
func (t *Tree[V]) Contains(key string) bool {
	t.mu.RLock()
	_, ok := t.get(key)
	t.mu.RUnlock()

	return ok
}

// ContainsBytes returns true if the byte slice key exists in the tree.
func (t *Tree[V]) ContainsBytes(key []byte) bool {
	return t.Contains(bytesToString(key))
}

// Get retrieves the value associated with the given key.
// Returns the value and true if found, or the zero value and false otherwise.
func (t *Tree[V]) Get(key string) (V, bool) {
	t.mu.RLock()
	v, ok := t.get(key)
	t.mu.RUnlock()

	return v, ok
}

// GetBytes retrieves the value associated with the given byte slice key.
func (t *Tree[V]) GetBytes(key []byte) (V, bool) {
	return t.Get(bytesToString(key))
}

func (t *Tree[V]) get(key string) (V, bool) {
	n := &t.root

	for {
		if len(key) == 0 {
			if n.hasValue {
				return n.value, true
			}

			var zero V

			return zero, false
		}

		child := n.findChild(key[0])
		if child == nil {
			var zero V

			return zero, false
		}

		if len(key) < len(child.prefix) || key[:len(child.prefix)] != child.prefix {
			var zero V

			return zero, false
		}

		key = key[len(child.prefix):]
		n = child
	}
}

// Delete removes the key from the tree.
// Returns true if the key was found and deleted, false otherwise.
func (t *Tree[V]) Delete(key string) bool {
	t.mu.Lock()
	result := t.delete(key)
	t.mu.Unlock()

	return result
}

// DeleteBytes removes a byte slice key from the tree.
func (t *Tree[V]) DeleteBytes(key []byte) bool {
	return t.Delete(bytesToString(key))
}

func (t *Tree[V]) delete(key string) bool {
	n := &t.root
	var parent *node[V]
	var parentByte byte

	for {
		if len(key) == 0 {
			if !n.hasValue {
				return false
			}

			var zero V

			n.value = zero
			n.hasValue = false
			t.size--

			t.compactNode(n, parent, parentByte)

			return true
		}

		child := n.findChild(key[0])
		if child == nil {
			return false
		}

		if len(key) < len(child.prefix) || key[:len(child.prefix)] != child.prefix {
			return false
		}

		parent = n
		parentByte = key[0]
		key = key[len(child.prefix):]
		n = child
	}
}

// compactNode merges a node with its single child if possible after deletion.
func (t *Tree[V]) compactNode(n, parent *node[V], parentByte byte) {
	if parent == nil || n == &t.root {
		return
	}

	if !n.hasValue && len(n.children) == 0 {
		parent.removeChild(parentByte)

		if parent != &t.root && !parent.hasValue && len(parent.children) == 1 {
			only := parent.children[0]
			parent.prefix += only.prefix
			parent.value = only.value
			parent.hasValue = only.hasValue
			parent.children = only.children
		}

		return
	}

	if !n.hasValue && len(n.children) == 1 {
		only := n.children[0]
		n.prefix += only.prefix
		n.value = only.value
		n.hasValue = only.hasValue
		n.children = only.children
	}
}

// ShortestPrefix returns the key-value pair for the shortest prefix of the given key.
// Returns the matching key, its value, and true if a prefix match was found.
func (t *Tree[V]) ShortestPrefix(key string) (string, V, bool) {
	t.mu.RLock()
	k, v, ok := t.shortestPrefix(key)
	t.mu.RUnlock()

	return k, v, ok
}

// ShortestPrefixBytes returns the shortest prefix match for a byte slice key.
func (t *Tree[V]) ShortestPrefixBytes(key []byte) (string, V, bool) {
	return t.ShortestPrefix(bytesToString(key))
}

func (t *Tree[V]) shortestPrefix(key string) (string, V, bool) {
	origKey := key
	n := &t.root

	if n.hasValue {
		return "", n.value, true
	}

	consumed := 0

	for len(key) > 0 {
		child := n.findChild(key[0])
		if child == nil {
			break
		}

		if len(key) < len(child.prefix) || key[:len(child.prefix)] != child.prefix {
			break
		}

		consumed += len(child.prefix)
		key = key[len(child.prefix):]
		n = child

		if n.hasValue {
			return origKey[:consumed], n.value, true
		}
	}

	var zero V

	return "", zero, false
}

// LongestPrefix returns the key-value pair for the longest prefix of the given key.
// Returns the matching key, its value, and true if a prefix match was found.
func (t *Tree[V]) LongestPrefix(key string) (string, V, bool) {
	t.mu.RLock()
	k, v, ok := t.longestPrefix(key)
	t.mu.RUnlock()

	return k, v, ok
}

// LongestPrefixBytes returns the longest prefix match for a byte slice key.
func (t *Tree[V]) LongestPrefixBytes(key []byte) (string, V, bool) {
	return t.LongestPrefix(bytesToString(key))
}

func (t *Tree[V]) longestPrefix(key string) (string, V, bool) {
	origKey := key
	n := &t.root

	var lastValue V
	var found bool
	var lastConsumed int

	if n.hasValue {
		lastValue = n.value
		found = true
	}

	consumed := 0

	for len(key) > 0 {
		child := n.findChild(key[0])
		if child == nil {
			break
		}

		if len(key) < len(child.prefix) || key[:len(child.prefix)] != child.prefix {
			break
		}

		consumed += len(child.prefix)
		key = key[len(child.prefix):]
		n = child

		if n.hasValue {
			lastValue = n.value
			lastConsumed = consumed
			found = true
		}
	}

	if !found {
		var zero V

		return "", zero, false
	}

	return origKey[:lastConsumed], lastValue, true
}

// PrefixSearch returns all key-value pairs where the key starts with the given prefix.
func (t *Tree[V]) PrefixSearch(prefix string) map[string]V {
	t.mu.RLock()
	result := t.prefixSearch(prefix)
	t.mu.RUnlock()

	return result
}

// PrefixSearchBytes returns all key-value pairs where the key starts with the given byte slice prefix.
func (t *Tree[V]) PrefixSearchBytes(prefix []byte) map[string]V {
	return t.PrefixSearch(bytesToString(prefix))
}

func (t *Tree[V]) prefixSearch(prefix string) map[string]V {
	n := &t.root
	buf := make([]byte, 0, 64)

	for len(prefix) > 0 {
		child := n.findChild(prefix[0])
		if child == nil {
			return nil
		}

		if len(prefix) <= len(child.prefix) {
			if child.prefix[:len(prefix)] != prefix {
				return nil
			}

			buf = append(buf, child.prefix...)

			return t.collectAll(child, buf)
		}

		if prefix[:len(child.prefix)] != child.prefix {
			return nil
		}

		buf = append(buf, child.prefix...)
		prefix = prefix[len(child.prefix):]
		n = child
	}

	return t.collectAll(n, buf)
}

func (t *Tree[V]) collectAll(n *node[V], prefix []byte) map[string]V {
	var stats subtreeStats

	collectStats(n, len(prefix), &stats)

	// Single contiguous buffer for all keys. Pre-sized exactly so
	// append never reallocates, keeping unsafe.String references valid.
	keyBuf := make([]byte, 0, stats.keyBytes)
	results := make(map[string]V, stats.count)

	t.collectInto(n, prefix, &keyBuf, results)

	return results
}

func (t *Tree[V]) collectInto(n *node[V], buf []byte, keyBuf *[]byte, results map[string]V) {
	if n.hasValue {
		start := len(*keyBuf)
		*keyBuf = append(*keyBuf, buf...)
		results[unsafe.String(&(*keyBuf)[start], len(buf))] = n.value
	}

	for _, child := range n.children {
		t.collectInto(child, append(buf, child.prefix...), keyBuf, results)
	}
}

type subtreeStats struct {
	count    int
	keyBytes int
}

func collectStats[V any](n *node[V], prefixLen int, stats *subtreeStats) {
	if n.hasValue {
		stats.count++
		stats.keyBytes += prefixLen
	}

	for _, child := range n.children {
		collectStats(child, prefixLen+len(child.prefix), stats)
	}
}

// WalkPrefix iterates over all key-value pairs where the key starts with the given prefix.
// The callback function is called for each entry. If the callback returns false,
// iteration stops early.
func (t *Tree[V]) WalkPrefix(prefix string, fn func(key string, value V) bool) {
	t.mu.RLock()
	t.walkPrefix(prefix, fn)
	t.mu.RUnlock()
}

// WalkPrefixBytes is like WalkPrefix but accepts a byte slice key.
func (t *Tree[V]) WalkPrefixBytes(prefix []byte, fn func(key string, value V) bool) {
	t.WalkPrefix(bytesToString(prefix), fn)
}

func (t *Tree[V]) walkPrefix(prefix string, fn func(string, V) bool) {
	n := &t.root
	buf := make([]byte, 0, 64)

	for len(prefix) > 0 {
		child := n.findChild(prefix[0])
		if child == nil {
			return
		}

		if len(prefix) <= len(child.prefix) {
			if child.prefix[:len(prefix)] != prefix {
				return
			}

			buf = append(buf, child.prefix...)

			var stats subtreeStats

			collectStats(child, len(buf), &stats)

			keyBuf := make([]byte, 0, stats.keyBytes)
			t.walkZeroCopy(child, buf, &keyBuf, fn)

			return
		}

		if prefix[:len(child.prefix)] != child.prefix {
			return
		}

		buf = append(buf, child.prefix...)
		prefix = prefix[len(child.prefix):]
		n = child
	}

	var stats subtreeStats

	collectStats(n, len(buf), &stats)

	keyBuf := make([]byte, 0, stats.keyBytes)
	t.walkZeroCopy(n, buf, &keyBuf, fn)
}

// HasPrefix returns true if at least one key in the tree starts with the given prefix.
func (t *Tree[V]) HasPrefix(prefix string) bool {
	t.mu.RLock()
	result := t.hasPrefix(prefix)
	t.mu.RUnlock()

	return result
}

// HasPrefixBytes returns true if at least one key starts with the given byte slice prefix.
func (t *Tree[V]) HasPrefixBytes(prefix []byte) bool {
	return t.HasPrefix(bytesToString(prefix))
}

func (t *Tree[V]) hasPrefix(prefix string) bool {
	n := &t.root

	for len(prefix) > 0 {
		child := n.findChild(prefix[0])
		if child == nil {
			return false
		}

		if len(prefix) <= len(child.prefix) {
			return child.prefix[:len(prefix)] == prefix
		}

		if prefix[:len(child.prefix)] != child.prefix {
			return false
		}

		prefix = prefix[len(child.prefix):]
		n = child
	}

	return true
}

// DeletePrefix removes all keys that start with the given prefix.
// Returns the number of deleted entries.
func (t *Tree[V]) DeletePrefix(prefix string) int {
	t.mu.Lock()
	count := t.deletePrefix(prefix)
	t.mu.Unlock()

	return count
}

// DeletePrefixBytes removes all keys starting with the given byte slice prefix.
func (t *Tree[V]) DeletePrefixBytes(prefix []byte) int {
	return t.DeletePrefix(bytesToString(prefix))
}

func (t *Tree[V]) deletePrefix(prefix string) int {
	n := &t.root
	var parent *node[V]

	for len(prefix) > 0 {
		child := n.findChild(prefix[0])
		if child == nil {
			return 0
		}

		if len(prefix) <= len(child.prefix) {
			if child.prefix[:len(prefix)] != prefix {
				return 0
			}

			// Count entries in subtree, then remove the child
			count := countEntries(child)
			n.removeChild(prefix[0])
			t.size -= count

			if parent != nil && !n.hasValue && len(n.children) == 1 && n != &t.root {
				only := n.children[0]
				n.prefix += only.prefix
				n.value = only.value
				n.hasValue = only.hasValue
				n.children = only.children
			}

			return count
		}

		if prefix[:len(child.prefix)] != child.prefix {
			return 0
		}

		parent = n
		prefix = prefix[len(child.prefix):]
		n = child
	}

	// prefix was empty — delete everything under n (which is root)
	count := countEntries(n)

	var zero V

	n.value = zero
	n.hasValue = false
	n.children = nil

	t.size -= count

	return count
}

func countEntries[V any](n *node[V]) int {
	count := 0
	if n.hasValue {
		count = 1
	}

	for _, child := range n.children {
		count += countEntries(child)
	}

	return count
}

// Merge adds all entries from another tree into this tree.
// Existing keys are overwritten with values from the other tree.
func (t *Tree[V]) Merge(other *Tree[V]) {
	other.mu.RLock()
	t.mu.Lock()

	other.walk(&other.root, make([]byte, 0, 64), func(key string, value V) bool {
		t.insert(key, value)
		return true
	})

	t.mu.Unlock()
	other.mu.RUnlock()
}

// Keys returns all keys in the tree.
func (t *Tree[V]) Keys() []string {
	t.mu.RLock()

	keys := make([]string, 0, t.size)
	t.walk(&t.root, make([]byte, 0, 64), func(key string, _ V) bool {
		keys = append(keys, key)
		return true
	})

	t.mu.RUnlock()

	return keys
}

// Values returns all values in the tree.
func (t *Tree[V]) Values() []V {
	t.mu.RLock()

	values := make([]V, 0, t.size)
	t.walk(&t.root, make([]byte, 0, 64), func(_ string, value V) bool {
		values = append(values, value)
		return true
	})

	t.mu.RUnlock()

	return values
}

// Clear removes all entries from the tree.
func (t *Tree[V]) Clear() {
	t.mu.Lock()
	t.root = node[V]{}
	t.size = 0
	t.mu.Unlock()
}

// Walk iterates over all key-value pairs in the tree.
// The callback function is called for each entry. If the callback returns false,
// iteration stops early.
func (t *Tree[V]) Walk(fn func(key string, value V) bool) {
	t.mu.RLock()
	buf := make([]byte, 0, 64)
	t.walk(&t.root, buf, fn)
	t.mu.RUnlock()
}

func (t *Tree[V]) walk(n *node[V], buf []byte, fn func(string, V) bool) bool {
	if n.hasValue {
		if !fn(string(buf), n.value) {
			return false
		}
	}

	for _, child := range n.children {
		if !t.walk(child, append(buf, child.prefix...), fn) {
			return false
		}
	}

	return true
}

func (t *Tree[V]) walkZeroCopy(n *node[V], buf []byte, keyBuf *[]byte, fn func(string, V) bool) bool {
	if n.hasValue {
		start := len(*keyBuf)
		*keyBuf = append(*keyBuf, buf...)

		if !fn(unsafe.String(&(*keyBuf)[start], len(buf)), n.value) {
			return false
		}
	}

	for _, child := range n.children {
		if !t.walkZeroCopy(child, append(buf, child.prefix...), keyBuf, fn) {
			return false
		}
	}

	return true
}

func (n *node[V]) findChild(b byte) *node[V] {
	for _, child := range n.children {
		if child.prefix[0] == b {
			return child
		}
	}

	return nil
}

func (n *node[V]) replaceChild(b byte, replacement *node[V]) {
	for i, child := range n.children {
		if child.prefix[0] == b {
			n.children[i] = replacement

			return
		}
	}
}

func (n *node[V]) removeChild(b byte) {
	for i, child := range n.children {
		if child.prefix[0] == b {
			last := len(n.children) - 1
			n.children[i] = n.children[last]
			n.children[last] = nil
			n.children = n.children[:last]

			return
		}
	}
}

func longestCommonPrefix(a, b string) int {
	maxLen := min(len(a), len(b))

	for i := range maxLen {
		if a[i] != b[i] {
			return i
		}
	}

	return maxLen
}

// bytesToString converts []byte to string without allocation.
func bytesToString(b []byte) string {
	if len(b) == 0 {
		return ""
	}

	return unsafe.String(&b[0], len(b))
}
