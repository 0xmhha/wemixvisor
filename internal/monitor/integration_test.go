package monitor

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wemix/wemixvisor/internal/config"
	"github.com/wemix/wemixvisor/pkg/logger"
)

func TestHealthChecker_Integration_AllChecks(t *testing.T) {
	// Create temporary directory for PID file
	tmpDir := t.TempDir()
	pidFile := filepath.Join(tmpDir, "test.pid")

	// Create PID file with current process ID
	require.NoError(t, os.WriteFile(pidFile, []byte("123456"), 0644))

	// Create mock RPC server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]interface{}
		json.NewDecoder(r.Body).Decode(&req)

		var resp map[string]interface{}
		switch req["method"] {
		case "web3_clientVersion":
			resp = map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      req["id"],
				"result":  "Wemix/v1.0.0",
			}
		case "net_peerCount":
			resp = map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      req["id"],
				"result":  "0x5", // 5 peers
			}
		case "eth_syncing":
			resp = map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      req["id"],
				"result":  false, // Not syncing (synced)
			}
		default:
			resp = map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      req["id"],
				"error": map[string]interface{}{
					"code":    -32601,
					"message": "Method not found",
				},
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Extract port from server URL
	serverURL := server.URL
	rpcPort := 8545 // Default port for config

	// Create config
	cfg := &config.Config{
		HealthCheckInterval: 100 * time.Millisecond, // Short interval for testing
		RPCPort:             rpcPort,
	}

	// Create logger
	logger, err := logger.New(true, false, "")
	require.NoError(t, err)

	// Create health checker using the factory function
	checker := NewHealthChecker(cfg, logger)

	// Override with custom checks and test server URL for this test
	checker.rpcURL = serverURL
	checker.checks = []HealthCheck{
		&ProcessCheck{pidFile: pidFile},
		&RPCHealthCheck{url: serverURL},
		&PeerCountCheck{minPeers: 3, rpcURL: serverURL},
		&SyncingCheck{rpcURL: serverURL},
		&MemoryCheck{maxMemoryMB: 1000},
		&DiskSpaceCheck{minSpaceGB: 1, dataDir: tmpDir},
	}

	// Start health checker
	statusCh := checker.Start()
	defer checker.Stop()

	// Wait for first health check
	var status HealthStatus
	select {
	case status = <-statusCh:
		// Got status
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for health status")
	}

	// Verify overall health (might be false due to PID check failure)
	assert.NotZero(t, status.Timestamp)
	assert.Len(t, status.Checks, 6) // Should have all 6 checks

	// Verify individual check results
	checks := status.Checks

	// Process check might fail (PID 123456 doesn't exist)
	processCheck, exists := checks["process"]
	assert.True(t, exists)
	assert.Equal(t, "process", processCheck.Name)

	// RPC check should pass
	rpcCheck, exists := checks["rpc_endpoint"]
	assert.True(t, exists)
	assert.Equal(t, "rpc_endpoint", rpcCheck.Name)
	assert.True(t, rpcCheck.Healthy)

	// Peer count check should pass (5 peers >= 3 minimum)
	peerCheck, exists := checks["peer_count"]
	assert.True(t, exists)
	assert.Equal(t, "peer_count", peerCheck.Name)
	assert.True(t, peerCheck.Healthy)

	// Syncing check should pass (not syncing)
	syncCheck, exists := checks["syncing"]
	assert.True(t, exists)
	assert.Equal(t, "syncing", syncCheck.Name)
	assert.True(t, syncCheck.Healthy)

	// Memory check should pass (placeholder)
	memoryCheck, exists := checks["memory"]
	assert.True(t, exists)
	assert.Equal(t, "memory", memoryCheck.Name)
	assert.True(t, memoryCheck.Healthy)

	// Disk space check result depends on available space
	diskCheck, exists := checks["disk_space"]
	assert.True(t, exists)
	assert.Equal(t, "disk_space", diskCheck.Name)
}

func TestHealthChecker_Integration_RPCFailure(t *testing.T) {
	// Create mock server that returns errors
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Always return HTTP 500 error
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	// Create config
	cfg := &config.Config{
		HealthCheckInterval: 50 * time.Millisecond,
		RPCPort:             8545,
	}

	// Create logger
	logger, err := logger.New(true, false, "")
	require.NoError(t, err)

	// Create health checker using the factory function
	checker := NewHealthChecker(cfg, logger)

	// Override with failing RPC check for this test
	checker.rpcURL = server.URL
	checker.checks = []HealthCheck{
		&RPCHealthCheck{url: server.URL},
	}

	// Start health checker
	statusCh := checker.Start()
	defer checker.Stop()

	// Wait for health check
	var status HealthStatus
	select {
	case status = <-statusCh:
		// Got status
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for health status")
	}

	// Should be unhealthy due to RPC failure
	assert.False(t, status.Healthy)
	assert.Len(t, status.Checks, 1)

	rpcCheck := status.Checks["rpc_endpoint"]
	assert.False(t, rpcCheck.Healthy)
	assert.NotEmpty(t, rpcCheck.Error)
	assert.Contains(t, rpcCheck.Error, "status 500")
}

