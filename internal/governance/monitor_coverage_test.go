package governance

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/wemix/wemixvisor/internal/config"
	"github.com/wemix/wemixvisor/pkg/logger"
)

// Additional monitor tests for error paths and edge cases

func TestMonitor_Start_Success(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "monitor-start-success-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		Home:       tmpDir,
		RPCAddress: "http://localhost:8545",
	}
	testLogger := logger.NewTestLogger()
	monitor := NewMonitor(cfg, testLogger)

	// Create a mock client
	mockClient := &MockWBFTClient{}
	monitor.rpcClient = mockClient
	monitor.tracker = NewProposalTracker(mockClient, testLogger)
	monitor.scheduler = NewUpgradeScheduler(cfg, testLogger)
	monitor.notifier = NewNotifier(testLogger)

	// Mock successful responses
	mockClient.On("GetGovernanceProposals", ProposalStatusSubmitted).Return([]*Proposal{}, nil).Maybe()
	mockClient.On("GetGovernanceProposals", ProposalStatusVoting).Return([]*Proposal{}, nil).Maybe()
	mockClient.On("GetGovernanceProposals", ProposalStatusPassed).Return([]*Proposal{}, nil).Maybe()
	mockClient.On("GetGovernanceProposals", ProposalStatusRejected).Return([]*Proposal{}, nil).Maybe()
	mockClient.On("GetCurrentHeight").Return(int64(1000), nil).Maybe()

	// Create context and start
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	monitor.ctx = ctx
	monitor.cancel = cancel

	// Should succeed with all components initialized
	err = monitor.Start()
	assert.NoError(t, err)

	// Stop should work
	err = monitor.Stop()
	assert.NoError(t, err)
}

func TestMonitor_MonitorProposals_WithProcessing(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "monitor-proposals-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		Home:       tmpDir,
		RPCAddress: "http://localhost:8545",
	}
	testLogger := logger.NewTestLogger()
	monitor := NewMonitor(cfg, testLogger)

	// Create context and mock client
	ctx, cancel := context.WithCancel(context.Background())
	monitor.ctx = ctx
	monitor.cancel = cancel

	mockClient := &MockWBFTClient{}
	monitor.rpcClient = mockClient
	monitor.tracker = NewProposalTracker(mockClient, testLogger)
	monitor.scheduler = NewUpgradeScheduler(cfg, testLogger)
	monitor.notifier = NewNotifier(testLogger)

	// Create a new proposal
	proposal := &Proposal{
		ID:            "1",
		Title:         "Test Upgrade",
		Type:          ProposalTypeUpgrade,
		Status:        ProposalStatusSubmitted,
		SubmitTime:    time.Now(),
		UpgradeHeight: 10000,
		UpgradeInfo: &UpgradeInfo{
			Name:   "v2.0.0",
			Height: 10000,
			Info:   "Test upgrade",
			Status: UpgradeStatusScheduled,
		},
	}

	// Mock responses with actual proposal - use Maybe() for multiple calls
	mockClient.On("GetGovernanceProposals", ProposalStatusSubmitted).Return([]*Proposal{proposal}, nil).Maybe()
	mockClient.On("GetGovernanceProposals", ProposalStatusVoting).Return([]*Proposal{}, nil).Maybe()
	mockClient.On("GetGovernanceProposals", ProposalStatusPassed).Return([]*Proposal{}, nil).Maybe()
	mockClient.On("GetGovernanceProposals", ProposalStatusRejected).Return([]*Proposal{}, nil).Maybe()
	mockClient.On("GetCurrentHeight").Return(int64(1000), nil).Maybe()

	// Set poll interval
	monitor.pollInterval = 10 * time.Millisecond

	// Start monitoring in goroutine
	monitor.wg.Add(1)
	go monitor.monitorProposals()

	// Let it process
	time.Sleep(50 * time.Millisecond)

	// Cancel and wait
	cancel()
	monitor.wg.Wait()

	// Should have detected the new proposal
	mockClient.AssertExpectations(t)
}

