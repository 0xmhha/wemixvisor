package node

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatus_MarshalJSON(t *testing.T) {
	startTime := time.Now()
	status := Status{
		State:        StateRunning,
		PID:          12345,
		StartTime:    startTime,
		Uptime:       30 * time.Minute,
		RestartCount: 2,
		Version:      "v1.0.0",
		Network:      "mainnet",
		Binary:       "/usr/bin/wemixd",
	}

	data, err := json.Marshal(status)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	// Check that the state_string field is present
	assert.Equal(t, "running", result["state_string"])
	assert.Equal(t, float64(12345), result["pid"])
	assert.Equal(t, float64(2), result["restart_count"])
	assert.Equal(t, "v1.0.0", result["version"])
	assert.Equal(t, "mainnet", result["network"])
	assert.Equal(t, "/usr/bin/wemixd", result["binary"])
	assert.NotEmpty(t, result["uptime_string"])
}

func TestHealthStatus(t *testing.T) {
	health := HealthStatus{
		Healthy:   true,
		Timestamp: time.Now(),
		Checks: map[string]CheckResult{
			"rpc": {
				Name:    "rpc",
				Healthy: true,
				Latency: 12,
			},
			"peers": {
				Name:    "peers",
				Healthy: true,
				Details: map[string]interface{}{
					"peer_count": 8,
					"min_peers":  1,
				},
			},
			"sync": {
				Name:    "sync",
				Healthy: false,
				Error:   "node is syncing",
				Details: map[string]interface{}{
					"current_block": 1000,
					"highest_block": 2000,
				},
			},
		},
	}

	// Test JSON marshaling
	data, err := json.Marshal(health)
	require.NoError(t, err)

	var result HealthStatus
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	assert.Equal(t, health.Healthy, result.Healthy)
	assert.Len(t, result.Checks, 3)

	// Check individual checks
	assert.True(t, result.Checks["rpc"].Healthy)
	assert.Equal(t, int64(12), result.Checks["rpc"].Latency)

	assert.True(t, result.Checks["peers"].Healthy)
	assert.Equal(t, float64(8), result.Checks["peers"].Details["peer_count"])

	assert.False(t, result.Checks["sync"].Healthy)
	assert.Equal(t, "node is syncing", result.Checks["sync"].Error)
}

func TestCheckResult(t *testing.T) {
	tests := []struct {
		name   string
		result CheckResult
		want   bool
	}{
		{
			name: "healthy check",
			result: CheckResult{
				Name:    "test",
				Healthy: true,
				Latency: 10,
			},
			want: true,
		},
		{
			name: "unhealthy check",
			result: CheckResult{
				Name:    "test",
				Healthy: false,
				Error:   "connection refused",
			},
			want: false,
		},
		{
			name: "check with details",
			result: CheckResult{
				Name:    "test",
				Healthy: true,
				Details: map[string]interface{}{
					"key": "value",
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.result.Healthy)

			// Test JSON marshaling
			data, err := json.Marshal(tt.result)
			require.NoError(t, err)

			var unmarshaled CheckResult
			err = json.Unmarshal(data, &unmarshaled)
			require.NoError(t, err)

			assert.Equal(t, tt.result.Name, unmarshaled.Name)
			assert.Equal(t, tt.result.Healthy, unmarshaled.Healthy)

			if tt.result.Error != "" {
				assert.Equal(t, tt.result.Error, unmarshaled.Error)
			}

			if tt.result.Latency > 0 {
				assert.Equal(t, tt.result.Latency, unmarshaled.Latency)
			}
		})
	}
}