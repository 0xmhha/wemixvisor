package logger

import "go.uber.org/zap"

// NewTestLogger creates a logger for testing
func NewTestLogger() *Logger {
	// Use a no-op logger for tests to avoid output
	return &Logger{zap.NewNop()}
}

// NewDevelopmentLogger creates a development logger for debugging tests
func NewDevelopmentLogger() *Logger {
	logger, _ := New(true, false, "")
	return logger
}