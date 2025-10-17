package config

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/pelletier/go-toml/v2"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	"github.com/wemix/wemixvisor/pkg/logger"
)

// Manager manages configuration with hot-reload support
type Manager struct {
	// Configuration sources
	wemixvisorConfig *WemixvisorConfig
	nodeConfig       *NodeConfig
	mergedConfig     *Config

	// Template management
	templates *TemplateManager

	// Validation and migration
	validator *Validator
	migrator  *Migrator

	// File watching
	watcher    *fsnotify.Watcher
	configPath string

	// Synchronization
	mu       sync.RWMutex
	updateCh chan ConfigUpdate

	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	logger *logger.Logger
}

// ConfigUpdate represents a configuration change event
type ConfigUpdate struct {
	Type      UpdateType
	Path      string
	OldConfig *Config
	NewConfig *Config
	Error     error
}

// UpdateType defines the type of configuration update
type UpdateType int

const (
	UpdateTypeReload UpdateType = iota
	UpdateTypeHotReload
	UpdateTypeMigration
	UpdateTypeTemplate
)

// WemixvisorConfig represents Wemixvisor-specific configuration
type WemixvisorConfig struct {
	// Core settings
	Home string `toml:"home" yaml:"home" json:"home"`
	Name string `toml:"name" yaml:"name" json:"name"`

	// Process management
	MaxRestarts   int           `toml:"max_restarts" yaml:"max_restarts" json:"max_restarts"`
	RestartDelay  time.Duration `toml:"restart_delay" yaml:"restart_delay" json:"restart_delay"`
	ShutdownGrace time.Duration `toml:"shutdown_grace" yaml:"shutdown_grace" json:"shutdown_grace"`

	// Monitoring
	HealthCheckInterval time.Duration `toml:"health_check_interval" yaml:"health_check_interval" json:"health_check_interval"`
	MetricsInterval     time.Duration `toml:"metrics_interval" yaml:"metrics_interval" json:"metrics_interval"`
	MetricsEnabled      bool          `toml:"metrics_enabled" yaml:"metrics_enabled" json:"metrics_enabled"`

	// Upgrade settings
	AllowDownloadBinaries bool          `toml:"allow_download_binaries" yaml:"allow_download_binaries" json:"allow_download_binaries"`
	AutoBackup            bool          `toml:"auto_backup" yaml:"auto_backup" json:"auto_backup"`
	PreUpgradeTimeout     time.Duration `toml:"pre_upgrade_timeout" yaml:"pre_upgrade_timeout" json:"pre_upgrade_timeout"`

	// Logging
	LogLevel      string `toml:"log_level" yaml:"log_level" json:"log_level"`
	LogFormat     string `toml:"log_format" yaml:"log_format" json:"log_format"`
	LogTimeFormat string `toml:"log_time_format" yaml:"log_time_format" json:"log_time_format"`
}

// NodeConfig represents node-specific configuration
type NodeConfig struct {
	// Network settings
	NetworkID uint64 `toml:"network_id" yaml:"network_id" json:"network_id"`
	ChainID   string `toml:"chain_id" yaml:"chain_id" json:"chain_id"`

	// RPC settings
	RPCAddr string   `toml:"rpc_addr" yaml:"rpc_addr" json:"rpc_addr"`
	RPCPort int      `toml:"rpc_port" yaml:"rpc_port" json:"rpc_port"`
	RPCAPIs []string `toml:"rpc_apis" yaml:"rpc_apis" json:"rpc_apis"`

	// WebSocket settings
	WSAddr string   `toml:"ws_addr" yaml:"ws_addr" json:"ws_addr"`
	WSPort int      `toml:"ws_port" yaml:"ws_port" json:"ws_port"`
	WSAPIs []string `toml:"ws_apis" yaml:"ws_apis" json:"ws_apis"`

	// P2P settings
	P2PPort   int      `toml:"p2p_port" yaml:"p2p_port" json:"p2p_port"`
	Bootnodes []string `toml:"bootnodes" yaml:"bootnodes" json:"bootnodes"`
	MaxPeers  int      `toml:"max_peers" yaml:"max_peers" json:"max_peers"`

	// Consensus settings (WBFT)
	ValidatorMode bool   `toml:"validator_mode" yaml:"validator_mode" json:"validator_mode"`
	ValidatorKey  string `toml:"validator_key" yaml:"validator_key" json:"validator_key"`

	// Data directory
	DataDir string `toml:"data_dir" yaml:"data_dir" json:"data_dir"`

	// Additional geth flags
	ExtraFlags []string `toml:"extra_flags" yaml:"extra_flags" json:"extra_flags"`
}

