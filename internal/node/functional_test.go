package node

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wemix/wemixvisor/internal/config"
	"github.com/wemix/wemixvisor/pkg/logger"
)

// Functional tests for Phase 2 and Phase 3 features
// These tests verify that all implemented features work correctly

func TestPhase2Phase3_Functional(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping functional tests in short mode")
	}
	t.Run("BasicLifecycle", testBasicLifecycle)
	t.Run("GracefulShutdown", testGracefulShutdownFunctional)
	t.Run("HealthChecks", testHealthChecksFunctional)
	t.Run("VersionDetection", testVersionDetectionFunctional)
	t.Run("ProcessManagement", testProcessManagementFunctional)
	t.Run("StateManagement", testStateManagementFunctional)
}

func testBasicLifecycle(t *testing.T) {
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

	// Test complete lifecycle
	assert.Equal(t, StateStopped, manager.GetState())
	assert.False(t, manager.IsHealthy())
	assert.Equal(t, 0, manager.GetPID())

	// Start
	err := manager.Start([]string{"--test"})
	require.NoError(t, err)
	assert.Equal(t, StateRunning, manager.GetState())
	assert.True(t, manager.IsHealthy())
	assert.NotZero(t, manager.GetPID())

	// Status
	status := manager.GetStatus()
	assert.Equal(t, StateRunning, status.State)
	assert.NotZero(t, status.PID)
	assert.NotZero(t, status.StartTime)

	// Stop
	err = manager.Stop()
	require.NoError(t, err)
	assert.Equal(t, StateStopped, manager.GetState())
	assert.False(t, manager.IsHealthy())
	assert.Equal(t, 0, manager.GetPID())
}

func testGracefulShutdownFunctional(t *testing.T) {
	homeDir := t.TempDir()
	binDir := filepath.Join(homeDir, "wemixvisor", "current", "bin")
	require.NoError(t, os.MkdirAll(binDir, 0755))

	// Binary that responds to SIGTERM
	mockBin := filepath.Join(binDir, "wemixd")
	script := `#!/bin/sh
trap 'echo "Graceful shutdown"; exit 0' TERM
while true; do sleep 1; done`
	err := os.WriteFile(mockBin, []byte(script), 0755)
	require.NoError(t, err)

	cfg := &config.Config{
		Home:          homeDir,
		Name:          "wemixd",
		ShutdownGrace: 3 * time.Second,
	}

	logger := logger.NewTestLogger()
	manager := NewManager(cfg, logger)

	err = manager.Start([]string{"--test"})
	require.NoError(t, err)

	// Graceful shutdown should work within timeout
	start := time.Now()
	err = manager.Stop()
	elapsed := time.Since(start)

	assert.NoError(t, err)
	assert.Less(t, elapsed, cfg.ShutdownGrace+1*time.Second)
	assert.Equal(t, StateStopped, manager.GetState())
}

func testHealthChecksFunctional(t *testing.T) {
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

	// Health check before start
	assert.False(t, manager.IsHealthy())

	// Start and verify health
	err := manager.Start([]string{"--test"})
	require.NoError(t, err)
	assert.True(t, manager.IsHealthy())

	// Continuous health checks
	for i := 0; i < 3; i++ {
		time.Sleep(100 * time.Millisecond)
		assert.True(t, manager.IsHealthy())
	}

	// Stop and verify health
	err = manager.Stop()
	require.NoError(t, err)
	assert.False(t, manager.IsHealthy())
}

func testVersionDetectionFunctional(t *testing.T) {
	homeDir := t.TempDir()
	binDir := filepath.Join(homeDir, "wemixvisor", "current", "bin")
	require.NoError(t, os.MkdirAll(binDir, 0755))

	// Test version command
	mockBin := filepath.Join(binDir, "wemixd")
	script := `#!/bin/sh
if [ "$1" = "version" ]; then
  echo "wemixd version 1.0.0"
  exit 0
fi
trap 'exit 0' TERM
while true; do sleep 1; done`
	err := os.WriteFile(mockBin, []byte(script), 0755)
	require.NoError(t, err)

	cfg := &config.Config{
		Home:          homeDir,
		Name:          "wemixd",
		ShutdownGrace: 2 * time.Second,
	}

	logger := logger.NewTestLogger()
	manager := NewManager(cfg, logger)

	// Version should be unknown before start
	assert.Equal(t, "unknown", manager.GetVersion())

	// Start and check version
	err = manager.Start([]string{"--test"})
	require.NoError(t, err)

	// Version should be detected
	version := manager.GetVersion()
	assert.Contains(t, version, "1.0.0")

	manager.Stop()
}

func testProcessManagementFunctional(t *testing.T) {
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

	// PID management
	assert.Equal(t, 0, manager.GetPID())

	err := manager.Start([]string{"--test"})
	require.NoError(t, err)

	pid := manager.GetPID()
	assert.NotZero(t, pid)

	// Process should be running
	assert.True(t, manager.IsHealthy())

	err = manager.Stop()
	require.NoError(t, err)

	// PID should be reset
	assert.Equal(t, 0, manager.GetPID())
}

func testStateManagementFunctional(t *testing.T) {
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

	// Test state transitions
	assert.Equal(t, StateStopped, manager.GetState())

	// Start
	err := manager.Start([]string{"--test"})
	require.NoError(t, err)
	assert.Equal(t, StateRunning, manager.GetState())

	// Restart
	err = manager.Restart()
	require.NoError(t, err)
	assert.Equal(t, StateRunning, manager.GetState())
	assert.Equal(t, 1, manager.restartCount)

	// Stop
	err = manager.Stop()
	require.NoError(t, err)
	assert.Equal(t, StateStopped, manager.GetState())

	// Error conditions
	err = manager.Stop() // Stop when already stopped
	assert.Error(t, err)

	err = manager.Start([]string{"--test"})
	require.NoError(t, err)

	err = manager.Start([]string{"--test"}) // Start when already running
	assert.Error(t, err)

	manager.Stop()
}

