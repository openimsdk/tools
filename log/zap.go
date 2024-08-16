package log

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	rotatelogs "github.com/openimsdk/tools/log/file-rotatelogs"
	"github.com/openimsdk/tools/utils/stringutil"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/openimsdk/protocol/constant"
	"github.com/openimsdk/tools/mcontext"
)

type LogFormatter interface {
	Format() any
}

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
func InitLoggerFromConfig(
	loggerPrefixName, moduleName string,
	logLevel int,
	isStdout bool,
	isJson bool,
	logLocation string,
	rotateCount uint,
	rotationTime uint,
	moduleVersion string,
	isSimplify bool,
) error {
	l, err := NewZapLogger(loggerPrefixName, moduleName, logLevel, isStdout, isJson, logLocation,
		rotateCount, rotationTime, moduleVersion, isSimplify)
	if err != nil {
		return err
	}
	pkgLogger = l.WithCallDepth(callDepth)
	if isJson {
		pkgLogger = pkgLogger.WithName(moduleName)
	}
	return nil
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

func InitSDKConfig(
	loggerPrefixName, moduleName string,
	sdkType, platformName string,
	logLevel int,
	isStdout bool,
	isJson bool,
	logLocation string,
	rotateCount uint,
	rotationTime uint,
	moduleVersion string,
	isSimplify bool,
) error {
	// 调用 NewSDKZapLogger 进行自定义初始化
	l, err := NewSDKZapLogger(loggerPrefixName, moduleName, sdkType, platformName, logLevel, isStdout, isJson, logLocation, rotateCount, rotationTime, moduleVersion, isSimplify)
	if err != nil {
		return err
	}
	pkgLogger = l.WithCallDepth(callDepth)
	if isJson {
		pkgLogger = pkgLogger.WithName(moduleName)
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
	rotationTime     time.Duration
	sdkType          string
	platformName     string
	// file             string
	// line             int
	isSimplify bool
}

func NewZapLogger(
	loggerPrefixName, moduleName string,
	logLevel int,
	isStdout bool,
	isJson bool,
	logLocation string,
	rotateCount uint,
	rotationTime uint,
	moduleVersion string,
	isSimplify bool,
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
	zl := &ZapLogger{level: logLevelMap[logLevel],
		moduleName:       moduleName,
		loggerPrefixName: loggerPrefixName,
		rotationTime:     time.Duration(rotationTime) * time.Hour,
		moduleVersion:    moduleVersion,
		isSimplify:       isSimplify,
	}
	opts, err := zl.cores(isStdout, isJson, logLocation, rotateCount)
	if err != nil {
		return nil, err
	}
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
	opts, err := zl.consoleCores(outPut, isJson)
	if err != nil {
		return nil, err
	}
	l, err := zapConfig.Build(opts)
	if err != nil {
		return nil, err
	}
	zl.zap = l.Sugar()
	return zl, nil
}

func NewSDKZapLogger(
	loggerPrefixName, moduleName string, sdkType, platformName string,
	logLevel int,
	isStdout bool,
	isJson bool,
	logLocation string,
	rotateCount uint,
	rotationTime uint,
	moduleVersion string,
	// file string,
	// line int,
	isSimplify bool,
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
	zl := &ZapLogger{level: logLevelMap[logLevel],
		moduleName:       moduleName,
		loggerPrefixName: loggerPrefixName,
		rotationTime:     time.Duration(rotationTime) * time.Hour,
		moduleVersion:    moduleVersion,
		sdkType:          sdkType,
		platformName:     platformName,
		// file:             file,
		// line:             line,
		isSimplify: isSimplify,
	}
	opts, err := zl.cores(isStdout, isJson, logLocation, rotateCount)
	if err != nil {
		return nil, err
	}
	l, err := zapConfig.Build(opts)
	if err != nil {
		return nil, err
	}
	zl.zap = l.Sugar()
	return zl, nil
}

func (l *ZapLogger) cores(isStdout bool, isJson bool, logLocation string, rotateCount uint) (zap.Option, error) {
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
		c.EncodeCaller = l.combinedCallerEncoder
		fileEncoder = zapcore.NewConsoleEncoder(c)
	}
	fileEncoder = &alignEncoder{Encoder: fileEncoder}
	writer, err := l.getWriter(logLocation, rotateCount)
	if err != nil {
		return nil, err
	}
	var cores []zapcore.Core
	if logLocation != "" {
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
	}), nil
}

