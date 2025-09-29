package metrics

import (
	"context"
	"runtime"
	"sync"
	"time"

	"go.uber.org/zap"
	"github.com/wemix/wemixvisor/internal/config"
	"github.com/wemix/wemixvisor/pkg/logger"
)

// Metrics represents system metrics
type Metrics struct {
	Timestamp      time.Time `json:"timestamp"`
	NodeUptime     int64     `json:"node_uptime_seconds"`
	RestartCount   int       `json:"restart_count"`
	MemoryUsageMB  float64   `json:"memory_usage_mb"`
	CPUUsagePercent float64  `json:"cpu_usage_percent"`
	DiskUsageGB    float64   `json:"disk_usage_gb"`
	PeerCount      int       `json:"peer_count"`
	BlockHeight    int64     `json:"block_height"`
	SyncProgress   float64   `json:"sync_progress"`
	Healthy        bool      `json:"healthy"`
}

// MetricsCollector collects and stores system metrics
type MetricsCollector struct {
	config   *config.Config
	logger   *logger.Logger
	nodeInfo NodeInfoProvider

	// Metrics storage
	currentMetrics Metrics
	metricsMutex   sync.RWMutex

	// Collection control
	ctx              context.Context
	cancel           context.CancelFunc
	collectionTicker *time.Ticker
}

// NodeInfoProvider provides node information for metrics
type NodeInfoProvider interface {
	GetUptime() time.Duration
	GetRestartCount() int
	IsHealthy() bool
	GetPID() int
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(cfg *config.Config, logger *logger.Logger, nodeInfo NodeInfoProvider) *MetricsCollector {
	ctx, cancel := context.WithCancel(context.Background())

	interval := 60 * time.Second // Default 1 minute
	if cfg.MetricsInterval > 0 {
		interval = cfg.MetricsInterval
	}

	return &MetricsCollector{
		config:           cfg,
		logger:           logger,
		nodeInfo:         nodeInfo,
		ctx:              ctx,
		cancel:           cancel,
		collectionTicker: time.NewTicker(interval),
	}
}

// Start begins metrics collection
func (c *MetricsCollector) Start() {
	c.logger.Info("starting metrics collection")

	// Start collection loop (initial metrics will be collected in the goroutine)
	// This prevents deadlock when called while Manager holds a lock
	go c.run()
}

// Stop stops metrics collection
func (c *MetricsCollector) Stop() {
	c.logger.Info("stopping metrics collection")
	c.cancel()
	if c.collectionTicker != nil {
		c.collectionTicker.Stop()
	}
}

// GetMetrics returns the current metrics
func (c *MetricsCollector) GetMetrics() Metrics {
	c.metricsMutex.RLock()
	defer c.metricsMutex.RUnlock()
	return c.currentMetrics
}

// run is the main collection loop
func (c *MetricsCollector) run() {
	// Collect initial metrics
	c.collectMetrics()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-c.collectionTicker.C:
			c.collectMetrics()
		}
	}
}

// collectMetrics collects current system metrics
func (c *MetricsCollector) collectMetrics() {
	c.logger.Debug("collecting metrics")

	metrics := Metrics{
		Timestamp:    time.Now(),
		RestartCount: c.nodeInfo.GetRestartCount(),
		Healthy:      c.nodeInfo.IsHealthy(),
	}

	// Node uptime
	if uptime := c.nodeInfo.GetUptime(); uptime > 0 {
		metrics.NodeUptime = int64(uptime.Seconds())
	}

	// Memory usage
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	metrics.MemoryUsageMB = float64(memStats.Alloc) / (1024 * 1024)

	// TODO: Add CPU usage collection (requires platform-specific code)
	metrics.CPUUsagePercent = 0.0

	// TODO: Add disk usage collection
	metrics.DiskUsageGB = 0.0

	// TODO: Add RPC-based metrics (peer count, block height, sync progress)
	metrics.PeerCount = 0
	metrics.BlockHeight = 0
	metrics.SyncProgress = 0.0

	// Store metrics
	c.metricsMutex.Lock()
	c.currentMetrics = metrics
	c.metricsMutex.Unlock()

	c.logger.Debug("metrics collected",
		zap.Time("timestamp", metrics.Timestamp),
		zap.Int64("uptime", metrics.NodeUptime),
		zap.Float64("memory_mb", metrics.MemoryUsageMB),
		zap.Bool("healthy", metrics.Healthy))
}

// ExportMetrics exports metrics in a specific format
func (c *MetricsCollector) ExportMetrics(format string) ([]byte, error) {
	metrics := c.GetMetrics()

	switch format {
	case "prometheus":
		return c.exportPrometheus(metrics)
	case "json":
		return c.exportJSON(metrics)
	default:
		return c.exportJSON(metrics)
	}
}

// exportJSON exports metrics as JSON
func (c *MetricsCollector) exportJSON(metrics Metrics) ([]byte, error) {
	// This would use json.Marshal in practice
	// For now, return a simple string representation
	result := `{
  "timestamp": "` + metrics.Timestamp.Format(time.RFC3339) + `",
  "node_uptime_seconds": ` + string(rune(metrics.NodeUptime)) + `,
  "restart_count": ` + string(rune(metrics.RestartCount)) + `,
  "memory_usage_mb": ` + string(rune(int(metrics.MemoryUsageMB))) + `,
  "healthy": ` + string(rune(boolToInt(metrics.Healthy))) + `
}`
	return []byte(result), nil
}

// exportPrometheus exports metrics in Prometheus format
func (c *MetricsCollector) exportPrometheus(metrics Metrics) ([]byte, error) {
	result := `# HELP wemixvisor_uptime_seconds Node uptime in seconds
# TYPE wemixvisor_uptime_seconds gauge
wemixvisor_uptime_seconds ` + string(rune(metrics.NodeUptime)) + `

# HELP wemixvisor_restart_count Total number of node restarts
# TYPE wemixvisor_restart_count counter
wemixvisor_restart_count ` + string(rune(metrics.RestartCount)) + `

# HELP wemixvisor_memory_usage_mb Memory usage in megabytes
# TYPE wemixvisor_memory_usage_mb gauge
wemixvisor_memory_usage_mb ` + string(rune(int(metrics.MemoryUsageMB))) + `

# HELP wemixvisor_healthy Node health status (1=healthy, 0=unhealthy)
# TYPE wemixvisor_healthy gauge
wemixvisor_healthy ` + string(rune(boolToInt(metrics.Healthy))) + `
`
	return []byte(result), nil
}

// boolToInt converts boolean to integer for metrics
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}