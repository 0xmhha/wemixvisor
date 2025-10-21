package node

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"
	"github.com/wemix/wemixvisor/internal/config"
	"github.com/wemix/wemixvisor/internal/metrics"
	"github.com/wemix/wemixvisor/internal/monitor"
	"github.com/wemix/wemixvisor/pkg/logger"
)

// Manager handles the lifecycle of a node process
type Manager struct {
	// Core components
	config *config.Config
	logger *logger.Logger

	// Process management
	cmd        *exec.Cmd
	process    *os.Process
	state      NodeState
	stateMutex sync.RWMutex

	// CLI pass-through
	nodeArgs    []string
	nodeOptions map[string]string

	// Monitoring
	startTime        time.Time
	restartCount     int
	maxRestarts      int
	healthChecker    *monitor.HealthChecker
	metricsCollector *metrics.Collector

	// Channels for lifecycle management
	stopCh    chan struct{}
	restartCh chan struct{}
	errorCh   chan error
	doneCh    chan struct{}

	// Context for graceful shutdown
	ctx    context.Context
	cancel context.CancelFunc
}

// NewManager creates a new node manager
func NewManager(cfg *config.Config, logger *logger.Logger) *Manager {
	ctx, cancel := context.WithCancel(context.Background())

	maxRestarts := 5
	if cfg.MaxRestarts > 0 {
		maxRestarts = cfg.MaxRestarts
	}

	// Create health checker
	healthChecker := monitor.NewHealthChecker(cfg, logger)

	manager := &Manager{
		config:        cfg,
		logger:        logger,
		state:         StateStopped,
		nodeOptions:   make(map[string]string),
		maxRestarts:   maxRestarts,
		healthChecker: healthChecker,
		stopCh:        make(chan struct{}),
		restartCh:     make(chan struct{}),
		errorCh:       make(chan error, 10),
		doneCh:        make(chan struct{}),
		ctx:           ctx,
		cancel:        cancel,
	}

	// Create metrics collector
	if cfg.MetricsEnabled {
		collectorConfig := &metrics.CollectorConfig{
			Enabled:             true,
			CollectionInterval:  cfg.MetricsCollectionInterval,
			EnableSystemMetrics: cfg.EnableSystemMetrics,
			EnableAppMetrics:    cfg.EnableAppMetrics,
			EnableGovMetrics:    cfg.EnableGovMetrics,
			EnablePerfMetrics:   cfg.EnablePerfMetrics,
			PrometheusPort:      cfg.MetricsPort,
			PrometheusPath:      cfg.MetricsPath,
		}
		manager.metricsCollector = metrics.NewCollector(collectorConfig, logger)

		// Set callbacks for node-specific metrics
		manager.metricsCollector.SetNodeHeightCallback(func() (int64, error) {
			// TODO: implement actual height fetching
			return 0, nil
		})
	}

	return manager
}

// Start starts the node with the given arguments
func (m *Manager) Start(args []string) error {
	m.stateMutex.Lock()
	// Note: We manually unlock the mutex to prevent deadlock with metrics collector

	if m.state != StateStopped {
		m.stateMutex.Unlock()
		return fmt.Errorf("node is not in stopped state: %v", m.state)
	}

	m.state = StateStarting
	m.nodeArgs = args

	// Build command
	cmdPath := m.config.CurrentBin()
	if cmdPath == "" {
		m.state = StateError
		m.stateMutex.Unlock()
		return fmt.Errorf("no binary found")
	}

	// Ensure binary exists and is executable
	if _, err := os.Stat(cmdPath); err != nil {
		m.state = StateError
		m.stateMutex.Unlock()
		return fmt.Errorf("binary not found: %w", err)
	}

	m.logger.Info("starting node",
		zap.String("binary", cmdPath),
		zap.Strings("args", args))

	cmd := exec.CommandContext(m.ctx, cmdPath, args...)

	// Set environment variables
	cmd.Env = m.buildEnvironment()

	// Set working directory to home
	if m.config.Home != "" {
		cmd.Dir = m.config.Home
	}

	// Setup stdout/stderr
	logFile := m.getLogFile()
	if logFile != nil {
		cmd.Stdout = logFile
		cmd.Stderr = logFile
	} else {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	// Set process group for clean shutdown and zombie prevention
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
		Pgid:    0, // Create new process group
	}

	// Start the process
	if err := cmd.Start(); err != nil {
		m.state = StateError
		m.stateMutex.Unlock()
		return fmt.Errorf("failed to start node: %w", err)
	}

	m.cmd = cmd
	m.process = cmd.Process
	m.startTime = time.Now()
	m.state = StateRunning
	pid := m.process.Pid

	// Unlock mutex before starting other components to prevent deadlock
	m.stateMutex.Unlock()

	// Start monitoring goroutine
	go m.monitor()

	// Start health monitoring
	go func() {
		healthStatusCh := m.healthChecker.Start()
		for {
			select {
			case status := <-healthStatusCh:
				if !status.Healthy {
					m.logger.Warn("health check failed",
						zap.Bool("healthy", status.Healthy),
						zap.Any("checks", status.Checks))
				}
			case <-m.ctx.Done():
				return
			}
		}
	}()

	// Start metrics collection (safe to call after mutex unlock)
	if m.metricsCollector != nil {
		m.metricsCollector.Start()
	}

	m.logger.Info("node started successfully",
		zap.Int("pid", pid),
		zap.Strings("args", args))

	return nil
}

