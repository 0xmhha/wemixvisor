package metrics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/wemix/wemixvisor/internal/config"
	"github.com/wemix/wemixvisor/pkg/logger"
)

// Mock NodeInfoProvider for testing
type mockNodeInfoProvider struct {
	uptime       time.Duration
	restartCount int
	healthy      bool
	pid          int
}

func (m *mockNodeInfoProvider) GetUptime() time.Duration {
	return m.uptime
}

func (m *mockNodeInfoProvider) GetRestartCount() int {
	return m.restartCount
}

func (m *mockNodeInfoProvider) IsHealthy() bool {
	return m.healthy
}

func (m *mockNodeInfoProvider) GetPID() int {
	return m.pid
}

func TestNewMetricsCollector(t *testing.T) {
	cfg := &config.Config{
		MetricsInterval: 30 * time.Second,
	}
	logger, _ := logger.New(true, false, "")
	nodeInfo := &mockNodeInfoProvider{}

	collector := NewMetricsCollector(cfg, logger, nodeInfo)

	assert.NotNil(t, collector)
	assert.Equal(t, cfg, collector.config)
	assert.Equal(t, logger, collector.logger)
	assert.Equal(t, nodeInfo, collector.nodeInfo)
	assert.NotNil(t, collector.ctx)
	assert.NotNil(t, collector.cancel)
	assert.NotNil(t, collector.collectionTicker)
}

func TestNewMetricsCollector_DefaultInterval(t *testing.T) {
	cfg := &config.Config{} // No interval set
	logger, _ := logger.New(true, false, "")
	nodeInfo := &mockNodeInfoProvider{}

	collector := NewMetricsCollector(cfg, logger, nodeInfo)

	assert.NotNil(t, collector)
	assert.NotNil(t, collector.collectionTicker)
	// Can't easily test the exact interval of a ticker, just verify it's not nil
}

func TestMetricsCollector_CollectMetrics(t *testing.T) {
	cfg := &config.Config{}
	logger, _ := logger.New(true, false, "")
	nodeInfo := &mockNodeInfoProvider{
		uptime:       2 * time.Hour,
		restartCount: 3,
		healthy:      true,
		pid:          1234,
	}

	collector := NewMetricsCollector(cfg, logger, nodeInfo)

	// Collect metrics
	collector.collectMetrics()

	// Get metrics
	metrics := collector.GetMetrics()

	// Verify basic metrics
	assert.True(t, metrics.Healthy)
	assert.Equal(t, 3, metrics.RestartCount)
	assert.Equal(t, int64(2*60*60), metrics.NodeUptime) // 2 hours in seconds
	assert.True(t, metrics.MemoryUsageMB > 0)           // Should have some memory usage
	assert.NotZero(t, metrics.Timestamp)
}

func TestMetricsCollector_CollectMetrics_ZeroUptime(t *testing.T) {
	cfg := &config.Config{}
	logger, _ := logger.New(true, false, "")
	nodeInfo := &mockNodeInfoProvider{
		uptime:       0, // No uptime
		restartCount: 0,
		healthy:      false,
		pid:          0,
	}

	collector := NewMetricsCollector(cfg, logger, nodeInfo)

	// Collect metrics
	collector.collectMetrics()

	// Get metrics
	metrics := collector.GetMetrics()

	// Verify metrics
	assert.False(t, metrics.Healthy)
	assert.Equal(t, 0, metrics.RestartCount)
	assert.Equal(t, int64(0), metrics.NodeUptime)
}

func TestMetricsCollector_StartStop(t *testing.T) {
	cfg := &config.Config{
		MetricsInterval: 10 * time.Millisecond, // Very short for testing
	}
	logger, _ := logger.New(true, false, "")
	nodeInfo := &mockNodeInfoProvider{
		healthy: true,
	}

	collector := NewMetricsCollector(cfg, logger, nodeInfo)

	// Start collector
	collector.Start()

	// Wait a bit for collection to happen
	time.Sleep(50 * time.Millisecond)

	// Get metrics (should have been collected)
	metrics := collector.GetMetrics()
	assert.True(t, metrics.Healthy)
	assert.NotZero(t, metrics.Timestamp)

	// Stop collector
	collector.Stop()

	// Verify context is cancelled
	select {
	case <-collector.ctx.Done():
		// Context cancelled as expected
	case <-time.After(100 * time.Millisecond):
		t.Fatal("context should be cancelled after Stop()")
	}
}

