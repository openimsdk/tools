package version

import (
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestGet verifies that the Get function returns expected fields correctly set.
func TestGet(t *testing.T) {
	v := Get()

	assert.NotEmpty(t, v.GoVersion, "GoVersion should not be empty")
	assert.NotEmpty(t, v.Compiler, "Compiler should not be empty")
	assert.NotEmpty(t, v.Platform, "Platform should not be empty")
	assert.True(t, strings.Contains(v.Platform, runtime.GOOS), "Platform should contain runtime.GOOS")
	assert.True(t, strings.Contains(v.Platform, runtime.GOARCH), "Platform should contain runtime.GOARCH")
}

// TestGetSingleVersion verifies that the GetSingleVersion function returns the gitVersion.
func TestGetSingleVersion(t *testing.T) {
	version := GetSingleVersion()
	assert.Equal(t, gitVersion, version, "gitVersion should match the global gitVersion variable")
}
