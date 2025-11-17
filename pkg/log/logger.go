package log

// Logger is a minimal interface compatible with stdlib loggers.
type Logger interface {
	Printf(format string, v ...interface{})
}

// NoopLogger discards all log messages.
type NoopLogger struct{}

func (NoopLogger) Printf(string, ...interface{}) {}
