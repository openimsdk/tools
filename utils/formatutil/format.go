package formatutil

import (
	"fmt"
	"strings"
)

// ProgressBar generates a formatted progress bar string.
func ProgressBar(name string, progress, total int) string {
	var (
		percentage     float64
		barLength      = 50
		progressLength int
	)

	if total == 0 {
		percentage = 0
		progressLength = 0
	} else {
		percentage = float64(progress) / float64(total) * 100
		barLength = 50
		progressLength = int(percentage / 100 * float64(barLength))
	}
	progressLength = min(progressLength, barLength)
	bar := strings.Repeat("█", progressLength) + strings.Repeat(" ", barLength-progressLength)
	return fmt.Sprintf("\r%s: [%s] %3.0f%% (%d/%d)", name, bar, percentage, progress, total)
}
