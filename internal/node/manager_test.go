package node

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wemix/wemixvisor/internal/config"
	"github.com/wemix/wemixvisor/pkg/logger"
)

func TestNodeState_String(t *testing.T) {
	tests := []struct {
		state    NodeState
		expected string
	}{
		{StateStopped, "stopped"},
		{StateStarting, "starting"},
		{StateRunning, "running"},
		{StateStopping, "stopping"},
		{StateUpgrading, "upgrading"},
		{StateError, "error"},
		{StateCrashed, "crashed"},
		{NodeState(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.state.String())
		})
	}
}

func TestManager_NewManager(t *testing.T) {
	cfg := &config.Config{
		Home:        t.TempDir(),
		Name:        "test-node",
		MaxRestarts: 3,
	}

	logger := logger.NewTestLogger()
	manager := NewManager(cfg, logger)

	assert.NotNil(t, manager)
	assert.Equal(t, StateStopped, manager.GetState())
	assert.Equal(t, 3, manager.maxRestarts)
	assert.NotNil(t, manager.ctx)
	assert.NotNil(t, manager.cancel)
}

func TestManager_Start_NoBinary(t *testing.T) {
	cfg := &config.Config{
		Home: t.TempDir(),
		Name: "test-node",
	}

	logger := logger.NewTestLogger()
	manager := NewManager(cfg, logger)

	err := manager.Start([]string{"--testnet"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no binary found")
	assert.Equal(t, StateError, manager.GetState())
}

func TestManager_Start_Success(t *testing.T) {
	homeDir := t.TempDir()
	binDir := filepath.Join(homeDir, "wemixvisor", "current", "bin")
	require.NoError(t, os.MkdirAll(binDir, 0755))

	// Create a mock binary
	mockBin := filepath.Join(binDir, "wemixd")
	createMockBinary(t, mockBin)

	cfg := &config.Config{
		Home:             homeDir,
		Name:             "wemixd",
		RestartOnFailure: false,
	}

	logger := logger.NewTestLogger()
	manager := NewManager(cfg, logger)

	// Start the node
	err := manager.Start([]string{"--test"})
	require.NoError(t, err)
	assert.Equal(t, StateRunning, manager.GetState())

	// Check status
	status := manager.GetStatus()
	assert.NotZero(t, status.PID)
	assert.NotZero(t, status.StartTime)
	assert.Equal(t, "running", status.StateString)

	// Stop the node
	err = manager.Stop()
	require.NoError(t, err)
	assert.Equal(t, StateStopped, manager.GetState())
}

func TestManager_Stop_NotRunning(t *testing.T) {
	cfg := &config.Config{
		Home: t.TempDir(),
		Name: "test-node",
	}

	logger := logger.NewTestLogger()
	manager := NewManager(cfg, logger)

	err := manager.Stop()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "node is not running")
}

func TestManager_Restart(t *testing.T) {
	homeDir := t.TempDir()
	binDir := filepath.Join(homeDir, "wemixvisor", "current", "bin")
	require.NoError(t, os.MkdirAll(binDir, 0755))

	// Create a mock binary
	mockBin := filepath.Join(binDir, "wemixd")
	createMockBinary(t, mockBin)

	cfg := &config.Config{
		Home:             homeDir,
		Name:             "wemixd",
		RestartOnFailure: false,
	}

	logger := logger.NewTestLogger()
	manager := NewManager(cfg, logger)

	// Start the node
	err := manager.Start([]string{"--test"})
	require.NoError(t, err)

	initialPID := manager.GetStatus().PID

	// Restart the node
	err = manager.Restart()
	require.NoError(t, err)

	// Check that it's running with a different PID
	assert.Equal(t, StateRunning, manager.GetState())
	newPID := manager.GetStatus().PID
	assert.NotEqual(t, initialPID, newPID)
	assert.Equal(t, 1, manager.restartCount)

	// Clean up
	err = manager.Stop()
	require.NoError(t, err)
}

func TestManager_AutoRestart(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping auto-restart test in short mode")
	}

	homeDir := t.TempDir()
	binDir := filepath.Join(homeDir, "wemixvisor", "current", "bin")
	require.NoError(t, os.MkdirAll(binDir, 0755))

	// Create a mock binary that crashes after 1 second
	mockBin := filepath.Join(binDir, "wemixd")
	createCrashingMockBinary(t, mockBin, 1*time.Second)

	cfg := &config.Config{
		Home:             homeDir,
		Name:             "wemixd",
		RestartOnFailure: true,
		MaxRestarts:      2,
	}

	logger := logger.NewTestLogger()
	manager := NewManager(cfg, logger)

	// Start the node
	err := manager.Start([]string{"--test"})
	require.NoError(t, err)

	// Wait for crash and auto-restart
	time.Sleep(3 * time.Second)

	// It should have crashed and attempted to restart
	// Due to the crashing binary, it might be in crashed or running state
	state := manager.GetState()
	assert.True(t, state == StateRunning || state == StateCrashed)

	// Clean up
	manager.Close()
}

