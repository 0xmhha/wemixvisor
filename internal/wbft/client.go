package wbft

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/wemix/wemixvisor/internal/config"
	"github.com/wemix/wemixvisor/pkg/logger"
	"go.uber.org/zap"
)

// Client provides WBFT consensus interaction
type Client struct {
	cfg        *config.Config
	logger     *logger.Logger
	httpClient *http.Client
	rpcURL     string
}

// NewClient creates a new WBFT client
func NewClient(cfg *config.Config, logger *logger.Logger) *Client {
	rpcURL := cfg.RPCAddress
	if !strings.HasPrefix(rpcURL, "http://") && !strings.HasPrefix(rpcURL, "https://") {
		rpcURL = fmt.Sprintf("http://%s", cfg.RPCAddress)
	}

	return &Client{
		cfg:    cfg,
		logger: logger,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		rpcURL: rpcURL,
	}
}

// ConsensusState represents the current consensus state
type ConsensusState struct {
	Height          int64    `json:"height"`
	Round           int      `json:"round"`
	Step            string   `json:"step"`
	ValidatorIndex  int      `json:"validator_index"`
	IsValidator     bool     `json:"is_validator"`
	IsSyncing       bool     `json:"is_syncing"`
	CatchingUp      bool     `json:"catching_up"`
	ValidatorCount  int      `json:"validator_count"`
	ValidatorPower  int64    `json:"validator_power"`
	TotalPower      int64    `json:"total_power"`
	Validators      []string `json:"validators"`
}

// NodeStatus represents the node's status
type NodeStatus struct {
	NodeID        string    `json:"node_id"`
	ListenAddress string    `json:"listen_addr"`
	Network       string    `json:"network"`
	Version       string    `json:"version"`
	Channels      string    `json:"channels"`
	Moniker       string    `json:"moniker"`
	SyncInfo      SyncInfo  `json:"sync_info"`
	ValidatorInfo Validator `json:"validator_info"`
}

// SyncInfo represents synchronization information
type SyncInfo struct {
	LatestBlockHash     string    `json:"latest_block_hash"`
	LatestAppHash       string    `json:"latest_app_hash"`
	LatestBlockHeight   int64     `json:"latest_block_height"`
	LatestBlockTime     time.Time `json:"latest_block_time"`
	EarliestBlockHash   string    `json:"earliest_block_hash"`
	EarliestAppHash     string    `json:"earliest_app_hash"`
	EarliestBlockHeight int64     `json:"earliest_block_height"`
	EarliestBlockTime   time.Time `json:"earliest_block_time"`
	CatchingUp          bool      `json:"catching_up"`
}

// Validator represents validator information
type Validator struct {
	Address     string `json:"address"`
	PubKey      string `json:"pub_key"`
	VotingPower int64  `json:"voting_power"`
}

// GetConsensusState retrieves the current consensus state
func (c *Client) GetConsensusState(ctx context.Context) (*ConsensusState, error) {
	// Get node status
	status, err := c.getStatus(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get node status: %w", err)
	}

	// Get consensus state
	consensusResp, err := c.rpcCall(ctx, "consensus_state", nil)
	if err != nil {
		// Fallback to basic state from status
		return &ConsensusState{
			Height:     status.SyncInfo.LatestBlockHeight,
			IsSyncing:  status.SyncInfo.CatchingUp,
			CatchingUp: status.SyncInfo.CatchingUp,
		}, nil
	}

	var state ConsensusState
	if err := json.Unmarshal(consensusResp, &state); err != nil {
		// Fallback to basic state
		return &ConsensusState{
			Height:     status.SyncInfo.LatestBlockHeight,
			IsSyncing:  status.SyncInfo.CatchingUp,
			CatchingUp: status.SyncInfo.CatchingUp,
		}, nil
	}

	// Update with status info
	state.Height = status.SyncInfo.LatestBlockHeight
	state.CatchingUp = status.SyncInfo.CatchingUp

	return &state, nil
}

// GetCurrentHeight returns the current block height
func (c *Client) GetCurrentHeight(ctx context.Context) (int64, error) {
	status, err := c.getStatus(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get status: %w", err)
	}

	return status.SyncInfo.LatestBlockHeight, nil
}

// IsValidator checks if the node is a validator
func (c *Client) IsValidator(ctx context.Context) (bool, error) {
	if !c.cfg.ValidatorMode {
		return false, nil
	}

	status, err := c.getStatus(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get status: %w", err)
	}

	// Check if node has voting power
	return status.ValidatorInfo.VotingPower > 0, nil
}

