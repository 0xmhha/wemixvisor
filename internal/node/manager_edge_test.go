package node

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wemix/wemixvisor/internal/config"
	"github.com/wemix/wemixvisor/pkg/logger"
)

// TestManager_MultipleStopCalls tests calling Stop multiple times
func TestManager_MultipleStopCalls(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping functional test in short mode")
	}
	homeDir := t.TempDir()
	binDir := filepath.Join(homeDir, "wemixvisor", "current", "bin")
	require.NoError(t, os.MkdirAll(binDir, 0755))

	mockBin := filepath.Join(binDir, "wemixd")
	createMockBinary(t, mockBin)

	cfg := &config.Config{
		Home:          homeDir,
		Name:          "wemixd",
		ShutdownGrace: 1 * time.Second,
	}

	logger := logger.NewTestLogger()
	manager := NewManager(cfg, logger)

	// Start the node
	err := manager.Start([]string{"--test"})
	require.NoError(t, err)

	// First stop should succeed
	err = manager.Stop()
	assert.NoError(t, err)

	// Second stop should fail gracefully
	err = manager.Stop()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not running")

	// Third stop should also fail
	err = manager.Stop()
	assert.Error(t, err)
}

// TestManager_RestartFromVariousStates tests restart from different states
func TestManager_RestartFromVariousStates(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping functional test in short mode")
	}
	tests := []struct {
		name         string
		initialState NodeState
		expectError  bool
	}{
		{"from stopped", StateStopped, false},
		{"from running", StateRunning, false},
		{"from crashed", StateCrashed, false},
		{"from error", StateError, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			homeDir := t.TempDir()
			binDir := filepath.Join(homeDir, "wemixvisor", "current", "bin")
			require.NoError(t, os.MkdirAll(binDir, 0755))

			mockBin := filepath.Join(binDir, "wemixd")
			createMockBinary(t, mockBin)

			cfg := &config.Config{
				Home:          homeDir,
				Name:          "wemixd",
				ShutdownGrace: 1 * time.Second,
			}

			logger := logger.NewTestLogger()
			manager := NewManager(cfg, logger)

			// Set initial state
			if tt.initialState == StateRunning {
				err := manager.Start([]string{"--test"})
				require.NoError(t, err)
			} else {
				// Manually set state for testing
				manager.stateMutex.Lock()
				manager.state = tt.initialState
				manager.nodeArgs = []string{"--test"}
				manager.stateMutex.Unlock()
			}

			// Attempt restart
			err := manager.Restart()

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, StateRunning, manager.GetState())

				// Clean up
				manager.Stop()
			}
		})
	}
}

// TestManager_RapidStartStop tests rapid start/stop cycles
func TestManager_RapidStartStop(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping functional test in short mode")
	}
	homeDir := t.TempDir()
	binDir := filepath.Join(homeDir, "wemixvisor", "current", "bin")
	require.NoError(t, os.MkdirAll(binDir, 0755))

	mockBin := filepath.Join(binDir, "wemixd")
	createMockBinary(t, mockBin)

	cfg := &config.Config{
		Home:          homeDir,
		Name:          "wemixd",
		ShutdownGrace: 500 * time.Millisecond,
	}

	logger := logger.NewTestLogger()
	manager := NewManager(cfg, logger)

	// Perform rapid start/stop cycles
	for i := 0; i < 3; i++ {
		err := manager.Start([]string{"--test"})
		require.NoError(t, err)
		assert.Equal(t, StateRunning, manager.GetState())

		err = manager.Stop()
		require.NoError(t, err)
		assert.Equal(t, StateStopped, manager.GetState())

		// Very short delay between cycles
		time.Sleep(100 * time.Millisecond)
	}
}

// TestManager_EmptyArguments tests starting with empty arguments
func TestManager_EmptyArguments(t *testing.T) {
	homeDir := t.TempDir()
	binDir := filepath.Join(homeDir, "wemixvisor", "current", "bin")
	require.NoError(t, os.MkdirAll(binDir, 0755))

	mockBin := filepath.Join(binDir, "wemixd")
	createMockBinary(t, mockBin)

	cfg := &config.Config{
		Home:          homeDir,
		Name:          "wemixd",
		ShutdownGrace: 1 * time.Second,
	}

	logger := logger.NewTestLogger()
	manager := NewManager(cfg, logger)

	// Start with empty arguments
	err := manager.Start([]string{})
	require.NoError(t, err)
	assert.Equal(t, StateRunning, manager.GetState())

	// Clean up
	err = manager.Stop()
	require.NoError(t, err)
}