// NewManager creates a new configuration manager
func NewManager(configPath string, logger *logger.Logger) (*Manager, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Create file watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create watcher: %w", err)
	}

	m := &Manager{
		configPath: configPath,
		watcher:    watcher,
		updateCh:   make(chan ConfigUpdate, 10),
		ctx:        ctx,
		cancel:     cancel,
		logger:     logger,
	}

	// Initialize components
	m.templates = NewTemplateManager(logger)
	m.validator = NewValidator(logger)
	m.migrator = NewMigrator(logger)

	// Load initial configuration
	if err := m.loadConfig(); err != nil {
		cancel()
		watcher.Close()
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Start watching for changes
	if err := m.startWatching(); err != nil {
		cancel()
		watcher.Close()
		return nil, fmt.Errorf("failed to start watching: %w", err)
	}

	return m, nil
}

// GetConfig returns the current merged configuration
func (m *Manager) GetConfig() *Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.mergedConfig
}

// GetWemixvisorConfig returns the wemixvisor configuration
func (m *Manager) GetWemixvisorConfig() *WemixvisorConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.wemixvisorConfig
}

// GetNodeConfig returns the node configuration
func (m *Manager) GetNodeConfig() *NodeConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.nodeConfig
}

// UpdateConfig updates the configuration with hot-reload
func (m *Manager) UpdateConfig(key string, value interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Create backup of current config
	oldConfig := m.cloneConfig(m.mergedConfig)

	// Update the configuration
	if err := m.updateField(key, value); err != nil {
		return fmt.Errorf("failed to update field %s: %w", key, err)
	}

	// Validate the new configuration
	if err := m.validator.Validate(m.mergedConfig); err != nil {
		// Rollback on validation failure
		m.mergedConfig = oldConfig
		return fmt.Errorf("validation failed: %w", err)
	}

	// Save to disk
	if err := m.saveConfig(); err != nil {
		// Rollback on save failure
		m.mergedConfig = oldConfig
		return fmt.Errorf("failed to save config: %w", err)
	}

	// Send update notification
	m.notifyUpdate(UpdateTypeHotReload, oldConfig, m.mergedConfig)

	return nil
}

// ApplyTemplate applies a network template
func (m *Manager) ApplyTemplate(templateName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Get the template
	template, err := m.templates.GetTemplate(templateName)
	if err != nil {
		return fmt.Errorf("failed to get template %s: %w", templateName, err)
	}

	// Create backup
	oldConfig := m.cloneConfig(m.mergedConfig)

	// Apply template
	if err := m.applyTemplateConfig(template); err != nil {
		return fmt.Errorf("failed to apply template: %w", err)
	}

	// Validate
	if err := m.validator.Validate(m.mergedConfig); err != nil {
		m.mergedConfig = oldConfig
		return fmt.Errorf("validation failed: %w", err)
	}

	// Save
	if err := m.saveConfig(); err != nil {
		m.mergedConfig = oldConfig
		return fmt.Errorf("failed to save: %w", err)
	}

	// Notify
	m.notifyUpdate(UpdateTypeTemplate, oldConfig, m.mergedConfig)

	return nil
}

// Validate validates the current configuration
func (m *Manager) Validate() error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.validator.Validate(m.mergedConfig)
}

// Migrate migrates configuration from an older version
func (m *Manager) Migrate(fromVersion, toVersion string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Create backup
	oldConfig := m.cloneConfig(m.mergedConfig)

	// Perform migration
	newConfig, err := m.migrator.Migrate(m.mergedConfig, fromVersion, toVersion)
	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	// Validate migrated config
	if err := m.validator.Validate(newConfig); err != nil {
		return fmt.Errorf("validation failed after migration: %w", err)
	}

	// Update config
	m.mergedConfig = newConfig
	m.splitConfig() // Update individual configs

	// Save
	if err := m.saveConfig(); err != nil {
		m.mergedConfig = oldConfig
		return fmt.Errorf("failed to save migrated config: %w", err)
	}

	// Notify
	m.notifyUpdate(UpdateTypeMigration, oldConfig, m.mergedConfig)

	return nil
}

// GetUpdateChannel returns the update notification channel
func (m *Manager) GetUpdateChannel() <-chan ConfigUpdate {
	return m.updateCh
}

// Stop stops the configuration manager
func (m *Manager) Stop() {
	m.cancel()
	m.watcher.Close()
	close(m.updateCh)
}