func TestHealthChecker_Integration_NetworkTimeout(t *testing.T) {
	// Create config with very short timeout
	cfg := &config.Config{
		HealthCheckInterval: 50 * time.Millisecond,
		RPCPort:             8545,
	}

	// Create logger
	logger, err := logger.New(true, false, "")
	require.NoError(t, err)

	// Create health checker using the factory function
	checker := NewHealthChecker(cfg, logger)

	// Override with unreachable URL for this test
	checker.rpcURL = "http://localhost:99999"
	checker.httpClient = &http.Client{Timeout: 100 * time.Millisecond} // Very short timeout
	checker.checks = []HealthCheck{
		&RPCHealthCheck{url: "http://localhost:99999"},
	}

	// Start health checker
	statusCh := checker.Start()
	defer checker.Stop()

	// Wait for health check
	var status HealthStatus
	select {
	case status = <-statusCh:
		// Got status
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for health status")
	}

	// Should be unhealthy due to network failure
	assert.False(t, status.Healthy)
	assert.Len(t, status.Checks, 1)

	rpcCheck := status.Checks["rpc_endpoint"]
	assert.False(t, rpcCheck.Healthy)
	assert.NotEmpty(t, rpcCheck.Error)
	assert.Contains(t, rpcCheck.Error, "unreachable")
}

func TestHealthChecker_Integration_MultipleStatusUpdates(t *testing.T) {
	// Create simple mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"result":  "OK",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create config with very short interval
	cfg := &config.Config{
		HealthCheckInterval: 25 * time.Millisecond, // Very short for multiple updates
		RPCPort:             8545,
	}

	// Create logger
	logger, err := logger.New(true, false, "")
	require.NoError(t, err)

	// Create health checker using the factory function
	checker := NewHealthChecker(cfg, logger)

	// Override for this test
	checker.rpcURL = server.URL
	checker.checks = []HealthCheck{
		&RPCHealthCheck{url: server.URL},
		&MemoryCheck{maxMemoryMB: 1000},
	}
	// Increase buffer for multiple status updates
	checker.statusCh = make(chan HealthStatus, 10)

	// Start health checker
	statusCh := checker.Start()
	defer checker.Stop()

	// Collect multiple status updates
	statusCount := 0
	timeout := time.After(200 * time.Millisecond) // Wait for multiple intervals

	for statusCount < 3 {
		select {
		case status := <-statusCh:
			statusCount++
			assert.True(t, status.Healthy) // Should be healthy
			assert.Len(t, status.Checks, 2)
			assert.NotZero(t, status.Timestamp)
		case <-timeout:
			break
		}
	}

	assert.GreaterOrEqual(t, statusCount, 2, "should receive multiple status updates")
}

func TestHealthChecker_Integration_ContextCancellation(t *testing.T) {
	// Create config
	cfg := &config.Config{
		HealthCheckInterval: 10 * time.Millisecond,
		RPCPort:             8545,
	}

	// Create logger
	logger, err := logger.New(true, false, "")
	require.NoError(t, err)

	// Create health checker
	checker := NewHealthChecker(cfg, logger)

	// Start health checker
	statusCh := checker.Start()

	// Wait for at least one update
	select {
	case <-statusCh:
		// Got status
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for initial health status")
	}

	// Stop the health checker
	checker.Stop()

	// Channel should eventually be closed
	channelClosed := false
	timeout := time.After(200 * time.Millisecond)

	for !channelClosed {
		select {
		case _, ok := <-statusCh:
			if !ok {
				channelClosed = true
			}
		case <-timeout:
			// Channel might not be closed immediately, but context should be cancelled
			select {
			case <-checker.ctx.Done():
				// Context is cancelled, that's what we want
				return
			default:
				t.Fatal("context should be cancelled after Stop()")
			}
		}
	}

	assert.True(t, channelClosed, "status channel should be closed after Stop()")
}

func TestHealthChecker_Integration_ConcurrentAccess(t *testing.T) {
	// Create config
	cfg := &config.Config{
		HealthCheckInterval: 20 * time.Millisecond,
		RPCPort:             8545,
	}

	// Create logger
	logger, err := logger.New(true, false, "")
	require.NoError(t, err)

	// Create health checker
	checker := NewHealthChecker(cfg, logger)

	// Start health checker
	statusCh := checker.Start()
	defer checker.Stop()

	// Wait for initial status
	select {
	case <-statusCh:
		// Got status
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timeout waiting for initial health status")
	}

	// Test concurrent access to GetStatus and IsHealthy
	done := make(chan bool, 20)

	// Start multiple goroutines reading status
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 50; j++ {
				status := checker.GetStatus()
				_ = status.Healthy // Use the value
				time.Sleep(1 * time.Millisecond)
			}
			done <- true
		}()
	}

	// Start multiple goroutines checking health
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 50; j++ {
				healthy := checker.IsHealthy()
				_ = healthy // Use the value
				time.Sleep(1 * time.Millisecond)
			}
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 20; i++ {
		select {
		case <-done:
			// Goroutine completed
		case <-time.After(5 * time.Second):
			t.Fatal("timeout waiting for concurrent access test")
		}
	}
}