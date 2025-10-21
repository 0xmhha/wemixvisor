package metrics

import (
	"context"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wemix/wemixvisor/pkg/logger"
)

// waitForSnapshot polls for a non-nil snapshot with timeout
func waitForSnapshot(collector *Collector, timeout time.Duration) *MetricsSnapshot {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if snapshot := collector.GetSnapshot(); snapshot != nil {
			return snapshot
		}
		time.Sleep(10 * time.Millisecond)
	}
	return nil
}

// TestNewCollector tests the creation of a new collector
func TestNewCollector(t *testing.T) {
	tests := []struct {
		name   string
		config *CollectorConfig
	}{
		{
			name: "default configuration",
			config: &CollectorConfig{
				Enabled:            true,
				CollectionInterval: 1 * time.Second,
				PrometheusPort:     9090,
				EnableSystemMetrics: true,
				EnableAppMetrics:   true,
				EnableGovMetrics:   true,
				EnablePerfMetrics:  true,
			},
		},
		{
			name: "minimal configuration",
			config: &CollectorConfig{
				Enabled:            false,
				CollectionInterval: 5 * time.Second,
				PrometheusPort:     9091,
				EnableSystemMetrics: false,
				EnableAppMetrics:   false,
				EnableGovMetrics:   false,
				EnablePerfMetrics:  false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			logger := logger.NewTestLogger()

			// Act
			collector := NewCollector(tt.config, logger)

			// Assert
			assert.NotNil(t, collector)
			assert.Equal(t, tt.config, collector.config)
			assert.NotNil(t, collector.logger)
			assert.NotNil(t, collector.registry)
			assert.NotNil(t, collector.startTime)
		})
	}
}

// TestCollectorStartStop tests starting and stopping the collector
func TestCollectorStartStop(t *testing.T) {
	tests := []struct {
		name         string
		enabled      bool
		expectError  bool
		waitDuration time.Duration
	}{
		{
			name:         "start and stop enabled collector",
			enabled:      true,
			expectError:  false,
			waitDuration: 2 * time.Second, // CPU collection takes ~1s
		},
		{
			name:         "start disabled collector",
			enabled:      false,
			expectError:  false,
			waitDuration: 10 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			config := &CollectorConfig{
				Enabled:             tt.enabled,
				CollectionInterval:  50 * time.Millisecond,
				EnableSystemMetrics: true,
				EnableAppMetrics:    true,
			}
			logger := logger.NewTestLogger()
			collector := NewCollector(config, logger)

			// Act
			err := collector.Start()

			// Assert
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// Wait for collection to run
			if tt.enabled {
				// Poll for snapshot with timeout
				snapshot := waitForSnapshot(collector, tt.waitDuration)

				// Verify collection happened
				assert.NotNil(t, snapshot, "snapshot should not be nil after collection")
			}

			// Stop collector
			err = collector.Stop()
			assert.NoError(t, err)

			// Verify context is cancelled
			if tt.enabled {
				assert.NotNil(t, collector.ctx)
				select {
				case <-collector.ctx.Done():
					// Context properly cancelled
				case <-time.After(100 * time.Millisecond):
					t.Error("context was not cancelled after Stop")
				}
			}
		})
	}
}

// TestCollectorGetSnapshot tests retrieving metrics snapshots
func TestCollectorGetSnapshot(t *testing.T) {
	// Arrange
	config := &CollectorConfig{
		Enabled:             true,
		CollectionInterval:  100 * time.Millisecond,
		EnableSystemMetrics: true,
		EnableAppMetrics:    true,
		EnableGovMetrics:    true,
		EnablePerfMetrics:   true,
	}
	logger := logger.NewTestLogger()
	collector := NewCollector(config, logger)

	// Act - before collection
	snapshot1 := collector.GetSnapshot()
	assert.Nil(t, snapshot1, "snapshot should be nil before collection")

	// Start and wait for collection
	err := collector.Start()
	require.NoError(t, err)
	defer collector.Stop()

	// Act - after collection
	snapshot2 := waitForSnapshot(collector, 2*time.Second) // CPU collection takes ~1s

	// Assert
	assert.NotNil(t, snapshot2)
	assert.NotZero(t, snapshot2.Timestamp)
	assert.NotNil(t, snapshot2.System)
	assert.NotNil(t, snapshot2.Application)
	assert.NotNil(t, snapshot2.Governance)
	assert.NotNil(t, snapshot2.Performance)
}

