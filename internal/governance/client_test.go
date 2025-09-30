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

func TestWBFTClient_GetBlock_Success(t *testing.T) {
	testLogger := logger.NewTestLogger()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := `{
			"jsonrpc": "2.0",
			"result": {
				"block": {
					"header": {
						"height": "1000",
						"time": "2024-01-01T00:00:00Z",
						"hash": "0xabc123"
					},
					"data": {
						"txs": ["tx1", "tx2", "tx3"]
					}
				}
			},
			"id": 1
		}`
		w.Write([]byte(response))
	}))
	defer server.Close()

	client, err := NewWBFTClient(server.URL, testLogger)
	assert.NoError(t, err)

	block, err := client.GetBlock(1000)
	assert.NoError(t, err)
	assert.NotNil(t, block)
	assert.Equal(t, int64(1000), block.Height)
	assert.Equal(t, "0xabc123", block.Hash)
	assert.Equal(t, 3, block.TxCount)
}

func TestWBFTClient_GetGovernanceProposals_Success(t *testing.T) {
	testLogger := logger.NewTestLogger()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := `{
			"jsonrpc": "2.0",
			"result": {
				"proposals": [
					{
						"id": "1",
						"title": "Test Proposal",
						"description": "Test Description",
						"status": "PROPOSAL_STATUS_VOTING_PERIOD",
						"submit_time": "2024-01-01T00:00:00Z",
						"voting_start_time": "2024-01-01T00:00:00Z",
						"voting_end_time": "2024-01-02T00:00:00Z",
						"content": {
							"@type": "/cosmos.gov.v1.TextProposal",
							"title": "Test Proposal",
							"description": "Test Description"
						}
					}
				]
			},
			"id": 1
		}`
		w.Write([]byte(response))
	}))
	defer server.Close()

	client, err := NewWBFTClient(server.URL, testLogger)
	assert.NoError(t, err)

	proposals, err := client.GetGovernanceProposals(ProposalStatusVoting)
	assert.NoError(t, err)
	assert.NotNil(t, proposals)
	assert.Len(t, proposals, 1)
	assert.Equal(t, "1", proposals[0].ID)
	assert.Equal(t, "Test Proposal", proposals[0].Title)
}

func TestWBFTClient_GetProposal_Success(t *testing.T) {
	testLogger := logger.NewTestLogger()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := `{
			"jsonrpc": "2.0",
			"result": {
				"proposal": {
					"id": "1",
					"title": "Test Proposal",
					"description": "Test Description",
					"status": "PROPOSAL_STATUS_PASSED",
					"submit_time": "2024-01-01T00:00:00Z"
				}
			},
			"id": 1
		}`
		w.Write([]byte(response))
	}))
	defer server.Close()

	client, err := NewWBFTClient(server.URL, testLogger)
	assert.NoError(t, err)

	proposal, err := client.GetProposal("1")
	assert.NoError(t, err)
	assert.NotNil(t, proposal)
	assert.Equal(t, "1", proposal.ID)
	assert.Equal(t, "Test Proposal", proposal.Title)
}

func TestWBFTClient_GetValidators_Success(t *testing.T) {
	testLogger := logger.NewTestLogger()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := `{
			"jsonrpc": "2.0",
			"result": {
				"validators": [
					{
						"operator_address": "wemixvaloper1abc",
						"consensus_pubkey": {
							"@type": "/cosmos.crypto.ed25519.PubKey",
							"key": "base64key"
						},
						"status": "BOND_STATUS_BONDED",
						"tokens": "1000000",
						"delegator_shares": "1000000.000000000000000000",
						"moniker": "Validator1",
						"commission": {
							"commission_rates": {
								"rate": "0.100000000000000000"
							}
						}
					}
				],
				"pagination": {
					"total": "1"
				}
			},
			"id": 1
		}`
		w.Write([]byte(response))
	}))
	defer server.Close()

	client, err := NewWBFTClient(server.URL, testLogger)
	assert.NoError(t, err)

	validators, err := client.GetValidators()
	assert.NoError(t, err)
	assert.NotNil(t, validators)
	assert.Len(t, validators, 1)
	assert.Equal(t, "wemixvaloper1abc", validators[0].OperatorAddress)
	assert.Equal(t, "Validator1", validators[0].Moniker)
}

