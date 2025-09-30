package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wemix/wemixvisor/pkg/logger"
)

func TestNewValidator(t *testing.T) {
	logger := logger.NewTestLogger()
	validator := NewValidator(logger)

	assert.NotNil(t, validator)
	assert.NotEmpty(t, validator.rules)
}

func TestValidator_Validate(t *testing.T) {
	logger := logger.NewTestLogger()
	validator := NewValidator(logger)

	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
			errMsg:  "configuration is nil",
		},
		{
			name: "valid config",
			config: &Config{
				Home: "/tmp/wemixvisor",
				Name: "wemixd",
			},
			wantErr: false,
		},
		{
			name: "empty home",
			config: &Config{
				Home: "",
				Name: "wemixd",
			},
			wantErr: true,
			errMsg:  "home directory is required",
		},
		{
			name: "empty name",
			config: &Config{
				Home: "/tmp/wemixvisor",
				Name: "",
			},
			wantErr: true,
			errMsg:  "daemon name is required",
		},
		{
			name: "invalid name characters",
			config: &Config{
				Home: "/tmp/wemixvisor",
				Name: "wemix/d",
			},
			wantErr: true,
			errMsg:  "invalid characters",
		},
		{
			name: "invalid port",
			config: &Config{
				Home:    "/tmp/wemixvisor",
				Name:    "wemixd",
				RPCPort: 99999,
			},
			wantErr: true,
			errMsg:  "invalid",
		},
		{
			name: "negative shutdown grace",
			config: &Config{
				Home:          "/tmp/wemixvisor",
				Name:          "wemixd",
				ShutdownGrace: -1 * time.Second,
			},
			wantErr: true,
			errMsg:  "cannot be negative",
		},
		{
			name: "shutdown grace too long",
			config: &Config{
				Home:          "/tmp/wemixvisor",
				Name:          "wemixd",
				ShutdownGrace: 15 * time.Minute,
			},
			wantErr: true,
			errMsg:  "too long",
		},
		{
			name: "unsafe download without checksum",
			config: &Config{
				Home:                  "/tmp/wemixvisor",
				Name:                  "wemixd",
				AllowDownloadBinaries: true,
				UnsafeSkipChecksum:    true,
			},
			wantErr: true,
			errMsg:  "extremely unsafe",
		},
		{
			name: "incompatible validator settings",
			config: &Config{
				Home:          "/tmp/wemixvisor",
				Name:          "wemixd",
				ValidatorMode: true,
				DisableRecase: true,
			},
			wantErr: true,
			errMsg:  "requires recase",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidator_ValidateField(t *testing.T) {
	logger := logger.NewTestLogger()
	validator := NewValidator(logger)

	tests := []struct {
		name    string
		field   string
		value   interface{}
		wantErr bool
	}{
		{
			name:    "valid home path",
			field:   "home",
			value:   "/tmp/wemixvisor",
			wantErr: false,
		},
		{
			name:    "empty home path",
			field:   "home",
			value:   "",
			wantErr: true,
		},
		{
			name:    "valid name",
			field:   "name",
			value:   "wemixd",
			wantErr: false,
		},
		{
			name:    "invalid name",
			field:   "name",
			value:   "wemix/d",
			wantErr: true,
		},
		{
			name:    "valid port",
			field:   "rpc_port",
			value:   8545,
			wantErr: false,
		},
		{
			name:    "invalid port",
			field:   "rpc_port",
			value:   99999,
			wantErr: true,
		},
		{
			name:    "valid network ID",
			field:   "network_id",
			value:   uint64(1112),
			wantErr: false,
		},
		{
			name:    "zero network ID",
			field:   "network_id",
			value:   uint64(0),
			wantErr: true,
		},
		{
			name:    "valid chain ID",
			field:   "chain_id",
			value:   "1112",
			wantErr: false,
		},
		{
			name:    "invalid chain ID",
			field:   "chain_id",
			value:   "abc",
			wantErr: true,
		},
		{
			name:    "valid max peers",
			field:   "max_peers",
			value:   50,
			wantErr: false,
		},
		{
			name:    "negative max peers",
			field:   "max_peers",
			value:   -1,
			wantErr: true,
		},
		{
			name:    "too many max peers",
			field:   "max_peers",
			value:   300,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateField(tt.field, tt.value)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPathValidationRule(t *testing.T) {
	rule := &pathValidationRule{}
	assert.Equal(t, "PathValidation", rule.Name())

	tests := []struct {
		name    string
		config  *Config
		setup   func()
		wantErr bool
	}{
		{
			name: "valid paths",
			config: &Config{
				Home:           "/tmp/wemixvisor",
				DataBackupPath: "/tmp/backup",
			},
			wantErr: false,
		},
		{
			name: "empty home",
			config: &Config{
				Home: "",
			},
			wantErr: true,
		},
		{
			name: "home with tilde",
			config: &Config{
				Home: "~/wemixvisor",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}
			err := rule.Validate(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBinaryValidationRule(t *testing.T) {
	rule := &binaryValidationRule{}
	assert.Equal(t, "BinaryValidation", rule.Name())

	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid binary name",
			config: &Config{
				Name: "wemixd",
			},
			wantErr: false,
		},
		{
			name: "empty binary name",
			config: &Config{
				Name: "",
			},
			wantErr: true,
		},
		{
			name: "binary name with slash",
			config: &Config{
				Name: "wemix/d",
			},
			wantErr: true,
		},
		{
			name: "binary name with special chars",
			config: &Config{
				Name: "wemix*d",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rule.Validate(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPortValidationRule(t *testing.T) {
	rule := &portValidationRule{}
	assert.Equal(t, "PortValidation", rule.Name())

	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid port",
			config: &Config{
				RPCPort: 8545,
			},
			wantErr: false,
		},
		{
			name: "port zero",
			config: &Config{
				RPCPort: 0,
			},
			wantErr: false, // 0 means not specified
		},
		{
			name: "port too high",
			config: &Config{
				RPCPort: 99999,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rule.Validate(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNetworkValidationRule(t *testing.T) {
	rule := &networkValidationRule{}
	assert.Equal(t, "NetworkValidation", rule.Name())

	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid RPC address with port",
			config: &Config{
				RPCAddress: "localhost:8545",
			},
			wantErr: false,
		},
		{
			name: "valid IP address",
			config: &Config{
				RPCAddress: "127.0.0.1:8545",
			},
			wantErr: false,
		},
		{
			name: "invalid address",
			config: &Config{
				RPCAddress: "not-an-address",
			},
			wantErr: true,
		},
		{
			name: "empty RPC address",
			config: &Config{
				RPCAddress: "",
			},
			wantErr: false, // Empty is allowed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rule.Validate(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestResourceValidationRule(t *testing.T) {
	rule := &resourceValidationRule{}
	assert.Equal(t, "ResourceValidation", rule.Name())

	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid resources",
			config: &Config{
				ShutdownGrace:       30 * time.Second,
				PollInterval:        1 * time.Second,
				RestartDelay:        5 * time.Second,
				HealthCheckInterval: 30 * time.Second,
				MetricsInterval:     60 * time.Second,
				MaxRestarts:         5,
				PreUpgradeMaxRetries: 3,
			},
			wantErr: false,
		},
		{
			name: "negative shutdown grace",
			config: &Config{
				ShutdownGrace: -1 * time.Second,
			},
			wantErr: true,
			errMsg:  "cannot be negative",
		},
		{
			name: "shutdown grace too long",
			config: &Config{
				ShutdownGrace: 15 * time.Minute,
			},
			wantErr: true,
			errMsg:  "too long",
		},
		{
			name: "poll interval too short",
			config: &Config{
				PollInterval: 50 * time.Millisecond,
			},
			wantErr: true,
			errMsg:  "too short",
		},
		{
			name: "health check interval too short",
			config: &Config{
				HealthCheckInterval: 2 * time.Second,
			},
			wantErr: true,
			errMsg:  "too short",
		},
		{
			name: "metrics interval too short",
			config: &Config{
				MetricsInterval: 5 * time.Second,
			},
			wantErr: true,
			errMsg:  "too short",
		},
		{
			name: "negative max restarts",
			config: &Config{
				MaxRestarts: -1,
			},
			wantErr: true,
			errMsg:  "cannot be negative",
		},
		{
			name: "max restarts too high",
			config: &Config{
				MaxRestarts: 200,
			},
			wantErr: true,
			errMsg:  "too high",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rule.Validate(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSecurityValidationRule(t *testing.T) {
	rule := &securityValidationRule{}
	assert.Equal(t, "SecurityValidation", rule.Name())

	// Create a temp executable script
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "preupgrade.sh")
	require.NoError(t, os.WriteFile(scriptPath, []byte("#!/bin/sh\necho test"), 0755))

	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "safe config",
			config: &Config{
				AllowDownloadBinaries: false,
				UnsafeSkipChecksum:    false,
			},
			wantErr: false,
		},
		{
			name: "unsafe download without checksum",
			config: &Config{
				AllowDownloadBinaries: true,
				UnsafeSkipChecksum:    true,
			},
			wantErr: true,
			errMsg:  "extremely unsafe",
		},
		{
			name: "valid custom pre-upgrade script",
			config: &Config{
				CustomPreUpgrade: scriptPath,
			},
			wantErr: false,
		},
		{
			name: "non-existent pre-upgrade script",
			config: &Config{
				CustomPreUpgrade: "/nonexistent/script.sh",
			},
			wantErr: true,
			errMsg:  "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rule.Validate(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCompatibilityValidationRule(t *testing.T) {
	rule := &compatibilityValidationRule{}
	assert.Equal(t, "CompatibilityValidation", rule.Name())

	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "compatible settings",
			config: &Config{
				ValidatorMode: true,
				DisableRecase: false,
			},
			wantErr: false,
		},
		{
			name: "incompatible validator settings",
			config: &Config{
				ValidatorMode: true,
				DisableRecase: true,
			},
			wantErr: true,
			errMsg:  "requires recase",
		},
		{
			name: "valid time format",
			config: &Config{
				TimeFormatLogs: "rfc3339",
			},
			wantErr: false,
		},
		{
			name: "invalid time format",
			config: &Config{
				TimeFormatLogs: "invalid",
			},
			wantErr: true,
			errMsg:  "invalid time format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rule.Validate(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidator_AddRule(t *testing.T) {
	logger := logger.NewTestLogger()
	validator := NewValidator(logger)

	// Create custom rule
	customRule := &mockValidationRule{
		name: "CustomRule",
		validateFunc: func(cfg *Config) error {
			return nil
		},
	}

	// Add custom rule
	initialCount := len(validator.rules)
	validator.AddRule(customRule)
	assert.Equal(t, initialCount+1, len(validator.rules))
}

// mockValidationRule for testing
type mockValidationRule struct {
	name         string
	validateFunc func(*Config) error
}

func (r *mockValidationRule) Name() string {
	return r.name
}

func (r *mockValidationRule) Validate(cfg *Config) error {
	if r.validateFunc != nil {
		return r.validateFunc(cfg)
	}
	return nil
}