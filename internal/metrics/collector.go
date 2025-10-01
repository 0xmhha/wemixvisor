package metrics

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/wemix/wemixvisor/pkg/logger"
)

// Collector collects various metrics from the system and application
type Collector struct {
	config *CollectorConfig
	logger *logger.Logger

	// Prometheus metrics
	registry *prometheus.Registry

	// System metrics
	cpuUsage    prometheus.Gauge
	memoryUsage prometheus.Gauge
	diskUsage   prometheus.Gauge
	goroutines  prometheus.Gauge

	// Network metrics
	networkRxBytes prometheus.Counter
	networkTxBytes prometheus.Counter

	// Application metrics
	upgradeTotal   prometheus.Counter
	upgradeSuccess prometheus.Counter
	upgradeFailed  prometheus.Counter
	upgradePending prometheus.Gauge

	// Process metrics
	processRestarts prometheus.Counter
	processUptime   prometheus.Gauge

	// Node metrics
	nodeHeight  prometheus.Gauge
	nodePeers   prometheus.Gauge
	nodeSyncing prometheus.Gauge

	// Governance metrics
	proposalTotal    prometheus.Gauge
	proposalVoting   prometheus.Gauge
	proposalPassed   prometheus.Counter
	proposalRejected prometheus.Counter

	// Voting metrics
	votingPower   prometheus.Gauge
	votingTurnout prometheus.Gauge

	// Validator metrics
	validatorActive prometheus.Gauge
	validatorJailed prometheus.Gauge

	// Performance metrics
	rpcLatency prometheus.Histogram
	apiLatency prometheus.Histogram
	tps        prometheus.Gauge

	// Internal state
	mu            sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
	lastSnapshot  *MetricsSnapshot
	startTime     time.Time

	// Callbacks for application-specific metrics
	nodeHeightFunc    func() (int64, error)
	nodePeersFunc     func() (int, error)
	nodeSyncingFunc   func() (bool, error)
	proposalStatsFunc func() (*GovernanceMetrics, error)
}

// NewCollector creates a new metrics collector
func NewCollector(config *CollectorConfig, logger *logger.Logger) *Collector {
	registry := prometheus.NewRegistry()

	c := &Collector{
		config:    config,
		logger:    logger,
		registry:  registry,
		startTime: time.Now(),
	}

	// Initialize Prometheus metrics
	c.initPrometheusMetrics()

	// Register metrics with registry
	c.registerMetrics()

	return c
}

// initPrometheusMetrics initializes all Prometheus metrics
func (c *Collector) initPrometheusMetrics() {
	// System metrics
	c.cpuUsage = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "wemixvisor_cpu_usage_percent",
		Help: "Current CPU usage percentage",
	})

	c.memoryUsage = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "wemixvisor_memory_usage_percent",
		Help: "Current memory usage percentage",
	})

	c.diskUsage = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "wemixvisor_disk_usage_percent",
		Help: "Current disk usage percentage",
	})

	c.goroutines = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "wemixvisor_goroutines",
		Help: "Number of goroutines",
	})

	// Network metrics
	c.networkRxBytes = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "wemixvisor_network_rx_bytes_total",
		Help: "Total network bytes received",
	})

	c.networkTxBytes = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "wemixvisor_network_tx_bytes_total",
		Help: "Total network bytes transmitted",
	})

	// Application metrics
	c.upgradeTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "wemixvisor_upgrades_total",
		Help: "Total number of upgrades attempted",
	})

	c.upgradeSuccess = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "wemixvisor_upgrades_success_total",
		Help: "Total number of successful upgrades",
	})

	c.upgradeFailed = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "wemixvisor_upgrades_failed_total",
		Help: "Total number of failed upgrades",
	})

	c.upgradePending = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "wemixvisor_upgrades_pending",
		Help: "Number of pending upgrades",
	})

	// Process metrics
	c.processRestarts = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "wemixvisor_process_restarts_total",
		Help: "Total number of process restarts",
	})

	c.processUptime = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "wemixvisor_process_uptime_seconds",
		Help: "Process uptime in seconds",
	})

	// Node metrics
	c.nodeHeight = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "wemixvisor_node_height",
		Help: "Current blockchain height",
	})

	c.nodePeers = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "wemixvisor_node_peers",
		Help: "Number of connected peers",
	})

	c.nodeSyncing = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "wemixvisor_node_syncing",
		Help: "Whether node is syncing (1) or not (0)",
	})

	// Governance metrics
	c.proposalTotal = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "wemixvisor_proposals_total",
		Help: "Total number of proposals",
	})

	c.proposalVoting = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "wemixvisor_proposals_voting",
		Help: "Number of proposals in voting period",
	})

	c.proposalPassed = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "wemixvisor_proposals_passed_total",
		Help: "Total number of passed proposals",
	})

	c.proposalRejected = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "wemixvisor_proposals_rejected_total",
		Help: "Total number of rejected proposals",
	})

	// Voting metrics
	c.votingPower = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "wemixvisor_voting_power",
		Help: "Current voting power",
	})

	c.votingTurnout = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "wemixvisor_voting_turnout_percent",
		Help: "Average voting turnout percentage",
	})

	// Validator metrics
	c.validatorActive = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "wemixvisor_validators_active",
		Help: "Number of active validators",
	})

	c.validatorJailed = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "wemixvisor_validators_jailed",
		Help: "Number of jailed validators",
	})

	// Performance metrics
	c.rpcLatency = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "wemixvisor_rpc_latency_milliseconds",
		Help:    "RPC call latency in milliseconds",
		Buckets: prometheus.DefBuckets,
	})

	c.apiLatency = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "wemixvisor_api_latency_milliseconds",
		Help:    "API call latency in milliseconds",
		Buckets: prometheus.DefBuckets,
	})

	c.tps = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "wemixvisor_transactions_per_second",
		Help: "Transactions per second",
	})
}

