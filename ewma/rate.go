package ewma

import (
	"math"
	"sync"
	"time"
)

// Rate implements automatic rate tracking with exponential decay.
// Unlike EWMA which requires manual Tick() calls, Rate automatically
// calculates rates based on time elapsed.
//
// Use cases:
//   - Request rate monitoring (automatic decay)
//   - Event frequency tracking
//   - Real-time metrics without ticker
type Rate struct {
	mu         sync.RWMutex
	rate       float64
	lastUpdate time.Time
	halfLife   time.Duration
	decayRate  float64
}

// NewRate creates a new Rate with the specified half-life.
//
// Half-life determines how quickly old events decay:
//   - Short half-life (1s-10s): Quick response to changes
//   - Medium half-life (30s-60s): Balanced
//   - Long half-life (5m-15m): Smooth, stable rates
func NewRate(halfLife time.Duration) *Rate {
	if halfLife <= 0 {
		halfLife = 60 * time.Second
	}

	return &Rate{
		halfLife:  halfLife,
		decayRate: math.Ln2 / halfLife.Seconds(),
	}
}

// Add records one or more events.
// Rate is automatically updated with exponential decay.
func (r *Rate) Add(n float64) {
	r.AddAt(n, time.Now())
}

// AddAt records events at a specific time.
// Useful for testing or processing historical data.
func (r *Rate) AddAt(n float64, t time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.lastUpdate.IsZero() {
		r.rate = n
		r.lastUpdate = t
		return
	}

	// Apply decay
	elapsed := t.Sub(r.lastUpdate).Seconds()
	if elapsed > 0 {
		decayFactor := math.Exp(-r.decayRate * elapsed)
		r.rate = r.rate*decayFactor + n
		r.lastUpdate = t
	} else {
		// Same or earlier time, just add
		r.rate += n
	}
}

// Rate returns the current rate with automatic decay applied.
func (r *Rate) Rate() float64 {
	return r.RateAt(time.Now())
}

// RateAt returns the rate at a specific time with decay applied.
func (r *Rate) RateAt(t time.Time) float64 {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.lastUpdate.IsZero() {
		return 0
	}

	elapsed := t.Sub(r.lastUpdate).Seconds()
	if elapsed <= 0 {
		return r.rate
	}

	decayFactor := math.Exp(-r.decayRate * elapsed)
	return r.rate * decayFactor
}

// Set sets the rate directly.
// Useful for initialization or testing.
func (r *Rate) Set(rate float64) {
	r.SetAt(rate, time.Now())
}

// SetAt sets the rate at a specific time.
func (r *Rate) SetAt(rate float64, t time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.rate = rate
	r.lastUpdate = t
}

// Reset clears the rate and resets to initial state.
func (r *Rate) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.rate = 0
	r.lastUpdate = time.Time{}
}

// HalfLife returns the decay half-life.
func (r *Rate) HalfLife() time.Duration {
	return r.halfLife
}

// RateSnapshot represents a point-in-time view of a Rate.
type RateSnapshot struct {
	Rate       float64
	LastUpdate time.Time
	HalfLife   time.Duration
}

// Snapshot returns a consistent snapshot of the current state.
func (r *Rate) Snapshot() RateSnapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return RateSnapshot{
		Rate:       r.rate,
		LastUpdate: r.lastUpdate,
		HalfLife:   r.halfLife,
	}
}
