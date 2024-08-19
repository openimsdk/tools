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
	// 初始化日志系统
	err := InitSDKLogger(
		"testLogger",   // loggerPrefixName
		"testModule",   // moduleName
		"TestSDK",      // sdkType
		"TestPlatform", // platformName
		5,              // logLevel (INFO)
		true,           // isStdout
		false,          // isJson
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

	SDKLog(context.Background(), 4, "cmd/abc.go", 666, "This is a test message", nil, []any{"key", "value", "key", "key", 1})
	ZDebug(context.TODO(), "msg")

	// SDKLog(context.Background(), 4, "cmd/abc.go", 666, "This is a test message", nil, []any{"key", "value"})

	// SDKLog(context.Background(), 4, "cmd/abc.go", 666, "This is a test message", nil, []any{"key", "value"})

	// 读取输出
	w.Close()
	out, _ := os.ReadFile(r.Name())
	os.Stdout = stdout

	// 验证日志输出是否包含自定义的 [file:line]
	_ = string(out)
	// assert.Contains(t, output, "This is a test message")
	// assert.Contains(t, output, "[TestSDK/TestPlatform]") // 确认 sdkType 和 platformName 出现在日志中
	// assert.Contains(t, output, "[test_file.go:123]")     // 验证 file 和 line 被正确传入
	// assert.Contains(t, output, "key")
	// assert.Contains(t, output, "value")
}

// func TestSDKLog6666(t *testing.T) {
// 	// 初始化日志系统
// 	err := InitSDKConfig(
// 		"testLogger",   // loggerPrefixName
// 		"testModule",   // moduleName
// 		"TestSDK",      // sdkType
// 		"TestPlatform", // platformName
// 		4,              // logLevel (INFO)
// 		true,           // isStdout
// 		false,          // isJson
// 		"./logs",       // logLocation
// 		5,              // rotateCount
// 		24,             // rotationTime
// 		"1.0.0",        // moduleVersion
// 		false,          // isSimplify
// 	)
// 	assert.NoError(t, err)

// 	// 捕获标准输出
// 	// var buf bytes.Buffer
// 	stdout := os.Stdout
// 	r, w, _ := os.Pipe()
// 	os.Stdout = w

// 	// 设置 logger 将输出定向到捕获的缓冲区
// 	logger := zap.NewExample()
// 	defer logger.Sync()

// 	// 调用 SDKLog，并传入自定义 file 和 line
// 	SDKLog6(context.Background(), 4, "test_file.go", 123, "This is a test message", nil, []any{"key", "value"})

// 	// 读取输出
// 	w.Close()
// 	out, _ := os.ReadFile(r.Name())
// 	os.Stdout = stdout

// 	// 验证日志输出是否包含自定义的 [file:line]
// 	_ = string(out)
// 	// assert.Contains(t, output, "This is a test message")
// 	// assert.Contains(t, output, "[TestSDK/TestPlatform]") // 确认 sdkType 和 platformName 出现在日志中
// 	// assert.Contains(t, output, "[test_file.go:123]")     // 验证 file 和 line 被正确传入
// 	// assert.Contains(t, output, "key")
// 	// assert.Contains(t, output, "value")
// }
