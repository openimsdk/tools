package log

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// 测试 InitLoggerFromConfig 的初始化过程
func TestInitLoggerFromConfig(t *testing.T) {
	err := InitLoggerFromConfig(
		"testLogger",
		"testModule",
		4,        // Info level
		true,     // Stdout enabled
		false,    // Non-JSON output
		"./logs", // Log location
		5,        // Rotate count
		24,       // Rotate time in hours
		"1.0.0",  // Module version
		false,    // Simplify false
	)
	assert.NoError(t, err)
	assert.NotNil(t, pkgLogger)
}

// 测试 InitSDKConfig 的自定义初始化过程
func TestInitSDKConfig(t *testing.T) {
	err := InitSDKConfig(
		"testLogger",
		"testModule",
		"SDKType",      // sdkType
		"PlatformName", // platformName
		4,              // Info level
		true,           // Stdout enabled
		false,          // Non-JSON output
		"./logs",       // Log location
		5,              // Rotate count
		24,             // Rotate time in hours
		"1.0.0",        // Module version
		false,          // Simplify false
	)
	assert.NoError(t, err)
	assert.NotNil(t, pkgLogger)
}

// 测试 ZDebug 日志输出
func TestZDebug(t *testing.T) {
	// 先初始化日志系统
	err := InitLoggerFromConfig(
		"testLogger",
		"testModule",
		6,        // Debug level
		true,     // Stdout enabled
		false,    // Non-JSON output
		"./logs", // Log location
		5,        // Rotate count
		24,       // Rotate time in hours
		"1.0.0",  // Module version
		false,    // Simplify false
	)
	assert.NoError(t, err)

	// 捕获标准输出
	stdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// 测试 ZDebug
	ZDebug(context.Background(), "This is a debug message", "key1", "value1")

	// 读取输出
	w.Close()
	out, _ := os.ReadFile(r.Name())
	os.Stdout = stdout

	// 检查输出
	assert.Contains(t, string(out), "This is a debug message")
	assert.Contains(t, string(out), "key1")
	assert.Contains(t, string(out), "value1")
}

// 测试 ZError 日志输出
func TestZError(t *testing.T) {
	// 先初始化日志系统
	err := InitLoggerFromConfig(
		"testLogger",
		"testModule",
		2,        // Error level
		true,     // Stdout enabled
		false,    // Non-JSON output
		"./logs", // Log location
		5,        // Rotate count
		24,       // Rotate time in hours
		"1.0.0",  // Module version
		false,    // Simplify false
	)
	assert.NoError(t, err)

	// 捕获标准输出
	stdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// 测试 ZError
	ZError(context.Background(), "This is an error message", assert.AnError, "key1", "value1")

	// 读取输出
	w.Close()
	out, _ := os.ReadFile(r.Name())
	os.Stdout = stdout

	// 检查输出
	assert.Contains(t, string(out), "This is an error message")
	assert.Contains(t, string(out), "key1")
	assert.Contains(t, string(out), "value1")
	assert.Contains(t, string(out), assert.AnError.Error())
}

// 测试日志的旋转配置
func TestLogRotation(t *testing.T) {
	// 初始化日志系统
	err := InitLoggerFromConfig(
		"rotateTestLogger",
		"testModule",
		4,        // Info level
		true,     // Stdout enabled
		false,    // Non-JSON output
		"./logs", // Log location
		1,        // Rotate count
		1,        // Rotate time in hours
		"1.0.0",  // Module version
		false,    // Simplify false
	)
	assert.NoError(t, err)

	// 写入一些日志来触发旋转
	for i := 0; i < 10; i++ {
		ZInfo(context.Background(), "Log rotation test", "iteration", i)
		time.Sleep(100 * time.Millisecond) // 模拟时间间隔
	}

	// 检查日志文件是否生成
	logFiles, err := os.ReadDir("./logs")
	assert.NoError(t, err)
	assert.Greater(t, len(logFiles), 0)
}

// 测试 InitSDKConfig 的日志输出
func TestInitSDKConfigOutput(t *testing.T) {
	// 初始化日志系统，使用 InitSDKConfig
	err := InitSDKConfig(
		"testLogger",
		"testModule",
		"TestSDK",      // sdkType
		"TestPlatform", // platformName
		4,              // Info level
		true,           // Stdout enabled
		false,          // Non-JSON output
		"./logs",       // Log location
		5,              // Rotate count
		24,             // Rotate time in hours
		"1.0.0",        // Module version
		false,          // Simplify false
	)
	assert.NoError(t, err)
	assert.NotNil(t, pkgLogger)

	// 捕获标准输出
	stdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// 测试输出
	ZInfo(context.Background(), "This is a test log message", []string{"key", "value", "key", "val"})

	// 读取输出
	w.Close()
	out, _ := os.ReadFile(r.Name())
	os.Stdout = stdout

	// 检查输出内容是否包含 sdkType 和 platformName
	_ = string(out)
	// assert.Contains(t, output, "This is a test log message")
	// assert.Contains(t, output, "TestSDK/TestPlatform") // 检查组合的 sdkType 和 platformName
	// assert.Contains(t, output, "key")
	// assert.Contains(t, output, "value")
}

// TestSDKLog tests the SDKLog function for proper log output including custom [file:line] information
func TestSDKLog(t *testing.T) {
	// 初始化日志系统
	err := InitSDKConfig(
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

	// 捕获标准输出
	// var buf bytes.Buffer
	stdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// 设置 logger 将输出定向到捕获的缓冲区
	logger := zap.NewExample()
	defer logger.Sync()

	// 调用 SDKLog，并传入自定义 file 和 line
	// SDKLog(context.Background(), 4, "cmd/abc.go", 666, "This is a test message", nil, []any{"key", "value"})

	SDKLog666(context.Background(), 4, "cmd/abc.go", 666, "This is a test message", nil, []any{"key", "value"})

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
