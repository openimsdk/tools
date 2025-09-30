package log

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestName(t *testing.T) {
	AsyncWrite = false
	sdkType := "TestSDK"
	platformName := runtime.GOOS
	err := InitLoggerFromConfig(
		"testLogger", // loggerPrefixName
		"testModule", // moduleName
		sdkType,      // sdkType
		platformName, // platformName
		5,            // logLevel (Debug)
		false,        // isStdout
		false,        // isJson
		".",          // logLocation
		5,            // rotateCount
		24,           // rotationTime
		"1.0.0",      // moduleVersion
		false,        // isSimplify
	)
	assert.NoError(t, err)

	ctx := context.Background()

	const count = 1000000
	start := time.Now()
	for i := 0; i < count; i++ {
		ZDebug(ctx, "test debug message", "key", "value", "log_index", i)
	}
	Flush()
	end := time.Now()
	duration := end.Sub(start)

	t.Log("cost:", duration)
	t.Log("avg:", duration/time.Duration(count))

	// 7.015554167s 7.015µs
	// 3.207912375s 3.207µs

}
