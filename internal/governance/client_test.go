package governance

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/wemix/wemixvisor/pkg/logger"
)

func TestNewWBFTClient(t *testing.T) {
	testLogger := logger.NewTestLogger()

	tests := []struct {
		name        string
		baseURL     string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "empty URL",
			baseURL:     "",
			expectError: true,
			errorMsg:    "baseURL cannot be empty",
		},
		{
			name:        "valid URL",
			baseURL:     "http://localhost:8545",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewWBFTClient(tt.baseURL, testLogger)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, client)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
				assert.Equal(t, tt.baseURL, client.baseURL)
				assert.Equal(t, testLogger, client.logger)
				assert.Equal(t, 30*time.Second, client.timeout)
			}
		})
	}
}

func TestWBFTClient_Close(t *testing.T) {
	testLogger := logger.NewTestLogger()
	client, err := NewWBFTClient("http://localhost:8545", testLogger)
	assert.NoError(t, err)

	err = client.Close()
	assert.NoError(t, err)
}

func TestWBFTClient_SetTimeout(t *testing.T) {
	testLogger := logger.NewTestLogger()
	client, err := NewWBFTClient("http://localhost:8545", testLogger)
	assert.NoError(t, err)

	newTimeout := 60 * time.Second
	client.SetTimeout(newTimeout)
	assert.Equal(t, newTimeout, client.timeout)
	assert.Equal(t, newTimeout, client.httpClient.Timeout)
}

func TestWBFTClient_GetCurrentHeight(t *testing.T) {
	testLogger := logger.NewTestLogger()

	t.Run("successful response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			response := `{
				"jsonrpc": "2.0",
				"result": {
					"sync_info": {
						"latest_block_height": "1000"
					}
				},
				"id": 1
			}`
			w.Write([]byte(response))
		}))
		defer server.Close()

		client, err := NewWBFTClient(server.URL, testLogger)
		assert.NoError(t, err)

		height, err := client.GetCurrentHeight()
		assert.NoError(t, err)
		assert.Equal(t, int64(1000), height)
	})

	t.Run("error response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			response := `{
				"jsonrpc": "2.0",
				"error": {
					"code": -32603,
					"message": "Internal error"
				},
				"id": 1
			}`
			w.Write([]byte(response))
		}))
		defer server.Close()

		client, err := NewWBFTClient(server.URL, testLogger)
		assert.NoError(t, err)

		_, err = client.GetCurrentHeight()
		assert.Error(t, err)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`invalid json`))
		}))
		defer server.Close()

		client, err := NewWBFTClient(server.URL, testLogger)
		assert.NoError(t, err)

		_, err = client.GetCurrentHeight()
		assert.Error(t, err)
	})
}

func TestWBFTClient_GetBlock(t *testing.T) {
	testLogger := logger.NewTestLogger()

	t.Run("error response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			response := `{
				"jsonrpc": "2.0",
				"error": {
					"code": -32603,
					"message": "Block not found"
				},
				"id": 1
			}`
			w.Write([]byte(response))
		}))
		defer server.Close()

		client, err := NewWBFTClient(server.URL, testLogger)
		assert.NoError(t, err)

		block, err := client.GetBlock(999)
		assert.Error(t, err)
		assert.Nil(t, block)
	})
}

func TestWBFTClient_GetGovernanceProposals(t *testing.T) {
	testLogger := logger.NewTestLogger()

	t.Run("error response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			response := `{
				"jsonrpc": "2.0",
				"error": {
					"code": -32603,
					"message": "Internal error"
				},
				"id": 1
			}`
			w.Write([]byte(response))
		}))
		defer server.Close()

		client, err := NewWBFTClient(server.URL, testLogger)
		assert.NoError(t, err)

		proposals, err := client.GetGovernanceProposals(ProposalStatusPassed)
		assert.Error(t, err)
		assert.Nil(t, proposals)
	})
}

func TestWBFTClient_GetProposal(t *testing.T) {
	testLogger := logger.NewTestLogger()

	t.Run("proposal not found", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			response := `{
				"jsonrpc": "2.0",
				"error": {
					"code": -32603,
					"message": "Proposal not found"
				},
				"id": 1
			}`
			w.Write([]byte(response))
		}))
		defer server.Close()

		client, err := NewWBFTClient(server.URL, testLogger)
		assert.NoError(t, err)

		proposal, err := client.GetProposal("999")
		assert.Error(t, err)
		assert.Nil(t, proposal)
	})
}

func TestWBFTClient_GetValidators(t *testing.T) {
	testLogger := logger.NewTestLogger()

	t.Run("error response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			response := `{
				"jsonrpc": "2.0",
				"error": {
					"code": -32603,
					"message": "Internal error"
				},
				"id": 1
			}`
			w.Write([]byte(response))
		}))
		defer server.Close()

		client, err := NewWBFTClient(server.URL, testLogger)
		assert.NoError(t, err)

		validators, err := client.GetValidators()
		assert.Error(t, err)
		assert.Nil(t, validators)
	})
}

func TestWBFTClient_GetGovernanceParams(t *testing.T) {
	testLogger := logger.NewTestLogger()

	t.Run("error response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			response := `{
				"jsonrpc": "2.0",
				"error": {
					"code": -32603,
					"message": "Internal error"
				},
				"id": 1
			}`
			w.Write([]byte(response))
		}))
		defer server.Close()

		client, err := NewWBFTClient(server.URL, testLogger)
		assert.NoError(t, err)

		params, err := client.GetGovernanceParams()
		assert.Error(t, err)
		assert.Nil(t, params)
	})
}

func TestWBFTClient_NetworkError(t *testing.T) {
	testLogger := logger.NewTestLogger()

	// Create client with invalid URL to test network errors
	client, err := NewWBFTClient("http://localhost:99999", testLogger)
	assert.NoError(t, err)

	// Set short timeout to fail quickly
	client.SetTimeout(100 * time.Millisecond)

	// Test network error scenarios
	t.Run("GetCurrentHeight network error", func(t *testing.T) {
		_, err := client.GetCurrentHeight()
		assert.Error(t, err)
	})

	t.Run("GetBlock network error", func(t *testing.T) {
		_, err := client.GetBlock(1000)
		assert.Error(t, err)
	})

	t.Run("GetGovernanceProposals network error", func(t *testing.T) {
		_, err := client.GetGovernanceProposals(ProposalStatusSubmitted)
		assert.Error(t, err)
	})

	t.Run("GetProposal network error", func(t *testing.T) {
		_, err := client.GetProposal("1")
		assert.Error(t, err)
	})

	t.Run("GetValidators network error", func(t *testing.T) {
		_, err := client.GetValidators()
		assert.Error(t, err)
	})

	t.Run("GetGovernanceParams network error", func(t *testing.T) {
		_, err := client.GetGovernanceParams()
		assert.Error(t, err)
	})
}

