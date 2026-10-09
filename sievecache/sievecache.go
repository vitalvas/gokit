// Package sievecache implements the SIEVE cache eviction algorithm.
//
// Reference: "SIEVE is Simpler than LRU: an Efficient Turn-Key Eviction
// Algorithm for Web Caches" (Zhang et al., NSDI 2024).
package sievecache

import (
	"sync"
	"sync/atomic"
	"time"
)

// Cache is a SIEVE cache: a FIFO queue with one visited bit per entry and a
// sweeping eviction hand. Unlike LRU or ARC, a cache hit does not move the
// entry - it only sets the visited bit - so hits take a shared read lock and
// scale across cores. New entries that are never accessed again are evicted
// quickly (quick demotion), which keeps one-hit wonders from polluting the
// cache.
//
// Properties:
//   - O(1) per operation (amortized for eviction sweeps)
//   - Hits never mutate the queue: read lock + one atomic bit
//   - Strong hit ratios on skewed (Zipfian) workloads
//   - Not scan-resistant: poor fit for loop- or scan-heavy access patterns
//   - Optional lazy TTL per entry (expired entries are evicted first)
//   - Thread-safe
type Cache[K comparable, V any] struct {
	mu       sync.RWMutex
	items    map[K]*entry[K, V]
	head     *entry[K, V] // newest
	tail     *entry[K, V] // oldest
	hand     *entry[K, V] // eviction hand, sweeps tail to head
	maxItems int
	onEvict  func(K, V)
}

// entry is a node in the queue. next points toward the tail (older), prev
// toward the head (newer). visited is atomic so concurrent Gets under the
// shared read lock can set it.
type entry[K comparable, V any] struct {
	key       K
	value     V
	expiresAt int64 // unix nanoseconds, 0 = no expiry
	visited   atomic.Bool
	prev      *entry[K, V]
	next      *entry[K, V]
}

// New creates a new SIEVE cache holding at most maxItems entries.
// A maxItems below 1 defaults to 1000.
func New[K comparable, V any](maxItems int) *Cache[K, V] {
	if maxItems < 1 {
		maxItems = 1000
	}

	return &Cache[K, V]{
		items:    make(map[K]*entry[K, V], maxItems),
		maxItems: maxItems,
	}
}

// OnEvict sets a callback invoked whenever an entry is evicted to make
// room. The callback runs after the cache lock is released, so it may
// safely call back into the cache. Delete and Clear do not trigger it.
// Pass nil to disable.
func (c *Cache[K, V]) OnEvict(fn func(key K, value V)) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.onEvict = fn
}

// Get retrieves a value from the cache. Returns the value and true if found
// and not expired, or the zero value and false otherwise.
func (c *Cache[K, V]) Get(key K) (V, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	e, ok := c.items[key]
	if !ok || c.expired(e, time.Now().UnixNano()) {
		var zero V
		return zero, false
	}

	e.visited.Store(true)

	return e.value, true
}

// Peek retrieves a value without marking the entry visited, leaving its
// eviction priority unchanged.
func (c *Cache[K, V]) Peek(key K) (V, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	e, ok := c.items[key]
	if !ok || c.expired(e, time.Now().UnixNano()) {
		var zero V
		return zero, false
	}

	return e.value, true
}

// Set adds or updates a key-value pair. A positive ttl makes the entry
// expire after that duration; zero or negative means no expiry. Updating an
// existing key counts as an access.
func (c *Cache[K, V]) Set(key K, value V, ttl time.Duration) {
	expiresAt := expiry(ttl)

	c.mu.Lock()
	evictedKey, evictedValue, evicted, onEvict := c.set(key, value, expiresAt)
	c.mu.Unlock()

	if evicted && onEvict != nil {
		onEvict(evictedKey, evictedValue)
	}
}

