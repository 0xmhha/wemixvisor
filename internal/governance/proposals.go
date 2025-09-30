package governance

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/wemix/wemixvisor/pkg/logger"
	"go.uber.org/zap"
)

// ProposalTracker manages tracking of governance proposals
type ProposalTracker struct {
	client    WBFTClientInterface
	logger    *logger.Logger

	// State management
	mu                sync.RWMutex
	proposals         map[string]*Proposal
	activeProposals   map[string]*Proposal
	completedProposals map[string]*Proposal
	lastSyncHeight    int64
	lastSyncTime      time.Time

	// Configuration
	syncInterval      time.Duration
	maxProposalAge    time.Duration
	enableAutoSync    bool
}

// NewProposalTracker creates a new proposal tracker
func NewProposalTracker(client WBFTClientInterface, logger *logger.Logger) *ProposalTracker {
	return &ProposalTracker{
		client:            client,
		logger:            logger,
		proposals:         make(map[string]*Proposal),
		activeProposals:   make(map[string]*Proposal),
		completedProposals: make(map[string]*Proposal),
		syncInterval:      30 * time.Second,
		maxProposalAge:    30 * 24 * time.Hour, // 30 days
		enableAutoSync:    true,
	}
}

// Start begins tracking proposals
func (pt *ProposalTracker) Start() error {
	pt.logger.Info("starting proposal tracker")

	// Set autoSync enabled before initial sync
	pt.mu.Lock()
	pt.enableAutoSync = true
	pt.mu.Unlock()

	// Initial sync (without holding lock)
	if err := pt.syncProposals(); err != nil {
		return fmt.Errorf("initial proposal sync failed: %w", err)
	}

	pt.logger.Info("proposal tracker started successfully")
	return nil
}

// Stop stops the proposal tracker
func (pt *ProposalTracker) Stop() error {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	pt.logger.Info("stopping proposal tracker")
	pt.enableAutoSync = false

	return nil
}

// FetchLatest fetches the latest proposals from the blockchain
func (pt *ProposalTracker) FetchLatest() ([]*Proposal, error) {
	pt.logger.Debug("fetching latest proposals")

	// Fetch proposals in different statuses
	statuses := []ProposalStatus{
		ProposalStatusSubmitted,
		ProposalStatusVoting,
		ProposalStatusPassed,
		ProposalStatusRejected,
	}

	var allProposals []*Proposal
	for _, status := range statuses {
		proposals, err := pt.client.GetGovernanceProposals(status)
		if err != nil {
			pt.logger.Error("failed to fetch proposals",
				zap.String("status", string(status)),
				zap.Error(err))
			continue
		}
		allProposals = append(allProposals, proposals...)
	}

	pt.logger.Debug("fetched proposals", zap.Int("count", len(allProposals)))

	// Update internal state
	pt.mu.Lock()
	defer pt.mu.Unlock()

	newProposals := make([]*Proposal, 0)
	for _, proposal := range allProposals {
		existing, exists := pt.proposals[proposal.ID]
		if !exists {
			// New proposal
			pt.proposals[proposal.ID] = proposal
			newProposals = append(newProposals, proposal)
			pt.logger.Info("new proposal detected",
				zap.String("id", proposal.ID),
				zap.String("title", proposal.Title),
				zap.String("type", string(proposal.Type)))
		} else {
			// Update existing proposal
			if existing.Status != proposal.Status {
				pt.logger.Info("proposal status changed",
					zap.String("id", proposal.ID),
					zap.String("old_status", string(existing.Status)),
					zap.String("new_status", string(proposal.Status)))
			}
			pt.proposals[proposal.ID] = proposal
		}

		// Update active/completed maps
		pt.updateProposalCategories(proposal)
	}

	pt.lastSyncTime = time.Now()

	return newProposals, nil
}

// GetActive returns currently active proposals
func (pt *ProposalTracker) GetActive() ([]*Proposal, error) {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	proposals := make([]*Proposal, 0, len(pt.activeProposals))
	for _, proposal := range pt.activeProposals {
		proposals = append(proposals, proposal)
	}

	// Sort by submission time (newest first)
	sort.Slice(proposals, func(i, j int) bool {
		return proposals[i].SubmitTime.After(proposals[j].SubmitTime)
	})

	return proposals, nil
}

// GetAll returns all tracked proposals
func (pt *ProposalTracker) GetAll() ([]*Proposal, error) {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	proposals := make([]*Proposal, 0, len(pt.proposals))
	for _, proposal := range pt.proposals {
		proposals = append(proposals, proposal)
	}

	// Sort by submission time (newest first)
	sort.Slice(proposals, func(i, j int) bool {
		return proposals[i].SubmitTime.After(proposals[j].SubmitTime)
	})

	return proposals, nil
}

// GetByID returns a proposal by its ID
func (pt *ProposalTracker) GetByID(proposalID string) (*Proposal, error) {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	proposal, exists := pt.proposals[proposalID]
	if !exists {
		return nil, fmt.Errorf("proposal not found: %s", proposalID)
	}

	return proposal, nil
}

// GetByType returns proposals of a specific type
func (pt *ProposalTracker) GetByType(proposalType ProposalType) ([]*Proposal, error) {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	var proposals []*Proposal
	for _, proposal := range pt.proposals {
		if proposal.Type == proposalType {
			proposals = append(proposals, proposal)
		}
	}

	// Sort by submission time (newest first)
	sort.Slice(proposals, func(i, j int) bool {
		return proposals[i].SubmitTime.After(proposals[j].SubmitTime)
	})

	return proposals, nil
}