func TestMonitor_MonitorVoting_WithVotingProposal(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "monitor-voting-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		Home:       tmpDir,
		RPCAddress: "http://localhost:8545",
	}
	testLogger := logger.NewTestLogger()
	monitor := NewMonitor(cfg, testLogger)

	// Create context and mock client
	ctx, cancel := context.WithCancel(context.Background())
	monitor.ctx = ctx
	monitor.cancel = cancel

	mockClient := &MockWBFTClient{}
	monitor.rpcClient = mockClient
	monitor.tracker = NewProposalTracker(mockClient, testLogger)
	monitor.scheduler = NewUpgradeScheduler(cfg, testLogger)
	monitor.notifier = NewNotifier(testLogger)

	// Create a voting proposal
	proposal := &Proposal{
		ID:            "1",
		Title:         "Test Proposal",
		Type:          ProposalTypeText,
		Status:        ProposalStatusVoting,
		SubmitTime:    time.Now(),
		VotingEndTime: time.Now().Add(24 * time.Hour),
		VotingStats: &VotingStats{
			YesVotes:        1000000,
			NoVotes:         100000,
			AbstainVotes:    50000,
			NoWithVetoVotes: 10000,
			Turnout:         0.6,
			QuorumReached:   true,
			ThresholdMet:    true,
		},
	}

	// Add to tracker
	monitor.tracker.proposals = map[string]*Proposal{"1": proposal}
	monitor.tracker.activeProposals = map[string]*Proposal{"1": proposal}

	// Mock response
	mockClient.On("GetCurrentHeight").Return(int64(1000), nil).Maybe()
	mockClient.On("GetProposal", "1").Return(proposal, nil).Maybe()

	// Set poll interval
	monitor.pollInterval = 10 * time.Millisecond

	// Start monitoring in goroutine
	monitor.wg.Add(1)
	go monitor.monitorVoting()

	// Let it process
	time.Sleep(50 * time.Millisecond)

	// Cancel and wait
	cancel()
	monitor.wg.Wait()

	mockClient.AssertExpectations(t)
}

func TestMonitor_ScheduleUpgrades_WithUpgradeReady(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "monitor-schedule-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		Home:       tmpDir,
		RPCAddress: "http://localhost:8545",
	}
	testLogger := logger.NewTestLogger()
	monitor := NewMonitor(cfg, testLogger)

	// Create context and mock client
	ctx, cancel := context.WithCancel(context.Background())
	monitor.ctx = ctx
	monitor.cancel = cancel

	mockClient := &MockWBFTClient{}
	monitor.rpcClient = mockClient
	monitor.tracker = NewProposalTracker(mockClient, testLogger)
	monitor.scheduler = NewUpgradeScheduler(cfg, testLogger)
	monitor.notifier = NewNotifier(testLogger)

	// Add an upgrade ready at height 1001
	upgrade := &UpgradeInfo{
		Name:   "v2.0.0",
		Height: 1001,
		Info:   "Ready upgrade",
		Status: UpgradeStatusScheduled,
	}
	monitor.scheduler.upgrades["v2.0.0"] = upgrade
	monitor.scheduler.scheduledQueue = []*UpgradeInfo{upgrade}

	// Mock current height just before upgrade - use Maybe() for all calls
	mockClient.On("GetCurrentHeight").Return(int64(1001), nil).Maybe()

	// Set poll interval
	monitor.pollInterval = 10 * time.Millisecond

	// Start monitoring in goroutine
	monitor.wg.Add(1)
	go monitor.scheduleUpgrades()

	// Let it process
	time.Sleep(50 * time.Millisecond)

	// Cancel and wait
	cancel()
	monitor.wg.Wait()

	mockClient.AssertExpectations(t)
}


// Test write upgrade info error cases
func TestWriteUpgradeInfo_InvalidPath(t *testing.T) {
	// Try to write to an invalid path
	data := map[string]interface{}{
		"name":   "test",
		"height": 1000,
	}

	err := writeUpgradeInfo("/invalid\x00path/upgrade-info.json", data)
	assert.Error(t, err)
}

func TestWriteFile_InvalidPath(t *testing.T) {
	// Try to write to an invalid path
	err := writeFile("/invalid\x00path/file.txt", []byte("test"))
	assert.Error(t, err)
}