package cli

import (
	"github.com/spf13/cobra"
	"github.com/wemix/wemixvisor/internal/config"
	"github.com/wemix/wemixvisor/pkg/logger"
)

// NewRootCommand creates the root command for wemixvisor
func NewRootCommand(cfg *config.Config, logger *logger.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "wemixvisor",
		Short: "WBFT Node Lifecycle Manager",
		Long: `Wemixvisor is a process manager for WBFT-based blockchain node upgrades.
It manages the lifecycle of the node binary, handling upgrades seamlessly.`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// Add global flags
	cmd.PersistentFlags().StringVar(&cfg.Home, "home", cfg.Home, "Home directory for wemixvisor")
	cmd.PersistentFlags().StringVar(&cfg.Name, "name", cfg.Name, "Name of the binary to manage")
	cmd.PersistentFlags().BoolVar(&cfg.RestartAfterUpgrade, "restart-after-upgrade", cfg.RestartAfterUpgrade, "Restart after upgrade")
	cmd.PersistentFlags().BoolVar(&cfg.AllowDownloadBinaries, "allow-download-binaries", cfg.AllowDownloadBinaries, "Allow automatic binary downloads")
	cmd.PersistentFlags().BoolVar(&cfg.UnsafeSkipBackup, "unsafe-skip-backup", cfg.UnsafeSkipBackup, "Skip backup during upgrade")

	// Add subcommands
	cmd.AddCommand(NewStartCommand(cfg, logger))
	cmd.AddCommand(NewRunCommand(cfg, logger))
	cmd.AddCommand(NewVersionCommand())
	cmd.AddCommand(NewInitCommand(cfg, logger))
	cmd.AddCommand(NewConfigCommand())
	cmd.AddCommand(NewBackupCommand(cfg, logger))
	cmd.AddCommand(NewStatusCommand(cfg, logger))
	cmd.AddCommand(NewStopCommand(cfg, logger))
	cmd.AddCommand(NewRestartCommand(cfg, logger))

	// Phase 7: Advanced monitoring and management commands
	cmd.AddCommand(NewAPICommand(cfg, logger))
	cmd.AddCommand(NewMetricsCommand(cfg, logger))
	cmd.AddCommand(NewProfileCommand(cfg, logger))

	// Phase 8: Upgrade automation commands
	cmd.AddCommand(NewUpgradeCommand(cfg, logger))

	return cmd
}