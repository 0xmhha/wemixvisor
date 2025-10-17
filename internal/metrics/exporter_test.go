package metrics

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wemix/wemixvisor/pkg/logger"
)

// waitForServer polls until server is ready or timeout
func waitForServer(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 1 * time.Second}

	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			resp.Body.Close()
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("server not ready after %v", timeout)
}

// TestNewExporter tests the creation of a new exporter
func TestNewExporter(t *testing.T) {
	tests := []struct {
		name         string
		port         int
		path         string
		expectedPort int
		expectedPath string
	}{
		{
			name:         "default values",
			port:         0,
			path:         "",
			expectedPort: 9090,
			expectedPath: "/metrics",
		},
		{
			name:         "custom values",
			port:         8080,
			path:         "/prometheus",
			expectedPort: 8080,
			expectedPath: "/prometheus",
		},
		{
			name:         "custom port only",
			port:         9091,
			path:         "",
			expectedPort: 9091,
			expectedPath: "/metrics",
		},
		{
			name:         "custom path only",
			port:         0,
			path:         "/custom",
			expectedPort: 9090,
			expectedPath: "/custom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			config := &CollectorConfig{
				Enabled:             true,
				CollectionInterval:  1 * time.Second,
				EnableSystemMetrics: true,
			}
			logger := logger.NewTestLogger()
			collector := NewCollector(config, logger)

			// Act
			exporter := NewExporter(collector, tt.port, tt.path, logger)

			// Assert
			assert.NotNil(t, exporter)
			assert.Equal(t, tt.expectedPort, exporter.port)
			assert.Equal(t, tt.expectedPath, exporter.path)
			assert.NotNil(t, exporter.collector)
			assert.NotNil(t, exporter.logger)
			assert.Nil(t, exporter.server, "server should not be initialized until Start() is called")
		})
	}
}

// TestExporterStartStop tests starting and stopping the exporter
func TestExporterStartStop(t *testing.T) {
	// Arrange
	config := &CollectorConfig{
		Enabled:             true,
		CollectionInterval:  1 * time.Second,
		EnableSystemMetrics: true,
	}
	logger := logger.NewTestLogger()
	collector := NewCollector(config, logger)
	exporter := NewExporter(collector, 19090, "/metrics", logger)

	// Act - Start
	err := exporter.Start()
	require.NoError(t, err)
	assert.NotNil(t, exporter.server, "server should be initialized after Start()")

	// Wait for server to start
	err = waitForServer("http://localhost:19090/health", 3*time.Second)
	require.NoError(t, err, "server should be accessible after Start()")

	// Act - Stop
	err = exporter.Stop()

	// Assert
	assert.NoError(t, err)
}

// TestExporterMultipleStop tests calling Stop multiple times
func TestExporterMultipleStop(t *testing.T) {
	// Arrange
	config := &CollectorConfig{
		Enabled:             true,
		CollectionInterval:  1 * time.Second,
		EnableSystemMetrics: true,
	}
	logger := logger.NewTestLogger()
	collector := NewCollector(config, logger)
	exporter := NewExporter(collector, 19091, "/metrics", logger)

	// Act & Assert - Stop before Start
	err := exporter.Stop()
	assert.NoError(t, err, "Stop before Start should not error")

	// Start server
	err = exporter.Start()
	require.NoError(t, err)
	err = waitForServer("http://localhost:19091/health", 3*time.Second)
	require.NoError(t, err)

	// Act & Assert - Multiple Stop calls
	err = exporter.Stop()
	assert.NoError(t, err, "first Stop should succeed")

	err = exporter.Stop()
	assert.NoError(t, err, "second Stop should not error")
}

// TestHealthHandler tests the health check endpoint
func TestHealthHandler(t *testing.T) {
	// Arrange
	config := &CollectorConfig{
		Enabled:             true,
		CollectionInterval:  1 * time.Second,
		EnableSystemMetrics: true,
	}
	logger := logger.NewTestLogger()
	collector := NewCollector(config, logger)
	exporter := NewExporter(collector, 19092, "/metrics", logger)

	err := exporter.Start()
	require.NoError(t, err)
	defer exporter.Stop()

	err = waitForServer("http://localhost:19092/health", 3*time.Second)
	require.NoError(t, err)

	// Act
	resp, err := http.Get("http://localhost:19092/health")
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	// Assert
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	assert.JSONEq(t, `{"status":"healthy"}`, string(body))
}

