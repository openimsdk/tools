package log

import "context"

type Logger interface {
	// Debug logs a message at the debug level including any supplementary key-value pairs.
	// Useful for detailed output for debugging purposes.
	Debug(ctx context.Context, msg string, keysAndValues ...any)

	// Info logs a message at the info level along with any supplementary key-value pairs.
	// Ideal for general operational messages that inform about the system's state.
	Info(ctx context.Context, msg string, keysAndValues ...any)

	// Warn logs a message at the warning level, indicating potential issues in the system.
	// It includes an error object and any supplementary key-value pairs.
	Warn(ctx context.Context, msg string, err error, keysAndValues ...any)

	// Error logs a message at the error level, indicating serious problems that need attention.
	// It includes an error object and any supplementary key-value pairs.
	Error(ctx context.Context, msg string, err error, keysAndValues ...any)

	// Panic logs a message at the panic level, indicating a critical error like nil pointer exception that requires immediate attention.
	// It includes an error object and any supplementary key-value pairs.
	Panic(ctx context.Context, msg string, err error, keysAndValues ...any)

	// WithValues returns a new Logger instance that will include the specified key-value pairs
	// in all subsequent log messages. Useful for adding consistent context to a series of logs.
	WithValues(keysAndValues ...any) Logger

	// WithName returns a new Logger instance prefixed with the specified name.
	// This is helpful for distinguishing logs generated from different sources or components.
	WithName(name string) Logger

	// WithCallDepth returns a new Logger instance that adjusts the call depth for identifying
	// the source of log messages. Useful in wrapper or middleware layers to maintain accurate log source information.
	WithCallDepth(depth int) Logger

	// Flush flushes any buffered log entries, ensuring that all logs are written out.
	Flush()
}
