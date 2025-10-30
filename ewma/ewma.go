package ewma

import (
	"math"
	"sync"
	"time"
)

const (
	// Interval is the standard tick interval (5 seconds)
	Interval = 5 * time.Second
)

// EWMA represents an exponentially weighted moving average.
// It is commonly used for:
// - Rate calculation (requests per second)
// - Load metrics smoothing
// - Response time tracking
// - Resource utilization monitoring
//
// EWMA gives more weight to recent values and exponentially less weight to older values.
type EWMA struct {
	mu          sync.RWMutex
	alpha       float64
	rate        float64
	uncounted   uint64
	initialized bool
	interval    time.Duration
}

// New creates a new EWMA with the specified alpha value.
//
// Alpha controls the decay rate:
//   - Higher alpha (close to 1): More weight to recent values
//   - Lower alpha (close to 0): More smoothing, slower response
//
// Common patterns:
//   - Use New1MinuteEWMA(), New5MinuteEWMA(), New15MinuteEWMA() for standard time windows
//   - Use NewWithAlpha() for custom decay rates
func New(alpha float64, interval time.Duration) *EWMA {
	if interval <= 0 {
		interval = Interval
	}

	return &EWMA{
		alpha:    alpha,
		interval: interval,
	}
}

// NewWithAlpha creates an EWMA with custom alpha value.
func NewWithAlpha(alpha float64) *EWMA {
	return New(alpha, Interval)
}

// New1MinuteEWMA creates an EWMA with a 1-minute decay time.
// Best for: Short-term trends, quick reaction to changes
func New1MinuteEWMA() *EWMA {
	return New(1-math.Exp(-float64(Interval)/float64(time.Minute)), Interval)
}

// New5MinuteEWMA creates an EWMA with a 5-minute decay time.
// Best for: Medium-term trends, balanced responsiveness
func New5MinuteEWMA() *EWMA {
	return New(1-math.Exp(-float64(Interval)/float64(5*time.Minute)), Interval)
}

// New15MinuteEWMA creates an EWMA with a 15-minute decay time.
// Best for: Long-term trends, heavy smoothing
func New15MinuteEWMA() *EWMA {
	return New(1-math.Exp(-float64(Interval)/float64(15*time.Minute)), Interval)
}

// Add records one or more events.
func (e *EWMA) Add(n uint64) {
	e.mu.Lock()
	e.uncounted += n
	e.mu.Unlock()
}

// Update records a single event.
// Equivalent to Add(1).
func (e *EWMA) Update() {
	e.Add(1)
}

// UpdateWithValue records an event with a specific value.
// Useful for tracking metrics like response time or bytes transferred.
func (e *EWMA) UpdateWithValue(value float64) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.initialized {
		e.rate = value
		e.initialized = true
	} else {
		e.rate = e.alpha*value + (1-e.alpha)*e.rate
	}
}

// Tick updates the moving average.
// Should be called at regular intervals (default: every 5 seconds).
//
// For typical usage with a ticker:
//
//	ticker := time.NewTicker(ewma.Interval)
//	go func() {
//	    for range ticker.C {
//	        myEWMA.Tick()
//	    }
//	}()
func (e *EWMA) Tick() {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Calculate instantaneous rate (events per interval)
	instantRate := float64(e.uncounted) / e.interval.Seconds()
	e.uncounted = 0

	if !e.initialized {
		e.rate = instantRate
		e.initialized = true
	} else {
		e.rate = e.alpha*instantRate + (1-e.alpha)*e.rate
	}
}

// Rate returns the current rate (events per second).
func (e *EWMA) Rate() float64 {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.rate
}

// Snapshot returns a snapshot of the current state.
// Use this for consistent reads of multiple values.
func (e *EWMA) Snapshot() Snapshot {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return Snapshot{
		Rate:        e.rate,
		Uncounted:   e.uncounted,
		Initialized: e.initialized,
	}
}

// Reset clears all data and resets to initial state.
func (e *EWMA) Reset() {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.rate = 0
	e.uncounted = 0
	e.initialized = false
}

// Set sets the rate directly.
// Useful for initialization or testing.
func (e *EWMA) Set(rate float64) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.rate = rate
	e.initialized = true
}

// Snapshot represents a point-in-time view of an EWMA.
type Snapshot struct {
	Rate        float64
	Uncounted   uint64
	Initialized bool
}

// MovingAverage represents a set of EWMAs for different time windows.
// Provides convenient access to 1-minute, 5-minute, and 15-minute rates.
type MovingAverage struct {
	m1  *EWMA
	m5  *EWMA
	m15 *EWMA
}

// NewMovingAverage creates a new MovingAverage with standard time windows.
func NewMovingAverage() *MovingAverage {
	return &MovingAverage{
		m1:  New1MinuteEWMA(),
		m5:  New5MinuteEWMA(),
		m15: New15MinuteEWMA(),
	}
}

// Add records one or more events across all time windows.
func (ma *MovingAverage) Add(n uint64) {
	ma.m1.Add(n)
	ma.m5.Add(n)
	ma.m15.Add(n)
}

// Update records a single event across all time windows.
func (ma *MovingAverage) Update() {
	ma.Add(1)
}

// Tick updates all moving averages.
func (ma *MovingAverage) Tick() {
	ma.m1.Tick()
	ma.m5.Tick()
	ma.m15.Tick()
}

// Rate1 returns the 1-minute rate.
func (ma *MovingAverage) Rate1() float64 {
	return ma.m1.Rate()
}

// Rate5 returns the 5-minute rate.
func (ma *MovingAverage) Rate5() float64 {
	return ma.m5.Rate()
}

// Rate15 returns the 15-minute rate.
func (ma *MovingAverage) Rate15() float64 {
	return ma.m15.Rate()
}

// Rates returns all three rates.
func (ma *MovingAverage) Rates() (m1, m5, m15 float64) {
	return ma.m1.Rate(), ma.m5.Rate(), ma.m15.Rate()
}

// Reset clears all moving averages.
func (ma *MovingAverage) Reset() {
	ma.m1.Reset()
	ma.m5.Reset()
	ma.m15.Reset()
}

// Snapshot returns snapshots of all time windows.
func (ma *MovingAverage) Snapshot() (s1, s5, s15 Snapshot) {
	return ma.m1.Snapshot(), ma.m5.Snapshot(), ma.m15.Snapshot()
}
