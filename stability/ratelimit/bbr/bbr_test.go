package bbr

import (
	"fmt"
	"os"
	"runtime/trace"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/openimsdk/tools/stability/ratelimit"
)

// mockCPU is used for testing; allows controlling CPU usage
func mockCPU(usage int64) func() {
	old := atomic.LoadInt64(&gCPU)
	atomic.StoreInt64(&gCPU, usage)
	return func() {
		atomic.StoreInt64(&gCPU, old)
	}
}

// TestNewBBRLimiter tests BBR limiter initialization
func TestNewBBRLimiter(t *testing.T) {
	// Test default configuration
	limiter := NewBBRLimiter()
	if limiter.opts.Window != time.Second*10 {
		t.Errorf("expected window to be 10s, got %v", limiter.opts.Window)
	}
	if limiter.opts.CPUThreshold != 800 {
		t.Errorf("expected CPU threshold to be 800, got %d", limiter.opts.CPUThreshold)
	}

	// Test custom configuration
	limiter = NewBBRLimiter(
		WithWindow(time.Second*5),
		WithBucket(50),
		WithCPUThreshold(700),
		WithCPUQuota(0.8),
	)

	if limiter.opts.Window != time.Second*5 {
		t.Errorf("expected window to be 5s, got %v", limiter.opts.Window)
	}
	if limiter.opts.Bucket != 50 {
		t.Errorf("expected bucket count to be 50, got %d", limiter.opts.Bucket)
	}
	if limiter.opts.CPUThreshold != 700 {
		t.Errorf("expected CPU threshold to be 700, got %d", limiter.opts.CPUThreshold)
	}
	if limiter.opts.CPUQuota != 0.8 {
		t.Errorf("expected CPU quota to be 0.8, got %f", limiter.opts.CPUQuota)
	}
}

// TestAllowWhenCpuLow tests that requests are not limited when CPU load is low
func TestAllowWhenCpuLow(t *testing.T) {
	// Simulate low CPU usage
	restore := mockCPU(500) // 50%
	defer restore()

	limiter := NewBBRLimiter(
		WithCPUThreshold(800), // set a high threshold
	)

	// Continuous requests; expect all to pass
	for i := 0; i < 100; i++ {
		done, err := limiter.Allow()
		if err != nil {
			t.Errorf("request was rate limited when CPU usage is low, i=%d, err=%v", i, err)
		} else {
			done(ratelimit.DoneInfo{})
		}
	}
}

// TestLimitWhenCpuHigh tests that rate limiting starts when CPU load is high
func TestLimitWhenCpuHigh(t *testing.T) {
	// Simulate high CPU usage
	restore := mockCPU(900) // 90%
	defer restore()

	limiter := NewBBRLimiter(
		WithCPUThreshold(800), // set a lower threshold
	)

	// First batch of requests; should not be rate limited yet
	for i := 0; i < 10; i++ {
		done, err := limiter.Allow()
		if err != nil {
			t.Logf("first batch request #%d was rate limited", i)
		} else {
			done(ratelimit.DoneInfo{})
		}
	}

	// Simulate heavy load exceeding processing capacity
	var wg sync.WaitGroup
	wg.Add(100)

	// Launch 100 concurrent requests
	rejected := int32(0)
	for i := 0; i < 100; i++ {
		go func() {
			defer wg.Done()
			done, err := limiter.Allow()
			if err != nil {
				atomic.AddInt32(&rejected, 1)
			} else {
				// Simulate request processing time
				time.Sleep(time.Millisecond * 10)
				done(ratelimit.DoneInfo{})
			}
		}()
	}
	wg.Wait()

	// With high CPU load, some requests should be rejected
	if rejected == 0 {
		t.Error("expected some requests to be rate limited under high CPU usage, but all passed")
	} else {
		t.Logf("under high CPU usage, %d requests were rate limited", rejected)
	}
}

// TestCoolingTimeWorks tests the cooling period functionality
func TestCoolingTimeWorks(t *testing.T) {
	// Start with high CPU
	restore := mockCPU(900)

	limiter := NewBBRLimiter(
		WithCPUThreshold(800),
	)

	// Trigger rate limiting
	var lastErr error
	for i := 0; i < 50; i++ {
		_, err := limiter.Allow()
		lastErr = err
	}

	// Check that rate limiting has started
	if lastErr == nil {
		t.Fatal("expected rate limiting to have been triggered, but none occurred")
	}

	// Restore CPU to low load
	restore()
	restore = mockCPU(500)
	defer restore()

	// During cooling period, rate limiting should still be in effect
	if _, err := limiter.Allow(); err == nil {
		t.Error("expected rate limiting to continue during cooling period, but request passed")
	}

	// Wait for cooling period to end
	time.Sleep(time.Second * 2)

	// After cooling, requests should no longer be rate limited
	done, err := limiter.Allow()
	if err != nil {
		t.Error("expected no rate limiting after cooling period, but request was rejected")
	} else {
		done(ratelimit.DoneInfo{})
	}
}

