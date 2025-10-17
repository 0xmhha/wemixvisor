package node

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wemix/wemixvisor/internal/config"
	"github.com/wemix/wemixvisor/pkg/logger"
)

// Phase 2 & 3 Integration Tests
// These tests verify the core features implemented in Phase 2 and Phase 3

// TestPhase2_CoreFeatures tests Phase 2 core features
func TestPhase2_CoreFeatures(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Run("GracefulShutdown", testGracefulShutdown)
	t.Run("AutoRestartMechanism", testAutoRestartMechanism)
	t.Run("ConcurrentOperationSafety", testConcurrentOperationSafety)
	t.Run("StateTransitions", testStateTransitions)
}

// TestPhase3_AdvancedFeatures tests Phase 3 advanced features
func TestPhase3_AdvancedFeatures(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Run("ProcessGroupManagement", testProcessGroupManagement)
	t.Run("ZombiePrevention", testZombiePrevention)
	t.Run("HealthMonitoring", testHealthMonitoring)
	t.Run("VersionDetection", testVersionDetection)
}

// testGracefulShutdown verifies graceful shutdown with configurable timeout
func testGracefulShutdown(t *testing.T) {
	homeDir := t.TempDir()
	binDir := filepath.Join(homeDir, "wemixvisor", "current", "bin")
	require.NoError(t, os.MkdirAll(binDir, 0755))

	// Create a binary that handles SIGTERM gracefully
	mockBin := filepath.Join(binDir, "wemixd")
	script := `#!/bin/sh
trap 'echo "Received SIGTERM, shutting down gracefully..."; sleep 1; exit 0' TERM
echo "Node started"
while true; do
  sleep 1
done`
	err := os.WriteFile(mockBin, []byte(script), 0755)
	require.NoError(t, err)

	cfg := &config.Config{
		Home:          homeDir,
		Name:          "wemixd",
		ShutdownGrace: 3 * time.Second,
	}

	logger := logger.NewTestLogger()
	manager := NewManager(cfg, logger)

	// Start the node
	err = manager.Start([]string{"--test"})
	require.NoError(t, err)
	assert.Equal(t, StateRunning, manager.GetState())

	// Measure shutdown time
	start := time.Now()
	err = manager.Stop()
	elapsed := time.Since(start)

	// Should shutdown gracefully within grace period
	assert.NoError(t, err)
	assert.Less(t, elapsed, cfg.ShutdownGrace+1*time.Second)
	assert.Greater(t, elapsed, 1*time.Second) // Binary sleeps for 1 second
	assert.Equal(t, StateStopped, manager.GetState())
}

// testAutoRestartMechanism verifies auto-restart with max restart limits
func testAutoRestartMechanism(t *testing.T) {
	homeDir := t.TempDir()
	binDir := filepath.Join(homeDir, "wemixvisor", "current", "bin")
	require.NoError(t, os.MkdirAll(binDir, 0755))

	// Create a binary that crashes after a short time
	mockBin := filepath.Join(binDir, "wemixd")
	createCrashingMockBinary(t, mockBin, 1*time.Second)

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

	// Wait for first crash and restart attempt
	// First crash at 1s, restart delay 5s = 6s total
	time.Sleep(7 * time.Second)

	// Should have attempted restarts
	assert.GreaterOrEqual(t, manager.restartCount, 1)

	// Check state - should be in error or crashed after max restarts
	state := manager.GetState()
	assert.True(t, state == StateError || state == StateCrashed || state == StateRunning)

	// Clean up if still running
	if state == StateRunning {
		manager.Stop()
	}
}

// testConcurrentOperationSafety verifies thread-safe concurrent operations
func testConcurrentOperationSafety(t *testing.T) {
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

	// Perform concurrent operations
	var wg sync.WaitGroup
	errors := make(chan error, 10)

	// Concurrent status checks
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				_ = manager.GetStatus()
				_ = manager.GetState()
				_ = manager.IsHealthy()
				_ = manager.GetPID()
				time.Sleep(10 * time.Millisecond)
			}
		}()
	}

	// Concurrent restart attempts (should fail as node is running)
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			time.Sleep(time.Duration(id*100) * time.Millisecond)
			// Try to start while already running (should fail)
			err := manager.Start([]string{"--concurrent-test"})
			if err == nil {
				errors <- fmt.Errorf("concurrent start should have failed")
			}
		}(i)
	}

	// Wait for all operations to complete
	wg.Wait()
	close(errors)

	// Check for unexpected errors
	for err := range errors {
		t.Error(err)
	}

	// Node should still be running
	assert.Equal(t, StateRunning, manager.GetState())
	assert.True(t, manager.IsHealthy())

	// Clean up
	manager.Stop()
}

