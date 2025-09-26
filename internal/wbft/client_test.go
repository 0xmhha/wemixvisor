package wbft

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/wemix/wemixvisor/internal/config"
	"github.com/wemix/wemixvisor/pkg/logger"
)

func TestNewClient(t *testing.T) {
	cfg := &config.Config{
		Home:       t.TempDir(),
		Name:       "wemixd",
		RPCAddress: "localhost:8545",
	}
	logger, _ := logger.New(false, true, "")

	client := NewClient(cfg, logger)
	if client == nil {
		t.Fatal("expected client to be non-nil")
	}
	if client.cfg != cfg {
		t.Error("expected cfg to be set")
	}
	if client.logger != logger {
		t.Error("expected logger to be set")
	}
	if client.rpcURL != "http://localhost:8545" {
		t.Errorf("expected RPC URL to be http://localhost:8545, got %s", client.rpcURL)
	}
}

func TestGetCurrentHeight(t *testing.T) {
	// Create mock RPC server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse request
		var req map[string]interface{}
		json.NewDecoder(r.Body).Decode(&req)

		if req["method"] == "status" {
			resp := map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      req["id"],
				"result": map[string]interface{}{
					"node_info": map[string]interface{}{
						"network": "wemix-testnet",
					},
					"sync_info": map[string]interface{}{
						"latest_block_height": float64(1234567),
						"catching_up":         false,
					},
					"validator_info": map[string]interface{}{
						"voting_power": float64(100),
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	cfg := &config.Config{
		Home:       t.TempDir(),
		Name:       "wemixd",
		RPCAddress: server.URL,
	}
	logger, _ := logger.New(false, true, "")
	client := NewClient(cfg, logger)

	ctx := context.Background()
	height, err := client.GetCurrentHeight(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if height != 1234567 {
		t.Errorf("expected height 1234567, got %d", height)
	}
}

func TestIsValidator(t *testing.T) {
	tests := []struct {
		name          string
		validatorMode bool
		votingPower   float64
		expected      bool
	}{
		{
			name:          "validator with power",
			validatorMode: true,
			votingPower:   100,
			expected:      true,
		},
		{
			name:          "validator without power",
			validatorMode: true,
			votingPower:   0,
			expected:      false,
		},
		{
			name:          "non-validator mode",
			validatorMode: false,
			votingPower:   100,
			expected:      false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create mock RPC server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var req map[string]interface{}
				json.NewDecoder(r.Body).Decode(&req)

				if req["method"] == "status" {
					resp := map[string]interface{}{
						"jsonrpc": "2.0",
						"id":      req["id"],
						"result": map[string]interface{}{
							"sync_info": map[string]interface{}{
								"latest_block_height": float64(1000000),
								"catching_up":         false,
							},
							"validator_info": map[string]interface{}{
								"voting_power": tc.votingPower,
							},
						},
					}
					json.NewEncoder(w).Encode(resp)
				}
			}))
			defer server.Close()

			cfg := &config.Config{
				Home:          t.TempDir(),
				Name:          "wemixd",
				RPCAddress:    server.URL,
				ValidatorMode: tc.validatorMode,
			}
			logger, _ := logger.New(false, true, "")
			client := NewClient(cfg, logger)

			ctx := context.Background()
			isValidator, err := client.IsValidator(ctx)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if isValidator != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, isValidator)
			}
		})
	}
}

