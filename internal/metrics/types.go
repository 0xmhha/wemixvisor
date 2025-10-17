// Package metrics provides metrics collection and export functionality
package metrics

import (
	"time"
)

// MetricType represents the type of metric
type MetricType string

const (
	// MetricTypeGauge represents a gauge metric
	MetricTypeGauge MetricType = "gauge"
	// MetricTypeCounter represents a counter metric
	MetricTypeCounter MetricType = "counter"
	// MetricTypeHistogram represents a histogram metric
	MetricTypeHistogram MetricType = "histogram"
	// MetricTypeSummary represents a summary metric
	MetricTypeSummary MetricType = "summary"
)

// MetricCategory represents the category of metric
type MetricCategory string

const (
	// CategorySystem represents system-level metrics
	CategorySystem MetricCategory = "system"
	// CategoryApplication represents application-level metrics
	CategoryApplication MetricCategory = "application"
	// CategoryBusiness represents business-level metrics
	CategoryBusiness MetricCategory = "business"
	// CategoryGovernance represents governance-related metrics
	CategoryGovernance MetricCategory = "governance"
	// CategoryPerformance represents performance metrics
	CategoryPerformance MetricCategory = "performance"
)

// Metric represents a single metric data point
type Metric struct {
	Name        string            `json:"name"`
	Type        MetricType        `json:"type"`
	Category    MetricCategory    `json:"category"`
	Value       float64           `json:"value"`
	Labels      map[string]string `json:"labels,omitempty"`
	Timestamp   time.Time         `json:"timestamp"`
	Description string            `json:"description,omitempty"`
}

// SystemMetrics holds system-level metrics
type SystemMetrics struct {
	CPUUsage         float64   `json:"cpu_usage"`
	MemoryUsage      float64   `json:"memory_usage"`
	MemoryTotal      uint64    `json:"memory_total"`
	MemoryAvailable  uint64    `json:"memory_available"`
	DiskUsage        float64   `json:"disk_usage"`
	DiskTotal        uint64    `json:"disk_total"`
	DiskAvailable    uint64    `json:"disk_available"`
	NetworkRxBytes   uint64    `json:"network_rx_bytes"`
	NetworkTxBytes   uint64    `json:"network_tx_bytes"`
	NetworkRxPackets uint64    `json:"network_rx_packets"`
	NetworkTxPackets uint64    `json:"network_tx_packets"`
	Goroutines       int       `json:"goroutines"`
	Uptime           int64     `json:"uptime"`
	Timestamp        time.Time `json:"timestamp"`
}

// ApplicationMetrics holds application-level metrics
type ApplicationMetrics struct {
	// Upgrade metrics
	UpgradeTotal      int64 `json:"upgrade_total"`
	UpgradeSuccess    int64 `json:"upgrade_success"`
	UpgradeFailed     int64 `json:"upgrade_failed"`
	UpgradePending    int64 `json:"upgrade_pending"`
	LastUpgradeTime   int64 `json:"last_upgrade_time"`
	LastUpgradeHeight int64 `json:"last_upgrade_height"`

	// Process metrics
	ProcessRestarts   int64  `json:"process_restarts"`
	ProcessUptime     int64  `json:"process_uptime"`
	ProcessStatus     string `json:"process_status"`
	ProcessPID        int    `json:"process_pid"`

	// Node metrics
	NodeSyncing       bool   `json:"node_syncing"`
	NodeHeight        int64  `json:"node_height"`
	NodePeers         int    `json:"node_peers"`
	NodeVersion       string `json:"node_version"`
	NodeLatestVersion string `json:"node_latest_version"`

	Timestamp time.Time `json:"timestamp"`
}

// GovernanceMetrics holds governance-related metrics
type GovernanceMetrics struct {
	// Proposal metrics
	ProposalTotal     int64   `json:"proposal_total"`
	ProposalVoting    int64   `json:"proposal_voting"`
	ProposalPassed    int64   `json:"proposal_passed"`
	ProposalRejected  int64   `json:"proposal_rejected"`
	ProposalExecuted  int64   `json:"proposal_executed"`

	// Voting metrics
	VotingPower       float64 `json:"voting_power"`
	VotingTurnout     float64 `json:"voting_turnout"`
	VotesTotal        int64   `json:"votes_total"`
	VotesParticipated int64   `json:"votes_participated"`

	// Validator metrics
	ValidatorActive   int64   `json:"validator_active"`
	ValidatorJailed   int64   `json:"validator_jailed"`
	ValidatorTombstoned int64 `json:"validator_tombstoned"`
	ValidatorBonded   int64   `json:"validator_bonded"`

	Timestamp time.Time `json:"timestamp"`
}

// PerformanceMetrics holds performance-related metrics
type PerformanceMetrics struct {
	// Latency metrics (in milliseconds)
	RPCLatency      float64 `json:"rpc_latency"`
	APILatency      float64 `json:"api_latency"`
	DatabaseLatency float64 `json:"database_latency"`
	NetworkLatency  float64 `json:"network_latency"`

	// Throughput metrics
	TransactionsPerSecond float64 `json:"transactions_per_second"`
	BlocksPerSecond       float64 `json:"blocks_per_second"`
	RequestsPerSecond     float64 `json:"requests_per_second"`

	// Resource efficiency
	CPUEfficiency    float64 `json:"cpu_efficiency"`
	MemoryEfficiency float64 `json:"memory_efficiency"`
	CacheHitRate     float64 `json:"cache_hit_rate"`
	CacheMissRate    float64 `json:"cache_miss_rate"`

	Timestamp time.Time `json:"timestamp"`
}

// AlertLevel represents the severity level of an alert
type AlertLevel string

const (
	// AlertLevelInfo represents informational alerts
	AlertLevelInfo AlertLevel = "info"
	// AlertLevelWarning represents warning alerts
	AlertLevelWarning AlertLevel = "warning"
	// AlertLevelError represents error alerts
	AlertLevelError AlertLevel = "error"
	// AlertLevelCritical represents critical alerts
	AlertLevelCritical AlertLevel = "critical"
)

// Alert represents an alert generated by the monitoring system
type Alert struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Level       AlertLevel        `json:"level"`
	Message     string            `json:"message"`
	Description string            `json:"description,omitempty"`
	Source      string            `json:"source"`
	Metric      string            `json:"metric"`
	Value       float64           `json:"value"`
	Threshold   float64           `json:"threshold"`
	Labels      map[string]string `json:"labels,omitempty"`
	Timestamp   time.Time         `json:"timestamp"`
	ResolvedAt  *time.Time        `json:"resolved_at,omitempty"`
}

// MetricsSnapshot represents a complete snapshot of all metrics
type MetricsSnapshot struct {
	System      *SystemMetrics      `json:"system"`
	Application *ApplicationMetrics `json:"application"`
	Governance  *GovernanceMetrics  `json:"governance"`
	Performance *PerformanceMetrics `json:"performance"`
	Timestamp   time.Time           `json:"timestamp"`
}

// CollectorConfig represents configuration for metrics collector
type CollectorConfig struct {
	Enabled             bool          `json:"enabled"`
	CollectionInterval  time.Duration `json:"collection_interval"`
	RetentionPeriod     time.Duration `json:"retention_period"`
	EnableSystemMetrics bool          `json:"enable_system_metrics"`
	EnableAppMetrics    bool          `json:"enable_app_metrics"`
	EnableGovMetrics    bool          `json:"enable_gov_metrics"`
	EnablePerfMetrics   bool          `json:"enable_perf_metrics"`
	PrometheusPort      int           `json:"prometheus_port"`
	PrometheusPath      string        `json:"prometheus_path"`
}