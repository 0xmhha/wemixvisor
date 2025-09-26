package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Config holds all configuration for Wemixvisor
type Config struct {
	// Core settings
	Home       string `mapstructure:"daemon_home"`
	Name       string `mapstructure:"daemon_name"`
	Args       []string

	// Upgrade settings
	AllowDownloadBinaries    bool          `mapstructure:"daemon_allow_download_binaries"`
	DownloadMustHaveChecksum bool          `mapstructure:"daemon_download_must_have_checksum"`
	RestartAfterUpgrade      bool          `mapstructure:"daemon_restart_after_upgrade"`
	RestartDelay             time.Duration `mapstructure:"daemon_restart_delay"`

	// Process management
	ShutdownGrace time.Duration `mapstructure:"daemon_shutdown_grace"`
	PollInterval  time.Duration `mapstructure:"daemon_poll_interval"`

	// Backup settings
	UnsafeSkipBackup bool   `mapstructure:"unsafe_skip_backup"`
	DataBackupPath   string `mapstructure:"daemon_data_backup_dir"`

	// Pre-upgrade settings
	PreUpgradeMaxRetries int    `mapstructure:"daemon_preupgrade_max_retries"`
	CustomPreUpgrade     string `mapstructure:"cosmovisor_custom_preupgrade"`

	// WBFT specific settings
	RPCAddress      string `mapstructure:"daemon_rpc_address"`
	ValidatorMode   bool   `mapstructure:"validator_mode"`
	DisableRecase   bool   `mapstructure:"cosmovisor_disable_recase"`

	// Download settings
	DownloadURLs       map[string]string `mapstructure:"download_urls"`
	UnsafeSkipChecksum bool              `mapstructure:"unsafe_skip_checksum"`

	// Logging
	DisableLogs     bool   `mapstructure:"cosmovisor_disable_logs"`
	ColorLogs       bool   `mapstructure:"cosmovisor_color_logs"`
	TimeFormatLogs  string `mapstructure:"cosmovisor_timeformat_logs"`
}

// DefaultConfig returns a Config with default values
func DefaultConfig() *Config {
	home := os.Getenv("DAEMON_HOME")
	if home == "" {
		home = filepath.Join(os.Getenv("HOME"), ".wemixd")
	}

	return &Config{
		Home:                     home,
		Name:                     "wemixd",
		AllowDownloadBinaries:    false,
		DownloadMustHaveChecksum: false,
		RestartAfterUpgrade:      true,
		RestartDelay:             0,
		ShutdownGrace:            0,
		PollInterval:             300 * time.Millisecond,
		UnsafeSkipBackup:         false,
		DataBackupPath:           filepath.Join(home, "backups"),
		PreUpgradeMaxRetries:     0,
		CustomPreUpgrade:         "",
		RPCAddress:               "localhost:8545",
		ValidatorMode:            false,
		DisableRecase:            false,
		DisableLogs:              false,
		ColorLogs:                true,
		TimeFormatLogs:           "kitchen",
	}
}

// WemixvisorDir returns the wemixvisor directory path
func (c *Config) WemixvisorDir() string {
	return filepath.Join(c.Home, "wemixvisor")
}

// CurrentDir returns the current binary directory path
func (c *Config) CurrentDir() string {
	return filepath.Join(c.WemixvisorDir(), "current")
}

// GenesisDir returns the genesis binary directory path
func (c *Config) GenesisDir() string {
	return filepath.Join(c.WemixvisorDir(), "genesis")
}

// UpgradesDir returns the upgrades directory path
func (c *Config) UpgradesDir() string {
	return filepath.Join(c.WemixvisorDir(), "upgrades")
}

// UpgradeDir returns the directory for a specific upgrade
func (c *Config) UpgradeDir(name string) string {
	return filepath.Join(c.UpgradesDir(), name)
}

// CurrentBin returns the current binary path
func (c *Config) CurrentBin() string {
	return filepath.Join(c.CurrentDir(), "bin", c.Name)
}

// GenesisBin returns the genesis binary path
func (c *Config) GenesisBin() string {
	return filepath.Join(c.GenesisDir(), "bin", c.Name)
}

// UpgradeBin returns the binary path for a specific upgrade
func (c *Config) UpgradeBin(name string) string {
	return filepath.Join(c.UpgradeDir(name), "bin", c.Name)
}

// UpgradeInfoFilePath returns the upgrade-info.json file path
func (c *Config) UpgradeInfoFilePath() string {
	return filepath.Join(c.Home, "data", "upgrade-info.json")
}

// SymLinkToGenesis creates a symbolic link from current to genesis
func (c *Config) SymLinkToGenesis() error {
	current := c.CurrentDir()
	genesis := c.GenesisDir()

	// Remove existing link if it exists
	if err := os.RemoveAll(current); err != nil {
		return fmt.Errorf("failed to remove current link: %w", err)
	}

	// Create relative path for the symlink
	relPath, err := filepath.Rel(filepath.Dir(current), genesis)
	if err != nil {
		return fmt.Errorf("failed to create relative path: %w", err)
	}

	// Create the symbolic link
	if err := os.Symlink(relPath, current); err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}

	return nil
}

// SetCurrentUpgrade updates the current symbolic link to point to an upgrade
func (c *Config) SetCurrentUpgrade(name string) error {
	current := c.CurrentDir()
	upgradeDir := c.UpgradeDir(name)

	// Verify upgrade directory exists
	if _, err := os.Stat(upgradeDir); err != nil {
		return fmt.Errorf("upgrade directory does not exist: %w", err)
	}

	// Remove existing link
	if err := os.RemoveAll(current); err != nil {
		return fmt.Errorf("failed to remove current link: %w", err)
	}

	// Create relative path for the symlink
	relPath, err := filepath.Rel(filepath.Dir(current), upgradeDir)
	if err != nil {
		return fmt.Errorf("failed to create relative path: %w", err)
	}

	// Create new symbolic link
	if err := os.Symlink(relPath, current); err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}

	return nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Home == "" {
		return fmt.Errorf("daemon home directory not set")
	}

	if c.Name == "" {
		return fmt.Errorf("daemon name not set")
	}

	if c.PollInterval < 100*time.Millisecond {
		return fmt.Errorf("poll interval too short (minimum 100ms)")
	}

	return nil
}