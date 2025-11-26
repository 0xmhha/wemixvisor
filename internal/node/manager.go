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

// Default values for manager configuration
const (
	DefaultMaxRestarts        = 5
	DefaultShutdownGrace      = 30 * time.Second
	DefaultRestartDelay       = 2 * time.Second
	DefaultAutoRestartDelay   = 5 * time.Second
	DefaultVersionTimeout     = 5 * time.Second
	DefaultProcessExitWait    = 100 * time.Millisecond
	ErrorChannelBufferSize    = 10
)

// Manager handles the lifecycle of a node process
type Manager struct {
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
func NewManager(cfg *config.Config, log *logger.Logger) *Manager {
	ctx, cancel := context.WithCancel(context.Background())

	maxRestarts := DefaultMaxRestarts
	if cfg.MaxRestarts > 0 {
		maxRestarts = cfg.MaxRestarts
	}

	healthChecker := monitor.NewHealthChecker(cfg, log)

	manager := &Manager{
		config:        cfg,
		logger:        log,
		state:         StateStopped,
		nodeOptions:   make(map[string]string),
		maxRestarts:   maxRestarts,
		healthChecker: healthChecker,
		stopCh:        make(chan struct{}),
		restartCh:     make(chan struct{}),
		errorCh:       make(chan error, ErrorChannelBufferSize),
		doneCh:        make(chan struct{}),
		ctx:           ctx,
		cancel:        cancel,
	}

	manager.initMetricsCollector(cfg, log)

	return manager
}

// initMetricsCollector initializes the metrics collector if enabled
func (m *Manager) initMetricsCollector(cfg *config.Config, log *logger.Logger) {
	if !cfg.MetricsEnabled {
		return
	}

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
	m.metricsCollector = metrics.NewCollector(collectorConfig, log)

	m.metricsCollector.SetNodeHeightCallback(func() (int64, error) {
		return 0, nil
	})
}

// Start starts the node with the given arguments
func (m *Manager) Start(args []string) error {
	m.stateMutex.Lock()

	if m.state != StateStopped {
		m.stateMutex.Unlock()
		return fmt.Errorf("node is not in stopped state: %v", m.state)
	}

	m.state = StateStarting
	m.nodeArgs = args

	cmdPath := m.config.CurrentBin()
	if cmdPath == "" {
		m.state = StateError
		m.stateMutex.Unlock()
		return fmt.Errorf("no binary path configured")
	}

	if _, err := os.Stat(cmdPath); err != nil {
		m.state = StateError
		m.stateMutex.Unlock()
		return fmt.Errorf("binary not found at %s: %w", cmdPath, err)
	}

	m.logger.Info("starting node",
		zap.String("binary", cmdPath),
		zap.Strings("args", args))

	if err := m.startProcess(cmdPath, args); err != nil {
		m.state = StateError
		m.stateMutex.Unlock()
		return err
	}

	m.stateMutex.Unlock()

	m.startMonitoring()

	m.logger.Info("node started successfully",
		zap.Int("pid", m.process.Pid),
		zap.Strings("args", args))

	return nil
}

// startProcess creates and starts the node process
func (m *Manager) startProcess(cmdPath string, args []string) error {
	cmd := exec.CommandContext(m.ctx, cmdPath, args...)
	cmd.Env = m.buildEnvironment()

	if m.config.Home != "" {
		cmd.Dir = m.config.Home
	}

	m.setupProcessOutput(cmd)

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
		Pgid:    0,
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start process: %w", err)
	}

	m.cmd = cmd
	m.process = cmd.Process
	m.startTime = time.Now()
	m.state = StateRunning

	return nil
}

