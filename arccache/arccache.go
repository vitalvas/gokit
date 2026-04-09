package arccache

import (
	"sync"
	"time"
)

// ARC implements the Adaptive Replacement Cache algorithm.
//
// ARC maintains four internal lists to adaptively balance between recency
// and frequency:
//   - T1: recently accessed items (seen once)
//   - T2: frequently accessed items (seen at least twice)
//   - B1: ghost entries evicted from T1 (keys only, no values)
//   - B2: ghost entries evicted from T2 (keys only, no values)
//
// The parameter p (target size for T1) adapts based on cache misses:
//   - Miss in B1 (recently evicted): increase p (favor recency)
//   - Miss in B2 (frequently evicted): decrease p (favor frequency)
//
// Reference: "ARC: A Self-Tuning, Low Overhead Replacement Cache"
// (Megiddo & Modha, FAST 2003).
//
// Properties:
//   - O(1) per operation (hash map lookups + doubly linked list operations)
//   - Self-tuning: automatically adapts to workload
//   - Scan-resistant: does not degrade under sequential access patterns
//   - Thread-safe: all operations protected by mutex
//   - Optional TTL per entry
//   - Optional byte-size tracking with eviction
type ARC[K comparable, V any] struct {
	mu sync.Mutex

	// ARC lists
	t1 *list[K, V] // recent cache entries
	t2 *list[K, V] // frequent cache entries
	b1 *list[K, V] // ghost entries evicted from t1
	b2 *list[K, V] // ghost entries evicted from t2

	// index for O(1) lookup
	items map[K]*element[K, V]

	// ARC adaptation parameter: target size for t1
	p int

	// capacity
	maxItems int

	// byte tracking
	maxBytes int
	bytes    int
	sizeFunc func(K, V) int

	// TTL
	defaultTTL time.Duration

	// eviction callback
	onEvict func(K, V)

	// background cleanup
	cleanup  *time.Ticker
	stopOnce sync.Once
	done     chan struct{}
}

// Options configures ARC cache behavior.
type Options[K comparable, V any] struct {
	// MaxItems is the maximum number of items in the cache.
	// Zero or negative defaults to 1000.
	MaxItems int

	// MaxBytes is the maximum byte size of the cache.
	// Zero means no byte limit (only MaxItems is enforced).
	// Requires SizeFunc to be set; ignored if SizeFunc is nil.
	MaxBytes int

	// SizeFunc returns the byte size of a key-value pair.
	// Required when MaxBytes is set. If nil, MaxBytes is ignored.
	SizeFunc func(K, V) int

	// DefaultTTL is the default time-to-live for entries.
	// Zero means no expiration (entries live until evicted).
	DefaultTTL time.Duration

	// OnEvict is called when an entry is evicted from the cache.
	// Called with the mutex held; must not call back into the cache.
	OnEvict func(K, V)

	// CleanupInterval controls background expired entry removal.
	// Zero or negative means lazy cleanup only (on access and eviction).
	CleanupInterval time.Duration
}

// New creates a new ARC cache with the specified maximum number of items.
func New[K comparable, V any](maxItems int) *ARC[K, V] {
	return NewWithOptions(Options[K, V]{MaxItems: maxItems})
}

// NewWithOptions creates a new ARC cache with the specified options.
func NewWithOptions[K comparable, V any](opts Options[K, V]) *ARC[K, V] {
	if opts.MaxItems <= 0 {
		opts.MaxItems = 1000
	}

	// Ignore MaxBytes if no SizeFunc provided
	if opts.SizeFunc == nil {
		opts.MaxBytes = 0
	}

	c := &ARC[K, V]{
		t1:         newList[K, V](),
		t2:         newList[K, V](),
		b1:         newList[K, V](),
		b2:         newList[K, V](),
		items:      make(map[K]*element[K, V], opts.MaxItems),
		maxItems:   opts.MaxItems,
		maxBytes:   opts.MaxBytes,
		sizeFunc:   opts.SizeFunc,
		defaultTTL: opts.DefaultTTL,
		onEvict:    opts.OnEvict,
		done:       make(chan struct{}),
	}

	if opts.CleanupInterval > 0 {
		c.cleanup = time.NewTicker(opts.CleanupInterval)
		go c.cleanupLoop()
	}

	return c
}

// Stop stops the background cleanup goroutine if one was started.
// Safe to call multiple times.
func (c *ARC[K, V]) Stop() {
	c.stopOnce.Do(func() {
		if c.cleanup != nil {
			c.cleanup.Stop()
		}
		close(c.done)
	})
}

func (c *ARC[K, V]) cleanupLoop() {
	for {
		select {
		case <-c.done:
			return
		case <-c.cleanup.C:
			c.removeExpired()
		}
	}
}

func (c *ARC[K, V]) removeExpired() {
	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()

	c.removeExpiredFrom(c.t1, now)
	c.removeExpiredFrom(c.t2, now)
}

