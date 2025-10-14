package bbr

import (
	"math"
	"sync/atomic"
	"time"

	"github.com/openimsdk/tools/stability/internal/cpu"
	"github.com/openimsdk/tools/stability/internal/window"
	"github.com/openimsdk/tools/stability/ratelimit"
)

var (
	gCPU  int64
	decay = 0.95

	_ ratelimit.Limiter = (*BBR)(nil)
)

type (
	cpuGetter func() int64

	// Option function for bbr limiter
	Option func(*options)
)

func init() {
	go cpuProc()
}

func cpuProc() {
	ticker := time.NewTicker(time.Millisecond * 500)
	defer func() {
		ticker.Stop()
		if err := recover(); err != nil {
			go cpuProc()
		}
	}()

	for range ticker.C {
		// Use the stats obtained from cpu package
		stat := &cpu.Stat{}
		cpu.ReadStat(stat)
		stat.Usage = min(stat.Usage, 1000)
		prevCPU := atomic.LoadInt64(&gCPU)
		curCPU := int64(float64(prevCPU)*decay + float64(stat.Usage)*(1.0-decay))
		atomic.StoreInt64(&gCPU, curCPU)
	}
}

type counterCache struct {
	val  int64
	time time.Time
}

type Stat struct {
	CPU         int64
	InFlight    int64
	MaxInFlight int64
	MinRt       int64
	MaxPass     int64
}

type options struct {
	Window       time.Duration
	Bucket       int
	CPUThreshold int64
	CPUQuota     float64
}

func WithWindow(d time.Duration) Option {
	return func(o *options) {
		o.Window = d
	}
}

func WithBucket(bucket int) Option {
	return func(o *options) {
		o.Bucket = bucket
	}
}

func WithCPUThreshold(threshold int64) Option {
	return func(o *options) {
		o.CPUThreshold = threshold
	}
}

func WithCPUQuota(quota float64) Option {
	return func(o *options) {
		o.CPUQuota = quota
	}
}

type BBR struct {
	cpu             cpuGetter
	passStat        window.RollingCounter
	rtStat          window.RollingCounter
	inFlight        int64
	bucketPerSecond int64
	bucketDuration  time.Duration

	prevDropTime atomic.Value
	maxPASSCache atomic.Value
	minRtCache   atomic.Value

	opts options
}

func NewBBRLimiter(opts ...Option) *BBR {
	opt := options{
		Window:       time.Second * 10,
		Bucket:       100,
		CPUThreshold: 800,
	}

	for _, o := range opts {
		o(&opt)
	}

	bucketDuration := opt.Window / time.Duration(opt.Bucket)
	statOpt := window.RollingCounterOpts{
		Size:           opt.Bucket,
		BucketDuration: bucketDuration,
	}
	passStat := window.NewRollingCounter(statOpt)
	rtStat := window.NewRollingCounter(statOpt)

	limiter := &BBR{
		opts:            opt,
		passStat:        passStat,
		rtStat:          rtStat,
		bucketDuration:  bucketDuration,
		bucketPerSecond: int64(time.Second / bucketDuration),
		cpu:             func() int64 { return atomic.LoadInt64(&gCPU) },
	}

	if opt.CPUQuota != 0 {
		limiter.cpu = func() int64 {
			return int64(float64(atomic.LoadInt64(&gCPU)) * float64(opt.CPUQuota))
		}
	}

	return limiter
}

func (l *BBR) Allow() (ratelimit.DoneFunc, error) {
	if l.shouldDrop() {
		return nil, ratelimit.ErrLimitExceeded
	}

	atomic.AddInt64(&l.inFlight, 1)
	start := time.Now()

	return func(ratelimit.DoneInfo) {
		// compute elapsed time and ceil to milliseconds
		elapsed := time.Since(start)
		rt := int64((elapsed + time.Millisecond - time.Nanosecond) / time.Millisecond)
		if rt > 0 {
			l.rtStat.Add(rt)
		}

		atomic.AddInt64(&l.inFlight, -1)
		l.passStat.Add(1)
	}, nil
}

