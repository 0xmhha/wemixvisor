package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"go.uber.org/zap"
	"github.com/wemix/wemixvisor/internal/config"
	"github.com/wemix/wemixvisor/internal/node"
	"github.com/wemix/wemixvisor/pkg/logger"
)

// CommandHandler handles CLI commands
type CommandHandler struct {
	config  *config.Config
	logger  *logger.Logger
	manager NodeManager
	parser  *Parser
}

// NewCommandHandler creates a new command handler
func NewCommandHandler(cfg *config.Config, logger *logger.Logger) *CommandHandler {
	return &CommandHandler{
		config:  cfg,
		logger:  logger,
		parser:  NewParser(),
		manager: node.NewManager(cfg, logger),
	}
}

// Execute executes a CLI command
func (h *CommandHandler) Execute(args []string) error {
	// Parse arguments
	parsed, err := h.parser.Parse(args)
	if err != nil {
		return fmt.Errorf("failed to parse arguments: %w", err)
	}

	// Apply wemixvisor options to config
	h.applyOptions(parsed.WemixvisorOpts)

	// Execute command
	switch parsed.Command {
	case "init":
		initCmd := NewInitCommand(h.config, h.logger)
		return initCmd.Execute(parsed.NodeArgs)
	case "start":
		return h.handleStart(parsed)
	case "stop":
		return h.handleStop()
	case "restart":
		return h.handleRestart(parsed)
	case "status":
		return h.handleStatus()
	case "logs":
		return h.handleLogs(parsed)
	case "version":
		return h.handleVersion()
	case "run":
		return h.handleRun(parsed)
	default:
		return fmt.Errorf("unknown command: %s", parsed.Command)
	}
}

// handleStart handles the start command
func (h *CommandHandler) handleStart(parsed *ParsedArgs) error {
	h.logger.Info("starting node", zap.Strings("args", parsed.NodeArgs))

	// Check if already running
	if h.manager.GetState() == node.StateRunning {
		return fmt.Errorf("node is already running")
	}

	// Start the node with parsed arguments
	if err := h.manager.Start(parsed.NodeArgs); err != nil {
		return fmt.Errorf("failed to start node: %w", err)
	}

	h.logger.Info("node started successfully", zap.Int("pid", h.manager.GetPID()))

	// If running in foreground mode, wait for signals
	if !h.config.Daemon {
		return h.waitForSignal()
	}

	return nil
}

// handleStop handles the stop command
func (h *CommandHandler) handleStop() error {
	h.logger.Info("stopping node")

	if h.manager.GetState() != node.StateRunning {
		return fmt.Errorf("node is not running")
	}

	if err := h.manager.Stop(); err != nil {
		return fmt.Errorf("failed to stop node: %w", err)
	}

	h.logger.Info("node stopped successfully")
	return nil
}

// handleRestart handles the restart command
func (h *CommandHandler) handleRestart(parsed *ParsedArgs) error {
	h.logger.Info("restarting node")

	// If new arguments provided, update them
	if len(parsed.NodeArgs) > 0 {
		// Store new arguments for restart
		h.manager.SetNodeArgs(parsed.NodeArgs)
	}

	if err := h.manager.Restart(); err != nil {
		return fmt.Errorf("failed to restart node: %w", err)
	}

	h.logger.Info("node restarted successfully", zap.Int("pid", h.manager.GetPID()))
	return nil
}

