package log

import (
	"context"
	"fmt"
)

type ZkLogger struct{}

func NewZkLogger() *ZkLogger {
	return &ZkLogger{}
}

func (l *ZkLogger) Printf(format string, a ...any) {
	ZInfo(context.Background(), "zookeeper output", "msg", fmt.Sprintf(format, a...))
}
