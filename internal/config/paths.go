package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// Directory and file name constants
const (
	WemixvisorDirName   = "wemixvisor"
	CurrentDirName      = "current"
	GenesisDirName      = "genesis"
	UpgradesDirName     = "upgrades"
	BinDirName          = "bin"
	DataDirName         = "data"
	UpgradeInfoFileName = "upgrade-info.json"
)

// PathProvider defines methods for accessing wemixvisor paths
type PathProvider interface {
	WemixvisorDir() string
	CurrentDir() string
	GenesisDir() string
	UpgradesDir() string
	UpgradeDir(name string) string
	CurrentBin() string
	GenesisBin() string
	UpgradeBin(name string) string
	UpgradeInfoFilePath() string
}

// WemixvisorDir returns the wemixvisor directory path
func (c *Config) WemixvisorDir() string {
	return filepath.Join(c.Home, WemixvisorDirName)
}

// CurrentDir returns the current binary directory path
func (c *Config) CurrentDir() string {
	return filepath.Join(c.WemixvisorDir(), CurrentDirName)
}

// GenesisDir returns the genesis binary directory path
func (c *Config) GenesisDir() string {
	return filepath.Join(c.WemixvisorDir(), GenesisDirName)
}

// UpgradesDir returns the upgrades directory path
func (c *Config) UpgradesDir() string {
	return filepath.Join(c.WemixvisorDir(), UpgradesDirName)
}

// UpgradeDir returns the directory for a specific upgrade
func (c *Config) UpgradeDir(name string) string {
	return filepath.Join(c.UpgradesDir(), name)
}

// CurrentBin returns the current binary path
func (c *Config) CurrentBin() string {
	return filepath.Join(c.CurrentDir(), BinDirName, c.Name)
}

// GenesisBin returns the genesis binary path
func (c *Config) GenesisBin() string {
	return filepath.Join(c.GenesisDir(), BinDirName, c.Name)
}

// UpgradeBin returns the binary path for a specific upgrade
func (c *Config) UpgradeBin(name string) string {
	return filepath.Join(c.UpgradeDir(name), BinDirName, c.Name)
}

// UpgradeInfoFilePath returns the upgrade-info.json file path
func (c *Config) UpgradeInfoFilePath() string {
	return filepath.Join(c.Home, DataDirName, UpgradeInfoFileName)
}

// SymlinkManager handles symbolic link operations for binary versions
type SymlinkManager struct {
	config *Config
}

// NewSymlinkManager creates a new SymlinkManager
func NewSymlinkManager(cfg *Config) *SymlinkManager {
	return &SymlinkManager{config: cfg}
}

// LinkToGenesis creates a symbolic link from current to genesis
func (s *SymlinkManager) LinkToGenesis() error {
	return s.createSymlink(s.config.GenesisDir())
}

// LinkToUpgrade creates a symbolic link from current to specified upgrade
func (s *SymlinkManager) LinkToUpgrade(name string) error {
	upgradeDir := s.config.UpgradeDir(name)

	if _, err := os.Stat(upgradeDir); err != nil {
		return fmt.Errorf("upgrade directory does not exist: %w", err)
	}

	return s.createSymlink(upgradeDir)
}

// createSymlink creates a relative symbolic link from current to target
func (s *SymlinkManager) createSymlink(targetDir string) error {
	currentDir := s.config.CurrentDir()

	if err := os.RemoveAll(currentDir); err != nil {
		return fmt.Errorf("failed to remove current link: %w", err)
	}

	relPath, err := filepath.Rel(filepath.Dir(currentDir), targetDir)
	if err != nil {
		return fmt.Errorf("failed to create relative path: %w", err)
	}

	if err := os.Symlink(relPath, currentDir); err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}

	return nil
}

// SymLinkToGenesis creates a symbolic link from current to genesis
// Deprecated: Use NewSymlinkManager().LinkToGenesis() instead
func (c *Config) SymLinkToGenesis() error {
	return NewSymlinkManager(c).LinkToGenesis()
}

// SetCurrentUpgrade updates the current symbolic link to point to an upgrade
// Deprecated: Use NewSymlinkManager().LinkToUpgrade() instead
func (c *Config) SetCurrentUpgrade(name string) error {
	return NewSymlinkManager(c).LinkToUpgrade(name)
}
