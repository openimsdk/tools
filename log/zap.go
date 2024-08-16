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
	file             string
	line             int
	isSimplify       bool
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
	fixedLength := 50 // 控制 [sdkType/platformName] 的长度

	// 构造 [sdkType/platformName]
	sdkPlatform := fmt.Sprintf("[%s/%s]", l.sdkType, l.platformName)
	sdkPlatformFormatted := stringutil.FormatString(sdkPlatform, fixedLength, true)

	// // 构造 [file:line]
	// fileLine := fmt.Sprintf("[%s:%d]", caller.TrimmedPath(), caller.Line)
	fileLine := fmt.Sprintf("[%s:%d]", caller.File, caller.Line)
	fileLineFormatted := stringutil.FormatString(fileLine, fixedLength, true)

	// 输出 [sdkType/platformName] 和 [file:line] 到日志中
	enc.AppendString(sdkPlatformFormatted)
	enc.AppendString(fileLineFormatted)
}

// type customCallerCore struct {
// 	zapcore.Core
// 	file string
// 	line int
// }

// func (c *customCallerCore) With(fields []zapcore.Field) zapcore.Core {
// 	// 保持原有 core 的行为，传递 fields
// 	return &customCallerCore{
// 		Core: c.Core.With(fields),
// 		file: c.file,
// 		line: c.line,
// 	}
// }

// func (c *customCallerCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
// 	// // 在写日志时，不替换原有 caller，而是在日志字段中追加自定义的 caller 信息
// 	// newFields := append(fields, zap.String("customCaller", fmt.Sprintf("[%s:%d]", c.file, c.line)))
// 	// return c.Core.Write(entry, newFields)

// 	// 替换 Entry.Caller 为自定义的 file 和 line 信息
// 	entry.Caller = zapcore.EntryCaller{
// 		File:    c.file,
// 		Line:    c.line,
// 		Defined: true, // 表示该 Caller 已经定义
// 	}
// 	return c.Core.Write(entry, fields)
// }

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
	// ZDebug(ctx, "pkgLogger contents", )

	// // 构造自定义的 EntryCaller 信息
	// customCaller := zapcore.EntryCaller{
	// 	File: file,
	// 	Line: line,
	// }

	// 创建自定义的 Core，附加自定义的 Caller 信息
	// customCore := &customCallerCore{
	// 	Core: pkgLogger.zap.Desugar().Core(),
	// 	file: file,
	// 	line: line,
	// }

	// loggerWithCaller := pkgLogger.zap.Desugar().WithOptions(zap.WrapCore(func(c zapcore.Core) zapcore.Core {
	// 	return customCore
	// })).Sugar()

	customCaller := fmt.Sprintf("[%s:%d]", file, line)
	// loggerWithCaller := pkgLogger.WithValues("caller", customCaller)
	// loggerWithCaller := pkgLogger.ToZap().WithOptions(zap.WithCaller(false)).With(zap.String("caller", customCaller))
	// loggerWithCaller := pkgLogger.ToZap().WithOptions(zap.WithCaller(false)).With(zap.String("caller", customCaller))
	// pkgLogger.zap = pkgLogger.zap.With(zap.String("file", file), zap.Int("line", line))

	// 使用 zap.WithOptions 忽略原有的自动 caller 信息，并追加自定义的 caller 信息
	// loggerWithCaller := pkgLogger.zap.WithOptions(zap.AddCallerSkip(-1)).With(zap.String("caller", customCaller))
	// 生成自定义的 zap logger，将 caller 替换为自定义的 caller 信息

	// loggerWithCaller := pkgLogger.zap.With(zap.String("caller", fmt.Sprintf("[%s:%d]", customCaller.File, customCaller.Line)))
	loggerWithCaller := pkgLogger.zap.With(zap.String("caller", customCaller))

	// fullMsg := fmt.Sprintf("%s\t%s", customCaller, msg)
	fullMsg := msg

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

func SDKLog666(ctx context.Context, logLevel int, file string, line int, msg string, err error, keysAndValues []any) {
	// 确保 pkgLogger 是正确的类型
	pkgLogger, ok := pkgLogger.(*ZapLogger)
	if !ok {
		// 处理错误
		return
	}

	// // 构造自定义的 Caller 信息
	// customCaller := zapcore.EntryCaller{
	// 	Defined: true,
	// 	File:    file,
	// 	Line:    line,
	// }

	// 创建一个自定义的 Core，并设置 file 和 line

	// customCore := &customCallerCore{
	// 	Core: pkgLogger.zap.Desugar().Core(),
	// 	file: file,
	// 	line: line,
	// }

	core := &customCheckedCore{
		Core: pkgLogger.zap.Desugar().Core(),
		file: file,
		line: line,
	}
	

	pkgLogger.zap.Desugar().Core().Check(pkgLogger.zap.Desugar().Core().Write(zapcore.Entry, []zapcore.Field), *zapcore.CheckedEntry)
	pkgLogger.zap.Desugar().WithOptions(zapcore.)

	// 包装 Core 并传入自定义的 Caller 信息
	loggerWithCustomCaller := pkgLogger.zap.Desugar().WithOptions(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
		// return customCore
		return &customCheckedCore{
			Core: core,
			file: file,
			line: line,
		}
	})).Sugar()

	// // 使用自定义的 caller
	// loggerWithCaller := pkgLogger.ToZap().WithOptions(zap.WithCaller(false)).With(zap.String("caller", fmt.Sprintf("[%s:%d]", customCaller.File, customCaller.Line)))

	// // 设置自定义的 file 和 line 信息
	// pkgLogger.file = file
	// pkgLogger.line = line

	// loggerWithCaller := pkgLogger.zap

	// 根据 logLevel 调用不同的日志级别
	switch logLevel {
	case 6:
		loggerWithCustomCaller.Debugw(msg, keysAndValues...)
	case 4:
		loggerWithCustomCaller.Infow(msg, keysAndValues...)
	case 3:
		loggerWithCustomCaller.Warnw(msg, keysAndValues...)
	case 2:
		loggerWithCustomCaller.Errorw(msg, keysAndValues...)
	}
}

// customCheckedCore wraps a zapcore.Core and modifies the file and line in CheckedEntry.
type customCheckedCore struct {
	zapcore.Core
	file string
	line int
}

// Check intercepts the logging decision and modifies the Entry's Caller.
func (c *customCheckedCore) Check(entry zapcore.Entry, checked *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	// 调用原始的 Check 逻辑
	checkedEntry := c.Core.Check(entry, checked)
	if checkedEntry != nil {
		// 修改 Caller 信息
		entry.Caller = zapcore.EntryCaller{
			File:    c.file,
			Line:    c.line,
			Defined: true,
		}
		// 将自定义的 Entry 注入到 CheckedEntry 中
		checkedEntry = checkedEntry.AddCore(entry, c)
	}
	return checkedEntry
}

// Write writes the modified Entry and preserves the original behavior.
func (c *customCheckedCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	// 保持原有的写日志逻辑
	return c.Core.Write(entry, fields)
}

// With ensures that fields are added to the Core while preserving the custom behavior.
func (c *customCheckedCore) With(fields []zapcore.Field) zapcore.Core {
	return &customCheckedCore{
		Core: c.Core.With(fields),
		file: c.file,
		line: c.line,
	}
}

func SDKLogG(ctx context.Context, logLevel int, file string, line int, msg string, err error, keysAndValues []any) {

}
