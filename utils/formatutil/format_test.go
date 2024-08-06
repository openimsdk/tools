package formatutil

import (
	"fmt"
	"testing"
)

func TestProgressBar(t *testing.T) {
	result := ProgressBar("Task", 5, 10)
	fmt.Print(result)
}
