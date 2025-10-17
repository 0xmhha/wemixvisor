package governance

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/wemix/wemixvisor/pkg/logger"
)

// MockWBFTClient is a mock implementation of WBFTClientInterface for testing
type MockWBFTClient struct {
	mock.Mock
}

func (m *MockWBFTClient) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockWBFTClient) GetCurrentHeight() (int64, error) {
	args := m.Called()
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockWBFTClient) GetBlock(height int64) (*BlockInfo, error) {
	args := m.Called(height)
	return args.Get(0).(*BlockInfo), args.Error(1)
}

func (m *MockWBFTClient) GetGovernanceProposals(status ProposalStatus) ([]*Proposal, error) {
	args := m.Called(status)
	return args.Get(0).([]*Proposal), args.Error(1)
}

func (m *MockWBFTClient) GetProposal(proposalID string) (*Proposal, error) {
	args := m.Called(proposalID)
	return args.Get(0).(*Proposal), args.Error(1)
}

func (m *MockWBFTClient) GetValidators() ([]*ValidatorInfo, error) {
	args := m.Called()
	return args.Get(0).([]*ValidatorInfo), args.Error(1)
}

func (m *MockWBFTClient) GetGovernanceParams() (*GovernanceParams, error) {
	args := m.Called()
	return args.Get(0).(*GovernanceParams), args.Error(1)
}

func (m *MockWBFTClient) SetTimeout(timeout time.Duration) {
	m.Called(timeout)
}

func TestNewProposalTracker(t *testing.T) {
	mockClient := &MockWBFTClient{}
	testLogger := logger.NewTestLogger()

	tracker := NewProposalTracker(mockClient, testLogger)

	assert.NotNil(t, tracker)
	assert.Equal(t, mockClient, tracker.client)
	assert.Equal(t, testLogger, tracker.logger)
	assert.Equal(t, 30*time.Second, tracker.syncInterval)
	assert.Equal(t, 30*24*time.Hour, tracker.maxProposalAge)
	assert.True(t, tracker.enableAutoSync)
	assert.Empty(t, tracker.proposals)
	assert.Empty(t, tracker.activeProposals)
	assert.Empty(t, tracker.completedProposals)
}

func TestProposalTracker_FetchLatest(t *testing.T) {
	mockClient := &MockWBFTClient{}
	testLogger := logger.NewTestLogger()
	tracker := NewProposalTracker(mockClient, testLogger)

	// Create test proposals
	testProposals := []*Proposal{
		{
			ID:         "1",
			Title:      "Test Proposal 1",
			Type:       ProposalTypeText,
			Status:     ProposalStatusVoting,
			SubmitTime: time.Now(),
		},
		{
			ID:         "2",
			Title:      "Test Proposal 2",
			Type:       ProposalTypeUpgrade,
			Status:     ProposalStatusSubmitted,
			SubmitTime: time.Now(),
		},
	}

	// Mock the client calls
	mockClient.On("GetGovernanceProposals", ProposalStatusSubmitted).Return([]*Proposal{testProposals[1]}, nil)
	mockClient.On("GetGovernanceProposals", ProposalStatusVoting).Return([]*Proposal{testProposals[0]}, nil)
	mockClient.On("GetGovernanceProposals", ProposalStatusPassed).Return([]*Proposal{}, nil)
	mockClient.On("GetGovernanceProposals", ProposalStatusRejected).Return([]*Proposal{}, nil)

	newProposals, err := tracker.FetchLatest()

	assert.NoError(t, err)
	assert.Len(t, newProposals, 2)
	assert.Len(t, tracker.proposals, 2)
	assert.Len(t, tracker.activeProposals, 2)
	assert.Empty(t, tracker.completedProposals)

	mockClient.AssertExpectations(t)
}

func TestProposalTracker_GetActive(t *testing.T) {
	mockClient := &MockWBFTClient{}
	testLogger := logger.NewTestLogger()
	tracker := NewProposalTracker(mockClient, testLogger)

	// Add test proposals
	activeProposal := &Proposal{
		ID:         "1",
		Status:     ProposalStatusVoting,
		SubmitTime: time.Now(),
	}
	completedProposal := &Proposal{
		ID:         "2",
		Status:     ProposalStatusPassed,
		SubmitTime: time.Now(),
	}

	tracker.proposals["1"] = activeProposal
	tracker.proposals["2"] = completedProposal
	tracker.activeProposals["1"] = activeProposal
	tracker.completedProposals["2"] = completedProposal

	active, err := tracker.GetActive()

	assert.NoError(t, err)
	assert.Len(t, active, 1)
	assert.Equal(t, "1", active[0].ID)
}

