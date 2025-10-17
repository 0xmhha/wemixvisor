package governance

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/wemix/wemixvisor/internal/config"
	"github.com/wemix/wemixvisor/pkg/logger"
)

// Integration tests that test multiple components together

func TestIntegration_MonitorWithMockClient(t *testing.T) {
	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "wemixvisor-test-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		Home:       tmpDir,
		RPCAddress: "http://localhost:8545", // Will be replaced with mock
	}
	testLogger := logger.NewTestLogger()
	monitor := NewMonitor(cfg, testLogger)

	// Replace client with mock
	mockClient := &MockWBFTClient{}
	monitor.rpcClient = mockClient

	// Initialize components manually
	monitor.tracker = NewProposalTracker(mockClient, testLogger)
	monitor.scheduler = NewUpgradeScheduler(cfg, testLogger)
	monitor.notifier = NewNotifier(testLogger)

	// Mock successful responses
	mockClient.On("GetCurrentHeight").Return(int64(1000), nil)
	mockClient.On("GetGovernanceProposals", ProposalStatusSubmitted).Return([]*Proposal{}, nil)
	mockClient.On("GetGovernanceProposals", ProposalStatusVoting).Return([]*Proposal{}, nil)
	mockClient.On("GetGovernanceProposals", ProposalStatusPassed).Return([]*Proposal{}, nil)
	mockClient.On("GetGovernanceProposals", ProposalStatusRejected).Return([]*Proposal{}, nil)

	// Test GetProposals
	proposals, err := monitor.GetProposals()
	assert.NoError(t, err)
	assert.NotNil(t, proposals)

	// Test GetUpgradeQueue
	upgrades, err := monitor.GetUpgradeQueue()
	assert.NoError(t, err)
	assert.NotNil(t, upgrades)

	// Test ForceSync
	err = monitor.ForceSync()
	assert.NoError(t, err)

	mockClient.AssertExpectations(t)
}

func TestIntegration_ProposalLifecycle(t *testing.T) {
	// Test complete proposal lifecycle from submission to execution
	testLogger := logger.NewTestLogger()
	mockClient := &MockWBFTClient{}

	tracker := NewProposalTracker(mockClient, testLogger)
	notifier := NewNotifier(testLogger)

	// Create a proposal that goes through all states
	proposal := &Proposal{
		ID:          "1",
		Title:       "Test Upgrade",
		Description: "Test upgrade proposal",
		Type:        ProposalTypeUpgrade,
		Status:      ProposalStatusSubmitted,
		SubmitTime:  time.Now(),
		UpgradeInfo: &UpgradeInfo{
			Name:   "v2.0.0",
			Height: 10000,
			Info:   "Upgrade to v2.0.0",
		},
		UpgradeHeight: 10000,
	}

	// Mock the proposal in different states
	mockClient.On("GetGovernanceProposals", ProposalStatusSubmitted).Return([]*Proposal{proposal}, nil).Once()
	mockClient.On("GetGovernanceProposals", ProposalStatusVoting).Return([]*Proposal{}, nil).Once()
	mockClient.On("GetGovernanceProposals", ProposalStatusPassed).Return([]*Proposal{}, nil).Once()
	mockClient.On("GetGovernanceProposals", ProposalStatusRejected).Return([]*Proposal{}, nil).Once()

	// Fetch proposals
	newProposals, err := tracker.FetchLatest()
	assert.NoError(t, err)
	assert.Len(t, newProposals, 1)

	// Notify new proposal
	notifier.NotifyNewProposal(proposal)
	assert.Len(t, notifier.notifications, 1)

	// Move to voting
	proposal.Status = ProposalStatusVoting
	proposal.VotingEndTime = time.Now().Add(48 * time.Hour)
	notifier.NotifyVotingStarted(proposal)

	// Add voting stats
	proposal.VotingStats = &VotingStats{
		YesVotes:        1000000,
		NoVotes:         100000,
		AbstainVotes:    50000,
		NoWithVetoVotes: 10000,
		Turnout:         0.58,
	}

	// Check quorum
	if proposal.VotingStats.Turnout > 0.5 {
		notifier.NotifyQuorumReached(proposal)
	}

	// Move to passed
	proposal.Status = ProposalStatusPassed
	notifier.NotifyVotingEnded(proposal)
	notifier.NotifyProposalPassed(proposal)

	// Schedule upgrade
	tmpDir, _ := os.MkdirTemp("", "scheduler-test-*")
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{Home: tmpDir}
	scheduler := NewUpgradeScheduler(cfg, testLogger)
	scheduler.SetValidationEnabled(false) // Disable for test

	err = scheduler.ScheduleUpgrade(proposal)
	assert.NoError(t, err)

	// Check upgrade is scheduled
	queue, err := scheduler.GetQueue()
	assert.NoError(t, err)
	assert.Len(t, queue, 1)

	// Trigger upgrade notification
	notifier.NotifyUpgradeScheduled(proposal)

	// Verify all notifications were created
	assert.GreaterOrEqual(t, len(notifier.notifications), 5)

	mockClient.AssertExpectations(t)
}

