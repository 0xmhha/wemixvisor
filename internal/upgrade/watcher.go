package upgrade

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/wemix/wemixvisor/internal/config"
	"github.com/wemix/wemixvisor/pkg/logger"
	"github.com/wemix/wemixvisor/pkg/types"
	"go.uber.org/zap"
)

// FileWatcher monitors the upgrade-info.json file for changes
type FileWatcher struct {
	cfg          *config.Config
	logger       *logger.Logger
	filename     string
	interval     time.Duration
	lastModTime  time.Time
	currentInfo  *types.UpgradeInfo
	needsUpdate  bool
	initialized  bool
	mu           sync.RWMutex
	stopChan     chan struct{}
	stoppedChan  chan struct{}
}

// NewFileWatcher creates a new FileWatcher instance
func NewFileWatcher(cfg *config.Config, logger *logger.Logger) *FileWatcher {
	return &FileWatcher{
		cfg:         cfg,
		logger:      logger,
		filename:    cfg.UpgradeInfoFilePath(),
		interval:    cfg.PollInterval,
		stopChan:    make(chan struct{}),
		stoppedChan: make(chan struct{}),
	}
}

// Start begins monitoring for upgrades
func (fw *FileWatcher) Start() error {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	if fw.initialized {
		return fmt.Errorf("file watcher already started")
	}

	// Check if upgrade info file exists
	if info, err := fw.checkFile(); err == nil && info != nil {
		fw.currentInfo = info
		fw.logger.Info("loaded existing upgrade info",
			zap.String("name", info.Name),
			zap.Int64("height", info.Height))
	}

	fw.initialized = true

	// Start monitoring in background
	go fw.monitor()

	fw.logger.Info("started upgrade file watcher",
		zap.String("file", fw.filename),
		zap.Duration("interval", fw.interval))

	return nil
}

// Stop stops the file watcher
func (fw *FileWatcher) Stop() {
	fw.mu.Lock()
	if !fw.initialized {
		fw.mu.Unlock()
		return
	}
	fw.initialized = false
	fw.mu.Unlock()

	close(fw.stopChan)
	<-fw.stoppedChan

	fw.logger.Info("stopped upgrade file watcher")
}

// monitor continuously checks for upgrade file changes
func (fw *FileWatcher) monitor() {
	defer close(fw.stoppedChan)

	ticker := time.NewTicker(fw.interval)
	defer ticker.Stop()

	for {
		select {
		case <-fw.stopChan:
			return
		case <-ticker.C:
			if fw.checkForUpdate() {
				fw.logger.Info("upgrade detected",
					zap.String("name", fw.currentInfo.Name),
					zap.Int64("height", fw.currentInfo.Height))
			}
		}
	}
}

// checkForUpdate checks if the upgrade file has been updated
func (fw *FileWatcher) checkForUpdate() bool {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	info, err := fw.checkFile()
	if err != nil {
		// File doesn't exist or is invalid - not an error condition
		return false
	}

	// If no upgrade info found (file doesn't exist), no update
	if info == nil {
		return false
	}

	// Check if this is a new upgrade
	if fw.currentInfo == nil {
		// First upgrade detected
		fw.currentInfo = info
		fw.needsUpdate = true
		return true
	}

	// Check if upgrade has changed
	if info.Name != fw.currentInfo.Name ||
		info.Height != fw.currentInfo.Height {
		fw.currentInfo = info
		fw.needsUpdate = true
		return true
	}

	return false
}

// checkFile checks the upgrade info file
func (fw *FileWatcher) checkFile() (*types.UpgradeInfo, error) {
	// Check if file exists
	stat, err := os.Stat(fw.filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // File doesn't exist yet
		}
		return nil, fmt.Errorf("failed to stat upgrade file: %w", err)
	}

	// Check if file is empty
	if stat.Size() == 0 {
		return nil, fmt.Errorf("upgrade file is empty")
	}

	// Check modification time
	if !stat.ModTime().After(fw.lastModTime) {
		return nil, nil // File hasn't been modified
	}

	// Parse the file
	info, err := types.ParseUpgradeInfoFile(fw.filename)
	if err != nil {
		return nil, fmt.Errorf("failed to parse upgrade file: %w", err)
	}

	fw.lastModTime = stat.ModTime()
	return info, nil
}

// GetCurrentUpgrade returns the current upgrade info
func (fw *FileWatcher) GetCurrentUpgrade() *types.UpgradeInfo {
	fw.mu.RLock()
	defer fw.mu.RUnlock()
	return fw.currentInfo
}

// NeedsUpdate returns true if an update is needed
func (fw *FileWatcher) NeedsUpdate() bool {
	fw.mu.RLock()
	defer fw.mu.RUnlock()
	return fw.needsUpdate
}

// ClearUpdateFlag clears the update flag
func (fw *FileWatcher) ClearUpdateFlag() {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	fw.needsUpdate = false
}

// WaitForUpgrade blocks until an upgrade is detected or context is cancelled
func (fw *FileWatcher) WaitForUpgrade() *types.UpgradeInfo {
	ticker := time.NewTicker(fw.interval)
	defer ticker.Stop()

	for {
		select {
		case <-fw.stopChan:
			return nil
		case <-ticker.C:
			if fw.NeedsUpdate() {
				info := fw.GetCurrentUpgrade()
				fw.ClearUpdateFlag()
				return info
			}
		}
	}
}

// CheckHeight checks the current block height (placeholder for WBFT integration)
func (fw *FileWatcher) CheckHeight() (int64, error) {
	// TODO: Implement actual height checking via RPC
	// For now, return a mock value
	return 0, nil
}

// CreateUpgradeDir creates the directory for an upgrade if it doesn't exist
func CreateUpgradeDir(cfg *config.Config, name string) error {
	upgradeDir := cfg.UpgradeDir(name)
	binDir := filepath.Join(upgradeDir, "bin")

	if err := os.MkdirAll(binDir, 0755); err != nil {
		return fmt.Errorf("failed to create upgrade directory: %w", err)
	}

	return nil
}