package governance

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/wemix/wemixvisor/pkg/logger"
)

// Test error handling paths to improve coverage to 90%+

func TestWBFTClient_GetCurrentHeight_InvalidJSON(t *testing.T) {
	testLogger := logger.NewTestLogger()

	server := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Send malformed JSON in result
		response := `{
			"jsonrpc": "2.0",
			"result": "invalid json here",
			"id": 1
		}`
		w.Write([]byte(response))
	}))
	defer server.Close()

	client, err := NewWBFTClient(server.URL, testLogger)
	assert.NoError(t, err)

	_, err = client.GetCurrentHeight()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse status response")
}

func TestWBFTClient_GetCurrentHeight_InvalidHeightFormat(t *testing.T) {
	testLogger := logger.NewTestLogger()

	server := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Send non-numeric height
		response := `{
			"jsonrpc": "2.0",
			"result": {
				"sync_info": {
					"latest_block_height": "not-a-number"
				}
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
	assert.Contains(t, err.Error(), "failed to parse height")
}

func TestWBFTClient_GetBlock_InvalidJSON(t *testing.T) {
	testLogger := logger.NewTestLogger()

	server := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Send malformed JSON
		response := `{
			"jsonrpc": "2.0",
			"result": "not a valid block structure",
			"id": 1
		}`
		w.Write([]byte(response))
	}))
	defer server.Close()

	client, err := NewWBFTClient(server.URL, testLogger)
	assert.NoError(t, err)

	_, err = client.GetBlock(1000)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse block response")
}

func TestWBFTClient_GetGovernanceProposals_InvalidJSON(t *testing.T) {
	testLogger := logger.NewTestLogger()

	server := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Send invalid JSON structure
		response := `{
			"jsonrpc": "2.0",
			"result": "invalid proposals data",
			"id": 1
		}`
		w.Write([]byte(response))
	}))
	defer server.Close()

	client, err := NewWBFTClient(server.URL, testLogger)
	assert.NoError(t, err)

	_, err = client.GetGovernanceProposals(ProposalStatusVoting)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse proposals response")
}

func TestWBFTClient_GetProposal_InvalidJSON(t *testing.T) {
	testLogger := logger.NewTestLogger()

	server := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Send invalid JSON structure
		response := `{
			"jsonrpc": "2.0",
			"result": "not a valid proposal",
			"id": 1
		}`
		w.Write([]byte(response))
	}))
	defer server.Close()

	client, err := NewWBFTClient(server.URL, testLogger)
	assert.NoError(t, err)

	_, err = client.GetProposal("1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse proposal response")
}

func TestWBFTClient_GetValidators_InvalidJSON(t *testing.T) {
	testLogger := logger.NewTestLogger()

	server := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Send invalid JSON structure
		response := `{
			"jsonrpc": "2.0",
			"result": "not valid validators",
			"id": 1
		}`
		w.Write([]byte(response))
	}))
	defer server.Close()

	client, err := NewWBFTClient(server.URL, testLogger)
	assert.NoError(t, err)

	_, err = client.GetValidators()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse validators response")
}

func TestWBFTClient_GetValidators_InvalidVotingPower(t *testing.T) {
	testLogger := logger.NewTestLogger()

	server := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Send non-numeric voting power
		response := `{
			"jsonrpc": "2.0",
			"result": {
				"validators": [
					{
						"address": "wemixvaloper1abc",
						"pub_key": "base64key",
						"voting_power": "not-a-number"
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

	validators, err := client.GetValidators()
	assert.NoError(t, err)
	assert.Len(t, validators, 1)
	// Should default to 0 when parse fails
	assert.Equal(t, int64(0), validators[0].VotingPower)
}

func TestWBFTClient_GetGovernanceParams_InvalidJSON(t *testing.T) {
	testLogger := logger.NewTestLogger()

	server := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Send invalid JSON structure
		response := `{
			"jsonrpc": "2.0",
			"result": "invalid params",
			"id": 1
		}`
		w.Write([]byte(response))
	}))
	defer server.Close()

	client, err := NewWBFTClient(server.URL, testLogger)
	assert.NoError(t, err)

	_, err = client.GetGovernanceParams()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse governance params response")
}

func TestWBFTClient_GetGovernanceParams_InvalidDuration(t *testing.T) {
	testLogger := logger.NewTestLogger()

	server := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Send invalid duration format
		response := `{
			"jsonrpc": "2.0",
			"result": {
				"voting_params": {
					"voting_period": "invalid-duration"
				},
				"deposit_params": {
					"min_deposit": []
				},
				"tally_params": {
					"quorum": "0.334",
					"threshold": "0.5",
					"veto_threshold": "0.334"
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
	// Should use default duration when parse fails
	assert.Equal(t, 14*24*time.Hour, params.VotingPeriod)
}

func TestWBFTClient_MakeRequest_InvalidJSON(t *testing.T) {
	testLogger := logger.NewTestLogger()

	server := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Send completely invalid JSON
		w.Write([]byte("not json at all {"))
	}))
	defer server.Close()

	client, err := NewWBFTClient(server.URL, testLogger)
	assert.NoError(t, err)

	ctx := context.Background()
	_, err = client.makeRequest(ctx, "test", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode response")
}

func TestWBFTClient_MakeRequest_EmptyResponse(t *testing.T) {
	testLogger := logger.NewTestLogger()

	server := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Send empty response
		w.Write([]byte(""))
	}))
	defer server.Close()

	client, err := NewWBFTClient(server.URL, testLogger)
	assert.NoError(t, err)

	ctx := context.Background()
	_, err = client.makeRequest(ctx, "test", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode response")
}

// Test ProposalType parameter case
func TestWBFTClient_ParseProposalContent_ParameterType(t *testing.T) {
	testLogger := logger.NewTestLogger()

	server := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := `{
			"jsonrpc": "2.0",
			"result": {
				"proposals": [
					{
						"proposal_id": "1",
						"status": "PROPOSAL_STATUS_VOTING_PERIOD",
						"submit_time": "2024-01-01T00:00:00Z",
						"voting_start_time": "2024-01-01T00:00:00Z",
						"voting_end_time": "2024-01-02T00:00:00Z",
						"content": {
							"@type": "/cosmos.gov.v1beta1.TextProposal",
							"title": "Text Proposal",
							"description": "Test Description"
						},
						"final_tally_result": {
							"yes": "0",
							"no": "0",
							"abstain": "0",
							"no_with_veto": "0"
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
	assert.Len(t, proposals, 1)
	// TextProposal should map to ProposalTypeText
	assert.Equal(t, ProposalTypeText, proposals[0].Type)
}
