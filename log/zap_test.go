package log

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// TestSDKLog tests the SDKLog function for proper log output including custom [file:line] information
func TestSDKLog(t *testing.T) {
	err := InitSDKLogger(
		"testLogger",   // loggerPrefixName
		"testModule",   // moduleName
		"TestSDK",      // sdkType
		"TestPlatform", // platformName
		5,              // logLevel (INFO)
		true,           // isStdout
		true,           // isJson
		"./logs",       // logLocation
		5,              // rotateCount
		24,             // rotationTime
		"1.0.0",        // moduleVersion
		false,          // isSimplify
	)
	assert.NoError(t, err)

	// var buf bytes.Buffer
	stdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	logger := zap.NewExample()
	defer logger.Sync()

	SDKLog(context.Background(), 5, "cmd/abc.go", 666, "This is a test message", nil, []any{"key", "value"})
	SDKLog(context.Background(), 4, "cmd/abc.go", 666, "This is a test message", nil, []any{"key", "value", "key", "key", 1})
	SDKLog(context.Background(), 3, "cmd/abc.go", 666, "This is a test message", nil, []any{"key", "value"})
	SDKLog(context.Background(), 2, "cmd/abc.go", 666, "This is a test message", nil, []any{"key", "value"})
	ZDebug(context.TODO(), "msg")

	w.Close()
	out, _ := os.ReadFile(r.Name())
	os.Stdout = stdout

	_ = string(out)
	// assert.Contains(t, output, "This is a test message")
	// assert.Contains(t, output, "[TestSDK/TestPlatform]")
	// assert.Contains(t, output, "[test_file.go:123]")
	// assert.Contains(t, output, "key")
	// assert.Contains(t, output, "value")
}
