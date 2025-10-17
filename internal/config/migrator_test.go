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

func TestNewMigrator(t *testing.T) {
	logger := logger.NewTestLogger()
	migrator := NewMigrator(logger)

	assert.NotNil(t, migrator)
	assert.NotEmpty(t, migrator.migrations)
}

func TestMigrator_Migrate(t *testing.T) {
	logger := logger.NewTestLogger()
	migrator := NewMigrator(logger)

	tests := []struct {
		name        string
		config      *Config
		fromVersion string
		toVersion   string
		wantErr     bool
		validate    func(t *testing.T, cfg *Config)
	}{
		{
			name:        "nil config",
			config:      nil,
			fromVersion: "0.1.0",
			toVersion:   "0.2.0",
			wantErr:     true,
		},
		{
			name: "same version",
			config: &Config{
				Home: "/tmp/wemixvisor",
				Name: "wemixd",
			},
			fromVersion: "0.2.0",
			toVersion:   "0.2.0",
			wantErr:     false,
		},
		{
			name: "downgrade attempt",
			config: &Config{
				Home: "/tmp/wemixvisor",
				Name: "wemixd",
			},
			fromVersion: "0.3.0",
			toVersion:   "0.2.0",
			wantErr:     true,
		},
		{
			name: "v0.1.0 to v0.2.0",
			config: &Config{
				Home: "/tmp/wemixvisor",
				Name: "wemixd",
			},
			fromVersion: "0.1.0",
			toVersion:   "0.2.0",
			wantErr:     false,
			validate: func(t *testing.T, cfg *Config) {
				assert.NotEmpty(t, cfg.DataBackupPath)
				assert.Equal(t, 30*time.Second, cfg.ShutdownGrace)
				assert.Equal(t, 3, cfg.PreUpgradeMaxRetries)
			},
		},
		{
			name: "v0.2.0 to v0.3.0",
			config: &Config{
				Home: "/tmp/wemixvisor",
				Name: "wemixd",
			},
			fromVersion: "0.2.0",
			toVersion:   "0.3.0",
			wantErr:     false,
			validate: func(t *testing.T, cfg *Config) {
				assert.NotEmpty(t, cfg.RPCAddress)
				assert.NotNil(t, cfg.DownloadURLs)
			},
		},
		{
			name: "v0.3.0 to v0.4.0",
			config: &Config{
				Home: "/tmp/wemixvisor",
				Name: "wemixd",
			},
			fromVersion: "0.3.0",
			toVersion:   "0.4.0",
			wantErr:     false,
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, 30*time.Second, cfg.HealthCheckInterval)
				assert.Equal(t, 60*time.Second, cfg.MetricsInterval)
				assert.Equal(t, 5, cfg.MaxRestarts)
				assert.Equal(t, 8545, cfg.RPCPort)
			},
		},
		{
			name: "v0.4.0 to v0.5.0",
			config: &Config{
				Home: "/tmp/wemixvisor",
				Name: "wemixd",
			},
			fromVersion: "0.4.0",
			toVersion:   "0.5.0",
			wantErr:     false,
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "0.5.0", cfg.ConfigVersion)
				assert.NotZero(t, cfg.NetworkID)
				assert.NotEmpty(t, cfg.ChainID)
			},
		},
		{
			name: "multi-version migration",
			config: &Config{
				Home: "/tmp/wemixvisor",
				Name: "wemixd",
			},
			fromVersion: "0.1.0",
			toVersion:   "0.5.0",
			wantErr:     false,
			validate: func(t *testing.T, cfg *Config) {
				// Should have all fields from all migrations
				assert.NotEmpty(t, cfg.DataBackupPath)
				assert.Equal(t, 30*time.Second, cfg.ShutdownGrace)
				assert.NotEmpty(t, cfg.RPCAddress)
				assert.NotNil(t, cfg.DownloadURLs)
				assert.Equal(t, 30*time.Second, cfg.HealthCheckInterval)
				assert.Equal(t, "0.5.0", cfg.ConfigVersion)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newConfig, err := migrator.Migrate(tt.config, tt.fromVersion, tt.toVersion)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, newConfig)
				if tt.validate != nil {
					tt.validate(t, newConfig)
				}
			}
		})
	}
}

