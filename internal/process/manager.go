package process

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/wemix/wemixvisor/internal/backup"
	"github.com/wemix/wemixvisor/internal/config"
	"github.com/wemix/wemixvisor/internal/hooks"
	"github.com/wemix/wemixvisor/internal/upgrade"
	"github.com/wemix/wemixvisor/pkg/logger"
	"github.com/wemix/wemixvisor/pkg/types"
	"go.uber.org/zap"
)

// Manager manages the blockchain node process
type Manager struct {
	cfg         *config.Config
	logger      *logger.Logger
	watcher     *upgrade.FileWatcher
	backup      *backup.Manager
	preHook     *hooks.PreUpgradeHook
	cmd         *exec.Cmd
	mu          sync.Mutex
	running     bool
	stopChan    chan struct{}
	stoppedChan chan struct{}
}

// NewManager creates a new process manager
func NewManager(cfg *config.Config, logger *logger.Logger) *Manager {
	return &Manager{
		cfg:         cfg,
		logger:      logger,
		watcher:     upgrade.NewFileWatcher(cfg, logger),
		backup:      backup.NewManager(cfg, logger),
		preHook:     hooks.NewPreUpgradeHook(cfg, logger),
		stopChan:    make(chan struct{}),
		stoppedChan: make(chan struct{}),
	}
}

// Run starts the managed process and monitors for upgrades
func (m *Manager) Run(ctx context.Context) error {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return fmt.Errorf("process manager already running")
	}
	m.running = true
	m.mu.Unlock()

	defer func() {
		m.mu.Lock()
		m.running = false
		close(m.stoppedChan)
		m.mu.Unlock()
	}()

	// Ensure current symlink exists
	if err := m.ensureCurrentLink(); err != nil {
		return fmt.Errorf("failed to ensure current link: %w", err)
	}

	// Start the file watcher
	if err := m.watcher.Start(); err != nil {
		return fmt.Errorf("failed to start file watcher: %w", err)
	}
	defer m.watcher.Stop()

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)

	// Main loop
	for {
		select {
		case <-ctx.Done():
			m.logger.Info("context cancelled, stopping process manager")
			return m.stopProcess()
		case sig := <-sigChan:
			m.logger.Info("received signal", zap.String("signal", sig.String()))
			return m.handleSignal(sig)
		default:
			// Run the process
			if err := m.runProcess(ctx); err != nil {
				m.logger.Error("process failed", zap.Error(err))

				// Check if we should restart
				if !m.cfg.RestartAfterUpgrade {
					return err
				}

				// Check for upgrade
				if upgradeInfo := m.watcher.GetCurrentUpgrade(); upgradeInfo != nil {
					if err := m.performUpgrade(upgradeInfo); err != nil {
						return fmt.Errorf("upgrade failed: %w", err)
					}

					// Apply restart delay
					if m.cfg.RestartDelay > 0 {
						m.logger.Info("waiting before restart",
							zap.Duration("delay", m.cfg.RestartDelay))
						time.Sleep(m.cfg.RestartDelay)
					}

					continue // Restart with new binary
				}

				return err // No upgrade available, exit
			}
		}
	}
}

// ensureCurrentLink ensures the current symlink exists
func (m *Manager) ensureCurrentLink() error {
	currentDir := m.cfg.CurrentDir()

	// Check if current link exists
	if _, err := os.Stat(currentDir); err == nil {
		// Link exists, verify it's valid
		if _, err := os.Stat(m.cfg.CurrentBin()); err == nil {
			return nil // Valid link exists
		}
		m.logger.Warn("current link exists but binary not found, recreating")
	}

	// Create link to genesis
	if err := m.cfg.SymLinkToGenesis(); err != nil {
		return fmt.Errorf("failed to create genesis link: %w", err)
	}

	// Verify genesis binary exists
	if _, err := os.Stat(m.cfg.GenesisBin()); err != nil {
		return fmt.Errorf("genesis binary not found: %w", err)
	}

	m.logger.Info("created symlink to genesis")
	return nil
}

// runProcess runs the managed process
func (m *Manager) runProcess(ctx context.Context) error {
	binPath := m.cfg.CurrentBin()

	// Verify binary exists
	if _, err := os.Stat(binPath); err != nil {
		return fmt.Errorf("binary not found: %w", err)
	}

	m.logger.Info("starting process",
		zap.String("binary", binPath),
		zap.Strings("args", m.cfg.Args))

	// Create the command
	m.cmd = exec.CommandContext(ctx, binPath, m.cfg.Args...)
	m.cmd.Stdout = os.Stdout
	m.cmd.Stderr = os.Stderr
	m.cmd.Stdin = os.Stdin

	// Set process group for proper signal handling
	m.cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	// Start the process
	if err := m.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start process: %w", err)
	}

	m.logger.Info("process started", zap.Int("pid", m.cmd.Process.Pid))

	// Wait for process to exit or upgrade signal
	processDone := make(chan error, 1)
	go func() {
		processDone <- m.cmd.Wait()
	}()

	upgradeTicker := time.NewTicker(m.cfg.PollInterval)
	defer upgradeTicker.Stop()

	for {
		select {
		case err := <-processDone:
			if err != nil {
				m.logger.Error("process exited with error", zap.Error(err))
			} else {
				m.logger.Info("process exited normally")
			}
			return err

		case <-upgradeTicker.C:
			if m.watcher.NeedsUpdate() {
				m.logger.Info("upgrade detected, stopping process")
				return m.stopProcess()
			}

		case <-ctx.Done():
			m.logger.Info("context cancelled, stopping process")
			return m.stopProcess()
		}
	}
}

