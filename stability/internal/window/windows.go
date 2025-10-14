package window

// window implements a sliding time window
type Window struct {
	buckets []Bucket
	size    int
}

type Options struct {
	Size int
}

func NewWindow(opt Options) *Window {
	buckets := make([]Bucket, opt.Size)
	for offset := range buckets {
		buckets[offset].Points = make([]float64, 0)
		nextOffset := offset + 1
		if nextOffset == opt.Size {
			nextOffset = 0
		}
		buckets[offset].next = &buckets[nextOffset]
	}

	return &Window{buckets: buckets, size: opt.Size}
}

// ResetWindow resets all buckets in the window
func (w *Window) ResetWindow() {
	for offset := range w.buckets {
		w.ResetBucket(offset)
	}
}

// ResetBucket resets the bucket at the specified offset
func (w *Window) ResetBucket(offset int) {
	w.buckets[offset%w.size].Reset()
}

// ResetBuckets resets count buckets starting from offset
func (w *Window) ResetBuckets(offset int, count int) {
	for i := range count {
		w.ResetBucket(offset + i)
	}
}

// Append appends a value to the bucket at the specified offset
func (w *Window) Append(offset int, val float64) {
	w.buckets[offset%w.size].Append(val)
}

// Add adds a value to the latest point in the bucket at the specified offset
func (w *Window) Add(offset int, val float64) {
	offset %= w.size
	if w.buckets[offset].Count == 0 {
		w.buckets[offset].Append(val)
		return
	}
	w.buckets[offset].Add(0, val)
}

// Bucket returns the bucket at the specified offset
func (w *Window) Bucket(offset int) Bucket {
	return w.buckets[offset%w.size]
}

// Size returns the window size
func (w *Window) Size() int {
	return w.size
}