func TestMigrator_BackupAndRestore(t *testing.T) {
	logger := logger.NewTestLogger()
	migrator := NewMigrator(logger)

	tmpDir := t.TempDir()
	backupPath := filepath.Join(tmpDir, "config")

	// Create test config
	originalConfig := &Config{
		Home:          "/tmp/wemixvisor",
		Name:          "wemixd",
		NetworkID:     1112,
		ChainID:       "1112",
		ConfigVersion: "0.5.0",
		MaxRestarts:   5,
	}

	// Backup config
	err := migrator.BackupConfig(originalConfig, backupPath)
	require.NoError(t, err)

	// Check backup file exists
	files, err := filepath.Glob(backupPath + "*.backup")
	require.NoError(t, err)
	require.Len(t, files, 1)

	// Restore config
	restoredConfig, err := migrator.RestoreConfig(files[0])
	require.NoError(t, err)

	// Compare configs
	assert.Equal(t, originalConfig.Home, restoredConfig.Home)
	assert.Equal(t, originalConfig.Name, restoredConfig.Name)
	assert.Equal(t, originalConfig.NetworkID, restoredConfig.NetworkID)
	assert.Equal(t, originalConfig.ChainID, restoredConfig.ChainID)
	assert.Equal(t, originalConfig.ConfigVersion, restoredConfig.ConfigVersion)
	assert.Equal(t, originalConfig.MaxRestarts, restoredConfig.MaxRestarts)
}