// TestCollectorSystemMetrics tests system metrics collection
func TestCollectorSystemMetrics(t *testing.T) {
	// Arrange
	config := &CollectorConfig{
		Enabled:             true,
		CollectionInterval:  100 * time.Millisecond,
		EnableSystemMetrics: true,
		EnableAppMetrics:    false,
		EnableGovMetrics:    false,
		EnablePerfMetrics:   false,
	}
	logger := logger.NewTestLogger()
	collector := NewCollector(config, logger)

	// Act
	err := collector.Start()
	require.NoError(t, err)
	defer collector.Stop()

	snapshot := waitForSnapshot(collector, 2*time.Second) // CPU collection takes ~1s

	// Assert
	require.NotNil(t, snapshot)
	require.NotNil(t, snapshot.System)

	// System metrics should have reasonable values
	assert.GreaterOrEqual(t, snapshot.System.CPUUsage, 0.0)
	assert.LessOrEqual(t, snapshot.System.CPUUsage, 100.0)
	assert.GreaterOrEqual(t, snapshot.System.MemoryUsage, 0.0)
	assert.LessOrEqual(t, snapshot.System.MemoryUsage, 100.0)
	assert.Greater(t, snapshot.System.MemoryTotal, uint64(0))
	assert.Greater(t, snapshot.System.Goroutines, 0)
	assert.Greater(t, snapshot.System.Uptime, int64(0))
}

// TestCollectorCallbacks tests callback functions
func TestCollectorCallbacks(t *testing.T) {
	// Arrange
	config := &CollectorConfig{
		Enabled:             true,
		CollectionInterval:  100 * time.Millisecond,
		EnableSystemMetrics: false,
		EnableAppMetrics:    true,
		EnableGovMetrics:    false,
		EnablePerfMetrics:   false,
	}
	logger := logger.NewTestLogger()
	collector := NewCollector(config, logger)

	// Set up callbacks
	expectedHeight := int64(12345)
	expectedPeers := 42
	expectedSyncing := true

	collector.SetNodeHeightCallback(func() (int64, error) {
		return expectedHeight, nil
	})

	collector.SetNodePeersCallback(func() (int, error) {
		return expectedPeers, nil
	})

	collector.SetNodeSyncingCallback(func() (bool, error) {
		return expectedSyncing, nil
	})

	// Act
	err := collector.Start()
	require.NoError(t, err)
	defer collector.Stop()

	snapshot := waitForSnapshot(collector, 2*time.Second) // CPU collection takes ~1s

	// Assert
	require.NotNil(t, snapshot)
	require.NotNil(t, snapshot.Application)
	assert.Equal(t, expectedHeight, snapshot.Application.NodeHeight)
	assert.Equal(t, expectedPeers, snapshot.Application.NodePeers)
	assert.Equal(t, expectedSyncing, snapshot.Application.NodeSyncing)
}

// TestCollectorIncrementCounters tests counter increment methods
func TestCollectorIncrementCounters(t *testing.T) {
	// Arrange
	config := &CollectorConfig{
		Enabled:          true,
		EnableAppMetrics: true,
	}
	logger := logger.NewTestLogger()
	collector := NewCollector(config, logger)

	// Act
	collector.IncrementUpgradeTotal()
	collector.IncrementUpgradeSuccess()
	collector.IncrementUpgradeFailed()
	collector.IncrementProcessRestarts()

	// Assert - verify metrics were registered and can be collected
	metricFamilies, err := collector.registry.Gather()
	require.NoError(t, err)
	assert.NotEmpty(t, metricFamilies)

	// Check for expected metrics
	metricNames := make(map[string]bool)
	for _, mf := range metricFamilies {
		metricNames[mf.GetName()] = true
	}

	assert.True(t, metricNames["wemixvisor_upgrades_total"])
	assert.True(t, metricNames["wemixvisor_upgrades_success_total"])
	assert.True(t, metricNames["wemixvisor_upgrades_failed_total"])
	assert.True(t, metricNames["wemixvisor_process_restarts_total"])
}

