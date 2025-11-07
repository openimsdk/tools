package sre

import (
	"math"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/openimsdk/tools/stability/circuitbreaker"
	"github.com/openimsdk/tools/stability/internal/window"
)

// Option is sre breaker option function.
type Option func(*options)

const (
	StateClosed int32 = iota
	StateOpen
)

var (
	_ circuitbreaker.CircuitBreaker = (*sreBreaker)(nil)
)

type options struct {
	success float64
	request int64
	bucket  int
	window  time.Duration
}

// sreBreaker implements the SRE circuit breaker algorithm
type sreBreaker struct {
	stat window.RollingCounter
	// window          *window              // sliding window for metrics

	r        *rand.Rand
	randLock sync.RWMutex

	k       float64
	request int64

	state int32 // circuitbreaker state
}

func WithSuccess(success float64) Option {
	return func(o *options) {
		o.success = success
	}
}

func WithRequest(request int64) Option {
	return func(o *options) {
		o.request = request
	}
}

func WithBucket(bucket int) Option {
	return func(o *options) {
		o.bucket = bucket
	}
}

func WithWindow(window time.Duration) Option {
	return func(o *options) {
		o.window = window
	}
}

func NewSREBraker(opts ...Option) circuitbreaker.CircuitBreaker {
	opt := options{
		success: 0.6,
		request: 100,
		bucket:  10,
		window:  3 * time.Second,
	}

	for _, o := range opts {
		o(&opt)
	}

	counterOpt := window.RollingCounterOpts{
		Size:           opt.bucket,
		BucketDuration: opt.window / time.Duration(opt.bucket),
	}

	stat := window.NewRollingCounter(counterOpt)

	return &sreBreaker{
		stat:    stat,
		r:       rand.New(rand.NewSource(time.Now().UnixNano())),
		request: opt.request,
		k:       1 / opt.success, // SRE算法中k值计算公式
		state:   StateClosed,
	}
}

func (b *sreBreaker) getStat() (success, total int64) {
	b.stat.Reduce(func(iterator window.Iterator) float64 {
		for iterator.Next() {
			bucket := iterator.Bucket()
			total += bucket.Count
			for _, p := range bucket.Points {
				success += int64(p)
			}
		}
		return 0
	})

	return
}

func (b *sreBreaker) Allow() error {
	success, total := b.getStat()

	requests := b.k * float64(success)

	if total < b.request || float64(total) < requests {
		atomic.CompareAndSwapInt32(&b.state, StateOpen, StateClosed)
		return nil
	}

	atomic.CompareAndSwapInt32(&b.state, StateClosed, StateOpen)

	dropProb := math.Max(0, (float64(total)-requests)/float64(total+1))
	drop := b.trueOnProb(dropProb)
	if drop {
		return circuitbreaker.ErrNotAllowed
	}

	return nil
}

func (b *sreBreaker) MarkSuccess() {
	b.stat.Add(1)
}

func (b *sreBreaker) MarkFailed() {
	b.stat.Add(0)
}

func (b *sreBreaker) trueOnProb(prob float64) (truth bool) {
	b.randLock.Lock()
	truth = b.r.Float64() < prob
	b.randLock.Unlock()
	return
}
