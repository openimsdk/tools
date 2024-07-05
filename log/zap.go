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
	"github.com/openimsdk/tools/utils/stringutil"
	"gopkg.in/natefinch/lumberjack.v2"
	"os"
	"path/filepath"
	"time"

	"github.com/openimsdk/protocol/constant"
	"github.com/openimsdk/tools/mcontext"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	pkgLogger   Logger
	osStdout    Logger
	sp          = string(filepath.Separator)
	logLevelMap = map[int]zapcore.Level{
		6: zapcore.DebugLevel,
		5: zapcore.DebugLevel,
		4: zapcore.InfoLevel,
		3: zapcore.WarnLevel,
		2: zapcore.ErrorLevel,
		1: zapcore.FatalLevel,
		0: zapcore.PanicLevel,
	}
)

const callDepth = 2
const hoursPerDay = 24

// InitFromConfig initializes a Zap-based logger.
func InitFromConfig(
	loggerPrefixName, moduleName string,
	logLevel int,
	isStdout bool,
	isJson bool,
	logLocation string,
	rotateTime uint,
	maxBackups int,
	maxSize int,
	maxAge int,
	compress bool,
	moduleVersion string,
) error {
	l, err := NewZapLogger(loggerPrefixName, moduleName, logLevel, isStdout, isJson, logLocation,
		rotateTime, maxBackups, maxSize, maxAge, compress, moduleVersion)
	if err != nil {
		return err
	}
	setPkgLogger(isJson, moduleName, l)
	return nil
}

func setPkgLogger(isJson bool, moduleName string, l *ZapLogger) {
	pkgLogger = l.WithCallDepth(callDepth)
	if isJson {
		pkgLogger = pkgLogger.WithName(moduleName)
	}
}

// InitConsoleLogger init osStdout and osStderr.
func InitConsoleLogger(moduleName string,
	logLevel int,
	isJson bool, moduleVersion string) error {
	l, err := NewConsoleZapLogger(moduleName, logLevel, isJson, moduleVersion, os.Stdout)
	if err != nil {
		return err
	}
	osStdout = l.WithCallDepth(callDepth)
	if isJson {
		osStdout = osStdout.WithName(moduleName)
	}
	return nil

}

func ZDebug(ctx context.Context, msg string, keysAndValues ...any) {
	if pkgLogger == nil {
		return
	}
	pkgLogger.Debug(ctx, msg, keysAndValues...)
}

func ZInfo(ctx context.Context, msg string, keysAndValues ...any) {
	if pkgLogger == nil {
		return
	}
	pkgLogger.Info(ctx, msg, keysAndValues...)
}

func ZWarn(ctx context.Context, msg string, err error, keysAndValues ...any) {
	if pkgLogger == nil {
		return
	}
	pkgLogger.Warn(ctx, msg, err, keysAndValues...)
}

func ZError(ctx context.Context, msg string, err error, keysAndValues ...any) {
	if pkgLogger == nil {
		return
	}
	pkgLogger.Error(ctx, msg, err, keysAndValues...)
}

func CInfo(ctx context.Context, msg string, keysAndValues ...any) {
	if osStdout == nil {
		return
	}
	osStdout.Info(ctx, msg, keysAndValues...)
}

type ZapLogger struct {
	zap              *zap.SugaredLogger
	level            zapcore.Level
	moduleName       string
	moduleVersion    string
	loggerPrefixName string
	logLocation      string
	rotationTime     time.Duration
	maxBackups       int
	maxSize          int
	maxAge           int
	compress         bool
}

func NewZapLogger(
	loggerPrefixName, moduleName string,
	logLevel int,
	isStdout bool,
	isJson bool,
	logLocation string,
	rotationTime uint,
	maxBackups int,
	maxSize int,
	maxAge int,
	compress bool,
	moduleVersion string,
) (*ZapLogger, error) {
	zapConfig := zap.Config{
		Level:             zap.NewAtomicLevelAt(logLevelMap[logLevel]),
		DisableStacktrace: true,
	}
	if isJson {
		zapConfig.Encoding = "json"
	} else {
		zapConfig.Encoding = "console"
	}
	zl := &ZapLogger{
		level:            logLevelMap[logLevel],
		moduleName:       moduleName,
		moduleVersion:    moduleVersion,
		loggerPrefixName: loggerPrefixName,
		logLocation:      logLocation,
		rotationTime:     time.Duration(rotationTime) * time.Hour,
		maxBackups:       maxBackups,
		maxSize:          maxSize,
		maxAge:           maxAge,
		compress:         compress,
	}
	opts := zl.cores(isStdout, isJson)

	l, err := zapConfig.Build(opts)
	if err != nil {
		return nil, err
	}
	zl.zap = l.Sugar()
	return zl, nil
}