// TestReadyHandler tests the readiness check endpoint
func TestReadyHandler(t *testing.T) {
	tests := []struct {
		name               string
		startCollector     bool
		waitForSnapshot    bool
		expectedStatus     int
		expectedBodySubstr string
	}{
		{
			name:               "not ready - no snapshot",
			startCollector:     false,
			waitForSnapshot:    false,
			expectedStatus:     http.StatusServiceUnavailable,
			expectedBodySubstr: `"status":"not_ready"`,
		},
		{
			name:               "ready - snapshot available",
			startCollector:     true,
			waitForSnapshot:    true,
			expectedStatus:     http.StatusOK,
			expectedBodySubstr: `"status":"ready"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			config := &CollectorConfig{
				Enabled:             true,
				CollectionInterval:  50 * time.Millisecond,
				EnableSystemMetrics: true,
			}
			logger := logger.NewTestLogger()
			collector := NewCollector(config, logger)

			// Use unique port for each test
			port := 19093
			if tt.name == "ready - snapshot available" {
				port = 19094
			}

			exporter := NewExporter(collector, port, "/metrics", logger)

			if tt.startCollector {
				err := collector.Start()
				require.NoError(t, err)
				defer collector.Stop()

				if tt.waitForSnapshot {
					// Wait for at least one collection cycle
					time.Sleep(2 * time.Second) // CPU collection takes ~1s
				}
			}

			err := exporter.Start()
			require.NoError(t, err)
			defer exporter.Stop()

			err = waitForServer(fmt.Sprintf("http://localhost:%d/health", port), 3*time.Second)
			require.NoError(t, err)

			// Act
			resp, err := http.Get(fmt.Sprintf("http://localhost:%d/ready", port))
			require.NoError(t, err)
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			// Assert
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
			assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
			assert.Contains(t, string(body), tt.expectedBodySubstr)
		})
	}
}

// TestMetricsInfoHandler tests the metrics info endpoint
func TestMetricsInfoHandler(t *testing.T) {
	// Arrange
	config := &CollectorConfig{
		Enabled:             true,
		CollectionInterval:  1 * time.Second,
		EnableSystemMetrics: true,
		EnableAppMetrics:    true,
		EnableGovMetrics:    false,
		EnablePerfMetrics:   false,
	}
	logger := logger.NewTestLogger()
	collector := NewCollector(config, logger)
	exporter := NewExporter(collector, 19095, "/custom", logger)

	err := exporter.Start()
	require.NoError(t, err)
	defer exporter.Stop()

	err = waitForServer("http://localhost:19095/health", 3*time.Second)
	require.NoError(t, err)

	// Act
	resp, err := http.Get("http://localhost:19095/api/metrics/info")
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	// Assert
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	bodyStr := string(body)
	assert.Contains(t, bodyStr, `"version": "0.7.0"`)
	assert.Contains(t, bodyStr, `"port": 19095`)
	assert.Contains(t, bodyStr, `"path": "/custom"`)
	assert.Contains(t, bodyStr, `"enabled": true`)
	assert.Contains(t, bodyStr, `"system_metrics": true`)
	assert.Contains(t, bodyStr, `"app_metrics": true`)
	assert.Contains(t, bodyStr, `"gov_metrics": false`)
	assert.Contains(t, bodyStr, `"perf_metrics": false`)
}

// TestMetricsEndpoint tests the Prometheus metrics endpoint
func TestMetricsEndpoint(t *testing.T) {
	// Arrange
	config := &CollectorConfig{
		Enabled:             true,
		CollectionInterval:  100 * time.Millisecond,
		EnableSystemMetrics: true,
		EnableAppMetrics:    true,
	}
	logger := logger.NewTestLogger()
	collector := NewCollector(config, logger)

	// Start collector to generate metrics
	err := collector.Start()
	require.NoError(t, err)
	defer collector.Stop()

	// Increment some counters
	collector.IncrementUpgradeTotal()
	collector.IncrementUpgradeSuccess()
	collector.SetUpgradePending(3)

	exporter := NewExporter(collector, 19096, "/metrics", logger)
	err = exporter.Start()
	require.NoError(t, err)
	defer exporter.Stop()

	err = waitForServer("http://localhost:19096/health", 3*time.Second)
	require.NoError(t, err)

	// Wait for collection
	time.Sleep(2 * time.Second) // CPU collection takes ~1s

	// Act
	resp, err := http.Get("http://localhost:19096/metrics")
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	// Assert
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	bodyStr := string(body)
	// Check for some expected metrics
	assert.Contains(t, bodyStr, "wemixvisor_cpu_usage_percent")
	assert.Contains(t, bodyStr, "wemixvisor_memory_usage_percent")
	assert.Contains(t, bodyStr, "wemixvisor_upgrades_total")
	assert.Contains(t, bodyStr, "wemixvisor_upgrades_success_total")
	assert.Contains(t, bodyStr, "wemixvisor_upgrades_pending")
}

// TestGetURL tests the GetURL helper function
func TestGetURL(t *testing.T) {
	tests := []struct {
		name        string
		port        int
		path        string
		expectedURL string
	}{
		{
			name:        "default values",
			port:        0, // Will default to 9090
			path:        "", // Will default to /metrics
			expectedURL: "http://localhost:9090/metrics",
		},
		{
			name:        "custom values",
			port:        8080,
			path:        "/prometheus",
			expectedURL: "http://localhost:8080/prometheus",
		},
		{
			name:        "custom port with default path",
			port:        9091,
			path:        "",
			expectedURL: "http://localhost:9091/metrics",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			config := &CollectorConfig{
				Enabled:             true,
				CollectionInterval:  1 * time.Second,
				EnableSystemMetrics: true,
			}
			logger := logger.NewTestLogger()
			collector := NewCollector(config, logger)
			exporter := NewExporter(collector, tt.port, tt.path, logger)

			// Act
			url := exporter.GetURL()

			// Assert
			assert.Equal(t, tt.expectedURL, url)
		})
	}
}

// TestExporterConcurrentRequests tests concurrent HTTP requests
func TestExporterConcurrentRequests(t *testing.T) {
	// Arrange
	config := &CollectorConfig{
		Enabled:             true,
		CollectionInterval:  100 * time.Millisecond,
		EnableSystemMetrics: true,
		EnableAppMetrics:    true,
	}
	logger := logger.NewTestLogger()
	collector := NewCollector(config, logger)

	err := collector.Start()
	require.NoError(t, err)
	defer collector.Stop()

	exporter := NewExporter(collector, 19097, "/metrics", logger)
	err = exporter.Start()
	require.NoError(t, err)
	defer exporter.Stop()

	err = waitForServer("http://localhost:19097/health", 3*time.Second)
	require.NoError(t, err)

	// Wait for initial collection
	time.Sleep(2 * time.Second)

	// Act - concurrent requests
	done := make(chan bool)
	errors := make(chan error, 30)
	goroutines := 10

	endpoints := []string{
		"http://localhost:19097/health",
		"http://localhost:19097/ready",
		"http://localhost:19097/metrics",
	}

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			client := &http.Client{Timeout: 5 * time.Second}
			for j := 0; j < 10; j++ {
				endpoint := endpoints[j%len(endpoints)]
				resp, err := client.Get(endpoint)
				if err != nil {
					errors <- fmt.Errorf("request %d-%d failed: %w", id, j, err)
					continue
				}
				resp.Body.Close()

				if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusServiceUnavailable {
					errors <- fmt.Errorf("unexpected status %d from %s", resp.StatusCode, endpoint)
				}
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < goroutines; i++ {
		<-done
	}
	close(errors)

	// Assert - no errors
	errorList := []error{}
	for err := range errors {
		errorList = append(errorList, err)
	}
	assert.Empty(t, errorList, "should handle concurrent requests without errors")
}

// TestExporterIntegration tests the complete exporter workflow
func TestExporterIntegration(t *testing.T) {
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

	// Start collector
	err := collector.Start()
	require.NoError(t, err)
	defer collector.Stop()

	// Set callbacks
	collector.SetNodeHeightCallback(func() (int64, error) {
		return 12345, nil
	})

	// Create and start exporter
	exporter := NewExporter(collector, 19098, "/metrics", logger)
	err = exporter.Start()
	require.NoError(t, err)

	err = waitForServer("http://localhost:19098/health", 3*time.Second)
	require.NoError(t, err)

	// Wait for collection
	time.Sleep(2 * time.Second)

	// Test all endpoints
	client := &http.Client{Timeout: 5 * time.Second}

	tests := []struct {
		name           string
		endpoint       string
		expectedStatus int
		bodyCheck      func(string) bool
	}{
		{
			name:           "health endpoint",
			endpoint:       "http://localhost:19098/health",
			expectedStatus: http.StatusOK,
			bodyCheck: func(body string) bool {
				return strings.Contains(body, `"status":"healthy"`)
			},
		},
		{
			name:           "ready endpoint",
			endpoint:       "http://localhost:19098/ready",
			expectedStatus: http.StatusOK,
			bodyCheck: func(body string) bool {
				return strings.Contains(body, `"status":"ready"`)
			},
		},
		{
			name:           "metrics info endpoint",
			endpoint:       "http://localhost:19098/api/metrics/info",
			expectedStatus: http.StatusOK,
			bodyCheck: func(body string) bool {
				return strings.Contains(body, `"version": "0.7.0"`) &&
					strings.Contains(body, `"port": 19098`)
			},
		},
		{
			name:           "metrics endpoint",
			endpoint:       "http://localhost:19098/metrics",
			expectedStatus: http.StatusOK,
			bodyCheck: func(body string) bool {
				return strings.Contains(body, "wemixvisor_cpu_usage_percent") &&
					strings.Contains(body, "wemixvisor_node_height")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := client.Get(tt.endpoint)
			require.NoError(t, err)
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
			assert.True(t, tt.bodyCheck(string(body)), "body check failed for %s", tt.endpoint)
		})
	}

	// Stop exporter
	err = exporter.Stop()
	assert.NoError(t, err)

	// Verify server is stopped
	time.Sleep(500 * time.Millisecond)
	_, err = client.Get("http://localhost:19098/health")
	assert.Error(t, err, "server should not respond after Stop()")
}

// TestExporterStopContext tests graceful shutdown with context timeout
func TestExporterStopContext(t *testing.T) {
	// Arrange
	config := &CollectorConfig{
		Enabled:             true,
		CollectionInterval:  1 * time.Second,
		EnableSystemMetrics: true,
	}
	logger := logger.NewTestLogger()
	collector := NewCollector(config, logger)
	exporter := NewExporter(collector, 19099, "/metrics", logger)

	err := exporter.Start()
	require.NoError(t, err)

	err = waitForServer("http://localhost:19099/health", 3*time.Second)
	require.NoError(t, err)

	// Act - Stop should complete within timeout
	start := time.Now()
	err = exporter.Stop()
	elapsed := time.Since(start)

	// Assert
	assert.NoError(t, err)
	assert.Less(t, elapsed, 11*time.Second, "Stop should complete within context timeout (10s)")
}

// Benchmark tests for performance validation
func BenchmarkHealthHandler(b *testing.B) {
	config := &CollectorConfig{
		Enabled:             true,
		CollectionInterval:  1 * time.Second,
		EnableSystemMetrics: true,
	}
	logger := logger.NewTestLogger()
	collector := NewCollector(config, logger)
	exporter := NewExporter(collector, 29090, "/metrics", logger)

	err := exporter.Start()
	if err != nil {
		b.Fatal(err)
	}
	defer exporter.Stop()

	err = waitForServer("http://localhost:29090/health", 3*time.Second)
	if err != nil {
		b.Fatal(err)
	}

	client := &http.Client{Timeout: 5 * time.Second}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := client.Get("http://localhost:29090/health")
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}
