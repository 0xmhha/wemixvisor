package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"github.com/spf13/cobra"
	"github.com/wemix/wemixvisor/internal/config"
	"github.com/wemix/wemixvisor/pkg/logger"
)

// InitCommand handles the initialization of wemixvisor directory structure
type InitCommand struct {
	config *config.Config
	logger *logger.Logger
}

// NewInitCommand creates a new init command handler
func NewInitCommand(cfg *config.Config, logger *logger.Logger) *InitCommand {
	return &InitCommand{
		config: cfg,
		logger: logger,
	}
}

// Execute performs the initialization
func (c *InitCommand) Execute(args []string) error {
	// Parse init-specific arguments
	home := c.config.Home
	if home == "" {
		home = os.Getenv("DAEMON_HOME")
		if home == "" {
			return fmt.Errorf("DAEMON_HOME not set")
		}
	}

	// Check if directory already exists
	if info, err := os.Stat(home); err == nil {
		if !info.IsDir() {
			return fmt.Errorf("%s exists but is not a directory", home)
		}

		// Check if already initialized
		wemixvisorDir := filepath.Join(home, "wemixvisor")
		if _, err := os.Stat(wemixvisorDir); err == nil {
			c.logger.Info("wemixvisor already initialized", zap.String("home", home))
			return nil
		}
	}

	c.logger.Info("initializing wemixvisor", zap.String("home", home))

	// Create directory structure
	dirs := []string{
		home,
		filepath.Join(home, "wemixvisor"),
		filepath.Join(home, "wemixvisor", "genesis"),
		filepath.Join(home, "wemixvisor", "genesis", "bin"),
		filepath.Join(home, "wemixvisor", "upgrades"),
		filepath.Join(home, "wemixvisor", "backup"),
		filepath.Join(home, "data"),
		filepath.Join(home, "logs"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
		c.logger.Debug("created directory", zap.String("path", dir))
	}

	// Create initial config file
	configPath := filepath.Join(home, "wemixvisor", "config.toml")
	if err := c.createInitialConfig(configPath); err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}

	// Create symbolic link for current version
	currentLink := filepath.Join(home, "wemixvisor", "current")
	genesisDir := filepath.Join(home, "wemixvisor", "genesis")

	// Remove existing symlink if it exists
	if _, err := os.Lstat(currentLink); err == nil {
		if err := os.Remove(currentLink); err != nil {
			return fmt.Errorf("failed to remove existing symlink: %w", err)
		}
	}

	// Create relative symlink
	if err := os.Symlink("genesis", currentLink); err != nil {
		return fmt.Errorf("failed to create current symlink: %w", err)
	}

	c.logger.Info("wemixvisor initialized successfully",
		zap.String("home", home),
		zap.String("genesis", genesisDir),
		zap.String("config", configPath))

	// Print success message
	fmt.Printf("✅ Wemixvisor initialized successfully!\n\n")
	fmt.Printf("Directory structure created at: %s\n", home)
	fmt.Printf("├── wemixvisor/\n")
	fmt.Printf("│   ├── genesis/       # Initial binary location\n")
	fmt.Printf("│   │   └── bin/       # Place your wemixd binary here\n")
	fmt.Printf("│   ├── upgrades/      # Upgrade binaries\n")
	fmt.Printf("│   ├── backup/        # Backup storage\n")
	fmt.Printf("│   ├── current        # Symlink to active version\n")
	fmt.Printf("│   └── config.toml    # Configuration file\n")
	fmt.Printf("├── data/              # Node data directory\n")
	fmt.Printf("└── logs/              # Log files\n\n")

	fmt.Printf("Next steps:\n")
	fmt.Printf("1. Copy your wemixd binary to: %s\n", filepath.Join(home, "wemixvisor", "genesis", "bin", "wemixd"))
	fmt.Printf("2. Edit configuration: %s\n", configPath)
	fmt.Printf("3. Start the node: wemixvisor start\n")

	return nil
}

// createInitialConfig creates the initial configuration file
func (c *InitCommand) createInitialConfig(path string) error {
	content := `# Wemixvisor Configuration

# Network configuration
network = "mainnet"  # mainnet, testnet, or custom

# Process management
restart_on_failure = true
max_restarts = 5
shutdown_grace_period = "30s"

# Health monitoring
health_check_interval = "30s"
rpc_port = 8545

# Backup configuration
backup_enabled = true
backup_count = 3

# Logging
log_level = "info"
log_file = "logs/node.log"

# Environment variables for the node process
[environment]
# WEMIX_HOME = ""
# WEMIX_NETWORK = ""
`

	// Write config file
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}