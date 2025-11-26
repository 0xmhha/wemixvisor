package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Default configuration values
const (
	DefaultDaemonName            = "wemixd"
	DefaultShutdownGrace         = 30 * time.Second
	DefaultPollInterval          = 300 * time.Millisecond
	DefaultMaxRestarts           = 5
	DefaultHealthCheckInterval   = 30 * time.Second
	DefaultRPCPort               = 8545
	DefaultNetwork               = "mainnet"
	DefaultRPCAddress            = "localhost:8545"
	DefaultTimeFormatLogs        = "kitchen"
	DefaultMetricsPort           = 9090
	DefaultMetricsPath           = "/metrics"
	DefaultMetricsInterval       = 15 * time.Second
	DefaultAPIPort               = 8080
	DefaultAPIHost               = "0.0.0.0"
	DefaultAPIRateLimit          = 100
	DefaultAlertingEvalInterval  = 30 * time.Second
	DefaultAlertingRetention     = 24 * time.Hour
	DefaultCacheSize             = 1000
	DefaultCacheTTL              = 1 * time.Hour
	DefaultMaxConnections        = 100
	DefaultMaxWorkers            = 10
	DefaultGCPercent             = 100
	DefaultProfileInterval       = 30 * time.Second
	DefaultHeightPollInterval    = 5 * time.Second
	DefaultConfigVersion         = "0.8.0"
	MinPollInterval              = 100 * time.Millisecond
)

// Config holds all configuration for Wemixvisor
type Config struct {
	// Core settings
	Home string `mapstructure:"daemon_home"`
	Name string `mapstructure:"daemon_name"`
	Args []string

	// Upgrade settings
	AllowDownloadBinaries    bool          `mapstructure:"daemon_allow_download_binaries"`
	DownloadMustHaveChecksum bool          `mapstructure:"daemon_download_must_have_checksum"`
	RestartAfterUpgrade      bool          `mapstructure:"daemon_restart_after_upgrade"`
	RestartDelay             time.Duration `mapstructure:"daemon_restart_delay"`

	// Process management
	ShutdownGrace    time.Duration `mapstructure:"daemon_shutdown_grace"`
	PollInterval     time.Duration `mapstructure:"daemon_poll_interval"`
	RestartOnFailure bool          `mapstructure:"daemon_restart_on_failure"`
	MaxRestarts      int           `mapstructure:"daemon_max_restarts"`

	// Health and monitoring
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
	RPCAddress    string `mapstructure:"daemon_rpc_address"`
	ValidatorMode bool   `mapstructure:"validator_mode"`
	DisableRecase bool   `mapstructure:"cosmovisor_disable_recase"`

	// Network settings
	NetworkID uint64 `mapstructure:"network_id"`
	ChainID   string `mapstructure:"chain_id"`

	// Download settings
	DownloadURLs       map[string]string `mapstructure:"download_urls"`
	UnsafeSkipChecksum bool              `mapstructure:"unsafe_skip_checksum"`

	// Logging
	DisableLogs    bool   `mapstructure:"cosmovisor_disable_logs"`
	ColorLogs      bool   `mapstructure:"cosmovisor_color_logs"`
	TimeFormatLogs string `mapstructure:"cosmovisor_timeformat_logs"`

	// Metrics settings
	MetricsEnabled            bool          `mapstructure:"metrics_enabled"`
	MetricsPort               int           `mapstructure:"metrics_port"`
	MetricsPath               string        `mapstructure:"metrics_path"`
	MetricsCollectionInterval time.Duration `mapstructure:"metrics_collection_interval"`
	EnableSystemMetrics       bool          `mapstructure:"enable_system_metrics"`
	EnableAppMetrics          bool          `mapstructure:"enable_app_metrics"`
	EnableGovMetrics          bool          `mapstructure:"enable_gov_metrics"`
	EnablePerfMetrics         bool          `mapstructure:"enable_perf_metrics"`

	// API Server settings
	APIEnabled     bool     `mapstructure:"api_enabled"`
	APIPort        int      `mapstructure:"api_port"`
	APIHost        string   `mapstructure:"api_host"`
	APIEnableAuth  bool     `mapstructure:"api_enable_auth"`
	APIKey         string   `mapstructure:"api_key"`
	APIJWTSecret   string   `mapstructure:"api_jwt_secret"`
	APICORSOrigins []string `mapstructure:"api_cors_origins"`
	APIRateLimit   int      `mapstructure:"api_rate_limit"`

	// Alerting settings
	AlertingEnabled      bool          `mapstructure:"alerting_enabled"`
	AlertingEvalInterval time.Duration `mapstructure:"alerting_evaluation_interval"`
	AlertingRetention    time.Duration `mapstructure:"alerting_retention"`
	AlertingChannels     []string      `mapstructure:"alerting_channels"`

	// Performance settings
	EnableCaching   bool          `mapstructure:"enable_caching"`
	CacheSize       int           `mapstructure:"cache_size"`
	CacheTTL        time.Duration `mapstructure:"cache_ttl"`
	EnablePooling   bool          `mapstructure:"enable_pooling"`
	MaxConnections  int           `mapstructure:"max_connections"`
	MaxWorkers      int           `mapstructure:"max_workers"`
	EnableGCTuning  bool          `mapstructure:"enable_gc_tuning"`
	GCPercent       int           `mapstructure:"gc_percent"`
	EnableProfiling bool          `mapstructure:"enable_profiling"`
	ProfileInterval time.Duration `mapstructure:"profile_interval"`

	// Governance settings
	GovernanceEnabled bool `mapstructure:"governance_enabled"`

	// Upgrade Automation settings
	UpgradeEnabled     bool          `mapstructure:"upgrade_enabled"`
	HeightPollInterval time.Duration `mapstructure:"height_poll_interval"`

	// CLI options
	Daemon     bool `mapstructure:"daemon"`
	JSONOutput bool `mapstructure:"json_output"`
	Quiet      bool `mapstructure:"quiet"`

	// Configuration version
	ConfigVersion string `mapstructure:"config_version"`
}