func TestMetricsCollector_GetMetrics_ThreadSafe(t *testing.T) {
	cfg := &config.Config{}
	logger, _ := logger.New(true, false, "")
	nodeInfo := &mockNodeInfoProvider{
		healthy: true,
	}

	collector := NewMetricsCollector(cfg, logger, nodeInfo)

	// Collect initial metrics
	collector.collectMetrics()

	// Test concurrent access
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				metrics := collector.GetMetrics()
				assert.True(t, metrics.Healthy)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestMetricsCollector_ExportJSON(t *testing.T) {
	cfg := &config.Config{}
	logger, _ := logger.New(true, false, "")
	nodeInfo := &mockNodeInfoProvider{
		uptime:       1 * time.Hour,
		restartCount: 2,
		healthy:      true,
	}

	collector := NewMetricsCollector(cfg, logger, nodeInfo)
	collector.collectMetrics()

	// Export as JSON
	data, err := collector.ExportMetrics("json")
	assert.NoError(t, err)
	assert.NotEmpty(t, data)

	// Verify JSON contains expected fields
	jsonStr := string(data)
	assert.Contains(t, jsonStr, "timestamp")
	assert.Contains(t, jsonStr, "node_uptime_seconds")
	assert.Contains(t, jsonStr, "restart_count")
	assert.Contains(t, jsonStr, "memory_usage_mb")
	assert.Contains(t, jsonStr, "healthy")
}

func TestMetricsCollector_ExportPrometheus(t *testing.T) {
	cfg := &config.Config{}
	logger, _ := logger.New(true, false, "")
	nodeInfo := &mockNodeInfoProvider{
		uptime:       1 * time.Hour,
		restartCount: 2,
		healthy:      true,
	}

	collector := NewMetricsCollector(cfg, logger, nodeInfo)
	collector.collectMetrics()

	// Export as Prometheus
	data, err := collector.ExportMetrics("prometheus")
	assert.NoError(t, err)
	assert.NotEmpty(t, data)

	// Verify Prometheus format
	promStr := string(data)
	assert.Contains(t, promStr, "# HELP wemixvisor_uptime_seconds")
	assert.Contains(t, promStr, "# TYPE wemixvisor_uptime_seconds gauge")
	assert.Contains(t, promStr, "wemixvisor_uptime_seconds")
	assert.Contains(t, promStr, "# HELP wemixvisor_restart_count")
	assert.Contains(t, promStr, "# TYPE wemixvisor_restart_count counter")
	assert.Contains(t, promStr, "wemixvisor_restart_count")
	assert.Contains(t, promStr, "# HELP wemixvisor_memory_usage_mb")
	assert.Contains(t, promStr, "wemixvisor_memory_usage_mb")
	assert.Contains(t, promStr, "# HELP wemixvisor_healthy")
	assert.Contains(t, promStr, "wemixvisor_healthy")
}

func TestMetricsCollector_ExportDefaultFormat(t *testing.T) {
	cfg := &config.Config{}
	logger, _ := logger.New(true, false, "")
	nodeInfo := &mockNodeInfoProvider{}

	collector := NewMetricsCollector(cfg, logger, nodeInfo)
	collector.collectMetrics()

	// Export with unknown format (should default to JSON)
	data, err := collector.ExportMetrics("unknown")
	assert.NoError(t, err)
	assert.NotEmpty(t, data)

	// Should be JSON format
	jsonStr := string(data)
	assert.Contains(t, jsonStr, "{")
	assert.Contains(t, jsonStr, "timestamp")
}

func TestBoolToInt(t *testing.T) {
	assert.Equal(t, 1, boolToInt(true))
	assert.Equal(t, 0, boolToInt(false))
}

func TestMetricsCollector_ContextCancellation(t *testing.T) {
	cfg := &config.Config{
		MetricsInterval: 1 * time.Millisecond,
	}
	logger, _ := logger.New(true, false, "")
	nodeInfo := &mockNodeInfoProvider{}

	collector := NewMetricsCollector(cfg, logger, nodeInfo)

	// Start collector
	collector.Start()

	// Cancel context directly
	collector.cancel()

	// Context should be done
	select {
	case <-collector.ctx.Done():
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Fatal("context should be done after cancel")
	}
}

func TestMetricsCollector_Run_Lifecycle(t *testing.T) {
	cfg := &config.Config{
		MetricsInterval: 5 * time.Millisecond, // Very short interval
	}
	logger, _ := logger.New(true, false, "")
	nodeInfo := &mockNodeInfoProvider{
		healthy: true,
	}

	collector := NewMetricsCollector(cfg, logger, nodeInfo)

	// Start the run goroutine
	go collector.run()

	// Wait for a few collection cycles
	time.Sleep(20 * time.Millisecond)

	// Metrics should have been collected
	metrics := collector.GetMetrics()
	assert.True(t, metrics.Healthy)
	assert.NotZero(t, metrics.Timestamp)

	// Stop the collector
	collector.cancel()

	// Verify goroutine exits
	select {
	case <-collector.ctx.Done():
		// Context cancelled as expected
	case <-time.After(100 * time.Millisecond):
		t.Fatal("run goroutine should exit after context cancellation")
	}
}