// stopProcess stops the running process gracefully
func (m *Manager) stopProcess() error {
	if m.cmd == nil || m.cmd.Process == nil {
		return nil
	}

	m.logger.Info("stopping process", zap.Int("pid", m.cmd.Process.Pid))

	// Send SIGTERM for graceful shutdown
	if err := m.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		m.logger.Warn("failed to send SIGTERM", zap.Error(err))
		// Try to kill the process
		return m.cmd.Process.Kill()
	}

	// Wait for graceful shutdown or timeout
	if m.cfg.ShutdownGrace > 0 {
		done := make(chan struct{})
		go func() {
			m.cmd.Wait()
			close(done)
		}()

		select {
		case <-done:
			m.logger.Info("process stopped gracefully")
		case <-time.After(m.cfg.ShutdownGrace):
			m.logger.Warn("grace period exceeded, killing process")
			if err := m.cmd.Process.Kill(); err != nil {
				return fmt.Errorf("failed to kill process: %w", err)
			}
		}
	} else {
		// No grace period, wait for process to exit
		m.cmd.Wait()
	}

	return nil
}

// handleSignal handles system signals
func (m *Manager) handleSignal(sig os.Signal) error {
	switch sig {
	case syscall.SIGTERM, syscall.SIGINT:
		m.logger.Info("received termination signal, stopping")
		return m.stopProcess()
	case syscall.SIGQUIT:
		m.logger.Info("received quit signal, stopping immediately")
		if m.cmd != nil && m.cmd.Process != nil {
			return m.cmd.Process.Kill()
		}
	}
	return nil
}

// performUpgrade performs an upgrade to a new binary version
func (m *Manager) performUpgrade(info *types.UpgradeInfo) error {
	m.logger.Info("performing upgrade",
		zap.String("name", info.Name),
		zap.Int64("height", info.Height))

	// Step 1: Validate the upgrade
	if err := m.preHook.ValidateUpgrade(info); err != nil {
		return fmt.Errorf("upgrade validation failed: %w", err)
	}

	// Step 2: Create backup before upgrade
	backupPath, err := m.backup.CreateBackup(fmt.Sprintf("pre-upgrade-%s", info.Name))
	if err != nil {
		m.logger.Error("backup failed", zap.Error(err))
		if !m.cfg.UnsafeSkipBackup {
			return fmt.Errorf("backup failed and unsafe_skip_backup is false: %w", err)
		}
		m.logger.Warn("continuing upgrade without backup (unsafe_skip_backup=true)")
	} else if backupPath != "" {
		m.logger.Info("backup created", zap.String("path", backupPath))
	}

	// Step 3: Run pre-upgrade hook
	if err := m.preHook.Execute(info); err != nil {
		m.logger.Error("pre-upgrade hook failed", zap.Error(err))
		// Attempt to restore backup if hook fails
		if backupPath != "" {
			m.logger.Info("attempting to restore backup after hook failure")
			if restoreErr := m.backup.RestoreBackup(backupPath); restoreErr != nil {
				m.logger.Error("backup restore failed", zap.Error(restoreErr))
			}
		}
		return fmt.Errorf("pre-upgrade hook failed: %w", err)
	}

	// Step 4: Verify upgrade binary exists
	upgradeBin := m.cfg.UpgradeBin(info.Name)
	if _, err := os.Stat(upgradeBin); err != nil {
		if !m.cfg.AllowDownloadBinaries {
			return fmt.Errorf("upgrade binary not found and auto-download disabled: %w", err)
		}
		// TODO: Implement binary download
		return fmt.Errorf("binary download not yet implemented")
	}

	// Step 5: Update the symlink
	if err := m.cfg.SetCurrentUpgrade(info.Name); err != nil {
		// Attempt to restore backup if symlink update fails
		if backupPath != "" {
			m.logger.Info("attempting to restore backup after symlink failure")
			if restoreErr := m.backup.RestoreBackup(backupPath); restoreErr != nil {
				m.logger.Error("backup restore failed", zap.Error(restoreErr))
			}
		}
		return fmt.Errorf("failed to update symlink: %w", err)
	}

	// Step 6: Clean old backups (optional)
	if err := m.backup.CleanOldBackups(7 * 24 * time.Hour); err != nil {
		m.logger.Warn("failed to clean old backups", zap.Error(err))
		// Non-critical error, continue
	}

	m.logger.Info("upgrade completed successfully",
		zap.String("name", info.Name))

	return nil
}