func (l *ZapLogger) consoleCores(outPut *os.File, isJson bool) (zap.Option, error) {
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
		c.EncodeCaller = l.combinedCallerEncoder
		fileEncoder = zapcore.NewConsoleEncoder(c)
	}
	var cores []zapcore.Core
	cores = append(cores, zapcore.NewCore(fileEncoder, zapcore.Lock(outPut), zap.NewAtomicLevelAt(l.level)))

	return zap.WrapCore(func(c zapcore.Core) zapcore.Core {
		return zapcore.NewTee(cores...)
	}), nil
}

func (l *ZapLogger) customCallerEncoder(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
	fixedLength := 50
	trimmedPath := caller.TrimmedPath()
	trimmedPath = "[" + trimmedPath + "]"
	s := stringutil.FormatString(trimmedPath, fixedLength, true)
	enc.AppendString(s)
}

// func (l *ZapLogger) customCallerEncoderB(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
// 	fixedLength := 50
// 	trimmedPath := caller.TrimmedPath()
// 	sdkPlatform := fmt.Sprintf("[%s/%s]", l.sdkType, l.platformName)
// 	trimmedPath = fmt.Sprintf("%s [%s]", sdkPlatform, trimmedPath)
// 	s := stringutil.FormatString(trimmedPath, fixedLength, true)
// 	enc.AppendString(s)
// }

// func (l *ZapLogger) combinedCallerEncoder(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
// 	fixedLength := 50

// 	// 获取文件和行号
// 	trimmedPath := caller.TrimmedPath()
// 	trimmedPath = "[" + trimmedPath + "]"

// 	// 获取 sdkType 和 platformName
// 	sdkPlatform := fmt.Sprintf("[%s/%s]", l.sdkType, l.platformName)

// 	// 合并两部分
// 	combinedOutput := fmt.Sprintf("%s %s", sdkPlatform, trimmedPath)

// 	// 调整格式
// 	s := stringutil.FormatString(combinedOutput, fixedLength, true)

// 	// 输出到日志中
// 	enc.AppendString(s)
// }

func (l *ZapLogger) combinedCallerEncoder(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
	fixedLength := 25 // 控制 [sdkType/platformName] 的长度

	// 构造 [sdkType/platformName]
	sdkPlatform := fmt.Sprintf("[%s/%s]", l.sdkType, l.platformName)
	sdkPlatformFormatted := stringutil.FormatString(sdkPlatform, fixedLength, true)

	// // 构造 [file:line]
	// fileLine := fmt.Sprintf("[%s:%d]", caller.TrimmedPath(), caller.Line)
	// fileLineFormatted := stringutil.FormatString(fileLine, fixedLength, true)

	// 输出 [sdkType/platformName] 和 [file:line] 到日志中
	enc.AppendString(sdkPlatformFormatted)
	// enc.AppendString(fileLineFormatted)
}