// GetUpgradeProposals returns all upgrade proposals
func (pt *ProposalTracker) GetUpgradeProposals() ([]*Proposal, error) {
	return pt.GetByType(ProposalTypeUpgrade)
}

// UpdateVotingStatus updates the voting status of a proposal
func (pt *ProposalTracker) UpdateVotingStatus(proposalID string) error {
	pt.logger.Debug("updating voting status", zap.String("proposal_id", proposalID))

	// Fetch fresh proposal data
	proposal, err := pt.client.GetProposal(proposalID)
	if err != nil {
		return fmt.Errorf("failed to fetch proposal %s: %w", proposalID, err)
	}

	pt.mu.Lock()
	defer pt.mu.Unlock()

	// Update the proposal in our tracking
	existing, exists := pt.proposals[proposalID]
	if !exists {
		pt.proposals[proposalID] = proposal
	} else {
		// Log status changes
		if existing.Status != proposal.Status {
			pt.logger.Info("proposal voting status updated",
				zap.String("id", proposalID),
				zap.String("old_status", string(existing.Status)),
				zap.String("new_status", string(proposal.Status)))
		}
		pt.proposals[proposalID] = proposal
	}

	// Update categories
	pt.updateProposalCategories(proposal)

	return nil
}

// Sync forces synchronization with the blockchain
func (pt *ProposalTracker) Sync() error {
	pt.logger.Info("forcing proposal synchronization")

	_, err := pt.FetchLatest()
	if err != nil {
		return fmt.Errorf("sync failed: %w", err)
	}

	// Update sync height
	height, err := pt.client.GetCurrentHeight()
	if err != nil {
		pt.logger.Warn("failed to get current height during sync", zap.Error(err))
	} else {
		pt.mu.Lock()
		pt.lastSyncHeight = height
		pt.mu.Unlock()
	}

	pt.logger.Info("proposal synchronization completed")
	return nil
}

// GetSyncStatus returns the current sync status
func (pt *ProposalTracker) GetSyncStatus() *SyncStatus {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	return &SyncStatus{
		LastSyncHeight:  pt.lastSyncHeight,
		LastSyncTime:    pt.lastSyncTime,
		IsSyncing:       false, // This could be enhanced with actual sync status
		ActiveProposals: len(pt.activeProposals),
	}
}

// CleanupOld removes old completed proposals from memory
func (pt *ProposalTracker) CleanupOld() error {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	cutoff := time.Now().Add(-pt.maxProposalAge)
	removed := 0

	for id, proposal := range pt.completedProposals {
		if proposal.VotingEndTime.Before(cutoff) {
			delete(pt.proposals, id)
			delete(pt.completedProposals, id)
			removed++
		}
	}

	if removed > 0 {
		pt.logger.Info("cleaned up old proposals", zap.Int("removed", removed))
	}

	return nil
}

// SetSyncInterval sets the sync interval for automatic updates
func (pt *ProposalTracker) SetSyncInterval(interval time.Duration) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	pt.syncInterval = interval
	pt.logger.Info("sync interval updated", zap.Duration("interval", interval))
}

// SetMaxProposalAge sets the maximum age for keeping completed proposals
func (pt *ProposalTracker) SetMaxProposalAge(age time.Duration) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	pt.maxProposalAge = age
	pt.logger.Info("max proposal age updated", zap.Duration("age", age))
}

// GetProposalStats returns statistics about tracked proposals
func (pt *ProposalTracker) GetProposalStats() map[string]interface{} {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["total_proposals"] = len(pt.proposals)
	stats["active_proposals"] = len(pt.activeProposals)
	stats["completed_proposals"] = len(pt.completedProposals)
	stats["last_sync_time"] = pt.lastSyncTime
	stats["last_sync_height"] = pt.lastSyncHeight

	// Count by type
	typeCounts := make(map[string]int)
	statusCounts := make(map[string]int)

	for _, proposal := range pt.proposals {
		typeCounts[string(proposal.Type)]++
		statusCounts[string(proposal.Status)]++
	}

	stats["by_type"] = typeCounts
	stats["by_status"] = statusCounts

	return stats
}

// syncProposals performs the actual synchronization
func (pt *ProposalTracker) syncProposals() error {
	if !pt.enableAutoSync {
		return nil
	}

	_, err := pt.FetchLatest()
	return err
}

// updateProposalCategories updates the active and completed proposal maps
func (pt *ProposalTracker) updateProposalCategories(proposal *Proposal) {
	// Remove from both maps first
	delete(pt.activeProposals, proposal.ID)
	delete(pt.completedProposals, proposal.ID)

	// Add to appropriate map
	switch proposal.Status {
	case ProposalStatusSubmitted, ProposalStatusVoting:
		pt.activeProposals[proposal.ID] = proposal
	case ProposalStatusPassed, ProposalStatusRejected, ProposalStatusFailed, ProposalStatusExpired:
		pt.completedProposals[proposal.ID] = proposal
	}
}

// isProposalActive returns true if the proposal is still active
func (pt *ProposalTracker) isProposalActive(proposal *Proposal) bool {
	return proposal.Status == ProposalStatusSubmitted || proposal.Status == ProposalStatusVoting
}

// isProposalCompleted returns true if the proposal is completed
func (pt *ProposalTracker) isProposalCompleted(proposal *Proposal) bool {
	return proposal.Status == ProposalStatusPassed ||
		   proposal.Status == ProposalStatusRejected ||
		   proposal.Status == ProposalStatusFailed ||
		   proposal.Status == ProposalStatusExpired
}