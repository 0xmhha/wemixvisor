package governance

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/wemix/wemixvisor/internal/config"
	"github.com/wemix/wemixvisor/pkg/logger"
	"go.uber.org/zap"
)

// Monitor watches for governance proposals and upgrade events
type Monitor struct {
	cfg       *config.Config
	logger    *logger.Logger
	rpcClient WBFTClientInterface
	tracker   *ProposalTracker
	scheduler *UpgradeScheduler
	notifier  *Notifier

	// State management
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	mu     sync.RWMutex

	// Configuration
	pollInterval    time.Duration
	proposalTimeout time.Duration
	enabled         bool
}

// NewMonitor creates a new governance monitor
func NewMonitor(cfg *config.Config, logger *logger.Logger) *Monitor {
	ctx, cancel := context.WithCancel(context.Background())

	return &Monitor{
		cfg:    cfg,
		logger: logger,
		ctx:    ctx,
		cancel: cancel,

		// Default configuration
		pollInterval:    30 * time.Second,
		proposalTimeout: 24 * time.Hour,
		enabled:         true,
	}
}

// Start begins monitoring governance proposals
func (m *Monitor) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.enabled {
		m.logger.Info("governance monitoring is disabled")
		return nil
	}

	m.logger.Info("starting governance monitor")

	// Initialize components
	if err := m.initializeComponents(); err != nil {
		return fmt.Errorf("failed to initialize components: %w", err)
	}

	// Start monitoring goroutines
	m.wg.Add(3)
	go m.monitorProposals()
	go m.monitorVoting()
	go m.scheduleUpgrades()

	m.logger.Info("governance monitor started successfully")
	return nil
}

// Stop stops the governance monitor
func (m *Monitor) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.logger.Info("stopping governance monitor")

	// Cancel context and wait for goroutines
	m.cancel()
	m.wg.Wait()

	// Cleanup components
	if m.rpcClient != nil {
		m.rpcClient.Close()
	}

	m.logger.Info("governance monitor stopped")
	return nil
}

// GetProposals returns currently tracked proposals
func (m *Monitor) GetProposals() ([]*Proposal, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.tracker == nil {
		return nil, fmt.Errorf("proposal tracker not initialized")
	}

	return m.tracker.GetActive()
}

// GetUpgradeQueue returns pending upgrades
func (m *Monitor) GetUpgradeQueue() ([]*UpgradeInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.scheduler == nil {
		return nil, fmt.Errorf("upgrade scheduler not initialized")
	}

	return m.scheduler.GetQueue()
}

// ForceSync forces synchronization with the blockchain
func (m *Monitor) ForceSync() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.tracker == nil {
		return fmt.Errorf("proposal tracker not initialized")
	}

	m.logger.Info("forcing governance sync")
	return m.tracker.Sync()
}

// initializeComponents initializes all monitor components
func (m *Monitor) initializeComponents() error {
	var err error

	// Initialize RPC client
	m.rpcClient, err = NewWBFTClient(m.cfg.RPCAddress, m.logger)
	if err != nil {
		return fmt.Errorf("failed to create RPC client: %w", err)
	}

	// Initialize proposal tracker
	m.tracker = NewProposalTracker(m.rpcClient, m.logger)

	// Initialize upgrade scheduler
	m.scheduler = NewUpgradeScheduler(m.cfg, m.logger)

	// Initialize notifier
	m.notifier = NewNotifier(m.logger)

	return nil
}

// monitorProposals monitors for new governance proposals
func (m *Monitor) monitorProposals() {
	defer m.wg.Done()

	ticker := time.NewTicker(m.pollInterval)
	defer ticker.Stop()

	m.logger.Info("starting proposal monitoring")

	for {
		select {
		case <-m.ctx.Done():
			m.logger.Info("proposal monitoring stopped")
			return

		case <-ticker.C:
			if err := m.checkNewProposals(); err != nil {
				m.logger.Error("failed to check new proposals", zap.Error(err))
			}
		}
	}
}

// monitorVoting monitors voting progress on active proposals
func (m *Monitor) monitorVoting() {
	defer m.wg.Done()

	ticker := time.NewTicker(m.pollInterval / 2) // Check voting more frequently
	defer ticker.Stop()

	m.logger.Info("starting voting monitoring")

	for {
		select {
		case <-m.ctx.Done():
			m.logger.Info("voting monitoring stopped")
			return

		case <-ticker.C:
			if err := m.checkVotingProgress(); err != nil {
				m.logger.Error("failed to check voting progress", zap.Error(err))
			}
		}
	}
}

// scheduleUpgrades manages the upgrade scheduling process
func (m *Monitor) scheduleUpgrades() {
	defer m.wg.Done()

	ticker := time.NewTicker(time.Minute) // Check upgrades every minute
	defer ticker.Stop()

	m.logger.Info("starting upgrade scheduling")

	for {
		select {
		case <-m.ctx.Done():
			m.logger.Info("upgrade scheduling stopped")
			return

		case <-ticker.C:
			if err := m.processUpgradeQueue(); err != nil {
				m.logger.Error("failed to process upgrade queue", zap.Error(err))
			}
		}
	}
}