// Stop stops the node gracefully
func (m *Manager) Stop() error {
	m.stateMutex.Lock()

	if m.state != StateRunning {
		m.stateMutex.Unlock()
		return fmt.Errorf("node is not running")
	}

	m.state = StateStopping
	pid := m.process.Pid
	process := m.process
	m.stateMutex.Unlock()

	m.logger.Info("stopping node", zap.Int("pid", pid))

	// Send SIGTERM to the entire process group for graceful shutdown
	if err := syscall.Kill(-pid, syscall.SIGTERM); err != nil {
		m.logger.Error("failed to send SIGTERM to process group", zap.Error(err))
		// Fallback to individual process
		if err := process.Signal(syscall.SIGTERM); err != nil {
			m.logger.Error("failed to send SIGTERM to process", zap.Error(err))
			// Force kill as last resort
			if err := syscall.Kill(-pid, syscall.SIGKILL); err != nil {
				return fmt.Errorf("failed to kill process: %w", err)
			}
		}
	}

	shutdownGrace := 30 * time.Second
	if m.config.ShutdownGrace > 0 {
		shutdownGrace = m.config.ShutdownGrace
	}

	// Wait for the monitor goroutine to detect process exit
	// or timeout if it takes too long
	timer := time.NewTimer(shutdownGrace)
	defer timer.Stop()

	select {
	case <-m.doneCh:
		m.logger.Info("node stopped gracefully")
	case <-timer.C:
		m.logger.Warn("grace period exceeded, forcing shutdown")
		// Kill the entire process group
		if err := syscall.Kill(-pid, syscall.SIGKILL); err != nil {
			m.logger.Error("failed to kill process group", zap.Error(err))
			// Fallback to individual process
			if err := process.Kill(); err != nil {
				m.logger.Error("failed to kill process", zap.Error(err))
			}
		}
		// Give it a moment to die
		time.Sleep(100 * time.Millisecond)
	}

	// Stop health monitoring and metrics collection
	m.healthChecker.Stop()
	if m.metricsCollector != nil {
		m.metricsCollector.Stop()
	}

	m.stateMutex.Lock()
	m.state = StateStopped
	m.cmd = nil
	m.process = nil
	if m.doneCh != nil {
		close(m.doneCh)
		m.doneCh = nil
	}
	m.stateMutex.Unlock()

	return nil
}

// Restart restarts the node with the same arguments
func (m *Manager) Restart() error {
	m.logger.Info("restarting node")

	// Save current args
	args := m.nodeArgs

	// Stop the node if it's running
	currentState := m.GetState()
	if currentState == StateRunning {
		if err := m.Stop(); err != nil {
			return fmt.Errorf("failed to stop node: %w", err)
		}
	} else if currentState != StateStopped {
		// If in error or other state, reset to stopped
		m.stateMutex.Lock()
		m.state = StateStopped
		m.cmd = nil
		m.process = nil
		m.stateMutex.Unlock()
	}

	// Wait briefly before restart
	time.Sleep(2 * time.Second)

	// Reset done channel
	m.stateMutex.Lock()
	if m.doneCh == nil {
		m.doneCh = make(chan struct{})
	}
	m.stateMutex.Unlock()

	// Start with same args
	if err := m.Start(args); err != nil {
		return fmt.Errorf("failed to start node: %w", err)
	}

	m.restartCount++
	m.logger.Info("node restarted successfully", zap.Int("restart_count", m.restartCount))

	return nil
}