// registerMetrics registers all metrics with the Prometheus registry
func (c *Collector) registerMetrics() {
	// System metrics
	if c.config.EnableSystemMetrics {
		c.registry.MustRegister(c.cpuUsage)
		c.registry.MustRegister(c.memoryUsage)
		c.registry.MustRegister(c.diskUsage)
		c.registry.MustRegister(c.goroutines)
		c.registry.MustRegister(c.networkRxBytes)
		c.registry.MustRegister(c.networkTxBytes)
	}

	// Application metrics
	if c.config.EnableAppMetrics {
		c.registry.MustRegister(c.upgradeTotal)
		c.registry.MustRegister(c.upgradeSuccess)
		c.registry.MustRegister(c.upgradeFailed)
		c.registry.MustRegister(c.upgradePending)
		c.registry.MustRegister(c.processRestarts)
		c.registry.MustRegister(c.processUptime)
		c.registry.MustRegister(c.nodeHeight)
		c.registry.MustRegister(c.nodePeers)
		c.registry.MustRegister(c.nodeSyncing)
	}

	// Governance metrics
	if c.config.EnableGovMetrics {
		c.registry.MustRegister(c.proposalTotal)
		c.registry.MustRegister(c.proposalVoting)
		c.registry.MustRegister(c.proposalPassed)
		c.registry.MustRegister(c.proposalRejected)
		c.registry.MustRegister(c.votingPower)
		c.registry.MustRegister(c.votingTurnout)
		c.registry.MustRegister(c.validatorActive)
		c.registry.MustRegister(c.validatorJailed)
	}

	// Performance metrics
	if c.config.EnablePerfMetrics {
		c.registry.MustRegister(c.rpcLatency)
		c.registry.MustRegister(c.apiLatency)
		c.registry.MustRegister(c.tps)
	}
}

// Start starts the metrics collection
func (c *Collector) Start() error {
	if !c.config.Enabled {
		c.logger.Info("Metrics collection is disabled")
		return nil
	}

	c.ctx, c.cancel = context.WithCancel(context.Background())

	// Start collection loop
	go c.collectionLoop()

	c.logger.Info("Metrics collector started",
		"interval", c.config.CollectionInterval,
		"prometheus_port", c.config.PrometheusPort)

	return nil
}

// Stop stops the metrics collection
func (c *Collector) Stop() error {
	if c.cancel != nil {
		c.cancel()
	}

	c.logger.Info("Metrics collector stopped")
	return nil
}

// collectionLoop runs the periodic metrics collection
func (c *Collector) collectionLoop() {
	ticker := time.NewTicker(c.config.CollectionInterval)
	defer ticker.Stop()

	// Collect immediately on start
	c.collect()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.collect()
		}
	}
}