// setupProcessOutput configures stdout/stderr for the process
func (m *Manager) setupProcessOutput(cmd *exec.Cmd) {
	logFile := m.getLogFile()
	if logFile != nil {
		cmd.Stdout = logFile
		cmd.Stderr = logFile
	} else {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
}

// startMonitoring starts all monitoring goroutines
func (m *Manager) startMonitoring() {
	go m.monitor()
	go m.monitorHealth()

	if m.metricsCollector != nil {
		m.metricsCollector.Start()
	}
}

// monitorHealth monitors health status updates
func (m *Manager) monitorHealth() {
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

	if err := m.sendStopSignal(pid, process); err != nil {
		return err
	}

	m.waitForShutdown(pid, process)
	m.stopMonitoring()
	m.cleanupState()

	return nil
}

// sendStopSignal sends SIGTERM to the process group
func (m *Manager) sendStopSignal(pid int, process *os.Process) error {
	if err := syscall.Kill(-pid, syscall.SIGTERM); err != nil {
		m.logger.Error("failed to send SIGTERM to process group", zap.Error(err))
		if err := process.Signal(syscall.SIGTERM); err != nil {
			m.logger.Error("failed to send SIGTERM to process", zap.Error(err))
			if err := syscall.Kill(-pid, syscall.SIGKILL); err != nil {
				return fmt.Errorf("failed to kill process: %w", err)
			}
		}
	}
	return nil
}

// waitForShutdown waits for the process to exit gracefully or forces termination
func (m *Manager) waitForShutdown(pid int, process *os.Process) {
	shutdownGrace := DefaultShutdownGrace
	if m.config.ShutdownGrace > 0 {
		shutdownGrace = m.config.ShutdownGrace
	}

	timer := time.NewTimer(shutdownGrace)
	defer timer.Stop()

	select {
	case <-m.doneCh:
		m.logger.Info("node stopped gracefully")
	case <-timer.C:
		m.logger.Warn("grace period exceeded, forcing shutdown")
		m.forceKill(pid, process)
	}
}

// forceKill forcefully terminates the process
func (m *Manager) forceKill(pid int, process *os.Process) {
	if err := syscall.Kill(-pid, syscall.SIGKILL); err != nil {
		m.logger.Error("failed to kill process group", zap.Error(err))
		if err := process.Kill(); err != nil {
			m.logger.Error("failed to kill process", zap.Error(err))
		}
	}
	time.Sleep(DefaultProcessExitWait)
}

// stopMonitoring stops health and metrics monitoring
func (m *Manager) stopMonitoring() {
	m.healthChecker.Stop()
	if m.metricsCollector != nil {
		m.metricsCollector.Stop()
	}
}

// cleanupState resets manager state after stop
func (m *Manager) cleanupState() {
	m.stateMutex.Lock()
	m.state = StateStopped
	m.cmd = nil
	m.process = nil
	if m.doneCh != nil {
		close(m.doneCh)
		m.doneCh = nil
	}
	m.stateMutex.Unlock()
}

// Restart restarts the node with the same arguments
func (m *Manager) Restart() error {
	m.logger.Info("restarting node")

	args := m.nodeArgs

	currentState := m.GetState()
	if currentState == StateRunning {
		if err := m.Stop(); err != nil {
			return fmt.Errorf("failed to stop node for restart: %w", err)
		}
	} else if currentState != StateStopped {
		m.resetState()
	}

	time.Sleep(DefaultRestartDelay)

	m.stateMutex.Lock()
	if m.doneCh == nil {
		m.doneCh = make(chan struct{})
	}
	m.stateMutex.Unlock()

	if err := m.Start(args); err != nil {
		return fmt.Errorf("failed to start node after restart: %w", err)
	}

	m.restartCount++
	m.logger.Info("node restarted successfully", zap.Int("restart_count", m.restartCount))

	return nil
}

// resetState resets manager state to stopped
func (m *Manager) resetState() {
	m.stateMutex.Lock()
	m.state = StateStopped
	m.cmd = nil
	m.process = nil
	m.stateMutex.Unlock()
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

	if status.Binary != "" && m.state == StateRunning {
		status.Version = m.GetVersion()
	}

	if m.state == StateRunning && m.healthChecker != nil {
		status.Health = m.buildHealthStatus()
	}

	return status
}

// buildHealthStatus builds health status from health checker
func (m *Manager) buildHealthStatus() *HealthStatus {
	healthStatus := m.healthChecker.GetStatus()
	health := &HealthStatus{
		Healthy:   healthStatus.Healthy,
		Timestamp: healthStatus.Timestamp,
		Checks:    make(map[string]CheckResult),
	}

	for name, check := range healthStatus.Checks {
		health.Checks[name] = CheckResult{
			Name:    check.Name,
			Healthy: check.Healthy,
			Error:   check.Error,
		}
	}

	return health
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

	err := cmd.Wait()
	go m.reapZombies()

	m.stateMutex.Lock()
	defer m.stateMutex.Unlock()

	if m.state == StateStopping {
		return
	}

	m.handleProcessCrash(err)
}

// handleProcessCrash handles unexpected process termination
func (m *Manager) handleProcessCrash(err error) {
	m.state = StateCrashed
	m.logger.Error("node process crashed unexpectedly", zap.Error(err))

	if m.doneCh != nil {
		close(m.doneCh)
		m.doneCh = nil
	}

	if m.shouldAutoRestart() {
		m.scheduleAutoRestart()
	} else {
		m.state = StateError
		if m.restartCount >= m.maxRestarts {
			m.logger.Error("max restart attempts reached",
				zap.Int("restart_count", m.restartCount),
				zap.Int("max", m.maxRestarts))
		}
	}
}

// shouldAutoRestart returns true if auto-restart should be attempted
func (m *Manager) shouldAutoRestart() bool {
	return m.config.RestartOnFailure && m.restartCount < m.maxRestarts
}

// scheduleAutoRestart schedules an automatic restart
func (m *Manager) scheduleAutoRestart() {
	m.logger.Info("attempting auto-restart",
		zap.Int("attempt", m.restartCount+1),
		zap.Int("max", m.maxRestarts))

	go func() {
		time.Sleep(DefaultAutoRestartDelay)
		if err := m.Restart(); err != nil {
			m.logger.Error("auto-restart failed", zap.Error(err))
			m.errorCh <- err
		}
	}()
}

// buildEnvironment builds the environment variables for the node
func (m *Manager) buildEnvironment() []string {
	env := os.Environ()

	if m.config.Home != "" {
		env = append(env, fmt.Sprintf("WEMIX_HOME=%s", m.config.Home))
	}
	if m.config.Network != "" {
		env = append(env, fmt.Sprintf("WEMIX_NETWORK=%s", m.config.Network))
	}

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

	logDir := filepath.Dir(logPath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		m.logger.Error("failed to create log directory", zap.Error(err))
		return nil
	}

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
		return "", fmt.Errorf("no binary path configured")
	}

	ctx, cancel := context.WithTimeout(context.Background(), DefaultVersionTimeout)
	defer cancel()

	versionCmds := [][]string{
		{"version"},
		{"--version"},
		{"-version"},
		{"-v"},
	}

	for _, args := range versionCmds {
		cmd := exec.CommandContext(ctx, cmdPath, args...)
		output, err := cmd.Output()

		if err == nil && len(output) > 0 {
			lines := strings.Split(strings.TrimSpace(string(output)), "\n")
			if len(lines) > 0 && lines[0] != "" {
				return lines[0], nil
			}
		}

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

	if m.state != StateRunning {
		return "unknown"
	}

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