func NewConsoleZapLogger(
	moduleName string,
	logLevel int,
	isJson bool,
	moduleVersion string,
	outPut *os.File) (*ZapLogger, error) {
	zapConfig := zap.Config{
		Level:             zap.NewAtomicLevelAt(logLevelMap[logLevel]),
		DisableStacktrace: true,
	}
	if isJson {
		zapConfig.Encoding = "json"
	} else {
		zapConfig.Encoding = "console"
	}
	zl := &ZapLogger{level: logLevelMap[logLevel], moduleName: moduleName, moduleVersion: moduleVersion}
	opts := zl.consoleCores(outPut, isJson)

	l, err := zapConfig.Build(opts)
	if err != nil {
		return nil, err
	}
	zl.zap = l.Sugar()
	return zl, nil
}

func (l *ZapLogger) cores(isStdout bool, isJson bool) zap.Option {
	c := zap.NewProductionEncoderConfig()
	c.EncodeTime = l.timeEncoder
	c.EncodeDuration = zapcore.SecondsDurationEncoder
	c.MessageKey = "msg"
	c.LevelKey = "level"
	c.TimeKey = "time"
	c.CallerKey = "caller"
	c.NameKey = "logger"
	var fileEncoder zapcore.Encoder
	if isJson {
		c.EncodeLevel = zapcore.CapitalLevelEncoder
		fileEncoder = zapcore.NewJSONEncoder(c)
		fileEncoder.AddInt("PID", os.Getpid())
		fileEncoder.AddString("version", l.moduleVersion)
	} else {
		c.EncodeLevel = l.capitalColorLevelEncoder
		c.EncodeCaller = l.customCallerEncoder
		fileEncoder = zapcore.NewConsoleEncoder(c)
	}
	fileEncoder = &alignEncoder{Encoder: fileEncoder}
	writer := l.getWriter()

	var cores []zapcore.Core
	if l.logLocation != "" {
		cores = []zapcore.Core{
			zapcore.NewCore(fileEncoder, writer, zap.NewAtomicLevelAt(l.level)),
		}
	}
	if isStdout {
		cores = append(cores, zapcore.NewCore(fileEncoder, zapcore.Lock(os.Stdout), zap.NewAtomicLevelAt(l.level)))
		// cores = append(cores, zapcore.NewCore(fileEncoder, zapcore.Lock(os.Stderr), zap.NewAtomicLevelAt(l.level)))
	}
	return zap.WrapCore(func(c zapcore.Core) zapcore.Core {
		return zapcore.NewTee(cores...)
	})
}

// setupLogRotation set to rotate according to l.rotationTime
func (l *ZapLogger) setupLogRotation(lumLogger *lumberjack.Logger) {
	ticker := time.NewTicker(l.rotationTime)
	go func() {
		for range ticker.C {
			fileName := l.getLogFileName()
			lumLogger.Filename = fileName
			err := lumLogger.Rotate()
			if err != nil {
				ZError(context.TODO(), "rotate log field", err)
			}
		}
	}()
}

func (l *ZapLogger) consoleCores(outPut *os.File, isJson bool) zap.Option {
	c := zap.NewProductionEncoderConfig()
	c.EncodeTime = l.timeEncoder
	c.EncodeDuration = zapcore.SecondsDurationEncoder
	c.MessageKey = "msg"
	c.LevelKey = "level"
	c.TimeKey = "time"
	c.CallerKey = "caller"
	c.NameKey = "logger"
	var fileEncoder zapcore.Encoder
	if isJson {
		c.EncodeLevel = zapcore.CapitalLevelEncoder
		fileEncoder = zapcore.NewJSONEncoder(c)
		fileEncoder.AddInt("PID", os.Getpid())
		fileEncoder.AddString("version", l.moduleVersion)
	} else {
		c.EncodeLevel = l.capitalColorLevelEncoder
		c.EncodeCaller = l.customCallerEncoder
		fileEncoder = zapcore.NewConsoleEncoder(c)
	}
	var cores []zapcore.Core
	cores = append(cores, zapcore.NewCore(fileEncoder, zapcore.Lock(outPut), zap.NewAtomicLevelAt(l.level)))

	return zap.WrapCore(func(c zapcore.Core) zapcore.Core {
		return zapcore.NewTee(cores...)
	})
}

func (l *ZapLogger) customCallerEncoder(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
	fixedLength := 50
	trimmedPath := caller.TrimmedPath()
	trimmedPath = "[" + trimmedPath + "]"
	s := stringutil.FormatString(trimmedPath, fixedLength, true)
	enc.AppendString(s)
}

func (l *ZapLogger) timeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	layout := "2006-01-02 15:04:05.000"
	type appendTimeEncoder interface {
		AppendTimeLayout(time.Time, string)
	}
	if enc, ok := enc.(appendTimeEncoder); ok {
		enc.AppendTimeLayout(t, layout)
		return
	}
	enc.AppendString(t.Format(layout))
}

func (l *ZapLogger) getWriter() zapcore.WriteSyncer {
	fileName := l.getLogFileName()
	lumberjackLogger := &lumberjack.Logger{
		Filename:   fileName,     // log file path
		MaxSize:    l.maxSize,    // maximum size of each log file (in MB)
		MaxBackups: l.maxBackups, // maximum number of retained old log files
		MaxAge:     l.maxAge,     // maximum number of days to retain old log files
		Compress:   l.compress,   // whether compress old log files
	}
	l.setupLogRotation(lumberjackLogger) // Set to rotate according to time
	return zapcore.AddSync(lumberjackLogger)
}

