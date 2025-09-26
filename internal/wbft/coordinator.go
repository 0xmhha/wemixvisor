package wbft

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/wemix/wemixvisor/internal/batch"
	"github.com/wemix/wemixvisor/internal/config"
	"github.com/wemix/wemixvisor/pkg/logger"
	"github.com/wemix/wemixvisor/pkg/types"
	"go.uber.org/zap"
)

// Coordinator coordinates upgrades with WBFT consensus
type Coordinator struct {
	cfg          *config.Config
	logger       *logger.Logger
	client       *Client
	batchManager *batch.BatchManager
	mu           sync.Mutex
	activePlan   *batch.UpgradePlan
}

// NewCoordinator creates a new WBFT coordinator
func NewCoordinator(cfg *config.Config, logger *logger.Logger) *Coordinator {
	return &Coordinator{
		cfg:          cfg,
		logger:       logger,
		client:       NewClient(cfg, logger),
		batchManager: batch.NewBatchManager(cfg, logger),
	}
}

// LoadPlan loads a batch upgrade plan
func (c *Coordinator) LoadPlan(filename string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	plan, err := c.batchManager.LoadPlan(filename)
	if err != nil {
		return fmt.Errorf("failed to load plan: %w", err)
	}

	// Validate plan
	if err := c.batchManager.ValidatePlan(plan); err != nil {
		return fmt.Errorf("plan validation failed: %w", err)
	}

	c.activePlan = plan
	c.logger.Info("upgrade plan loaded",
		zap.String("name", plan.Name),
		zap.Int("upgrades", len(plan.Upgrades)))

	return nil
}

// MonitorAndCoordinate monitors consensus and coordinates upgrades
func (c *Coordinator) MonitorAndCoordinate(ctx context.Context) error {
	// Check if we have an active plan
	c.mu.Lock()
	if c.activePlan == nil {
		c.mu.Unlock()
		return fmt.Errorf("no active upgrade plan")
	}
	plan := c.activePlan
	c.mu.Unlock()

	// Check node readiness
	if err := c.client.CheckReadiness(ctx); err != nil {
		return fmt.Errorf("node not ready: %w", err)
	}

	// Get current height
	currentHeight, err := c.client.GetCurrentHeight(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current height: %w", err)
	}

	c.logger.Info("starting upgrade coordination",
		zap.String("plan", plan.Name),
		zap.Int64("current_height", currentHeight))

	// Monitor consensus state
	stateChan := c.client.MonitorConsensus(ctx, 5*time.Second)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case state := <-stateChan:
			if state == nil {
				continue
			}

			// Check for upcoming upgrades
			nextUpgrade := c.batchManager.GetNextUpgrade(plan, state.Height)
			if nextUpgrade == nil {
				c.logger.Info("all upgrades completed",
					zap.String("plan", plan.Name))
				return nil
			}

			// Check if we're approaching upgrade height
			blocksUntilUpgrade := nextUpgrade.Height - state.Height
			if blocksUntilUpgrade <= 100 && blocksUntilUpgrade > 0 {
				c.logger.Info("approaching upgrade height",
					zap.String("upgrade", nextUpgrade.Name),
					zap.Int64("height", nextUpgrade.Height),
					zap.Int64("blocks_remaining", blocksUntilUpgrade))

				// Prepare for upgrade if within 10 blocks
				if blocksUntilUpgrade <= 10 {
					if err := c.prepareUpgrade(ctx, nextUpgrade); err != nil {
						c.logger.Error("failed to prepare upgrade",
							zap.Error(err))
					}
				}
			}

			// Check if we've reached upgrade height
			if state.Height >= nextUpgrade.Height {
				c.logger.Info("upgrade height reached",
					zap.String("upgrade", nextUpgrade.Name),
					zap.Int64("height", state.Height))

				// For validators, wait for consensus participation
				if c.cfg.ValidatorMode {
					if err := c.waitForConsensusParticipation(ctx, state); err != nil {
						c.logger.Warn("consensus participation check failed",
							zap.Error(err))
					}
				}

				// Trigger upgrade
				if err := c.triggerUpgrade(ctx, nextUpgrade); err != nil {
					return fmt.Errorf("upgrade failed: %w", err)
				}
			}
		}
	}
}

// prepareUpgrade prepares for an upcoming upgrade
func (c *Coordinator) prepareUpgrade(ctx context.Context, upgrade *types.UpgradeInfo) error {
	c.logger.Info("preparing for upgrade",
		zap.String("name", upgrade.Name),
		zap.Int64("height", upgrade.Height))

	// Write upgrade-info.json
	if err := c.writeUpgradeInfo(upgrade); err != nil {
		return fmt.Errorf("failed to write upgrade info: %w", err)
	}

	// For validators, check consensus participation
	if c.cfg.ValidatorMode {
		isValidator, err := c.client.IsValidator(ctx)
		if err != nil {
			c.logger.Warn("failed to check validator status",
				zap.Error(err))
		} else if isValidator {
			c.logger.Info("validator preparing for upgrade",
				zap.String("upgrade", upgrade.Name))
		}
	}

	return nil
}

