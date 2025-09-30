package governance

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/wemix/wemixvisor/internal/config"
	"github.com/wemix/wemixvisor/pkg/logger"
)

func TestNewUpgradeScheduler(t *testing.T) {
	cfg := &config.Config{Home: "/tmp/test"}
	testLogger := logger.NewTestLogger()

	scheduler := NewUpgradeScheduler(cfg, testLogger)

	assert.NotNil(t, scheduler)
	assert.Equal(t, cfg, scheduler.cfg)
	assert.Equal(t, testLogger, scheduler.logger)
	assert.True(t, scheduler.enabled)
	assert.Equal(t, 10*time.Minute, scheduler.minUpgradeDelay)
	assert.Equal(t, 1, scheduler.maxConcurrentUpgrades)
	assert.True(t, scheduler.validationEnabled)
	assert.Empty(t, scheduler.upgrades)
	assert.Empty(t, scheduler.scheduledQueue)
	assert.Empty(t, scheduler.completedQueue)
}

func TestUpgradeScheduler_ScheduleUpgrade(t *testing.T) {
	cfg := &config.Config{Home: "/tmp/test"}
	testLogger := logger.NewTestLogger()
	scheduler := NewUpgradeScheduler(cfg, testLogger)

	// Disable validation for this test
	scheduler.SetValidationEnabled(false)

	proposal := &Proposal{
		ID:            "1",
		Type:          ProposalTypeUpgrade,
		UpgradeHeight: 1000,
		UpgradeInfo: &UpgradeInfo{
			Name:   "test-upgrade",
			Height: 1000,
			Info:   "Test upgrade",
		},
	}

	err := scheduler.ScheduleUpgrade(proposal)

	assert.NoError(t, err)
	assert.Len(t, scheduler.upgrades, 1)
	assert.Len(t, scheduler.scheduledQueue, 1)
	assert.Contains(t, scheduler.upgrades, "test-upgrade")

	upgrade := scheduler.upgrades["test-upgrade"]
	assert.Equal(t, "test-upgrade", upgrade.Name)
	assert.Equal(t, int64(1000), upgrade.Height)
	assert.Equal(t, UpgradeStatusScheduled, upgrade.Status)
}

func TestUpgradeScheduler_ScheduleUpgrade_NonUpgradeProposal(t *testing.T) {
	cfg := &config.Config{Home: "/tmp/test"}
	testLogger := logger.NewTestLogger()
	scheduler := NewUpgradeScheduler(cfg, testLogger)

	proposal := &Proposal{
		ID:   "1",
		Type: ProposalTypeText, // Not an upgrade proposal
	}

	err := scheduler.ScheduleUpgrade(proposal)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not an upgrade proposal")
}

func TestUpgradeScheduler_ScheduleUpgrade_MissingUpgradeInfo(t *testing.T) {
	cfg := &config.Config{Home: "/tmp/test"}
	testLogger := logger.NewTestLogger()
	scheduler := NewUpgradeScheduler(cfg, testLogger)

	proposal := &Proposal{
		ID:          "1",
		Type:        ProposalTypeUpgrade,
		UpgradeInfo: nil, // Missing upgrade info
	}

	err := scheduler.ScheduleUpgrade(proposal)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing upgrade info")
}

func TestUpgradeScheduler_GetQueue(t *testing.T) {
	cfg := &config.Config{Home: "/tmp/test"}
	testLogger := logger.NewTestLogger()
	scheduler := NewUpgradeScheduler(cfg, testLogger)

	// Add test upgrades
	upgrade1 := &UpgradeInfo{Name: "upgrade1", Height: 1000}
	upgrade2 := &UpgradeInfo{Name: "upgrade2", Height: 2000}

	scheduler.upgrades["upgrade1"] = upgrade1
	scheduler.upgrades["upgrade2"] = upgrade2
	scheduler.scheduledQueue = []*UpgradeInfo{upgrade1, upgrade2}

	queue, err := scheduler.GetQueue()

	assert.NoError(t, err)
	assert.Len(t, queue, 2)
	// Verify content is correct
	assert.Equal(t, len(scheduler.scheduledQueue), len(queue))
}

