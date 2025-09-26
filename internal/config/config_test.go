package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	// Save original env
	originalHome := os.Getenv("DAEMON_HOME")
	defer os.Setenv("DAEMON_HOME", originalHome)

	// Test with no env set
	os.Unsetenv("DAEMON_HOME")
	cfg := DefaultConfig()

	if cfg.Name != "wemixd" {
		t.Errorf("expected daemon name to be 'wemixd', got %s", cfg.Name)
	}

	if cfg.PollInterval != 300*time.Millisecond {
		t.Errorf("expected poll interval to be 300ms, got %v", cfg.PollInterval)
	}

	if cfg.RestartAfterUpgrade != true {
		t.Errorf("expected restart after upgrade to be true")
	}

	// Test with env set
	testHome := "/test/home"
	os.Setenv("DAEMON_HOME", testHome)
	cfg = DefaultConfig()

	if cfg.Home != testHome {
		t.Errorf("expected home to be %s, got %s", testHome, cfg.Home)
	}
}

func TestConfigPaths(t *testing.T) {
	cfg := &Config{
		Home: "/test/home",
		Name: "wemixd",
	}

	tests := []struct {
		name     string
		method   func() string
		expected string
	}{
		{
			name:     "WemixvisorDir",
			method:   cfg.WemixvisorDir,
			expected: "/test/home/wemixvisor",
		},
		{
			name:     "CurrentDir",
			method:   cfg.CurrentDir,
			expected: "/test/home/wemixvisor/current",
		},
		{
			name:     "GenesisDir",
			method:   cfg.GenesisDir,
			expected: "/test/home/wemixvisor/genesis",
		},
		{
			name:     "UpgradesDir",
			method:   cfg.UpgradesDir,
			expected: "/test/home/wemixvisor/upgrades",
		},
		{
			name:     "CurrentBin",
			method:   cfg.CurrentBin,
			expected: "/test/home/wemixvisor/current/bin/wemixd",
		},
		{
			name:     "GenesisBin",
			method:   cfg.GenesisBin,
			expected: "/test/home/wemixvisor/genesis/bin/wemixd",
		},
		{
			name:     "UpgradeInfoFilePath",
			method:   cfg.UpgradeInfoFilePath,
			expected: "/test/home/data/upgrade-info.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.method()
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestUpgradeDir(t *testing.T) {
	cfg := &Config{
		Home: "/test/home",
		Name: "wemixd",
	}

	upgradeName := "v2.0.0"
	expected := "/test/home/wemixvisor/upgrades/v2.0.0"
	result := cfg.UpgradeDir(upgradeName)

	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestUpgradeBin(t *testing.T) {
	cfg := &Config{
		Home: "/test/home",
		Name: "wemixd",
	}

	upgradeName := "v2.0.0"
	expected := "/test/home/wemixvisor/upgrades/v2.0.0/bin/wemixd"
	result := cfg.UpgradeBin(upgradeName)

	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name      string
		cfg       *Config
		shouldErr bool
		errMsg    string
	}{
		{
			name: "valid config",
			cfg: &Config{
				Home:         "/test/home",
				Name:         "wemixd",
				PollInterval: 300 * time.Millisecond,
			},
			shouldErr: false,
		},
		{
			name: "missing home",
			cfg: &Config{
				Name:         "wemixd",
				PollInterval: 300 * time.Millisecond,
			},
			shouldErr: true,
			errMsg:    "daemon home directory not set",
		},
		{
			name: "missing name",
			cfg: &Config{
				Home:         "/test/home",
				PollInterval: 300 * time.Millisecond,
			},
			shouldErr: true,
			errMsg:    "daemon name not set",
		},
		{
			name: "poll interval too short",
			cfg: &Config{
				Home:         "/test/home",
				Name:         "wemixd",
				PollInterval: 50 * time.Millisecond,
			},
			shouldErr: true,
			errMsg:    "poll interval too short",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.shouldErr {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("expected error message '%s', got '%s'", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestSymLinkToGenesis(t *testing.T) {
	// Create temp directory for testing
	tmpDir := t.TempDir()

	cfg := &Config{
		Home: tmpDir,
		Name: "wemixd",
	}

	// Create genesis directory
	genesisDir := cfg.GenesisDir()
	err := os.MkdirAll(genesisDir, 0755)
	if err != nil {
		t.Fatalf("failed to create genesis dir: %v", err)
	}

	// Create symbolic link
	err = cfg.SymLinkToGenesis()
	if err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	// Verify symlink exists
	currentDir := cfg.CurrentDir()
	info, err := os.Lstat(currentDir)
	if err != nil {
		t.Fatalf("failed to stat current dir: %v", err)
	}

	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("current dir is not a symlink")
	}

	// Verify symlink target
	target, err := os.Readlink(currentDir)
	if err != nil {
		t.Fatalf("failed to read symlink: %v", err)
	}

	expectedTarget := "genesis"
	if target != expectedTarget {
		t.Errorf("expected symlink target to be '%s', got '%s'", expectedTarget, target)
	}
}

func TestSetCurrentUpgrade(t *testing.T) {
	// Create temp directory for testing
	tmpDir := t.TempDir()

	cfg := &Config{
		Home: tmpDir,
		Name: "wemixd",
	}

	// Create upgrade directory
	upgradeName := "v2.0.0"
	upgradeDir := cfg.UpgradeDir(upgradeName)
	err := os.MkdirAll(upgradeDir, 0755)
	if err != nil {
		t.Fatalf("failed to create upgrade dir: %v", err)
	}

	// Set current upgrade
	err = cfg.SetCurrentUpgrade(upgradeName)
	if err != nil {
		t.Fatalf("failed to set current upgrade: %v", err)
	}

	// Verify symlink exists
	currentDir := cfg.CurrentDir()
	info, err := os.Lstat(currentDir)
	if err != nil {
		t.Fatalf("failed to stat current dir: %v", err)
	}

	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("current dir is not a symlink")
	}

	// Verify symlink target
	target, err := os.Readlink(currentDir)
	if err != nil {
		t.Fatalf("failed to read symlink: %v", err)
	}

	expectedTarget := filepath.Join("upgrades", upgradeName)
	if target != expectedTarget {
		t.Errorf("expected symlink target to be '%s', got '%s'", expectedTarget, target)
	}

	// Test with non-existent upgrade
	err = cfg.SetCurrentUpgrade("non-existent")
	if err == nil {
		t.Error("expected error for non-existent upgrade")
	}
}