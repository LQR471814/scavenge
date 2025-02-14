package telemetry

import "log/slog"

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

var defaultLogger Logger = NewSlogLogger(slog.Default(), true)

// DefaultLogger returns the default logger being used by all components in scavenge.
func DefaultLogger() Logger {
	return defaultLogger
}

// SetDefaultLogger sets the default logger being used by all components in scavenge.
func SetDefaultLogger(log Logger) {
	defaultLogger = log
}

// Debug calls Debug on the default Logger.
func Debug(component, msg string, args ...any) {
	defaultLogger.Debug(component, msg, args...)
}

// Info calls Info on the default Logger.
func Info(component, msg string, args ...any) {
	defaultLogger.Info(component, msg, args...)
}

// Warn calls Warn on the default Logger.
func Warn(component, msg string, args ...any) {
	defaultLogger.Warn(component, msg, args...)
}

// Error calls Error on the default Logger.
func Error(component, msg string, args ...any) {
	defaultLogger.Error(component, msg, args...)
}

var defaultPerf Perf
