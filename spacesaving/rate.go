package spacesaving

import (
	"bytes"
	"encoding/gob"
	"math"
	"sort"
	"sync"
	"time"
)

// RateTracker implements Space-Saving with exponential decay for tracking rates.
// Instead of counting occurrences, it tracks events per second with time-based decay.
//
// Use cases:
//   - Request rate tracking (req/sec)
//   - Network bandwidth monitoring (bytes/sec)
//   - Real-time trending (recent activity matters more)
//   - Adaptive load balancing
//
// Properties:
//   - Exponential decay: Recent events weighted more heavily
//   - Rate measurement: Events per second instead of raw counts
//   - Thread-safe: All operations are protected by mutex
//   - Constant memory: O(k) where k is the number of counters
type RateTracker struct {
	mu        sync.RWMutex
	counters  map[string]*rateCounter
	minHeap   *rateMinHeap
	capacity  int
	halfLife  time.Duration
	decayRate float64
}

// rateCounter represents a rate counter with exponential decay.
type rateCounter struct {
	Item       string
	Rate       float64   // Events per second
	ErrorRate  float64   // Error bound on rate
	LastUpdate time.Time // Last time this counter was updated
	index      int       // Index in the min-heap
}

// NewRateTracker creates a new RateTracker with the specified capacity and half-life.
//
// The half-life determines how quickly old events decay:
//   - Short half-life (1s-10s): Tracks very recent trends
//   - Medium half-life (30s-60s): Balanced recency and stability
//   - Long half-life (5m-15m): Stable long-term rates
//
// Typical configurations:
//   - Real-time trending: capacity=100, halfLife=10s
//   - Request monitoring: capacity=500, halfLife=60s
//   - Network analysis: capacity=1000, halfLife=5m
func NewRateTracker(capacity int, halfLife time.Duration) *RateTracker {
	if capacity <= 0 {
		capacity = 100
	}
	if halfLife <= 0 {
		halfLife = 60 * time.Second
	}

	return &RateTracker{
		counters:  make(map[string]*rateCounter, capacity),
		minHeap:   newRateMinHeap(capacity),
		capacity:  capacity,
		halfLife:  halfLife,
		decayRate: math.Ln2 / halfLife.Seconds(),
	}
}

// Touch records an event for the item at the current time.
// Returns the current rate (events per second) after the update.
func (rt *RateTracker) Touch(item string) float64 {
	return rt.TouchAt(item, time.Now())
}

// TouchAt records an event for the item at the specified time.
// Useful for processing historical data or testing.
func (rt *RateTracker) TouchAt(item string, t time.Time) float64 {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	// If item already tracked, update its rate
	if rc, exists := rt.counters[item]; exists {
		rt.updateRate(rc, t, 1.0)
		rt.minHeap.fix(rc.index)
		return rc.Rate
	}

	// If we haven't reached capacity, add new counter
	if len(rt.counters) < rt.capacity {
		rc := &rateCounter{
			Item:       item,
			Rate:       1.0,
			ErrorRate:  0,
			LastUpdate: t,
		}
		rt.counters[item] = rc
		rt.minHeap.push(rc)
		return 1.0
	}

	// Replace minimum counter
	minCounter := rt.minHeap.min()
	rt.decay(minCounter, t)

	delete(rt.counters, minCounter.Item)

	minCounter.Item = item
	minCounter.ErrorRate = minCounter.Rate
	minCounter.Rate += 1.0
	minCounter.LastUpdate = t

	rt.counters[item] = minCounter
	rt.minHeap.fix(0)

	return minCounter.Rate
}

// updateRate applies exponential decay and adds the new event.
func (rt *RateTracker) updateRate(rc *rateCounter, t time.Time, increment float64) {
	rt.decay(rc, t)
	rc.Rate += increment
	rc.LastUpdate = t
}

// decay applies exponential decay based on time elapsed.
func (rt *RateTracker) decay(rc *rateCounter, t time.Time) {
	if rc.LastUpdate.IsZero() {
		return
	}

	elapsed := t.Sub(rc.LastUpdate).Seconds()
	if elapsed <= 0 {
		return
	}

	decayFactor := math.Exp(-rt.decayRate * elapsed)
	rc.Rate *= decayFactor
	rc.ErrorRate *= decayFactor
}

// Rate returns the current rate for the item.
// If the item is not being tracked, returns an estimate based on the minimum rate.
func (rt *RateTracker) Rate(item string) (rate float64, errorRate float64) {
	return rt.RateAt(item, time.Now())
}

// RateAt returns the rate for the item at the specified time.
func (rt *RateTracker) RateAt(item string, t time.Time) (rate float64, errorRate float64) {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	if rc, exists := rt.counters[item]; exists {
		// Apply decay to get current rate
		elapsed := t.Sub(rc.LastUpdate).Seconds()
		if elapsed > 0 {
			decayFactor := math.Exp(-rt.decayRate * elapsed)
			rate = rc.Rate * decayFactor
			errorRate = rc.ErrorRate * decayFactor
		} else {
			rate = rc.Rate
			errorRate = rc.ErrorRate
		}
		return rate, errorRate
	}

	// Item not tracked
	if rt.minHeap.size > 0 {
		minCounter := rt.minHeap.min()
		elapsed := t.Sub(minCounter.LastUpdate).Seconds()
		if elapsed > 0 {
			decayFactor := math.Exp(-rt.decayRate * elapsed)
			return 0, minCounter.Rate * decayFactor
		}
		return 0, minCounter.Rate
	}

	return 0, 0
}

