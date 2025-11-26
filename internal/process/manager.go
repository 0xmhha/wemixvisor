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

	"go.uber.org/zap"

	"github.com/wemix/wemixvisor/internal/backup"
	"github.com/wemix/wemixvisor/internal/config"
	"github.com/wemix/wemixvisor/internal/download"
	"github.com/wemix/wemixvisor/internal/hooks"
	"github.com/wemix/wemixvisor/internal/upgrade"
	"github.com/wemix/wemixvisor/pkg/logger"
	"github.com/wemix/wemixvisor/pkg/types"
)

// Default values for process manager
const (
	DefaultBackupRetention = 7 * 24 * time.Hour
)

// Manager manages the blockchain node process
type Manager struct {
	cfg        *config.Config
	logger     *logger.Logger
	watcher    *upgrade.FileWatcher
	backup     *backup.Manager
	preHook    *hooks.PreUpgradeHook
	downloader *download.Downloader

	cmd         *exec.Cmd
	mu          sync.Mutex
	running     bool
	stopChan    chan struct{}
	stoppedChan chan struct{}
}

// NewManager creates a new process manager
func NewManager(cfg *config.Config, log *logger.Logger) *Manager {
	return &Manager{
		cfg:         cfg,
		logger:      log,
		watcher:     upgrade.NewFileWatcher(cfg, log),
		backup:      backup.NewManager(cfg, log),
		preHook:     hooks.NewPreUpgradeHook(cfg, log),
		downloader:  download.NewDownloader(cfg, log),
		stopChan:    make(chan struct{}),
		stoppedChan: make(chan struct{}),
	}
}

// Run starts the managed process and monitors for upgrades
func (m *Manager) Run(ctx context.Context) error {
	if err := m.setRunningState(); err != nil {
		return err
	}
	defer m.clearRunningState()

	if err := m.ensureCurrentLink(); err != nil {
		return fmt.Errorf("failed to ensure current link: %w", err)
	}

	if err := m.watcher.Start(); err != nil {
		return fmt.Errorf("failed to start file watcher: %w", err)
	}
	defer m.watcher.Stop()

	return m.mainLoop(ctx)
}

// setRunningState marks the manager as running
func (m *Manager) setRunningState() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return fmt.Errorf("process manager already running")
	}
	m.running = true
	return nil
}

// clearRunningState clears the running state
func (m *Manager) clearRunningState() {
	m.mu.Lock()
	m.running = false
	close(m.stoppedChan)
	m.mu.Unlock()
}

// mainLoop is the main process management loop
func (m *Manager) mainLoop(ctx context.Context) error {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)

	for {
		select {
		case <-ctx.Done():
			m.logger.Info("context cancelled, stopping process manager")
			return m.stopProcess()
		case sig := <-sigChan:
			m.logger.Info("received signal", zap.String("signal", sig.String()))
			return m.handleSignal(sig)
		default:
			if err := m.runProcessCycle(ctx); err != nil {
				return err
			}
		}
	}
}

// runProcessCycle runs one cycle of the process
func (m *Manager) runProcessCycle(ctx context.Context) error {
	if err := m.runProcess(ctx); err != nil {
		m.logger.Error("process failed", zap.Error(err))

		if !m.cfg.RestartAfterUpgrade {
			return err
		}

		if upgradeInfo := m.watcher.GetCurrentUpgrade(); upgradeInfo != nil {
			return m.handleUpgradeAndRestart(upgradeInfo)
		}

		return err
	}
	return nil
}

// handleUpgradeAndRestart handles an upgrade and prepares for restart
func (m *Manager) handleUpgradeAndRestart(upgradeInfo *types.UpgradeInfo) error {
	if err := m.performUpgrade(upgradeInfo); err != nil {
		return fmt.Errorf("upgrade failed: %w", err)
	}

	if m.cfg.RestartDelay > 0 {
		m.logger.Info("waiting before restart",
			zap.Duration("delay", m.cfg.RestartDelay))
		time.Sleep(m.cfg.RestartDelay)
	}

	return nil
}

// ensureCurrentLink ensures the current symlink exists
func (m *Manager) ensureCurrentLink() error {
	currentDir := m.cfg.CurrentDir()

	if _, err := os.Stat(currentDir); err == nil {
		if _, err := os.Stat(m.cfg.CurrentBin()); err == nil {
			return nil
		}
		m.logger.Warn("current link exists but binary not found, recreating")
	}

	if err := m.cfg.SymLinkToGenesis(); err != nil {
		return fmt.Errorf("failed to create genesis link: %w", err)
	}

	if _, err := os.Stat(m.cfg.GenesisBin()); err != nil {
		return fmt.Errorf("genesis binary not found: %w", err)
	}

	m.logger.Info("created symlink to genesis")
	return nil
}