// GetState returns the current node state
func (m *Manager) GetState() NodeState {
	m.stateMutex.RLock()
	defer m.stateMutex.RUnlock()
	return m.state
}

// GetStatus returns detailed status information
func (m *Manager) GetStatus() *Status {
	m.stateMutex.RLock()
	defer m.stateMutex.RUnlock()

	status := &Status{
		State:        m.state,
		StateString:  m.state.String(),
		StartTime:    m.startTime,
		RestartCount: m.restartCount,
		Network:      m.config.Network,
		Binary:       m.config.CurrentBin(),
	}

	if m.process != nil {
		status.PID = m.process.Pid
		status.Uptime = time.Since(m.startTime)
	}

	// Try to get version from binary (only if running)
	if status.Binary != "" && m.state == StateRunning {
		status.Version = m.GetVersion()
	}

	// Get health status if node is running
	if m.state == StateRunning && m.healthChecker != nil {
		healthStatus := m.healthChecker.GetStatus()
		status.Health = &HealthStatus{
			Healthy:   healthStatus.Healthy,
			Timestamp: healthStatus.Timestamp,
			Checks:    make(map[string]CheckResult),
		}

		// Convert monitor.CheckResult to node.CheckResult
		for name, check := range healthStatus.Checks {
			status.Health.Checks[name] = CheckResult{
				Name:    check.Name,
				Healthy: check.Healthy,
				Error:   check.Error,
			}
		}
	}

	return status
}

// Wait waits for the node to exit
func (m *Manager) Wait() <-chan struct{} {
	return m.doneCh
}

// monitor monitors the node process and handles crashes
func (m *Manager) monitor() {
	m.stateMutex.RLock()
	cmd := m.cmd
	m.stateMutex.RUnlock()

	if cmd == nil {
		return
	}

	// Wait for process to exit
	err := cmd.Wait()

	// Reap any zombie child processes
	go m.reapZombies()

	m.stateMutex.Lock()
	defer m.stateMutex.Unlock()

	// Check if this was an expected shutdown
	if m.state == StateStopping {
		return
	}

	// Process crashed unexpectedly
	m.state = StateCrashed
	m.logger.Error("node process crashed unexpectedly", zap.Error(err))

	// Close done channel to signal process exit
	if m.doneCh != nil {
		close(m.doneCh)
		m.doneCh = nil
	}

	// Check if we should auto-restart
	if m.config.RestartOnFailure && m.restartCount < m.maxRestarts {
		m.logger.Info("attempting auto-restart",
			zap.Int("attempt", m.restartCount+1),
			zap.Int("max", m.maxRestarts))

		go func() {
			time.Sleep(5 * time.Second) // Wait before restart
			if err := m.Restart(); err != nil {
				m.logger.Error("auto-restart failed", zap.Error(err))
				m.errorCh <- err
			}
		}()
	} else {
		m.state = StateError
		if m.restartCount >= m.maxRestarts {
			m.logger.Error("max restart attempts reached",
				zap.Int("restart_count", m.restartCount),
				zap.Int("max", m.maxRestarts))
		}
	}
}

