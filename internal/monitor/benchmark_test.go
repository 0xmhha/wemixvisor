package monitor

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/wemix/wemixvisor/internal/config"
	"github.com/wemix/wemixvisor/pkg/logger"
)

// BenchmarkHealthChecker_ProcessCheck benchmarks process check performance
func BenchmarkHealthChecker_ProcessCheck(b *testing.B) {
	tmpDir := b.TempDir()
	pidFile := tmpDir + "/test.pid"

	check := &ProcessCheck{pidFile: pidFile}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = check.Check(ctx)
	}
}

// BenchmarkHealthChecker_RPCCheck benchmarks RPC health check performance
func BenchmarkHealthChecker_RPCCheck(b *testing.B) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"OK"}`))
	}))
	defer server.Close()

	check := &RPCHealthCheck{url: server.URL}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = check.Check(ctx)
	}
}

// BenchmarkHealthChecker_MemoryCheck benchmarks memory check performance
func BenchmarkHealthChecker_MemoryCheck(b *testing.B) {
	check := &MemoryCheck{maxMemoryMB: 1000}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = check.Check(ctx)
	}
}

// BenchmarkHealthChecker_DiskSpaceCheck benchmarks disk space check performance
func BenchmarkHealthChecker_DiskSpaceCheck(b *testing.B) {
	tmpDir := b.TempDir()
	check := &DiskSpaceCheck{minSpaceGB: 10, dataDir: tmpDir}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = check.Check(ctx)
	}
}

// BenchmarkHealthChecker_FullHealthCheck benchmarks complete health check cycle
func BenchmarkHealthChecker_FullHealthCheck(b *testing.B) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"OK"}`))
	}))
	defer server.Close()

	tmpDir := b.TempDir()

	// Create individual checks for benchmarking
	checks := []HealthCheck{
		&RPCHealthCheck{url: server.URL},
		&MemoryCheck{maxMemoryMB: 1000},
		&DiskSpaceCheck{minSpaceGB: 1, dataDir: tmpDir},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Benchmark individual checks instead of private runChecks method
		for _, check := range checks {
			check.Check(context.Background())
		}
	}
}

// BenchmarkHealthChecker_ConcurrentChecks benchmarks concurrent health checking
func BenchmarkHealthChecker_ConcurrentChecks(b *testing.B) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"OK"}`))
	}))
	defer server.Close()

	tmpDir := b.TempDir()

	// Create individual checks for benchmarking
	checks := []HealthCheck{
		&RPCHealthCheck{url: server.URL},
		&MemoryCheck{maxMemoryMB: 1000},
		&DiskSpaceCheck{minSpaceGB: 1, dataDir: tmpDir},
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Benchmark individual checks instead of private runChecks method
			for _, check := range checks {
				check.Check(context.Background())
			}
		}
	})
}

// BenchmarkHealthChecker_GetStatus benchmarks status retrieval performance
func BenchmarkHealthChecker_GetStatus(b *testing.B) {
	cfg := &config.Config{
		HealthCheckInterval: 1 * time.Second,
	}
	logger, _ := logger.New(true, false, "")

	checker := NewHealthChecker(cfg, logger)

	// Initialize with a status - start the checker to populate status
	statusChan := checker.Start()
	defer checker.Stop()
	time.Sleep(100 * time.Millisecond) // Allow time for initial status

	// Drain initial status
	select {
	case <-statusChan:
	default:
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = checker.GetStatus()
	}
}

// BenchmarkHealthChecker_IsHealthy benchmarks health status check performance
func BenchmarkHealthChecker_IsHealthy(b *testing.B) {
	cfg := &config.Config{
		HealthCheckInterval: 1 * time.Second,
	}
	logger, _ := logger.New(true, false, "")

	checker := NewHealthChecker(cfg, logger)

	// Initialize with a status - start the checker to populate status
	statusChan := checker.Start()
	defer checker.Stop()
	time.Sleep(100 * time.Millisecond) // Allow time for initial status

	// Drain initial status
	select {
	case <-statusChan:
	default:
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = checker.IsHealthy()
	}
}