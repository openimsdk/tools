package window

import (
	"sync/atomic"
)

// Bucket is a basic structure that stores multiple float64 data points
type Bucket struct {
	Points []float64 // stores individual data points
	Count  int64     // number of data points in this bucket
	next   *Bucket   // points to the next bucket, forming a circular structure
}

// NewBucket creates a new bucket
func NewBucket() *Bucket {
	return &Bucket{
		Points: make([]float64, 0),
	}
}

// Append adds the given value to the bucket
func (b *Bucket) Append(val float64) {
	b.Points = append(b.Points, val)
	atomic.AddInt64(&b.Count, 1)
}

// Add adds the value at the specified offset
func (b *Bucket) Add(offset int, val float64) {
	b.Points[offset] += val
	atomic.AddInt64(&b.Count, 1)
}

// Reset clears the bucket
func (b *Bucket) Reset() {
	b.Points = b.Points[:0]
	atomic.StoreInt64(&b.Count, 0)
}

// Next return next bucket
func (b *Bucket) Next() *Bucket {
	return b.next
}
