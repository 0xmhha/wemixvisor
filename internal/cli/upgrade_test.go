package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wemix/wemixvisor/internal/config"
	"github.com/wemix/wemixvisor/pkg/logger"
	"github.com/wemix/wemixvisor/pkg/types"
)

func TestScheduleCommand_Success(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Home: tmpDir,
		Name: "wemixd",
	}

	// Ensure wemixvisor directory exists
	wemixvisorDir := filepath.Join(tmpDir, "wemixvisor")
	require.NoError(t, os.MkdirAll(wemixvisorDir, 0755))

	log, err := logger.New(false, false, "")
	require.NoError(t, err)

	cmd := newScheduleCommand(cfg, log)

	// Act
	cmd.SetArgs([]string{"v1.2.0", "1000000"})
	err = cmd.Execute()

	// Assert
	assert.NoError(t, err, "schedule command should succeed")

	// Verify upgrade-info.json was created
	upgradeInfoPath := cfg.UpgradeInfoFilePath()
	require.FileExists(t, upgradeInfoPath, "upgrade-info.json should be created")

	// Verify content
	upgradeInfo, err := types.ParseUpgradeInfoFile(upgradeInfoPath)
	require.NoError(t, err)
	assert.Equal(t, "v1.2.0", upgradeInfo.Name)
	assert.Equal(t, int64(1000000), upgradeInfo.Height)
}

func TestScheduleCommand_WithMetadata(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Home: tmpDir,
		Name: "wemixd",
	}

	wemixvisorDir := filepath.Join(tmpDir, "wemixvisor")
	require.NoError(t, os.MkdirAll(wemixvisorDir, 0755))

	log, err := logger.New(false, false, "")
	require.NoError(t, err)

	cmd := newScheduleCommand(cfg, log)

	// Act
	cmd.SetArgs([]string{
		"v1.2.0", "1000000",
		"--checksum", "abc123",
		"--info", "Test upgrade",
	})
	err = cmd.Execute()

	// Assert
	assert.NoError(t, err)

	upgradeInfo, err := types.ParseUpgradeInfoFile(cfg.UpgradeInfoFilePath())
	require.NoError(t, err)
	assert.Equal(t, "abc123", upgradeInfo.Info["checksum"])
	assert.Equal(t, "Test upgrade", upgradeInfo.Info["description"])
}

func TestScheduleCommand_InvalidHeight(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Home: tmpDir,
		Name: "wemixd",
	}

	log, err := logger.New(false, false, "")
	require.NoError(t, err)

	cmd := newScheduleCommand(cfg, log)

	// Act
	cmd.SetArgs([]string{"v1.2.0", "invalid"})
	err = cmd.Execute()

	// Assert
	assert.Error(t, err, "should fail with invalid height")
	assert.Contains(t, err.Error(), "invalid height")
}

func TestScheduleCommand_NegativeHeight(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Home: tmpDir,
		Name: "wemixd",
	}

	log, err := logger.New(false, false, "")
	require.NoError(t, err)

	cmd := newScheduleCommand(cfg, log)

	// Act
	// Use "--" to prevent -100 from being parsed as a flag
	cmd.SetArgs([]string{"--", "v1.2.0", "-100"})
	err = cmd.Execute()

	// Assert
	assert.Error(t, err, "should fail with negative height")
	assert.Contains(t, err.Error(), "must be positive")
}

func TestUpgradeStatusCommand_NoUpgrade(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Home: tmpDir,
		Name: "wemixd",
	}

	wemixvisorDir := filepath.Join(tmpDir, "wemixvisor")
	require.NoError(t, os.MkdirAll(wemixvisorDir, 0755))

	log, err := logger.New(false, false, "")
	require.NoError(t, err)

	cmd := newUpgradeStatusCommand(cfg, log)

	// Act
	cmd.SetArgs([]string{})
	err = cmd.Execute()

	// Assert
	assert.NoError(t, err, "status command should succeed even with no upgrade")
}