// GetOrSet returns the existing value for key if present and not expired,
// marking it visited; otherwise it stores value with ttl and returns it.
// The boolean reports whether an existing value was found.
func (c *Cache[K, V]) GetOrSet(key K, value V, ttl time.Duration) (V, bool) {
	if v, ok := c.Get(key); ok {
		return v, true
	}

	expiresAt := expiry(ttl)

	c.mu.Lock()

	// Recheck: another goroutine may have stored between the locks.
	if e, ok := c.items[key]; ok && !c.expired(e, time.Now().UnixNano()) {
		v := e.value
		e.visited.Store(true)
		c.mu.Unlock()

		return v, true
	}

	evictedKey, evictedValue, evicted, onEvict := c.set(key, value, expiresAt)
	c.mu.Unlock()

	if evicted && onEvict != nil {
		onEvict(evictedKey, evictedValue)
	}

	return value, false
}

// set stores a new or updated entry, evicting (and recycling the node of)
// one entry when at capacity. It reports what was evicted so the caller can
// run the callback outside the lock. Caller must hold the write lock.
func (c *Cache[K, V]) set(key K, value V, expiresAt int64) (evictedKey K, evictedValue V, evicted bool, onEvict func(K, V)) {
	if e, ok := c.items[key]; ok {
		e.value = value
		e.expiresAt = expiresAt
		e.visited.Store(true)
		return evictedKey, evictedValue, false, nil
	}

	var e *entry[K, V]

	if len(c.items) >= c.maxItems {
		e = c.evict()
		evictedKey, evictedValue, evicted = e.key, e.value, true

		e.key = key
		e.value = value
		e.expiresAt = expiresAt
		e.visited.Store(false)
		e.prev = nil
	} else {
		e = &entry[K, V]{key: key, value: value, expiresAt: expiresAt}
	}

	e.next = c.head
	if c.head != nil {
		c.head.prev = e
	}
	c.head = e
	if c.tail == nil {
		c.tail = e
	}

	c.items[key] = e

	return evictedKey, evictedValue, evicted, c.onEvict
}

// Delete removes an entry from the cache.
func (c *Cache[K, V]) Delete(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if e, ok := c.items[key]; ok {
		c.remove(e)
	}
}

// Len returns the number of entries in the cache, including expired entries
// that have not yet been evicted.
func (c *Cache[K, V]) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.items)
}

// Clear removes all entries from the cache.
func (c *Cache[K, V]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[K]*entry[K, V], c.maxItems)
	c.head = nil
	c.tail = nil
	c.hand = nil
}

// evict removes one entry and returns its unlinked node for reuse. The hand
// resumes where the previous sweep stopped, moving from tail toward head:
// visited entries get their bit cleared and survive; the first unvisited or
// expired entry is evicted. Caller must hold the write lock and guarantee
// the cache is non-empty.
func (c *Cache[K, V]) evict() *entry[K, V] {
	now := time.Now().UnixNano()

	hand := c.hand
	if hand == nil {
		hand = c.tail
	}

	for hand.visited.Load() && !c.expired(hand, now) {
		hand.visited.Store(false)

		hand = hand.prev
		if hand == nil {
			hand = c.tail
		}
	}

	// The next sweep resumes from the neighbor of the evicted entry.
	c.hand = hand.prev
	c.remove(hand)

	return hand
}

// remove unlinks an entry from the queue and the index, advancing the hand
// if it points at the removed entry. Caller must hold the write lock.
func (c *Cache[K, V]) remove(e *entry[K, V]) {
	if c.hand == e {
		c.hand = e.prev
	}

	if e.prev != nil {
		e.prev.next = e.next
	} else {
		c.head = e.next
	}

	if e.next != nil {
		e.next.prev = e.prev
	} else {
		c.tail = e.prev
	}

	delete(c.items, e.key)
}

func (c *Cache[K, V]) expired(e *entry[K, V], now int64) bool {
	return e.expiresAt != 0 && now > e.expiresAt
}

// expiry converts a ttl to an absolute unix-nanosecond deadline, 0 for none.
func expiry(ttl time.Duration) int64 {
	if ttl <= 0 {
		return 0
	}

	return time.Now().Add(ttl).UnixNano()
}
