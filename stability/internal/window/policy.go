package window

import (
	"sync"
	"time"
)

// RollingWindowOptions contains parameters for creating a rolling window
type RollingWindowOptions struct {
	Size           int           // Window size (number of buckets)
	BucketDuration time.Duration // Duration of each bucket
}

// RollingWindow is a time-based rolling window
type RollingWindow struct {
	mu             sync.RWMutex
	window         *Window
	offset         int           // Position of the currently active bucket
	bucketDuration time.Duration // Duration of each bucket
	lastUpdateTime time.Time     // Last update time
}

// NewRollingWindow creates a new rolling window
func NewRollingWindow(opts RollingWindowOptions) *RollingWindow {
	window := NewWindow(Options{Size: opts.Size})
	return &RollingWindow{
		window:         window,
		bucketDuration: opts.BucketDuration,
		lastUpdateTime: time.Now(),
	}
}

// timespan returns the number of buckets that have passed since the last update
func (r *RollingWindow) timespan() int {
	now := time.Now()
	duration := now.Sub(r.lastUpdateTime)
	span := int(duration / r.bucketDuration)
	if span > 0 {
		r.lastUpdateTime = r.lastUpdateTime.Add(time.Duration(span) * r.bucketDuration)
	}
	return span
}

// Add adds a value to the current bucket and rolls the window according to time
func (r *RollingWindow) Add(val float64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// compute passed buckets and roll the window
	span := r.timespan()
	if span > 0 {
		// if passed buckets >= window size, reset the entire window
		if span >= r.window.Size() {
			r.window.ResetWindow()
			r.offset = 0
		} else {
			// otherwise only reset the passed buckets
			r.window.ResetBuckets(r.offset+1, span)
			r.offset = (r.offset + span) % r.window.Size()
		}
	}

	// append value to the current bucket
	r.window.Append(r.offset, val)
}

// Reduce applies an aggregation function to the data in the window
func (r *RollingWindow) Reduce(f func(Iterator) float64) float64 {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// first roll the window to ensure data is up to date
	span := r.timespan()
	if span > 0 {
		// here it's only calculating, not modifying the window, so no lock is needed
		if span >= r.window.Size() {
			return 0 // all data expired
		}
	}

	// compute number of valid buckets
	var validCount int
	if span >= r.window.Size() {
		validCount = 0
	} else {
		validCount = r.window.Size() - span
	}

	// return an iterator starting from the earliest valid bucket
	offset := (r.offset + span + 1) % r.window.Size()
	return f(r.window.Iterator(offset, validCount))
}