func (c *ARC[K, V]) removeExpiredFrom(l *list[K, V], now time.Time) {
	sentinel := &l.root
	e := l.head()
	for e != nil && e != sentinel {
		next := e.next
		if !e.expiresAt.IsZero() && now.After(e.expiresAt) {
			l.remove(e)
			delete(c.items, e.key)
			c.updateBytes(-c.entrySize(e))
			c.notifyEvict(e)
		}
		e = next
	}
}

// Get retrieves a value from the cache. Returns the value and true if found
// and not expired, or the zero value and false otherwise.
func (c *ARC[K, V]) Get(key K) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	e, ok := c.items[key]
	if !ok {
		var zero V
		return zero, false
	}

	// Ghost entries have no value
	if e.ghost {
		var zero V
		return zero, false
	}

	// Check expiration
	if c.isExpired(e) {
		c.removeEntry(e)
		var zero V
		return zero, false
	}

	// Promote to T2 on second access, or move to front within T2
	switch e.list {
	case c.t1:
		c.t1.remove(e)
		e.list = c.t2
		c.t2.pushFront(e)
	case c.t2:
		c.t2.moveToFront(e)
	}

	return e.value, true
}

// Set adds or updates a key-value pair in the cache.
// If ttl is zero, the default TTL is used. If both are zero, the entry
// does not expire.
func (c *ARC[K, V]) Set(key K, value V, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if ttl == 0 {
		ttl = c.defaultTTL
	}

	var expiresAt time.Time
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl)
	}

	if e, ok := c.items[key]; ok {
		if !e.ghost {
			// Update existing entry
			oldSize := c.entrySize(e)
			e.value = value
			e.expiresAt = expiresAt
			c.updateBytes(c.entrySize(e) - oldSize)

			switch e.list {
			case c.t1:
				c.t1.remove(e)
				e.list = c.t2
				c.t2.pushFront(e)
			case c.t2:
				c.t2.moveToFront(e)
			}

			c.enforceBytes()
			return
		}

		// Ghost hit: adapt and reinsert
		c.handleGhostHit(e, value, expiresAt)
		return
	}

	// Complete miss: add new entry
	c.handleMiss(key, value, expiresAt)
}

// Delete removes an entry from the cache.
func (c *ARC[K, V]) Delete(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()

	e, ok := c.items[key]
	if !ok {
		return
	}

	if e.ghost {
		e.list.remove(e)
		delete(c.items, key)
		return
	}

	c.removeEntry(e)
}

// Len returns the number of non-ghost entries in the cache.
func (c *ARC[K, V]) Len() int {
	c.mu.Lock()
	n := c.t1.len() + c.t2.len()
	c.mu.Unlock()

	return n
}

// Bytes returns the current tracked byte usage.
// Returns 0 if no SizeFunc was configured.
func (c *ARC[K, V]) Bytes() int {
	c.mu.Lock()
	n := c.bytes
	c.mu.Unlock()

	return n
}

// Clear removes all entries from the cache and resets the adaptation parameter.
func (c *ARC[K, V]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.t1.clear()
	c.t2.clear()
	c.b1.clear()
	c.b2.clear()
	c.items = make(map[K]*element[K, V], c.maxItems)
	c.p = 0
	c.bytes = 0
}

// handleGhostHit processes a hit on a ghost entry (B1 or B2).
// Caller must hold c.mu.
func (c *ARC[K, V]) handleGhostHit(e *element[K, V], value V, expiresAt time.Time) {
	switch e.list {
	case c.b1:
		// Ghost hit in B1: increase p (favor recency)
		delta := 1
		if c.b2.len() > c.b1.len() {
			delta = c.b2.len() / c.b1.len()
		}
		c.p = min(c.p+delta, c.maxItems)

		c.replace(false)
		c.b1.remove(e)

	case c.b2:
		// Ghost hit in B2: decrease p (favor frequency)
		delta := 1
		if c.b1.len() > c.b2.len() {
			delta = c.b1.len() / c.b2.len()
		}
		c.p = max(c.p-delta, 0)

		c.replace(true)
		c.b2.remove(e)

	default:
		return
	}

	e.value = value
	e.expiresAt = expiresAt
	e.ghost = false
	e.list = c.t2
	c.t2.pushFront(e)
	c.updateBytes(c.entrySize(e))

	c.enforceBytes()
}