// buildEnvironment builds the environment variables for the node
func (m *Manager) buildEnvironment() []string {
	env := os.Environ()

	// Add custom environment variables
	if m.config.Home != "" {
		env = append(env, fmt.Sprintf("WEMIX_HOME=%s", m.config.Home))
	}
	if m.config.Network != "" {
		env = append(env, fmt.Sprintf("WEMIX_NETWORK=%s", m.config.Network))
	}

	// Add any custom environment variables from config
	for k, v := range m.config.Environment {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	return env
}

// getLogFile returns the log file for node output
func (m *Manager) getLogFile() *os.File {
	if m.config.LogFile == "" {
		return nil
	}

	logPath := m.config.LogFile
	if !filepath.IsAbs(logPath) {
		logPath = filepath.Join(m.config.Home, logPath)
	}

	// Ensure log directory exists
	logDir := filepath.Dir(logPath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		m.logger.Error("failed to create log directory", zap.Error(err))
		return nil
	}

	// Open or create log file
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		m.logger.Error("failed to open log file", zap.Error(err))
		return nil
	}

	return file
}

// getBinaryVersion tries to get the version of the binary
func (m *Manager) getBinaryVersion() (string, error) {
	cmdPath := m.config.CurrentBin()
	if cmdPath == "" {
		return "", fmt.Errorf("no binary found")
	}

	// Create a context with timeout to prevent hanging
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Try different version command patterns
	versionCmds := [][]string{
		{"version"},
		{"--version"},
		{"-version"},
		{"-v"},
	}

	for _, args := range versionCmds {
		cmd := exec.CommandContext(ctx, cmdPath, args...)
		output, err := cmd.Output()

		// If command succeeded and returned non-empty output
		if err == nil && len(output) > 0 {
			// Trim whitespace and return first line
			lines := strings.Split(strings.TrimSpace(string(output)), "\n")
			if len(lines) > 0 && lines[0] != "" {
				return lines[0], nil
			}
		}

		// Check if context timed out
		if ctx.Err() != nil {
			return "", fmt.Errorf("timeout getting version: %w", ctx.Err())
		}
	}

	return "", fmt.Errorf("unable to determine binary version")
}

// GetVersion returns the cached version or fetches it if not cached
func (m *Manager) GetVersion() string {
	m.stateMutex.RLock()
	defer m.stateMutex.RUnlock()

	// Only get version if running
	if m.state != StateRunning {
		return "unknown"
	}

	// Try to get version (with caching in the future)
	version, err := m.getBinaryVersion()
	if err != nil {
		m.logger.Debug("failed to get binary version", zap.Error(err))
		return "unknown"
	}

	return version
}

// reapZombies reaps any zombie child processes
func (m *Manager) reapZombies() {
	for {
		var status syscall.WaitStatus
		pid, err := syscall.Wait4(-1, &status, syscall.WNOHANG, nil)
		if err != nil || pid <= 0 {
			break
		}
		m.logger.Debug("reaped zombie process", zap.Int("pid", pid))
	}
}

// IsHealthy checks if the node process is healthy
func (m *Manager) IsHealthy() bool {
	m.stateMutex.RLock()
	defer m.stateMutex.RUnlock()

	if m.state != StateRunning || m.process == nil {
		return false
	}

	// Check if process is still alive
	err := m.process.Signal(syscall.Signal(0))
	return err == nil
}

// GetPID returns the process ID if running
func (m *Manager) GetPID() int {
	m.stateMutex.RLock()
	defer m.stateMutex.RUnlock()

	if m.process != nil {
		return m.process.Pid
	}
	return 0
}

// SetNodeArgs sets the node arguments for next start
func (m *Manager) SetNodeArgs(args []string) {
	m.stateMutex.Lock()
	defer m.stateMutex.Unlock()
	m.nodeArgs = args
}

// GetUptime returns the uptime of the node
func (m *Manager) GetUptime() time.Duration {
	m.stateMutex.RLock()
	defer m.stateMutex.RUnlock()

	if m.state == StateRunning && !m.startTime.IsZero() {
		return time.Since(m.startTime)
	}
	return 0
}

// GetRestartCount returns the number of restarts
func (m *Manager) GetRestartCount() int {
	m.stateMutex.RLock()
	defer m.stateMutex.RUnlock()
	return m.restartCount
}

// GetMetrics returns current metrics
func (m *Manager) GetMetrics() *metrics.MetricsSnapshot {
	if m.metricsCollector != nil {
		return m.metricsCollector.GetSnapshot()
	}
	return nil
}

// Close gracefully shuts down the manager
func (m *Manager) Close() error {
	m.cancel()
	if m.GetState() == StateRunning {
		return m.Stop()
	}
	return nil
}