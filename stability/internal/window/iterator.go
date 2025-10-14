package window

import "fmt"

// Iterator returns an iterator over count buckets starting from the specified offset
func (w *Window) Iterator(offset int, count int) Iterator {
	return Iterator{
		count: count,
		cur:   &w.buckets[offset%w.size],
	}
}

// Iterator is an iterator for traversing buckets in the window
type Iterator struct {
	count         int     // total number of buckets to iterate
	iteratedCount int     // number of buckets already iterated
	cur           *Bucket // pointer to the current bucket
}

// Next returns true if there are still buckets left to iterate
func (i *Iterator) Next() bool {
	return i.count != i.iteratedCount
}

// Bucket returns the current bucket
func (i *Iterator) Bucket() Bucket {
	if !(i.Next()) {
		panic(fmt.Errorf("window: iteration out of range iteratedCount: %d count: %d", i.iteratedCount, i.count))
	}
	bucket := *i.cur
	i.iteratedCount++
	i.cur = i.cur.Next()
	return bucket
}