func TestProposalTracker_GetByID(t *testing.T) {
	mockClient := &MockWBFTClient{}
	testLogger := logger.NewTestLogger()
	tracker := NewProposalTracker(mockClient, testLogger)

	testProposal := &Proposal{
		ID:     "1",
		Title:  "Test Proposal",
		Status: ProposalStatusVoting,
	}

	tracker.proposals["1"] = testProposal

	// Test existing proposal
	proposal, err := tracker.GetByID("1")
	assert.NoError(t, err)
	assert.Equal(t, testProposal, proposal)

	// Test non-existing proposal
	proposal, err = tracker.GetByID("999")
	assert.Error(t, err)
	assert.Nil(t, proposal)
	assert.Contains(t, err.Error(), "proposal not found")
}

func TestProposalTracker_GetByType(t *testing.T) {
	mockClient := &MockWBFTClient{}
	testLogger := logger.NewTestLogger()
	tracker := NewProposalTracker(mockClient, testLogger)

	// Add test proposals of different types
	upgradeProposal := &Proposal{
		ID:         "1",
		Type:       ProposalTypeUpgrade,
		SubmitTime: time.Now(),
	}
	textProposal := &Proposal{
		ID:         "2",
		Type:       ProposalTypeText,
		SubmitTime: time.Now().Add(-time.Hour),
	}

	tracker.proposals["1"] = upgradeProposal
	tracker.proposals["2"] = textProposal

	// Test getting upgrade proposals
	upgrades, err := tracker.GetByType(ProposalTypeUpgrade)
	assert.NoError(t, err)
	assert.Len(t, upgrades, 1)
	assert.Equal(t, "1", upgrades[0].ID)

	// Test getting text proposals
	texts, err := tracker.GetByType(ProposalTypeText)
	assert.NoError(t, err)
	assert.Len(t, texts, 1)
	assert.Equal(t, "2", texts[0].ID)
}

func TestProposalTracker_UpdateVotingStatus(t *testing.T) {
	mockClient := &MockWBFTClient{}
	testLogger := logger.NewTestLogger()
	tracker := NewProposalTracker(mockClient, testLogger)

	updatedProposal := &Proposal{
		ID:     "1",
		Status: ProposalStatusPassed,
	}

	mockClient.On("GetProposal", "1").Return(updatedProposal, nil)

	err := tracker.UpdateVotingStatus("1")

	assert.NoError(t, err)
	assert.Contains(t, tracker.proposals, "1")
	assert.Equal(t, ProposalStatusPassed, tracker.proposals["1"].Status)

	mockClient.AssertExpectations(t)
}

func TestProposalTracker_GetUpgradeProposals(t *testing.T) {
	mockClient := &MockWBFTClient{}
	testLogger := logger.NewTestLogger()
	tracker := NewProposalTracker(mockClient, testLogger)

	// Add test proposals
	upgradeProposal := &Proposal{
		ID:         "1",
		Type:       ProposalTypeUpgrade,
		SubmitTime: time.Now(),
	}
	textProposal := &Proposal{
		ID:         "2",
		Type:       ProposalTypeText,
		SubmitTime: time.Now(),
	}

	tracker.proposals["1"] = upgradeProposal
	tracker.proposals["2"] = textProposal

	upgrades, err := tracker.GetUpgradeProposals()

	assert.NoError(t, err)
	assert.Len(t, upgrades, 1)
	assert.Equal(t, ProposalTypeUpgrade, upgrades[0].Type)
}

func TestProposalTracker_CleanupOld(t *testing.T) {
	mockClient := &MockWBFTClient{}
	testLogger := logger.NewTestLogger()
	tracker := NewProposalTracker(mockClient, testLogger)

	// Add old completed proposal
	oldProposal := &Proposal{
		ID:            "old",
		Status:        ProposalStatusPassed,
		VotingEndTime: time.Now().Add(-40 * 24 * time.Hour), // 40 days ago
	}

	// Add recent completed proposal
	recentProposal := &Proposal{
		ID:            "recent",
		Status:        ProposalStatusPassed,
		VotingEndTime: time.Now().Add(-5 * 24 * time.Hour), // 5 days ago
	}

	tracker.proposals["old"] = oldProposal
	tracker.proposals["recent"] = recentProposal
	tracker.completedProposals["old"] = oldProposal
	tracker.completedProposals["recent"] = recentProposal

	err := tracker.CleanupOld()

	assert.NoError(t, err)
	assert.NotContains(t, tracker.proposals, "old")
	assert.Contains(t, tracker.proposals, "recent")
	assert.NotContains(t, tracker.completedProposals, "old")
	assert.Contains(t, tracker.completedProposals, "recent")
}

