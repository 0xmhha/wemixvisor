package monitor

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/wemix/wemixvisor/internal/config"
	"github.com/wemix/wemixvisor/pkg/logger"
)

func TestNewHealthChecker(t *testing.T) {
	cfg := &config.Config{
		HealthCheckInterval: 10 * time.Second,
		RPCPort:             8545,
	}
	logger, _ := logger.New(true, false, "")

	checker := NewHealthChecker(cfg, logger)
	assert.NotNil(t, checker)
	assert.Equal(t, 10*time.Second, checker.checkInterval)
	assert.Equal(t, "http://localhost:8545", checker.rpcURL)
	assert.Len(t, checker.checks, 4) // Should have 4 default checks
}

func TestHealthChecker_Start_Stop(t *testing.T) {
	cfg := &config.Config{
		HealthCheckInterval: 100 * time.Millisecond, // Short interval for testing
	}
	logger, _ := logger.New(true, false, "")

	checker := NewHealthChecker(cfg, logger)
	statusCh := checker.Start()

	// Should receive at least one status update
	select {
	case status := <-statusCh:
		assert.NotNil(t, status)
		assert.NotZero(t, status.Timestamp)
		assert.NotNil(t, status.Checks)
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for health status")
	}

	// Stop the checker
	checker.Stop()

	// Channel should be closed
	select {
	case _, ok := <-statusCh:
		assert.False(t, ok, "channel should be closed")
	case <-time.After(100 * time.Millisecond):
		// Channel might already be closed
	}
}

func TestHealthChecker_GetStatus(t *testing.T) {
	cfg := &config.Config{}
	logger, _ := logger.New(true, false, "")

	checker := NewHealthChecker(cfg, logger)

	// Initial status should be empty
	status := checker.GetStatus()
	assert.False(t, status.Healthy)
	assert.Empty(t, status.Checks)

	// Start checker and wait for first check
	statusCh := checker.Start()
	defer checker.Stop()

	// Wait for first status update
	select {
	case <-statusCh:
		// Status received
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for health status")
	}

	// Now get status should have data
	status = checker.GetStatus()
	assert.NotZero(t, status.Timestamp)
	assert.NotEmpty(t, status.Checks)
}

func TestHealthChecker_IsHealthy(t *testing.T) {
	cfg := &config.Config{}
	logger, _ := logger.New(true, false, "")

	checker := NewHealthChecker(cfg, logger)

	// Initially should be unhealthy (no checks performed)
	assert.False(t, checker.IsHealthy())

	// Manually set a healthy status
	checker.statusMutex.Lock()
	checker.lastStatus = HealthStatus{
		Healthy:   true,
		Timestamp: time.Now(),
	}
	checker.statusMutex.Unlock()

	assert.True(t, checker.IsHealthy())
}

func TestHealthChecker_performChecks(t *testing.T) {
	cfg := &config.Config{}
	logger, _ := logger.New(true, false, "")

	checker := NewHealthChecker(cfg, logger)

	// Add a simple test check that always passes
	checker.checks = []HealthCheck{
		&mockHealthCheck{name: "test_pass", shouldFail: false},
		&mockHealthCheck{name: "test_fail", shouldFail: true},
	}

	// Perform checks
	checker.performChecks()

	// Get status
	status := checker.GetStatus()
	assert.False(t, status.Healthy) // Should be unhealthy due to failed check
	assert.Len(t, status.Checks, 2)

	// Check individual results
	assert.True(t, status.Checks["test_pass"].Healthy)
	assert.False(t, status.Checks["test_fail"].Healthy)
	assert.NotEmpty(t, status.Checks["test_fail"].Error)
}

// Mock health check for testing
type mockHealthCheck struct {
	name       string
	shouldFail bool
}

func (m *mockHealthCheck) Name() string {
	return m.name
}

func (m *mockHealthCheck) Check(ctx context.Context) error {
	if m.shouldFail {
		return assert.AnError
	}
	return nil
}