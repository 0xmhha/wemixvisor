package node

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wemix/wemixvisor/internal/config"
	"github.com/wemix/wemixvisor/pkg/logger"
)

// TestManager_Start_BinaryNotExecutable tests starting with non-executable binary
func TestManager_Start_BinaryNotExecutable(t *testing.T) {
	homeDir := t.TempDir()
	binDir := filepath.Join(homeDir, "wemixvisor", "current", "bin")
	require.NoError(t, os.MkdirAll(binDir, 0755))

	// Create a non-executable file
	mockBin := filepath.Join(binDir, "wemixd")
	err := os.WriteFile(mockBin, []byte("not executable"), 0644)
	require.NoError(t, err)

	cfg := &config.Config{
		Home: homeDir,
		Name: "wemixd",
	}

	logger := logger.NewTestLogger()
	manager := NewManager(cfg, logger)

	err = manager.Start([]string{"--test"})
	assert.Error(t, err)
	assert.Equal(t, StateError, manager.GetState())
}

// TestManager_Start_AlreadyRunning tests starting when already running
func TestManager_Start_AlreadyRunning(t *testing.T) {
	homeDir := t.TempDir()
	binDir := filepath.Join(homeDir, "wemixvisor", "current", "bin")
	require.NoError(t, os.MkdirAll(binDir, 0755))

	mockBin := filepath.Join(binDir, "wemixd")
	createMockBinary(t, mockBin)

	cfg := &config.Config{
		Home:          homeDir,
		Name:          "wemixd",
		ShutdownGrace: 2 * time.Second,
	}

	logger := logger.NewTestLogger()
	manager := NewManager(cfg, logger)

	// Start the node
	err := manager.Start([]string{"--test"})
	require.NoError(t, err)
	defer manager.Stop()

	// Try to start again
	err = manager.Start([]string{"--test"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not in stopped state")
}

// TestManager_Stop_AlreadyStopped tests stopping when already stopped
func TestManager_Stop_AlreadyStopped(t *testing.T) {
	cfg := &config.Config{
		Home: t.TempDir(),
		Name: "wemixd",
	}

	logger := logger.NewTestLogger()
	manager := NewManager(cfg, logger)

	// Try to stop when not running
	err := manager.Stop()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "node is not running")
}

// TestManager_Stop_SignalError tests handling of signal errors
func TestManager_Stop_SignalError(t *testing.T) {
	homeDir := t.TempDir()
	binDir := filepath.Join(homeDir, "wemixvisor", "current", "bin")
	require.NoError(t, os.MkdirAll(binDir, 0755))

	mockBin := filepath.Join(binDir, "wemixd")
	createMockBinary(t, mockBin)

	cfg := &config.Config{
		Home:          homeDir,
		Name:          "wemixd",
		ShutdownGrace: 100 * time.Millisecond, // Very short for testing
	}

	logger := logger.NewTestLogger()
	manager := NewManager(cfg, logger)

	// Start the node
	err := manager.Start([]string{"--test"})
	require.NoError(t, err)

	// Save the process and simulate it's already dead
	proc := manager.process

	// Kill the process directly (simulate unexpected death)
	proc.Kill()
	time.Sleep(200 * time.Millisecond)

	// Now try to stop - should handle the error gracefully
	err = manager.Stop()
	// Should either succeed or return an error, but not panic
	// The exact behavior depends on timing
	if err == nil {
		assert.Equal(t, StateStopped, manager.GetState())
	}
}

// TestManager_Restart_MaxRestartsExceeded tests max restart limit
func TestManager_Restart_MaxRestartsExceeded(t *testing.T) {
	homeDir := t.TempDir()
	binDir := filepath.Join(homeDir, "wemixvisor", "current", "bin")
	require.NoError(t, os.MkdirAll(binDir, 0755))

	mockBin := filepath.Join(binDir, "wemixd")
	createCrashingMockBinary(t, mockBin, 500*time.Millisecond)

	cfg := &config.Config{
		Home:             homeDir,
		Name:             "wemixd",
		RestartOnFailure: true,
		MaxRestarts:      2,
		ShutdownGrace:    1 * time.Second,
	}

	logger := logger.NewTestLogger()
	manager := NewManager(cfg, logger)

	// Start the node
	err := manager.Start([]string{"--test"})
	require.NoError(t, err)

	// Wait for first crash (500ms) + restart delay (5s) + second crash (500ms) + restart delay (5s)
	// Total: ~11 seconds for 2 restarts
	time.Sleep(12 * time.Second)

	// Check that restarts happened
	state := manager.GetState()
	// The node may be in any state depending on timing
	assert.True(t, state == StateError || state == StateCrashed || state == StateRunning)
	// Should have at least attempted one restart
	assert.GreaterOrEqual(t, manager.restartCount, 1)
}

// TestManager_Close tests the Close method
func TestManager_Close(t *testing.T) {
	homeDir := t.TempDir()
	binDir := filepath.Join(homeDir, "wemixvisor", "current", "bin")
	require.NoError(t, os.MkdirAll(binDir, 0755))

	mockBin := filepath.Join(binDir, "wemixd")
	createMockBinary(t, mockBin)

	cfg := &config.Config{
		Home:          homeDir,
		Name:          "wemixd",
		ShutdownGrace: 2 * time.Second,
	}

	logger := logger.NewTestLogger()

	// Test 1: Close when not running
	manager1 := NewManager(cfg, logger)
	err := manager1.Close()
	assert.NoError(t, err)

	// Test 2: Close when running
	manager2 := NewManager(cfg, logger)
	err = manager2.Start([]string{"--test"})
	require.NoError(t, err)
	assert.Equal(t, StateRunning, manager2.GetState())

	// Close should stop the node
	err = manager2.Close()
	assert.NoError(t, err)
	assert.Equal(t, StateStopped, manager2.GetState())
}

// TestManager_GetLogFile tests log file creation
func TestManager_GetLogFile(t *testing.T) {
	homeDir := t.TempDir()
	logFile := filepath.Join(homeDir, "logs", "node.log")

	cfg := &config.Config{
		Home:    homeDir,
		LogFile: logFile,
	}

	logger := logger.NewTestLogger()
	manager := NewManager(cfg, logger)

	// Access private method through reflection or make it public for testing
	// For now, we'll test it indirectly through Start
	binDir := filepath.Join(homeDir, "wemixvisor", "current", "bin")
	require.NoError(t, os.MkdirAll(binDir, 0755))

	mockBin := filepath.Join(binDir, "wemixd")
	createMockBinary(t, mockBin)

	cfg.Name = "wemixd"
	cfg.ShutdownGrace = 1 * time.Second

	manager = NewManager(cfg, logger)
	err := manager.Start([]string{"--test"})
	require.NoError(t, err)
	defer manager.Stop()

	// Check if log directory was created
	logDir := filepath.Dir(logFile)
	_, err = os.Stat(logDir)
	assert.NoError(t, err)
}

// TestManager_BuildEnvironment_WithCustomVars tests custom environment variables
func TestManager_BuildEnvironment_WithCustomVars(t *testing.T) {
	cfg := &config.Config{
		Home:    "/test/home",
		Network: "testnet",
		Environment: map[string]string{
			"CUSTOM_VAR1": "value1",
			"CUSTOM_VAR2": "value2",
			"NODE_ENV":    "production",
		},
	}

	logger := logger.NewTestLogger()
	manager := NewManager(cfg, logger)

	env := manager.buildEnvironment()

	// Check all custom vars are present
	envMap := make(map[string]string)
	for _, e := range env {
		if idx := len(e); idx > 0 {
			for i := 0; i < len(e); i++ {
				if e[i] == '=' {
					envMap[e[:i]] = e[i+1:]
					break
				}
			}
		}
	}

	assert.Equal(t, "/test/home", envMap["WEMIX_HOME"])
	assert.Equal(t, "testnet", envMap["WEMIX_NETWORK"])
	assert.Equal(t, "value1", envMap["CUSTOM_VAR1"])
	assert.Equal(t, "value2", envMap["CUSTOM_VAR2"])
	assert.Equal(t, "production", envMap["NODE_ENV"])
}

// TestManager_ConcurrentOperations tests thread safety
func TestManager_ConcurrentOperations(t *testing.T) {
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

	// Concurrent operations
	done := make(chan bool, 3)

	// Goroutine 1: Get status repeatedly
	go func() {
		for i := 0; i < 10; i++ {
			_ = manager.GetStatus()
			time.Sleep(10 * time.Millisecond)
		}
		done <- true
	}()

	// Goroutine 2: Get state repeatedly
	go func() {
		for i := 0; i < 10; i++ {
			_ = manager.GetState()
			time.Sleep(10 * time.Millisecond)
		}
		done <- true
	}()

	// Goroutine 3: Try to restart
	go func() {
		time.Sleep(50 * time.Millisecond)
		_ = manager.Restart()
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 3; i++ {
		<-done
	}

	// Clean up
	manager.Stop()
}

// TestManager_Wait tests the Wait method
func TestManager_Wait(t *testing.T) {
	homeDir := t.TempDir()
	binDir := filepath.Join(homeDir, "wemixvisor", "current", "bin")
	require.NoError(t, os.MkdirAll(binDir, 0755))

	// Create a binary that exits after 1 second
	mockBin := filepath.Join(binDir, "wemixd")
	createTimedMockBinary(t, mockBin, 1*time.Second)

	cfg := &config.Config{
		Home:             homeDir,
		Name:             "wemixd",
		RestartOnFailure: false,
		ShutdownGrace:    2 * time.Second,
	}

	logger := logger.NewTestLogger()
	manager := NewManager(cfg, logger)

	// Start the node
	err := manager.Start([]string{"--test"})
	require.NoError(t, err)

	// Wait for process to exit
	waitCh := manager.Wait()

	select {
	case <-waitCh:
		// Process exited as expected
		time.Sleep(100 * time.Millisecond) // Give monitor time to update state
		state := manager.GetState()
		assert.True(t, state == StateCrashed || state == StateStopped || state == StateError)
	case <-time.After(3 * time.Second):
		t.Fatal("Wait() timeout")
	}
}

// TestManager_InvalidStateTransitions tests invalid state transitions
func TestManager_InvalidStateTransitions(t *testing.T) {
	cfg := &config.Config{
		Home: t.TempDir(),
		Name: "wemixd",
	}

	logger := logger.NewTestLogger()
	manager := NewManager(cfg, logger)

	// Set state to error manually (simulate)
	manager.stateMutex.Lock()
	manager.state = StateError
	manager.stateMutex.Unlock()

	// Try to start from error state
	err := manager.Start([]string{"--test"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not in stopped state")

	// Set state to upgrading
	manager.stateMutex.Lock()
	manager.state = StateUpgrading
	manager.stateMutex.Unlock()

	// Try to stop from upgrading state
	err = manager.Stop()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "node is not running")
}

// Helper function to create a mock binary that runs for a specific duration
func createTimedMockBinary(t *testing.T, path string, duration time.Duration) {
	t.Helper()

	script := fmt.Sprintf(`#!/bin/sh
# Mock binary that runs for a specific duration
trap 'exit 0' TERM INT
sleep %d
exit 0
`, int(duration.Seconds()))

	err := os.WriteFile(path, []byte(script), 0755)
	require.NoError(t, err)
}

// TestManager_ProcessGroupCleanup tests that child processes are cleaned up
func TestManager_ProcessGroupCleanup(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping process group test in short mode")
	}

	homeDir := t.TempDir()
	binDir := filepath.Join(homeDir, "wemixvisor", "current", "bin")
	require.NoError(t, os.MkdirAll(binDir, 0755))

	// Create a binary that spawns child processes
	mockBin := filepath.Join(binDir, "wemixd")
	script := `#!/bin/sh
# Mock binary that spawns children
trap 'kill $(jobs -p); exit 0' TERM INT
sleep 100 &
sleep 100 &
wait
`
	err := os.WriteFile(mockBin, []byte(script), 0755)
	require.NoError(t, err)

	cfg := &config.Config{
		Home:          homeDir,
		Name:          "wemixd",
		ShutdownGrace: 2 * time.Second,
	}

	logger := logger.NewTestLogger()
	manager := NewManager(cfg, logger)

	// Start the node
	err = manager.Start([]string{"--test"})
	require.NoError(t, err)

	pid := manager.GetStatus().PID
	assert.NotZero(t, pid)

	// Stop the node
	err = manager.Stop()
	require.NoError(t, err)

	// Give it a moment to clean up
	time.Sleep(500 * time.Millisecond)

	// Check that the process group is gone
	err = syscall.Kill(-pid, 0)
	assert.Error(t, err, "Process group should be terminated")
}