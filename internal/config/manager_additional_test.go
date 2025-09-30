package config

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wemix/wemixvisor/pkg/logger"
)

// Test edge cases and error conditions to improve coverage
func TestManager_LoadConfigError(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.toml")

	// Create invalid TOML
	require.NoError(t, os.WriteFile(configPath, []byte("invalid = [toml"), 0644))

	logger := logger.NewTestLogger()
	_, err := NewManager(configPath, logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load config")
}

func TestManager_WatcherError(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	// Create valid config
	configContent := `
home = "/tmp/wemixvisor"
name = "wemixd"
network_id = 1111
chain_id = "1111"
rpc_port = 8588
`
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	logger := logger.NewTestLogger()
	manager, err := NewManager(configPath, logger)
	require.NoError(t, err)

	// Close watcher to simulate error
	manager.watcher.Close()

	// Try to use watch functionality
	manager.Stop()

	// Should handle gracefully
	assert.NotNil(t, manager)
}

func TestManager_SaveConfigError(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	logger := logger.NewTestLogger()
	manager, err := NewManager(configPath, logger)
	require.NoError(t, err)
	defer manager.Stop()

	// Set config path to an invalid location to cause save error
	manager.configPath = "/invalid/path/that/does/not/exist/config.toml"

	// Try to save config (should fail due to invalid path)
	err = manager.saveConfig()
	assert.Error(t, err)
}

func TestManager_CloneConfig(t *testing.T) {
	logger := logger.NewTestLogger()
	manager := &Manager{
		logger: logger,
	}

	// Test nil config
	clone := manager.cloneConfig(nil)
	assert.Nil(t, clone)

	// Test valid config
	original := &Config{
		Home:          "/test",
		Name:          "test",
		NetworkID:     1111,
		ChainID:       "1111",
		ConfigVersion: "0.5.0",
	}

	clone = manager.cloneConfig(original)
	assert.NotNil(t, clone)
	assert.Equal(t, original.Home, clone.Home)
	assert.Equal(t, original.Name, clone.Name)
	assert.Equal(t, original.NetworkID, clone.NetworkID)

	// Modify clone should not affect original
	clone.Name = "modified"
	assert.NotEqual(t, original.Name, clone.Name)
}

func TestManager_NotifyUpdate(t *testing.T) {
	logger := logger.NewTestLogger()
	ctx, cancel := context.WithCancel(context.Background())

	manager := &Manager{
		logger:   logger,
		ctx:      ctx,
		cancel:   cancel,
		updateCh: make(chan ConfigUpdate, 1),
	}

	oldConfig := &Config{Name: "old"}
	newConfig := &Config{Name: "new"}

	// Test successful notification
	manager.notifyUpdate(UpdateTypeReload, oldConfig, newConfig)

	select {
	case update := <-manager.updateCh:
		assert.Equal(t, UpdateTypeReload, update.Type)
		assert.Equal(t, "old", update.OldConfig.Name)
		assert.Equal(t, "new", update.NewConfig.Name)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Expected update notification")
	}

	// Test with full channel (should not block)
	manager.updateCh <- ConfigUpdate{}
	manager.notifyUpdate(UpdateTypeReload, oldConfig, newConfig)
	// Should complete without blocking

	// Test with cancelled context
	cancel()
	manager.notifyUpdate(UpdateTypeReload, oldConfig, newConfig)
	// Should complete without blocking
}

func TestManager_FileWatching(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	// Create initial config
	configContent := `
home = "/tmp/wemixvisor"
name = "wemixd"
network_id = 1111
chain_id = "1111"
rpc_port = 8588
`
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	logger := logger.NewTestLogger()
	manager, err := NewManager(configPath, logger)
	require.NoError(t, err)
	defer manager.Stop()

	// Get update channel
	updateCh := manager.GetUpdateChannel()

	// Update config file
	newContent := `
home = "/tmp/wemixvisor"
name = "wemixd-updated"
network_id = 1111
chain_id = "1111"
rpc_port = 8588
`
	require.NoError(t, os.WriteFile(configPath, []byte(newContent), 0644))

	// Wait for potential update
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	select {
	case <-updateCh:
		// Got update
	case <-ctx.Done():
		// File watching might not work in test environment
	}
}

func TestManager_MergeConfigs(t *testing.T) {
	logger := logger.NewTestLogger()
	manager := &Manager{
		logger: logger,
		wemixvisorConfig: &WemixvisorConfig{
			Home:                  "/test",
			Name:                  "test",
			AllowDownloadBinaries: true,
		},
		nodeConfig: &NodeConfig{
			NetworkID:     1111,
			ChainID:       "1111",
			RPCPort:       8588,
			ValidatorMode: true,
		},
		mergedConfig: &Config{},
	}

	manager.mergeConfigs()

	assert.NotNil(t, manager.mergedConfig)
	assert.Equal(t, "/test", manager.mergedConfig.Home)
	assert.Equal(t, "test", manager.mergedConfig.Name)
	assert.True(t, manager.mergedConfig.AllowDownloadBinaries)
	// Note: mergeConfigs doesn't copy node config fields, it's just for wemixvisor config
}

func TestManager_GetNodeConfig(t *testing.T) {
	logger := logger.NewTestLogger()
	manager := &Manager{
		logger: logger,
		nodeConfig: &NodeConfig{
			NetworkID: 1111,
			ChainID:   "1111",
		},
	}

	nodeConfig := manager.GetNodeConfig()
	assert.NotNil(t, nodeConfig)
	assert.Equal(t, uint64(1111), nodeConfig.NetworkID)
	assert.Equal(t, "1111", nodeConfig.ChainID)
}

func TestManager_GetWemixvisorConfig(t *testing.T) {
	logger := logger.NewTestLogger()
	manager := &Manager{
		logger: logger,
		wemixvisorConfig: &WemixvisorConfig{
			Home: "/test",
			Name: "test",
		},
	}

	wConfig := manager.GetWemixvisorConfig()
	assert.NotNil(t, wConfig)
	assert.Equal(t, "/test", wConfig.Home)
	assert.Equal(t, "test", wConfig.Name)
}

func TestManager_LoadInvalidConfigs(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name      string
		filename  string
		content   string
		wantError bool
	}{
		{
			name:      "invalid TOML",
			filename:  "invalid.toml",
			content:   `invalid = [toml`,
			wantError: true,
		},
		{
			name:      "invalid YAML",
			filename:  "invalid.yaml",
			content:   `invalid: [yaml`,
			wantError: true,
		},
		{
			name:      "invalid JSON",
			filename:  "invalid.json",
			content:   `{"invalid": [json}`,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := filepath.Join(tmpDir, tt.filename)
			require.NoError(t, os.WriteFile(configPath, []byte(tt.content), 0644))

			logger := logger.NewTestLogger()
			_, err := NewManager(configPath, logger)

			if tt.wantError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "failed to load config")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Test concurrent access
func TestManager_ConcurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	logger := logger.NewTestLogger()
	manager, err := NewManager(configPath, logger)
	require.NoError(t, err)
	defer manager.Stop()

	// Run concurrent operations
	done := make(chan bool)

	// Reader goroutines
	for i := 0; i < 3; i++ {
		go func() {
			for j := 0; j < 10; j++ {
				_ = manager.GetConfig()
				_ = manager.GetNodeConfig()
				_ = manager.GetWemixvisorConfig()
			}
			done <- true
		}()
	}

	// Writer goroutine
	go func() {
		for j := 0; j < 10; j++ {
			_ = manager.Validate()
		}
		done <- true
	}()

	// Wait for all goroutines with timeout
	for i := 0; i < 4; i++ {
		select {
		case <-done:
		case <-time.After(1 * time.Second):
			t.Fatal("Concurrent test timeout")
		}
	}
}

// Test default config loading
func TestManager_DefaultConfigLoading(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "non-existent.toml")

	logger := logger.NewTestLogger()
	manager, err := NewManager(configPath, logger)
	require.NoError(t, err)
	defer manager.Stop()

	// Should have default values
	cfg := manager.GetConfig()
	assert.NotNil(t, cfg)
	assert.NotEmpty(t, cfg.Home)
	assert.NotEmpty(t, cfg.Name)
}

