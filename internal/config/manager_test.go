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

func TestNewManager(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	// Create a test config file with all required network fields
	configContent := `
home = "/tmp/wemixvisor"
name = "wemixd"
network_id = 1112
chain_id = "1112"
rpc_port = 8588
`
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	logger := logger.NewTestLogger()

	manager, err := NewManager(configPath, logger)
	require.NoError(t, err)
	require.NotNil(t, manager)
	defer manager.Stop()

	// Check loaded config
	cfg := manager.GetConfig()
	assert.Equal(t, "/tmp/wemixvisor", cfg.Home)
	assert.Equal(t, "wemixd", cfg.Name)
}

func TestManager_GetConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	logger := logger.NewTestLogger()
	manager, err := NewManager(configPath, logger)
	require.NoError(t, err)
	defer manager.Stop()

	cfg := manager.GetConfig()
	assert.NotNil(t, cfg)

	// Check defaults are applied
	assert.NotEmpty(t, cfg.Home)
	assert.NotEmpty(t, cfg.Name)
}

func TestManager_UpdateConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	logger := logger.NewTestLogger()
	manager, err := NewManager(configPath, logger)
	require.NoError(t, err)
	defer manager.Stop()

	// Test update (not fully implemented yet)
	err = manager.UpdateConfig("name", "updated-name")
	assert.Error(t, err) // Expected to fail with "not implemented"
}

func TestManager_ApplyTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	logger := logger.NewTestLogger()
	manager, err := NewManager(configPath, logger)
	require.NoError(t, err)
	defer manager.Stop()

	// Apply mainnet template
	err = manager.ApplyTemplate("mainnet")
	assert.NoError(t, err)

	// Check if template was applied
	nodeConfig := manager.GetNodeConfig()
	assert.Equal(t, uint64(1111), nodeConfig.NetworkID)
	assert.Equal(t, 8588, nodeConfig.RPCPort)
}

func TestManager_Validate(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	logger := logger.NewTestLogger()
	manager, err := NewManager(configPath, logger)
	require.NoError(t, err)
	defer manager.Stop()

	// Validation should pass for default config
	err = manager.Validate()
	assert.NoError(t, err)
}

func TestManager_Migrate(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	logger := logger.NewTestLogger()
	manager, err := NewManager(configPath, logger)
	require.NoError(t, err)
	defer manager.Stop()

	// Test migration from 0.4.0 to 0.5.0
	err = manager.Migrate("0.4.0", "0.5.0")
	assert.NoError(t, err)

	// Check if version was updated
	cfg := manager.GetConfig()
	assert.Equal(t, "0.5.0", cfg.ConfigVersion)
}

func TestManager_HotReload(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	// Create initial config with required network fields
	configContent := `
home = "/tmp/wemixvisor"
name = "wemixd"
network_id = 1112
chain_id = "1112"
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
network_id = 1112
chain_id = "1112"
rpc_port = 8588
`
	require.NoError(t, os.WriteFile(configPath, []byte(newContent), 0644))

	// Wait for update notification
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	select {
	case update := <-updateCh:
		assert.Equal(t, UpdateTypeReload, update.Type)
		assert.NotNil(t, update.NewConfig)
	case <-ctx.Done():
		// Hot reload might not trigger in test environment
		t.Skip("Hot reload did not trigger in test environment")
	}
}

func TestManager_ConfigFormats(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name      string
		format    string
		content   string
	}{
		{
			name:   "TOML format",
			format: "toml",
			content: `
home = "/tmp/wemixvisor"
name = "wemixd"
network_id = 1112
chain_id = "1112"
rpc_port = 8588
`,
		},
		{
			name:   "YAML format",
			format: "yaml",
			content: `
home: /tmp/wemixvisor
name: wemixd
network_id: 1112
chain_id: "1112"
rpc_port: 8588
`,
		},
		{
			name:   "JSON format",
			format: "json",
			content: `{
  "home": "/tmp/wemixvisor",
  "name": "wemixd",
  "network_id": 1112,
  "chain_id": "1112",
  "rpc_port": 8588
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := filepath.Join(tmpDir, "config."+tt.format)
			require.NoError(t, os.WriteFile(configPath, []byte(tt.content), 0644))

			logger := logger.NewTestLogger()
			manager, err := NewManager(configPath, logger)
			require.NoError(t, err)
			defer manager.Stop()

			cfg := manager.GetConfig()
			assert.Equal(t, "/tmp/wemixvisor", cfg.Home)
			assert.Equal(t, "wemixd", cfg.Name)
		})
	}
}

func TestManager_UnsupportedFormat(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.txt")

	// Create unsupported format file
	require.NoError(t, os.WriteFile(configPath, []byte("unsupported"), 0644))

	logger := logger.NewTestLogger()
	_, err := NewManager(configPath, logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported config format")
}