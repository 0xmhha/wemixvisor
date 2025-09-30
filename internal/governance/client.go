package governance

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/wemix/wemixvisor/pkg/logger"
	"go.uber.org/zap"
)

// WBFTClientInterface defines the interface for WBFT client operations
type WBFTClientInterface interface {
	Close() error
	GetCurrentHeight() (int64, error)
	GetBlock(height int64) (*BlockInfo, error)
	GetGovernanceProposals(status ProposalStatus) ([]*Proposal, error)
	GetProposal(proposalID string) (*Proposal, error)
	GetValidators() ([]*ValidatorInfo, error)
	GetGovernanceParams() (*GovernanceParams, error)
	SetTimeout(timeout time.Duration)
}

// WBFTClient represents a client for communicating with WBFT blockchain nodes
type WBFTClient struct {
	baseURL    string
	httpClient *http.Client
	logger     *logger.Logger
	timeout    time.Duration
}

// RPCRequest represents a JSON-RPC request
type RPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
	ID      int         `json:"id"`
}

// RPCResponse represents a JSON-RPC response
type RPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
	ID      int             `json:"id"`
}

// RPCError represents a JSON-RPC error
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data,omitempty"`
}

// NewWBFTClient creates a new WBFT RPC client
func NewWBFTClient(baseURL string, logger *logger.Logger) (*WBFTClient, error) {
	if baseURL == "" {
		return nil, fmt.Errorf("baseURL cannot be empty")
	}

	return &WBFTClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger:  logger,
		timeout: 30 * time.Second,
	}, nil
}

// Close closes the client and cleans up resources
func (c *WBFTClient) Close() error {
	// Close any persistent connections
	c.httpClient.CloseIdleConnections()
	return nil
}

// GetCurrentHeight returns the current blockchain height
func (c *WBFTClient) GetCurrentHeight() (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	resp, err := c.makeRequest(ctx, "status", nil)
	if err != nil {
		return 0, fmt.Errorf("failed to get status: %w", err)
	}

	var status struct {
		SyncInfo struct {
			LatestBlockHeight string `json:"latest_block_height"`
		} `json:"sync_info"`
	}

	if err := json.Unmarshal(resp.Result, &status); err != nil {
		return 0, fmt.Errorf("failed to parse status response: %w", err)
	}

	var height int64
	if _, err := fmt.Sscanf(status.SyncInfo.LatestBlockHeight, "%d", &height); err != nil {
		return 0, fmt.Errorf("failed to parse height: %w", err)
	}

	return height, nil
}

// GetBlock returns information about a specific block
func (c *WBFTClient) GetBlock(height int64) (*BlockInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	params := map[string]interface{}{
		"height": fmt.Sprintf("%d", height),
	}

	resp, err := c.makeRequest(ctx, "block", params)
	if err != nil {
		return nil, fmt.Errorf("failed to get block: %w", err)
	}

	var blockResp struct {
		Block struct {
			Header struct {
				Height string    `json:"height"`
				Time   time.Time `json:"time"`
				Hash   string    `json:"hash"`
			} `json:"header"`
			Data struct {
				Txs []string `json:"txs"`
			} `json:"data"`
		} `json:"block"`
	}

	if err := json.Unmarshal(resp.Result, &blockResp); err != nil {
		return nil, fmt.Errorf("failed to parse block response: %w", err)
	}

	var blockHeight int64
	if _, err := fmt.Sscanf(blockResp.Block.Header.Height, "%d", &blockHeight); err != nil {
		return nil, fmt.Errorf("failed to parse block height: %w", err)
	}

	return &BlockInfo{
		Height:  blockHeight,
		Hash:    blockResp.Block.Header.Hash,
		Time:    blockResp.Block.Header.Time,
		TxCount: len(blockResp.Block.Data.Txs),
	}, nil
}

