package gcra

import (
	"bytes"
	"encoding/gob"
	"errors"
	"math"
	"sync"
	"time"
)

// ErrInvalidData is returned by Import when the data is not a valid
// exported limiter state.
var ErrInvalidData = errors.New("invalid limiter data")

// Caps keeping all TAT arithmetic free of int64 overflow: a key's TAT never
// leads now by more than tau + interval <= 3/4 of math.MaxInt64 nanoseconds.
const (
	maxInterval = int64(math.MaxInt64 / 4)
	maxTau      = int64(math.MaxInt64 / 2)
)

// Limiter implements the Generic Cell Rate Algorithm (GCRA) for rate limiting.
// It is the virtual-scheduling form of the leaky bucket: each key may make up
// to burst requests instantly, then sustain limit requests per period. Unlike
// a fixed window counter, GCRA enforces a smooth rate with no bursts at
// window boundaries and stores only one timestamp per key.
//
// All timestamps are monotonic nanoseconds relative to the limiter's creation
// time, so entries are pointer-free int64 values: 8 bytes per key beside the
// key itself, invisible to the garbage collector, immune to wall-clock jumps.
//
// Properties:
//   - O(1) per operation (map lookup + integer comparison)
//   - O(n) memory where n = number of tracked keys (one int64 per key)
//   - Smooth rate enforcement, no window-edge bursts
//   - Thread-safe
//   - Max tracked keys with oldest-state eviction
type Limiter struct {
	// ponytail: single mutex; shard entries by key hash if cross-core contention matters
	mu              sync.Mutex
	entries         map[string]int64 // key -> theoretical arrival time (TAT), ns since base
	base            time.Time        // monotonic reference for all stored times
	interval        int64            // emission interval in ns: period / limit
	tau             int64            // burst tolerance in ns: (burst - 1) * interval
	maxKeys         int
	cleanupInterval time.Duration
	cleanup         *time.Ticker
	stopOnce        sync.Once
	done            chan struct{}
}

// New creates a new Limiter allowing limit requests per period with the given
// burst capacity (the number of requests a key may make instantly from a cold
// start; burst below 1 is treated as 1). When maxKeys is reached, the key
// whose state drains soonest is evicted.
//
// If cleanupInterval is positive, a background goroutine runs at that interval
// to remove fully drained entries. If zero or negative, no background cleanup
// is performed and drained entries are only removed lazily during eviction.
// Call Stop() to release the background goroutine when done.
func New(limit int, period time.Duration, burst, maxKeys int, cleanupInterval time.Duration) *Limiter {
	if limit <= 0 {
		limit = 1
	}

	if period <= 0 {
		period = time.Second
	}

	if burst < 1 {
		burst = 1
	}

	if maxKeys <= 0 {
		maxKeys = 1000
	}

	interval := int64(period) / int64(limit)
	if interval <= 0 {
		interval = 1
	}

	if interval > maxInterval {
		interval = maxInterval
	}

	tau := maxTau
	if int64(burst-1) <= maxTau/interval {
		tau = int64(burst-1) * interval
	}

	l := &Limiter{
		entries:         make(map[string]int64, maxKeys),
		base:            time.Now(),
		interval:        interval,
		tau:             tau,
		maxKeys:         maxKeys,
		cleanupInterval: cleanupInterval,
		done:            make(chan struct{}),
	}

	if cleanupInterval > 0 {
		l.cleanup = time.NewTicker(cleanupInterval)
		go l.cleanupLoop()
	}

	return l
}

// limiterData is used for gob encoding/decoding. Entry values are absolute
// wall-clock TATs (UnixNano) so state is portable across processes.
type limiterData struct {
	Interval        int64
	Tau             int64
	MaxKeys         int
	CleanupInterval int64
	Entries         map[string]int64
}