func TestUpgradeScheduler_GetUpgrade(t *testing.T) {
	cfg := &config.Config{Home: "/tmp/test"}
	testLogger := logger.NewTestLogger()
	scheduler := NewUpgradeScheduler(cfg, testLogger)

	testUpgrade := &UpgradeInfo{Name: "test-upgrade", Height: 1000}
	scheduler.upgrades["test-upgrade"] = testUpgrade

	// Test existing upgrade
	upgrade, err := scheduler.GetUpgrade("test-upgrade")
	assert.NoError(t, err)
	assert.Equal(t, testUpgrade, upgrade)

	// Test non-existing upgrade
	upgrade, err = scheduler.GetUpgrade("non-existing")
	assert.Error(t, err)
	assert.Nil(t, upgrade)
	assert.Contains(t, err.Error(), "upgrade not found")
}

func TestUpgradeScheduler_GetCurrentUpgrade(t *testing.T) {
	cfg := &config.Config{Home: "/tmp/test"}
	testLogger := logger.NewTestLogger()
	scheduler := NewUpgradeScheduler(cfg, testLogger)

	// Test when no current upgrade
	upgrade, err := scheduler.GetCurrentUpgrade()
	assert.Error(t, err)
	assert.Nil(t, upgrade)
	assert.Contains(t, err.Error(), "no upgrade currently in progress")

	// Test when there is a current upgrade
	currentUpgrade := &UpgradeInfo{Name: "current", Status: UpgradeStatusInProgress}
	scheduler.currentUpgrade = currentUpgrade

	upgrade, err = scheduler.GetCurrentUpgrade()
	assert.NoError(t, err)
	assert.Equal(t, currentUpgrade, upgrade)
}

func TestUpgradeScheduler_GetNextUpgrade(t *testing.T) {
	cfg := &config.Config{Home: "/tmp/test"}
	testLogger := logger.NewTestLogger()
	scheduler := NewUpgradeScheduler(cfg, testLogger)

	// Test when no upgrades scheduled
	upgrade, err := scheduler.GetNextUpgrade()
	assert.Error(t, err)
	assert.Nil(t, upgrade)
	assert.Contains(t, err.Error(), "no upgrades scheduled")

	// Test when there are scheduled upgrades
	nextUpgrade := &UpgradeInfo{Name: "next", Height: 1000}
	scheduler.scheduledQueue = []*UpgradeInfo{nextUpgrade}

	upgrade, err = scheduler.GetNextUpgrade()
	assert.NoError(t, err)
	assert.Equal(t, nextUpgrade, upgrade)
}

func TestUpgradeScheduler_UpdateStatus(t *testing.T) {
	cfg := &config.Config{Home: "/tmp/test"}
	testLogger := logger.NewTestLogger()
	scheduler := NewUpgradeScheduler(cfg, testLogger)

	// Add test upgrade
	testUpgrade := &UpgradeInfo{
		Name:   "test-upgrade",
		Height: 1000,
		Status: UpgradeStatusScheduled,
	}
	scheduler.upgrades["test-upgrade"] = testUpgrade
	scheduler.scheduledQueue = []*UpgradeInfo{testUpgrade}

	// Update to in progress
	updateUpgrade := &UpgradeInfo{
		Name:   "test-upgrade",
		Status: UpgradeStatusInProgress,
	}

	err := scheduler.UpdateStatus(updateUpgrade)

	assert.NoError(t, err)
	assert.Equal(t, UpgradeStatusInProgress, scheduler.upgrades["test-upgrade"].Status)
	assert.Equal(t, testUpgrade, scheduler.currentUpgrade)
	assert.NotNil(t, scheduler.upgrades["test-upgrade"].StartedTime)

	// Update to completed
	updateUpgrade.Status = UpgradeStatusCompleted
	err = scheduler.UpdateStatus(updateUpgrade)

	assert.NoError(t, err)
	assert.Equal(t, UpgradeStatusCompleted, scheduler.upgrades["test-upgrade"].Status)
	assert.Nil(t, scheduler.currentUpgrade)
	assert.NotNil(t, scheduler.upgrades["test-upgrade"].CompletedTime)
	assert.Len(t, scheduler.scheduledQueue, 0) // Should be moved to completed
	assert.Len(t, scheduler.completedQueue, 1)
}

