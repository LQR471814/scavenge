package scavenge

import "context"

// Logger is an interface containing logging reporters.
//
// Note: the args of any logging methods (Debug, Info, Warn, Error) are a list of keys and values in alternating order
// like it is in [log/slog](https://pkg.go.dev/log/slog).
type Logger interface {
	Debug(component, msg string, args ...any)
	Info(component, msg string, args ...any)
	Warn(component, msg string, args ...any)
	Error(component, msg string, args ...any)
}

// Perf is an interface containing performance reporters.
type Perf interface {
	Elapsed(component, operation string, milliseconds uint64)
}

type logCtxKeyType int

var logCtxKey logCtxKeyType

func setLogCtx(ctx context.Context, log Logger) context.Context {
	return context.WithValue(ctx, logCtxKey, log)
}

// GetLoggerFromContext retrieves a Logger from the given context,
// it will panic if the Logger is not there.
func GetLoggerFromContext(ctx context.Context) Logger {
	value := ctx.Value(logCtxKey).(Logger)
	return value
}