// shouldDrop determines if the request should be dropped
func (l *BBR) shouldDrop() bool {
	now := time.Duration(time.Now().UnixNano())
	if l.cpu() < l.opts.CPUThreshold {
		// CPU load is below the threshold
		prevDropTime, _ := l.prevDropTime.Load().(time.Duration)
		if prevDropTime == 0 {
			// Never triggered rate limiting before, allow directly
			return false
		}
		if time.Duration(now-prevDropTime) <= time.Second {
			// Within cooldown period (1 second), check current in-flight count
			inFlight := atomic.LoadInt64(&l.inFlight)
			return inFlight > 1 && inFlight > l.maxPass()
		}
		// Cooldown has passed, reset rate limiting state
		l.prevDropTime.Store(time.Duration(0))
		return false
	}

	// CPU load exceeds threshold
	inFlight := atomic.LoadInt64(&l.inFlight)

	maxInflight := l.maxInFlight()
	drop := (inFlight > 1 && inFlight > maxInflight)

	if drop {
		prevDrop, _ := l.prevDropTime.Load().(time.Duration)
		if prevDrop != 0 {
			// Already in rate limiting, return result directly
			return drop
		}
		// Record the start time of rate limiting
		l.prevDropTime.Store(now)
	}
	return drop
}

// maxPass calculates maximum allowed pass requests per second
func (l *BBR) maxPass() int64 {
	passCache := l.maxPASSCache.Load()
	if passCache != nil {
		ps := passCache.(*counterCache)
		if l.timespan(ps.time) < 1 {
			return ps.val
		}
	}

	// Calculate the maximum pass count in the current window
	rawMaxPass := int64(l.passStat.Reduce(func(iterator window.Iterator) float64 {
		var result = 1.0
		for i := 1; iterator.Next() && i < l.opts.Bucket; i++ {
			bucket := iterator.Bucket()
			count := 0.0
			for _, p := range bucket.Points {
				count += p
			}
			result = math.Max(result, count)
		}
		return result
	}))

	l.maxPASSCache.Store(&counterCache{
		val:  rawMaxPass,
		time: time.Now(),
	})
	return rawMaxPass
}

func (l *BBR) timespan(lastTime time.Time) int {
	v := int(time.Since(lastTime) / l.bucketDuration)
	if v > -1 {
		return v
	}
	return l.opts.Bucket
}

// minRt calculates minimum response time
func (l *BBR) minRt() int64 {
	rtCache := l.minRtCache.Load()
	if rtCache != nil {
		rc := rtCache.(*counterCache)
		if l.timespan(rc.time) < 1 {
			return rc.val
		}
	}

	// Calculate the minimum response time in the current window
	rawMinRt := l.rtStat.Min()
	if rawMinRt <= 0 {
		rawMinRt = 1 // avoid division by zero
	}
	rawMinRT := int64(math.Ceil(l.rtStat.Reduce(func(iterator window.Iterator) float64 {
		var result = math.MaxFloat64
		for i := 1; iterator.Next() && i < l.opts.Bucket; i++ {
			bucket := iterator.Bucket()
			if len(bucket.Points) == 0 {
				continue
			}
			total := 0.0
			for _, p := range bucket.Points {
				total += p
			}
			avg := total / float64(bucket.Count)
			result = math.Min(result, avg)
		}
		return result
	})))
	if rawMinRT <= 0 {
		rawMinRT = 1
	}
	l.minRtCache.Store(&counterCache{
		val:  rawMinRT,
		time: time.Now(),
	})
	return rawMinRT
}

func (l *BBR) maxInFlight() int64 {
	return int64(math.Floor(float64(l.maxPass()*l.minRt()*l.bucketPerSecond)/1000.0) + 0.5)
}

func (l *BBR) Stat() Stat {
	return Stat{
		CPU:         l.cpu(),
		MinRt:       l.minRt(),
		MaxPass:     l.maxPass(),
		MaxInFlight: l.maxInFlight(),
		InFlight:    atomic.LoadInt64(&l.inFlight),
	}
}