func TestProposalTracker_GetSyncStatus(t *testing.T) {
	mockClient := &MockWBFTClient{}
	testLogger := logger.NewTestLogger()
	tracker := NewProposalTracker(mockClient, testLogger)

	// Add some test data
	tracker.lastSyncHeight = 1000
	tracker.lastSyncTime = time.Now()
	tracker.activeProposals["1"] = &Proposal{ID: "1"}

	status := tracker.GetSyncStatus()

	assert.NotNil(t, status)
	assert.Equal(t, int64(1000), status.LastSyncHeight)
	assert.Equal(t, 1, status.ActiveProposals)
	assert.False(t, status.IsSyncing)
}

func TestProposalTracker_GetProposalStats(t *testing.T) {
	mockClient := &MockWBFTClient{}
	testLogger := logger.NewTestLogger()
	tracker := NewProposalTracker(mockClient, testLogger)

	// Add test proposals
	tracker.proposals["1"] = &Proposal{ID: "1", Type: ProposalTypeUpgrade, Status: ProposalStatusVoting}
	tracker.proposals["2"] = &Proposal{ID: "2", Type: ProposalTypeText, Status: ProposalStatusPassed}
	tracker.activeProposals["1"] = tracker.proposals["1"]
	tracker.completedProposals["2"] = tracker.proposals["2"]

	stats := tracker.GetProposalStats()

	assert.Equal(t, 2, stats["total_proposals"])
	assert.Equal(t, 1, stats["active_proposals"])
	assert.Equal(t, 1, stats["completed_proposals"])

	byType := stats["by_type"].(map[string]int)
	assert.Equal(t, 1, byType["upgrade"])
	assert.Equal(t, 1, byType["text"])

	byStatus := stats["by_status"].(map[string]int)
	assert.Equal(t, 1, byStatus["voting"])
	assert.Equal(t, 1, byStatus["passed"])
}

func TestProposalTracker_SetSyncInterval(t *testing.T) {
	mockClient := &MockWBFTClient{}
	testLogger := logger.NewTestLogger()
	tracker := NewProposalTracker(mockClient, testLogger)

	newInterval := 60 * time.Second
	tracker.SetSyncInterval(newInterval)

	assert.Equal(t, newInterval, tracker.syncInterval)
}

func TestProposalTracker_SetMaxProposalAge(t *testing.T) {
	mockClient := &MockWBFTClient{}
	testLogger := logger.NewTestLogger()
	tracker := NewProposalTracker(mockClient, testLogger)

	newAge := 60 * 24 * time.Hour
	tracker.SetMaxProposalAge(newAge)

	assert.Equal(t, newAge, tracker.maxProposalAge)
}

func TestProposalTracker_Stop(t *testing.T) {
	mockClient := &MockWBFTClient{}
	testLogger := logger.NewTestLogger()
	tracker := NewProposalTracker(mockClient, testLogger)

	err := tracker.Stop()

	assert.NoError(t, err)
	assert.False(t, tracker.enableAutoSync)
}

func TestProposalTracker_Sync_Success(t *testing.T) {
	mockClient := &MockWBFTClient{}
	testLogger := logger.NewTestLogger()
	tracker := NewProposalTracker(mockClient, testLogger)

	// Mock successful fetch of all proposal statuses
	mockClient.On("GetGovernanceProposals", ProposalStatusSubmitted).Return([]*Proposal{}, nil)
	mockClient.On("GetGovernanceProposals", ProposalStatusVoting).Return([]*Proposal{}, nil)
	mockClient.On("GetGovernanceProposals", ProposalStatusPassed).Return([]*Proposal{}, nil)
	mockClient.On("GetGovernanceProposals", ProposalStatusRejected).Return([]*Proposal{}, nil)
	mockClient.On("GetCurrentHeight").Return(int64(1000), nil)

	err := tracker.Sync()
	assert.NoError(t, err)

	mockClient.AssertExpectations(t)
}

func TestProposalTracker_FetchLatest_Error(t *testing.T) {
	mockClient := &MockWBFTClient{}
	testLogger := logger.NewTestLogger()
	tracker := NewProposalTracker(mockClient, testLogger)

	// Mock error for some statuses but success for others
	mockClient.On("GetGovernanceProposals", ProposalStatusSubmitted).Return([]*Proposal{}, nil)
	mockClient.On("GetGovernanceProposals", ProposalStatusVoting).Return([]*Proposal{}, assert.AnError)
	mockClient.On("GetGovernanceProposals", ProposalStatusPassed).Return([]*Proposal{}, nil)
	mockClient.On("GetGovernanceProposals", ProposalStatusRejected).Return([]*Proposal{}, nil)

	proposals, err := tracker.FetchLatest()

	// Should succeed even if some calls fail
	assert.NoError(t, err)
	assert.Empty(t, proposals)

	mockClient.AssertExpectations(t)
}

