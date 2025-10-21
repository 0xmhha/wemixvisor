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

	// Phase 4: Node lifecycle management
	RestartOnFailure    bool              `mapstructure:"daemon_restart_on_failure"`
	MaxRestarts         int               `mapstructure:"daemon_max_restarts"`
	HealthCheckInterval time.Duration     `mapstructure:"daemon_health_check_interval"`
	MetricsInterval     time.Duration     `mapstructure:"daemon_metrics_interval"`
	RPCPort             int               `mapstructure:"daemon_rpc_port"`
	LogFile             string            `mapstructure:"daemon_log_file"`
	Environment         map[string]string `mapstructure:"daemon_environment"`
	Network             string            `mapstructure:"daemon_network"`
	Debug               bool              `mapstructure:"daemon_debug"`

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

	// Network settings
	NetworkID uint64 `mapstructure:"network_id"`
	ChainID   string `mapstructure:"chain_id"`

	// Download settings
	DownloadURLs       map[string]string `mapstructure:"download_urls"`
	UnsafeSkipChecksum bool              `mapstructure:"unsafe_skip_checksum"`

	// Logging
	DisableLogs     bool   `mapstructure:"cosmovisor_disable_logs"`
	ColorLogs       bool   `mapstructure:"cosmovisor_color_logs"`
	TimeFormatLogs  string `mapstructure:"cosmovisor_timeformat_logs"`

	// Phase 7: Metrics settings
	MetricsEnabled          bool          `mapstructure:"metrics_enabled"`
	MetricsPort             int           `mapstructure:"metrics_port"`
	MetricsPath             string        `mapstructure:"metrics_path"`
	MetricsCollectionInterval time.Duration `mapstructure:"metrics_collection_interval"`
	EnableSystemMetrics     bool          `mapstructure:"enable_system_metrics"`
	EnableAppMetrics        bool          `mapstructure:"enable_app_metrics"`
	EnableGovMetrics        bool          `mapstructure:"enable_gov_metrics"`
	EnablePerfMetrics       bool          `mapstructure:"enable_perf_metrics"`

	// Phase 7: API Server settings
	APIEnabled      bool     `mapstructure:"api_enabled"`
	APIPort         int      `mapstructure:"api_port"`
	APIHost         string   `mapstructure:"api_host"`
	APIEnableAuth   bool     `mapstructure:"api_enable_auth"`
	APIKey          string   `mapstructure:"api_key"`
	APIJWTSecret    string   `mapstructure:"api_jwt_secret"`
	APICORSOrigins  []string `mapstructure:"api_cors_origins"`
	APIRateLimit    int      `mapstructure:"api_rate_limit"`

	// Phase 7: Alerting settings
	AlertingEnabled         bool          `mapstructure:"alerting_enabled"`
	AlertingEvalInterval    time.Duration `mapstructure:"alerting_evaluation_interval"`
	AlertingRetention       time.Duration `mapstructure:"alerting_retention"`
	AlertingChannels        []string      `mapstructure:"alerting_channels"`

	// Phase 7: Performance settings
	EnableCaching       bool          `mapstructure:"enable_caching"`
	CacheSize           int           `mapstructure:"cache_size"`
	CacheTTL            time.Duration `mapstructure:"cache_ttl"`
	EnablePooling       bool          `mapstructure:"enable_pooling"`
	MaxConnections      int           `mapstructure:"max_connections"`
	MaxWorkers          int           `mapstructure:"max_workers"`
	EnableGCTuning      bool          `mapstructure:"enable_gc_tuning"`
	GCPercent           int           `mapstructure:"gc_percent"`
	EnableProfiling     bool          `mapstructure:"enable_profiling"`
	ProfileInterval     time.Duration `mapstructure:"profile_interval"`

	// Governance settings
	GovernanceEnabled bool `mapstructure:"governance_enabled"`

	// Phase 8: Upgrade Automation settings
	UpgradeEnabled      bool          `mapstructure:"upgrade_enabled"`
	HeightPollInterval  time.Duration `mapstructure:"height_poll_interval"`

	// CLI options
	Daemon      bool `mapstructure:"daemon"`       // Run in background
	JSONOutput  bool `mapstructure:"json_output"`  // Output in JSON format
	Quiet       bool `mapstructure:"quiet"`        // Suppress output

	// Configuration version
	ConfigVersion string `mapstructure:"config_version"`
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
		ShutdownGrace:            30 * time.Second,
		PollInterval:             300 * time.Millisecond,
		RestartOnFailure:         true,
		MaxRestarts:              5,
		HealthCheckInterval:      30 * time.Second,
		RPCPort:                  8545,
		LogFile:                  "",
		Environment:              make(map[string]string),
		Network:                  "mainnet",
		Debug:                    false,
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

		// Phase 7: Metrics defaults
		MetricsEnabled:            false,
		MetricsPort:               9090,
		MetricsPath:               "/metrics",
		MetricsCollectionInterval: 15 * time.Second,
		EnableSystemMetrics:       true,
		EnableAppMetrics:          true,
		EnableGovMetrics:          true,
		EnablePerfMetrics:         true,

		// Phase 7: API Server defaults
		APIEnabled:      false,
		APIPort:         8080,
		APIHost:         "0.0.0.0",
		APIEnableAuth:   false,
		APIKey:          "",
		APIJWTSecret:    "",
		APICORSOrigins:  []string{"*"},
		APIRateLimit:    100,

		// Phase 7: Alerting defaults
		AlertingEnabled:      false,
		AlertingEvalInterval: 30 * time.Second,
		AlertingRetention:    24 * time.Hour,
		AlertingChannels:     []string{},

		// Phase 7: Performance defaults
		EnableCaching:       true,
		CacheSize:           1000,
		CacheTTL:            1 * time.Hour,
		EnablePooling:       true,
		MaxConnections:      100,
		MaxWorkers:          10,
		EnableGCTuning:      false,
		GCPercent:           100,
		EnableProfiling:     false,
		ProfileInterval:     30 * time.Second,

		GovernanceEnabled: false,

		// Phase 8: Upgrade Automation defaults
		UpgradeEnabled:     true,
		HeightPollInterval: 5 * time.Second,

		ConfigVersion: "0.8.0",
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