func TestUpgradeStatusCommand_WithScheduledUpgrade(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Home: tmpDir,
		Name: "wemixd",
	}

	wemixvisorDir := filepath.Join(tmpDir, "wemixvisor")
	require.NoError(t, os.MkdirAll(wemixvisorDir, 0755))

	log, err := logger.New(false, false, "")
	require.NoError(t, err)

	// Create upgrade info
	upgradeInfo := &types.UpgradeInfo{
		Name:   "v1.2.0",
		Height: 1000000,
	}
	// Ensure data directory exists
	upgradeInfoPath := cfg.UpgradeInfoFilePath()
	require.NoError(t, os.MkdirAll(filepath.Dir(upgradeInfoPath), 0755))
	err = types.WriteUpgradeInfoFile(upgradeInfoPath, upgradeInfo)
	require.NoError(t, err)

	cmd := newUpgradeStatusCommand(cfg, log)

	// Act
	cmd.SetArgs([]string{})
	err = cmd.Execute()

	// Assert
	assert.NoError(t, err)
}

func TestCancelCommand_Success(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Home:  tmpDir,
		Name:  "wemixd",
		Quiet: true, // Skip confirmation
	}

	wemixvisorDir := filepath.Join(tmpDir, "wemixvisor")
	require.NoError(t, os.MkdirAll(wemixvisorDir, 0755))

	log, err := logger.New(false, false, "")
	require.NoError(t, err)

	// Create upgrade info
	upgradeInfo := &types.UpgradeInfo{
		Name:   "v1.2.0",
		Height: 1000000,
	}
	// Ensure data directory exists
	upgradeInfoPath := cfg.UpgradeInfoFilePath()
	require.NoError(t, os.MkdirAll(filepath.Dir(upgradeInfoPath), 0755))
	err = types.WriteUpgradeInfoFile(upgradeInfoPath, upgradeInfo)
	require.NoError(t, err)

	cmd := newCancelCommand(cfg, log)

	// Act
	cmd.SetArgs([]string{"--force"})
	err = cmd.Execute()

	// Assert
	assert.NoError(t, err)

	// Verify file was removed
	_, err = os.Stat(cfg.UpgradeInfoFilePath())
	assert.True(t, os.IsNotExist(err), "upgrade-info.json should be removed")
}

func TestCancelCommand_NoUpgrade(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Home: tmpDir,
		Name: "wemixd",
	}

	wemixvisorDir := filepath.Join(tmpDir, "wemixvisor")
	require.NoError(t, os.MkdirAll(wemixvisorDir, 0755))

	log, err := logger.New(false, false, "")
	require.NoError(t, err)

	cmd := newCancelCommand(cfg, log)

	// Act
	cmd.SetArgs([]string{"--force"})
	err = cmd.Execute()

	// Assert
	assert.NoError(t, err, "cancel should succeed even with no upgrade")
}

func TestUpgradeCommand_Integration(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Home:  tmpDir,
		Name:  "wemixd",
		Quiet: true,
	}

	wemixvisorDir := filepath.Join(tmpDir, "wemixvisor")
	require.NoError(t, os.MkdirAll(wemixvisorDir, 0755))

	log, err := logger.New(false, false, "")
	require.NoError(t, err)

	// Test full workflow: schedule -> status -> cancel

	// Step 1: Schedule
	scheduleCmd := newScheduleCommand(cfg, log)
	scheduleCmd.SetArgs([]string{"v1.2.0", "1000000"})
	err = scheduleCmd.Execute()
	require.NoError(t, err, "schedule should succeed")

	// Step 2: Status
	statusCmd := newUpgradeStatusCommand(cfg, log)
	statusCmd.SetArgs([]string{})
	err = statusCmd.Execute()
	require.NoError(t, err, "status should succeed")

	// Verify upgrade exists
	upgradeInfo, err := types.ParseUpgradeInfoFile(cfg.UpgradeInfoFilePath())
	require.NoError(t, err)
	assert.Equal(t, "v1.2.0", upgradeInfo.Name)

	// Step 3: Cancel
	cancelCmd := newCancelCommand(cfg, log)
	cancelCmd.SetArgs([]string{"--force"})
	err = cancelCmd.Execute()
	require.NoError(t, err, "cancel should succeed")

	// Verify file was removed
	_, err = os.Stat(cfg.UpgradeInfoFilePath())
	assert.True(t, os.IsNotExist(err), "upgrade-info.json should be removed")

	// Step 4: Status after cancel
	statusCmd2 := newUpgradeStatusCommand(cfg, log)
	statusCmd2.SetArgs([]string{})
	err = statusCmd2.Execute()
	require.NoError(t, err, "status should succeed after cancel")
}