// WaitForHeight waits until the node reaches a specific height
func (c *Client) WaitForHeight(ctx context.Context, targetHeight int64, timeout time.Duration) error {
	c.logger.Info("waiting for block height",
		zap.Int64("target", targetHeight),
		zap.Duration("timeout", timeout))

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	startTime := time.Now()
	lastLogTime := startTime

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for height %d", targetHeight)
		case <-ticker.C:
			currentHeight, err := c.GetCurrentHeight(ctx)
			if err != nil {
				c.logger.Warn("failed to get current height",
					zap.Error(err))
				continue
			}

			if currentHeight >= targetHeight {
				c.logger.Info("reached target height",
					zap.Int64("height", currentHeight),
					zap.Duration("elapsed", time.Since(startTime)))
				return nil
			}

			// Log progress every 10 seconds
			if time.Since(lastLogTime) >= 10*time.Second {
				blocksRemaining := targetHeight - currentHeight
				c.logger.Info("waiting for height",
					zap.Int64("current", currentHeight),
					zap.Int64("target", targetHeight),
					zap.Int64("remaining", blocksRemaining))
				lastLogTime = time.Now()
			}
		}
	}
}

// MonitorConsensus monitors consensus state changes
func (c *Client) MonitorConsensus(ctx context.Context, interval time.Duration) <-chan *ConsensusState {
	stateChan := make(chan *ConsensusState)

	go func() {
		defer close(stateChan)

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		var lastState *ConsensusState

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				state, err := c.GetConsensusState(ctx)
				if err != nil {
					c.logger.Warn("failed to get consensus state",
						zap.Error(err))
					continue
				}

				// Send state if it has changed
				if lastState == nil || hasStateChanged(lastState, state) {
					select {
					case stateChan <- state:
						lastState = state
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()

	return stateChan
}

// CheckReadiness checks if the node is ready for upgrade
func (c *Client) CheckReadiness(ctx context.Context) error {
	state, err := c.GetConsensusState(ctx)
	if err != nil {
		return fmt.Errorf("failed to get consensus state: %w", err)
	}

	if state.IsSyncing || state.CatchingUp {
		return fmt.Errorf("node is still syncing")
	}

	// For validators, check if participating in consensus
	if c.cfg.ValidatorMode {
		isValidator, err := c.IsValidator(ctx)
		if err != nil {
			return fmt.Errorf("failed to check validator status: %w", err)
		}

		if !isValidator {
			c.logger.Warn("node is configured as validator but not in validator set")
		}
	}

	c.logger.Info("node is ready for upgrade",
		zap.Int64("height", state.Height),
		zap.Bool("is_validator", state.IsValidator))

	return nil
}

// getStatus retrieves node status via RPC
func (c *Client) getStatus(ctx context.Context) (*NodeStatus, error) {
	resp, err := c.rpcCall(ctx, "status", nil)
	if err != nil {
		return nil, err
	}

	var status NodeStatus
	if err := json.Unmarshal(resp, &status); err != nil {
		return nil, fmt.Errorf("failed to unmarshal status: %w", err)
	}

	return &status, nil
}

// rpcCall makes an RPC call to the node
func (c *Client) rpcCall(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
	// Construct JSON-RPC request
	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  method,
		"params":  params,
	}

	reqData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", c.rpcURL, strings.NewReader(string(reqData)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("RPC request failed: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var rpcResp struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&rpcResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Check for RPC error
	if rpcResp.Error != nil {
		return nil, fmt.Errorf("RPC error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	return rpcResp.Result, nil
}

// hasStateChanged checks if consensus state has changed significantly
func hasStateChanged(old, new *ConsensusState) bool {
	if old.Height != new.Height {
		return true
	}
	if old.Round != new.Round {
		return true
	}
	if old.Step != new.Step {
		return true
	}
	if old.IsSyncing != new.IsSyncing {
		return true
	}
	if old.IsValidator != new.IsValidator {
		return true
	}
	return false
}

// GetBlockNumber returns the current block number (alias for GetCurrentHeight)
func (c *Client) GetBlockNumber(ctx context.Context) (*big.Int, error) {
	height, err := c.GetCurrentHeight(ctx)
	if err != nil {
		return nil, err
	}
	return big.NewInt(height), nil
}