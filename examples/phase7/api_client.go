package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// APIClient is a client for the Wemixvisor API
type APIClient struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

// NewAPIClient creates a new API client
func NewAPIClient(baseURL, apiKey string) *APIClient {
	return &APIClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// makeRequest makes an HTTP request to the API
func (c *APIClient) makeRequest(method, path string, body interface{}) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, c.baseURL+path, reqBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-API-Key", c.apiKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error: %s", string(respBody))
	}

	return respBody, nil
}

// GetStatus gets the system status
func (c *APIClient) GetStatus() (map[string]interface{}, error) {
	resp, err := c.makeRequest("GET", "/api/v1/status", nil)
	if err != nil {
		return nil, err
	}

	var status map[string]interface{}
	err = json.Unmarshal(resp, &status)
	return status, err
}

// GetMetrics gets current metrics
func (c *APIClient) GetMetrics() (map[string]interface{}, error) {
	resp, err := c.makeRequest("GET", "/api/v1/metrics", nil)
	if err != nil {
		return nil, err
	}

	var metrics map[string]interface{}
	err = json.Unmarshal(resp, &metrics)
	return metrics, err
}

// GetUpgrades lists all upgrades
func (c *APIClient) GetUpgrades() ([]map[string]interface{}, error) {
	resp, err := c.makeRequest("GET", "/api/v1/upgrades", nil)
	if err != nil {
		return nil, err
	}

	var upgrades []map[string]interface{}
	err = json.Unmarshal(resp, &upgrades)
	return upgrades, err
}

// ScheduleUpgrade schedules a new upgrade
func (c *APIClient) ScheduleUpgrade(name string, height int64) error {
	upgrade := map[string]interface{}{
		"name":   name,
		"height": height,
	}

	_, err := c.makeRequest("POST", "/api/v1/upgrades", upgrade)
	return err
}

// GetProposals gets governance proposals
func (c *APIClient) GetProposals() ([]map[string]interface{}, error) {
	resp, err := c.makeRequest("GET", "/api/v1/governance/proposals", nil)
	if err != nil {
		return nil, err
	}

	var proposals []map[string]interface{}
	err = json.Unmarshal(resp, &proposals)
	return proposals, err
}

// VoteOnProposal votes on a governance proposal
func (c *APIClient) VoteOnProposal(proposalID string, vote string) error {
	voteReq := map[string]interface{}{
		"proposal_id": proposalID,
		"vote":        vote,
	}

	_, err := c.makeRequest("POST", "/api/v1/governance/vote", voteReq)
	return err
}

// GetAlerts gets active alerts
func (c *APIClient) GetAlerts() ([]map[string]interface{}, error) {
	resp, err := c.makeRequest("GET", "/api/v1/alerts", nil)
	if err != nil {
		return nil, err
	}

	var alerts []map[string]interface{}
	err = json.Unmarshal(resp, &alerts)
	return alerts, err
}

// TestAlertChannel tests an alert channel
func (c *APIClient) TestAlertChannel(channel string) error {
	testReq := map[string]interface{}{
		"channel": channel,
	}

	_, err := c.makeRequest("POST", "/api/v1/alerts/test", testReq)
	return err
}

func main() {
	// Create API client
	client := NewAPIClient("http://localhost:8080", "your-api-key")

	// Example 1: Get system status
	fmt.Println("=== System Status ===")
	status, err := client.GetStatus()
	if err != nil {
		fmt.Printf("Error getting status: %v\n", err)
	} else {
		fmt.Printf("Status: %v\n", status)
	}

	// Example 2: Get metrics
	fmt.Println("\n=== Metrics ===")
	metrics, err := client.GetMetrics()
	if err != nil {
		fmt.Printf("Error getting metrics: %v\n", err)
	} else {
		for key, value := range metrics {
			fmt.Printf("%s: %v\n", key, value)
		}
	}

	// Example 3: List upgrades
	fmt.Println("\n=== Upgrades ===")
	upgrades, err := client.GetUpgrades()
	if err != nil {
		fmt.Printf("Error getting upgrades: %v\n", err)
	} else {
		for _, upgrade := range upgrades {
			fmt.Printf("Upgrade: %v\n", upgrade)
		}
	}

	// Example 4: Get proposals
	fmt.Println("\n=== Governance Proposals ===")
	proposals, err := client.GetProposals()
	if err != nil {
		fmt.Printf("Error getting proposals: %v\n", err)
	} else {
		for _, proposal := range proposals {
			fmt.Printf("Proposal: %v\n", proposal)
		}
	}

	// Example 5: Get active alerts
	fmt.Println("\n=== Active Alerts ===")
	alerts, err := client.GetAlerts()
	if err != nil {
		fmt.Printf("Error getting alerts: %v\n", err)
	} else {
		if len(alerts) == 0 {
			fmt.Println("No active alerts")
		} else {
			for _, alert := range alerts {
				fmt.Printf("Alert: %v\n", alert)
			}
		}
	}

	// Example 6: Schedule an upgrade (commented out to avoid accidental execution)
	// err = client.ScheduleUpgrade("v1.2.0", 1000000)
	// if err != nil {
	//     fmt.Printf("Error scheduling upgrade: %v\n", err)
	// } else {
	//     fmt.Println("Upgrade scheduled successfully")
	// }

	// Example 7: Vote on proposal (commented out to avoid accidental execution)
	// err = client.VoteOnProposal("1", "yes")
	// if err != nil {
	//     fmt.Printf("Error voting: %v\n", err)
	// } else {
	//     fmt.Println("Vote cast successfully")
	// }
}