package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/wemix/wemixvisor/pkg/logger"
)

// Migrator handles configuration migration between versions
type Migrator struct {
	migrations map[string]MigrationFunc
	logger     *logger.Logger
}

// MigrationFunc defines a migration function signature
type MigrationFunc func(*Config) (*Config, error)

// NewMigrator creates a new configuration migrator
func NewMigrator(logger *logger.Logger) *Migrator {
	m := &Migrator{
		migrations: make(map[string]MigrationFunc),
		logger:     logger,
	}

	// Register migrations
	m.registerMigrations()

	return m
}

// Migrate migrates configuration from one version to another
func (m *Migrator) Migrate(cfg *Config, fromVersion, toVersion string) (*Config, error) {
	if cfg == nil {
		return nil, fmt.Errorf("configuration is nil")
	}

	// Parse versions
	from, err := parseVersion(fromVersion)
	if err != nil {
		return nil, fmt.Errorf("invalid from version %s: %w", fromVersion, err)
	}

	to, err := parseVersion(toVersion)
	if err != nil {
		return nil, fmt.Errorf("invalid to version %s: %w", toVersion, err)
	}

	// Check if migration is needed
	if from.Equal(to) {
		m.logger.Info("no migration needed")
		return cfg, nil
	}

	if from.GreaterThan(to) {
		return nil, fmt.Errorf("cannot downgrade from %s to %s", fromVersion, toVersion)
	}

	// Create a copy of the config
	migrated := m.cloneConfig(cfg)

	// Apply migrations in sequence
	current := from
	for !current.Equal(to) {
		next := m.getNextVersion(current, to)
		migrationKey := fmt.Sprintf("%s->%s", current.String(), next.String())

		migration, exists := m.migrations[migrationKey]
		if !exists {
			// Try generic migration
			migration = m.genericMigration
		}

		m.logger.Info("applying migration from " + current.String() + " to " + next.String())

		migrated, err = migration(migrated)
		if err != nil {
			return nil, fmt.Errorf("migration %s failed: %w", migrationKey, err)
		}

		current = next
	}

	m.logger.Info("migration completed from " + fromVersion + " to " + toVersion)
	return migrated, nil
}

// BackupConfig creates a backup of the current configuration
func (m *Migrator) BackupConfig(cfg *Config, path string) error {
	if cfg == nil {
		return fmt.Errorf("configuration is nil")
	}

	// Create backup directory if needed
	backupDir := filepath.Dir(path)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Add timestamp to backup filename
	timestamp := time.Now().Format("20060102-150405")
	backupPath := fmt.Sprintf("%s.%s.backup", path, timestamp)

	// Marshal config to JSON
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write backup file
	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write backup: %w", err)
	}

	m.logger.Info("configuration backed up to " + backupPath)
	return nil
}

// RestoreConfig restores configuration from a backup
func (m *Migrator) RestoreConfig(backupPath string) (*Config, error) {
	// Read backup file
	data, err := os.ReadFile(backupPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup: %w", err)
	}

	// Unmarshal config
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal backup: %w", err)
	}

	m.logger.Info("configuration restored from " + backupPath)
	return &cfg, nil
}

// registerMigrations registers all migration functions
func (m *Migrator) registerMigrations() {
	// v0.1.0 -> v0.2.0: Add backup and hook support
	m.migrations["0.1.0->0.2.0"] = m.migrateV1ToV2

	// v0.2.0 -> v0.3.0: Add WBFT and batch upgrade support
	m.migrations["0.2.0->0.3.0"] = m.migrateV2ToV3

	// v0.3.0 -> v0.4.0: Add lifecycle management
	m.migrations["0.3.0->0.4.0"] = m.migrateV3ToV4

	// v0.4.0 -> v0.5.0: Add configuration management
	m.migrations["0.4.0->0.5.0"] = m.migrateV4ToV5
}

// migrateV1ToV2 migrates from v0.1.0 to v0.2.0
func (m *Migrator) migrateV1ToV2(cfg *Config) (*Config, error) {
	// Add new fields with defaults
	if cfg.DataBackupPath == "" {
		cfg.DataBackupPath = filepath.Join(cfg.Home, "backups")
	}

	if cfg.ShutdownGrace == 0 {
		cfg.ShutdownGrace = 30 * time.Second
	}

	if cfg.PreUpgradeMaxRetries == 0 {
		cfg.PreUpgradeMaxRetries = 3
	}

	return cfg, nil
}

// migrateV2ToV3 migrates from v0.2.0 to v0.3.0
func (m *Migrator) migrateV2ToV3(cfg *Config) (*Config, error) {
	// Add WBFT support fields
	if cfg.RPCAddress == "" {
		cfg.RPCAddress = "localhost:8545"
	}

	// Add download URLs map if not exists
	if cfg.DownloadURLs == nil {
		cfg.DownloadURLs = make(map[string]string)
	}

	return cfg, nil
}

