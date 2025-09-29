package node

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"
	"github.com/wemix/wemixvisor/internal/config"
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
	startTime    time.Time
	restartCount int
	maxRestarts  int

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

	return &Manager{
		config:      cfg,
		logger:      logger,
		state:       StateStopped,
		nodeOptions: make(map[string]string),
		maxRestarts: maxRestarts,
		stopCh:      make(chan struct{}),
		restartCh:   make(chan struct{}),
		errorCh:     make(chan error, 10),
		doneCh:      make(chan struct{}),
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Start starts the node with the given arguments
func (m *Manager) Start(args []string) error {
	m.stateMutex.Lock()
	defer m.stateMutex.Unlock()

	if m.state != StateStopped {
		return fmt.Errorf("node is not in stopped state: %v", m.state)
	}

	m.state = StateStarting
	m.nodeArgs = args

	// Build command
	cmdPath := m.config.CurrentBin()
	if cmdPath == "" {
		m.state = StateError
		return fmt.Errorf("no binary found")
	}

	// Ensure binary exists and is executable
	if _, err := os.Stat(cmdPath); err != nil {
		m.state = StateError
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

	// Set process group for clean shutdown
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	// Start the process
	if err := cmd.Start(); err != nil {
		m.state = StateError
		return fmt.Errorf("failed to start node: %w", err)
	}

	m.cmd = cmd
	m.process = cmd.Process
	m.startTime = time.Now()
	m.state = StateRunning

	// Start monitoring goroutine
	go m.monitor()

	m.logger.Info("node started successfully",
		zap.Int("pid", m.process.Pid),
		zap.Strings("args", args))

	return nil
}

// Stop stops the node gracefully
func (m *Manager) Stop() error {
	m.stateMutex.Lock()
	defer m.stateMutex.Unlock()

	if m.state != StateRunning {
		return fmt.Errorf("node is not running")
	}

	m.state = StateStopping
	m.logger.Info("stopping node", zap.Int("pid", m.process.Pid))

	// Send SIGTERM for graceful shutdown
	if err := m.process.Signal(syscall.SIGTERM); err != nil {
		m.logger.Error("failed to send SIGTERM", zap.Error(err))
		// Try to kill the process group
		if err := syscall.Kill(-m.process.Pid, syscall.SIGKILL); err != nil {
			return fmt.Errorf("failed to kill process: %w", err)
		}
	}

	// Wait for graceful shutdown or timeout
	done := make(chan error, 1)
	go func() {
		done <- m.cmd.Wait()
	}()

	shutdownGrace := 30 * time.Second
	if m.config.ShutdownGrace > 0 {
		shutdownGrace = m.config.ShutdownGrace
	}

	select {
	case <-done:
		m.logger.Info("node stopped gracefully")
	case <-time.After(shutdownGrace):
		m.logger.Warn("grace period exceeded, forcing shutdown")
		if err := m.process.Kill(); err != nil {
			m.logger.Error("failed to kill process", zap.Error(err))
		}
		// Kill the process group
		syscall.Kill(-m.process.Pid, syscall.SIGKILL)
	}

	m.state = StateStopped
	m.cmd = nil
	m.process = nil
	close(m.doneCh)

	return nil
}

// Restart restarts the node with the same arguments
func (m *Manager) Restart() error {
	m.logger.Info("restarting node")

	// Save current args
	args := m.nodeArgs

	// Stop the node
	if m.GetState() == StateRunning {
		if err := m.Stop(); err != nil {
			return fmt.Errorf("failed to stop node: %w", err)
		}
	}

	// Wait briefly before restart
	time.Sleep(2 * time.Second)

	// Reset done channel
	m.doneCh = make(chan struct{})

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
		// Skip version check for now - will be implemented properly later
		// if version, err := m.getBinaryVersion(); err == nil {
		// 	status.Version = version
		// }
	}

	return status
}

// Wait waits for the node to exit
func (m *Manager) Wait() <-chan struct{} {
	return m.doneCh
}

// monitor monitors the node process and handles crashes
func (m *Manager) monitor() {
	if m.cmd == nil {
		return
	}

	// Wait for process to exit
	err := m.cmd.Wait()

	m.stateMutex.Lock()
	defer m.stateMutex.Unlock()

	// Check if this was an expected shutdown
	if m.state == StateStopping {
		return
	}

	// Process crashed unexpectedly
	m.state = StateCrashed
	m.logger.Error("node process crashed unexpectedly", zap.Error(err))

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

	// Try to get version with --version flag
	cmd := exec.Command(cmdPath, "version")
	output, err := cmd.Output()
	if err != nil {
		// Try with --version
		cmd = exec.Command(cmdPath, "--version")
		output, err = cmd.Output()
		if err != nil {
			return "", err
		}
	}

	return string(output), nil
}

// Close gracefully shuts down the manager
func (m *Manager) Close() error {
	m.cancel()
	if m.GetState() == StateRunning {
		return m.Stop()
	}
	return nil
}