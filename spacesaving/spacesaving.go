package spacesaving

import (
	"bytes"
	"encoding/gob"
	"sort"
	"sync"
)

// SpaceSaving implements the Space-Saving algorithm for finding top-k frequent items.
// It maintains a fixed number of counters and provides approximate frequency counts
// with guaranteed error bounds.
//
// The algorithm is used for:
//   - Heavy hitters detection in network traffic
//   - Trending items in social media
//   - Popular products in e-commerce
//   - Frequent queries in databases
//
// Properties:
//   - Constant memory: O(k) where k is the number of counters
//   - Guaranteed error bound: count is overestimated by at most the count of the evicted item
//   - Thread-safe: All operations are protected by mutex
type SpaceSaving struct {
	mu       sync.RWMutex
	counters map[string]*counter
	minHeap  *minHeap
	capacity int
}

// counter represents a frequency counter for an item.
type counter struct {
	Item  string
	Count uint64
	Error uint64 // Maximum overestimation
	index int    // Index in the min-heap
}

// New creates a new SpaceSaving structure with the specified capacity.
// The capacity determines how many items to track.
//
// Typical values:
//   - Small streams: 100-1000 counters
//   - Medium streams: 1000-10000 counters
//   - Large streams: 10000+ counters
func New(capacity int) *SpaceSaving {
	if capacity <= 0 {
		capacity = 100
	}

	return &SpaceSaving{
		counters: make(map[string]*counter, capacity),
		minHeap:  newMinHeap(capacity),
		capacity: capacity,
	}
}

// Add records an occurrence of the item.
// Returns the approximate count after the addition.
func (ss *SpaceSaving) Add(item string) uint64 {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	// If item already tracked, increment its counter
	if c, exists := ss.counters[item]; exists {
		c.Count++
		ss.minHeap.fix(c.index)
		return c.Count
	}

	// If we haven't reached capacity, add new counter
	if len(ss.counters) < ss.capacity {
		c := &counter{
			Item:  item,
			Count: 1,
			Error: 0,
		}
		ss.counters[item] = c
		ss.minHeap.push(c)
		return 1
	}

	// Replace minimum counter
	minCounter := ss.minHeap.min()
	delete(ss.counters, minCounter.Item)

	minCounter.Item = item
	minCounter.Error = minCounter.Count
	minCounter.Count++

	ss.counters[item] = minCounter
	ss.minHeap.fix(0)

	return minCounter.Count
}

// Count returns the approximate count for the item.
// If the item is not being tracked, returns 0.
//
// Note: The count may be overestimated by at most the Error value.
func (ss *SpaceSaving) Count(item string) (count uint64, err uint64) {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	if c, exists := ss.counters[item]; exists {
		return c.Count, c.Error
	}

	// Item not tracked - could be up to minCount
	if ss.minHeap.size > 0 {
		minCounter := ss.minHeap.min()
		return 0, minCounter.Count
	}

	return 0, 0
}

// Item represents an item with its frequency information.
type Item struct {
	Value string
	Count uint64
	Error uint64
}

// Top returns the top n most frequent items.
// If n is greater than the number of tracked items, returns all tracked items.
// Items are sorted by count in descending order.
func (ss *SpaceSaving) Top(n int) []Item {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	if n <= 0 {
		return nil
	}

	// Get all items
	items := make([]Item, 0, len(ss.counters))
	for _, c := range ss.counters {
		items = append(items, Item{
			Value: c.Item,
			Count: c.Count,
			Error: c.Error,
		})
	}

	// Sort by count descending
	sort.Slice(items, func(i, j int) bool {
		if items[i].Count == items[j].Count {
			return items[i].Value < items[j].Value // Stable sort
		}
		return items[i].Count > items[j].Count
	})

	// Return top n
	if n > len(items) {
		n = len(items)
	}

	return items[:n]
}

// All returns all tracked items sorted by frequency (descending).
func (ss *SpaceSaving) All() []Item {
	return ss.Top(ss.capacity)
}

// Size returns the number of items currently being tracked.
func (ss *SpaceSaving) Size() int {
	ss.mu.RLock()
	defer ss.mu.RUnlock()
	return len(ss.counters)
}

// Capacity returns the maximum number of items that can be tracked.
func (ss *SpaceSaving) Capacity() int {
	return ss.capacity
}

// Reset clears all counters.
func (ss *SpaceSaving) Reset() {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	ss.counters = make(map[string]*counter, ss.capacity)
	ss.minHeap = newMinHeap(ss.capacity)
}

// Export serializes the SpaceSaving structure.
func (ss *SpaceSaving) Export() ([]byte, error) {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	// Convert to exportable format
	items := make([]Item, 0, len(ss.counters))
	for _, c := range ss.counters {
		items = append(items, Item{
			Value: c.Item,
			Count: c.Count,
			Error: c.Error,
		})
	}

	data := struct {
		Capacity int
		Items    []Item
	}{
		Capacity: ss.capacity,
		Items:    items,
	}

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(data); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Import deserializes a SpaceSaving structure.
func Import(data []byte) (*SpaceSaving, error) {
	var importData struct {
		Capacity int
		Items    []Item
	}

	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	if err := dec.Decode(&importData); err != nil {
		return nil, err
	}

	ss := New(importData.Capacity)

	for _, item := range importData.Items {
		c := &counter{
			Item:  item.Value,
			Count: item.Count,
			Error: item.Error,
		}
		ss.counters[item.Value] = c
		ss.minHeap.push(c)
	}

	return ss, nil
}

// minHeap implements a min-heap for counters.
type minHeap struct {
	items []*counter
	size  int
}

func newMinHeap(capacity int) *minHeap {
	return &minHeap{
		items: make([]*counter, 0, capacity),
		size:  0,
	}
}

func (h *minHeap) push(c *counter) {
	c.index = h.size
	h.items = append(h.items, c)
	h.size++
	h.up(c.index)
}

func (h *minHeap) min() *counter {
	if h.size == 0 {
		return nil
	}
	return h.items[0]
}

func (h *minHeap) fix(i int) {
	if !h.down(i) {
		h.up(i)
	}
}

func (h *minHeap) up(i int) {
	for {
		parent := (i - 1) / 2
		if parent == i || h.items[parent].Count <= h.items[i].Count {
			break
		}
		h.swap(parent, i)
		i = parent
	}
}

func (h *minHeap) down(i int) bool {
	i0 := i
	for {
		left := 2*i + 1
		if left >= h.size {
			break
		}

		j := left
		right := left + 1
		if right < h.size && h.items[right].Count < h.items[left].Count {
			j = right
		}

		if h.items[i].Count <= h.items[j].Count {
			break
		}

		h.swap(i, j)
		i = j
	}
	return i > i0
}

func (h *minHeap) swap(i, j int) {
	h.items[i], h.items[j] = h.items[j], h.items[i]
	h.items[i].index = i
	h.items[j].index = j
}