// handleStatus handles the status command
func (h *CommandHandler) handleStatus() error {
	status := h.manager.GetStatus()

	// Check output format
	if h.config.JSONOutput {
		// JSON output
		data, err := json.MarshalIndent(status, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal status: %w", err)
		}
		fmt.Println(string(data))
	} else {
		// Human-readable output
		fmt.Printf("Node Status: %s\n", status.StateString)
		if status.PID > 0 {
			fmt.Printf("PID: %d\n", status.PID)
			fmt.Printf("Uptime: %s\n", formatDuration(status.Uptime))
		}
		fmt.Printf("Network: %s\n", status.Network)
		if status.Version != "" {
			fmt.Printf("Version: %s\n", status.Version)
		}
		fmt.Printf("Restart Count: %d\n", status.RestartCount)
		if status.Binary != "" {
			fmt.Printf("Binary: %s\n", status.Binary)
		}

		// Display health status if available
		if status.Health != nil {
			fmt.Printf("\nHealth Status: ")
			if status.Health.Healthy {
				fmt.Printf("✅ Healthy\n")
			} else {
				fmt.Printf("❌ Unhealthy\n")
			}

			// Show individual health checks
			if len(status.Health.Checks) > 0 {
				fmt.Printf("Health Checks:\n")
				for name, check := range status.Health.Checks {
					if check.Healthy {
						fmt.Printf("  ✅ %s: OK\n", name)
					} else {
						fmt.Printf("  ❌ %s: %s\n", name, check.Error)
					}
				}
			}
		}
	}

	return nil
}

// handleLogs handles the logs command
func (h *CommandHandler) handleLogs(parsed *ParsedArgs) error {
	// Check for log options
	follow := false
	tail := 100 // Default tail lines

	for _, arg := range parsed.NodeArgs {
		if arg == "--follow" || arg == "-f" {
			follow = true
		}
		// Parse tail option (e.g., --tail=50)
		if strings.HasPrefix(arg, "--tail=") {
			if n, err := strconv.Atoi(strings.TrimPrefix(arg, "--tail=")); err == nil {
				tail = n
			}
		}
	}

	// Get log file path
	logFile := h.config.LogFile
	if logFile == "" {
		logFile = filepath.Join(h.config.Home, "logs", "node.log")
	}

	// Check if log file exists
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		return fmt.Errorf("log file not found: %s", logFile)
	}

	// Display logs
	if follow {
		return followLogs(logFile)
	} else {
		return tailLogs(logFile, tail)
	}
}

// handleVersion handles the version command
func (h *CommandHandler) handleVersion() error {
	// Wemixvisor version
	fmt.Printf("wemixvisor version: %s\n", Version)

	// Node version if available
	if h.manager.GetState() == node.StateRunning {
		nodeVersion := h.manager.GetVersion()
		if nodeVersion != "" && nodeVersion != "unknown" {
			fmt.Printf("node version: %s\n", nodeVersion)
		}
	}

	return nil
}

// handleRun handles the run command (starts and waits)
func (h *CommandHandler) handleRun(parsed *ParsedArgs) error {
	h.logger.Info("running node in foreground")

	// Start the node
	if err := h.manager.Start(parsed.NodeArgs); err != nil {
		return fmt.Errorf("failed to start node: %w", err)
	}

	h.logger.Info("node started successfully", zap.Int("pid", h.manager.GetPID()))

	// Wait for signals
	return h.waitForSignal()
}

// waitForSignal waits for interrupt signals
func (h *CommandHandler) waitForSignal() error {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	// Wait for signal
	sig := <-sigCh
	h.logger.Info("received signal", zap.String("signal", sig.String()))

	// Stop the node
	if h.manager.GetState() == node.StateRunning {
		if err := h.manager.Stop(); err != nil {
			return fmt.Errorf("failed to stop node: %w", err)
		}
	}

	return nil
}

// applyOptions applies wemixvisor options to config
func (h *CommandHandler) applyOptions(opts map[string]string) {
	for key, value := range opts {
		switch key {
		case "--home":
			h.config.Home = value
		case "--name":
			h.config.Name = value
		case "--network":
			h.config.Network = value
		case "--debug":
			h.config.Debug = value == "true"
		case "--json":
			h.config.JSONOutput = value == "true"
		case "--quiet":
			h.config.Quiet = value == "true"
		}
	}
}

// formatDuration formats a duration into a human-readable string
func formatDuration(d time.Duration) string {
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm %ds", days, hours, minutes, seconds)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	}
	if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}