// triggerUpgrade triggers the actual upgrade
func (c *Coordinator) triggerUpgrade(ctx context.Context, upgrade *types.UpgradeInfo) error {
	c.logger.Info("triggering upgrade",
		zap.String("name", upgrade.Name),
		zap.Int64("height", upgrade.Height))

	// The actual upgrade will be handled by the process manager
	// when it detects the upgrade-info.json file

	// Log upgrade status
	c.mu.Lock()
	if c.activePlan != nil {
		status := c.batchManager.GetPlanStatus(c.activePlan, upgrade.Height)
		c.logger.Info("upgrade plan progress",
			zap.String("plan", c.activePlan.Name),
			zap.Any("status", status))
	}
	c.mu.Unlock()

	return nil
}

// waitForConsensusParticipation waits for the validator to participate in consensus
func (c *Coordinator) waitForConsensusParticipation(ctx context.Context, state *ConsensusState) error {
	if !state.IsValidator {
		return nil
	}

	c.logger.Info("waiting for consensus participation",
		zap.Int64("height", state.Height),
		zap.Int("round", state.Round))

	// Wait for a few rounds to ensure stable consensus
	timeout := 30 * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	roundsParticipated := 0
	requiredRounds := 3

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for consensus participation")
		case <-ticker.C:
			newState, err := c.client.GetConsensusState(ctx)
			if err != nil {
				c.logger.Warn("failed to get consensus state",
					zap.Error(err))
				continue
			}

			if newState.IsValidator && newState.Round > state.Round {
				roundsParticipated++
				if roundsParticipated >= requiredRounds {
					c.logger.Info("validator participated in consensus",
						zap.Int("rounds", roundsParticipated))
					return nil
				}
			}
		}
	}
}

// writeUpgradeInfo writes the upgrade-info.json file
func (c *Coordinator) writeUpgradeInfo(upgrade *types.UpgradeInfo) error {
	// This will be detected by the FileWatcher in the process manager
	return types.WriteUpgradeInfoFile(c.cfg.UpgradeInfoFilePath(), upgrade)
}

// GetPlanStatus returns the status of the active plan
func (c *Coordinator) GetPlanStatus(ctx context.Context) (map[string]interface{}, error) {
	c.mu.Lock()
	plan := c.activePlan
	c.mu.Unlock()

	if plan == nil {
		return nil, fmt.Errorf("no active plan")
	}

	currentHeight, err := c.client.GetCurrentHeight(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current height: %w", err)
	}

	return c.batchManager.GetPlanStatus(plan, currentHeight), nil
}

// ValidateUpgradeSchedule validates that upgrades can be performed at scheduled heights
func (c *Coordinator) ValidateUpgradeSchedule(ctx context.Context, plan *batch.UpgradePlan) error {
	// Get current height
	currentHeight, err := c.client.GetCurrentHeight(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current height: %w", err)
	}

	// Check each upgrade
	for _, upgrade := range plan.Upgrades {
		if upgrade.Height <= currentHeight {
			c.logger.Warn("upgrade height already passed",
				zap.String("name", upgrade.Name),
				zap.Int64("height", upgrade.Height),
				zap.Int64("current", currentHeight))
		}

		// For validators, additional checks
		if c.cfg.ValidatorMode {
			blocksUntilUpgrade := upgrade.Height - currentHeight
			if blocksUntilUpgrade > 0 && blocksUntilUpgrade < 100 {
				c.logger.Warn("upgrade very close, may not have time to prepare",
					zap.String("name", upgrade.Name),
					zap.Int64("blocks", blocksUntilUpgrade))
			}
		}
	}

	return nil
}

// WaitForUpgradeHeight waits for a specific upgrade height
func (c *Coordinator) WaitForUpgradeHeight(ctx context.Context, upgrade *types.UpgradeInfo) error {
	c.logger.Info("waiting for upgrade height",
		zap.String("name", upgrade.Name),
		zap.Int64("height", upgrade.Height))

	// Calculate timeout based on current height and block time
	currentHeight, err := c.client.GetCurrentHeight(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current height: %w", err)
	}

	blocksRemaining := upgrade.Height - currentHeight
	if blocksRemaining <= 0 {
		c.logger.Info("upgrade height already reached")
		return nil
	}

	// Assume ~2 second block time for WBFT
	estimatedTime := time.Duration(blocksRemaining) * 2 * time.Second
	// Add 20% buffer
	timeout := estimatedTime + (estimatedTime / 5)

	return c.client.WaitForHeight(ctx, upgrade.Height, timeout)
}