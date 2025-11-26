package monitor

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/wemix/wemixvisor/internal/config"
	"github.com/wemix/wemixvisor/pkg/logger"
)

// Default values for health checker configuration
const (
	DefaultCheckInterval = 30 * time.Second
	DefaultCheckTimeout  = 5 * time.Second
	DefaultRPCPort       = 8545
	DefaultMinPeers      = 1
)

// HealthStatus represents the overall health status
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
	Details string `json:"details,omitempty"`
}

// HealthCheck represents a single health check
type HealthCheck interface {
	Name() string
	Check(ctx context.Context) error
}

// HealthChecker monitors the health of the node
type HealthChecker struct {
	config        *config.Config
	logger        *logger.Logger
	httpClient    *http.Client
	rpcURL        string
	checkInterval time.Duration
	checks        []HealthCheck

	lastStatus  HealthStatus
	statusMutex sync.RWMutex

	ctx    context.Context
	cancel context.CancelFunc

	statusCh  chan HealthStatus
	stopped   bool
	stopMutex sync.Mutex
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(cfg *config.Config, log *logger.Logger) *HealthChecker {
	ctx, cancel := context.WithCancel(context.Background())

	checkInterval := DefaultCheckInterval
	if cfg.HealthCheckInterval > 0 {
		checkInterval = cfg.HealthCheckInterval
	}

	rpcPort := DefaultRPCPort
	if cfg.RPCPort > 0 {
		rpcPort = cfg.RPCPort
	}

	rpcURL := fmt.Sprintf("http://localhost:%d", rpcPort)

	return &HealthChecker{
		config:        cfg,
		logger:        log,
		httpClient:    &http.Client{Timeout: DefaultCheckTimeout},
		rpcURL:        rpcURL,
		checkInterval: checkInterval,
		ctx:           ctx,
		cancel:        cancel,
		statusCh:      make(chan HealthStatus, 1),
		checks:        createDefaultChecks(rpcURL),
	}
}

// createDefaultChecks creates the default health checks
func createDefaultChecks(rpcURL string) []HealthCheck {
	return []HealthCheck{
		&ProcessCheck{},
		&RPCHealthCheck{url: rpcURL},
		&PeerCountCheck{minPeers: DefaultMinPeers, rpcURL: rpcURL},
		&SyncingCheck{rpcURL: rpcURL},
	}
}

// Start starts the health monitoring
func (h *HealthChecker) Start() <-chan HealthStatus {
	go h.run()
	return h.statusCh
}

// Stop stops the health monitoring
func (h *HealthChecker) Stop() {
	h.stopMutex.Lock()
	h.stopped = true
	h.stopMutex.Unlock()

	h.cancel()
}

// run is the main monitoring loop
func (h *HealthChecker) run() {
	ticker := time.NewTicker(h.checkInterval)
	defer ticker.Stop()
	defer h.cleanupOnExit()

	h.performChecks()

	for {
		select {
		case <-h.ctx.Done():
			return
		case <-ticker.C:
			h.performChecks()
		}
	}
}

// cleanupOnExit handles cleanup when the monitoring loop exits
func (h *HealthChecker) cleanupOnExit() {
	h.stopMutex.Lock()
	if !h.stopped {
		close(h.statusCh)
	}
	h.stopMutex.Unlock()
}

// performChecks performs all health checks
func (h *HealthChecker) performChecks() {
	status := HealthStatus{
		Healthy:   true,
		Timestamp: time.Now(),
		Checks:    make(map[string]CheckResult),
	}

	for _, check := range h.checks {
		result := h.executeCheck(check)
		status.Checks[check.Name()] = result

		if !result.Healthy {
			status.Healthy = false
		}
	}

	h.updateStatus(status)
	h.sendStatusUpdate(status)
}

// executeCheck executes a single health check with timeout
func (h *HealthChecker) executeCheck(check HealthCheck) CheckResult {
	result := CheckResult{Name: check.Name()}

	ctx, cancel := context.WithTimeout(h.ctx, DefaultCheckTimeout)
	defer cancel()

	if err := check.Check(ctx); err != nil {
		result.Healthy = false
		result.Error = err.Error()
		h.logger.Warn("health check failed",
			zap.String("check", check.Name()),
			zap.Error(err))
	} else {
		result.Healthy = true
		h.logger.Debug("health check passed",
			zap.String("check", check.Name()))
	}

	return result
}

// updateStatus updates the cached health status
func (h *HealthChecker) updateStatus(status HealthStatus) {
	h.statusMutex.Lock()
	h.lastStatus = status
	h.statusMutex.Unlock()
}

// sendStatusUpdate sends status update to the channel (non-blocking)
func (h *HealthChecker) sendStatusUpdate(status HealthStatus) {
	h.stopMutex.Lock()
	defer h.stopMutex.Unlock()

	if h.stopped {
		return
	}

	select {
	case h.statusCh <- status:
	default:
		// Channel full, skip this update
	}
}

// GetStatus returns the last health status
func (h *HealthChecker) GetStatus() HealthStatus {
	h.statusMutex.RLock()
	defer h.statusMutex.RUnlock()
	return h.lastStatus
}

// IsHealthy returns true if the node is healthy
func (h *HealthChecker) IsHealthy() bool {
	h.statusMutex.RLock()
	defer h.statusMutex.RUnlock()
	return h.lastStatus.Healthy
}
