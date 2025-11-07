package window

import (
	"time"
)

// Metric is a sample interface.
// Implementations of Metrics in metric package are Counter, Gauge,
// PointGauge, RollingCounter and RollingGauge.
type Metric interface {
	// Add adds the given value to the counter.
	Add(int64)
	// Value gets the current value.
	// If the metric's type is PointGauge, RollingCounter, RollingGauge,
	// it returns the sum value within the window.
	Value() int64
}

// Aggregation contains some common aggregation function.
// Each aggregation can compute summary statistics of window.
type Aggregation interface {
	// Min finds the min value within the window.
	Min() float64
	// Max finds the max value within the window.
	Max() float64
	// Avg computes average value within the window.
	Avg() float64
	// Sum computes sum value within the window.
	Sum() float64
}

// RollingCounter represents a ring window based on time duration.
// e.g. [[1], [3], [5]]
type RollingCounter interface {
	Metric
	Aggregation

	Timespan() int
	// Reduce applies the reduction function to all buckets within the window.
	Reduce(func(Iterator) float64) float64
}

// RollingCounterOpts contains the arguments for creating RollingCounter.
type RollingCounterOpts struct {
	Size           int
	BucketDuration time.Duration
}

type rollingCounter struct {
	policy *RollingWindow
}

// NewRollingCounter creates a new RollingCounter bases on RollingCounterOpts.
func NewRollingCounter(opts RollingCounterOpts) RollingCounter {
	opt := RollingWindowOptions(opts)

	return &rollingCounter{
		policy: NewRollingWindow(opt),
	}
}

// Add adds the given value to the counter.
func (r *rollingCounter) Add(val int64) {
	r.policy.Add(float64(val))
}

// Value gets the current value.
func (r *rollingCounter) Value() int64 {
	return int64(r.Sum())
}

// Timespan returns the time span of the rolling counter.
func (r *rollingCounter) Timespan() int {
	return r.policy.timespan()
}

// Reduce applies the reduction function to all buckets within the window.
func (r *rollingCounter) Reduce(f func(Iterator) float64) float64 {
	return r.policy.Reduce(f)
}

// Min returns the minimum value in the rolling window.
func (r *rollingCounter) Min() float64 {
	return r.policy.Reduce(Min)
}

// Max returns the maximum value in the rolling window.
func (r *rollingCounter) Max() float64 {
	return r.policy.Reduce(Max)
}

// Avg returns the average value in the rolling window.
func (r *rollingCounter) Avg() float64 {
	return r.policy.Reduce(Avg)
}

// Sum returns the sum of all values in the rolling window.
func (r *rollingCounter) Sum() float64 {
	return r.policy.Reduce(Sum)
}

// RollingGauge represents a ring window based on time duration.
type RollingGauge interface {
	Metric
	Aggregation

	// Latest returns the latest value.
	Latest() float64
	Timespan() int
	// Reduce applies the reduction function to all buckets within the window.
	Reduce(func(Iterator) float64) float64
}

// RollingGaugeOpts contains the arguments for creating RollingGauge.
type RollingGaugeOpts struct {
	Size           int
	BucketDuration time.Duration
}

type rollingGauge struct {
	policy *RollingWindow
}

// NewRollingGauge creates a new RollingGauge based on RollingGaugeOpts.
func NewRollingGauge(opts RollingGaugeOpts) RollingGauge {
	opt := RollingWindowOptions(opts)
	return &rollingGauge{
		policy: NewRollingWindow(opt),
	}
}

// Add adds the given value to the gauge.
func (r *rollingGauge) Add(val int64) {
	r.policy.Add(float64(val))
}

// Value gets the current value.
func (r *rollingGauge) Value() int64 {
	return int64(r.Sum())
}

// Latest returns the latest value.
func (r *rollingGauge) Latest() float64 {
	if r.policy.window.Size() == 0 {
		return 0
	}
	bucket := r.policy.window.Bucket(r.policy.offset)
	if bucket.Count == 0 {
		return 0
	}
	return bucket.Points[len(bucket.Points)-1]
}

// Timespan returns the time span of the rolling gauge.
func (r *rollingGauge) Timespan() int {
	return r.policy.timespan()
}

// Reduce applies the reduction function to all buckets within the window.
func (r *rollingGauge) Reduce(f func(Iterator) float64) float64 {
	return r.policy.Reduce(f)
}

// Min returns the minimum value in the rolling window.
func (r *rollingGauge) Min() float64 {
	return r.policy.Reduce(Min)
}

// Max returns the maximum value in the rolling window.
func (r *rollingGauge) Max() float64 {
	return r.policy.Reduce(Max)
}

// Avg returns the average value in the rolling window.
func (r *rollingGauge) Avg() float64 {
	return r.policy.Reduce(Avg)
}

// Sum returns the sum of all values in the rolling window.
func (r *rollingGauge) Sum() float64 {
	return r.policy.Reduce(Sum)
}
