package zookeeper

import (
	"context"
	"github.com/OpenIMSDK/tools/log"
)

type nilLog struct{}

func (nilLog) Debug(ctx context.Context, msg string, keysAndValues ...any) {}

func (nilLog) Info(ctx context.Context, msg string, keysAndValues ...any) {}

func (nilLog) Warn(ctx context.Context, msg string, err error, keysAndValues ...any) {}

func (nilLog) Error(ctx context.Context, msg string, err error, keysAndValues ...any) {}

func (nilLog) WithValues(keysAndValues ...any) log.Logger {
	return nilLog{}
}

func (nilLog) WithName(name string) log.Logger {
	return nilLog{}
}

func (nilLog) WithCallDepth(depth int) log.Logger {
	return nilLog{}
}
