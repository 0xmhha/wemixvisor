package main

import (
	"fmt"
	"os"

	"github.com/wemix/wemixvisor/internal/cli"
	"github.com/wemix/wemixvisor/internal/config"
	"github.com/wemix/wemixvisor/pkg/logger"
)

func main() {
	// Parse command-line arguments
	args := os.Args[1:]

	// Handle help flag
	if len(args) > 0 && (args[0] == "--help" || args[0] == "-h" || args[0] == "help") {
		printHelp()
		os.Exit(0)
	}

	// Load configuration
	cfg := config.DefaultConfig()

	// Apply environment variables
	applyEnvVars(cfg)

	// Initialize logger
	log, err := logger.New(
		false, // debug mode
		true,  // color logs
		"",    // log file
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	// Create CLI root command
	root := cli.NewRootCommand(cfg, log)

	// Execute command
	if err := root.Execute(); err != nil {
		log.Error("Command execution failed")
		os.Exit(1)
	}
}

// applyEnvVars applies environment variables to configuration
func applyEnvVars(cfg *config.Config) {
	// Core environment variables
	if val := os.Getenv("DAEMON_HOME"); val != "" {
		cfg.Home = val
	}
	if val := os.Getenv("DAEMON_NAME"); val != "" {
		cfg.Name = val
	}

	// Upgrade settings
	if val := os.Getenv("DAEMON_ALLOW_DOWNLOAD_BINARIES"); val == "true" {
		cfg.AllowDownloadBinaries = true
	}
	if val := os.Getenv("DAEMON_RESTART_AFTER_UPGRADE"); val == "false" {
		cfg.RestartAfterUpgrade = false
	}

	// Process management
	if val := os.Getenv("UNSAFE_SKIP_BACKUP"); val == "true" {
		cfg.UnsafeSkipBackup = true
	}

	// Network settings
	if val := os.Getenv("DAEMON_NETWORK"); val != "" {
		// Network can be used to set network-specific configuration
		// but it's not a direct field in Config
	}

	// Logging settings are handled separately
	// They are not part of the Config struct
}

// printHelp prints help information
func printHelp() {
	fmt.Println(`wemixvisor - WBFT Node Lifecycle Manager

Usage:
  wemixvisor <command> [flags] [node-args]

Core Commands:
  start     Start the node
  stop      Stop the node
  restart   Restart the node
  status    Show node status
  logs      Display node logs
  version   Show version information
  run       Start node in foreground
  init      Initialize wemixvisor home directory
  config    Manage configuration
  backup    Manage backups

Phase 7 - Advanced Monitoring Commands:
  api       Start API server with WebSocket support
  metrics   Collect and view system metrics
  profile   Performance profiling tools

Wemixvisor Flags:
  --home <path>      Set the home directory (default: ~/.wemixd)
  --network <name>   Set the network (mainnet/testnet)
  --debug            Enable debug mode
  --json             Output in JSON format
  --quiet            Suppress output
  --daemon           Run in background (for start command)

Node Arguments:
  All other arguments are passed through to the node binary (geth compatible).

Examples:
  # Start node with default settings
  wemixvisor start

  # Start node with custom datadir (passed to geth)
  wemixvisor start --datadir /custom/data --syncmode full

  # Start with wemixvisor options and geth options
  wemixvisor start --home /custom/home --debug --datadir /data --port 30303

  # Check status
  wemixvisor status

  # View logs
  wemixvisor logs --follow

  # Start API server with monitoring
  wemixvisor api --port 8080 --enable-metrics

  # View current metrics
  wemixvisor metrics show

  # Capture CPU profile
  wemixvisor profile cpu --duration 30

Environment Variables:
  DAEMON_HOME                      Home directory for wemixvisor
  DAEMON_NAME                      Name of the binary to manage
  DAEMON_NETWORK                   Network to connect to
  DAEMON_DEBUG                     Enable debug mode
  DAEMON_ALLOW_DOWNLOAD_BINARIES   Allow automatic binary downloads
  DAEMON_RESTART_AFTER_UPGRADE     Restart after upgrade
  UNSAFE_SKIP_BACKUP               Skip backup during upgrade
  COSMOVISOR_DISABLE_LOGS          Disable logging
  COSMOVISOR_COLOR_LOGS            Colorize log output

For more information, see the documentation at:
  https://github.com/wemix/wemixvisor`)
}