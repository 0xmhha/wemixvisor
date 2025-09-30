package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/wemix/wemixvisor/internal/config"
	"github.com/wemix/wemixvisor/pkg/logger"
)

// NewInitCommand creates the init command
func NewInitCommand(cfg *config.Config, logger *logger.Logger) *cobra.Command {
	var (
		genesis  string
		template string
	)

	cmd := &cobra.Command{
		Use:   "init [binary-path]",
		Short: "Initialize wemixvisor",
		Long:  `Initialize wemixvisor with the genesis binary.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			binaryPath := args[0]

			logger.Info("Initializing wemixvisor...")

			// Create directory structure
			dirs := []string{
				cfg.WemixvisorDir(),
				cfg.GenesisDir(),
				filepath.Join(cfg.GenesisDir(), "bin"),
				cfg.UpgradesDir(),
				filepath.Join(cfg.Home, "data"),
			}

			for _, dir := range dirs {
				if err := os.MkdirAll(dir, 0755); err != nil {
					return fmt.Errorf("failed to create directory %s: %w", dir, err)
				}
			}

			// Copy binary to genesis/bin
			targetPath := cfg.GenesisBin()
			if err := copyBinary(binaryPath, targetPath); err != nil {
				return fmt.Errorf("failed to copy binary: %w", err)
			}

			// Create symlink to genesis
			if err := cfg.SymLinkToGenesis(); err != nil {
				return fmt.Errorf("failed to create symlink: %w", err)
			}

			// Apply template if specified
			if template != "" {
				logger.Info("Applying template: " + template)
				// TODO: Apply template configuration
			}

			fmt.Println("Wemixvisor initialized successfully!")
			fmt.Printf("Home: %s\n", cfg.Home)
			fmt.Printf("Genesis binary: %s\n", targetPath)
			fmt.Printf("Current link: %s\n", cfg.CurrentDir())

			return nil
		},
	}

	cmd.Flags().StringVar(&genesis, "genesis", "", "Path to genesis file")
	cmd.Flags().StringVar(&template, "template", "", "Configuration template to apply")

	return cmd
}

// copyBinary copies a binary file from source to destination
func copyBinary(src, dst string) error {
	// Read source file
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("failed to read source binary: %w", err)
	}

	// Write to destination with execute permissions
	if err := os.WriteFile(dst, data, 0755); err != nil {
		return fmt.Errorf("failed to write destination binary: %w", err)
	}

	return nil
}