// DefaultConfig returns a Config with default values
func DefaultConfig() *Config {
	home := getDefaultHome()

	return &Config{
		Home:                 home,
		Name:                 DefaultDaemonName,
		RestartAfterUpgrade:  true,
		ShutdownGrace:        DefaultShutdownGrace,
		PollInterval:         DefaultPollInterval,
		RestartOnFailure:     true,
		MaxRestarts:          DefaultMaxRestarts,
		HealthCheckInterval:  DefaultHealthCheckInterval,
		RPCPort:              DefaultRPCPort,
		Environment:          make(map[string]string),
		Network:              DefaultNetwork,
		DataBackupPath:       filepath.Join(home, "backups"),
		RPCAddress:           DefaultRPCAddress,
		ColorLogs:            true,
		TimeFormatLogs:       DefaultTimeFormatLogs,

		// Metrics defaults
		MetricsPort:               DefaultMetricsPort,
		MetricsPath:               DefaultMetricsPath,
		MetricsCollectionInterval: DefaultMetricsInterval,
		EnableSystemMetrics:       true,
		EnableAppMetrics:          true,
		EnableGovMetrics:          true,
		EnablePerfMetrics:         true,

		// API Server defaults
		APIPort:        DefaultAPIPort,
		APIHost:        DefaultAPIHost,
		APICORSOrigins: []string{"*"},
		APIRateLimit:   DefaultAPIRateLimit,

		// Alerting defaults
		AlertingEvalInterval: DefaultAlertingEvalInterval,
		AlertingRetention:    DefaultAlertingRetention,
		AlertingChannels:     []string{},

		// Performance defaults
		EnableCaching:   true,
		CacheSize:       DefaultCacheSize,
		CacheTTL:        DefaultCacheTTL,
		EnablePooling:   true,
		MaxConnections:  DefaultMaxConnections,
		MaxWorkers:      DefaultMaxWorkers,
		GCPercent:       DefaultGCPercent,
		ProfileInterval: DefaultProfileInterval,

		// Upgrade Automation defaults
		UpgradeEnabled:     true,
		HeightPollInterval: DefaultHeightPollInterval,

		ConfigVersion: DefaultConfigVersion,
	}
}

// getDefaultHome returns the default daemon home directory
func getDefaultHome() string {
	if home := os.Getenv("DAEMON_HOME"); home != "" {
		return home
	}
	return filepath.Join(os.Getenv("HOME"), ".wemixd")
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Home == "" {
		return fmt.Errorf("daemon home directory not set")
	}
	if c.Name == "" {
		return fmt.Errorf("daemon name not set")
	}
	if c.PollInterval < MinPollInterval {
		return fmt.Errorf("poll interval too short (minimum %v)", MinPollInterval)
	}
	return nil
}