func TestUpgradeScheduler_CancelUpgrade(t *testing.T) {
	cfg := &config.Config{Home: "/tmp/test"}
	testLogger := logger.NewTestLogger()
	scheduler := NewUpgradeScheduler(cfg, testLogger)

	// Add test upgrade
	testUpgrade := &UpgradeInfo{
		Name:   "test-upgrade",
		Status: UpgradeStatusScheduled,
	}
	scheduler.upgrades["test-upgrade"] = testUpgrade
	scheduler.scheduledQueue = []*UpgradeInfo{testUpgrade}

	err := scheduler.CancelUpgrade("test-upgrade")

	assert.NoError(t, err)
	assert.Equal(t, UpgradeStatusCancelled, scheduler.upgrades["test-upgrade"].Status)
	assert.NotNil(t, scheduler.upgrades["test-upgrade"].CompletedTime)
	assert.Len(t, scheduler.scheduledQueue, 0)
	assert.Len(t, scheduler.completedQueue, 1)
}

func TestUpgradeScheduler_CancelUpgrade_InProgress(t *testing.T) {
	cfg := &config.Config{Home: "/tmp/test"}
	testLogger := logger.NewTestLogger()
	scheduler := NewUpgradeScheduler(cfg, testLogger)

	testUpgrade := &UpgradeInfo{
		Name:   "test-upgrade",
		Status: UpgradeStatusInProgress,
	}
	scheduler.upgrades["test-upgrade"] = testUpgrade

	err := scheduler.CancelUpgrade("test-upgrade")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot cancel upgrade in progress")
}

func TestUpgradeScheduler_IsUpgradeReady(t *testing.T) {
	cfg := &config.Config{Home: "/tmp/test"}
	testLogger := logger.NewTestLogger()
	scheduler := NewUpgradeScheduler(cfg, testLogger)

	// Test when no upgrades scheduled
	upgrade, ready := scheduler.IsUpgradeReady(1000)
	assert.False(t, ready)
	assert.Nil(t, upgrade)

	// Add test upgrade
	testUpgrade := &UpgradeInfo{
		Name:   "test-upgrade",
		Height: 1000,
		Status: UpgradeStatusScheduled,
	}
	scheduler.scheduledQueue = []*UpgradeInfo{testUpgrade}

	// Test when height not reached
	upgrade, ready = scheduler.IsUpgradeReady(999)
	assert.False(t, ready)
	assert.Nil(t, upgrade)

	// Test when height reached
	upgrade, ready = scheduler.IsUpgradeReady(1000)
	assert.True(t, ready)
	assert.Equal(t, testUpgrade, upgrade)

	// Test when height exceeded
	upgrade, ready = scheduler.IsUpgradeReady(1001)
	assert.True(t, ready)
	assert.Equal(t, testUpgrade, upgrade)
}

func TestUpgradeScheduler_GetUpgradeStats(t *testing.T) {
	cfg := &config.Config{Home: "/tmp/test"}
	testLogger := logger.NewTestLogger()
	scheduler := NewUpgradeScheduler(cfg, testLogger)

	// Add test upgrades
	scheduler.upgrades["upgrade1"] = &UpgradeInfo{Name: "upgrade1", Status: UpgradeStatusScheduled}
	scheduler.upgrades["upgrade2"] = &UpgradeInfo{Name: "upgrade2", Status: UpgradeStatusCompleted}
	scheduler.scheduledQueue = []*UpgradeInfo{scheduler.upgrades["upgrade1"]}
	scheduler.completedQueue = []*UpgradeInfo{scheduler.upgrades["upgrade2"]}

	stats := scheduler.GetUpgradeStats()

	assert.Equal(t, 2, stats["total_upgrades"])
	assert.Equal(t, 1, stats["scheduled_upgrades"])
	assert.Equal(t, 1, stats["completed_upgrades"])
	assert.Equal(t, false, stats["current_upgrade"])

	byStatus := stats["by_status"].(map[string]int)
	assert.Equal(t, 1, byStatus["scheduled"])
	assert.Equal(t, 1, byStatus["completed"])
}

func TestUpgradeScheduler_SetEnabled(t *testing.T) {
	cfg := &config.Config{Home: "/tmp/test"}
	testLogger := logger.NewTestLogger()
	scheduler := NewUpgradeScheduler(cfg, testLogger)

	scheduler.SetEnabled(false)
	assert.False(t, scheduler.enabled)

	scheduler.SetEnabled(true)
	assert.True(t, scheduler.enabled)
}

