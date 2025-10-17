package metrics

import (
	"testing"
	"time"

	"github.com/wemix/wemixvisor/internal/config"
	"github.com/wemix/wemixvisor/pkg/logger"
)

// BenchmarkMetricsCollector_CollectMetrics benchmarks the metrics collection performance
func BenchmarkMetricsCollector_CollectMetrics(b *testing.B) {
	cfg := &config.Config{
		MetricsInterval: 10 * time.Second,
	}
	logger, _ := logger.New(true, false, "")
	nodeInfo := &mockNodeInfoProvider{
		uptime:       2 * time.Hour,
		restartCount: 5,
		healthy:      true,
		pid:          1234,
	}

	collector := NewMetricsCollector(cfg, logger, nodeInfo)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.collectMetrics()
	}
}

// BenchmarkMetricsCollector_GetMetrics benchmarks the metrics retrieval performance
func BenchmarkMetricsCollector_GetMetrics(b *testing.B) {
	cfg := &config.Config{}
	logger, _ := logger.New(true, false, "")
	nodeInfo := &mockNodeInfoProvider{
		uptime:       2 * time.Hour,
		restartCount: 5,
		healthy:      true,
		pid:          1234,
	}

	collector := NewMetricsCollector(cfg, logger, nodeInfo)
	collector.collectMetrics() // Initialize metrics

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = collector.GetMetrics()
	}
}

// BenchmarkMetricsCollector_ExportJSON benchmarks JSON export performance
func BenchmarkMetricsCollector_ExportJSON(b *testing.B) {
	cfg := &config.Config{}
	logger, _ := logger.New(true, false, "")
	nodeInfo := &mockNodeInfoProvider{
		uptime:       2 * time.Hour,
		restartCount: 5,
		healthy:      true,
		pid:          1234,
	}

	collector := NewMetricsCollector(cfg, logger, nodeInfo)
	collector.collectMetrics() // Initialize metrics

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := collector.ExportMetrics("json")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMetricsCollector_ExportPrometheus benchmarks Prometheus export performance
func BenchmarkMetricsCollector_ExportPrometheus(b *testing.B) {
	cfg := &config.Config{}
	logger, _ := logger.New(true, false, "")
	nodeInfo := &mockNodeInfoProvider{
		uptime:       2 * time.Hour,
		restartCount: 5,
		healthy:      true,
		pid:          1234,
	}

	collector := NewMetricsCollector(cfg, logger, nodeInfo)
	collector.collectMetrics() // Initialize metrics

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := collector.ExportMetrics("prometheus")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMetricsCollector_ConcurrentAccess benchmarks concurrent access to metrics
func BenchmarkMetricsCollector_ConcurrentAccess(b *testing.B) {
	cfg := &config.Config{}
	logger, _ := logger.New(true, false, "")
	nodeInfo := &mockNodeInfoProvider{
		uptime:       2 * time.Hour,
		restartCount: 5,
		healthy:      true,
		pid:          1234,
	}

	collector := NewMetricsCollector(cfg, logger, nodeInfo)
	collector.collectMetrics() // Initialize metrics

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = collector.GetMetrics()
		}
	})
}

