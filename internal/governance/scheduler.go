package governance

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/wemix/wemixvisor/internal/config"
	"github.com/wemix/wemixvisor/pkg/logger"
	"go.uber.org/zap"
)

// UpgradeScheduler manages the scheduling and execution of upgrades
type UpgradeScheduler struct {
	cfg    *config.Config
	logger *logger.Logger

	// State management
	mu             sync.RWMutex
	upgrades       map[string]*UpgradeInfo
	scheduledQueue []*UpgradeInfo
	completedQueue []*UpgradeInfo
	currentUpgrade *UpgradeInfo

	// Configuration
	enabled               bool
	minUpgradeDelay       time.Duration
	maxConcurrentUpgrades int
	validationEnabled     bool
}

// NewUpgradeScheduler creates a new upgrade scheduler
func NewUpgradeScheduler(cfg *config.Config, logger *logger.Logger) *UpgradeScheduler {
	return &UpgradeScheduler{
		cfg:                   cfg,
		logger:                logger,
		upgrades:              make(map[string]*UpgradeInfo),
		scheduledQueue:        make([]*UpgradeInfo, 0),
		completedQueue:        make([]*UpgradeInfo, 0),
		enabled:               true,
		minUpgradeDelay:       10 * time.Minute, // Minimum 10 minutes before upgrade
		maxConcurrentUpgrades: 1,                // Only one upgrade at a time
		validationEnabled:     true,
	}
}

// Start begins the upgrade scheduler
func (us *UpgradeScheduler) Start() error {
	us.mu.Lock()
	defer us.mu.Unlock()

	if !us.enabled {
		us.logger.Info("upgrade scheduler is disabled")
		return nil
	}

	us.logger.Info("starting upgrade scheduler")

	// Load any persisted upgrade state
	if err := us.loadPersistedState(); err != nil {
		us.logger.Warn("failed to load persisted upgrade state", zap.Error(err))
	}

	us.logger.Info("upgrade scheduler started successfully")
	return nil
}

// Stop stops the upgrade scheduler
func (us *UpgradeScheduler) Stop() error {
	us.mu.Lock()
	defer us.mu.Unlock()

	us.logger.Info("stopping upgrade scheduler")

	// Save current state
	if err := us.saveCurrentState(); err != nil {
		us.logger.Warn("failed to save upgrade state", zap.Error(err))
	}

	us.enabled = false
	return nil
}

// ScheduleUpgrade schedules an upgrade from a governance proposal
func (us *UpgradeScheduler) ScheduleUpgrade(proposal *Proposal) error {
	if proposal.Type != ProposalTypeUpgrade {
		return fmt.Errorf("proposal %s is not an upgrade proposal", proposal.ID)
	}

	if proposal.UpgradeInfo == nil {
		return fmt.Errorf("proposal %s missing upgrade info", proposal.ID)
	}

	us.mu.Lock()
	defer us.mu.Unlock()

	// Validate the upgrade
	if us.validationEnabled {
		if err := us.validateUpgrade(proposal.UpgradeInfo); err != nil {
			return fmt.Errorf("upgrade validation failed: %w", err)
		}
	}

	// Create upgrade info
	upgrade := &UpgradeInfo{
		Name:          proposal.UpgradeInfo.Name,
		Height:        proposal.UpgradeHeight,
		Info:          proposal.UpgradeInfo.Info,
		Binaries:      proposal.UpgradeInfo.Binaries,
		UpgradeURL:    proposal.UpgradeInfo.UpgradeURL,
		ChecksumURL:   proposal.UpgradeInfo.ChecksumURL,
		Metadata:      proposal.UpgradeInfo.Metadata,
		Status:        UpgradeStatusScheduled,
		ScheduledTime: time.Now(),
	}

	// Store the upgrade
	us.upgrades[upgrade.Name] = upgrade
	us.scheduledQueue = append(us.scheduledQueue, upgrade)

	// Sort queue by height
	us.sortScheduledQueue()

	us.logger.Info("upgrade scheduled",
		zap.String("name", upgrade.Name),
		zap.Int64("height", upgrade.Height),
		zap.String("proposal_id", proposal.ID))

	return nil
}

// GetQueue returns the current upgrade queue
func (us *UpgradeScheduler) GetQueue() ([]*UpgradeInfo, error) {
	us.mu.RLock()
	defer us.mu.RUnlock()

	// Return a copy of the scheduled queue
	queue := make([]*UpgradeInfo, len(us.scheduledQueue))
	copy(queue, us.scheduledQueue)

	return queue, nil
}