func TestWaitForHeight(t *testing.T) {
	currentHeight := int64(1000)
	targetHeight := int64(1003)

	// Create mock RPC server that increments height
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]interface{}
		json.NewDecoder(r.Body).Decode(&req)

		if req["method"] == "status" {
			// Increment height on each call
			currentHeight++
			resp := map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      req["id"],
				"result": map[string]interface{}{
					"sync_info": map[string]interface{}{
						"latest_block_height": float64(currentHeight),
						"catching_up":         false,
					},
					"validator_info": map[string]interface{}{
						"voting_power": float64(0),
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	cfg := &config.Config{
		Home:       t.TempDir(),
		Name:       "wemixd",
		RPCAddress: server.URL,
	}
	logger, _ := logger.New(false, true, "")
	client := NewClient(cfg, logger)

	ctx := context.Background()
	err := client.WaitForHeight(ctx, targetHeight, 5*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify we reached target height
	if currentHeight < targetHeight {
		t.Errorf("did not reach target height: current %d, target %d", currentHeight, targetHeight)
	}
}

func TestWaitForHeightTimeout(t *testing.T) {
	// Create mock RPC server that returns fixed height
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]interface{}
		json.NewDecoder(r.Body).Decode(&req)

		if req["method"] == "status" {
			resp := map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      req["id"],
				"result": map[string]interface{}{
					"sync_info": map[string]interface{}{
						"latest_block_height": float64(1000),
						"catching_up":         false,
					},
					"validator_info": map[string]interface{}{
						"voting_power": float64(0),
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	cfg := &config.Config{
		Home:       t.TempDir(),
		Name:       "wemixd",
		RPCAddress: server.URL,
	}
	logger, _ := logger.New(false, true, "")
	client := NewClient(cfg, logger)

	ctx := context.Background()
	err := client.WaitForHeight(ctx, 2000, 2*time.Second)
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestCheckReadiness(t *testing.T) {
	tests := []struct {
		name        string
		syncing     bool
		validator   bool
		votingPower float64
		expectError bool
	}{
		{
			name:        "ready non-validator",
			syncing:     false,
			validator:   false,
			votingPower: 0,
			expectError: false,
		},
		{
			name:        "ready validator",
			syncing:     false,
			validator:   true,
			votingPower: 100,
			expectError: false,
		},
		{
			name:        "syncing node",
			syncing:     true,
			validator:   false,
			votingPower: 0,
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create mock RPC server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var req map[string]interface{}
				json.NewDecoder(r.Body).Decode(&req)

				switch req["method"] {
				case "status":
					resp := map[string]interface{}{
						"jsonrpc": "2.0",
						"id":      req["id"],
						"result": map[string]interface{}{
							"sync_info": map[string]interface{}{
								"latest_block_height": float64(1000000),
								"catching_up":         tc.syncing,
							},
							"validator_info": map[string]interface{}{
								"voting_power": tc.votingPower,
							},
						},
					}
					json.NewEncoder(w).Encode(resp)

				case "consensus_state":
					resp := map[string]interface{}{
						"jsonrpc": "2.0",
						"id":      req["id"],
						"result": map[string]interface{}{
							"height":       float64(1000000),
							"is_syncing":   tc.syncing,
							"is_validator": tc.votingPower > 0,
						},
					}
					json.NewEncoder(w).Encode(resp)
				}
			}))
			defer server.Close()

			cfg := &config.Config{
				Home:          t.TempDir(),
				Name:          "wemixd",
				RPCAddress:    server.URL,
				ValidatorMode: tc.validator,
			}
			logger, _ := logger.New(false, true, "")
			client := NewClient(cfg, logger)

			ctx := context.Background()
			err := client.CheckReadiness(ctx)
			if tc.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestMonitorConsensus(t *testing.T) {
	height := int64(1000)

	// Create mock RPC server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]interface{}
		json.NewDecoder(r.Body).Decode(&req)

		switch req["method"] {
		case "status":
			height++ // Increment height each call
			resp := map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      req["id"],
				"result": map[string]interface{}{
					"sync_info": map[string]interface{}{
						"latest_block_height": float64(height),
						"catching_up":         false,
					},
					"validator_info": map[string]interface{}{
						"voting_power": float64(100),
					},
				},
			}
			json.NewEncoder(w).Encode(resp)

		case "consensus_state":
			resp := map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      req["id"],
				"result": map[string]interface{}{
					"height":       float64(height),
					"round":        0,
					"step":         "commit",
					"is_validator": true,
					"is_syncing":   false,
				},
			}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	cfg := &config.Config{
		Home:       t.TempDir(),
		Name:       "wemixd",
		RPCAddress: server.URL,
	}
	logger, _ := logger.New(false, true, "")
	client := NewClient(cfg, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stateChan := client.MonitorConsensus(ctx, 500*time.Millisecond)

	// Collect a few state updates
	states := make([]*ConsensusState, 0)
	timeout := time.After(2 * time.Second)

	for {
		select {
		case state := <-stateChan:
			if state != nil {
				states = append(states, state)
				if len(states) >= 2 {
					goto done
				}
			}
		case <-timeout:
			goto done
		}
	}

done:
	if len(states) < 2 {
		t.Errorf("expected at least 2 state updates, got %d", len(states))
	}

	// Check that heights are increasing
	if len(states) >= 2 && states[1].Height <= states[0].Height {
		t.Error("heights should be increasing")
	}
}

func TestRPCError(t *testing.T) {
	// Create mock RPC server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"error": map[string]interface{}{
				"code":    -32601,
				"message": "Method not found",
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := &config.Config{
		Home:       t.TempDir(),
		Name:       "wemixd",
		RPCAddress: server.URL,
	}
	logger, _ := logger.New(false, true, "")
	client := NewClient(cfg, logger)

	ctx := context.Background()
	_, err := client.GetCurrentHeight(ctx)
	if err == nil {
		t.Fatal("expected RPC error")
	}

	expectedErr := "RPC error -32601: Method not found"
	if !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("expected error containing '%s', got '%s'", expectedErr, err.Error())
	}
}