func TestIntegration_UpgradeExecution(t *testing.T) {
	// Test upgrade execution flow
	tmpDir, err := os.MkdirTemp("", "upgrade-test-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{Home: tmpDir}
	testLogger := logger.NewTestLogger()

	scheduler := NewUpgradeScheduler(cfg, testLogger)
	notifier := NewNotifier(testLogger)

	// Create upgrade info
	upgrade := &UpgradeInfo{
		Name:   "v3.0.0",
		Height: 20000,
		Info:   "Major upgrade",
		Status: UpgradeStatusScheduled,
		Binaries: map[string]*BinaryInfo{
			"linux": {
				URL:      "https://example.com/binary",
				Checksum: "sha256:abc123",
			},
		},
	}

	// Schedule directly (bypass proposal)
	scheduler.upgrades[upgrade.Name] = upgrade
	scheduler.scheduledQueue = []*UpgradeInfo{upgrade}

	// Check if upgrade is ready
	readyUpgrade, ready := scheduler.IsUpgradeReady(20000)
	assert.True(t, ready)
	assert.Equal(t, upgrade, readyUpgrade)

	// Update status to in progress
	upgrade.Status = UpgradeStatusInProgress
	err = scheduler.UpdateStatus(upgrade)
	assert.NoError(t, err)

	// Notify upgrade triggered
	notifier.NotifyUpgradeTriggered(upgrade)

	// Simulate upgrade completion
	upgrade.Status = UpgradeStatusCompleted
	err = scheduler.UpdateStatus(upgrade)
	assert.NoError(t, err)

	// Notify upgrade completed
	notifier.NotifyUpgradeCompleted(upgrade)

	// Verify upgrade moved to completed queue
	assert.Len(t, scheduler.scheduledQueue, 0)
	assert.Len(t, scheduler.completedQueue, 1)
}

func TestIntegration_ErrorHandling(t *testing.T) {
	// Test error handling across components
	testLogger := logger.NewTestLogger()
	mockClient := &MockWBFTClient{}

	// Set up components
	tracker := NewProposalTracker(mockClient, testLogger)
	notifier := NewNotifier(testLogger)

	// Mock handler that fails
	failingHandler := &MockNotificationHandler{
		handlerType: "failing",
		enabled:     true,
	}
	failingHandler.On("Handle", mock.AnythingOfType("*governance.Notification")).Return(fmt.Errorf("handler error"))
	notifier.AddHandler(failingHandler)

	// Create proposal with invalid data
	invalidProposal := &Proposal{
		ID:    "",  // Invalid: empty ID
		Title: "",  // Invalid: empty title
		Type:  ProposalTypeUpgrade,
	}

	// Test tracker handling invalid proposal
	mockClient.On("GetGovernanceProposals", ProposalStatusSubmitted).Return([]*Proposal{invalidProposal}, nil)
	mockClient.On("GetGovernanceProposals", ProposalStatusVoting).Return([]*Proposal{}, nil)
	mockClient.On("GetGovernanceProposals", ProposalStatusPassed).Return([]*Proposal{}, nil)
	mockClient.On("GetGovernanceProposals", ProposalStatusRejected).Return([]*Proposal{}, nil)

	proposals, err := tracker.FetchLatest()
	assert.NoError(t, err) // Should not error even with invalid proposal
	assert.Len(t, proposals, 1)

	// Test notifier handling with failing handler
	notifier.NotifyNewProposal(invalidProposal)
	time.Sleep(50 * time.Millisecond) // Give handler time to fail

	// Notification should still be created despite handler failure
	assert.Len(t, notifier.notifications, 1)

	// Test scheduler with invalid upgrade
	tmpDir, _ := os.MkdirTemp("", "error-test-*")
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{Home: tmpDir}
	scheduler := NewUpgradeScheduler(cfg, testLogger)
	scheduler.SetValidationEnabled(true)

	// Try to schedule invalid proposal
	err = scheduler.ScheduleUpgrade(invalidProposal)
	assert.Error(t, err) // Should error with validation enabled

	mockClient.AssertExpectations(t)
	failingHandler.AssertExpectations(t)
}

func TestIntegration_ConcurrentOperations(t *testing.T) {
	// Test concurrent operations across components
	testLogger := logger.NewTestLogger()
	mockClient := &MockWBFTClient{}

	tracker := NewProposalTracker(mockClient, testLogger)
	notifier := NewNotifier(testLogger)

	// Add notification handler
	handler := &MockNotificationHandler{
		handlerType: "concurrent",
		enabled:     true,
	}
	handler.On("Handle", mock.AnythingOfType("*governance.Notification")).Return(nil)
	notifier.AddHandler(handler)

	// Create multiple proposals
	proposals := make([]*Proposal, 10)
	for i := 0; i < 10; i++ {
		proposals[i] = &Proposal{
			ID:         fmt.Sprintf("%d", i),
			Title:      fmt.Sprintf("Proposal %d", i),
			Type:       ProposalTypeText,
			Status:     ProposalStatusSubmitted,
			SubmitTime: time.Now(),
		}
	}

	// Mock concurrent fetches
	mockClient.On("GetGovernanceProposals", ProposalStatusSubmitted).Return(proposals, nil)
	mockClient.On("GetGovernanceProposals", ProposalStatusVoting).Return([]*Proposal{}, nil)
	mockClient.On("GetGovernanceProposals", ProposalStatusPassed).Return([]*Proposal{}, nil)
	mockClient.On("GetGovernanceProposals", ProposalStatusRejected).Return([]*Proposal{}, nil)

	// Concurrent operations
	done := make(chan bool, 3)

	// Fetch proposals concurrently
	go func() {
		newProposals, err := tracker.FetchLatest()
		assert.NoError(t, err)
		assert.Len(t, newProposals, 10)
		done <- true
	}()

	// Send notifications concurrently
	go func() {
		for _, proposal := range proposals {
			notifier.NotifyNewProposal(proposal)
		}
		done <- true
	}()

	// Get stats concurrently
	go func() {
		stats := tracker.GetProposalStats()
		assert.NotNil(t, stats)
		done <- true
	}()

	// Wait for all operations to complete
	for i := 0; i < 3; i++ {
		<-done
	}

	// Allow handlers to complete
	time.Sleep(100 * time.Millisecond)

	// Verify results
	assert.Len(t, notifier.notifications, 10)
	handler.AssertNumberOfCalls(t, "Handle", 10)

	mockClient.AssertExpectations(t)
}

func TestIntegration_WriteUpgradeInfo(t *testing.T) {
	// Test triggering upgrade which writes upgrade info internally
	tmpDir, err := os.MkdirTemp("", "writetest-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{Home: tmpDir}
	testLogger := logger.NewTestLogger()
	monitor := NewMonitor(cfg, testLogger)

	// Initialize notifier
	monitor.notifier = NewNotifier(testLogger)

	// Create upgrade info
	upgrade := &UpgradeInfo{
		Name:   "v4.0.0",
		Height: 30000,
		Info:   "Test upgrade info",
	}

	// Trigger upgrade which writes info internally
	err = monitor.triggerUpgrade(upgrade)
	assert.NoError(t, err)

	// Verify file was created
	upgradeInfoPath := filepath.Join(tmpDir, "data", "upgrade-info.json")
	assert.FileExists(t, upgradeInfoPath)

	// Read and verify content
	content, err := os.ReadFile(upgradeInfoPath)
	assert.NoError(t, err)
	assert.Contains(t, string(content), "v4.0.0")
	assert.Contains(t, string(content), "30000")
}

func TestIntegration_TriggerUpgrade(t *testing.T) {
	// Test triggering upgrade
	tmpDir, err := os.MkdirTemp("", "trigger-test-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{Home: tmpDir}
	testLogger := logger.NewTestLogger()
	monitor := NewMonitor(cfg, testLogger)

	// Initialize notifier
	monitor.notifier = NewNotifier(testLogger)

	// Create upgrade info
	upgrade := &UpgradeInfo{
		Name:   "v5.0.0",
		Height: 40000,
		Info:   "Critical upgrade",
	}

	// Trigger upgrade
	err = monitor.triggerUpgrade(upgrade)
	assert.NoError(t, err)

	// Verify notification was sent
	assert.Len(t, monitor.notifier.notifications, 1)

	// Verify upgrade info was written
	upgradeInfoPath := filepath.Join(tmpDir, "data", "upgrade-info.json")
	assert.FileExists(t, upgradeInfoPath)
}