func TestWBFTClient_GetGovernanceParams_Success(t *testing.T) {
	testLogger := logger.NewTestLogger()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := `{
			"jsonrpc": "2.0",
			"result": {
				"voting_params": {
					"voting_period": "172800s"
				},
				"deposit_params": {
					"min_deposit": [
						{
							"denom": "uwemix",
							"amount": "10000000"
						}
					],
					"max_deposit_period": "172800s"
				},
				"tally_params": {
					"quorum": "0.334000000000000000",
					"threshold": "0.500000000000000000",
					"veto_threshold": "0.334000000000000000"
				}
			},
			"id": 1
		}`
		w.Write([]byte(response))
	}))
	defer server.Close()

	client, err := NewWBFTClient(server.URL, testLogger)
	assert.NoError(t, err)

	params, err := client.GetGovernanceParams()
	assert.NoError(t, err)
	assert.NotNil(t, params)
	assert.Equal(t, "172800s", params.VotingPeriod)
	assert.Equal(t, "0.334000000000000000", params.QuorumThreshold)
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

func TestWBFTClient_MakeRequest_Error(t *testing.T) {
	testLogger := logger.NewTestLogger()

	t.Run("JSON RPC Error Response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			response := `{
				"jsonrpc": "2.0",
				"error": {
					"code": -32700,
					"message": "Parse error"
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
		assert.Contains(t, err.Error(), "Parse error")
	})

	t.Run("HTTP Error Status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		client, err := NewWBFTClient(server.URL, testLogger)
		assert.NoError(t, err)

		_, err = client.GetCurrentHeight()
		assert.Error(t, err)
	})
}

func TestWBFTClient_ParseProposalContent(t *testing.T) {
	testLogger := logger.NewTestLogger()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := `{
			"jsonrpc": "2.0",
			"result": {
				"proposals": [
					{
						"id": "1",
						"title": "Software Upgrade",
						"content": {
							"@type": "/cosmos.upgrade.v1beta1.SoftwareUpgradeProposal",
							"plan": {
								"name": "v2.0.0",
								"height": "10000",
								"info": "{\"binaries\":{\"linux\":{\"url\":\"https://example.com/binary\",\"checksum\":\"sha256:abc123\"}}}"
							}
						}
					},
					{
						"id": "2",
						"title": "Parameter Change",
						"content": {
							"@type": "/cosmos.params.v1beta1.ParameterChangeProposal",
							"changes": [
								{
									"subspace": "staking",
									"key": "MaxValidators",
									"value": "100"
								}
							]
						}
					},
					{
						"id": "3",
						"title": "Unknown Type",
						"content": {
							"@type": "/unknown.type",
							"data": "test"
						}
					}
				]
			},
			"id": 1
		}`
		w.Write([]byte(response))
	}))
	defer server.Close()

	client, err := NewWBFTClient(server.URL, testLogger)
	assert.NoError(t, err)

	proposals, err := client.GetGovernanceProposals(ProposalStatusSubmitted)
	assert.NoError(t, err)
	assert.Len(t, proposals, 3)

	// Check software upgrade proposal
	assert.Equal(t, ProposalTypeUpgrade, proposals[0].Type)
	assert.NotNil(t, proposals[0].UpgradeInfo)
	assert.Equal(t, "v2.0.0", proposals[0].UpgradeInfo.Name)

	// Check parameter change proposal defaults to text for now
	assert.Equal(t, ProposalTypeText, proposals[1].Type)

	// Check unknown type defaults to text
	assert.Equal(t, ProposalTypeText, proposals[2].Type)
}

