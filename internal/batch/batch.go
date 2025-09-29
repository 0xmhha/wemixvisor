package batch

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/wemix/wemixvisor/internal/config"
	"github.com/wemix/wemixvisor/pkg/logger"
	"github.com/wemix/wemixvisor/pkg/types"
	"go.uber.org/zap"
)

// UpgradePlan represents a planned batch upgrade
type UpgradePlan struct {
	Version     string              `json:"version"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	CreatedAt   time.Time           `json:"created_at"`
	Upgrades    []types.UpgradeInfo `json:"upgrades"`
}

// BatchManager manages batch upgrades
type BatchManager struct {
	cfg    *config.Config
	logger *logger.Logger
}

// NewBatchManager creates a new batch manager
func NewBatchManager(cfg *config.Config, logger *logger.Logger) *BatchManager {
	return &BatchManager{
		cfg:    cfg,
		logger: logger,
	}
}

// CreatePlan creates a new batch upgrade plan
func (m *BatchManager) CreatePlan(name, description string, upgrades []types.UpgradeInfo) (*UpgradePlan, error) {
	// Sort upgrades by height
	sort.Slice(upgrades, func(i, j int) bool {
		return upgrades[i].Height < upgrades[j].Height
	})

	// Validate upgrades
	for i, upgrade := range upgrades {
		if upgrade.Name == "" {
			return nil, fmt.Errorf("upgrade %d has no name", i)
		}
		if upgrade.Height <= 0 {
			return nil, fmt.Errorf("upgrade %s has invalid height: %d", upgrade.Name, upgrade.Height)
		}

		// Check for height conflicts
		if i > 0 && upgrades[i].Height <= upgrades[i-1].Height {
			return nil, fmt.Errorf("upgrade heights must be strictly increasing")
		}
	}

	plan := &UpgradePlan{
		Version:     "1.0",
		Name:        name,
		Description: description,
		CreatedAt:   time.Now(),
		Upgrades:    upgrades,
	}

	// Save plan to file
	if err := m.SavePlan(plan); err != nil {
		return nil, fmt.Errorf("failed to save plan: %w", err)
	}

	m.logger.Info("batch upgrade plan created",
		zap.String("name", name),
		zap.Int("upgrades", len(upgrades)))

	return plan, nil
}

// SavePlan saves a batch upgrade plan to file
func (m *BatchManager) SavePlan(plan *UpgradePlan) error {
	planDir := filepath.Join(m.cfg.WemixvisorDir(), "plans")
	if err := os.MkdirAll(planDir, 0755); err != nil {
		return fmt.Errorf("failed to create plans directory: %w", err)
	}

	filename := fmt.Sprintf("%s-%s.json",
		plan.Name,
		plan.CreatedAt.Format("20060102-150405"))
	planPath := filepath.Join(planDir, filename)

	data, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal plan: %w", err)
	}

	if err := os.WriteFile(planPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write plan file: %w", err)
	}

	m.logger.Info("plan saved",
		zap.String("path", planPath))

	return nil
}

// LoadPlan loads a batch upgrade plan from file
func (m *BatchManager) LoadPlan(filename string) (*UpgradePlan, error) {
	planPath := filepath.Join(m.cfg.WemixvisorDir(), "plans", filename)

	data, err := os.ReadFile(planPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read plan file: %w", err)
	}

	var plan UpgradePlan
	if err := json.Unmarshal(data, &plan); err != nil {
		return nil, fmt.Errorf("failed to unmarshal plan: %w", err)
	}

	m.logger.Info("plan loaded",
		zap.String("name", plan.Name),
		zap.Int("upgrades", len(plan.Upgrades)))

	return &plan, nil
}

// ListPlans lists all available batch upgrade plans
func (m *BatchManager) ListPlans() ([]string, error) {
	planDir := filepath.Join(m.cfg.WemixvisorDir(), "plans")

	// Check if plans directory exists
	if _, err := os.Stat(planDir); os.IsNotExist(err) {
		return []string{}, nil
	}

	entries, err := os.ReadDir(planDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read plans directory: %w", err)
	}

	var plans []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			plans = append(plans, entry.Name())
		}
	}

	sort.Strings(plans)
	return plans, nil
}

// ValidatePlan validates a batch upgrade plan
func (m *BatchManager) ValidatePlan(plan *UpgradePlan) error {
	if plan == nil {
		return fmt.Errorf("plan is nil")
	}

	if plan.Name == "" {
		return fmt.Errorf("plan has no name")
	}

	if len(plan.Upgrades) == 0 {
		return fmt.Errorf("plan has no upgrades")
	}

	// Check for duplicate names
	names := make(map[string]bool)
	for _, upgrade := range plan.Upgrades {
		if names[upgrade.Name] {
			return fmt.Errorf("duplicate upgrade name: %s", upgrade.Name)
		}
		names[upgrade.Name] = true
	}

	// Check for height ordering
	for i := 1; i < len(plan.Upgrades); i++ {
		if plan.Upgrades[i].Height <= plan.Upgrades[i-1].Height {
			return fmt.Errorf("upgrades not in height order: %s (height %d) comes after %s (height %d)",
				plan.Upgrades[i].Name, plan.Upgrades[i].Height,
				plan.Upgrades[i-1].Name, plan.Upgrades[i-1].Height)
		}
	}

	return nil
}

// GetNextUpgrade returns the next upgrade from the plan based on current height
func (m *BatchManager) GetNextUpgrade(plan *UpgradePlan, currentHeight int64) *types.UpgradeInfo {
	for _, upgrade := range plan.Upgrades {
		if upgrade.Height > currentHeight {
			return &upgrade
		}
	}
	return nil
}

// GetUpgradeByHeight returns the upgrade at a specific height
func (m *BatchManager) GetUpgradeByHeight(plan *UpgradePlan, height int64) *types.UpgradeInfo {
	for _, upgrade := range plan.Upgrades {
		if upgrade.Height == height {
			return &upgrade
		}
	}
	return nil
}

// PrepareBatchUpgrade prepares all binaries for a batch upgrade plan
func (m *BatchManager) PrepareBatchUpgrade(plan *UpgradePlan, downloader interface {
	EnsureUpgradeBinary(string) error
}) error {
	m.logger.Info("preparing batch upgrade",
		zap.String("plan", plan.Name),
		zap.Int("upgrades", len(plan.Upgrades)))

	// Ensure all upgrade binaries exist
	for _, upgrade := range plan.Upgrades {
		m.logger.Info("ensuring upgrade binary",
			zap.String("name", upgrade.Name),
			zap.Int64("height", upgrade.Height))

		if err := downloader.EnsureUpgradeBinary(upgrade.Name); err != nil {
			return fmt.Errorf("failed to ensure binary for %s: %w", upgrade.Name, err)
		}
	}

	m.logger.Info("batch upgrade preparation complete",
		zap.String("plan", plan.Name))

	return nil
}

// ExecutePlan writes upgrade info files for all upgrades in the plan
func (m *BatchManager) ExecutePlan(plan *UpgradePlan) error {
	// Validate plan first
	if err := m.ValidatePlan(plan); err != nil {
		return fmt.Errorf("plan validation failed: %w", err)
	}

	// Create upgrade-info files for each upgrade
	for _, upgrade := range plan.Upgrades {
		if err := m.writeUpgradeInfo(&upgrade); err != nil {
			return fmt.Errorf("failed to write upgrade info for %s: %w", upgrade.Name, err)
		}
	}

	m.logger.Info("batch plan executed",
		zap.String("plan", plan.Name),
		zap.Int("upgrades", len(plan.Upgrades)))

	return nil
}

// writeUpgradeInfo writes an upgrade-info.json file for a specific height
func (m *BatchManager) writeUpgradeInfo(info *types.UpgradeInfo) error {
	// Create height-specific directory
	heightDir := filepath.Join(m.cfg.Home, "data", "upgrades", fmt.Sprintf("%d", info.Height))
	if err := os.MkdirAll(heightDir, 0755); err != nil {
		return fmt.Errorf("failed to create height directory: %w", err)
	}

	// Write upgrade-info.json
	infoPath := filepath.Join(heightDir, "upgrade-info.json")
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal upgrade info: %w", err)
	}

	if err := os.WriteFile(infoPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write upgrade info: %w", err)
	}

	m.logger.Info("upgrade info written",
		zap.String("path", infoPath),
		zap.String("name", info.Name),
		zap.Int64("height", info.Height))

	return nil
}

// RemovePlan removes a batch upgrade plan
func (m *BatchManager) RemovePlan(filename string) error {
	planPath := filepath.Join(m.cfg.WemixvisorDir(), "plans", filename)

	if err := os.Remove(planPath); err != nil {
		return fmt.Errorf("failed to remove plan: %w", err)
	}

	m.logger.Info("plan removed",
		zap.String("filename", filename))

	return nil
}

// GetPlanStatus returns the status of a batch upgrade plan
func (m *BatchManager) GetPlanStatus(plan *UpgradePlan, currentHeight int64) map[string]interface{} {
	completed := 0
	pending := 0
	active := ""

	for _, upgrade := range plan.Upgrades {
		if upgrade.Height <= currentHeight {
			completed++
		} else {
			pending++
			if active == "" {
				active = upgrade.Name
			}
		}
	}

	return map[string]interface{}{
		"name":           plan.Name,
		"total_upgrades": len(plan.Upgrades),
		"completed":      completed,
		"pending":        pending,
		"active":         active,
		"current_height": currentHeight,
		"progress":       float64(completed) / float64(len(plan.Upgrades)) * 100,
	}
}