// handleMiss processes a complete cache miss.
// Caller must hold c.mu.
func (c *ARC[K, V]) handleMiss(key K, value V, expiresAt time.Time) {
	totalCache := c.t1.len() + c.t2.len()
	totalAll := totalCache + c.b1.len() + c.b2.len()

	if totalCache >= c.maxItems {
		// Cache is full, need to replace
		if c.t1.len() < c.maxItems {
			// Remove from ghost lists if total exceeds 2*maxItems
			if totalAll >= 2*c.maxItems {
				if c.b2.len() > 0 {
					c.removeGhost(c.b2)
				} else if c.b1.len() > 0 {
					c.removeGhost(c.b1)
				}
			}
			c.replace(false)
		} else {
			// T1 == maxItems (no T2 entries), evict from T1 to B1
			if e := c.t1.tail(); e != nil {
				c.t1.remove(e)
				c.updateBytes(-c.entrySize(e))
				c.notifyEvict(e)
				// Trim B1 if total would exceed 2*maxItems
				if c.b1.len()+c.b2.len() >= c.maxItems {
					c.removeGhost(c.b1)
				}
				c.makeGhost(e, c.b1)
			}
		}
	} else if totalAll >= c.maxItems {
		// Ghost lists pushing us over, trim
		if totalAll >= 2*c.maxItems && c.b2.len() > 0 {
			c.removeGhost(c.b2)
		} else if c.b1.len() > 0 {
			c.removeGhost(c.b1)
		}
	}

	e := &element[K, V]{
		key:       key,
		value:     value,
		expiresAt: expiresAt,
		list:      c.t1,
	}
	c.t1.pushFront(e)
	c.items[key] = e
	c.updateBytes(c.entrySize(e))

	c.enforceBytes()
}

// replace evicts one entry from either T1 or T2 to make room.
// The evicted entry becomes a ghost in B1 or B2.
// b2hit indicates whether the caller is handling a B2 ghost hit.
// Caller must hold c.mu.
func (c *ARC[K, V]) replace(b2hit bool) {
	t1Len := c.t1.len()
	if t1Len == 0 && c.t2.len() == 0 {
		return
	}

	// Evict from T1 if T1 is above target, or from T2 otherwise.
	// On a B2 hit, we use strict > to slightly favor keeping T1 entries.
	evictFromT1 := t1Len > 0 && (t1Len > c.p || (t1Len == c.p && b2hit) || c.t2.len() == 0)

	if evictFromT1 {
		if e := c.t1.tail(); e != nil {
			c.t1.remove(e)
			c.updateBytes(-c.entrySize(e))
			c.notifyEvict(e)
			c.makeGhost(e, c.b1)
		}
	} else {
		if e := c.t2.tail(); e != nil {
			c.t2.remove(e)
			c.updateBytes(-c.entrySize(e))
			c.notifyEvict(e)
			c.makeGhost(e, c.b2)
		}
	}
}

// enforceBytes evicts entries until byte usage is within MaxBytes.
// Caller must hold c.mu.
func (c *ARC[K, V]) enforceBytes() {
	if c.maxBytes <= 0 || c.sizeFunc == nil {
		return
	}

	for c.bytes > c.maxBytes && (c.t1.len()+c.t2.len()) > 0 {
		// Evict using the same T1/T2 preference as replace
		t1Len := c.t1.len()
		evictFromT1 := t1Len > 0 && (t1Len > c.p || c.t2.len() == 0)

		if evictFromT1 {
			if e := c.t1.tail(); e != nil {
				c.t1.remove(e)
				c.updateBytes(-c.entrySize(e))
				c.notifyEvict(e)
				c.makeGhost(e, c.b1)
			}
		} else {
			if e := c.t2.tail(); e != nil {
				c.t2.remove(e)
				c.updateBytes(-c.entrySize(e))
				c.notifyEvict(e)
				c.makeGhost(e, c.b2)
			}
		}
	}
}

// makeGhost converts an entry to a ghost (key-only) in the specified ghost list.
// Caller must hold c.mu.
func (c *ARC[K, V]) makeGhost(e *element[K, V], ghostList *list[K, V]) {
	var zero V
	e.value = zero
	e.ghost = true
	e.expiresAt = time.Time{}
	e.list = ghostList
	ghostList.pushFront(e)
}

// removeGhost removes the oldest ghost from a ghost list.
// Caller must hold c.mu.
func (c *ARC[K, V]) removeGhost(ghostList *list[K, V]) {
	if e := ghostList.tail(); e != nil {
		ghostList.remove(e)
		delete(c.items, e.key)
	}
}

// removeEntry removes a non-ghost entry and deletes it from the index.
// Caller must hold c.mu.
func (c *ARC[K, V]) removeEntry(e *element[K, V]) {
	e.list.remove(e)
	delete(c.items, e.key)
	c.updateBytes(-c.entrySize(e))
	c.notifyEvict(e)
}

func (c *ARC[K, V]) isExpired(e *element[K, V]) bool {
	return !e.expiresAt.IsZero() && time.Now().After(e.expiresAt)
}

func (c *ARC[K, V]) entrySize(e *element[K, V]) int {
	if c.sizeFunc == nil {
		return 0
	}
	return c.sizeFunc(e.key, e.value)
}

func (c *ARC[K, V]) updateBytes(delta int) {
	if c.sizeFunc == nil {
		return
	}
	c.bytes += delta
	if c.bytes < 0 {
		c.bytes = 0
	}
}

func (c *ARC[K, V]) notifyEvict(e *element[K, V]) {
	if c.onEvict != nil {
		c.onEvict(e.key, e.value)
	}
}