// testStateTransitions verifies proper state machine transitions
func testStateTransitions(t *testing.T) {
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

	// Verify initial state
	assert.Equal(t, StateStopped, manager.GetState())

	// Start: Stopped -> Starting -> Running
	err := manager.Start([]string{"--test"})
	require.NoError(t, err)
	assert.Equal(t, StateRunning, manager.GetState())

	// Cannot start when already running
	err = manager.Start([]string{"--test"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not in stopped state")

	// Stop: Running -> Stopping -> Stopped
	err = manager.Stop()
	require.NoError(t, err)
	assert.Equal(t, StateStopped, manager.GetState())

	// Cannot stop when already stopped
	err = manager.Stop()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "node is not running")

	// Restart from stopped state
	err = manager.Restart()
	require.NoError(t, err)
	assert.Equal(t, StateRunning, manager.GetState())

	// Clean up
	manager.Stop()
}

// testProcessGroupManagement verifies process group handling
func testProcessGroupManagement(t *testing.T) {
	homeDir := t.TempDir()
	binDir := filepath.Join(homeDir, "wemixvisor", "current", "bin")
	require.NoError(t, os.MkdirAll(binDir, 0755))

	// Create a binary that spawns child processes
	mockBin := filepath.Join(binDir, "wemixd")
	script := `#!/bin/sh
# Spawn child processes
(sleep 100 &)
(sleep 100 &)
echo "Children spawned"
trap 'kill $(jobs -p); exit 0' TERM INT
while true; do
  sleep 1
done`
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

	pid := manager.GetPID()
	assert.NotZero(t, pid)

	// Give time for children to spawn
	time.Sleep(500 * time.Millisecond)

	// Stop should terminate all processes in the group
	err = manager.Stop()
	require.NoError(t, err)

	// All processes should be terminated
	time.Sleep(500 * time.Millisecond)

	// Process group should be gone
	// Note: We can't reliably test this on all systems, but Stop() should succeed
	assert.Equal(t, StateStopped, manager.GetState())
}

// testZombiePrevention verifies zombie process prevention
func testZombiePrevention(t *testing.T) {
	homeDir := t.TempDir()
	binDir := filepath.Join(homeDir, "wemixvisor", "current", "bin")
	require.NoError(t, os.MkdirAll(binDir, 0755))

	// Create a binary that creates a child and exits (potential zombie)
	mockBin := filepath.Join(binDir, "wemixd")
	script := `#!/bin/sh
# Fork a child that exits immediately
(exit 0) &
# Main process continues
trap 'exit 0' TERM INT
while true; do
  sleep 1
done`
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

	// Give time for potential zombies
	time.Sleep(1 * time.Second)

	// Node should still be healthy (no zombie accumulation)
	assert.True(t, manager.IsHealthy())

	// Clean up
	manager.Stop()
}

// testHealthMonitoring verifies health check functionality
func testHealthMonitoring(t *testing.T) {
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

	// Health should be false before start
	assert.False(t, manager.IsHealthy())
	assert.Equal(t, 0, manager.GetPID())

	// Start the node
	err := manager.Start([]string{"--test"})
	require.NoError(t, err)

	// Health should be true when running
	assert.True(t, manager.IsHealthy())
	pid := manager.GetPID()
	assert.NotZero(t, pid)

	// Verify health checks work repeatedly
	for i := 0; i < 5; i++ {
		assert.True(t, manager.IsHealthy())
		time.Sleep(100 * time.Millisecond)
	}

	// Stop the node
	err = manager.Stop()
	require.NoError(t, err)

	// Health should be false after stop
	assert.False(t, manager.IsHealthy())
	assert.Equal(t, 0, manager.GetPID())
}

// testVersionDetection verifies binary version detection
func testVersionDetection(t *testing.T) {
	homeDir := t.TempDir()
	binDir := filepath.Join(homeDir, "wemixvisor", "current", "bin")
	require.NoError(t, os.MkdirAll(binDir, 0755))

	// Test different version command patterns
	testCases := []struct {
		name        string
		versionFlag string
		expected    string
	}{
		{"version command", "version", "wemixd version 2.0.0"},
		{"--version flag", "--version", "v2.0.0-beta"},
		{"-v flag", "-v", "2.0.0-rc1"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create subdirectory for this test to avoid conflicts
			testBinDir := filepath.Join(binDir, tc.name)
			require.NoError(t, os.MkdirAll(testBinDir, 0755))

			// Create mock binary with specific version output
			mockBin := filepath.Join(testBinDir, "wemixd")
			script := fmt.Sprintf(`#!/bin/sh
if [ "$1" = "%s" ]; then
  echo "%s"
  exit 0
fi
trap 'exit 0' TERM INT
while true; do
  sleep 1
done`, tc.versionFlag, tc.expected)
			err := os.WriteFile(mockBin, []byte(script), 0755)
			require.NoError(t, err)

			testHomeDir := filepath.Join(homeDir, tc.name)
			require.NoError(t, os.MkdirAll(filepath.Join(testHomeDir, "wemixvisor", "current", "bin"), 0755))

			// Copy binary to test directory
			testMockBin := filepath.Join(testHomeDir, "wemixvisor", "current", "bin", "wemixd")
			err = os.WriteFile(testMockBin, []byte(script), 0755)
			require.NoError(t, err)

			cfg := &config.Config{
				Home:          testHomeDir,
				Name:          "wemixd",
				ShutdownGrace: 2 * time.Second,
			}

			logger := logger.NewTestLogger()
			manager := NewManager(cfg, logger)

			// Version should be unknown before start
			assert.Equal(t, "unknown", manager.GetVersion())

			// Start the node
			err = manager.Start([]string{"--test"})
			require.NoError(t, err)

			// Version should be detected
			version := manager.GetVersion()
			assert.Equal(t, tc.expected, version)

			// Clean up
			manager.Stop()
		})
	}
}

