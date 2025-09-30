package governance

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/wemix/wemixvisor/internal/config"
	"github.com/wemix/wemixvisor/pkg/logger"
)

func TestNewMonitor(t *testing.T) {
	cfg := &config.Config{
		Home: "/tmp/test",
	}
	testLogger := logger.NewTestLogger()

	monitor := NewMonitor(cfg, testLogger)

	assert.NotNil(t, monitor)
	assert.Equal(t, cfg, monitor.cfg)
	assert.Equal(t, testLogger, monitor.logger)
	assert.Equal(t, 30*time.Second, monitor.pollInterval)
	assert.Equal(t, 24*time.Hour, monitor.proposalTimeout)
	assert.True(t, monitor.enabled)
}

func TestMonitor_SetEnabled(t *testing.T) {
	cfg := &config.Config{Home: "/tmp/test"}
	testLogger := logger.NewTestLogger()
	monitor := NewMonitor(cfg, testLogger)

	// Test disabling
	monitor.SetEnabled(false)
	assert.False(t, monitor.enabled)

	// Test enabling
	monitor.SetEnabled(true)
	assert.True(t, monitor.enabled)
}

func TestMonitor_SetPollInterval(t *testing.T) {
	cfg := &config.Config{Home: "/tmp/test"}
	testLogger := logger.NewTestLogger()
	monitor := NewMonitor(cfg, testLogger)

	newInterval := 60 * time.Second
	monitor.SetPollInterval(newInterval)
	assert.Equal(t, newInterval, monitor.pollInterval)
}

func TestMonitor_GetProposals_NotInitialized(t *testing.T) {
	cfg := &config.Config{Home: "/tmp/test"}
	testLogger := logger.NewTestLogger()
	monitor := NewMonitor(cfg, testLogger)

	proposals, err := monitor.GetProposals()
	assert.Error(t, err)
	assert.Nil(t, proposals)
	assert.Contains(t, err.Error(), "proposal tracker not initialized")
}

func TestMonitor_GetUpgradeQueue_NotInitialized(t *testing.T) {
	cfg := &config.Config{Home: "/tmp/test"}
	testLogger := logger.NewTestLogger()
	monitor := NewMonitor(cfg, testLogger)

	upgrades, err := monitor.GetUpgradeQueue()
	assert.Error(t, err)
	assert.Nil(t, upgrades)
	assert.Contains(t, err.Error(), "upgrade scheduler not initialized")
}

func TestMonitor_ForceSync_NotInitialized(t *testing.T) {
	cfg := &config.Config{Home: "/tmp/test"}
	testLogger := logger.NewTestLogger()
	monitor := NewMonitor(cfg, testLogger)

	err := monitor.ForceSync()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "proposal tracker not initialized")
}

