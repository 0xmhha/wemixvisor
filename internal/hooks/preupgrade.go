package hooks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/wemix/wemixvisor/internal/config"
	"github.com/wemix/wemixvisor/pkg/logger"
	"github.com/wemix/wemixvisor/pkg/types"
	"go.uber.org/zap"
)

// PreUpgradeHook manages pre-upgrade hook execution
type PreUpgradeHook struct {
	cfg    *config.Config
	logger *logger.Logger
}

// NewPreUpgradeHook creates a new pre-upgrade hook manager
func NewPreUpgradeHook(cfg *config.Config, logger *logger.Logger) *PreUpgradeHook {
	return &PreUpgradeHook{
		cfg:    cfg,
		logger: logger,
	}
}

// Execute runs the pre-upgrade hook for the given upgrade
func (h *PreUpgradeHook) Execute(info *types.UpgradeInfo) error {
	if info == nil {
		return fmt.Errorf("upgrade info is nil")
	}

	// Check for custom pre-upgrade script
	if h.cfg.CustomPreUpgrade != "" {
		return h.executeCustomScript(info)
	}

	// Check for standard pre-upgrade script in upgrade directory
	return h.executeStandardScript(info)
}

// executeCustomScript runs the custom pre-upgrade script
func (h *PreUpgradeHook) executeCustomScript(info *types.UpgradeInfo) error {
	scriptPath := h.cfg.CustomPreUpgrade

	// Check if script exists
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		h.logger.Warn("custom pre-upgrade script not found", zap.String("path", scriptPath))
		return nil
	}

	h.logger.Info("executing custom pre-upgrade script", zap.String("path", scriptPath))
	return h.runScript(scriptPath, info, h.cfg.PreUpgradeMaxRetries)
}

// executeStandardScript runs the standard pre-upgrade script
func (h *PreUpgradeHook) executeStandardScript(info *types.UpgradeInfo) error {
	// Look for pre-upgrade script in the upgrade directory
	upgradeDir := h.cfg.UpgradeDir(info.Name)
	scriptPath := filepath.Join(upgradeDir, "pre-upgrade")

	// Check if script exists
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		h.logger.Debug("no pre-upgrade script found", zap.String("path", scriptPath))
		return nil
	}

	h.logger.Info("executing pre-upgrade script", zap.String("path", scriptPath))
	return h.runScript(scriptPath, info, h.cfg.PreUpgradeMaxRetries)
}

// runScript executes a script with retries
func (h *PreUpgradeHook) runScript(scriptPath string, info *types.UpgradeInfo, maxRetries int) error {
	// Make script executable
	if err := os.Chmod(scriptPath, 0755); err != nil {
		return fmt.Errorf("failed to make script executable: %w", err)
	}

	// Set environment variables
	env := os.Environ()
	env = append(env, fmt.Sprintf("DAEMON_HOME=%s", h.cfg.Home))
	env = append(env, fmt.Sprintf("DAEMON_NAME=%s", h.cfg.Name))
	env = append(env, fmt.Sprintf("UPGRADE_NAME=%s", info.Name))
	env = append(env, fmt.Sprintf("UPGRADE_HEIGHT=%d", info.Height))
	if info.Info != nil && len(info.Info) > 0 {
		// Convert map to JSON string for environment variable
		if infoBytes, err := json.Marshal(info.Info); err == nil {
			env = append(env, fmt.Sprintf("UPGRADE_INFO=%s", string(infoBytes)))
		}
	}

	retries := 0
	for {
		if err := h.executeScriptOnce(scriptPath, env); err != nil {
			if retries < maxRetries {
				retries++
				h.logger.Warn("pre-upgrade script failed, retrying",
					zap.Error(err),
					zap.Int("retry", retries),
					zap.Int("max_retries", maxRetries))
				time.Sleep(time.Second * time.Duration(retries))
				continue
			}
			return fmt.Errorf("pre-upgrade script failed after %d retries: %w", maxRetries, err)
		}
		return nil
	}
}

// executeScriptOnce runs the script once with a timeout
func (h *PreUpgradeHook) executeScriptOnce(scriptPath string, env []string) error {
	// Create context with timeout (5 minutes default)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Create command
	cmd := exec.CommandContext(ctx, scriptPath)
	cmd.Env = env

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run command
	err := cmd.Run()

	// Log output
	if stdout.Len() > 0 {
		h.logger.Info("pre-upgrade script output", zap.String("stdout", strings.TrimSpace(stdout.String())))
	}
	if stderr.Len() > 0 {
		h.logger.Warn("pre-upgrade script stderr", zap.String("stderr", strings.TrimSpace(stderr.String())))
	}

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("script execution timeout")
		}
		return fmt.Errorf("script execution failed: %w", err)
	}

	return nil
}

// ValidateUpgrade performs pre-upgrade validation checks
func (h *PreUpgradeHook) ValidateUpgrade(info *types.UpgradeInfo) error {
	// Check if upgrade binary exists
	upgradeBin := h.cfg.UpgradeBin(info.Name)
	if _, err := os.Stat(upgradeBin); os.IsNotExist(err) {
		return fmt.Errorf("upgrade binary not found: %s", upgradeBin)
	}

	// Check if binary is executable
	fileInfo, err := os.Stat(upgradeBin)
	if err != nil {
		return fmt.Errorf("failed to stat upgrade binary: %w", err)
	}

	if fileInfo.Mode()&0111 == 0 {
		return fmt.Errorf("upgrade binary is not executable: %s", upgradeBin)
	}

	// Additional validation can be added here
	// - Check binary version
	// - Verify checksum
	// - Check dependencies

	h.logger.Info("upgrade validation passed", zap.String("name", info.Name))
	return nil
}