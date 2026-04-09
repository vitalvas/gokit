package fixedwindow

import (
	"sync"
	"time"
)

// Counter implements the Fixed Window Counter algorithm for rate limiting.
// It tracks event counts per key within fixed time windows. When a key exceeds
// its limit (via Allow), it is locked out for the remainder of the window.
// Counters reset automatically when the window expires.
//
// Properties:
//   - O(1) per operation (map lookup + counter increment)
//   - O(n) memory where n = number of tracked keys
//   - Per-key independent windows (start on first access, not wall-clock aligned)
//   - Hard lockout on exceed (all subsequent Allow calls denied until window reset)
//   - Thread-safe
//   - Max tracked keys with oldest-window eviction
type Counter struct {
	mu       sync.Mutex
	entries  map[string]*entry
	window   time.Duration
	maxKeys  int
	cleanup  *time.Ticker
	stopOnce sync.Once
	done     chan struct{}
}

type entry struct {
	count     int
	lockedOut bool
	windowEnd time.Time
}

// New creates a new Counter with the specified window duration and maximum number
// of tracked keys. When maxKeys is reached, the key with the oldest window is evicted.
//
// If cleanupInterval is positive, a background goroutine runs at that interval
// to remove expired entries. If zero or negative, no background cleanup is performed
// and expired entries are only cleaned up lazily during eviction.
// Call Stop() to release the background goroutine when done.
func New(window time.Duration, maxKeys int, cleanupInterval time.Duration) *Counter {
	if window <= 0 {
		window = time.Minute
	}

	if maxKeys <= 0 {
		maxKeys = 1000
	}

	c := &Counter{
		entries: make(map[string]*entry, maxKeys),
		window:  window,
		maxKeys: maxKeys,
		done:    make(chan struct{}),
	}

	if cleanupInterval > 0 {
		c.cleanup = time.NewTicker(cleanupInterval)
		go c.cleanupLoop()
	}

	return c
}

// Stop stops the background cleanup goroutine if one was started.
// It is safe to call multiple times.
func (c *Counter) Stop() {
	c.stopOnce.Do(func() {
		if c.cleanup != nil {
			c.cleanup.Stop()
		}
		close(c.done)
	})
}

func (c *Counter) cleanupLoop() {
	for {
		select {
		case <-c.done:
			return
		case <-c.cleanup.C:
			c.removeExpired()
		}
	}
}

func (c *Counter) removeExpired() {
	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()

	for key, e := range c.entries {
		if now.After(e.windowEnd) {
			delete(c.entries, key)
		}
	}
}

// Allow checks whether the key is within its rate limit and consumes one unit.
// It calls Add(key, 1) internally and returns true if the key's count
// has not exceeded the limit.
//
// Once a key exceeds the limit, it is locked out and all subsequent calls
// to Allow return false until the window resets.
func (c *Counter) Allow(key string, limit int) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	e := c.getOrCreate(key)

	e.count++

	if !e.lockedOut && e.count > limit {
		e.lockedOut = true
	}

	return !e.lockedOut
}

// Add increments the counter for the given key by delta and returns the new count.
// The counter is always incremented regardless of lockout status.
// Delta can be any positive value; zero or negative deltas are ignored and
// the current count is returned.
func (c *Counter) Add(key string, delta int) int {
	if delta <= 0 {
		return c.Count(key)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	e := c.getOrCreate(key)
	e.count += delta

	return e.count
}

// Count returns the current counter value for the key.
// Returns 0 if the key is not tracked or its window has expired.
func (c *Counter) Count(key string) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	e, ok := c.entries[key]
	if !ok {
		return 0
	}

	if time.Now().After(e.windowEnd) {
		delete(c.entries, key)
		return 0
	}

	return e.count
}

// IsLockedOut returns whether the key is currently locked out.
// Returns false if the key is not tracked or its window has expired.
func (c *Counter) IsLockedOut(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	e, ok := c.entries[key]
	if !ok {
		return false
	}

	if time.Now().After(e.windowEnd) {
		delete(c.entries, key)
		return false
	}

	return e.lockedOut
}

// Reset clears all tracked keys and their counters.
func (c *Counter) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]*entry, c.maxKeys)
}

// Len returns the number of currently tracked keys (including expired but not yet cleaned up).
func (c *Counter) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return len(c.entries)
}

// getOrCreate returns the entry for the key, creating it if it doesn't exist
// or resetting it if its window has expired. Caller must hold c.mu.
func (c *Counter) getOrCreate(key string) *entry {
	now := time.Now()

	if e, ok := c.entries[key]; ok {
		if now.After(e.windowEnd) {
			e.count = 0
			e.lockedOut = false
			e.windowEnd = now.Add(c.window)
		}
		return e
	}

	c.evictIfNeeded(now)

	e := &entry{
		windowEnd: now.Add(c.window),
	}
	c.entries[key] = e

	return e
}

// evictIfNeeded removes expired entries first, then evicts the oldest window
// if still at capacity. Caller must hold c.mu.
func (c *Counter) evictIfNeeded(now time.Time) {
	if len(c.entries) < c.maxKeys {
		return
	}

	// First pass: remove expired entries
	for key, e := range c.entries {
		if now.After(e.windowEnd) {
			delete(c.entries, key)
		}
	}

	if len(c.entries) < c.maxKeys {
		return
	}

	// Evict the key with the oldest (earliest) window end
	var oldestKey string
	var oldestEnd time.Time

	for key, e := range c.entries {
		if oldestKey == "" || e.windowEnd.Before(oldestEnd) {
			oldestKey = key
			oldestEnd = e.windowEnd
		}
	}

	delete(c.entries, oldestKey)
}