// GetGovernanceProposals returns a list of governance proposals
func (c *WBFTClient) GetGovernanceProposals(status ProposalStatus) ([]*Proposal, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	params := map[string]interface{}{
		"proposal_status": string(status),
	}

	resp, err := c.makeRequest(ctx, "gov/proposals", params)
	if err != nil {
		return nil, fmt.Errorf("failed to get proposals: %w", err)
	}

	var proposalsResp struct {
		Proposals []struct {
			ProposalID   string `json:"proposal_id"`
			Content      struct {
				Type        string `json:"@type"`
				Title       string `json:"title"`
				Description string `json:"description"`
				Plan        struct {
					Name   string `json:"name"`
					Height string `json:"height"`
					Info   string `json:"info"`
				} `json:"plan,omitempty"`
			} `json:"content"`
			Status           string    `json:"status"`
			FinalTallyResult struct {
				Yes        string `json:"yes"`
				Abstain    string `json:"abstain"`
				No         string `json:"no"`
				NoWithVeto string `json:"no_with_veto"`
			} `json:"final_tally_result"`
			SubmitTime      time.Time `json:"submit_time"`
			VotingStartTime time.Time `json:"voting_start_time"`
			VotingEndTime   time.Time `json:"voting_end_time"`
		} `json:"proposals"`
	}

	if err := json.Unmarshal(resp.Result, &proposalsResp); err != nil {
		return nil, fmt.Errorf("failed to parse proposals response: %w", err)
	}

	proposals := make([]*Proposal, 0, len(proposalsResp.Proposals))
	for _, p := range proposalsResp.Proposals {
		proposal := &Proposal{
			ID:            p.ProposalID,
			Title:         p.Content.Title,
			Description:   p.Content.Description,
			Status:        ProposalStatus(p.Status),
			SubmitTime:    p.SubmitTime,
			VotingEndTime: p.VotingEndTime,
		}

		// Determine proposal type based on content type
		switch p.Content.Type {
		case "/cosmos.upgrade.v1beta1.SoftwareUpgradeProposal":
			proposal.Type = ProposalTypeUpgrade
			if p.Content.Plan.Height != "" {
				var height int64
				if _, err := fmt.Sscanf(p.Content.Plan.Height, "%d", &height); err == nil {
					proposal.UpgradeHeight = height
				}
			}
			proposal.UpgradeInfo = &UpgradeInfo{
				Name:   p.Content.Plan.Name,
				Height: proposal.UpgradeHeight,
				Info:   p.Content.Plan.Info,
				Status: UpgradeStatusScheduled,
			}
		case "/cosmos.params.v1beta1.ParameterChangeProposal":
			proposal.Type = ProposalTypeParameter
		case "/cosmos.gov.v1beta1.TextProposal":
			proposal.Type = ProposalTypeText
		default:
			proposal.Type = ProposalTypeCommunity
		}

		// Parse voting statistics
		proposal.VotingStats = &VotingStats{
			QuorumReached: true, // This should be calculated based on actual parameters
			ThresholdMet:  proposal.Status == ProposalStatusPassed,
		}

		proposals = append(proposals, proposal)
	}

	return proposals, nil
}

// GetProposal returns a specific governance proposal
func (c *WBFTClient) GetProposal(proposalID string) (*Proposal, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	params := map[string]interface{}{
		"proposal_id": proposalID,
	}

	resp, err := c.makeRequest(ctx, "gov/proposal", params)
	if err != nil {
		return nil, fmt.Errorf("failed to get proposal: %w", err)
	}

	// Parse single proposal response (similar to GetGovernanceProposals but for one)
	var proposalResp struct {
		Proposal struct {
			ProposalID string `json:"proposal_id"`
			Content    struct {
				Type        string `json:"@type"`
				Title       string `json:"title"`
				Description string `json:"description"`
			} `json:"content"`
			Status        string    `json:"status"`
			SubmitTime    time.Time `json:"submit_time"`
			VotingEndTime time.Time `json:"voting_end_time"`
		} `json:"proposal"`
	}

	if err := json.Unmarshal(resp.Result, &proposalResp); err != nil {
		return nil, fmt.Errorf("failed to parse proposal response: %w", err)
	}

	return &Proposal{
		ID:            proposalResp.Proposal.ProposalID,
		Title:         proposalResp.Proposal.Content.Title,
		Description:   proposalResp.Proposal.Content.Description,
		Status:        ProposalStatus(proposalResp.Proposal.Status),
		SubmitTime:    proposalResp.Proposal.SubmitTime,
		VotingEndTime: proposalResp.Proposal.VotingEndTime,
	}, nil
}

