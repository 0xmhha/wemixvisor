package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	"testing"
)

// NewTestLogger creates a logger suitable for testing
func NewTestLogger() *Logger {
	return &Logger{zap.NewNop()}
}

// NewTestLoggerWithT creates a test logger that writes to testing.T
func NewTestLoggerWithT(t *testing.T) *Logger {
	return &Logger{zaptest.NewLogger(t)}
}

// Writer returns an io.Writer for the logger (for compatibility)
func (l *Logger) Writer() *LogWriter {
	return &LogWriter{logger: l}
}

// LogWriter implements io.Writer interface for logger
type LogWriter struct {
	logger *Logger
}

// Write implements io.Writer
func (lw *LogWriter) Write(p []byte) (n int, err error) {
	lw.logger.Info(string(p))
	return len(p), nil
}