// GetUpgrade returns a specific upgrade by name
func (us *UpgradeScheduler) GetUpgrade(name string) (*UpgradeInfo, error) {
	us.mu.RLock()
	defer us.mu.RUnlock()

	upgrade, exists := us.upgrades[name]
	if !exists {
		return nil, fmt.Errorf("upgrade not found: %s", name)
	}

	return upgrade, nil
}

// GetCurrentUpgrade returns the currently executing upgrade
func (us *UpgradeScheduler) GetCurrentUpgrade() (*UpgradeInfo, error) {
	us.mu.RLock()
	defer us.mu.RUnlock()

	if us.currentUpgrade == nil {
		return nil, fmt.Errorf("no upgrade currently in progress")
	}

	return us.currentUpgrade, nil
}

// GetNextUpgrade returns the next scheduled upgrade
func (us *UpgradeScheduler) GetNextUpgrade() (*UpgradeInfo, error) {
	us.mu.RLock()
	defer us.mu.RUnlock()

	if len(us.scheduledQueue) == 0 {
		return nil, fmt.Errorf("no upgrades scheduled")
	}

	return us.scheduledQueue[0], nil
}

// UpdateStatus updates the status of an upgrade
func (us *UpgradeScheduler) UpdateStatus(upgrade *UpgradeInfo) error {
	us.mu.Lock()
	defer us.mu.Unlock()

	existing, exists := us.upgrades[upgrade.Name]
	if !exists {
		return fmt.Errorf("upgrade not found: %s", upgrade.Name)
	}

	oldStatus := existing.Status
	existing.Status = upgrade.Status

	// Update timestamps based on status
	now := time.Now()
	switch upgrade.Status {
	case UpgradeStatusInProgress:
		if existing.StartedTime == nil {
			existing.StartedTime = &now
		}
		us.currentUpgrade = existing
	case UpgradeStatusCompleted, UpgradeStatusFailed, UpgradeStatusCancelled:
		if existing.CompletedTime == nil {
			existing.CompletedTime = &now
		}
		us.currentUpgrade = nil
		us.moveToCompleted(existing)
	}

	us.logger.Info("upgrade status updated",
		zap.String("name", upgrade.Name),
		zap.String("old_status", string(oldStatus)),
		zap.String("new_status", string(upgrade.Status)))

	return nil
}

// CancelUpgrade cancels a scheduled upgrade
func (us *UpgradeScheduler) CancelUpgrade(name string) error {
	us.mu.Lock()
	defer us.mu.Unlock()

	upgrade, exists := us.upgrades[name]
	if !exists {
		return fmt.Errorf("upgrade not found: %s", name)
	}

	if upgrade.Status == UpgradeStatusInProgress {
		return fmt.Errorf("cannot cancel upgrade in progress: %s", name)
	}

	upgrade.Status = UpgradeStatusCancelled
	now := time.Now()
	upgrade.CompletedTime = &now

	us.moveToCompleted(upgrade)

	us.logger.Info("upgrade cancelled", zap.String("name", name))
	return nil
}

// IsUpgradeReady checks if an upgrade is ready to be executed at the given height
func (us *UpgradeScheduler) IsUpgradeReady(currentHeight int64) (*UpgradeInfo, bool) {
	us.mu.RLock()
	defer us.mu.RUnlock()

	if len(us.scheduledQueue) == 0 {
		return nil, false
	}

	nextUpgrade := us.scheduledQueue[0]
	if nextUpgrade.Height <= currentHeight && nextUpgrade.Status == UpgradeStatusScheduled {
		return nextUpgrade, true
	}

	return nil, false
}

// GetUpgradeStats returns statistics about upgrades
func (us *UpgradeScheduler) GetUpgradeStats() map[string]interface{} {
	us.mu.RLock()
	defer us.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["total_upgrades"] = len(us.upgrades)
	stats["scheduled_upgrades"] = len(us.scheduledQueue)
	stats["completed_upgrades"] = len(us.completedQueue)
	stats["current_upgrade"] = us.currentUpgrade != nil

	// Count by status
	statusCounts := make(map[string]int)
	for _, upgrade := range us.upgrades {
		statusCounts[string(upgrade.Status)]++
	}
	stats["by_status"] = statusCounts

	return stats
}

