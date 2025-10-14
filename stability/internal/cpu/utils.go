package cpu

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// readFile reads file content and returns a trimmed string
// Handles gracefully if the file doesn't exist (important for macOS)
func readFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(content)), nil
}

// parseUint parses a string to uint64, handling negative values
func parseUint(s string) (uint64, error) {
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		// try parsing as int to handle negative values
		intValue, intErr := strconv.ParseInt(s, 10, 64)
		if intErr == nil && intValue < 0 {
			return 0, nil
		}
		return 0, err
	}
	return v, nil
}

// ParseUintList parses CPU list strings like "0-2,4,6-8"
func ParseUintList(val string) (map[int]bool, error) {
	if val == "" {
		return map[int]bool{}, nil
	}

	result := make(map[int]bool)
	parts := strings.Split(val, ",")
	errFormat := fmt.Errorf("invalid format: %s", val)

	for _, part := range parts {
		if !strings.Contains(part, "-") {
			num, err := strconv.Atoi(part)
			if err != nil {
				return nil, errFormat
			}
			result[num] = true
		} else {
			rangeParts := strings.SplitN(part, "-", 2)
			if len(rangeParts) != 2 {
				return nil, errFormat
			}

			min, err := strconv.Atoi(rangeParts[0])
			if err != nil {
				return nil, errFormat
			}

			max, err := strconv.Atoi(rangeParts[1])
			if err != nil {
				return nil, errFormat
			}

			if max < min {
				return nil, errFormat
			}

			for i := min; i <= max; i++ {
				result[i] = true
			}
		}
	}

	return result, nil
}

// readLines reads all lines from a file
func readLines(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	return lines, scanner.Err()
}