func TestMigrator_BackupConfig_Errors(t *testing.T) {
	logger := logger.NewTestLogger()
	migrator := NewMigrator(logger)

	tests := []struct {
		name    string
		config  *Config
		path    string
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			path:    "/tmp/backup",
			wantErr: true,
		},
		{
			name: "invalid path",
			config: &Config{
				Home: "/tmp/wemixvisor",
			},
			path:    "/nonexistent/deeply/nested/path/config",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := migrator.BackupConfig(tt.config, tt.path)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMigrator_RestoreConfig_Errors(t *testing.T) {
	logger := logger.NewTestLogger()
	migrator := NewMigrator(logger)

	tmpDir := t.TempDir()

	tests := []struct {
		name    string
		setup   func() string
		wantErr bool
	}{
		{
			name: "non-existent file",
			setup: func() string {
				return "/nonexistent/backup.json"
			},
			wantErr: true,
		},
		{
			name: "invalid JSON",
			setup: func() string {
				path := filepath.Join(tmpDir, "invalid.backup")
				os.WriteFile(path, []byte("not json"), 0644)
				return path
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backupPath := tt.setup()
			_, err := migrator.RestoreConfig(backupPath)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestVersion_ParseVersion(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *Version
		wantErr bool
	}{
		{
			name:  "standard version",
			input: "0.5.0",
			want:  &Version{Major: 0, Minor: 5, Patch: 0},
		},
		{
			name:  "version with v prefix",
			input: "v0.5.0",
			want:  &Version{Major: 0, Minor: 5, Patch: 0},
		},
		{
			name:  "version without patch",
			input: "1.2",
			want:  &Version{Major: 1, Minor: 2, Patch: 0},
		},
		{
			name:    "invalid version",
			input:   "invalid",
			wantErr: true,
		},
		{
			name:    "empty version",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseVersion(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestVersion_Comparison(t *testing.T) {
	v1 := &Version{Major: 1, Minor: 0, Patch: 0}
	v2 := &Version{Major: 1, Minor: 0, Patch: 0}
	v3 := &Version{Major: 1, Minor: 1, Patch: 0}
	v4 := &Version{Major: 2, Minor: 0, Patch: 0}
	v5 := &Version{Major: 1, Minor: 0, Patch: 1}

	// Test Equal
	assert.True(t, v1.Equal(v2))
	assert.False(t, v1.Equal(v3))

	// Test GreaterThan
	assert.True(t, v4.GreaterThan(v1))
	assert.True(t, v3.GreaterThan(v1))
	assert.True(t, v5.GreaterThan(v1))
	assert.False(t, v1.GreaterThan(v3))
	assert.False(t, v1.GreaterThan(v1))

	// Test LessThan
	assert.True(t, v1.LessThan(v4))
	assert.True(t, v1.LessThan(v3))
	assert.True(t, v1.LessThan(v5))
	assert.False(t, v3.LessThan(v1))
	assert.False(t, v1.LessThan(v1))
}

func TestVersion_String(t *testing.T) {
	v := &Version{Major: 1, Minor: 2, Patch: 3}
	assert.Equal(t, "1.2.3", v.String())
}

func TestMigrator_GetNextVersion(t *testing.T) {
	logger := logger.NewTestLogger()
	migrator := NewMigrator(logger)

	tests := []struct {
		name    string
		current *Version
		target  *Version
		want    *Version
	}{
		{
			name:    "next in sequence",
			current: &Version{Major: 0, Minor: 1, Patch: 0},
			target:  &Version{Major: 0, Minor: 5, Patch: 0},
			want:    &Version{Major: 0, Minor: 2, Patch: 0},
		},
		{
			name:    "direct to target",
			current: &Version{Major: 0, Minor: 4, Patch: 0},
			target:  &Version{Major: 0, Minor: 5, Patch: 0},
			want:    &Version{Major: 0, Minor: 5, Patch: 0},
		},
		{
			name:    "unknown version",
			current: &Version{Major: 99, Minor: 0, Patch: 0},
			target:  &Version{Major: 100, Minor: 0, Patch: 0},
			want:    &Version{Major: 100, Minor: 0, Patch: 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := migrator.getNextVersion(tt.current, tt.target)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMigrator_CloneConfig(t *testing.T) {
	logger := logger.NewTestLogger()
	migrator := NewMigrator(logger)

	original := &Config{
		Home:        "/tmp/wemixvisor",
		Name:        "wemixd",
		NetworkID:   1112,
		ChainID:     "1112",
		MaxRestarts: 5,
		DownloadURLs: map[string]string{
			"test": "url",
		},
	}

	// Clone config
	clone := migrator.cloneConfig(original)

	// Check that clone is equal but not the same object
	assert.Equal(t, original.Home, clone.Home)
	assert.Equal(t, original.Name, clone.Name)
	assert.Equal(t, original.NetworkID, clone.NetworkID)
	assert.Equal(t, original.ChainID, clone.ChainID)
	assert.Equal(t, original.MaxRestarts, clone.MaxRestarts)
	assert.Equal(t, original.DownloadURLs, clone.DownloadURLs)

	// Modify clone and ensure original is unchanged
	clone.Home = "/changed"
	clone.DownloadURLs["new"] = "value"
	assert.NotEqual(t, original.Home, clone.Home)
	assert.NotEqual(t, len(original.DownloadURLs), len(clone.DownloadURLs))
}

func TestMigrator_MigrationV1ToV2(t *testing.T) {
	logger := logger.NewTestLogger()
	migrator := NewMigrator(logger)

	cfg := &Config{
		Home: "/tmp/wemixvisor",
		Name: "wemixd",
	}

	result, err := migrator.migrateV1ToV2(cfg)
	require.NoError(t, err)

	assert.Equal(t, filepath.Join("/tmp/wemixvisor", "backups"), result.DataBackupPath)
	assert.Equal(t, 30*time.Second, result.ShutdownGrace)
	assert.Equal(t, 3, result.PreUpgradeMaxRetries)
}

func TestMigrator_MigrationV2ToV3(t *testing.T) {
	logger := logger.NewTestLogger()
	migrator := NewMigrator(logger)

	cfg := &Config{
		Home: "/tmp/wemixvisor",
		Name: "wemixd",
	}

	result, err := migrator.migrateV2ToV3(cfg)
	require.NoError(t, err)

	assert.Equal(t, "localhost:8545", result.RPCAddress)
	assert.NotNil(t, result.DownloadURLs)
}

func TestMigrator_MigrationV3ToV4(t *testing.T) {
	logger := logger.NewTestLogger()
	migrator := NewMigrator(logger)

	cfg := &Config{
		Home:       "/tmp/wemixvisor",
		Name:       "wemixd",
		RPCAddress: "localhost:8545",
	}

	result, err := migrator.migrateV3ToV4(cfg)
	require.NoError(t, err)

	assert.Equal(t, 30*time.Second, result.HealthCheckInterval)
	assert.Equal(t, 60*time.Second, result.MetricsInterval)
	assert.Equal(t, 5, result.MaxRestarts)
	assert.Equal(t, 8545, result.RPCPort)
}

func TestMigrator_MigrationV4ToV5(t *testing.T) {
	logger := logger.NewTestLogger()
	migrator := NewMigrator(logger)

	tests := []struct {
		name     string
		config   *Config
		validate func(t *testing.T, cfg *Config)
	}{
		{
			name: "validator node",
			config: &Config{
				Home:          "/tmp/wemixvisor",
				Name:          "wemixd",
				ValidatorMode: true,
			},
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "0.5.0", cfg.ConfigVersion)
				assert.Equal(t, uint64(1111), cfg.NetworkID)
				assert.Equal(t, "1111", cfg.ChainID)
			},
		},
		{
			name: "non-validator node",
			config: &Config{
				Home:          "/tmp/wemixvisor",
				Name:          "wemixd",
				ValidatorMode: false,
			},
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "0.5.0", cfg.ConfigVersion)
				assert.Equal(t, uint64(1112), cfg.NetworkID)
				assert.Equal(t, "1112", cfg.ChainID)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := migrator.migrateV4ToV5(tt.config)
			require.NoError(t, err)
			tt.validate(t, result)
		})
	}
}

func TestMigrator_GenericMigration(t *testing.T) {
	logger := logger.NewTestLogger()
	migrator := NewMigrator(logger)

	tests := []struct {
		name   string
		config *Config
	}{
		{
			name:   "empty config",
			config: &Config{},
		},
		{
			name: "partial config",
			config: &Config{
				Home: "/custom",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := migrator.genericMigration(tt.config)
			require.NoError(t, err)
			assert.NotEmpty(t, result.Home)
			assert.Equal(t, "wemixd", result.Name)
		})
	}
}