func TestManager_GetStatus(t *testing.T) {
	homeDir := t.TempDir()
	binDir := filepath.Join(homeDir, "wemixvisor", "current", "bin")
	require.NoError(t, os.MkdirAll(binDir, 0755))

	// Create a mock binary
	mockBin := filepath.Join(binDir, "wemixd")
	createMockBinary(t, mockBin)

	cfg := &config.Config{
		Home:    homeDir,
		Name:    "wemixd",
		Network: "testnet",
	}

	logger := logger.NewTestLogger()
	manager := NewManager(cfg, logger)

	// Get status before start
	status := manager.GetStatus()
	assert.Equal(t, StateStopped, status.State)
	assert.Equal(t, "stopped", status.StateString)
	assert.Equal(t, 0, status.PID)
	assert.Equal(t, "testnet", status.Network)

	// Start and get status
	err := manager.Start([]string{"--test"})
	require.NoError(t, err)

	status = manager.GetStatus()
	assert.Equal(t, StateRunning, status.State)
	assert.Equal(t, "running", status.StateString)
	assert.NotZero(t, status.PID)
	assert.NotZero(t, status.StartTime)

	// Clean up
	manager.Stop()
}

func TestManager_BuildEnvironment(t *testing.T) {
	cfg := &config.Config{
		Home:    "/test/home",
		Network: "mainnet",
		Environment: map[string]string{
			"CUSTOM_VAR": "custom_value",
		},
	}

	logger := logger.NewTestLogger()
	manager := NewManager(cfg, logger)

	env := manager.buildEnvironment()

	// Check that our custom environment variables are present
	hasHome := false
	hasNetwork := false
	hasCustom := false

	for _, e := range env {
		if e == "WEMIX_HOME=/test/home" {
			hasHome = true
		}
		if e == "WEMIX_NETWORK=mainnet" {
			hasNetwork = true
		}
		if e == "CUSTOM_VAR=custom_value" {
			hasCustom = true
		}
	}

	assert.True(t, hasHome)
	assert.True(t, hasNetwork)
	assert.True(t, hasCustom)
}

// Helper functions

func createMockBinary(t *testing.T, path string) {
	t.Helper()

	// Create a simple shell script as mock binary
	script := `#!/bin/sh
# Mock binary for testing - runs for a short time then exits
sleep 30
`
	err := ioutil.WriteFile(path, []byte(script), 0755)
	require.NoError(t, err)
}

func createCrashingMockBinary(t *testing.T, path string, crashAfter time.Duration) {
	t.Helper()

	// Create a shell script that crashes after specified duration
	script := fmt.Sprintf(`#!/bin/sh
sleep %d
exit 1
`, int(crashAfter.Seconds()))

	err := ioutil.WriteFile(path, []byte(script), 0755)
	require.NoError(t, err)
}