// TestAutoRestartFunctional tests auto-restart functionality
func TestAutoRestartFunctional(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping auto-restart test in short mode")
	}

	homeDir := t.TempDir()
	binDir := filepath.Join(homeDir, "wemixvisor", "current", "bin")
	require.NoError(t, os.MkdirAll(binDir, 0755))

	// Create crashing binary
	mockBin := filepath.Join(binDir, "wemixd")
	createCrashingMockBinary(t, mockBin, 1*time.Second)

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

	// Wait for crash and auto-restart
	time.Sleep(7 * time.Second)

	// Should have attempted at least one restart
	assert.GreaterOrEqual(t, manager.restartCount, 1)

	// Clean up
	if manager.GetState() == StateRunning {
		manager.Stop()
	}
}

// TestConcurrentAccessFunctional tests thread safety
func TestConcurrentAccessFunctional(t *testing.T) {
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

	err := manager.Start([]string{"--test"})
	require.NoError(t, err)

	// Concurrent read operations should be safe
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 50; j++ {
				_ = manager.GetState()
				_ = manager.IsHealthy()
				_ = manager.GetPID()
				_ = manager.GetStatus()
				_ = manager.GetVersion()
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Manager should still be healthy
	assert.True(t, manager.IsHealthy())
	assert.Equal(t, StateRunning, manager.GetState())

	manager.Stop()
}

// TestEnvironmentVariables tests environment variable handling
func TestEnvironmentVariables(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping functional test in short mode")
	}
	homeDir := t.TempDir()
	binDir := filepath.Join(homeDir, "wemixvisor", "current", "bin")
	require.NoError(t, os.MkdirAll(binDir, 0755))

	mockBin := filepath.Join(binDir, "wemixd")
	createMockBinary(t, mockBin)

	cfg := &config.Config{
		Home:    homeDir,
		Name:    "wemixd",
		Network: "testnet",
		Environment: map[string]string{
			"TEST_VAR": "test_value",
		},
		ShutdownGrace: 1 * time.Second,
	}

	logger := logger.NewTestLogger()
	manager := NewManager(cfg, logger)

	// Test environment building
	env := manager.buildEnvironment()

	foundWemixHome := false
	foundNetwork := false
	foundCustom := false

	for _, e := range env {
		if e == "WEMIX_HOME="+homeDir {
			foundWemixHome = true
		}
		if e == "WEMIX_NETWORK=testnet" {
			foundNetwork = true
		}
		if e == "TEST_VAR=test_value" {
			foundCustom = true
		}
	}

	assert.True(t, foundWemixHome)
	assert.True(t, foundNetwork)
	assert.True(t, foundCustom)

	// Test actual execution with environment
	err := manager.Start([]string{"--test"})
	require.NoError(t, err)
	assert.True(t, manager.IsHealthy())

	manager.Stop()
}

// TestStatusInformation tests status reporting
func TestStatusInformation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping functional test in short mode")
	}
	homeDir := t.TempDir()
	binDir := filepath.Join(homeDir, "wemixvisor", "current", "bin")
	require.NoError(t, os.MkdirAll(binDir, 0755))

	mockBin := filepath.Join(binDir, "wemixd")
	createVersionedMockBinary(t, mockBin, "v1.2.3")

	cfg := &config.Config{
		Home:          homeDir,
		Name:          "wemixd",
		Network:       "mainnet",
		ShutdownGrace: 1 * time.Second,
	}

	logger := logger.NewTestLogger()
	manager := NewManager(cfg, logger)

	// Status when stopped
	status := manager.GetStatus()
	assert.Equal(t, StateStopped, status.State)
	assert.Equal(t, "stopped", status.StateString)
	assert.Equal(t, 0, status.PID)
	assert.Equal(t, "mainnet", status.Network)

	// Start and check status
	err := manager.Start([]string{"--test"})
	require.NoError(t, err)

	status = manager.GetStatus()
	assert.Equal(t, StateRunning, status.State)
	assert.Equal(t, "running", status.StateString)
	assert.NotZero(t, status.PID)
	assert.NotZero(t, status.StartTime)
	assert.NotZero(t, status.Uptime)
	assert.Equal(t, "mainnet", status.Network)
	assert.Contains(t, status.Version, "v1.2.3")

	manager.Stop()
}

// TestErrorHandling tests error conditions
func TestErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping functional test in short mode")
	}
	homeDir := t.TempDir()

	cfg := &config.Config{
		Home: homeDir,
		Name: "nonexistent",
	}

	logger := logger.NewTestLogger()
	manager := NewManager(cfg, logger)

	// Start with nonexistent binary
	err := manager.Start([]string{"--test"})
	assert.Error(t, err)
	assert.Equal(t, StateError, manager.GetState())

	// Operations on errored manager
	assert.False(t, manager.IsHealthy())
	assert.Equal(t, 0, manager.GetPID())
	assert.Equal(t, "unknown", manager.GetVersion())

	// Stop should fail
	err = manager.Stop()
	assert.Error(t, err)

	// Restart should reset state and retry
	// First create the binary
	binDir := filepath.Join(homeDir, "wemixvisor", "current", "bin")
	require.NoError(t, os.MkdirAll(binDir, 0755))
	mockBin := filepath.Join(binDir, "nonexistent")
	createMockBinary(t, mockBin)

	err = manager.Restart()
	require.NoError(t, err)
	assert.Equal(t, StateRunning, manager.GetState())

	manager.Stop()
}