package log

// Logger is a minimal interface compatible with stdlib loggers.
type Logger interface {
	Printf(format string, v ...interface{})
}

// NoopLogger discards all log messages.
type NoopLogger struct{}

func (NoopLogger) Printf(string, ...interface{}) {}
func (NoopLogger) Infof(string, ...interface{})  {}
func (NoopLogger) Warnf(string, ...interface{})  {}

type infoLogger interface {
	Infof(format string, v ...interface{})
}

type warnLogger interface {
	Warnf(format string, v ...interface{})
}

// Infof logs at info level when supported, otherwise falls back to Printf.
func Infof(logger Logger, format string, v ...interface{}) {
	if logger == nil {
		return
	}
	if il, ok := logger.(infoLogger); ok {
		il.Infof(format, v...)
		return
	}
	logger.Printf(format, v...)
}

// Warnf logs at warn level when supported, otherwise falls back to Printf.
func Warnf(logger Logger, format string, v ...interface{}) {
	if logger == nil {
		return
	}
	if wl, ok := logger.(warnLogger); ok {
		wl.Warnf(format, v...)
		return
	}
	logger.Printf(format, v...)
}
