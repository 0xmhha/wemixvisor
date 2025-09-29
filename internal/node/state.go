package node

import (
	"encoding/json"
	"time"
)

// NodeState represents the current state of the node
type NodeState int

const (
	// StateStopped indicates the node is not running
	StateStopped NodeState = iota
	// StateStarting indicates the node is starting up
	StateStarting
	// StateRunning indicates the node is running normally
	StateRunning
	// StateStopping indicates the node is shutting down
	StateStopping
	// StateUpgrading indicates the node is being upgraded
	StateUpgrading
	// StateError indicates the node is in error state
	StateError
	// StateCrashed indicates the node crashed unexpectedly
	StateCrashed
)

// String returns the string representation of NodeState
func (s NodeState) String() string {
	switch s {
	case StateStopped:
		return "stopped"
	case StateStarting:
		return "starting"
	case StateRunning:
		return "running"
	case StateStopping:
		return "stopping"
	case StateUpgrading:
		return "upgrading"
	case StateError:
		return "error"
	case StateCrashed:
		return "crashed"
	default:
		return "unknown"
	}
}

// Status represents the current status of the node
type Status struct {
	State        NodeState     `json:"state"`
	StateString  string        `json:"state_string"`
	PID          int           `json:"pid"`
	StartTime    time.Time     `json:"start_time"`
	Uptime       time.Duration `json:"uptime"`
	RestartCount int           `json:"restart_count"`
	Version      string        `json:"version"`
	Network      string        `json:"network"`
	Binary       string        `json:"binary"`
	Health       *HealthStatus `json:"health,omitempty"`
}

// MarshalJSON implements json.Marshaler
func (s Status) MarshalJSON() ([]byte, error) {
	type Alias Status
	return json.Marshal(&struct {
		*Alias
		StateString string `json:"state_string"`
		UptimeStr   string `json:"uptime_string"`
	}{
		Alias:       (*Alias)(&s),
		StateString: s.State.String(),
		UptimeStr:   s.Uptime.String(),
	})
}

// HealthStatus represents the health status of the node
type HealthStatus struct {
	Healthy   bool                   `json:"healthy"`
	Timestamp time.Time              `json:"timestamp"`
	Checks    map[string]CheckResult `json:"checks"`
}

// CheckResult represents the result of a single health check
type CheckResult struct {
	Name    string `json:"name"`
	Healthy bool   `json:"healthy"`
	Error   string `json:"error,omitempty"`
	Latency int64  `json:"latency_ms,omitempty"`
	Details map[string]interface{} `json:"details,omitempty"`
}