func TestProposalTracker_UpdateVotingStatus_Error(t *testing.T) {
	mockClient := &MockWBFTClient{}
	testLogger := logger.NewTestLogger()
	tracker := NewProposalTracker(mockClient, testLogger)

	// Mock error for GetProposal
	mockClient.On("GetProposal", "1").Return((*Proposal)(nil), assert.AnError)

	err := tracker.UpdateVotingStatus("1")
	assert.Error(t, err)

	mockClient.AssertExpectations(t)
}

func TestProposalTracker_UpdateVotingStatus_Success(t *testing.T) {
	mockClient := &MockWBFTClient{}
	testLogger := logger.NewTestLogger()
	tracker := NewProposalTracker(mockClient, testLogger)

	// Create a test proposal
	testProposal := &Proposal{
		ID:     "1",
		Status: ProposalStatusVoting,
	}

	// Add proposal to tracker first
	tracker.proposals["1"] = testProposal

	// Mock successful GetProposal response
	updatedProposal := &Proposal{
		ID:     "1",
		Status: ProposalStatusPassed,
	}
	mockClient.On("GetProposal", "1").Return(updatedProposal, nil)

	err := tracker.UpdateVotingStatus("1")
	assert.NoError(t, err)

	// Check that proposal was updated
	assert.Equal(t, ProposalStatusPassed, tracker.proposals["1"].Status)

	mockClient.AssertExpectations(t)
}

func TestProposalTracker_GetActive_Empty(t *testing.T) {
	mockClient := &MockWBFTClient{}
	testLogger := logger.NewTestLogger()
	tracker := NewProposalTracker(mockClient, testLogger)

	proposals, err := tracker.GetActive()
	assert.NoError(t, err)
	assert.Empty(t, proposals)
}

func TestProposalTracker_GetActive_WithProposals(t *testing.T) {
	mockClient := &MockWBFTClient{}
	testLogger := logger.NewTestLogger()
	tracker := NewProposalTracker(mockClient, testLogger)

	// Add test proposals
	activeProposal := &Proposal{
		ID:     "1",
		Status: ProposalStatusVoting,
	}
	completedProposal := &Proposal{
		ID:     "2",
		Status: ProposalStatusPassed,
	}

	tracker.proposals["1"] = activeProposal
	tracker.proposals["2"] = completedProposal
	tracker.activeProposals["1"] = activeProposal

	proposals, err := tracker.GetActive()
	assert.NoError(t, err)
	assert.Len(t, proposals, 1)
	assert.Equal(t, "1", proposals[0].ID)
}

func TestProposalTracker_GetAll(t *testing.T) {
	mockClient := &MockWBFTClient{}
	testLogger := logger.NewTestLogger()
	tracker := NewProposalTracker(mockClient, testLogger)

	// Add test proposals
	activeProposal := &Proposal{
		ID:     "1",
		Status: ProposalStatusVoting,
	}
	completedProposal := &Proposal{
		ID:     "2",
		Status: ProposalStatusPassed,
	}

	tracker.proposals["1"] = activeProposal
	tracker.proposals["2"] = completedProposal

	proposals, err := tracker.GetAll()
	assert.NoError(t, err)
	assert.Len(t, proposals, 2)
}

func TestProposalTracker_GetByType_Additional(t *testing.T) {
	mockClient := &MockWBFTClient{}
	testLogger := logger.NewTestLogger()
	tracker := NewProposalTracker(mockClient, testLogger)

	// Add test proposals
	upgradeProposal := &Proposal{
		ID:   "1",
		Type: ProposalTypeUpgrade,
	}
	textProposal := &Proposal{
		ID:   "2",
		Type: ProposalTypeText,
	}

	tracker.proposals["1"] = upgradeProposal
	tracker.proposals["2"] = textProposal

	upgradeProposals, err := tracker.GetByType(ProposalTypeUpgrade)
	assert.NoError(t, err)
	assert.Len(t, upgradeProposals, 1)
	assert.Equal(t, "1", upgradeProposals[0].ID)

	textProposals, err := tracker.GetByType(ProposalTypeText)
	assert.NoError(t, err)
	assert.Len(t, textProposals, 1)
	assert.Equal(t, "2", textProposals[0].ID)
}

func TestProposalTracker_GetProposalStats_Additional(t *testing.T) {
	mockClient := &MockWBFTClient{}
	testLogger := logger.NewTestLogger()
	tracker := NewProposalTracker(mockClient, testLogger)

	// Add test proposals
	tracker.proposals["1"] = &Proposal{ID: "1", Status: ProposalStatusVoting, Type: ProposalTypeUpgrade}
	tracker.proposals["2"] = &Proposal{ID: "2", Status: ProposalStatusPassed, Type: ProposalTypeText}
	tracker.activeProposals["1"] = tracker.proposals["1"]
	tracker.completedProposals["2"] = tracker.proposals["2"]

	stats := tracker.GetProposalStats()

	assert.NotNil(t, stats)
	assert.Contains(t, stats, "total_proposals")
}