func SDKLog(ctx context.Context, logLevel int, file string, line int, msg string, err error, keysAndValues []any) {
	// Add [file:line] convert

	// switch logLevel {
	// case 6:
	// 	// sdklog.SDKDebug(ctx, path, line, msg, keysAndValues)
	// 	ZDebug(ctx, msg, keysAndValues...)
	// case 4:
	// 	// sdklog.SDKInfo(ctx, path, line, msg, keysAndValues)
	// 	ZInfo(ctx, msg, keysAndValues...)
	// case 3:
	// 	// sdklog.SDKWarn(ctx, path, line, msg, errs.New(err), keysAndValues)
	// 	ZWarn(ctx, msg, err, keysAndValues...)
	// case 2:
	// 	// sdklog.SDKError(ctx, path, line, msg, errs.New(err), keysAndValues)
	// 	ZError(ctx, msg, err, keysAndValues...)
	// }

	// 设置自定义的 [file:line] 信息

	// 构造自定义的 Caller
	pkgLogger, ok := pkgLogger.(*ZapLogger)
	if !ok {
		// 处理错误
		return
	}

	customCaller := fmt.Sprintf("[%s:%d]", file, line)
	// loggerWithCaller := pkgLogger.WithValues("caller", customCaller)
	// loggerWithCaller := pkgLogger.ToZap().WithOptions(zap.WithCaller(false)).With(zap.String("caller", customCaller))
	// loggerWithCaller := pkgLogger.ToZap().WithOptions(zap.WithCaller(false)).With(zap.String("caller", customCaller))
	// pkgLogger.zap = pkgLogger.zap.With(zap.String("file", file), zap.Int("line", line))

	// 使用 zap.WithOptions 忽略原有的自动 caller 信息，并追加自定义的 caller 信息
	loggerWithCaller := pkgLogger.zap.WithOptions(zap.AddCallerSkip(1)).With(zap.String("caller", customCaller))
	fullMsg := fmt.Sprintf("%s\t%s", customCaller, msg)

	// 根据 logLevel 调用不同的日志级别
	switch logLevel {
	case 6:
		loggerWithCaller.Debugw(fullMsg, keysAndValues...)
		// loggerWithCaller.Debug(ctx, msg, keysAndValues...)
		// pkgLogger.Debug(ctx, msg, keysAndValues...)
	case 4:
		loggerWithCaller.Infow(fullMsg, keysAndValues...)
		// loggerWithCaller.Info(ctx, msg, keysAndValues...)
		// pkgLogger.Info(ctx, msg, keysAndValues...)
	case 3:
		loggerWithCaller.Warnw(fullMsg, keysAndValues...)
		// loggerWithCaller.Warn(ctx, msg, err, keysAndValues...)
		// pkgLogger.Warn(ctx, msg, err, keysAndValues...)
	case 2:
		loggerWithCaller.Errorw(fullMsg, keysAndValues...)
		// loggerWithCaller.Error(ctx, msg, err, keysAndValues...)
		// pkgLogger.Error(ctx, msg, err, keysAndValues...)
	}
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

func (l *ZapLogger) getWriter(logLocation string, rorateCount uint) (zapcore.WriteSyncer, error) {
	var path string
	if l.rotationTime%(time.Hour*time.Duration(hoursPerDay)) == 0 {
		path = logLocation + sp + l.loggerPrefixName + ".%Y-%m-%d"
	} else if l.rotationTime%time.Hour == 0 {
		path = logLocation + sp + l.loggerPrefixName + ".%Y-%m-%d_%H"
	} else {
		path = logLocation + sp + l.loggerPrefixName + ".%Y-%m-%d_%H_%M_%S"
	}
	logf, err := rotatelogs.New(path,
		rotatelogs.WithRotationCount(rorateCount),
		rotatelogs.WithRotationTime(l.rotationTime),
	)
	if err != nil {
		return nil, err
	}
	return zapcore.AddSync(logf), nil
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

	if l.isSimplify {
		if len(keysAndValues)%2 == 0 {
			for i := 1; i < len(keysAndValues); i += 2 {

				if val, ok := keysAndValues[i].(LogFormatter); ok && val != nil {
					keysAndValues[i] = val.Format()
				}
			}
		} else {
			ZError(ctx, "keysAndValues length is not even", nil)
		}
	}

	for i := 1; i < len(keysAndValues); i += 2 {
		if s, ok := keysAndValues[i].(interface{ String() string }); ok {
			keysAndValues[i] = s.String()
		}
	}

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