func TestUpgradeScheduler_SetMinUpgradeDelay(t *testing.T) {
	cfg := &config.Config{Home: "/tmp/test"}
	testLogger := logger.NewTestLogger()
	scheduler := NewUpgradeScheduler(cfg, testLogger)

	newDelay := 30 * time.Minute
	scheduler.SetMinUpgradeDelay(newDelay)
	assert.Equal(t, newDelay, scheduler.minUpgradeDelay)
}

func TestUpgradeScheduler_SetValidationEnabled(t *testing.T) {
	cfg := &config.Config{Home: "/tmp/test"}
	testLogger := logger.NewTestLogger()
	scheduler := NewUpgradeScheduler(cfg, testLogger)

	scheduler.SetValidationEnabled(false)
	assert.False(t, scheduler.validationEnabled)

	scheduler.SetValidationEnabled(true)
	assert.True(t, scheduler.validationEnabled)
}

func TestUpgradeScheduler_ValidateUpgrade(t *testing.T) {
	cfg := &config.Config{Home: "/tmp/test"}
	testLogger := logger.NewTestLogger()
	scheduler := NewUpgradeScheduler(cfg, testLogger)

	tests := []struct {
		name        string
		upgrade     *UpgradeInfo
		expectError bool
		errorMsg    string
	}{
		{
			name:        "empty name",
			upgrade:     &UpgradeInfo{Name: "", Height: 1000},
			expectError: true,
			errorMsg:    "upgrade name cannot be empty",
		},
		{
			name:        "invalid height",
			upgrade:     &UpgradeInfo{Name: "test", Height: 0},
			expectError: true,
			errorMsg:    "invalid upgrade height",
		},
		{
			name:        "duplicate upgrade",
			upgrade:     &UpgradeInfo{Name: "existing", Height: 1000},
			expectError: true,
			errorMsg:    "already scheduled",
		},
		{
			name: "missing binary URL",
			upgrade: &UpgradeInfo{
				Name:   "test",
				Height: 1000,
				Binaries: map[string]*BinaryInfo{
					"linux": {URL: "", Checksum: "abc123"},
				},
			},
			expectError: true,
			errorMsg:    "missing binary URL",
		},
		{
			name: "missing binary checksum",
			upgrade: &UpgradeInfo{
				Name:   "test",
				Height: 1000,
				Binaries: map[string]*BinaryInfo{
					"linux": {URL: "https://example.com/binary", Checksum: ""},
				},
			},
			expectError: true,
			errorMsg:    "missing checksum",
		},
		{
			name: "valid upgrade",
			upgrade: &UpgradeInfo{
				Name:   "valid",
				Height: 1000,
				Binaries: map[string]*BinaryInfo{
					"linux": {
						URL:      "https://example.com/binary",
						Checksum: "abc123",
					},
				},
			},
			expectError: false,
		},
	}

	// Add existing upgrade for duplicate test
	scheduler.upgrades["existing"] = &UpgradeInfo{Name: "existing"}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := scheduler.validateUpgrade(tt.upgrade)
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

func TestUpgradeScheduler_CleanupOld(t *testing.T) {
	cfg := &config.Config{Home: "/tmp/test"}
	testLogger := logger.NewTestLogger()
	scheduler := NewUpgradeScheduler(cfg, testLogger)

	// Add old completed upgrade
	oldTime := time.Now().Add(-40 * 24 * time.Hour)
	oldUpgrade := &UpgradeInfo{
		Name:          "old-upgrade",
		Status:        UpgradeStatusCompleted,
		CompletedTime: &oldTime,
	}

	// Add recent completed upgrade
	recentTime := time.Now().Add(-5 * 24 * time.Hour)
	recentUpgrade := &UpgradeInfo{
		Name:          "recent-upgrade",
		Status:        UpgradeStatusCompleted,
		CompletedTime: &recentTime,
	}

	scheduler.upgrades["old-upgrade"] = oldUpgrade
	scheduler.upgrades["recent-upgrade"] = recentUpgrade
	scheduler.completedQueue = []*UpgradeInfo{oldUpgrade, recentUpgrade}

	maxAge := 30 * 24 * time.Hour
	err := scheduler.CleanupOld(maxAge)

	assert.NoError(t, err)
	assert.NotContains(t, scheduler.upgrades, "old-upgrade")
	assert.Contains(t, scheduler.upgrades, "recent-upgrade")
	assert.Len(t, scheduler.completedQueue, 1)
}