// RateItem represents an item with its rate information.
type RateItem struct {
	Value     string
	Rate      float64
	ErrorRate float64
}

// Top returns the top n items by rate at the current time.
func (rt *RateTracker) Top(n int) []RateItem {
	return rt.TopAt(n, time.Now())
}

// TopAt returns the top n items by rate at the specified time.
func (rt *RateTracker) TopAt(n int, t time.Time) []RateItem {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	if n <= 0 {
		return nil
	}

	// Get all items with current rates
	items := make([]RateItem, 0, len(rt.counters))
	for _, rc := range rt.counters {
		elapsed := t.Sub(rc.LastUpdate).Seconds()
		var rate, errorRate float64
		if elapsed > 0 {
			decayFactor := math.Exp(-rt.decayRate * elapsed)
			rate = rc.Rate * decayFactor
			errorRate = rc.ErrorRate * decayFactor
		} else {
			rate = rc.Rate
			errorRate = rc.ErrorRate
		}

		items = append(items, RateItem{
			Value:     rc.Item,
			Rate:      rate,
			ErrorRate: errorRate,
		})
	}

	// Sort by rate descending
	sort.Slice(items, func(i, j int) bool {
		if items[i].Rate == items[j].Rate {
			return items[i].Value < items[j].Value
		}
		return items[i].Rate > items[j].Rate
	})

	if n > len(items) {
		n = len(items)
	}

	return items[:n]
}

// All returns all tracked items sorted by rate.
func (rt *RateTracker) All() []RateItem {
	return rt.Top(rt.capacity)
}

// AllAt returns all tracked items sorted by rate at the specified time.
func (rt *RateTracker) AllAt(t time.Time) []RateItem {
	return rt.TopAt(rt.capacity, t)
}

// Size returns the number of items currently being tracked.
func (rt *RateTracker) Size() int {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	return len(rt.counters)
}

// Capacity returns the maximum number of items that can be tracked.
func (rt *RateTracker) Capacity() int {
	return rt.capacity
}

// HalfLife returns the decay half-life.
func (rt *RateTracker) HalfLife() time.Duration {
	return rt.halfLife
}

// Reset clears all counters.
func (rt *RateTracker) Reset() {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	rt.counters = make(map[string]*rateCounter, rt.capacity)
	rt.minHeap = newRateMinHeap(rt.capacity)
}

// Export serializes the RateTracker structure.
func (rt *RateTracker) Export() ([]byte, error) {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	now := time.Now()

	// Convert to exportable format
	items := make([]RateItem, 0, len(rt.counters))
	for _, rc := range rt.counters {
		elapsed := now.Sub(rc.LastUpdate).Seconds()
		var rate, errorRate float64
		if elapsed > 0 {
			decayFactor := math.Exp(-rt.decayRate * elapsed)
			rate = rc.Rate * decayFactor
			errorRate = rc.ErrorRate * decayFactor
		} else {
			rate = rc.Rate
			errorRate = rc.ErrorRate
		}

		items = append(items, RateItem{
			Value:     rc.Item,
			Rate:      rate,
			ErrorRate: errorRate,
		})
	}

	data := struct {
		Capacity int
		HalfLife time.Duration
		Items    []RateItem
	}{
		Capacity: rt.capacity,
		HalfLife: rt.halfLife,
		Items:    items,
	}

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(data); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// ImportRateTracker deserializes a RateTracker structure.
func ImportRateTracker(data []byte) (*RateTracker, error) {
	var importData struct {
		Capacity int
		HalfLife time.Duration
		Items    []RateItem
	}

	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	if err := dec.Decode(&importData); err != nil {
		return nil, err
	}

	rt := NewRateTracker(importData.Capacity, importData.HalfLife)
	now := time.Now()

	for _, item := range importData.Items {
		rc := &rateCounter{
			Item:       item.Value,
			Rate:       item.Rate,
			ErrorRate:  item.ErrorRate,
			LastUpdate: now,
		}
		rt.counters[item.Value] = rc
		rt.minHeap.push(rc)
	}

	return rt, nil
}

// rateMinHeap implements a min-heap for rate counters.
type rateMinHeap struct {
	items []*rateCounter
	size  int
}

func newRateMinHeap(capacity int) *rateMinHeap {
	return &rateMinHeap{
		items: make([]*rateCounter, 0, capacity),
		size:  0,
	}
}

func (h *rateMinHeap) push(rc *rateCounter) {
	rc.index = h.size
	h.items = append(h.items, rc)
	h.size++
	h.up(rc.index)
}

func (h *rateMinHeap) min() *rateCounter {
	if h.size == 0 {
		return nil
	}
	return h.items[0]
}

func (h *rateMinHeap) fix(i int) {
	if !h.down(i) {
		h.up(i)
	}
}

func (h *rateMinHeap) up(i int) {
	for {
		parent := (i - 1) / 2
		if parent == i || h.items[parent].Rate <= h.items[i].Rate {
			break
		}
		h.swap(parent, i)
		i = parent
	}
}

func (h *rateMinHeap) down(i int) bool {
	i0 := i
	for {
		left := 2*i + 1
		if left >= h.size {
			break
		}

		j := left
		right := left + 1
		if right < h.size && h.items[right].Rate < h.items[left].Rate {
			j = right
		}

		if h.items[i].Rate <= h.items[j].Rate {
			break
		}

		h.swap(i, j)
		i = j
	}
	return i > i0
}

func (h *rateMinHeap) swap(i, j int) {
	h.items[i], h.items[j] = h.items[j], h.items[i]
	h.items[i].index = i
	h.items[j].index = j
}
