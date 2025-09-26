package batch

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/wemix/wemixvisor/internal/config"
	"github.com/wemix/wemixvisor/pkg/logger"
	"github.com/wemix/wemixvisor/pkg/types"
)

func TestNewBatchManager(t *testing.T) {
	cfg := &config.Config{
		Home: t.TempDir(),
		Name: "wemixd",
	}
	logger, _ := logger.New(false, true, "")

	manager := NewBatchManager(cfg, logger)
	if manager == nil {
		t.Fatal("expected manager to be non-nil")
	}
	if manager.cfg != cfg {
		t.Error("expected cfg to be set")
	}
	if manager.logger != logger {
		t.Error("expected logger to be set")
	}
}

func TestCreatePlan(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Home: tmpDir,
		Name: "wemixd",
	}
	logger, _ := logger.New(false, true, "")
	manager := NewBatchManager(cfg, logger)

	upgrades := []types.UpgradeInfo{
		{
			Name:   "v2.0.0",
			Height: 1000000,
		},
		{
			Name:   "v3.0.0",
			Height: 2000000,
		},
		{
			Name:   "v2.5.0",
			Height: 1500000,
		},
	}

	plan, err := manager.CreatePlan("test-plan", "Test batch upgrade", upgrades)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if plan.Name != "test-plan" {
		t.Errorf("expected plan name 'test-plan', got '%s'", plan.Name)
	}

	if len(plan.Upgrades) != 3 {
		t.Errorf("expected 3 upgrades, got %d", len(plan.Upgrades))
	}

	// Check upgrades are sorted by height
	if plan.Upgrades[0].Name != "v2.0.0" {
		t.Errorf("expected first upgrade to be v2.0.0, got %s", plan.Upgrades[0].Name)
	}
	if plan.Upgrades[1].Name != "v2.5.0" {
		t.Errorf("expected second upgrade to be v2.5.0, got %s", plan.Upgrades[1].Name)
	}
	if plan.Upgrades[2].Name != "v3.0.0" {
		t.Errorf("expected third upgrade to be v3.0.0, got %s", plan.Upgrades[2].Name)
	}
}