// GetValidators returns a list of validators
func (c *WBFTClient) GetValidators() ([]*ValidatorInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	resp, err := c.makeRequest(ctx, "validators", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get validators: %w", err)
	}

	var validatorsResp struct {
		Validators []struct {
			Address     string `json:"address"`
			PubKey      string `json:"pub_key"`
			VotingPower string `json:"voting_power"`
		} `json:"validators"`
	}

	if err := json.Unmarshal(resp.Result, &validatorsResp); err != nil {
		return nil, fmt.Errorf("failed to parse validators response: %w", err)
	}

	validators := make([]*ValidatorInfo, 0, len(validatorsResp.Validators))
	for _, v := range validatorsResp.Validators {
		var votingPower int64
		if _, err := fmt.Sscanf(v.VotingPower, "%d", &votingPower); err != nil {
			votingPower = 0
		}

		validators = append(validators, &ValidatorInfo{
			OperatorAddress: v.Address,
			ConsensusPubkey: v.PubKey,
			VotingPower:     votingPower,
			Status:          "bonded", // Default status
		})
	}

	return validators, nil
}

// GetGovernanceParams returns governance parameters
func (c *WBFTClient) GetGovernanceParams() (*GovernanceParams, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	resp, err := c.makeRequest(ctx, "gov/params", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get governance params: %w", err)
	}

	var paramsResp struct {
		VotingParams struct {
			VotingPeriod string `json:"voting_period"`
		} `json:"voting_params"`
		DepositParams struct {
			MinDeposit []struct {
				Denom  string `json:"denom"`
				Amount string `json:"amount"`
			} `json:"min_deposit"`
		} `json:"deposit_params"`
		TallyParams struct {
			Quorum    string `json:"quorum"`
			Threshold string `json:"threshold"`
			Veto      string `json:"veto_threshold"`
		} `json:"tally_params"`
	}

	if err := json.Unmarshal(resp.Result, &paramsResp); err != nil {
		return nil, fmt.Errorf("failed to parse governance params response: %w", err)
	}

	// Parse voting period duration
	votingPeriod, err := time.ParseDuration(paramsResp.VotingParams.VotingPeriod)
	if err != nil {
		votingPeriod = 14 * 24 * time.Hour // Default to 14 days
	}

	minDeposit := ""
	if len(paramsResp.DepositParams.MinDeposit) > 0 {
		minDeposit = paramsResp.DepositParams.MinDeposit[0].Amount
	}

	return &GovernanceParams{
		VotingPeriod:      votingPeriod,
		MinDeposit:        minDeposit,
		QuorumThreshold:   paramsResp.TallyParams.Quorum,
		PassThreshold:     paramsResp.TallyParams.Threshold,
		VetoThreshold:     paramsResp.TallyParams.Veto,
		MinUpgradeDelay:   24 * time.Hour, // Default minimum delay
		EmergencyVotePeriod: 24 * time.Hour, // Default emergency voting period
	}, nil
}

// makeRequest makes a JSON-RPC request to the WBFT node
func (c *WBFTClient) makeRequest(ctx context.Context, method string, params interface{}) (*RPCResponse, error) {
	req := &RPCRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
		ID:      1,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	c.logger.Debug("Making RPC request", zap.String("method", method), zap.String("url", c.baseURL))

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to make HTTP request: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %d %s", httpResp.StatusCode, httpResp.Status)
	}

	var rpcResp RPCResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&rpcResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if rpcResp.Error != nil {
		return nil, fmt.Errorf("RPC error: %d - %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	return &rpcResp, nil
}

// SetTimeout sets the request timeout for the client
func (c *WBFTClient) SetTimeout(timeout time.Duration) {
	c.timeout = timeout
	c.httpClient.Timeout = timeout
}