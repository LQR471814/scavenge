package scavenge

// Logger is an interface containing logging reporters.
//
// Note: the [args] of any logging methods ([Debug], [Info], [Warn], [Error]) are a list of keys and values in alternating order
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
