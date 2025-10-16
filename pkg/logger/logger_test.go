package logger

import (
	"testing"

	"go.uber.org/zap"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name        string
		colorLogs   bool
		disableLogs bool
		timeFormat  string
		wantErr     bool
	}{
		{
			name:        "color logs enabled",
			colorLogs:   true,
			disableLogs: false,
			timeFormat:  "kitchen",
			wantErr:     false,
		},
		{
			name:        "color logs disabled",
			colorLogs:   false,
			disableLogs: false,
			timeFormat:  "rfc3339",
			wantErr:     false,
		},
		{
			name:        "logs disabled",
			colorLogs:   true,
			disableLogs: true,
			timeFormat:  "rfc3339nano",
			wantErr:     false,
		},
		{
			name:        "default time format",
			colorLogs:   false,
			disableLogs: false,
			timeFormat:  "iso8601",
			wantErr:     false,
		},
		{
			name:        "unknown time format uses default",
			colorLogs:   false,
			disableLogs: false,
			timeFormat:  "unknown",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := New(tt.colorLogs, tt.disableLogs, tt.timeFormat)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if logger == nil {
				t.Error("expected logger to be non-nil")
			}

			// Test that logger is functional
			if !tt.disableLogs {
				// These should not panic
				logger.Info("test info")
				logger.Debug("test debug")
				logger.Warn("test warn")
				logger.Error("test error")
			}
		})
	}
}

func TestLoggerMethods(t *testing.T) {
	logger, err := New(false, false, "kitchen")
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	// Test Info
	logger.Info("test info message", zap.String("key", "value"))

	// Test Debug
	logger.Debug("test debug message", zap.Int("count", 42))

	// Test Warn
	logger.Warn("test warning message", zap.Bool("flag", true))

	// Test Error
	logger.Error("test error message", zap.Error(nil))

	// Test With
	childLogger := logger.With(zap.String("component", "test"))
	childLogger.Info("child logger message")
}

func TestLoggerWithDisabled(t *testing.T) {
	logger, err := New(true, true, "kitchen")
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	// These should not produce any output when disabled
	logger.Info("should not be visible")
	logger.Debug("should not be visible")
	logger.Warn("should not be visible")
	logger.Error("should not be visible")

	// With should also work with disabled logger
	childLogger := logger.With(zap.String("component", "test"))
	childLogger.Info("child logger should not be visible")
}