func (l *ZapLogger) getLogFileName() string {
	var (
		now     = time.Now()
		timeStr string
	)

	if l.rotationTime%(time.Hour*time.Duration(hoursPerDay)) == 0 {
		timeStr = now.Format(".2006-01-02")
	} else if l.rotationTime%time.Hour == 0 {
		timeStr = now.Format(".2006-01-02_15")
	} else {
		timeStr = now.Format(".2006-01-02_15_04_05")
	}
	return l.logLocation + sp + l.loggerPrefixName + timeStr
}

func (l *ZapLogger) capitalColorLevelEncoder(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	s, ok := _levelToCapitalColorString[level]
	if !ok {
		s = _unknownLevelColor[zapcore.ErrorLevel]
	}
	pid := stringutil.FormatString(fmt.Sprintf("["+"PID:"+"%d"+"]", os.Getpid()), 15, true)
	color := _levelToColor[level]
	enc.AppendString(s)
	enc.AppendString(color.Add(pid))
	if l.moduleName != "" {
		moduleName := stringutil.FormatString(l.moduleName, 25, true)
		enc.AppendString(color.Add(moduleName))
	}
	if l.moduleVersion != "" {
		moduleVersion := stringutil.FormatString(fmt.Sprintf("["+"version:"+"%s"+"]", l.moduleVersion), 17, true)
		enc.AppendString(moduleVersion)
	}
}

func (l *ZapLogger) ToZap() *zap.SugaredLogger {
	return l.zap
}

func (l *ZapLogger) Debug(ctx context.Context, msg string, keysAndValues ...any) {
	if l.level > zapcore.DebugLevel {
		return
	}
	keysAndValues = l.kvAppend(ctx, keysAndValues)
	l.zap.Debugw(msg, keysAndValues...)
}

func (l *ZapLogger) Info(ctx context.Context, msg string, keysAndValues ...any) {
	if l.level > zapcore.InfoLevel {
		return
	}
	keysAndValues = l.kvAppend(ctx, keysAndValues)
	l.zap.Infow(msg, keysAndValues...)
}

func (l *ZapLogger) Warn(ctx context.Context, msg string, err error, keysAndValues ...any) {
	if l.level > zapcore.WarnLevel {
		return
	}
	if err != nil {
		keysAndValues = append(keysAndValues, "error", err.Error())
	}
	keysAndValues = l.kvAppend(ctx, keysAndValues)
	l.zap.Warnw(msg, keysAndValues...)
}

func (l *ZapLogger) Error(ctx context.Context, msg string, err error, keysAndValues ...any) {
	if l.level > zapcore.ErrorLevel {
		return
	}
	if err != nil {
		keysAndValues = append(keysAndValues, "error", err.Error())
	}
	keysAndValues = l.kvAppend(ctx, keysAndValues)
	l.zap.Errorw(msg, keysAndValues...)
}

func (l *ZapLogger) kvAppend(ctx context.Context, keysAndValues []any) []any {
	if ctx == nil {
		return keysAndValues
	}
	operationID := mcontext.GetOperationID(ctx)
	opUserID := mcontext.GetOpUserID(ctx)
	connID := mcontext.GetConnID(ctx)
	triggerID := mcontext.GetTriggerID(ctx)
	opUserPlatform := mcontext.GetOpUserPlatform(ctx)
	remoteAddr := mcontext.GetRemoteAddr(ctx)
	if opUserID != "" {
		keysAndValues = append([]any{constant.OpUserID, opUserID}, keysAndValues...)
	}
	if operationID != "" {
		keysAndValues = append([]any{constant.OperationID, operationID}, keysAndValues...)
	}
	if connID != "" {
		keysAndValues = append([]any{constant.ConnID, connID}, keysAndValues...)
	}
	if triggerID != "" {
		keysAndValues = append([]any{constant.TriggerID, triggerID}, keysAndValues...)
	}
	if opUserPlatform != "" {
		keysAndValues = append([]any{constant.OpUserPlatform, opUserPlatform}, keysAndValues...)
	}
	if remoteAddr != "" {
		keysAndValues = append([]any{constant.RemoteAddr, remoteAddr}, keysAndValues...)
	}
	return keysAndValues
}

func (l *ZapLogger) WithValues(keysAndValues ...any) Logger {
	dup := *l
	dup.zap = l.zap.With(keysAndValues...)
	return &dup
}

func (l *ZapLogger) WithName(name string) Logger {
	dup := *l
	dup.zap = l.zap.Named(name)
	return &dup
}

func (l *ZapLogger) WithCallDepth(depth int) Logger {
	dup := *l
	dup.zap = l.zap.WithOptions(zap.AddCallerSkip(depth))
	return &dup
}