// TestCollectorSetUpgradePending tests setting pending upgrades
func TestCollectorSetUpgradePending(t *testing.T) {
	tests := []struct {
		name  string
		count int
	}{
		{"zero pending", 0},
		{"one pending", 1},
		{"multiple pending", 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			config := &CollectorConfig{
				Enabled:          true,
				EnableAppMetrics: true,
			}
			logger := logger.NewTestLogger()
			collector := NewCollector(config, logger)

			// Act
			collector.SetUpgradePending(tt.count)

			// Assert - verify the metric was set
			metricFamilies, err := collector.registry.Gather()
			require.NoError(t, err)

			for _, mf := range metricFamilies {
				if mf.GetName() == "wemixvisor_upgrades_pending" {
					assert.Equal(t, float64(tt.count), mf.GetMetric()[0].GetGauge().GetValue())
					return
				}
			}
			t.Error("upgrades_pending metric not found")
		})
	}
}

// TestCollectorObserveLatency tests latency observation methods
func TestCollectorObserveLatency(t *testing.T) {
	tests := []struct {
		name      string
		rpcValues []float64
		apiValues []float64
	}{
		{
			name:      "single observation",
			rpcValues: []float64{10.5},
			apiValues: []float64{20.3},
		},
		{
			name:      "multiple observations",
			rpcValues: []float64{5.0, 10.0, 15.0, 20.0},
			apiValues: []float64{8.0, 12.0, 16.0, 24.0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			config := &CollectorConfig{
				Enabled:           true,
				EnablePerfMetrics: true,
			}
			logger := logger.NewTestLogger()
			collector := NewCollector(config, logger)

			// Act
			for _, val := range tt.rpcValues {
				collector.ObserveRPCLatency(val)
			}
			for _, val := range tt.apiValues {
				collector.ObserveAPILatency(val)
			}

			// Assert
			metricFamilies, err := collector.registry.Gather()
			require.NoError(t, err)

			foundRPC := false
			foundAPI := false

			for _, mf := range metricFamilies {
				switch mf.GetName() {
				case "wemixvisor_rpc_latency_milliseconds":
					foundRPC = true
					assert.Equal(t, uint64(len(tt.rpcValues)), mf.GetMetric()[0].GetHistogram().GetSampleCount())
				case "wemixvisor_api_latency_milliseconds":
					foundAPI = true
					assert.Equal(t, uint64(len(tt.apiValues)), mf.GetMetric()[0].GetHistogram().GetSampleCount())
				}
			}

			assert.True(t, foundRPC, "RPC latency metric not found")
			assert.True(t, foundAPI, "API latency metric not found")
		})
	}
}

// TestCollectorGetRegistry tests getting the Prometheus registry
func TestCollectorGetRegistry(t *testing.T) {
	// Arrange
	config := &CollectorConfig{
		Enabled: true,
	}
	logger := logger.NewTestLogger()
	collector := NewCollector(config, logger)

	// Act
	registry := collector.GetRegistry()

	// Assert
	assert.NotNil(t, registry)
	assert.IsType(t, &prometheus.Registry{}, registry)
	assert.Equal(t, collector.registry, registry)
}

// TestCollectorRecordError tests error recording
func TestCollectorRecordError(t *testing.T) {
	// Arrange
	config := &CollectorConfig{
		Enabled: true,
	}
	logger := logger.NewTestLogger()
	collector := NewCollector(config, logger)

	// Act & Assert - should not panic
	assert.NotPanics(t, func() {
		collector.RecordError("test_source", assert.AnError)
	})
}

// TestCollectorGenerateAlert tests alert generation
func TestCollectorGenerateAlert(t *testing.T) {
	// Arrange
	config := &CollectorConfig{
		Enabled: true,
	}
	logger := logger.NewTestLogger()
	collector := NewCollector(config, logger)

	alert := &Alert{
		Name:    "test_alert",
		Level:   "warning",
		Message: "Test alert message",
	}

	// Act & Assert - should not panic
	assert.NotPanics(t, func() {
		collector.GenerateAlert(alert)
	})
}

