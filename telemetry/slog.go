package telemetry

import (
	"fmt"
	"log/slog"
)

// SlogLogger implements Logger using a [log/slog](https://pkg.go.dev/log/slog) logger.
type SlogLogger struct {
	log *slog.Logger
	// stats map[string][]uint64
}

// NewSlogLogger creates a new SlogLogger using a given [slog.Logger]
//
// If enablePerf is false, all calls to Elapsed will be a no-op, otherwise,
// the elapsed statistics will be periodically printed to the console as
// DEBUG-level messages.
func NewSlogLogger(log *slog.Logger, enablePerf bool) SlogLogger {
	// var stats map[string][]uint64
	// if enablePerf {
	// 	stats = make(map[string][]uint64)
	// }
	return SlogLogger{
		log: log,
		// stats: stats,
	}
}

func (t SlogLogger) Debug(component, msg string, args ...any) {
	t.log.Debug(fmt.Sprintf("[%s] %s", component, msg), args...)
}

func (t SlogLogger) Info(component, msg string, args ...any) {
	t.log.Info(fmt.Sprintf("[%s] %s", component, msg), args...)
}

func (t SlogLogger) Warn(component, msg string, args ...any) {
	t.log.Warn(fmt.Sprintf("[%s] %s", component, msg), args...)
}

func (t SlogLogger) Error(component, msg string, args ...any) {
	t.log.Error(fmt.Sprintf("[%s] %s", component, msg), args...)
}

// func (t SlogLogger) Elapsed(component, operation string, milliseconds uint64) {
// 	if t.stats == nil {
// 		return
// 	}
// 	id := fmt.Sprintf("%s::%s", component, operation)
// 	t.stats[id] = append(t.stats[id], milliseconds)
//
// }