// runProcess runs the managed process
func (m *Manager) runProcess(ctx context.Context) error {
	binPath := m.cfg.CurrentBin()

	if _, err := os.Stat(binPath); err != nil {
		return fmt.Errorf("binary not found: %w", err)
	}

	m.logger.Info("starting process",
		zap.String("binary", binPath),
		zap.Strings("args", m.cfg.Args))

	m.cmd = m.createCommand(ctx, binPath)

	if err := m.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start process: %w", err)
	}

	m.logger.Info("process started", zap.Int("pid", m.cmd.Process.Pid))

	return m.waitForProcessOrUpgrade(ctx)
}

// createCommand creates the process command
func (m *Manager) createCommand(ctx context.Context, binPath string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, binPath, m.cfg.Args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return cmd
}

// waitForProcessOrUpgrade waits for the process to exit or an upgrade signal
func (m *Manager) waitForProcessOrUpgrade(ctx context.Context) error {
	processDone := make(chan error, 1)
	go func() {
		processDone <- m.cmd.Wait()
	}()

	upgradeTicker := time.NewTicker(m.cfg.PollInterval)
	defer upgradeTicker.Stop()

	for {
		select {
		case err := <-processDone:
			return m.handleProcessExit(err)
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

// handleProcessExit logs and returns the process exit error
func (m *Manager) handleProcessExit(err error) error {
	if err != nil {
		m.logger.Error("process exited with error", zap.Error(err))
	} else {
		m.logger.Info("process exited normally")
	}
	return err
}

// stopProcess stops the running process gracefully
func (m *Manager) stopProcess() error {
	if m.cmd == nil || m.cmd.Process == nil {
		return nil
	}

	m.logger.Info("stopping process", zap.Int("pid", m.cmd.Process.Pid))

	if err := m.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		m.logger.Warn("failed to send SIGTERM", zap.Error(err))
		return m.cmd.Process.Kill()
	}

	return m.waitForGracefulShutdown()
}

// waitForGracefulShutdown waits for the process to exit or times out
func (m *Manager) waitForGracefulShutdown() error {
	if m.cfg.ShutdownGrace <= 0 {
		m.cmd.Wait()
		return nil
	}

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

	if err := m.preHook.ValidateUpgrade(info); err != nil {
		return fmt.Errorf("upgrade validation failed: %w", err)
	}

	backupPath, err := m.createPreUpgradeBackup(info.Name)
	if err != nil {
		return err
	}

	if err := m.executeUpgrade(info, backupPath); err != nil {
		return err
	}

	m.cleanupOldBackups()

	m.logger.Info("upgrade completed successfully",
		zap.String("name", info.Name))

	return nil
}

// createPreUpgradeBackup creates a backup before upgrade
func (m *Manager) createPreUpgradeBackup(upgradeName string) (string, error) {
	backupPath, err := m.backup.CreateBackup(fmt.Sprintf("pre-upgrade-%s", upgradeName))
	if err != nil {
		m.logger.Error("backup failed", zap.Error(err))
		if !m.cfg.UnsafeSkipBackup {
			return "", fmt.Errorf("backup failed and unsafe_skip_backup is false: %w", err)
		}
		m.logger.Warn("continuing upgrade without backup (unsafe_skip_backup=true)")
		return "", nil
	}

	if backupPath != "" {
		m.logger.Info("backup created", zap.String("path", backupPath))
	}
	return backupPath, nil
}

// executeUpgrade executes the upgrade steps
func (m *Manager) executeUpgrade(info *types.UpgradeInfo, backupPath string) error {
	if err := m.preHook.Execute(info); err != nil {
		m.restoreBackupOnFailure(backupPath, "hook failure")
		return fmt.Errorf("pre-upgrade hook failed: %w", err)
	}

	if err := m.downloader.EnsureUpgradeBinary(info.Name); err != nil {
		m.restoreBackupOnFailure(backupPath, "download failure")
		return fmt.Errorf("failed to ensure upgrade binary: %w", err)
	}

	if err := m.cfg.SetCurrentUpgrade(info.Name); err != nil {
		m.restoreBackupOnFailure(backupPath, "symlink failure")
		return fmt.Errorf("failed to update symlink: %w", err)
	}

	return nil
}

// restoreBackupOnFailure attempts to restore backup after a failure
func (m *Manager) restoreBackupOnFailure(backupPath, reason string) {
	if backupPath == "" {
		return
	}

	m.logger.Info("attempting to restore backup after "+reason,
		zap.String("backup_path", backupPath))

	if err := m.backup.RestoreBackup(backupPath); err != nil {
		m.logger.Error("backup restore failed", zap.Error(err))
	}
}

// cleanupOldBackups removes old backups
func (m *Manager) cleanupOldBackups() {
	if err := m.backup.CleanOldBackups(DefaultBackupRetention); err != nil {
		m.logger.Warn("failed to clean old backups", zap.Error(err))
	}
}