// TestManager_VeryLongArguments tests starting with many/long arguments
func TestManager_VeryLongArguments(t *testing.T) {
	homeDir := t.TempDir()
	binDir := filepath.Join(homeDir, "wemixvisor", "current", "bin")
	require.NoError(t, os.MkdirAll(binDir, 0755))

	mockBin := filepath.Join(binDir, "wemixd")
	createMockBinary(t, mockBin)

	cfg := &config.Config{
		Home:          homeDir,
		Name:          "wemixd",
		ShutdownGrace: 1 * time.Second,
	}

	logger := logger.NewTestLogger()
	manager := NewManager(cfg, logger)

	// Create many arguments (but not excessively long to avoid system limits)
	args := make([]string, 50)
	for i := 0; i < 50; i++ {
		args[i] = fmt.Sprintf("--arg%d=value%d", i, i)
	}

	// Start with long arguments
	err := manager.Start(args)
	require.NoError(t, err)
	assert.Equal(t, StateRunning, manager.GetState())

	// Verify arguments were saved
	assert.Equal(t, args, manager.nodeArgs)

	// Clean up
	err = manager.Stop()
	require.NoError(t, err)
}

// TestManager_RestartCounterBehavior tests restart counter behavior
func TestManager_RestartCounterBehavior(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping functional test in short mode")
	}
	homeDir := t.TempDir()
	binDir := filepath.Join(homeDir, "wemixvisor", "current", "bin")
	require.NoError(t, os.MkdirAll(binDir, 0755))

	mockBin := filepath.Join(binDir, "wemixd")
	createMockBinary(t, mockBin)

	cfg := &config.Config{
		Home:          homeDir,
		Name:          "wemixd",
		ShutdownGrace: 1 * time.Second,
	}

	logger := logger.NewTestLogger()
	manager := NewManager(cfg, logger)

	// Start the node
	err := manager.Start([]string{"--test"})
	require.NoError(t, err)

	// Initial count should be 0
	assert.Equal(t, 0, manager.restartCount)

	// Restart manually
	err = manager.Restart()
	require.NoError(t, err)

	// Counter should increment
	assert.Equal(t, 1, manager.restartCount)

	// Restart again
	err = manager.Restart()
	require.NoError(t, err)

	// Counter should increment again
	assert.Equal(t, 2, manager.restartCount)

	// Clean up
	manager.Stop()
}

// TestManager_GetStatus_EdgeCases tests GetStatus in various states
func TestManager_GetStatus_EdgeCases(t *testing.T) {
	cfg := &config.Config{
		Home:    t.TempDir(),
		Name:    "wemixd",
		Network: "testnet",
	}

	logger := logger.NewTestLogger()
	manager := NewManager(cfg, logger)

	// Test GetStatus in various states
	states := []NodeState{
		StateStopped,
		StateStarting,
		StateError,
		StateCrashed,
		StateUpgrading,
	}

	for _, state := range states {
		t.Run(state.String(), func(t *testing.T) {
			manager.stateMutex.Lock()
			manager.state = state
			manager.stateMutex.Unlock()

			status := manager.GetStatus()
			assert.Equal(t, state, status.State)
			assert.Equal(t, state.String(), status.StateString)
			assert.Equal(t, "testnet", status.Network)
			assert.Equal(t, 0, status.PID) // No process in these states
		})
	}
}

// TestManager_NilChannelHandling tests handling of nil channels
func TestManager_NilChannelHandling(t *testing.T) {
	cfg := &config.Config{
		Home: t.TempDir(),
		Name: "wemixd",
	}

	logger := logger.NewTestLogger()
	manager := NewManager(cfg, logger)

	// Set doneCh to nil
	manager.stateMutex.Lock()
	manager.doneCh = nil
	manager.stateMutex.Unlock()

	// Wait should handle nil channel gracefully
	waitCh := manager.Wait()
	assert.Nil(t, waitCh)

	// Multiple closes should not panic
	manager.stateMutex.Lock()
	if manager.doneCh != nil {
		close(manager.doneCh)
		manager.doneCh = nil
	}
	// Second close attempt
	if manager.doneCh != nil {
		close(manager.doneCh)
		manager.doneCh = nil
	}
	manager.stateMutex.Unlock()

	// Should not panic
	assert.True(t, true)
}

// TestManager_LongRunningShutdown tests shutdown with long grace period
func TestManager_LongRunningShutdown(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping long running test in short mode")
	}

	homeDir := t.TempDir()
	binDir := filepath.Join(homeDir, "wemixvisor", "current", "bin")
	require.NoError(t, os.MkdirAll(binDir, 0755))

	// Create a binary that ignores SIGTERM
	mockBin := filepath.Join(binDir, "wemixd")
	script := `#!/bin/sh
# Mock binary that ignores SIGTERM
trap '' TERM
sleep 10
`
	err := os.WriteFile(mockBin, []byte(script), 0755)
	require.NoError(t, err)

	cfg := &config.Config{
		Home:          homeDir,
		Name:          "wemixd",
		ShutdownGrace: 1 * time.Second, // Short grace for testing
	}

	logger := logger.NewTestLogger()
	manager := NewManager(cfg, logger)

	// Start the node
	err = manager.Start([]string{"--test"})
	require.NoError(t, err)

	// Time the stop operation
	start := time.Now()
	err = manager.Stop()
	elapsed := time.Since(start)

	// Should have waited for grace period then forced kill
	assert.NoError(t, err)
	assert.True(t, elapsed >= cfg.ShutdownGrace)
	assert.True(t, elapsed < cfg.ShutdownGrace+2*time.Second)
	assert.Equal(t, StateStopped, manager.GetState())
}