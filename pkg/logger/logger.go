package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger is a wrapper around zap.Logger
type Logger struct {
	*zap.Logger
}

// New creates a new logger instance
func New(colorLogs bool, disableLogs bool, timeFormat string) (*Logger, error) {
	if disableLogs {
		return &Logger{zap.NewNop()}, nil
	}

	var config zap.Config
	if colorLogs {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		config = zap.NewProductionConfig()
	}

	// Set time format
	switch timeFormat {
	case "kitchen":
		config.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("3:04PM")
	case "rfc3339":
		config.EncoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder
	case "rfc3339nano":
		config.EncoderConfig.EncodeTime = zapcore.RFC3339NanoTimeEncoder
	default:
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	// Always output to stdout
	config.OutputPaths = []string{"stdout"}
	config.ErrorOutputPaths = []string{"stderr"}

	zapLogger, err := config.Build()
	if err != nil {
		return nil, err
	}

	return &Logger{zapLogger}, nil
}

// Info logs an info message
func (l *Logger) Info(msg string, fields ...zap.Field) {
	l.Logger.Info(msg, fields...)
}

// Error logs an error message
func (l *Logger) Error(msg string, fields ...zap.Field) {
	l.Logger.Error(msg, fields...)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, fields ...zap.Field) {
	l.Logger.Warn(msg, fields...)
}

// Debug logs a debug message
func (l *Logger) Debug(msg string, fields ...zap.Field) {
	l.Logger.Debug(msg, fields...)
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(msg string, fields ...zap.Field) {
	l.Logger.Fatal(msg, fields...)
	os.Exit(1)
}

// With creates a child logger with additional fields
func (l *Logger) With(fields ...zap.Field) *Logger {
	return &Logger{l.Logger.With(fields...)}
}