// TestParallelRequests tests concurrent request scenarios
func TestParallelRequests(t *testing.T) {
	restore := mockCPU(800) // boundary value
	defer restore()

	limiter := NewBBRLimiter(
		WithCPUThreshold(800),
		WithWindow(time.Second),
	)

	var wg sync.WaitGroup
	success := int32(0)
	rejected := int32(0)

	// Send 1000 concurrent requests
	for i := range 1000 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			done, err := limiter.Allow()
			if err != nil {
				atomic.AddInt32(&rejected, 1)
			} else {
				// Simulate variable processing time
				time.Sleep(time.Millisecond * time.Duration(i%10))
				done(ratelimit.DoneInfo{})
				atomic.AddInt32(&success, 1)
			}
		}()
	}

	wg.Wait()

	t.Logf("concurrency test results: success=%d, rejected=%d", success, rejected)
	// No hard assertions because results may vary with environment; just record data
}

// TestUnderPressure tests BBR limiter behavior under realistic pressure
func TestUnderPressure(t *testing.T) {
	var f *os.File
	var err error

	if os.Getenv("GOTRACE") != "1" {
		f, err = os.Create("test_under_pressure.trace")
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()

		err = trace.Start(f)
		if err != nil && err.Error() != "execution tracer already enabled" {
			t.Fatal(err)
		}
		defer trace.Stop()
	}

	// Simulate CPU slightly above threshold - 850/1000 = 85%
	// Not extreme high load; more realistic scenario
	restore := mockCPU(850)
	defer restore()

	limiter := NewBBRLimiter(
		WithCPUThreshold(800),
		WithWindow(time.Second),
		WithBucket(10), // increase bucket granularity for faster response to load changes
	)

	// Warm-up phase - submit at least 50 requests
	for i := 0; i < 50; i++ {
		done, err := limiter.Allow()
		if err == nil {
			// Simulate short processing
			time.Sleep(time.Millisecond * 10)
			done(ratelimit.DoneInfo{})
		}
		// Add a short delay to avoid instantaneous bursts
		time.Sleep(time.Millisecond * 5)
	}

	// Send requests in batches rather than a single large burst
	// This matches more realistic traffic patterns
	batches := 5
	requestsPerBatch := 40
	success := int32(0)
	rejected := int32(0)
	batchResults := make([]string, 0, batches)

	for b := 0; b < batches; b++ {
		batchSuccess := int32(0)
		batchRejected := int32(0)
		var wg sync.WaitGroup
		wg.Add(requestsPerBatch)

		for i := 0; i < requestsPerBatch; i++ {
			go func(idx int) {
				defer wg.Done()

				// Add small stagger to avoid simultaneous arrival
				time.Sleep(time.Millisecond * time.Duration(idx%5))

				done, err := limiter.Allow()
				if err != nil {
					atomic.AddInt32(&rejected, 1)
					atomic.AddInt32(&batchRejected, 1)
				} else {
					// Simulate realistic workload - processing time varies slightly
					processTime := time.Millisecond * time.Duration(20+idx%10)
					time.Sleep(processTime)
					done(ratelimit.DoneInfo{})
					atomic.AddInt32(&success, 1)
					atomic.AddInt32(&batchSuccess, 1)
				}
			}(i)
		}

		wg.Wait()

		// Record results per batch
		batchResults = append(batchResults,
			fmt.Sprintf("batch %d: success=%d (%.1f%%), rejected=%d (%.1f%%)",
				b+1,
				batchSuccess, float64(batchSuccess)*100/float64(requestsPerBatch),
				batchRejected, float64(batchRejected)*100/float64(requestsPerBatch)))

		// Short pause between batches to allow recovery
		time.Sleep(time.Millisecond * 200)
	}

	totalRequests := batches * requestsPerBatch
	successRate := float64(success) * 100 / float64(totalRequests)

	t.Logf("stress test results: total=%d, success=%d (%.1f%%), rejected=%d (%.1f%%)",
		totalRequests, success, successRate, rejected, 100-successRate)

	for _, result := range batchResults {
		t.Logf("%s", result)
	}

	// Ensure a reasonable pass rate - under high load at least 30% should pass
	// This is reasonable because BBR aims to protect the system, not fully block traffic
	if successRate < 30 {
		t.Errorf("pass rate too low (%.1f%%); expected at least 30%% of requests to pass", successRate)
	}

	// Also ensure some rate limiting occurred
	if successRate > 95 {
		t.Errorf("pass rate too high (%.1f%%); expected some rate limiting under high load", successRate)
	}
}
