package cascade

import (
	"context"
	"fmt"
	stdlog "log"
	"strings"
	"sync"

	sdklog "github.com/LumeraProtocol/sdk-go/pkg/log"
)

// supernodeLogger adapts sdk-go logging to the SuperNode SDK logger interface.
// It defaults to stdout when no sdk-go logger is configured.
type supernodeLogger struct {
	mu     sync.RWMutex
	logger sdklog.Logger
}

func newSupernodeLogger(logger sdklog.Logger) *supernodeLogger {
	return &supernodeLogger{logger: logger}
}

func (l *supernodeLogger) SetLogger(logger sdklog.Logger) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logger = logger
}

func (l *supernodeLogger) Debug(ctx context.Context, msg string, keysAndValues ...interface{}) {
	l.log(ctx, "DEBUG", msg, keysAndValues...)
}

func (l *supernodeLogger) Info(ctx context.Context, msg string, keysAndValues ...interface{}) {
	l.log(ctx, "INFO", msg, keysAndValues...)
}

func (l *supernodeLogger) Warn(ctx context.Context, msg string, keysAndValues ...interface{}) {
	l.log(ctx, "WARN", msg, keysAndValues...)
}

func (l *supernodeLogger) Error(ctx context.Context, msg string, keysAndValues ...interface{}) {
	l.log(ctx, "ERROR", msg, keysAndValues...)
}

func (l *supernodeLogger) log(ctx context.Context, level, msg string, keysAndValues ...interface{}) {
	_ = ctx
	line := fmt.Sprintf("[supernode] %s %s%s", level, msg, formatKV(keysAndValues))
	l.mu.RLock()
	logger := l.logger
	l.mu.RUnlock()
	if logger == nil {
		stdlog.Print(line)
		return
	}
	sdklog.Infof(logger, "%s", line)
}

func formatKV(keysAndValues []interface{}) string {
	if len(keysAndValues) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(" ")
	for i := 0; i < len(keysAndValues); i += 2 {
		if i > 0 {
			b.WriteString(" ")
		}
		key := keysAndValues[i]
		val := "(missing)"
		if i+1 < len(keysAndValues) {
			val = fmt.Sprintf("%v", keysAndValues[i+1])
		}
		b.WriteString(fmt.Sprintf("%v=%v", key, val))
	}
	return b.String()
}