// TestCollectorConcurrency tests concurrent access to collector
func TestCollectorConcurrency(t *testing.T) {
	// Arrange
	config := &CollectorConfig{
		Enabled:             true,
		CollectionInterval:  50 * time.Millisecond,
		EnableSystemMetrics: true,
		EnableAppMetrics:    true,
	}
	logger := logger.NewTestLogger()
	collector := NewCollector(config, logger)

	err := collector.Start()
	require.NoError(t, err)
	defer collector.Stop()

	// Act - concurrent reads and writes
	done := make(chan bool)
	goroutines := 10

	for i := 0; i < goroutines; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				// Read operations
				_ = collector.GetSnapshot()
				_ = collector.GetRegistry()

				// Write operations
				collector.IncrementUpgradeTotal()
				collector.ObserveRPCLatency(float64(j))
				collector.SetUpgradePending(j)

				time.Sleep(1 * time.Millisecond)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < goroutines; i++ {
		<-done
	}
}

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

	// Assert - no race conditions, still functional
	snapshot := waitForSnapshot(collector, 2*time.Second) // CPU collection takes ~1s
	assert.NotNil(t, snapshot)
}

// TestCollectorMetricsRegistration tests selective metrics registration
func TestCollectorMetricsRegistration(t *testing.T) {
	tests := []struct {
		name                string
		enableSystem        bool
		enableApp           bool
		enableGov           bool
		enablePerf          bool
		expectedMetricCount int
	}{
		{
			name:                "all metrics enabled",
			enableSystem:        true,
			enableApp:           true,
			enableGov:           true,
			enablePerf:          true,
			expectedMetricCount: 26, // System(6) + App(9) + Gov(8) + Perf(3)
		},
		{
			name:                "only system metrics",
			enableSystem:        true,
			enableApp:           false,
			enableGov:           false,
			enablePerf:          false,
			expectedMetricCount: 6, // System metrics only
		},
		{
			name:                "no metrics enabled",
			enableSystem:        false,
			enableApp:           false,
			enableGov:           false,
			enablePerf:          false,
			expectedMetricCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			config := &CollectorConfig{
				Enabled:             true,
				EnableSystemMetrics: tt.enableSystem,
				EnableAppMetrics:    tt.enableApp,
				EnableGovMetrics:    tt.enableGov,
				EnablePerfMetrics:   tt.enablePerf,
			}
			logger := logger.NewTestLogger()
			collector := NewCollector(config, logger)

			// Act
			metricFamilies, err := collector.registry.Gather()

			// Assert
			require.NoError(t, err)
			assert.Equal(t, tt.expectedMetricCount, len(metricFamilies),
				"expected %d metrics but got %d", tt.expectedMetricCount, len(metricFamilies))
		})
	}
}

// TestCollectorContext tests context cancellation
func TestCollectorContext(t *testing.T) {
	// Arrange
	config := &CollectorConfig{
		Enabled:            true,
		CollectionInterval: 10 * time.Millisecond,
	}
	logger := logger.NewTestLogger()
	collector := NewCollector(config, logger)

	// Act
	err := collector.Start()
	require.NoError(t, err)

	// Verify context is active
	assert.NotNil(t, collector.ctx)
	assert.NotNil(t, collector.cancel)

	select {
	case <-collector.ctx.Done():
		t.Error("context should not be cancelled yet")
	default:
		// Context is still active, good
	}

	// Stop collector
	err = collector.Stop()
	require.NoError(t, err)

	// Verify context is cancelled
	select {
	case <-collector.ctx.Done():
		// Context properly cancelled
	case <-time.After(100 * time.Millisecond):
		t.Error("context should be cancelled after Stop")
	}

	// Verify ctx.Err() returns context.Canceled
	assert.Equal(t, context.Canceled, collector.ctx.Err())
}

