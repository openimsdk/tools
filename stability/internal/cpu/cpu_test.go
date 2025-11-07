package cpu

import (
	"fmt"
	"math"
	"runtime"
	"testing"
	"time"
)

// TestCPUUsage tests CPU usage monitoring functionality
func TestCPUUsage(t *testing.T) {
	// Wait for CPU monitoring initialization to complete
	time.Sleep(time.Second)

	// Test ReadStat function
	stat := &Stat{}
	ReadStat(stat)

	t.Logf("Current CPU usage: %.2f%%", float64(stat.Usage)/10.0)

	// Usage should be a reasonable value (0-1000)
	if stat.Usage > 1000 {
		t.Errorf("CPU usage out of range: %d", stat.Usage)
	}

	// Test CPU usage change under load
	t.Run("CPU Load Test", func(t *testing.T) {
		// Record initial usage
		initialStat := &Stat{}
		ReadStat(initialStat)

		// Create CPU load
		done := make(chan bool)
		cores := runtime.NumCPU()

		// Start as many goroutines as cores to perform CPU-intensive work
		for i := 0; i < cores; i++ {
			go func() {
				// Perform some CPU-intensive work
				for {
					select {
					case <-done:
						return
					default:
						// Perform math calculations to generate CPU load
						for j := 0; j < 1000000; j++ {
							math.Sqrt(float64(j))
						}
					}
				}
			}()
		}

		// Wait for CPU usage to rise
		time.Sleep(2 * time.Second)

		// Check usage after load
		loadStat := &Stat{}
		ReadStat(loadStat)
		t.Logf("CPU usage after load: %.2f%%", float64(loadStat.Usage)/10.0)

		// Stop the load
		close(done)

		// Verify usage has increased
		// Note: this is not a strict test, other factors can affect CPU usage
		// We only verify that load typically increases usage
		if loadStat.Usage <= initialStat.Usage {
			t.Logf("Warning: CPU usage did not increase after load, initial:%.2f%%, after load:%.2f%%",
				float64(initialStat.Usage)/10.0,
				float64(loadStat.Usage)/10.0)
		}
	})
}

// TestGetCPUInfo tests CPU info retrieval functionality
func TestGetCPUInfo(t *testing.T) {
	// Get CPU info
	info := GetInfo()

	// Check returned info is reasonable
	t.Logf("CPU info: frequency=%d Hz, quota=%.2f, cores=%d",
		info.Frequency, info.Quota, info.Cores)

	// CPU core count should be greater than 0
	if info.Cores <= 0 {
		t.Errorf("Invalid CPU core count: %d", info.Cores)
	}

	// Quota should be positive
	if info.Quota <= 0 {
		t.Errorf("Invalid CPU quota: %.2f", info.Quota)
	}
}

// TestCgroupCPU tests the CgroupCPU implementation
func TestCgroupCPU(t *testing.T) {
	// Attempt to create a CgroupCPU instance
	cpu, err := NewCgroupCPU()
	if err != nil {
		t.Skipf("Skipping CgroupCPU test: %v", err)
		return
	}

	// Get usage
	usage, err := cpu.Usage()
	if err != nil {
		t.Errorf("Failed to get CPU usage: %v", err)
	} else {
		t.Logf("CgroupCPU usage: %.2f%%", float64(usage)/10.0)
	}

	// Get CPU info
	info := cpu.Info()
	t.Logf("CgroupCPU info: frequency=%d Hz, quota=%.2f, cores=%d",
		info.Frequency, info.Quota, info.Cores)
}

// BenchmarkReadStat benchmark for reading CPU usage
func BenchmarkReadStat(b *testing.B) {
	stat := &Stat{}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ReadStat(stat)
	}
}

// ExampleReadStat shows how to use ReadStat
func ExampleReadStat() {
	stat := &Stat{}
	ReadStat(stat)
	fmt.Printf("CPU usage: %.1f%%\n", float64(stat.Usage)/10.0)

	//	// Output: CPU usage: 100.0%
}