// checkNewProposals checks for new governance proposals
func (m *Monitor) checkNewProposals() error {
	proposals, err := m.tracker.FetchLatest()
	if err != nil {
		return fmt.Errorf("failed to fetch latest proposals: %w", err)
	}

	for _, proposal := range proposals {
		if proposal.Type == ProposalTypeUpgrade {
			m.logger.Info("new upgrade proposal detected",
				zap.String("id", proposal.ID),
				zap.String("title", proposal.Title),
				zap.Int64("height", proposal.UpgradeHeight))

			// Notify about new proposal
			m.notifier.NotifyNewProposal(proposal)

			// Pre-validate the upgrade
			if err := m.validateUpgradeProposal(proposal); err != nil {
				m.logger.Warn("upgrade proposal validation failed",
					zap.String("id", proposal.ID),
					zap.Error(err))
			}
		}
	}

	return nil
}

// checkVotingProgress monitors voting progress
func (m *Monitor) checkVotingProgress() error {
	proposals, err := m.tracker.GetActive()
	if err != nil {
		return fmt.Errorf("failed to get active proposals: %w", err)
	}

	for _, proposal := range proposals {
		// Update voting status
		if err := m.tracker.UpdateVotingStatus(proposal.ID); err != nil {
			m.logger.Error("failed to update voting status",
				zap.String("id", proposal.ID),
				zap.Error(err))
			continue
		}

		// Check if proposal passed
		if proposal.Status == ProposalStatusPassed && proposal.Type == ProposalTypeUpgrade {
			m.logger.Info("upgrade proposal passed",
				zap.String("id", proposal.ID),
				zap.Int64("height", proposal.UpgradeHeight))

			// Schedule the upgrade
			if err := m.scheduler.ScheduleUpgrade(proposal); err != nil {
				m.logger.Error("failed to schedule upgrade",
					zap.String("id", proposal.ID),
					zap.Error(err))
			} else {
				m.notifier.NotifyUpgradeScheduled(proposal)
			}
		}
	}

	return nil
}

// processUpgradeQueue processes pending upgrades
func (m *Monitor) processUpgradeQueue() error {
	currentHeight, err := m.rpcClient.GetCurrentHeight()
	if err != nil {
		return fmt.Errorf("failed to get current height: %w", err)
	}

	upgrades, err := m.scheduler.GetQueue()
	if err != nil {
		return fmt.Errorf("failed to get upgrade queue: %w", err)
	}

	for _, upgrade := range upgrades {
		// Check if upgrade should be triggered
		if upgrade.Height <= currentHeight && upgrade.Status == UpgradeStatusScheduled {
			m.logger.Info("triggering upgrade",
				zap.String("name", upgrade.Name),
				zap.Int64("height", upgrade.Height))

			if err := m.triggerUpgrade(upgrade); err != nil {
				m.logger.Error("failed to trigger upgrade",
					zap.String("name", upgrade.Name),
					zap.Error(err))
				upgrade.Status = UpgradeStatusFailed
			} else {
				upgrade.Status = UpgradeStatusInProgress
				m.notifier.NotifyUpgradeTriggered(upgrade)
			}

			// Update upgrade status
			if err := m.scheduler.UpdateStatus(upgrade); err != nil {
				m.logger.Error("failed to update upgrade status",
					zap.String("name", upgrade.Name),
					zap.Error(err))
			}
		}
	}

	return nil
}

// validateUpgradeProposal validates an upgrade proposal
func (m *Monitor) validateUpgradeProposal(proposal *Proposal) error {
	// Check if upgrade info is valid
	if proposal.UpgradeInfo == nil {
		return fmt.Errorf("missing upgrade info")
	}

	// Validate height only if RPC client is available
	if m.rpcClient != nil {
		currentHeight, err := m.rpcClient.GetCurrentHeight()
		if err != nil {
			return fmt.Errorf("failed to get current height: %w", err)
		}

		if proposal.UpgradeHeight <= currentHeight {
			return fmt.Errorf("upgrade height %d is not in the future (current: %d)",
				proposal.UpgradeHeight, currentHeight)
		}
	}

	// Validate binary info
	if proposal.UpgradeInfo.Binaries != nil {
		for platform, binary := range proposal.UpgradeInfo.Binaries {
			if binary.URL == "" {
				return fmt.Errorf("missing binary URL for platform %s", platform)
			}
			if binary.Checksum == "" {
				return fmt.Errorf("missing checksum for platform %s", platform)
			}
		}
	}

	m.logger.Info("upgrade proposal validation passed",
		zap.String("id", proposal.ID))

	return nil
}

// triggerUpgrade triggers an upgrade
func (m *Monitor) triggerUpgrade(upgrade *UpgradeInfo) error {
	// Create upgrade-info.json file
	upgradeInfoPath := m.cfg.UpgradeInfoFilePath()

	upgradeData := map[string]interface{}{
		"name":   upgrade.Name,
		"height": upgrade.Height,
		"info":   upgrade.Info,
	}

	if err := writeUpgradeInfo(upgradeInfoPath, upgradeData); err != nil {
		return fmt.Errorf("failed to write upgrade info: %w", err)
	}

	m.logger.Info("upgrade info file created",
		zap.String("path", upgradeInfoPath),
		zap.String("name", upgrade.Name))

	// Notify about the upgrade trigger
	if m.notifier != nil {
		m.notifier.NotifyUpgradeTriggered(upgrade)
	}

	return nil
}

// SetEnabled enables or disables governance monitoring
func (m *Monitor) SetEnabled(enabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.enabled = enabled
	m.logger.Info("governance monitoring enabled status changed",
		zap.Bool("enabled", enabled))
}

// SetPollInterval sets the polling interval
func (m *Monitor) SetPollInterval(interval time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.pollInterval = interval
	m.logger.Info("governance poll interval changed",
		zap.Duration("interval", interval))
}