// TestUpdatePrometheusNilMetrics tests nil metrics handling in update methods
func TestUpdatePrometheusNilMetrics(t *testing.T) {
	// Arrange
	config := &CollectorConfig{
		Enabled:             true,
		EnableSystemMetrics: true,
		EnableAppMetrics:    true,
		EnableGovMetrics:    true,
		EnablePerfMetrics:   true,
	}
	logger := logger.NewTestLogger()
	collector := NewCollector(config, logger)

	// Act & Assert - should not panic with nil metrics
	assert.NotPanics(t, func() {
		collector.updateSystemPrometheus(nil)
	})

	assert.NotPanics(t, func() {
		collector.updateApplicationPrometheus(nil)
	})

	assert.NotPanics(t, func() {
		collector.updateGovernancePrometheus(nil)
	})

	assert.NotPanics(t, func() {
		collector.updatePerformancePrometheus(nil)
	})
}

// TestCollectorGovernanceWithError tests governance metrics collection with error
func TestCollectorGovernanceWithError(t *testing.T) {
	// Arrange
	config := &CollectorConfig{
		Enabled:          true,
		EnableGovMetrics: true,
	}
	logger := logger.NewTestLogger()
	collector := NewCollector(config, logger)

	// Set callback that returns error
	collector.SetProposalStatsCallback(func() (*GovernanceMetrics, error) {
		return nil, assert.AnError
	})

	// Act
	metrics := collector.collectGovernanceMetrics()

	// Assert - should return empty metrics on error
	assert.NotNil(t, metrics)
	assert.Equal(t, int64(0), metrics.ProposalTotal)
	assert.Equal(t, int64(0), metrics.ProposalVoting)
}

// TestSetProposalStatsCallback tests SetProposalStatsCallback function
func TestSetProposalStatsCallback(t *testing.T) {
	// Arrange
	config := &CollectorConfig{
		Enabled:          true,
		EnableGovMetrics: true,
	}
	logger := logger.NewTestLogger()
	collector := NewCollector(config, logger)

	expectedMetrics := &GovernanceMetrics{
		Timestamp:        time.Now(),
		ProposalTotal:    100,
		ProposalVoting:   10,
		ProposalPassed:   80,
		ProposalRejected: 10,
		VotingPower:      1000000.0,
		VotingTurnout:    75.5,
		ValidatorActive:  50,
		ValidatorJailed:  5,
	}

	// Act - set callback
	collector.SetProposalStatsCallback(func() (*GovernanceMetrics, error) {
		return expectedMetrics, nil
	})

	// Verify callback is set and returns expected values
	metrics := collector.collectGovernanceMetrics()

	// Assert
	require.NotNil(t, metrics)
	assert.Equal(t, expectedMetrics.ProposalTotal, metrics.ProposalTotal)
	assert.Equal(t, expectedMetrics.ProposalVoting, metrics.ProposalVoting)
	assert.Equal(t, expectedMetrics.ProposalPassed, metrics.ProposalPassed)
	assert.Equal(t, expectedMetrics.ProposalRejected, metrics.ProposalRejected)
	assert.Equal(t, expectedMetrics.VotingPower, metrics.VotingPower)
	assert.Equal(t, expectedMetrics.VotingTurnout, metrics.VotingTurnout)
	assert.Equal(t, expectedMetrics.ValidatorActive, metrics.ValidatorActive)
	assert.Equal(t, expectedMetrics.ValidatorJailed, metrics.ValidatorJailed)
}

// Benchmark tests for performance validation
func BenchmarkCollectorCollection(b *testing.B) {
	config := &CollectorConfig{
		Enabled:             true,
		EnableSystemMetrics: true,
		EnableAppMetrics:    true,
		EnableGovMetrics:    true,
		EnablePerfMetrics:   true,
	}
	logger := logger.NewTestLogger()
	collector := NewCollector(config, logger)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.collect()
	}
}

func BenchmarkCollectorGetSnapshot(b *testing.B) {
	config := &CollectorConfig{
		Enabled: true,
	}
	logger := logger.NewTestLogger()
	collector := NewCollector(config, logger)
	collector.collect() // Ensure there's a snapshot

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = collector.GetSnapshot()
	}
}