// collect gathers all metrics
func (c *Collector) collect() {
	snapshot := &MetricsSnapshot{
		Timestamp: time.Now(),
	}

	// Collect system metrics
	if c.config.EnableSystemMetrics {
		snapshot.System = c.collectSystemMetrics()
		c.updateSystemPrometheus(snapshot.System)
	}

	// Collect application metrics
	if c.config.EnableAppMetrics {
		snapshot.Application = c.collectApplicationMetrics()
		c.updateApplicationPrometheus(snapshot.Application)
	}

	// Collect governance metrics
	if c.config.EnableGovMetrics {
		snapshot.Governance = c.collectGovernanceMetrics()
		c.updateGovernancePrometheus(snapshot.Governance)
	}

	// Collect performance metrics
	if c.config.EnablePerfMetrics {
		snapshot.Performance = c.collectPerformanceMetrics()
		c.updatePerformancePrometheus(snapshot.Performance)
	}

	// Store snapshot
	c.mu.Lock()
	c.lastSnapshot = snapshot
	c.mu.Unlock()

	c.logger.Debug("Metrics collected", "timestamp", snapshot.Timestamp)
}

// collectSystemMetrics collects system-level metrics
func (c *Collector) collectSystemMetrics() *SystemMetrics {
	metrics := &SystemMetrics{
		Timestamp: time.Now(),
	}

	// CPU usage
	if cpuPercent, err := cpu.Percent(time.Second, false); err == nil && len(cpuPercent) > 0 {
		metrics.CPUUsage = cpuPercent[0]
	}

	// Memory usage
	if memStat, err := mem.VirtualMemory(); err == nil {
		metrics.MemoryUsage = memStat.UsedPercent
		metrics.MemoryTotal = memStat.Total
		metrics.MemoryAvailable = memStat.Available
	}

	// Disk usage
	if diskStat, err := disk.Usage("/"); err == nil {
		metrics.DiskUsage = diskStat.UsedPercent
		metrics.DiskTotal = diskStat.Total
		metrics.DiskAvailable = diskStat.Free
	}

	// Network stats
	if netStats, err := net.IOCounters(false); err == nil && len(netStats) > 0 {
		metrics.NetworkRxBytes = netStats[0].BytesRecv
		metrics.NetworkTxBytes = netStats[0].BytesSent
		metrics.NetworkRxPackets = netStats[0].PacketsRecv
		metrics.NetworkTxPackets = netStats[0].PacketsSent
	}

	// Goroutines
	metrics.Goroutines = runtime.NumGoroutine()

	// Uptime
	metrics.Uptime = int64(time.Since(c.startTime).Seconds())

	return metrics
}

// collectApplicationMetrics collects application-level metrics
func (c *Collector) collectApplicationMetrics() *ApplicationMetrics {
	metrics := &ApplicationMetrics{
		Timestamp:     time.Now(),
		ProcessUptime: int64(time.Since(c.startTime).Seconds()),
	}

	// Node metrics (if callbacks are set)
	if c.nodeHeightFunc != nil {
		if height, err := c.nodeHeightFunc(); err == nil {
			metrics.NodeHeight = height
		}
	}

	if c.nodePeersFunc != nil {
		if peers, err := c.nodePeersFunc(); err == nil {
			metrics.NodePeers = peers
		}
	}

	if c.nodeSyncingFunc != nil {
		if syncing, err := c.nodeSyncingFunc(); err == nil {
			metrics.NodeSyncing = syncing
		}
	}

	return metrics
}

// collectGovernanceMetrics collects governance-related metrics
func (c *Collector) collectGovernanceMetrics() *GovernanceMetrics {
	metrics := &GovernanceMetrics{
		Timestamp: time.Now(),
	}

	// Get governance stats from callback
	if c.proposalStatsFunc != nil {
		if stats, err := c.proposalStatsFunc(); err == nil {
			return stats
		}
	}

	return metrics
}

// collectPerformanceMetrics collects performance-related metrics
func (c *Collector) collectPerformanceMetrics() *PerformanceMetrics {
	return &PerformanceMetrics{
		Timestamp: time.Now(),
		// Performance metrics are typically updated via ObserveXXX methods
	}
}

