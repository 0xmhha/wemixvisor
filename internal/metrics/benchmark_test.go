package metrics

import (
	"testing"
	"time"

	"github.com/wemix/wemixvisor/pkg/logger"
)

// BenchmarkCollector_Collect benchmarks the metrics collection performance
func BenchmarkCollector_Collect(b *testing.B) {
	config := &CollectorConfig{
		Enabled:             true,
		CollectionInterval:  10 * time.Second,
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

// BenchmarkCollector_GetSnapshot benchmarks the metrics retrieval performance
func BenchmarkCollector_GetSnapshot(b *testing.B) {
	config := &CollectorConfig{
		Enabled:             true,
		CollectionInterval:  10 * time.Second,
		EnableSystemMetrics: true,
		EnableAppMetrics:    true,
		EnableGovMetrics:    true,
		EnablePerfMetrics:   true,
	}
	logger := logger.NewTestLogger()
	collector := NewCollector(config, logger)

	// Initialize metrics
	collector.collect()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = collector.GetSnapshot()
	}
}

// BenchmarkCollector_PrometheusGather benchmarks Prometheus registry gathering
func BenchmarkCollector_PrometheusGather(b *testing.B) {
	config := &CollectorConfig{
		Enabled:             true,
		CollectionInterval:  10 * time.Second,
		EnableSystemMetrics: true,
		EnableAppMetrics:    true,
		EnableGovMetrics:    true,
		EnablePerfMetrics:   true,
	}
	logger := logger.NewTestLogger()
	collector := NewCollector(config, logger)

	// Initialize metrics
	collector.collect()
	registry := collector.GetRegistry()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := registry.Gather()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCollector_MetricUpdates benchmarks metric update operations
func BenchmarkCollector_MetricUpdates(b *testing.B) {
	config := &CollectorConfig{
		Enabled:             true,
		CollectionInterval:  10 * time.Second,
		EnableSystemMetrics: true,
		EnableAppMetrics:    true,
		EnableGovMetrics:    true,
		EnablePerfMetrics:   true,
	}
	logger := logger.NewTestLogger()
	collector := NewCollector(config, logger)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.IncrementUpgradeTotal()
		collector.SetUpgradePending(i % 10)
		collector.ObserveRPCLatency(float64(i % 100))
		collector.ObserveAPILatency(float64(i % 50))
	}
}

// BenchmarkCollector_ConcurrentAccess benchmarks concurrent access to metrics
func BenchmarkCollector_ConcurrentAccess(b *testing.B) {
	config := &CollectorConfig{
		Enabled:             true,
		CollectionInterval:  10 * time.Second,
		EnableSystemMetrics: true,
		EnableAppMetrics:    true,
		EnableGovMetrics:    true,
		EnablePerfMetrics:   true,
	}
	logger := logger.NewTestLogger()
	collector := NewCollector(config, logger)

	// Initialize metrics
	collector.collect()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = collector.GetSnapshot()
		}
	})
}
