package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wemix/wemixvisor/internal/config"
	"github.com/wemix/wemixvisor/internal/governance"
	"github.com/wemix/wemixvisor/internal/metrics"
	"github.com/wemix/wemixvisor/pkg/logger"
)

// setupTestServer creates a test server with minimal dependencies
func setupTestServer(t *testing.T, includeMonitor, includeCollector bool) *Server {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		Home:       "/tmp/wemixvisor-test",
		RPCAddress: "http://localhost:26657",
		Debug:      false,
		APIPort:    8080,
	}

	testLogger := logger.NewTestLogger()

	var monitor *governance.Monitor
	var collector *metrics.Collector

	if includeMonitor {
		monitor = governance.NewMonitor(cfg, testLogger)
	}

	if includeCollector {
		collectorConfig := &metrics.CollectorConfig{
			Enabled:             true,
			CollectionInterval:  1 * time.Second,
			EnableSystemMetrics: true,
		}
		collector = metrics.NewCollector(collectorConfig, testLogger)
		if err := collector.Start(); err != nil {
			t.Fatalf("failed to start collector: %v", err)
		}
		// Wait for first snapshot
		time.Sleep(100 * time.Millisecond)
	}

	server := NewServer(cfg, monitor, collector, testLogger)
	return server
}

// TestNewServer tests server initialization
func TestNewServer(t *testing.T) {
	// Arrange & Act
	server := setupTestServer(t, true, true)

	// Assert
	assert.NotNil(t, server)
	assert.NotNil(t, server.router)
	assert.NotNil(t, server.logger)
	assert.NotNil(t, server.config)
	assert.NotNil(t, server.monitor)
	assert.NotNil(t, server.collector)
	assert.Equal(t, 8080, server.port)
}

// TestHealthHandler tests the health check endpoint
func TestHealthHandler(t *testing.T) {
	// Arrange
	server := setupTestServer(t, false, false)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)

	// Act
	server.router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "healthy", response["status"])
	assert.NotNil(t, response["timestamp"])
}

// TestReadyHandler tests the readiness check endpoint
func TestReadyHandler(t *testing.T) {
	tests := []struct {
		name           string
		includeMonitor bool
		expectedStatus int
		expectedReady  bool
	}{
		{
			name:           "ready with monitor",
			includeMonitor: true,
			expectedStatus: http.StatusOK,
			expectedReady:  true,
		},
		{
			name:           "not ready without monitor",
			includeMonitor: false,
			expectedStatus: http.StatusServiceUnavailable,
			expectedReady:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			server := setupTestServer(t, tt.includeMonitor, false)
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/ready", nil)

			// Act
			server.router.ServeHTTP(w, req)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			if tt.expectedReady {
				assert.Equal(t, "ready", response["status"])
			} else {
				assert.Equal(t, "not_ready", response["status"])
			}
		})
	}
}

// TestGetStatus tests the status endpoint
func TestGetStatus(t *testing.T) {
	// Arrange
	server := setupTestServer(t, true, true)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/status", nil)

	// Act
	server.router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "running", response["status"])
	assert.Equal(t, "0.7.0", response["version"])
	assert.NotNil(t, response["governance"])
	assert.NotNil(t, response["metrics"])
}

// TestGetVersion tests the version endpoint
func TestGetVersion(t *testing.T) {
	// Arrange
	server := setupTestServer(t, false, false)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/version", nil)

	// Act
	server.router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "0.7.0", response["version"])
	assert.Equal(t, "v1", response["api"])
	assert.NotNil(t, response["build_time"])
	assert.NotNil(t, response["git_commit"])
}

// TestGetMetrics tests the metrics endpoint
func TestGetMetrics(t *testing.T) {
	tests := []struct {
		name              string
		includeCollector  bool
		expectedStatus    int
		shouldHaveMetrics bool
	}{
		{
			name:              "with collector",
			includeCollector:  true,
			expectedStatus:    http.StatusOK,
			shouldHaveMetrics: true,
		},
		{
			name:              "without collector",
			includeCollector:  false,
			expectedStatus:    http.StatusServiceUnavailable,
			shouldHaveMetrics: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			server := setupTestServer(t, false, tt.includeCollector)
			if tt.includeCollector {
				defer server.collector.Stop()
				// Wait longer for metrics to be collected
				time.Sleep(1200 * time.Millisecond)
			}

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/v1/metrics", nil)

			// Act
			server.router.ServeHTTP(w, req)

			// Assert
			// Accept both 200 (metrics available) and 204 (no metrics yet)
			if tt.shouldHaveMetrics {
				assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusNoContent)

				// Only parse JSON if we got a 200 response
				if w.Code == http.StatusOK && w.Body.Len() > 0 {
					var response map[string]interface{}
					err := json.Unmarshal(w.Body.Bytes(), &response)
					require.NoError(t, err)
					assert.True(t, response["system"] != nil || response["message"] != nil)
				}
			} else {
				assert.Equal(t, tt.expectedStatus, w.Code)
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.NotNil(t, response["error"])
			}
		})
	}
}