// Export serializes the limiter configuration and per-key state for storage
// or transmission. TATs are converted to wall-clock time, so a later Import
// credits keys for the real time elapsed in between.
func (l *Limiter) Export() ([]byte, error) {
	baseWall := l.base.UnixNano()

	l.mu.Lock()
	entries := make(map[string]int64, len(l.entries))
	for key, tat := range l.entries {
		entries[key] = baseWall + tat
	}
	l.mu.Unlock()

	data := limiterData{
		Interval:        l.interval,
		Tau:             l.tau,
		MaxKeys:         l.maxKeys,
		CleanupInterval: int64(l.cleanupInterval),
		Entries:         entries,
	}

	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(data); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Import deserializes a Limiter from exported data. Keys that fully drained
// since the export are dropped; remaining keys keep their pending capacity
// debt relative to the current wall-clock time.
// Call Stop() on the returned Limiter if background cleanup was configured.
func Import(data []byte) (*Limiter, error) {
	var d limiterData
	if err := gob.NewDecoder(bytes.NewBuffer(data)).Decode(&d); err != nil {
		return nil, err
	}

	if d.Interval < 1 || d.Interval > maxInterval || d.Tau < 0 || d.Tau > maxTau || d.MaxKeys < 1 {
		return nil, ErrInvalidData
	}

	l := &Limiter{
		entries:         make(map[string]int64, d.MaxKeys),
		base:            time.Now(),
		interval:        d.Interval,
		tau:             d.Tau,
		maxKeys:         d.MaxKeys,
		cleanupInterval: time.Duration(d.CleanupInterval),
		done:            make(chan struct{}),
	}

	baseWall := l.base.UnixNano()
	maxLead := d.Tau + d.Interval

	for key, wallTat := range d.Entries {
		tat := wallTat - baseWall
		if tat <= 0 {
			continue // drained while offline
		}

		// A valid TAT never leads now by more than tau + interval; clamp
		// anything larger so corrupted data cannot break overflow safety.
		if tat > maxLead {
			tat = maxLead
		}

		l.entries[key] = tat
	}

	if l.cleanupInterval > 0 {
		l.cleanup = time.NewTicker(l.cleanupInterval)
		go l.cleanupLoop()
	}

	return l, nil
}

// Stop stops the background cleanup goroutine if one was started.
// It is safe to call multiple times.
func (l *Limiter) Stop() {
	l.stopOnce.Do(func() {
		if l.cleanup != nil {
			l.cleanup.Stop()
		}
		close(l.done)
	})
}

// now returns the current time in monotonic nanoseconds since l.base.
func (l *Limiter) now() int64 {
	return int64(time.Since(l.base))
}

func (l *Limiter) cleanupLoop() {
	for {
		select {
		case <-l.done:
			return
		case <-l.cleanup.C:
			l.removeDrained(l.now())
		}
	}
}

// removeDrained deletes entries whose TAT is in the past: such a key has
// regained its full burst capacity and is indistinguishable from an untracked one.
func (l *Limiter) removeDrained(now int64) {
	l.mu.Lock()
	defer l.mu.Unlock()

	for key, tat := range l.entries {
		if tat <= now {
			delete(l.entries, key)
		}
	}
}

// Allow reports whether a single request for the key conforms to the rate
// limit, consuming capacity if it does. Non-conforming requests consume nothing.
func (l *Limiter) Allow(key string) bool {
	return l.allowN(key, 1, l.now())
}

// AllowN reports whether n requests for the key conform to the rate limit as
// a batch, consuming capacity for all n if they do. The batch is all-or-nothing:
// if n exceeds the currently available capacity, nothing is consumed.
// n less than 1 is always allowed and consumes nothing.
func (l *Limiter) AllowN(key string, n int) bool {
	return l.allowN(key, n, l.now())
}

func (l *Limiter) allowN(key string, n int, now int64) bool {
	if n < 1 {
		return true
	}

	// A batch larger than the burst capacity can never conform; rejecting it
	// up front also keeps the arithmetic below free of overflow.
	if int64(n-1) > l.tau/l.interval {
		return false
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	tat, ok := l.entries[key]
	if !ok || tat < now {
		tat = now
	}

	// The batch conforms if its last cell does: its theoretical arrival
	// time tat+(n-1)*interval may lag now by at most tau.
	if now < tat+int64(n-1)*l.interval-l.tau {
		return false
	}

	if !ok {
		l.evictIfNeeded(now)
	}

	l.entries[key] = tat + int64(n)*l.interval

	return true
}

// RetryAfter returns how long until a single request for the key would be
// allowed. Returns 0 if a request would be allowed now.
func (l *Limiter) RetryAfter(key string) time.Duration {
	return l.retryAfter(key, l.now())
}

func (l *Limiter) retryAfter(key string, now int64) time.Duration {
	l.mu.Lock()
	defer l.mu.Unlock()

	tat, ok := l.entries[key]
	if !ok {
		return 0
	}

	if d := tat - l.tau - now; d > 0 {
		return time.Duration(d)
	}

	return 0
}

// Reset clears all tracked keys.
func (l *Limiter) Reset() {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.entries = make(map[string]int64, l.maxKeys)
}

// Len returns the number of currently tracked keys (including drained but not
// yet cleaned up).
func (l *Limiter) Len() int {
	l.mu.Lock()
	defer l.mu.Unlock()

	return len(l.entries)
}

// evictIfNeeded removes drained entries first, then evicts the key with the
// earliest TAT if still at capacity. Caller must hold l.mu.
func (l *Limiter) evictIfNeeded(now int64) {
	if len(l.entries) < l.maxKeys {
		return
	}

	for key, tat := range l.entries {
		if tat <= now {
			delete(l.entries, key)
		}
	}

	if len(l.entries) < l.maxKeys {
		return
	}

	oldestKey := ""
	oldestTAT := int64(math.MaxInt64)

	for key, tat := range l.entries {
		if tat < oldestTAT {
			oldestKey = key
			oldestTAT = tat
		}
	}

	delete(l.entries, oldestKey)
}