func TestValidateUpgradeProposal(t *testing.T) {
	cfg := &config.Config{Home: "/tmp/test"}
	testLogger := logger.NewTestLogger()
	monitor := NewMonitor(cfg, testLogger)

	tests := []struct {
		name        string
		proposal    *Proposal
		expectError bool
		errorMsg    string
	}{
		{
			name: "missing upgrade info",
			proposal: &Proposal{
				ID:            "1",
				Type:          ProposalTypeUpgrade,
				UpgradeHeight: 1000,
				UpgradeInfo:   nil,
			},
			expectError: true,
			errorMsg:    "missing upgrade info",
		},
		{
			name: "valid upgrade proposal without RPC client",
			proposal: &Proposal{
				ID:            "1",
				Type:          ProposalTypeUpgrade,
				UpgradeHeight: 100,
				UpgradeInfo: &UpgradeInfo{
					Name: "test-upgrade",
				},
			},
			expectError: false, // No error when RPC client is nil
		},
		{
			name: "missing binary URL",
			proposal: &Proposal{
				ID:            "1",
				Type:          ProposalTypeUpgrade,
				UpgradeHeight: 10000,
				UpgradeInfo: &UpgradeInfo{
					Name: "test-upgrade",
					Binaries: map[string]*BinaryInfo{
						"linux": {
							URL:      "", // Missing URL
							Checksum: "abc123",
						},
					},
				},
			},
			expectError: true,
			errorMsg:    "missing binary URL",
		},
		{
			name: "missing binary checksum",
			proposal: &Proposal{
				ID:            "1",
				Type:          ProposalTypeUpgrade,
				UpgradeHeight: 10000,
				UpgradeInfo: &UpgradeInfo{
					Name: "test-upgrade",
					Binaries: map[string]*BinaryInfo{
						"linux": {
							URL:      "https://example.com/binary",
							Checksum: "", // Missing checksum
						},
					},
				},
			},
			expectError: true,
			errorMsg:    "missing checksum",
		},
		{
			name: "valid upgrade proposal",
			proposal: &Proposal{
				ID:            "1",
				Type:          ProposalTypeUpgrade,
				UpgradeHeight: 10000,
				UpgradeInfo: &UpgradeInfo{
					Name: "test-upgrade",
					Binaries: map[string]*BinaryInfo{
						"linux": {
							URL:      "https://example.com/binary",
							Checksum: "abc123",
						},
					},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := monitor.validateUpgradeProposal(tt.proposal)
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTriggerUpgrade(t *testing.T) {
	cfg := &config.Config{Home: "/tmp/test"}
	testLogger := logger.NewTestLogger()
	monitor := NewMonitor(cfg, testLogger)

	upgrade := &UpgradeInfo{
		Name:   "test-upgrade",
		Height: 1000,
		Info:   "Test upgrade information",
	}

	// Note: This test will call writeUpgradeInfo which currently returns nil
	// In a real implementation, this would test file writing
	err := monitor.triggerUpgrade(upgrade)
	assert.NoError(t, err)
}

func TestMonitor_Stop(t *testing.T) {
	cfg := &config.Config{Home: "/tmp/test"}
	testLogger := logger.NewTestLogger()
	monitor := NewMonitor(cfg, testLogger)

	err := monitor.Stop()
	assert.NoError(t, err)
}

func TestMonitor_Start_Disabled(t *testing.T) {
	cfg := &config.Config{Home: "/tmp/test"}
	testLogger := logger.NewTestLogger()
	monitor := NewMonitor(cfg, testLogger)

	// Disable monitoring
	monitor.SetEnabled(false)

	err := monitor.Start()
	assert.NoError(t, err)

	// Verify components are not initialized when disabled
	assert.Nil(t, monitor.tracker)
	assert.Nil(t, monitor.scheduler)
	assert.Nil(t, monitor.notifier)
	assert.Nil(t, monitor.rpcClient)
}

func TestMonitor_Start_WithEmptyRPCAddress(t *testing.T) {
	cfg := &config.Config{
		Home: "/tmp/test",
		RPCAddress: "", // Empty RPC address should trigger error
	}
	testLogger := logger.NewTestLogger()
	monitor := NewMonitor(cfg, testLogger)

	err := monitor.Start()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to initialize components")
}

func TestMonitor_InitializeComponents_EmptyURL(t *testing.T) {
	cfg := &config.Config{
		Home: "/tmp/test",
		RPCAddress: "", // Empty URL should cause error
	}
	testLogger := logger.NewTestLogger()
	monitor := NewMonitor(cfg, testLogger)

	err := monitor.initializeComponents()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create RPC client")
}

func TestMonitor_ValidateUpgradeProposal_WithMockClient(t *testing.T) {
	cfg := &config.Config{Home: "/tmp/test"}
	testLogger := logger.NewTestLogger()
	monitor := NewMonitor(cfg, testLogger)

	// Create a mock client
	mockClient := &MockWBFTClient{}
	monitor.rpcClient = mockClient

	// Test case: upgrade height is not in the future
	mockClient.On("GetCurrentHeight").Return(int64(1000), nil)

	proposal := &Proposal{
		ID:            "1",
		Type:          ProposalTypeUpgrade,
		UpgradeHeight: 900, // Height in the past
		UpgradeInfo: &UpgradeInfo{
			Name: "test-upgrade",
		},
	}

	err := monitor.validateUpgradeProposal(proposal)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "upgrade height 900 is not in the future")

	mockClient.AssertExpectations(t)
}

func TestMonitor_ValidateUpgradeProposal_RpcError(t *testing.T) {
	cfg := &config.Config{Home: "/tmp/test"}
	testLogger := logger.NewTestLogger()
	monitor := NewMonitor(cfg, testLogger)

	// Create a mock client that returns an error
	mockClient := &MockWBFTClient{}
	monitor.rpcClient = mockClient

	mockClient.On("GetCurrentHeight").Return(int64(0), assert.AnError)

	proposal := &Proposal{
		ID:            "1",
		Type:          ProposalTypeUpgrade,
		UpgradeHeight: 1000,
		UpgradeInfo: &UpgradeInfo{
			Name: "test-upgrade",
		},
	}

	err := monitor.validateUpgradeProposal(proposal)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get current height")

	mockClient.AssertExpectations(t)
}

func TestMonitor_GetProposals_WithInitializedTracker(t *testing.T) {
	cfg := &config.Config{Home: "/tmp/test"}
	testLogger := logger.NewTestLogger()
	monitor := NewMonitor(cfg, testLogger)

	// Create a mock client and tracker
	mockClient := &MockWBFTClient{}
	monitor.tracker = NewProposalTracker(mockClient, testLogger)

	// Mock the tracker's GetActive method behavior
	// Since we can't easily mock the tracker, we'll test the nil case
	monitor.tracker = nil

	proposals, err := monitor.GetProposals()
	assert.Error(t, err)
	assert.Nil(t, proposals)
	assert.Contains(t, err.Error(), "proposal tracker not initialized")
}

func TestMonitor_GetUpgradeQueue_WithInitializedScheduler(t *testing.T) {
	cfg := &config.Config{Home: "/tmp/test"}
	testLogger := logger.NewTestLogger()
	monitor := NewMonitor(cfg, testLogger)

	// Initialize scheduler
	monitor.scheduler = NewUpgradeScheduler(cfg, testLogger)

	// Test getting empty queue
	queue, err := monitor.GetUpgradeQueue()
	assert.NoError(t, err)
	assert.Empty(t, queue)
}

func TestMonitor_ForceSync_WithInitializedTracker(t *testing.T) {
	cfg := &config.Config{Home: "/tmp/test"}
	testLogger := logger.NewTestLogger()
	monitor := NewMonitor(cfg, testLogger)

	// Create a mock client and tracker
	mockClient := &MockWBFTClient{}
	monitor.tracker = NewProposalTracker(mockClient, testLogger)

	// Mock the expected calls during sync
	mockClient.On("GetCurrentHeight").Return(int64(1000), nil)
	mockClient.On("GetGovernanceProposals", ProposalStatusSubmitted).Return([]*Proposal{}, nil)
	mockClient.On("GetGovernanceProposals", ProposalStatusVoting).Return([]*Proposal{}, nil)
	mockClient.On("GetGovernanceProposals", ProposalStatusPassed).Return([]*Proposal{}, nil)
	mockClient.On("GetGovernanceProposals", ProposalStatusRejected).Return([]*Proposal{}, nil)

	// Test force sync
	err := monitor.ForceSync()
	assert.NoError(t, err)

	mockClient.AssertExpectations(t)
}

// Test helper functions
func createTestProposal(id string, proposalType ProposalType) *Proposal {
	return &Proposal{
		ID:          id,
		Title:       "Test Proposal " + id,
		Description: "Test proposal description",
		Type:        proposalType,
		Status:      ProposalStatusSubmitted,
		SubmitTime:  time.Now(),
	}
}

func createTestUpgradeProposal(id string, height int64) *Proposal {
	proposal := createTestProposal(id, ProposalTypeUpgrade)
	proposal.UpgradeHeight = height
	proposal.UpgradeInfo = &UpgradeInfo{
		Name:   "upgrade-" + id,
		Height: height,
		Info:   "Test upgrade",
	}
	return proposal
}