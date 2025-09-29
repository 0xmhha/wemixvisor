package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/wemix/wemixvisor/internal/config"
	"github.com/wemix/wemixvisor/internal/process"
	"github.com/wemix/wemixvisor/pkg/logger"
	"go.uber.org/zap"
)

// RunCmd creates the run command
func RunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run [command]",
		Short: "Run the blockchain node under wemixvisor management",
		Long: `Run starts the blockchain node process and monitors for upgrades.
The command and its arguments should be provided after 'run'.

Example:
  wemixvisor run start --home /path/to/node`,
		Args: cobra.MinimumNArgs(1),
		RunE: runExecute,
	}

	// Add flags
	cmd.Flags().String("daemon-home", "", "Home directory for the daemon")
	cmd.Flags().String("daemon-name", "", "Name of the daemon binary")
	cmd.Flags().Bool("allow-download-binaries", false, "Allow automatic binary downloads")
	cmd.Flags().Bool("restart-after-upgrade", true, "Restart after upgrade")
	cmd.Flags().Duration("shutdown-grace", 0, "Grace period for shutdown")
	cmd.Flags().Duration("poll-interval", 0, "Polling interval for upgrade checks")
	cmd.Flags().String("rpc-address", "", "RPC address for WBFT node")
	cmd.Flags().Bool("unsafe-skip-backup", false, "Skip backup before upgrade")

	return cmd
}

func runExecute(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := loadConfig(cmd)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Set the command arguments
	cfg.Args = args

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Initialize logger
	log, err := logger.New(cfg.ColorLogs, cfg.DisableLogs, cfg.TimeFormatLogs)
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}

	log.Info("starting wemixvisor",
		zap.String("home", cfg.Home),
		zap.String("name", cfg.Name),
		zap.Strings("args", cfg.Args))

	// Create and run process manager
	manager := process.NewManager(cfg, log)

	ctx := context.Background()
	if err := manager.Run(ctx); err != nil {
		log.Error("process manager failed", zap.Error(err))
		return err
	}

	return nil
}

func loadConfig(cmd *cobra.Command) (*config.Config, error) {
	cfg := config.DefaultConfig()

	// Set config from environment variables
	viper.SetEnvPrefix("DAEMON")
	viper.AutomaticEnv()

	// Override with command line flags
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

	// Set boolean flags
	if cmd.Flags().Changed("allow-download-binaries") {
		cfg.AllowDownloadBinaries, _ = cmd.Flags().GetBool("allow-download-binaries")
	} else if env := os.Getenv("DAEMON_ALLOW_DOWNLOAD_BINARIES"); env != "" {
		cfg.AllowDownloadBinaries = env == "true"
	}

	if cmd.Flags().Changed("restart-after-upgrade") {
		cfg.RestartAfterUpgrade, _ = cmd.Flags().GetBool("restart-after-upgrade")
	} else if env := os.Getenv("DAEMON_RESTART_AFTER_UPGRADE"); env != "" {
		cfg.RestartAfterUpgrade = env != "false"
	}

	if cmd.Flags().Changed("unsafe-skip-backup") {
		cfg.UnsafeSkipBackup, _ = cmd.Flags().GetBool("unsafe-skip-backup")
	} else if env := os.Getenv("UNSAFE_SKIP_BACKUP"); env != "" {
		cfg.UnsafeSkipBackup = env == "true"
	}

	// Set duration flags
	if cmd.Flags().Changed("shutdown-grace") {
		cfg.ShutdownGrace, _ = cmd.Flags().GetDuration("shutdown-grace")
	}

	if cmd.Flags().Changed("poll-interval") {
		cfg.PollInterval, _ = cmd.Flags().GetDuration("poll-interval")
	}

	// Set string flags
	if rpc := cmd.Flag("rpc-address").Value.String(); rpc != "" {
		cfg.RPCAddress = rpc
	} else if envRPC := os.Getenv("DAEMON_RPC_ADDRESS"); envRPC != "" {
		cfg.RPCAddress = envRPC
	}

	// Check for cosmovisor environment variables
	if env := os.Getenv("COSMOVISOR_DISABLE_LOGS"); env == "true" {
		cfg.DisableLogs = true
	}
	if env := os.Getenv("COSMOVISOR_COLOR_LOGS"); env == "false" {
		cfg.ColorLogs = false
	}
	if env := os.Getenv("COSMOVISOR_TIMEFORMAT_LOGS"); env != "" {
		cfg.TimeFormatLogs = env
	}

	return cfg, nil
}