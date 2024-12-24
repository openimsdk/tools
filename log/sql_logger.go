// Copyright © 2023 OpenIM. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package log

import (
	"context"
	"fmt"
	"time"

	"errors"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
	gormUtils "gorm.io/gorm/utils"
)

const nanosecondsToMilliseconds = 1e6

type SqlLogger struct {
	LogLevel                  gormLogger.LogLevel
	IgnoreRecordNotFoundError bool
	SlowThreshold             time.Duration
}

func NewSqlLogger(logLevel gormLogger.LogLevel, ignoreRecordNotFoundError bool, slowThreshold time.Duration) *SqlLogger {
	return &SqlLogger{
		LogLevel:                  logLevel,
		IgnoreRecordNotFoundError: ignoreRecordNotFoundError,
		SlowThreshold:             slowThreshold,
	}
}

func (l *SqlLogger) LogMode(logLevel gormLogger.LogLevel) gormLogger.Interface {
	newLogger := *l
	newLogger.LogLevel = logLevel
	return &newLogger
}

func (SqlLogger) Info(ctx context.Context, msg string, args ...any) {
	ZInfo(ctx, msg, "args", args)
}

func (SqlLogger) Warn(ctx context.Context, msg string, args ...any) {
	ZWarn(ctx, msg, nil, "args", args)
}

func (SqlLogger) Error(ctx context.Context, msg string, args ...any) {
	var err error = nil
	kvList := make([]any, 0)
	v, ok := args[0].(error)
	if ok {
		err = v
		for i := 1; i < len(args); i++ {
			kvList = append(kvList, fmt.Sprintf("args[%v]", i), args[i])
		}
	} else {
		for i := 0; i < len(args); i++ {
			kvList = append(kvList, fmt.Sprintf("args[%v]", i), args[i])
		}
	}
	ZError(ctx, msg, err, kvList...)
}

func (l *SqlLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if l.LogLevel <= gormLogger.Silent {
		return
	}
	elapsed := time.Since(begin)
	switch {
	case err != nil && l.LogLevel >= gormLogger.Error && (!errors.Is(err, gorm.ErrRecordNotFound) || !l.IgnoreRecordNotFoundError):
		sql, rows := fc()
		if rows == -1 {
			ZError(ctx, "sql exec detail", err, "gorm", gormUtils.FileWithLineNum(), "elapsed time", fmt.Sprintf("%f(ms)", float64(elapsed.Nanoseconds())/nanosecondsToMilliseconds), "sql", sql)
		} else {
			ZError(ctx, "sql exec detail", err, "gorm", gormUtils.FileWithLineNum(), "elapsed time", fmt.Sprintf("%f(ms)", float64(elapsed.Nanoseconds())/nanosecondsToMilliseconds), "rows", rows, "sql", sql)
		}
	case elapsed > l.SlowThreshold && l.SlowThreshold != 0 && l.LogLevel >= gormLogger.Warn:
		sql, rows := fc()
		slowLog := fmt.Sprintf("SLOW SQL >= %v", l.SlowThreshold)
		if rows == -1 {
			ZWarn(ctx, "sql exec detail", nil, "gorm", gormUtils.FileWithLineNum(), "slow sql", slowLog, "elapsed time", fmt.Sprintf("%f(ms)", float64(elapsed.Nanoseconds())/nanosecondsToMilliseconds), "sql", sql)
		} else {
			ZWarn(ctx, "sql exec detail", nil, "gorm", gormUtils.FileWithLineNum(), "slow sql", slowLog, "elapsed time", fmt.Sprintf("%f(ms)", float64(elapsed.Nanoseconds())/nanosecondsToMilliseconds), "rows", rows, "sql", sql)
		}
	case l.LogLevel == gormLogger.Info:
		sql, rows := fc()
		if rows == -1 {
			ZDebug(ctx, "sql exec detail", "gorm", gormUtils.FileWithLineNum(), "elapsed time", fmt.Sprintf("%f(ms)", float64(elapsed.Nanoseconds())/nanosecondsToMilliseconds), "sql", sql)
		} else {
			ZDebug(ctx, "sql exec detail", "gorm", gormUtils.FileWithLineNum(), "elapsed time", fmt.Sprintf("%f(ms)", float64(elapsed.Nanoseconds())/nanosecondsToMilliseconds), "rows", rows, "sql", sql)
		}
	}
}
