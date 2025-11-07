package cpu

import (
	"errors"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

// ErrNoCFSLimit indicates no CPU CFS limit is set
var ErrNoCFSLimit = errors.New("no CPU CFS limit")

// CgroupCPU implements CPU monitoring based on cgroup
type CgroupCPU struct {
	// cgroup path
	cgroupPath string

	// last statistics time
	lastSampleTime int64
	lastUsage      uint64

	// CPU information
	cores     int
	frequency uint64
	quota     float64
}

// Ensure CgroupCPU implements CPU interface
var _ CPU = (*CgroupCPU)(nil)

// NewCgroupCPU creates a cgroup-based CPU monitoring instance
func NewCgroupCPU() (*CgroupCPU, error) {
	// cgroup v2 root path
	cgroupPath := "/sys/fs/cgroup"

	// Check if cgroup v2 is supported
	if _, err := os.Stat(path.Join(cgroupPath, "cgroup.controllers")); err != nil {
		return nil, errors.New("cgroup v2 is not supported")
	}

	// Get CPU core count
	cores, err := countCPUCores()
	if err != nil {
		cores = 1 // Default to at least one core
	}

	// Get CPU frequency
	freq := getCPUMaxFreq()

	// Get CPU quota
	quota, err := getCPUQuota(cgroupPath)
	if err != nil {
		// If quota cannot be obtained, use core count as default
		quota = float64(cores)
	}

	return &CgroupCPU{
		cgroupPath: cgroupPath,
		cores:      cores,
		frequency:  freq,
		quota:      quota,
	}, nil
}

// Usage calculates current CPU usage percentage
func (c *CgroupCPU) Usage() (uint64, error) {
	// Get CPU usage amount
	cpuUsage, err := c.usageV2()
	if err != nil {
		return 0, err
	}

	// Calculate usage percentage (0-1000 range)
	now := time.Now().UnixNano()
	if c.lastSampleTime == 0 {
		c.lastSampleTime = now
		c.lastUsage = cpuUsage
		return 0, nil
	}

	// Time difference (nanoseconds)
	timeDelta := float64(now - c.lastSampleTime)
	if timeDelta <= 0 {
		return 0, nil
	}

	// Usage difference
	usageDelta := cpuUsage - c.lastUsage

	// Update last sample values
	c.lastSampleTime = now
	c.lastUsage = cpuUsage

	// Calculate usage (note: consider multiple cores)
	// cpuUsage is accumulated nanoseconds
	// Usage = CPU time used / (elapsed time * CPU cores) * 1000
	usage := uint64((float64(usageDelta) / timeDelta) * 1000 * float64(c.cores))

	// Limit to 1000
	if usage > 1000 {
		usage = 1000
	}

	return usage, nil
}

// usageV2 gets CPU usage from cgroup v2
func (c *CgroupCPU) usageV2() (uint64, error) {
	// In cgroup v2, CPU usage is in the usage_usec field of cpu.stat file
	statPath := path.Join(c.cgroupPath, "cpu.stat")
	content, err := readLines(statPath)
	if err != nil {
		return 0, err
	}

	for _, line := range content {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == "usage_usec" {
			value, err := strconv.ParseUint(fields[1], 10, 64)
			if err != nil {
				return 0, err
			}
			// Convert to nanoseconds
			return value * 1000, nil
		}
	}

	return 0, fmt.Errorf("usage_usec not found in cpu.stat")
}

// Info gets CPU information
func (c *CgroupCPU) Info() Info {
	return Info{
		Frequency: c.frequency,
		Quota:     c.quota,
		Cores:     c.cores,
	}
}

// getCPUQuota gets CPU quota
func getCPUQuota(cgroupPath string) (float64, error) {
	// In cgroup v2, quota is in cpu.max file
	maxPath := path.Join(cgroupPath, "cpu.max")
	content, err := readFile(maxPath)
	if err != nil {
		return 0, err
	}

	fields := strings.Fields(content)
	if len(fields) != 2 {
		return 0, fmt.Errorf("unexpected cpu.max format: %s", content)
	}

	// Format is "<quota> <period>"
	if fields[0] == "max" {
		// Unlimited
		cores, err := countCPUCores()
		if err != nil {
			cores = 1
		}
		return float64(cores), nil
	}

	quota, err := parseUint(fields[0])
	if err != nil {
		return 0, err
	}

	period, err := parseUint(fields[1])
	if err != nil {
		return 0, err
	}

	if period == 0 {
		return 0, ErrNoCFSLimit
	}

	return float64(quota) / float64(period), nil
}

// countCPUCores gets CPU core count
func countCPUCores() (int, error) {
	// Count processors from /proc/cpuinfo
	lines, err := readLines("/proc/cpuinfo")
	if err != nil {
		return 0, err
	}

	processors := make(map[string]bool)

	for _, line := range lines {
		if strings.HasPrefix(line, "processor") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				processors[fields[2]] = true
			}
		}
	}

	if len(processors) == 0 {
		return 1, nil // At least one core
	}

	return len(processors), nil
}

// getCPUMaxFreq gets maximum CPU frequency
func getCPUMaxFreq() uint64 {
	freqPath := "/sys/devices/system/cpu/cpu0/cpufreq/cpuinfo_max_freq"
	content, err := readFile(freqPath)
	if err != nil {
		return 0
	}

	freq, err := parseUint(content)
	if err != nil {
		return 0
	}

	return freq * 1000 // KHz to Hz
}