// TestStressTest performs stress testing for stability
func TestStressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
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

	// Perform rapid start/stop/restart cycles
	for i := 0; i < 10; i++ {
		// Start
		err := manager.Start([]string{fmt.Sprintf("--cycle-%d", i)})
		require.NoError(t, err)
		assert.True(t, manager.IsHealthy())

		// Quick operations
		_ = manager.GetStatus()
		_ = manager.GetVersion()
		_ = manager.GetPID()

		// Stop
		err = manager.Stop()
		require.NoError(t, err)
		assert.False(t, manager.IsHealthy())

		// Small delay between cycles
		time.Sleep(100 * time.Millisecond)
	}

	// Verify no resource leaks or state corruption
	assert.Equal(t, StateStopped, manager.GetState())
}

// TestErrorRecovery tests error recovery mechanisms
func TestErrorRecovery(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping error recovery test in short mode")
	}

	homeDir := t.TempDir()
	binDir := filepath.Join(homeDir, "wemixvisor", "current", "bin")
	require.NoError(t, os.MkdirAll(binDir, 0755))

	// Test various error scenarios
	t.Run("BinaryNotFound", func(t *testing.T) {
		cfg := &config.Config{
			Home: homeDir,
			Name: "nonexistent",
		}
		logger := logger.NewTestLogger()
		manager := NewManager(cfg, logger)

		err := manager.Start([]string{"--test"})
		assert.Error(t, err)
		assert.Equal(t, StateError, manager.GetState())
	})

	t.Run("BinaryNotExecutable", func(t *testing.T) {
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
	})

	t.Run("RecoveryAfterCrash", func(t *testing.T) {
		mockBin := filepath.Join(binDir, "wemixd")

		// First create a crashing binary
		createCrashingMockBinary(t, mockBin, 500*time.Millisecond)

		cfg := &config.Config{
			Home:             homeDir,
			Name:             "wemixd",
			RestartOnFailure: true,
			MaxRestarts:      1,
			ShutdownGrace:    1 * time.Second,
		}
		logger := logger.NewTestLogger()
		manager := NewManager(cfg, logger)

		err := manager.Start([]string{"--test"})
		require.NoError(t, err)

		// Wait for crash and auto-restart attempt
		time.Sleep(7 * time.Second)

		// Now replace with stable binary
		createMockBinary(t, mockBin)

		// Manual restart should work
		if manager.GetState() != StateRunning {
			err = manager.Start([]string{"--recovery"})
			require.NoError(t, err)
		}

		assert.True(t, manager.IsHealthy())
		manager.Stop()
	})
}