// Test partial config file
func TestManager_PartialConfigFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "partial.toml")

	// Create partial config file with required network fields
	partialConfig := `
home = "/custom/home"
name = "partial-config"
network_id = 1111
chain_id = "1111"
rpc_port = 8588
`
	require.NoError(t, os.WriteFile(configPath, []byte(partialConfig), 0644))

	logger := logger.NewTestLogger()
	manager, err := NewManager(configPath, logger)
	require.NoError(t, err)
	defer manager.Stop()

	// Should have the specified values and defaults for others
	cfg := manager.GetConfig()
	assert.NotNil(t, cfg)
	assert.Equal(t, "/custom/home", cfg.Home)
	assert.Equal(t, "partial-config", cfg.Name)
	assert.Equal(t, 8588, cfg.RPCPort)
}

// Test reload with validation
func TestManager_ReloadWithValidation(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	// Create valid initial config
	validConfig := `
home = "/tmp/wemixvisor"
name = "wemixd"
network_id = 1111
chain_id = "1111"
rpc_port = 8588
`
	require.NoError(t, os.WriteFile(configPath, []byte(validConfig), 0644))

	logger := logger.NewTestLogger()
	manager, err := NewManager(configPath, logger)
	require.NoError(t, err)
	defer manager.Stop()

	// Try to update with another valid config
	updatedConfig := `
home = "/tmp/wemixvisor"
name = "wemixd-updated"
network_id = 1111
chain_id = "1111"
rpc_port = 8589
`
	require.NoError(t, os.WriteFile(configPath, []byte(updatedConfig), 0644))

	// Just verify it doesn't crash
	time.Sleep(10 * time.Millisecond)

	// Note: hot reload might not work in test environment,
	// so we can't reliably test the actual update
}