// SetEnabled enables or disables the scheduler
func (us *UpgradeScheduler) SetEnabled(enabled bool) {
	us.mu.Lock()
	defer us.mu.Unlock()

	us.enabled = enabled
	us.logger.Info("upgrade scheduler enabled status changed", zap.Bool("enabled", enabled))
}

// SetMinUpgradeDelay sets the minimum delay before an upgrade can be executed
func (us *UpgradeScheduler) SetMinUpgradeDelay(delay time.Duration) {
	us.mu.Lock()
	defer us.mu.Unlock()

	us.minUpgradeDelay = delay
	us.logger.Info("minimum upgrade delay updated", zap.Duration("delay", delay))
}

// SetValidationEnabled enables or disables upgrade validation
func (us *UpgradeScheduler) SetValidationEnabled(enabled bool) {
	us.mu.Lock()
	defer us.mu.Unlock()

	us.validationEnabled = enabled
	us.logger.Info("upgrade validation enabled status changed", zap.Bool("enabled", enabled))
}

// validateUpgrade validates an upgrade before scheduling
func (us *UpgradeScheduler) validateUpgrade(upgrade *UpgradeInfo) error {
	// Check upgrade name
	if upgrade.Name == "" {
		return fmt.Errorf("upgrade name cannot be empty")
	}

	// Check if upgrade already exists
	if _, exists := us.upgrades[upgrade.Name]; exists {
		return fmt.Errorf("upgrade %s already scheduled", upgrade.Name)
	}

	// Check upgrade height
	if upgrade.Height <= 0 {
		return fmt.Errorf("invalid upgrade height: %d", upgrade.Height)
	}

	// Note: Minimum delay validation is handled at the height-based execution level.
	// Time-based validation is not applicable for height-based upgrades since
	// the actual execution time depends on block production rate.

	// Validate binaries if provided
	if upgrade.Binaries != nil {
		for platform, binary := range upgrade.Binaries {
			if binary.URL == "" {
				return fmt.Errorf("missing binary URL for platform %s", platform)
			}
			if binary.Checksum == "" {
				return fmt.Errorf("missing checksum for platform %s", platform)
			}
		}
	}

	return nil
}

// sortScheduledQueue sorts the scheduled queue by height
func (us *UpgradeScheduler) sortScheduledQueue() {
	sort.Slice(us.scheduledQueue, func(i, j int) bool {
		return us.scheduledQueue[i].Height < us.scheduledQueue[j].Height
	})
}

// moveToCompleted moves an upgrade from scheduled to completed queue
func (us *UpgradeScheduler) moveToCompleted(upgrade *UpgradeInfo) {
	// Remove from scheduled queue
	for i, scheduled := range us.scheduledQueue {
		if scheduled.Name == upgrade.Name {
			us.scheduledQueue = append(us.scheduledQueue[:i], us.scheduledQueue[i+1:]...)
			break
		}
	}

	// Add to completed queue
	us.completedQueue = append(us.completedQueue, upgrade)

	// Sort completed queue by completion time (newest first)
	sort.Slice(us.completedQueue, func(i, j int) bool {
		if us.completedQueue[i].CompletedTime == nil {
			return false
		}
		if us.completedQueue[j].CompletedTime == nil {
			return true
		}
		return us.completedQueue[i].CompletedTime.After(*us.completedQueue[j].CompletedTime)
	})
}

// loadPersistedState loads upgrade state from persistent storage
func (us *UpgradeScheduler) loadPersistedState() error {
	// This would load state from a file or database
	// For now, this is a placeholder
	us.logger.Debug("loading persisted upgrade state")
	return nil
}

// saveCurrentState saves the current upgrade state to persistent storage
func (us *UpgradeScheduler) saveCurrentState() error {
	// This would save state to a file or database
	// For now, this is a placeholder
	us.logger.Debug("saving current upgrade state")
	return nil
}

// CleanupOld removes old completed upgrades from memory
func (us *UpgradeScheduler) CleanupOld(maxAge time.Duration) error {
	us.mu.Lock()
	defer us.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	removed := 0

	newCompleted := make([]*UpgradeInfo, 0)
	for _, upgrade := range us.completedQueue {
		if upgrade.CompletedTime != nil && upgrade.CompletedTime.Before(cutoff) {
			delete(us.upgrades, upgrade.Name)
			removed++
		} else {
			newCompleted = append(newCompleted, upgrade)
		}
	}

	us.completedQueue = newCompleted

	if removed > 0 {
		us.logger.Info("cleaned up old upgrades", zap.Int("removed", removed))
	}

	return nil
}
