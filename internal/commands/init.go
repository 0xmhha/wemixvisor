package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/wemix/wemixvisor/internal/config"
)

// InitCmd creates the init command
func InitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init [genesis-binary]",
		Short: "Initialize wemixvisor with genesis binary",
		Long: `Initialize sets up the directory structure for wemixvisor
and copies the genesis binary to the appropriate location.

Example:
  wemixvisor init /path/to/wemixd`,
		Args: cobra.ExactArgs(1),
		RunE: initExecute,
	}

	cmd.Flags().String("daemon-home", "", "Home directory for the daemon")
	cmd.Flags().String("daemon-name", "", "Name of the daemon binary")

	return cmd
}

func initExecute(cmd *cobra.Command, args []string) error {
	// Get configuration
	cfg := config.DefaultConfig()

	// Override with flags
	if home := cmd.Flag("daemon-home").Value.String(); home != "" {
		cfg.Home = home
	} else if envHome := os.Getenv("DAEMON_HOME"); envHome != "" {
		cfg.Home = envHome
	}

	if name := cmd.Flag("daemon-name").Value.String(); name != "" {
		cfg.Name = name
	} else if envName := os.Getenv("DAEMON_NAME"); envName != "" {
		cfg.Name = envName
	}

	genesisBinary := args[0]

	// Verify genesis binary exists
	if _, err := os.Stat(genesisBinary); err != nil {
		return fmt.Errorf("genesis binary not found: %w", err)
	}

	// Create directory structure
	dirs := []string{
		cfg.WemixvisorDir(),
		cfg.GenesisDir(),
		filepath.Join(cfg.GenesisDir(), "bin"),
		cfg.UpgradesDir(),
		cfg.DataBackupPath,
		filepath.Join(cfg.Home, "data"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	fmt.Printf("Created wemixvisor directories at %s\n", cfg.WemixvisorDir())

	// Copy genesis binary
	targetPath := cfg.GenesisBin()
	if err := copyFile(genesisBinary, targetPath); err != nil {
		return fmt.Errorf("failed to copy genesis binary: %w", err)
	}

	// Make binary executable
	if err := os.Chmod(targetPath, 0755); err != nil {
		return fmt.Errorf("failed to set executable permission: %w", err)
	}

	fmt.Printf("Copied genesis binary to %s\n", targetPath)

	// Create symlink to genesis
	if err := cfg.SymLinkToGenesis(); err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}

	fmt.Printf("Created symlink: current -> genesis\n")
	fmt.Printf("\nInitialization complete!\n")
	fmt.Printf("You can now run: wemixvisor run %s\n", cfg.Name)

	return nil
}

func copyFile(src, dst string) error {
	// Read source file
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	// Write to destination
	return os.WriteFile(dst, data, 0755)
}