package monitor

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/wemix/wemixvisor/internal/config"
	"github.com/wemix/wemixvisor/pkg/logger"
	"go.uber.org/zap"
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

	// Status tracking
	lastStatus    HealthStatus
	statusMutex   sync.RWMutex

	// Context for lifecycle
	ctx    context.Context
	cancel context.CancelFunc

	// Channels
	statusCh chan HealthStatus

	// Stop tracking
	stopped   bool
	stopMutex sync.Mutex
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(cfg *config.Config, logger *logger.Logger) *HealthChecker {
	ctx, cancel := context.WithCancel(context.Background())

	checkInterval := 30 * time.Second
	if cfg.HealthCheckInterval > 0 {
		checkInterval = cfg.HealthCheckInterval
	}

	rpcPort := 8545
	if cfg.RPCPort > 0 {
		rpcPort = cfg.RPCPort
	}

	return &HealthChecker{
		config:        cfg,
		logger:        logger,
		httpClient:    &http.Client{Timeout: 5 * time.Second},
		rpcURL:        fmt.Sprintf("http://localhost:%d", rpcPort),
		checkInterval: checkInterval,
		ctx:          ctx,
		cancel:       cancel,
		statusCh:     make(chan HealthStatus, 1),
		checks: []HealthCheck{
			&ProcessCheck{},
			&RPCHealthCheck{url: fmt.Sprintf("http://localhost:%d", rpcPort)},
			&PeerCountCheck{minPeers: 1, rpcURL: fmt.Sprintf("http://localhost:%d", rpcPort)},
			&SyncingCheck{rpcURL: fmt.Sprintf("http://localhost:%d", rpcPort)},
		},
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
	// Don't close the channel here, let the goroutine handle it
}

// run is the main monitoring loop
func (h *HealthChecker) run() {
	ticker := time.NewTicker(h.checkInterval)
	defer ticker.Stop()
	defer func() {
		// Close channel only if not already stopped
		h.stopMutex.Lock()
		if !h.stopped {
			close(h.statusCh)
		}
		h.stopMutex.Unlock()
	}()

	// Perform initial check immediately
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

// performChecks performs all health checks
func (h *HealthChecker) performChecks() {
	status := HealthStatus{
		Healthy:   true,
		Timestamp: time.Now(),
		Checks:    make(map[string]CheckResult),
	}

	for _, check := range h.checks {
		result := CheckResult{
			Name: check.Name(),
		}

		// Create context with timeout for individual check
		ctx, cancel := context.WithTimeout(h.ctx, 5*time.Second)

		if err := check.Check(ctx); err != nil {
			result.Healthy = false
			result.Error = err.Error()
			status.Healthy = false
			h.logger.Warn("health check failed",
				zap.String("check", check.Name()),
				zap.Error(err))
		} else {
			result.Healthy = true
			h.logger.Debug("health check passed",
				zap.String("check", check.Name()))
		}

		cancel()
		status.Checks[check.Name()] = result
	}

	// Update status
	h.statusMutex.Lock()
	h.lastStatus = status
	h.statusMutex.Unlock()

	// Send status update (non-blocking)
	h.stopMutex.Lock()
	if !h.stopped {
		select {
		case h.statusCh <- status:
		default:
			// Channel full, skip this update
		}
	}
	h.stopMutex.Unlock()
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