func TestCreatePlanValidation(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Home: tmpDir,
		Name: "wemixd",
	}
	logger, _ := logger.New(false, true, "")
	manager := NewBatchManager(cfg, logger)

	tests := []struct {
		name      string
		upgrades  []types.UpgradeInfo
		expectErr bool
		errMsg    string
	}{
		{
			name: "empty name",
			upgrades: []types.UpgradeInfo{
				{Name: "", Height: 1000000},
			},
			expectErr: true,
			errMsg:    "has no name",
		},
		{
			name: "invalid height",
			upgrades: []types.UpgradeInfo{
				{Name: "v2.0.0", Height: 0},
			},
			expectErr: true,
			errMsg:    "invalid height",
		},
		{
			name: "duplicate height",
			upgrades: []types.UpgradeInfo{
				{Name: "v2.0.0", Height: 1000000},
				{Name: "v2.1.0", Height: 1000000},
			},
			expectErr: true,
			errMsg:    "heights must be strictly increasing",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := manager.CreatePlan("test", "test", tc.upgrades)
			if tc.expectErr {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestSaveAndLoadPlan(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Home: tmpDir,
		Name: "wemixd",
	}
	logger, _ := logger.New(false, true, "")
	manager := NewBatchManager(cfg, logger)

	// Create a plan
	plan := &UpgradePlan{
		Version:     "1.0",
		Name:        "test-plan",
		Description: "Test plan",
		CreatedAt:   time.Now(),
		Upgrades: []types.UpgradeInfo{
			{Name: "v2.0.0", Height: 1000000},
			{Name: "v3.0.0", Height: 2000000},
		},
	}

	// Save plan
	err := manager.SavePlan(plan)
	if err != nil {
		t.Fatalf("failed to save plan: %v", err)
	}

	// List plans
	plans, err := manager.ListPlans()
	if err != nil {
		t.Fatalf("failed to list plans: %v", err)
	}

	if len(plans) != 1 {
		t.Fatalf("expected 1 plan, got %d", len(plans))
	}

	// Load plan
	loadedPlan, err := manager.LoadPlan(plans[0])
	if err != nil {
		t.Fatalf("failed to load plan: %v", err)
	}

	if loadedPlan.Name != plan.Name {
		t.Errorf("plan name mismatch: got %s, want %s", loadedPlan.Name, plan.Name)
	}

	if len(loadedPlan.Upgrades) != len(plan.Upgrades) {
		t.Errorf("upgrade count mismatch: got %d, want %d",
			len(loadedPlan.Upgrades), len(plan.Upgrades))
	}
}

func TestValidatePlan(t *testing.T) {
	cfg := &config.Config{
		Home: t.TempDir(),
		Name: "wemixd",
	}
	logger, _ := logger.New(false, true, "")
	manager := NewBatchManager(cfg, logger)

	tests := []struct {
		name      string
		plan      *UpgradePlan
		expectErr bool
		errMsg    string
	}{
		{
			name:      "nil plan",
			plan:      nil,
			expectErr: true,
			errMsg:    "plan is nil",
		},
		{
			name: "empty name",
			plan: &UpgradePlan{
				Name:     "",
				Upgrades: []types.UpgradeInfo{{Name: "v2.0.0", Height: 1000000}},
			},
			expectErr: true,
			errMsg:    "has no name",
		},
		{
			name: "no upgrades",
			plan: &UpgradePlan{
				Name:     "test",
				Upgrades: []types.UpgradeInfo{},
			},
			expectErr: true,
			errMsg:    "has no upgrades",
		},
		{
			name: "duplicate names",
			plan: &UpgradePlan{
				Name: "test",
				Upgrades: []types.UpgradeInfo{
					{Name: "v2.0.0", Height: 1000000},
					{Name: "v2.0.0", Height: 2000000},
				},
			},
			expectErr: true,
			errMsg:    "duplicate upgrade name",
		},
		{
			name: "wrong height order",
			plan: &UpgradePlan{
				Name: "test",
				Upgrades: []types.UpgradeInfo{
					{Name: "v2.0.0", Height: 2000000},
					{Name: "v3.0.0", Height: 1000000},
				},
			},
			expectErr: true,
			errMsg:    "not in height order",
		},
		{
			name: "valid plan",
			plan: &UpgradePlan{
				Name: "test",
				Upgrades: []types.UpgradeInfo{
					{Name: "v2.0.0", Height: 1000000},
					{Name: "v3.0.0", Height: 2000000},
				},
			},
			expectErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := manager.ValidatePlan(tc.plan)
			if tc.expectErr {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestGetNextUpgrade(t *testing.T) {
	cfg := &config.Config{
		Home: t.TempDir(),
		Name: "wemixd",
	}
	logger, _ := logger.New(false, true, "")
	manager := NewBatchManager(cfg, logger)

	plan := &UpgradePlan{
		Name: "test",
		Upgrades: []types.UpgradeInfo{
			{Name: "v2.0.0", Height: 1000000},
			{Name: "v3.0.0", Height: 2000000},
			{Name: "v4.0.0", Height: 3000000},
		},
	}

	// Test getting next upgrade at different heights
	tests := []struct {
		currentHeight int64
		expectedName  string
		expectNil     bool
	}{
		{500000, "v2.0.0", false},
		{1000000, "v3.0.0", false},
		{1500000, "v3.0.0", false},
		{2500000, "v4.0.0", false},
		{3500000, "", true},
	}

	for _, tc := range tests {
		next := manager.GetNextUpgrade(plan, tc.currentHeight)
		if tc.expectNil {
			if next != nil {
				t.Errorf("expected nil at height %d, got %v", tc.currentHeight, next)
			}
		} else {
			if next == nil {
				t.Errorf("expected upgrade at height %d, got nil", tc.currentHeight)
			} else if next.Name != tc.expectedName {
				t.Errorf("at height %d: expected %s, got %s",
					tc.currentHeight, tc.expectedName, next.Name)
			}
		}
	}
}

func TestGetUpgradeByHeight(t *testing.T) {
	cfg := &config.Config{
		Home: t.TempDir(),
		Name: "wemixd",
	}
	logger, _ := logger.New(false, true, "")
	manager := NewBatchManager(cfg, logger)

	plan := &UpgradePlan{
		Name: "test",
		Upgrades: []types.UpgradeInfo{
			{Name: "v2.0.0", Height: 1000000},
			{Name: "v3.0.0", Height: 2000000},
		},
	}

	// Test getting upgrade at specific heights
	upgrade := manager.GetUpgradeByHeight(plan, 1000000)
	if upgrade == nil || upgrade.Name != "v2.0.0" {
		t.Errorf("expected v2.0.0 at height 1000000, got %v", upgrade)
	}

	upgrade = manager.GetUpgradeByHeight(plan, 2000000)
	if upgrade == nil || upgrade.Name != "v3.0.0" {
		t.Errorf("expected v3.0.0 at height 2000000, got %v", upgrade)
	}

	upgrade = manager.GetUpgradeByHeight(plan, 1500000)
	if upgrade != nil {
		t.Errorf("expected nil at height 1500000, got %v", upgrade)
	}
}

type mockDownloader struct {
	ensureCalled []string
	shouldFail   bool
}

func (m *mockDownloader) EnsureUpgradeBinary(name string) error {
	m.ensureCalled = append(m.ensureCalled, name)
	if m.shouldFail {
		return fmt.Errorf("mock download failure")
	}
	return nil
}

func TestPrepareBatchUpgrade(t *testing.T) {
	cfg := &config.Config{
		Home: t.TempDir(),
		Name: "wemixd",
	}
	logger, _ := logger.New(false, true, "")
	manager := NewBatchManager(cfg, logger)

	plan := &UpgradePlan{
		Name: "test",
		Upgrades: []types.UpgradeInfo{
			{Name: "v2.0.0", Height: 1000000},
			{Name: "v3.0.0", Height: 2000000},
		},
	}

	// Test successful preparation
	mockDL := &mockDownloader{}
	err := manager.PrepareBatchUpgrade(plan, mockDL)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(mockDL.ensureCalled) != 2 {
		t.Errorf("expected 2 calls to EnsureUpgradeBinary, got %d", len(mockDL.ensureCalled))
	}

	// Test failed preparation
	mockDL = &mockDownloader{shouldFail: true}
	err = manager.PrepareBatchUpgrade(plan, mockDL)
	if err == nil {
		t.Error("expected error for failed download")
	}
}

func TestExecutePlan(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Home: tmpDir,
		Name: "wemixd",
	}
	logger, _ := logger.New(false, true, "")
	manager := NewBatchManager(cfg, logger)

	plan := &UpgradePlan{
		Name: "test",
		Upgrades: []types.UpgradeInfo{
			{Name: "v2.0.0", Height: 1000000},
			{Name: "v3.0.0", Height: 2000000},
		},
	}

	err := manager.ExecutePlan(plan)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that upgrade-info files were created
	for _, upgrade := range plan.Upgrades {
		heightDir := filepath.Join(tmpDir, "data", "upgrades", fmt.Sprintf("%d", upgrade.Height))
		infoPath := filepath.Join(heightDir, "upgrade-info.json")

		data, err := os.ReadFile(infoPath)
		if err != nil {
			t.Errorf("failed to read upgrade info for %s: %v", upgrade.Name, err)
			continue
		}

		var info types.UpgradeInfo
		if err := json.Unmarshal(data, &info); err != nil {
			t.Errorf("failed to unmarshal upgrade info: %v", err)
			continue
		}

		if info.Name != upgrade.Name {
			t.Errorf("upgrade name mismatch: got %s, want %s", info.Name, upgrade.Name)
		}
		if info.Height != upgrade.Height {
			t.Errorf("upgrade height mismatch: got %d, want %d", info.Height, upgrade.Height)
		}
	}
}

func TestRemovePlan(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Home: tmpDir,
		Name: "wemixd",
	}
	logger, _ := logger.New(false, true, "")
	manager := NewBatchManager(cfg, logger)

	// Create a plan
	plan := &UpgradePlan{
		Name:      "test-plan",
		CreatedAt: time.Now(),
		Upgrades: []types.UpgradeInfo{
			{Name: "v2.0.0", Height: 1000000},
		},
	}

	// Save plan
	err := manager.SavePlan(plan)
	if err != nil {
		t.Fatalf("failed to save plan: %v", err)
	}

	// List plans to get filename
	plans, err := manager.ListPlans()
	if err != nil {
		t.Fatalf("failed to list plans: %v", err)
	}

	if len(plans) != 1 {
		t.Fatalf("expected 1 plan, got %d", len(plans))
	}

	// Remove plan
	err = manager.RemovePlan(plans[0])
	if err != nil {
		t.Fatalf("failed to remove plan: %v", err)
	}

	// List plans again
	plans, err = manager.ListPlans()
	if err != nil {
		t.Fatalf("failed to list plans: %v", err)
	}

	if len(plans) != 0 {
		t.Errorf("expected 0 plans after removal, got %d", len(plans))
	}
}

func TestGetPlanStatus(t *testing.T) {
	cfg := &config.Config{
		Home: t.TempDir(),
		Name: "wemixd",
	}
	logger, _ := logger.New(false, true, "")
	manager := NewBatchManager(cfg, logger)

	plan := &UpgradePlan{
		Name: "test",
		Upgrades: []types.UpgradeInfo{
			{Name: "v2.0.0", Height: 1000000},
			{Name: "v3.0.0", Height: 2000000},
			{Name: "v4.0.0", Height: 3000000},
		},
	}

	// Test status at different heights
	status := manager.GetPlanStatus(plan, 1500000)
	if status["completed"] != 1 {
		t.Errorf("expected 1 completed, got %v", status["completed"])
	}
	if status["pending"] != 2 {
		t.Errorf("expected 2 pending, got %v", status["pending"])
	}
	if status["active"] != "v3.0.0" {
		t.Errorf("expected active v3.0.0, got %v", status["active"])
	}

	// Test all completed
	status = manager.GetPlanStatus(plan, 3500000)
	if status["completed"] != 3 {
		t.Errorf("expected 3 completed, got %v", status["completed"])
	}
	if status["pending"] != 0 {
		t.Errorf("expected 0 pending, got %v", status["pending"])
	}
	if status["progress"] != 100.0 {
		t.Errorf("expected 100%% progress, got %v", status["progress"])
	}
}