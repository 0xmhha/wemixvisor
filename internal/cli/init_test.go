package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wemix/wemixvisor/internal/config"
	"github.com/wemix/wemixvisor/pkg/logger"
)

func TestInitCommand_Execute(t *testing.T) {
	// Create temporary directory for testing
	tmpDir := t.TempDir()

	// Create config and logger
	cfg := &config.Config{
		Home: tmpDir,
	}
	logger, err := logger.New(true, false, "")
	require.NoError(t, err)

	// Create init command
	initCmd := NewInitCommand(cfg, logger)

	// Execute init command
	err = initCmd.Execute([]string{})
	assert.NoError(t, err)

	// Verify directory structure
	expectedDirs := []string{
		tmpDir,
		filepath.Join(tmpDir, "wemixvisor"),
		filepath.Join(tmpDir, "wemixvisor", "genesis"),
		filepath.Join(tmpDir, "wemixvisor", "genesis", "bin"),
		filepath.Join(tmpDir, "wemixvisor", "upgrades"),
		filepath.Join(tmpDir, "wemixvisor", "backup"),
		filepath.Join(tmpDir, "data"),
		filepath.Join(tmpDir, "logs"),
	}

	for _, dir := range expectedDirs {
		info, err := os.Stat(dir)
		assert.NoError(t, err, "directory should exist: %s", dir)
		assert.True(t, info.IsDir(), "should be a directory: %s", dir)
	}

	// Verify config file exists
	configPath := filepath.Join(tmpDir, "wemixvisor", "config.toml")
	_, err = os.Stat(configPath)
	assert.NoError(t, err, "config.toml should exist")

	// Verify config file content
	content, err := os.ReadFile(configPath)
	assert.NoError(t, err)
	assert.Contains(t, string(content), "network = \"mainnet\"")
	assert.Contains(t, string(content), "restart_on_failure = true")
	assert.Contains(t, string(content), "health_check_interval = \"30s\"")

	// Verify symbolic link
	currentLink := filepath.Join(tmpDir, "wemixvisor", "current")
	target, err := os.Readlink(currentLink)
	assert.NoError(t, err)
	assert.Equal(t, "genesis", target)

	// Verify link points to existing directory
	linkTarget := filepath.Join(tmpDir, "wemixvisor", target)
	info, err := os.Stat(linkTarget)
	assert.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestInitCommand_Execute_AlreadyInitialized(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()

	// Pre-create wemixvisor directory
	wemixvisorDir := filepath.Join(tmpDir, "wemixvisor")
	err := os.MkdirAll(wemixvisorDir, 0755)
	require.NoError(t, err)

	// Create config and logger
	cfg := &config.Config{
		Home: tmpDir,
	}
	logger, err := logger.New(true, false, "")
	require.NoError(t, err)

	// Create init command
	initCmd := NewInitCommand(cfg, logger)

	// Execute init command (should not fail, just log that it's already initialized)
	err = initCmd.Execute([]string{})
	assert.NoError(t, err)
}

func TestInitCommand_Execute_InvalidHome(t *testing.T) {
	// Create config with file as home (should fail)
	tmpFile, err := os.CreateTemp("", "test-file")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	cfg := &config.Config{
		Home: tmpFile.Name(), // File instead of directory
	}
	logger, err := logger.New(true, false, "")
	require.NoError(t, err)

	// Create init command
	initCmd := NewInitCommand(cfg, logger)

	// Execute init command (should fail)
	err = initCmd.Execute([]string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exists but is not a directory")
}

func TestInitCommand_Execute_NoHome(t *testing.T) {
	// Save original environment
	originalHome := os.Getenv("DAEMON_HOME")
	defer os.Setenv("DAEMON_HOME", originalHome)

	// Clear DAEMON_HOME
	os.Unsetenv("DAEMON_HOME")

	// Create config without home
	cfg := &config.Config{}
	logger, err := logger.New(true, false, "")
	require.NoError(t, err)

	// Create init command
	initCmd := NewInitCommand(cfg, logger)

	// Execute init command (should fail)
	err = initCmd.Execute([]string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "DAEMON_HOME not set")
}

func TestInitCommand_Execute_WithEnvHome(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()

	// Save original environment
	originalHome := os.Getenv("DAEMON_HOME")
	defer os.Setenv("DAEMON_HOME", originalHome)

	// Set DAEMON_HOME environment variable
	os.Setenv("DAEMON_HOME", tmpDir)

	// Create config without home (should use env var)
	cfg := &config.Config{}
	logger, err := logger.New(true, false, "")
	require.NoError(t, err)

	// Create init command
	initCmd := NewInitCommand(cfg, logger)

	// Execute init command
	err = initCmd.Execute([]string{})
	assert.NoError(t, err)

	// Verify directory was created using env var path
	wemixvisorDir := filepath.Join(tmpDir, "wemixvisor")
	info, err := os.Stat(wemixvisorDir)
	assert.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestInitCommand_CreateInitialConfig(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.toml")

	// Create config and logger
	cfg := &config.Config{}
	logger, err := logger.New(true, false, "")
	require.NoError(t, err)

	// Create init command
	initCmd := NewInitCommand(cfg, logger)

	// Create initial config
	err = initCmd.createInitialConfig(configPath)
	assert.NoError(t, err)

	// Verify file exists
	_, err = os.Stat(configPath)
	assert.NoError(t, err)

	// Verify content
	content, err := os.ReadFile(configPath)
	assert.NoError(t, err)

	configStr := string(content)
	assert.Contains(t, configStr, "# Wemixvisor Configuration")
	assert.Contains(t, configStr, "network = \"mainnet\"")
	assert.Contains(t, configStr, "restart_on_failure = true")
	assert.Contains(t, configStr, "max_restarts = 5")
	assert.Contains(t, configStr, "shutdown_grace_period = \"30s\"")
	assert.Contains(t, configStr, "health_check_interval = \"30s\"")
	assert.Contains(t, configStr, "rpc_port = 8545")
	assert.Contains(t, configStr, "backup_enabled = true")
	assert.Contains(t, configStr, "backup_count = 3")
	assert.Contains(t, configStr, "log_level = \"info\"")
	assert.Contains(t, configStr, "[environment]")
}

func TestInitCommand_DirectoryPermissions(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()

	// Create config and logger
	cfg := &config.Config{
		Home: tmpDir,
	}
	logger, err := logger.New(true, false, "")
	require.NoError(t, err)

	// Create init command
	initCmd := NewInitCommand(cfg, logger)

	// Execute init command
	err = initCmd.Execute([]string{})
	assert.NoError(t, err)

	// Check directory permissions
	wemixvisorDir := filepath.Join(tmpDir, "wemixvisor")
	info, err := os.Stat(wemixvisorDir)
	assert.NoError(t, err)

	// Should have proper permissions (755)
	mode := info.Mode()
	assert.Equal(t, os.FileMode(0755), mode.Perm())
}