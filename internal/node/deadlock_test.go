package node

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/wemix/wemixvisor/internal/config"
	"github.com/wemix/wemixvisor/pkg/logger"
)

// TestNoDeadlock verifies that the metrics collector doesn't cause deadlock
func TestNoDeadlock(t *testing.T) {
	cfg := &config.Config{
		Home: t.TempDir(),
		Name: "test",
	}

	logger := logger.NewTestLogger()
	manager := NewManager(cfg, logger)

	// Mock the state to simulate a running node
	manager.stateMutex.Lock()
	manager.state = StateRunning
	manager.startTime = time.Now()
	manager.restartCount = 5
	// Note: IsHealthy() will return false because there's no actual process
	// This is expected in the test environment
	manager.stateMutex.Unlock()

	// Test that metrics collection doesn't deadlock
	done := make(chan bool)
	go func() {
		// This would deadlock in the old code
		count := manager.GetRestartCount()
		assert.Equal(t, 5, count)

		uptime := manager.GetUptime()
		assert.NotZero(t, uptime)

		// In test environment, IsHealthy returns false (no actual process)
		healthy := manager.IsHealthy()
		assert.False(t, healthy)

		done <- true
	}()

	// Wait for completion or timeout
	select {
	case <-done:
		// Success - no deadlock
		t.Log("No deadlock detected")
	case <-time.After(1 * time.Second):
		t.Fatal("Deadlock detected: GetRestartCount/GetUptime/IsHealthy did not complete")
	}
}

// TestConcurrentAccess tests concurrent access to manager state
func TestConcurrentAccess(t *testing.T) {
	cfg := &config.Config{
		Home: t.TempDir(),
		Name: "test",
	}

	logger := logger.NewTestLogger()
	manager := NewManager(cfg, logger)

	// Set initial state
	manager.stateMutex.Lock()
	manager.state = StateRunning
	manager.startTime = time.Now()
	manager.restartCount = 0
	manager.stateMutex.Unlock()

	// Run concurrent operations
	done := make(chan bool)

	// Simulate metrics collector accessing state
	go func() {
		for i := 0; i < 100; i++ {
			_ = manager.GetRestartCount()
			_ = manager.GetUptime()
			_ = manager.IsHealthy()
			time.Sleep(time.Millisecond)
		}
		done <- true
	}()

	// Simulate state changes
	go func() {
		for i := 0; i < 100; i++ {
			manager.stateMutex.Lock()
			manager.restartCount++
			manager.stateMutex.Unlock()
			time.Sleep(time.Millisecond)
		}
		done <- true
	}()

	// Wait for both goroutines
	timeout := time.After(5 * time.Second)
	for i := 0; i < 2; i++ {
		select {
		case <-done:
			// One goroutine completed
		case <-timeout:
			t.Fatal("Deadlock detected: Concurrent operations did not complete")
		}
	}

	// Verify final state
	assert.Equal(t, 100, manager.GetRestartCount())
}

// TestMetricsCollectorStart tests that metrics collector can start without deadlock
func TestMetricsCollectorStart(t *testing.T) {
	cfg := &config.Config{
		Home:                      t.TempDir(),
		Name:                      "test",
		MetricsEnabled:            true,
		MetricsCollectionInterval: 100 * time.Millisecond,
	}

	logger := logger.NewTestLogger()
	manager := NewManager(cfg, logger)

	// This simulates what happens during Start()
	done := make(chan bool)
	go func() {
		manager.stateMutex.Lock()
		manager.state = StateRunning
		manager.startTime = time.Now()

		// In the old code, this would deadlock because metrics collector
		// would try to acquire RLock while we hold Lock
		manager.metricsCollector.Start()

		manager.stateMutex.Unlock()
		done <- true
	}()

	// Wait for completion or timeout
	select {
	case <-done:
		t.Log("MetricsCollector started successfully without deadlock")

		// Give metrics collector time to run
		time.Sleep(200 * time.Millisecond)

		// Stop metrics collector
		manager.metricsCollector.Stop()

		// Verify metrics were collected
		metrics := manager.GetMetrics()
		assert.NotZero(t, metrics.Timestamp)

	case <-time.After(2 * time.Second):
		t.Fatal("Deadlock detected: MetricsCollector.Start() did not complete")
	}
}