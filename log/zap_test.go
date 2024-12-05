package log

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// TestSDKLog tests the SDKLog function for proper log output including custom [file:line] information
func TestSDKLog(t *testing.T) {
	sdkType := "TestSDK"
	platformName := "testPlatform"

	err := InitLoggerFromConfig(
		"testLogger", // loggerPrefixName
		"testModule", // moduleName
		sdkType,      // sdkType
		platformName, // platformName
		5,            // logLevel (Debug)
		true,         // isStdout
		false,        // isJson
		// "./logs",     // logLocation
		".",     // logLocation
		5,       // rotateCount
		24,      // rotationTime
		"1.0.0", // moduleVersion
		false,   // isSimplify
	)
	assert.NoError(t, err)

	// var buf bytes.Buffer
	stdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	logger := zap.NewExample()
	defer logger.Sync()

	ZDebug(context.Background(), "hello")
	SDKLog(context.Background(), 5, "cmd/abc.go", 666, "This is a test message", nil, []any{"key", "value"})
	SDKLog(context.Background(), 4, "cmd/abc.go", 666, "This is a test message", nil, []any{"key", "value", "key", "key", 1})
	SDKLog(context.Background(), 3, "cmd/abc.go", 666, "This is a test message", nil, []any{"key", "value"})
	SDKLog(context.Background(), 2, "cmd/abc.go", 666, "This is a test message", nil, []any{"key", "value"})
	ZWarn(context.TODO(), "msg", nil)
	ZInfo(context.TODO(), "msg", nil)
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

func TestDefaultLog(t *testing.T) {
	er := errors.New("error")
	ZInfo(context.Background(), "Are you OK?")
	ZDebug(context.Background(), "Hello")
	ZWarn(context.Background(), "3Q", er)
	ZError(context.Background(), "3Q very much", er)

	sdkType := "TestSDK"
	platformName := "testPlatform"

	err := InitLoggerFromConfig(
		"testLogger", // loggerPrefixName
		"testModule", // moduleName
		sdkType,      // sdkType
		platformName, // platformName
		int(5),       // logLevel (Debug)
		true,         // isStdout
		false,        // isJson
		"./logs",     // logLocation
		uint(5),      // rotateCount
		uint(24),     // rotationTime
		"1.0.0",      // moduleVersion
		false,        // isSimplify
	)
	assert.NoError(t, err)

	stdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	logger := zap.NewExample()
	defer logger.Sync()

	ZDebug(context.Background(), "hello")
	SDKLog(context.Background(), 5, "cmd/abc.go", 666, "This is a test message", nil, []any{"key", "value"})
	SDKLog(context.Background(), 4, "cmd/abc.go", 666, "This is a test message", nil, []any{"key", "value", "key", "key", 1})
	SDKLog(context.Background(), 3, "cmd/abc.go", 666, "This is a test message", nil, []any{"key", "value"})
	SDKLog(context.Background(), 2, "cmd/abc.go", 666, "This is a test message", nil, []any{"key", "value"})
	ZWarn(context.TODO(), "msg", nil)
	ZInfo(context.TODO(), "msg", nil)
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
