package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wemix/wemixvisor/internal/config"
	"github.com/wemix/wemixvisor/pkg/logger"
)

// NewStartCommand creates the start command
func NewStartCommand(cfg *config.Config, logger *logger.Logger) *cobra.Command {
	var daemon bool

	cmd := &cobra.Command{
		Use:   "start [flags] [node-args]",
		Short: "Start the node",
		Long:  `Start the managed node process.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			logger.Info("Starting node...")

			// TODO: Implement actual start logic
			fmt.Printf("Starting node at %s\n", cfg.CurrentBin())
			fmt.Printf("Home: %s\n", cfg.Home)
			fmt.Printf("Args: %v\n", args)

			if daemon {
				fmt.Println("Running in daemon mode")
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&daemon, "daemon", false, "Run in background")

	return cmd
}

// NewRunCommand creates the run command
func NewRunCommand(cfg *config.Config, logger *logger.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run [node-args]",
		Short: "Start node in foreground",
		Long:  `Start the managed node process in foreground mode.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			logger.Info("Running node in foreground...")

			// TODO: Implement actual run logic
			fmt.Printf("Running node at %s\n", cfg.CurrentBin())
			fmt.Printf("Home: %s\n", cfg.Home)
			fmt.Printf("Args: %v\n", args)

			return nil
		},
	}

	return cmd
}