// loadConfig loads configuration from disk
func (m *Manager) loadConfig() error {
	// Determine config format from extension
	ext := filepath.Ext(m.configPath)

	data, err := os.ReadFile(m.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Create default config if not exists
			m.applyDefaults()
			return m.saveConfig()
		}
		return fmt.Errorf("failed to read config: %w", err)
	}

	// Parse based on format
	switch ext {
	case ".toml":
		err = toml.Unmarshal(data, &m.wemixvisorConfig)
		if err == nil {
			err = toml.Unmarshal(data, &m.nodeConfig)
		}
	case ".yaml", ".yml":
		err = yaml.Unmarshal(data, &m.wemixvisorConfig)
		if err == nil {
			err = yaml.Unmarshal(data, &m.nodeConfig)
		}
	case ".json":
		err = json.Unmarshal(data, &m.wemixvisorConfig)
		if err == nil {
			err = json.Unmarshal(data, &m.nodeConfig)
		}
	default:
		return fmt.Errorf("unsupported config format: %s", ext)
	}

	if err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	// Merge configurations
	m.mergeConfigs()

	// Validate
	return m.validator.Validate(m.mergedConfig)
}

// saveConfig saves configuration to disk
func (m *Manager) saveConfig() error {
	ext := filepath.Ext(m.configPath)

	// Prepare combined config for saving
	combined := make(map[string]interface{})

	// Add wemixvisor config
	wData, _ := json.Marshal(m.wemixvisorConfig)
	var wMap map[string]interface{}
	json.Unmarshal(wData, &wMap)
	for k, v := range wMap {
		combined[k] = v
	}

	// Add node config
	nData, _ := json.Marshal(m.nodeConfig)
	var nMap map[string]interface{}
	json.Unmarshal(nData, &nMap)
	for k, v := range nMap {
		combined[k] = v
	}

	var data []byte
	var err error

	switch ext {
	case ".toml":
		data, err = toml.Marshal(combined)
	case ".yaml", ".yml":
		data, err = yaml.Marshal(combined)
	case ".json":
		data, err = json.MarshalIndent(combined, "", "  ")
	default:
		return fmt.Errorf("unsupported config format: %s", ext)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write atomically
	tmpPath := m.configPath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return os.Rename(tmpPath, m.configPath)
}

// startWatching starts watching the config file for changes
func (m *Manager) startWatching() error {
	// Watch the directory, not the file (for atomic writes)
	dir := filepath.Dir(m.configPath)
	if err := m.watcher.Add(dir); err != nil {
		return fmt.Errorf("failed to watch directory: %w", err)
	}

	// Start the watcher goroutine
	go m.watchLoop()

	return nil
}

// watchLoop handles file system events
func (m *Manager) watchLoop() {
	for {
		select {
		case <-m.ctx.Done():
			return

		case event, ok := <-m.watcher.Events:
			if !ok {
				return
			}

			// Check if it's our config file
			if filepath.Base(event.Name) != filepath.Base(m.configPath) {
				continue
			}

			// Handle the event
			if event.Op&fsnotify.Write == fsnotify.Write ||
				event.Op&fsnotify.Create == fsnotify.Create {
				m.handleConfigChange()
			}

		case err, ok := <-m.watcher.Errors:
			if !ok {
				return
			}
			m.logger.Error("watcher error", zap.Error(err))
		}
	}
}

// handleConfigChange handles configuration file changes
func (m *Manager) handleConfigChange() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Save current config
	oldConfig := m.cloneConfig(m.mergedConfig)

	// Reload config
	if err := m.loadConfig(); err != nil {
		m.logger.Error("failed to reload config", zap.Error(err))
		m.notifyUpdate(UpdateTypeReload, oldConfig, nil)
		return
	}

	// Notify successful reload
	m.notifyUpdate(UpdateTypeReload, oldConfig, m.mergedConfig)
}

// mergeConfigs merges wemixvisor and node configs into the main config
func (m *Manager) mergeConfigs() {
	m.mergedConfig = &Config{
		// From WemixvisorConfig
		Home:                  m.wemixvisorConfig.Home,
		Name:                  m.wemixvisorConfig.Name,
		RestartDelay:          m.wemixvisorConfig.RestartDelay,
		ShutdownGrace:         m.wemixvisorConfig.ShutdownGrace,
		AllowDownloadBinaries: m.wemixvisorConfig.AllowDownloadBinaries,
		UnsafeSkipBackup:      !m.wemixvisorConfig.AutoBackup,

		// From NodeConfig
		RPCAddress:    fmt.Sprintf("%s:%d", m.nodeConfig.RPCAddr, m.nodeConfig.RPCPort),
		RPCPort:       m.nodeConfig.RPCPort,
		ValidatorMode: m.nodeConfig.ValidatorMode,

		// New fields
		HealthCheckInterval: m.wemixvisorConfig.HealthCheckInterval,
		MetricsInterval:     m.wemixvisorConfig.MetricsInterval,
		MaxRestarts:         m.wemixvisorConfig.MaxRestarts,

		// Logging
		TimeFormatLogs: m.wemixvisorConfig.LogTimeFormat,
	}
}

