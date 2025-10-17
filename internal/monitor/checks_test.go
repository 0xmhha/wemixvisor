package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessCheck(t *testing.T) {
	// Test without PID file (should pass)
	check := &ProcessCheck{}
	err := check.Check(context.Background())
	assert.NoError(t, err)

	// Test with valid PID file (current process)
	tmpDir := t.TempDir()
	pidFile := filepath.Join(tmpDir, "test.pid")
	require.NoError(t, os.WriteFile(pidFile, []byte(fmt.Sprintf("%d", os.Getpid())), 0644))

	check = &ProcessCheck{pidFile: pidFile}
	err = check.Check(context.Background())
	assert.NoError(t, err)

	// Test with invalid PID file
	require.NoError(t, os.WriteFile(pidFile, []byte("not-a-number"), 0644))
	err = check.Check(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid PID")

	// Test with non-existent PID file
	check = &ProcessCheck{pidFile: "/non/existent/file.pid"}
	err = check.Check(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot read PID file")
}

func TestRPCHealthCheck(t *testing.T) {
	// Create a mock RPC server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request method
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// Decode request
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Check method
		if req["method"] == "web3_clientVersion" {
			// Return success response
			resp := map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      req["id"],
				"result":  "Wemix/v1.0.0",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		} else {
			// Return error response
			resp := map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      req["id"],
				"error": map[string]interface{}{
					"code":    -32601,
					"message": "Method not found",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	// Test successful health check
	check := &RPCHealthCheck{url: server.URL}
	err := check.Check(context.Background())
	assert.NoError(t, err)

	// Test with unreachable server
	check = &RPCHealthCheck{url: "http://localhost:99999"}
	err = check.Check(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unreachable")

	// Test with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()
	err = check.Check(ctx)
	assert.Error(t, err)
}

func TestPeerCountCheck(t *testing.T) {
	// Create mock RPC server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]interface{}
		json.NewDecoder(r.Body).Decode(&req)

		if req["method"] == "net_peerCount" {
			// Return 5 peers
			resp := map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      req["id"],
				"result":  "0x5", // 5 in hex
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	// Test with sufficient peers
	check := &PeerCountCheck{minPeers: 3, rpcURL: server.URL}
	err := check.Check(context.Background())
	assert.NoError(t, err)

	// Test with insufficient peers
	check = &PeerCountCheck{minPeers: 10, rpcURL: server.URL}
	err = check.Check(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insufficient peers")
}

func TestSyncingCheck(t *testing.T) {
	tests := []struct {
		name      string
		response  interface{}
		expectErr bool
		errMsg    string
	}{
		{
			name:      "synced",
			response:  false,
			expectErr: false,
		},
		{
			name: "syncing",
			response: map[string]interface{}{
				"currentBlock": "0x1234",
				"highestBlock": "0x5678",
			},
			expectErr: true,
			errMsg:    "node is syncing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock RPC server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var req map[string]interface{}
				json.NewDecoder(r.Body).Decode(&req)

				if req["method"] == "eth_syncing" {
					resp := map[string]interface{}{
						"jsonrpc": "2.0",
						"id":      req["id"],
						"result":  tt.response,
					}
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(resp)
				}
			}))
			defer server.Close()

			check := &SyncingCheck{rpcURL: server.URL}
			err := check.Check(context.Background())

			if tt.expectErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDiskSpaceCheck(t *testing.T) {
	// Use temp directory for testing
	tmpDir := t.TempDir()

	// This test is basic as actual disk space varies by system
	check := &DiskSpaceCheck{
		minSpaceGB: 1, // 1 GB minimum (most systems should have this)
		dataDir:    tmpDir,
	}

	err := check.Check(context.Background())
	// We can't guarantee the test environment has specific free space
	// so we just check that the function executes without panic
	if err != nil {
		assert.Contains(t, err.Error(), "insufficient disk space")
	}
}

func TestMemoryCheck(t *testing.T) {
	// Memory check is a placeholder in the current implementation
	check := &MemoryCheck{maxMemoryMB: 1000}
	err := check.Check(context.Background())
	assert.NoError(t, err) // Should always pass for now
}