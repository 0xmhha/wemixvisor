package hooks

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/wemix/wemixvisor/internal/config"
	"github.com/wemix/wemixvisor/pkg/logger"
	"github.com/wemix/wemixvisor/pkg/types"
)

func TestNewPreUpgradeHook(t *testing.T) {
	cfg := &config.Config{
		Home: t.TempDir(),
		Name: "wemixd",
	}
	logger, _ := logger.New(false, true, "")

	hook := NewPreUpgradeHook(cfg, logger)
	if hook == nil {
		t.Fatal("expected hook to be non-nil")
	}
	if hook.cfg != cfg {
		t.Error("expected cfg to be set")
	}
	if hook.logger != logger {
		t.Error("expected logger to be set")
	}
}

func TestValidateUpgrade(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Home: tmpDir,
		Name: "wemixd",
	}
	logger, _ := logger.New(false, true, "")
	hook := NewPreUpgradeHook(cfg, logger)

	// Create upgrade directory and binary
	upgradeDir := filepath.Join(tmpDir, "wemixvisor", "upgrades", "v2.0.0", "bin")
	os.MkdirAll(upgradeDir, 0755)

	upgradeBin := filepath.Join(upgradeDir, "wemixd")
	os.WriteFile(upgradeBin, []byte("#!/bin/bash\necho test"), 0755)

	info := &types.UpgradeInfo{
		Name:   "v2.0.0",
		Height: 1000000,
	}

	// Should pass validation
	err := hook.ValidateUpgrade(info)
	if err != nil {
		t.Errorf("unexpected validation error: %v", err)
	}
}

func TestValidateUpgradeNoBinary(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Home: tmpDir,
		Name: "wemixd",
	}
	logger, _ := logger.New(false, true, "")
	hook := NewPreUpgradeHook(cfg, logger)

	info := &types.UpgradeInfo{
		Name:   "v2.0.0",
		Height: 1000000,
	}

	// Should fail validation (binary doesn't exist)
	err := hook.ValidateUpgrade(info)
	if err == nil {
		t.Error("expected validation error for missing binary")
	}
}

func TestValidateUpgradeNotExecutable(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Home: tmpDir,
		Name: "wemixd",
	}
	logger, _ := logger.New(false, true, "")
	hook := NewPreUpgradeHook(cfg, logger)

	// Create upgrade directory and non-executable binary
	upgradeDir := filepath.Join(tmpDir, "wemixvisor", "upgrades", "v2.0.0", "bin")
	os.MkdirAll(upgradeDir, 0755)

	upgradeBin := filepath.Join(upgradeDir, "wemixd")
	os.WriteFile(upgradeBin, []byte("not executable"), 0644)

	info := &types.UpgradeInfo{
		Name:   "v2.0.0",
		Height: 1000000,
	}

	// Should fail validation (binary not executable)
	err := hook.ValidateUpgrade(info)
	if err == nil {
		t.Error("expected validation error for non-executable binary")
	}
}

func TestExecuteCustomScript(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping script test on Windows")
	}

	tmpDir := t.TempDir()

	// Create a custom script
	scriptPath := filepath.Join(tmpDir, "custom-pre-upgrade.sh")
	scriptContent := `#!/bin/bash
echo "Running pre-upgrade for $UPGRADE_NAME at height $UPGRADE_HEIGHT"
echo "Daemon: $DAEMON_NAME in $DAEMON_HOME"
exit 0
`
	os.WriteFile(scriptPath, []byte(scriptContent), 0755)

	cfg := &config.Config{
		Home:                 tmpDir,
		Name:                 "wemixd",
		CustomPreUpgrade:     scriptPath,
		PreUpgradeMaxRetries: 2,
	}
	logger, _ := logger.New(false, true, "")
	hook := NewPreUpgradeHook(cfg, logger)

	info := &types.UpgradeInfo{
		Name:   "v2.0.0",
		Height: 1000000,
		Info: map[string]interface{}{
			"message": "test upgrade",
		},
	}

	// Should execute successfully
	err := hook.Execute(info)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestExecuteCustomScriptFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping script test on Windows")
	}

	tmpDir := t.TempDir()

	// Create a failing custom script
	scriptPath := filepath.Join(tmpDir, "failing-pre-upgrade.sh")
	scriptContent := `#!/bin/bash
echo "This script will fail"
exit 1
`
	os.WriteFile(scriptPath, []byte(scriptContent), 0755)

	cfg := &config.Config{
		Home:                 tmpDir,
		Name:                 "wemixd",
		CustomPreUpgrade:     scriptPath,
		PreUpgradeMaxRetries: 1,
	}
	logger, _ := logger.New(false, true, "")
	hook := NewPreUpgradeHook(cfg, logger)

	info := &types.UpgradeInfo{
		Name:   "v2.0.0",
		Height: 1000000,
	}

	// Should fail after retries
	err := hook.Execute(info)
	if err == nil {
		t.Error("expected error for failing script")
	}
}

func TestExecuteStandardScript(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping script test on Windows")
	}

	tmpDir := t.TempDir()

	// Create standard pre-upgrade script in upgrade directory
	upgradeDir := filepath.Join(tmpDir, "wemixvisor", "upgrades", "v2.0.0")
	os.MkdirAll(upgradeDir, 0755)

	scriptPath := filepath.Join(upgradeDir, "pre-upgrade")
	scriptContent := `#!/bin/bash
echo "Standard pre-upgrade for $UPGRADE_NAME"
exit 0
`
	os.WriteFile(scriptPath, []byte(scriptContent), 0755)

	cfg := &config.Config{
		Home:                 tmpDir,
		Name:                 "wemixd",
		PreUpgradeMaxRetries: 0,
	}
	logger, _ := logger.New(false, true, "")
	hook := NewPreUpgradeHook(cfg, logger)

	info := &types.UpgradeInfo{
		Name:   "v2.0.0",
		Height: 1000000,
	}

	// Should execute successfully
	err := hook.Execute(info)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestExecuteNoScript(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Home: tmpDir,
		Name: "wemixd",
	}
	logger, _ := logger.New(false, true, "")
	hook := NewPreUpgradeHook(cfg, logger)

	info := &types.UpgradeInfo{
		Name:   "v2.0.0",
		Height: 1000000,
	}

	// Should succeed (no script to run)
	err := hook.Execute(info)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestExecuteNilInfo(t *testing.T) {
	cfg := &config.Config{
		Home: t.TempDir(),
		Name: "wemixd",
	}
	logger, _ := logger.New(false, true, "")
	hook := NewPreUpgradeHook(cfg, logger)

	// Should fail with nil info
	err := hook.Execute(nil)
	if err == nil {
		t.Error("expected error for nil info")
	}
}

func TestScriptTimeout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping script test on Windows")
	}

	// Note: This test will take 5 minutes if it doesn't work correctly
	// In real scenario, we'd want to make the timeout configurable for testing
	// For now, we'll skip the actual timeout test
	t.Skip("skipping timeout test (would take 5 minutes)")
}