// splitConfig splits the merged config back into individual configs
func (m *Manager) splitConfig() {
	// Update WemixvisorConfig
	m.wemixvisorConfig.Home = m.mergedConfig.Home
	m.wemixvisorConfig.Name = m.mergedConfig.Name
	m.wemixvisorConfig.RestartDelay = m.mergedConfig.RestartDelay
	m.wemixvisorConfig.ShutdownGrace = m.mergedConfig.ShutdownGrace
	m.wemixvisorConfig.AllowDownloadBinaries = m.mergedConfig.AllowDownloadBinaries
	m.wemixvisorConfig.AutoBackup = !m.mergedConfig.UnsafeSkipBackup

	// Update NodeConfig
	m.nodeConfig.RPCPort = m.mergedConfig.RPCPort
	m.nodeConfig.ValidatorMode = m.mergedConfig.ValidatorMode
}

// applyDefaults applies default configuration values
func (m *Manager) applyDefaults() {
	m.wemixvisorConfig = &WemixvisorConfig{
		Home:                  filepath.Join(os.Getenv("HOME"), ".wemixvisor"),
		Name:                  "wemixd",
		MaxRestarts:           5,
		RestartDelay:          1 * time.Second,
		ShutdownGrace:         30 * time.Second,
		HealthCheckInterval:   30 * time.Second,
		MetricsInterval:       60 * time.Second,
		MetricsEnabled:        true,
		AllowDownloadBinaries: true,
		AutoBackup:            true,
		PreUpgradeTimeout:     5 * time.Minute,
		LogLevel:              "info",
		LogFormat:             "json",
		LogTimeFormat:         "rfc3339",
	}

	m.nodeConfig = &NodeConfig{
		NetworkID: 1112,
		ChainID:   "1112",
		RPCAddr:   "127.0.0.1",
		RPCPort:   8545,
		RPCAPIs:   []string{"eth", "net", "web3"},
		WSAddr:    "127.0.0.1",
		WSPort:    8546,
		WSAPIs:    []string{"eth", "net", "web3"},
		P2PPort:   30303,
		MaxPeers:  25,
		DataDir:   filepath.Join(m.wemixvisorConfig.Home, "data"),
	}

	m.mergeConfigs()
}

// updateField updates a specific configuration field using reflection
func (m *Manager) updateField(key string, value interface{}) error {
	if m.mergedConfig == nil {
		return fmt.Errorf("merged config is nil")
	}

	// Get reflect.Value of mergedConfig
	configValue := reflect.ValueOf(m.mergedConfig).Elem()

	// Find the field by name
	field := configValue.FieldByName(key)
	if !field.IsValid() {
		return fmt.Errorf("field %s not found in configuration", key)
	}

	if !field.CanSet() {
		return fmt.Errorf("field %s cannot be set (may be unexported)", key)
	}

	// Convert value to the correct type and set
	valueReflect := reflect.ValueOf(value)

	// Handle type conversion
	if !valueReflect.Type().AssignableTo(field.Type()) {
		// Try to convert the value to the field type
		if valueReflect.Type().ConvertibleTo(field.Type()) {
			valueReflect = valueReflect.Convert(field.Type())
		} else {
			return fmt.Errorf("value type %v not assignable to field type %v", valueReflect.Type(), field.Type())
		}
	}

	// Set the field value
	field.Set(valueReflect)

	// Update individual configs to keep them in sync
	m.splitConfig()

	return nil
}

// applyTemplateConfig applies a template to the current config
func (m *Manager) applyTemplateConfig(template *Template) error {
	// Apply template values to both configs
	if err := m.templates.ApplyTemplate(template, m.wemixvisorConfig, m.nodeConfig); err != nil {
		return err
	}

	// Merge configs after template application
	m.mergeConfigs()
	return nil
}

// cloneConfig creates a deep copy of the config
func (m *Manager) cloneConfig(cfg *Config) *Config {
	if cfg == nil {
		return nil
	}

	// Deep copy using JSON marshal/unmarshal
	data, _ := json.Marshal(cfg)
	var clone Config
	json.Unmarshal(data, &clone)
	return &clone
}

// notifyUpdate sends an update notification
func (m *Manager) notifyUpdate(updateType UpdateType, oldConfig, newConfig *Config) {
	select {
	case m.updateCh <- ConfigUpdate{
		Type:      updateType,
		Path:      m.configPath,
		OldConfig: oldConfig,
		NewConfig: newConfig,
	}:
	case <-m.ctx.Done():
	default:
		// Channel full, log and continue
		m.logger.Warn("config update channel full, dropping notification")
	}
}
