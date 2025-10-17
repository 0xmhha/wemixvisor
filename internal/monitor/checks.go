package monitor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"syscall"
	"time"
)

// ProcessCheck checks if the node process is running
type ProcessCheck struct {
	pidFile string
}

func (c *ProcessCheck) Name() string {
	return "process"
}

func (c *ProcessCheck) Check(ctx context.Context) error {
	// If PID file is specified, check it
	if c.pidFile != "" {
		pidBytes, err := os.ReadFile(c.pidFile)
		if err != nil {
			return fmt.Errorf("cannot read PID file: %w", err)
		}

		var pid int
		if _, err := fmt.Sscanf(string(pidBytes), "%d", &pid); err != nil {
			return fmt.Errorf("invalid PID in file: %w", err)
		}

		// Check if process exists
		process, err := os.FindProcess(pid)
		if err != nil {
			return fmt.Errorf("process not found: %w", err)
		}

		// Send signal 0 to check if process is alive
		if err := process.Signal(syscall.Signal(0)); err != nil {
			return fmt.Errorf("process not responding: %w", err)
		}

		return nil
	}

	// If no PID file, this check always passes (assumes external management)
	return nil
}

// RPCHealthCheck checks if the RPC endpoint is responsive
type RPCHealthCheck struct {
	url string
}

func (c *RPCHealthCheck) Name() string {
	return "rpc_endpoint"
}

func (c *RPCHealthCheck) Check(ctx context.Context) error {
	// Prepare JSON-RPC request for web3_clientVersion
	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "web3_clientVersion",
		"params":  []interface{}{},
		"id":      1,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, "POST", c.url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("RPC endpoint unreachable: %w", err)
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("RPC endpoint returned status %d", resp.StatusCode)
	}

	// Parse response
	var result map[string]interface{}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for error in response
	if errObj, exists := result["error"]; exists {
		return fmt.Errorf("RPC error: %v", errObj)
	}

	return nil
}

// PeerCountCheck checks if the node has minimum peers
type PeerCountCheck struct {
	minPeers int
	rpcURL   string
}

func (c *PeerCountCheck) Name() string {
	return "peer_count"
}

func (c *PeerCountCheck) Check(ctx context.Context) error {
	// Prepare JSON-RPC request for net_peerCount
	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "net_peerCount",
		"params":  []interface{}{},
		"id":      1,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, "POST", c.rpcURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to check peer count: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var result map[string]interface{}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract peer count
	if resultValue, ok := result["result"]; ok {
		peerCountHex, ok := resultValue.(string)
		if !ok {
			return fmt.Errorf("unexpected peer count format")
		}

		// Convert hex string to int
		var peerCount int
		if _, err := fmt.Sscanf(peerCountHex, "0x%x", &peerCount); err != nil {
			// Try without 0x prefix
			if _, err := fmt.Sscanf(peerCountHex, "%x", &peerCount); err != nil {
				return fmt.Errorf("failed to parse peer count: %w", err)
			}
		}

		if peerCount < c.minPeers {
			return fmt.Errorf("insufficient peers: %d < %d", peerCount, c.minPeers)
		}

		return nil
	}

	return fmt.Errorf("no peer count in response")
}

// SyncingCheck checks if the node is syncing
type SyncingCheck struct {
	rpcURL string
}

func (c *SyncingCheck) Name() string {
	return "syncing"
}

func (c *SyncingCheck) Check(ctx context.Context) error {
	// Prepare JSON-RPC request for eth_syncing
	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "eth_syncing",
		"params":  []interface{}{},
		"id":      1,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, "POST", c.rpcURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to check sync status: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var result map[string]interface{}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Check sync status
	if resultValue, ok := result["result"]; ok {
		// If result is false, node is synced
		if syncing, ok := resultValue.(bool); ok && !syncing {
			return nil // Node is synced
		}

		// If result is an object, node is syncing
		if syncData, ok := resultValue.(map[string]interface{}); ok {
			// Extract sync progress
			currentBlock := syncData["currentBlock"]
			highestBlock := syncData["highestBlock"]
			return fmt.Errorf("node is syncing (current: %v, highest: %v)", currentBlock, highestBlock)
		}

		// Syncing but no details
		return fmt.Errorf("node is syncing")
	}

	return fmt.Errorf("unable to determine sync status")
}

// MemoryCheck checks memory usage
type MemoryCheck struct {
	maxMemoryMB int64
}

func (c *MemoryCheck) Name() string {
	return "memory"
}

func (c *MemoryCheck) Check(ctx context.Context) error {
	// This is a placeholder - in production, you would check actual memory usage
	// For now, we'll just pass the check
	return nil
}

// DiskSpaceCheck checks available disk space
type DiskSpaceCheck struct {
	minSpaceGB int64
	dataDir    string
}

func (c *DiskSpaceCheck) Name() string {
	return "disk_space"
}

func (c *DiskSpaceCheck) Check(ctx context.Context) error {
	// Get filesystem stats
	var stat syscall.Statfs_t
	if err := syscall.Statfs(c.dataDir, &stat); err != nil {
		return fmt.Errorf("failed to get disk stats: %w", err)
	}

	// Calculate available space in GB
	availableGB := (stat.Bavail * uint64(stat.Bsize)) / (1024 * 1024 * 1024)

	if int64(availableGB) < c.minSpaceGB {
		return fmt.Errorf("insufficient disk space: %d GB < %d GB", availableGB, c.minSpaceGB)
	}

	return nil
}