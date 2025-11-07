package cpu

import (
	"errors"
	"os"
	"path"
	"runtime"
	"testing"
	"time"
)

// TestCgroupCPU tests cgroup CPU monitoring functionality
func TestCgroupCPUinCG(t *testing.T) {
	// Skip test on non-Linux systems
	if runtime.GOOS != "linux" {
		t.Skip("CgroupCPU tests only run on Linux")
	}

	// Check if cgroup filesystem exists
	if _, err := os.Stat("/sys/fs/cgroup"); err != nil {
		t.Skip("Cgroup filesystem not available")
	}

	// Try to create a CgroupCPU instance
	cpu, err := NewCgroupCPU()
	if err != nil {
		// Creation failed; not necessarily an error — system might not support cgroup v2
		t.Logf("Could not create CgroupCPU: %v", err)
		t.Skip("CgroupCPU not available on this system")
	}

	// Test getting CPU usage
	usage, err := cpu.Usage()
	if err != nil {
		t.Errorf("Failed to get CPU usage: %v", err)
	} else {
		t.Logf("Current CPU usage: %.2f%%", float64(usage)/10.0)
	}

	// Get usage several times to ensure delta calculation works
	for i := 0; i < 3; i++ {
		time.Sleep(time.Millisecond * 100)
		usage, err = cpu.Usage()
		if err != nil {
			t.Errorf("Failed to get CPU usage in iteration %d: %v", i, err)
		} else {
			t.Logf("CPU usage iteration %d: %.2f%%", i, float64(usage)/10.0)
		}
	}

	// Test retrieving CPU info
	info := cpu.Info()
	t.Logf("CPU info: frequency=%d Hz, quota=%.2f, cores=%d",
		info.Frequency, info.Quota, info.Cores)

	// Validate returned info is reasonable
	if info.Cores <= 0 {
		t.Errorf("Invalid CPU core count: %d", info.Cores)
	}
	if info.Quota <= 0 {
		t.Errorf("Invalid CPU quota: %.2f", info.Quota)
	}
}

// TestCgroupQuotaParsing tests CPU quota parsing
func TestCgroupQuotaParsing(t *testing.T) {
	// Create temporary test directory
	tempDir := t.TempDir()
	fakeCgroupPath := path.Join(tempDir, "fake_cgroup")
	if err := os.MkdirAll(fakeCgroupPath, 0755); err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	// Test cases
	testCases := []struct {
		name     string
		content  string
		expected float64
		hasError bool
		errType  error
	}{
		{
			name:     "normal quota",
			content:  "100000 100000",
			expected: 1.0,
			hasError: false,
		},
		{
			name:     "unlimited quota",
			content:  "max 100000",
			expected: float64(runtime.NumCPU()),
			hasError: false,
		},
		{
			name:     "invalid quota format",
			content:  "invalid",
			hasError: true,
		},
		{
			name:     "zero period value",
			content:  "100000 0",
			hasError: true,
			errType:  ErrNoCFSLimit,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a fake cpu.max file
			cpuMaxPath := path.Join(fakeCgroupPath, "cpu.max")
			if err := os.WriteFile(cpuMaxPath, []byte(tc.content), 0644); err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			// Test parsing
			quota, err := getCPUQuota(fakeCgroupPath)

			// Check errors
			if tc.hasError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				if tc.errType != nil && !errors.Is(err, tc.errType) {
					t.Errorf("Expected error of type %v but got %v", tc.errType, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}

				if tc.content == "max 100000" {
					if quota <= 0 {
						t.Errorf("Expected positive quota but got %.2f", quota)
					}
				} else if quota != tc.expected {
					t.Errorf("Expected quota %.2f but got %.2f", tc.expected, quota)
				}
			}
		})
	}
}

// TestUsageV2Parsing tests CPU usage parsing from cpu.stat
func TestUsageV2Parsing(t *testing.T) {
	// Create temporary test directory
	tempDir := t.TempDir()
	fakeCgroupPath := path.Join(tempDir, "fake_cgroup")
	if err := os.MkdirAll(fakeCgroupPath, 0755); err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	// Create fake cpu.stat file with valid content
	cpuStatContent := `usage_usec 12345
user_usec 10000
system_usec 2345
nr_periods 0
nr_throttled 0
throttled_usec 0`

	cpuStatPath := path.Join(fakeCgroupPath, "cpu.stat")
	if err := os.WriteFile(cpuStatPath, []byte(cpuStatContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Create CgroupCPU instance
	cgroupCPU := &CgroupCPU{
		cgroupPath: fakeCgroupPath,
	}

	// Test usage reading
	usage, err := cgroupCPU.usageV2()
	if err != nil {
		t.Errorf("Failed to parse usage: %v", err)
	} else {
		// Expected value is 12345000 (microseconds to nanoseconds)
		expected := uint64(12345000)
		if usage != expected {
			t.Errorf("Expected usage %d but got %d", expected, usage)
		} else {
			t.Logf("Correctly parsed CPU usage: %d ns", usage)
		}
	}

	// Test invalid content
	invalidContent := "invalid content"
	if err := os.WriteFile(cpuStatPath, []byte(invalidContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	_, err = cgroupCPU.usageV2()
	if err == nil {
		t.Errorf("Expected error for invalid content but got none")
	} else {
		t.Logf("Correctly handled invalid content: %v", err)
	}
}

// TestCgroupCreateCustom tests creating a CgroupCPU with custom path
func TestCgroupCreateCustom(t *testing.T) {
	// Create a test CgroupCPU with custom path
	tempDir := t.TempDir()
	cgroupCPU := &CgroupCPU{
		cgroupPath: tempDir,
		cores:      4,
		frequency:  3000000000, // 3GHz
		quota:      2.0,        // 2 cores equivalent
	}

	// Verify the info
	info := cgroupCPU.Info()
	if info.Cores != 4 {
		t.Errorf("Expected cores=4, got %d", info.Cores)
	}
	if info.Frequency != 3000000000 {
		t.Errorf("Expected frequency=3000000000, got %d", info.Frequency)
	}
	if info.Quota != 2.0 {
		t.Errorf("Expected quota=2.0, got %.2f", info.Quota)
	}
}