// migrateV3ToV4 migrates from v0.3.0 to v0.4.0
func (m *Migrator) migrateV3ToV4(cfg *Config) (*Config, error) {
	// Add lifecycle management fields
	if cfg.HealthCheckInterval == 0 {
		cfg.HealthCheckInterval = 30 * time.Second
	}

	if cfg.MetricsInterval == 0 {
		cfg.MetricsInterval = 60 * time.Second
	}

	if cfg.MaxRestarts == 0 {
		cfg.MaxRestarts = 5
	}

	// Set default RPC port if not set
	if cfg.RPCPort == 0 {
		// Try to extract from RPCAddress
		if cfg.RPCAddress != "" {
			parts := strings.Split(cfg.RPCAddress, ":")
			if len(parts) == 2 {
				var port int
				fmt.Sscanf(parts[1], "%d", &port)
				cfg.RPCPort = port
			}
		}
		if cfg.RPCPort == 0 {
			cfg.RPCPort = 8545 // Default geth RPC port
		}
	}

	return cfg, nil
}

// migrateV4ToV5 migrates from v0.4.0 to v0.5.0
func (m *Migrator) migrateV4ToV5(cfg *Config) (*Config, error) {
	// v0.5.0 introduces split configuration model
	// The Config struct remains compatible, but we add new fields

	// Add template-related defaults
	if cfg.ConfigVersion == "" {
		cfg.ConfigVersion = "0.5.0"
	}

	// Ensure network ID is set
	if cfg.NetworkID == 0 {
		// Try to detect from validator mode
		if cfg.ValidatorMode {
			cfg.NetworkID = 1111 // Mainnet for validators
		} else {
			cfg.NetworkID = 1112 // Testnet default
		}
	}

	// Set chain ID if not set
	if cfg.ChainID == "" {
		cfg.ChainID = fmt.Sprintf("%d", cfg.NetworkID)
	}

	return cfg, nil
}

// genericMigration performs generic migration steps
func (m *Migrator) genericMigration(cfg *Config) (*Config, error) {
	// Generic migration just ensures required fields are set
	if cfg.Home == "" {
		cfg.Home = filepath.Join(os.Getenv("HOME"), ".wemixvisor")
	}

	if cfg.Name == "" {
		cfg.Name = "wemixd"
	}

	return cfg, nil
}

// getNextVersion determines the next version in migration path
func (m *Migrator) getNextVersion(current, target *Version) *Version {
	// Define version progression
	versions := []*Version{
		{Major: 0, Minor: 1, Patch: 0},
		{Major: 0, Minor: 2, Patch: 0},
		{Major: 0, Minor: 3, Patch: 0},
		{Major: 0, Minor: 4, Patch: 0},
		{Major: 0, Minor: 5, Patch: 0},
		{Major: 0, Minor: 6, Patch: 0},
		{Major: 0, Minor: 7, Patch: 0},
		{Major: 1, Minor: 0, Patch: 0},
	}

	for i, v := range versions {
		if current.Equal(v) && i+1 < len(versions) {
			next := versions[i+1]
			if next.GreaterThan(target) {
				return target
			}
			return next
		}
	}

	return target
}

// cloneConfig creates a deep copy of the configuration
func (m *Migrator) cloneConfig(cfg *Config) *Config {
	// Use JSON marshal/unmarshal for deep copy
	data, _ := json.Marshal(cfg)
	var clone Config
	json.Unmarshal(data, &clone)
	return &clone
}

// Version represents a semantic version
type Version struct {
	Major int
	Minor int
	Patch int
}

// parseVersion parses a version string
func parseVersion(s string) (*Version, error) {
	s = strings.TrimPrefix(s, "v")

	var v Version
	n, err := fmt.Sscanf(s, "%d.%d.%d", &v.Major, &v.Minor, &v.Patch)
	if err != nil || n != 3 {
		// Try without patch version
		n, err = fmt.Sscanf(s, "%d.%d", &v.Major, &v.Minor)
		if err != nil || n != 2 {
			return nil, fmt.Errorf("invalid version format: %s", s)
		}
	}

	return &v, nil
}

// String returns the string representation of the version
func (v *Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// Equal checks if two versions are equal
func (v *Version) Equal(other *Version) bool {
	return v.Major == other.Major && v.Minor == other.Minor && v.Patch == other.Patch
}

// GreaterThan checks if this version is greater than another
func (v *Version) GreaterThan(other *Version) bool {
	if v.Major > other.Major {
		return true
	}
	if v.Major < other.Major {
		return false
	}
	if v.Minor > other.Minor {
		return true
	}
	if v.Minor < other.Minor {
		return false
	}
	return v.Patch > other.Patch
}

// LessThan checks if this version is less than another
func (v *Version) LessThan(other *Version) bool {
	return other.GreaterThan(v)
}