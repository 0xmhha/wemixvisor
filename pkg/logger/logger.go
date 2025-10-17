package logger

import (
	"fmt"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger is a wrapper around zap.Logger
type Logger struct {
	*zap.Logger
}

// New creates a new logger instance
func New(debug bool, colorLogs bool, logFile string) (*Logger, error) {
	var config zap.Config
	if debug {
		config = zap.NewDevelopmentConfig()
		if colorLogs {
			config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		}
	} else {
		config = zap.NewProductionConfig()
	}

	// Set time format
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	// Configure output paths
	config.OutputPaths = []string{"stdout"}
	if logFile != "" {
		config.OutputPaths = append(config.OutputPaths, logFile)
	}
	config.ErrorOutputPaths = []string{"stderr"}

	zapLogger, err := config.Build()
	if err != nil {
		return nil, err
	}

	return &Logger{zapLogger}, nil
}

// Info logs an info message
func (l *Logger) Info(msg string, args ...interface{}) {
	l.Logger.Info(msg, toZapFields(args...)...)
}

// Error logs an error message
func (l *Logger) Error(msg string, args ...interface{}) {
	l.Logger.Error(msg, toZapFields(args...)...)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, args ...interface{}) {
	l.Logger.Warn(msg, toZapFields(args...)...)
}

// Debug logs a debug message
func (l *Logger) Debug(msg string, args ...interface{}) {
	l.Logger.Debug(msg, toZapFields(args...)...)
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(msg string, args ...interface{}) {
	l.Logger.Fatal(msg, toZapFields(args...)...)
	os.Exit(1)
}

// With creates a child logger with additional fields
func (l *Logger) With(fields ...zap.Field) *Logger {
	return &Logger{l.Logger.With(fields...)}
}

func toZapFields(args ...interface{}) []zap.Field {
	if len(args) == 0 {
		return nil
	}

	if field, ok := args[0].(zap.Field); ok {
		fields := make([]zap.Field, 0, len(args))
		fields = append(fields, field)
		for i := 1; i < len(args); i++ {
			if f, ok := args[i].(zap.Field); ok {
				fields = append(fields, f)
			}
		}
		return fields
	}

	fields := make([]zap.Field, 0, len(args)/2)
	for i := 0; i < len(args); {
		if i+1 >= len(args) {
			fields = append(fields, zap.Any(fmt.Sprintf("arg_%d", i), args[i]))
			break
		}

		key, ok := args[i].(string)
		if !ok {
			fields = append(fields, zap.Any(fmt.Sprintf("arg_%d", i), args[i]))
			i++
			continue
		}

		fields = append(fields, zap.Any(key, args[i+1]))
		i += 2
	}

	return fields
}