// TestGetMetricsSnapshot tests the metrics snapshot endpoint
func TestGetMetricsSnapshot(t *testing.T) {
	// Arrange
	server := setupTestServer(t, false, true)
	defer server.collector.Stop()

	// Wait for metrics to be collected
	time.Sleep(150 * time.Millisecond)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/metrics/snapshot", nil)

	// Act
	server.router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Should have metrics data or nil
	assert.True(t, response["system"] != nil || response == nil)
}

// TestGetUpgrades tests the upgrades list endpoint
func TestGetUpgrades(t *testing.T) {
	// Test without monitor (service unavailable)
	t.Run("without monitor", func(t *testing.T) {
		// Arrange
		server := setupTestServer(t, false, false)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/upgrades", nil)

		// Act
		server.router.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusServiceUnavailable, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.NotNil(t, response["error"])
	})
}

// TestGetUpgrade tests getting a specific upgrade
func TestGetUpgrade(t *testing.T) {
	// Arrange
	server := setupTestServer(t, true, false)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/upgrades/test-upgrade-1", nil)

	// Act
	server.router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "test-upgrade-1", response["id"])
	assert.NotNil(t, response["status"])
}

// TestScheduleUpgrade tests scheduling a new upgrade
func TestScheduleUpgrade(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    string
		expectedStatus int
		shouldSucceed  bool
	}{
		{
			name:           "valid upgrade request",
			requestBody:    `{"name":"test-upgrade","height":100000,"info":"Test upgrade"}`,
			expectedStatus: http.StatusCreated,
			shouldSucceed:  true,
		},
		{
			name:           "missing required fields",
			requestBody:    `{"name":"test-upgrade"}`,
			expectedStatus: http.StatusBadRequest,
			shouldSucceed:  false,
		},
		{
			name:           "invalid JSON",
			requestBody:    `{invalid json}`,
			expectedStatus: http.StatusBadRequest,
			shouldSucceed:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			server := setupTestServer(t, true, false)
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/v1/upgrades", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")

			// Act
			server.router.ServeHTTP(w, req)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			if tt.shouldSucceed {
				assert.NotNil(t, response["message"])
				assert.NotNil(t, response["upgrade"])
			} else {
				assert.NotNil(t, response["error"])
			}
		})
	}
}

// TestCancelUpgrade tests canceling an upgrade
func TestCancelUpgrade(t *testing.T) {
	// Arrange
	server := setupTestServer(t, true, false)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/upgrades/test-upgrade-1", nil)

	// Act
	server.router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotNil(t, response["message"])
	assert.Contains(t, response["message"], "test-upgrade-1")
}

// TestGetProposals tests getting governance proposals
func TestGetProposals(t *testing.T) {
	// Test without monitor (service unavailable)
	t.Run("without monitor", func(t *testing.T) {
		// Arrange
		server := setupTestServer(t, false, false)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/governance/proposals", nil)

		// Act
		server.router.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusServiceUnavailable, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.NotNil(t, response["error"])
	})
}

// TestGetConfig tests getting configuration
func TestGetConfig(t *testing.T) {
	// Arrange
	server := setupTestServer(t, false, false)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/config", nil)

	// Act
	server.router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotNil(t, response["home"])
	assert.NotNil(t, response["rpc_address"])
	assert.NotNil(t, response["debug"])
}

// TestUpdateConfig tests updating configuration
func TestUpdateConfig(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    string
		expectedStatus int
	}{
		{
			name:           "valid config update",
			requestBody:    `{"debug":true,"poll_interval":"5s"}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid JSON",
			requestBody:    `{invalid}`,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			server := setupTestServer(t, false, false)
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("PUT", "/api/v1/config", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")

			// Act
			server.router.ServeHTTP(w, req)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// TestGetLogs tests getting logs
func TestGetLogs(t *testing.T) {
	// Arrange
	server := setupTestServer(t, false, false)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/logs?limit=50&level=error", nil)

	// Act
	server.router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotNil(t, response["logs"])
	assert.Equal(t, "50", response["limit"])
}

// TestServerStartStop tests server lifecycle
func TestServerStartStop(t *testing.T) {
	// Arrange
	server := setupTestServer(t, false, false)

	// Act - Start
	err := server.Start()

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, server.server)

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Act - Stop
	err = server.Stop()

	// Assert
	assert.NoError(t, err)
}

// TestGinLogger tests the Gin logging middleware
func TestGinLogger(t *testing.T) {
	// Arrange
	testLogger := logger.NewTestLogger()
	middleware := ginLogger(testLogger)

	router := gin.New()
	router.Use(middleware)
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "test"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test?param=value", nil)

	// Act
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
}
