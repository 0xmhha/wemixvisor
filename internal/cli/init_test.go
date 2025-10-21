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

func TestInitCommand_Success(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()

	// Create a dummy binary for testing
	dummyBinary := filepath.Join(tmpDir, "wemixd")
	err := os.WriteFile(dummyBinary, []byte("#!/bin/sh\necho test"), 0755)
	require.NoError(t, err)

	cfg := &config.Config{
		Home: tmpDir,
		Name: "wemixd",
	}
	log, err := logger.New(false, false, "")
	require.NoError(t, err)

	cmd := NewInitCommand(cfg, log)

	// Act
	cmd.SetArgs([]string{dummyBinary})
	err = cmd.Execute()

	// Assert
	assert.NoError(t, err)

	// Verify directory structure
	expectedDirs := []string{
		filepath.Join(tmpDir, "wemixvisor"),
		filepath.Join(tmpDir, "wemixvisor", "genesis"),
		filepath.Join(tmpDir, "wemixvisor", "genesis", "bin"),
		filepath.Join(tmpDir, "wemixvisor", "upgrades"),
		filepath.Join(tmpDir, "data"),
	}

	for _, dir := range expectedDirs {
		info, err := os.Stat(dir)
		assert.NoError(t, err, "directory should exist: %s", dir)
		assert.True(t, info.IsDir(), "should be a directory: %s", dir)
	}

	// Verify genesis binary was copied
	genesisBin := cfg.GenesisBin()
	info, err := os.Stat(genesisBin)
	assert.NoError(t, err, "genesis binary should exist")
	assert.False(t, info.IsDir())
	assert.Equal(t, os.FileMode(0755), info.Mode().Perm())

	// Verify symbolic link
	currentLink := cfg.CurrentDir()
	target, err := os.Readlink(currentLink)
	assert.NoError(t, err, "current link should exist")
	assert.Equal(t, "genesis", filepath.Base(target))
}

func TestInitCommand_NonExistentBinary(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()

	cfg := &config.Config{
		Home: tmpDir,
		Name: "wemixd",
	}
	log, err := logger.New(false, false, "")
	require.NoError(t, err)

	cmd := NewInitCommand(cfg, log)

	// Act
	cmd.SetArgs([]string{"/nonexistent/binary"})
	err = cmd.Execute()

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to copy binary")
}

func TestInitCommand_NoArguments(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()

	cfg := &config.Config{
		Home: tmpDir,
		Name: "wemixd",
	}
	log, err := logger.New(false, false, "")
	require.NoError(t, err)

	cmd := NewInitCommand(cfg, log)

	// Act
	cmd.SetArgs([]string{})
	err = cmd.Execute()

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 1 arg(s), received 0")
}

func TestInitCommand_WithTemplate(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()

	// Create a dummy binary
	dummyBinary := filepath.Join(tmpDir, "wemixd")
	err := os.WriteFile(dummyBinary, []byte("#!/bin/sh\necho test"), 0755)
	require.NoError(t, err)

	cfg := &config.Config{
		Home: tmpDir,
		Name: "wemixd",
	}
	log, err := logger.New(false, false, "")
	require.NoError(t, err)

	cmd := NewInitCommand(cfg, log)

	// Act
	cmd.SetArgs([]string{dummyBinary, "--template", "testnet"})
	err = cmd.Execute()

	// Assert
	assert.NoError(t, err)

	// Verify directories created
	assert.DirExists(t, filepath.Join(tmpDir, "wemixvisor"))
}

func TestInitCommand_DirectoryPermissions(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()

	// Create a dummy binary
	dummyBinary := filepath.Join(tmpDir, "wemixd")
	err := os.WriteFile(dummyBinary, []byte("#!/bin/sh\necho test"), 0755)
	require.NoError(t, err)

	cfg := &config.Config{
		Home: tmpDir,
		Name: "wemixd",
	}
	log, err := logger.New(false, false, "")
	require.NoError(t, err)

	cmd := NewInitCommand(cfg, log)

	// Act
	cmd.SetArgs([]string{dummyBinary})
	err = cmd.Execute()
	require.NoError(t, err)

	// Assert - Check directory permissions
	wemixvisorDir := filepath.Join(tmpDir, "wemixvisor")
	info, err := os.Stat(wemixvisorDir)
	assert.NoError(t, err)

	// Should have proper permissions (755)
	mode := info.Mode()
	assert.Equal(t, os.FileMode(0755), mode.Perm())
}

func TestInitCommand_AlreadyInitialized(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()

	// Create a dummy binary
	dummyBinary := filepath.Join(tmpDir, "wemixd")
	err := os.WriteFile(dummyBinary, []byte("#!/bin/sh\necho test"), 0755)
	require.NoError(t, err)

	cfg := &config.Config{
		Home: tmpDir,
		Name: "wemixd",
	}
	log, err := logger.New(false, false, "")
	require.NoError(t, err)

	cmd := NewInitCommand(cfg, log)

	// Act - First initialization
	cmd.SetArgs([]string{dummyBinary})
	err = cmd.Execute()
	require.NoError(t, err)

	// Act - Second initialization (should succeed, overwriting)
	cmd2 := NewInitCommand(cfg, log)
	cmd2.SetArgs([]string{dummyBinary})
	err = cmd2.Execute()

	// Assert - Should succeed (idempotent)
	assert.NoError(t, err)
}