// updateSystemPrometheus updates Prometheus metrics for system
func (c *Collector) updateSystemPrometheus(metrics *SystemMetrics) {
	if metrics == nil {
		return
	}

	c.cpuUsage.Set(metrics.CPUUsage)
	c.memoryUsage.Set(metrics.MemoryUsage)
	c.diskUsage.Set(metrics.DiskUsage)
	c.goroutines.Set(float64(metrics.Goroutines))
	c.networkRxBytes.Add(float64(metrics.NetworkRxBytes))
	c.networkTxBytes.Add(float64(metrics.NetworkTxBytes))
}

// updateApplicationPrometheus updates Prometheus metrics for application
func (c *Collector) updateApplicationPrometheus(metrics *ApplicationMetrics) {
	if metrics == nil {
		return
	}

	c.processUptime.Set(float64(metrics.ProcessUptime))
	c.nodeHeight.Set(float64(metrics.NodeHeight))
	c.nodePeers.Set(float64(metrics.NodePeers))
	if metrics.NodeSyncing {
		c.nodeSyncing.Set(1)
	} else {
		c.nodeSyncing.Set(0)
	}
}

// updateGovernancePrometheus updates Prometheus metrics for governance
func (c *Collector) updateGovernancePrometheus(metrics *GovernanceMetrics) {
	if metrics == nil {
		return
	}

	c.proposalTotal.Set(float64(metrics.ProposalTotal))
	c.proposalVoting.Set(float64(metrics.ProposalVoting))
	c.votingPower.Set(metrics.VotingPower)
	c.votingTurnout.Set(metrics.VotingTurnout)
	c.validatorActive.Set(float64(metrics.ValidatorActive))
	c.validatorJailed.Set(float64(metrics.ValidatorJailed))
}

// updatePerformancePrometheus updates Prometheus metrics for performance
func (c *Collector) updatePerformancePrometheus(metrics *PerformanceMetrics) {
	if metrics == nil {
		return
	}

	c.tps.Set(metrics.TransactionsPerSecond)
}

// GetSnapshot returns the latest metrics snapshot
func (c *Collector) GetSnapshot() *MetricsSnapshot {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastSnapshot
}

// GetRegistry returns the Prometheus registry
func (c *Collector) GetRegistry() *prometheus.Registry {
	return c.registry
}

// SetNodeHeightCallback sets the callback for getting node height
func (c *Collector) SetNodeHeightCallback(fn func() (int64, error)) {
	c.nodeHeightFunc = fn
}

// SetNodePeersCallback sets the callback for getting node peers count
func (c *Collector) SetNodePeersCallback(fn func() (int, error)) {
	c.nodePeersFunc = fn
}

// SetNodeSyncingCallback sets the callback for getting node syncing status
func (c *Collector) SetNodeSyncingCallback(fn func() (bool, error)) {
	c.nodeSyncingFunc = fn
}

// SetProposalStatsCallback sets the callback for getting proposal statistics
func (c *Collector) SetProposalStatsCallback(fn func() (*GovernanceMetrics, error)) {
	c.proposalStatsFunc = fn
}

// IncrementUpgradeTotal increments the total upgrade counter
func (c *Collector) IncrementUpgradeTotal() {
	c.upgradeTotal.Inc()
}

// IncrementUpgradeSuccess increments the successful upgrade counter
func (c *Collector) IncrementUpgradeSuccess() {
	c.upgradeSuccess.Inc()
}

// IncrementUpgradeFailed increments the failed upgrade counter
func (c *Collector) IncrementUpgradeFailed() {
	c.upgradeFailed.Inc()
}

// SetUpgradePending sets the number of pending upgrades
func (c *Collector) SetUpgradePending(count int) {
	c.upgradePending.Set(float64(count))
}

// IncrementProcessRestarts increments the process restart counter
func (c *Collector) IncrementProcessRestarts() {
	c.processRestarts.Inc()
}

// ObserveRPCLatency records an RPC latency observation
func (c *Collector) ObserveRPCLatency(latencyMS float64) {
	c.rpcLatency.Observe(latencyMS)
}

// ObserveAPILatency records an API latency observation
func (c *Collector) ObserveAPILatency(latencyMS float64) {
	c.apiLatency.Observe(latencyMS)
}

// RecordError records an error metric
func (c *Collector) RecordError(source string, err error) {
	c.logger.Error("Metric error recorded",
		"source", source,
		"error", err.Error())
}

// GenerateAlert generates an alert based on metrics
func (c *Collector) GenerateAlert(alert *Alert) {
	c.logger.Warn("Alert generated",
		"name", alert.Name,
		"level", alert.Level,
		"message", alert.Message)
}