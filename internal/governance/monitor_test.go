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