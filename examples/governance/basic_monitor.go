// Package main demonstrates basic governance monitoring setup
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
	"github.com/wemix/wemixvisor/internal/config"
	"github.com/wemix/wemixvisor/internal/governance"
	"github.com/wemix/wemixvisor/pkg/logger"
)

func main() {
	// Create logger
	logger, err := logger.New(true, false, "iso8601")
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}

	// Create configuration
	cfg := &config.Config{
		Home:       os.Getenv("WEMIXVISOR_HOME"),
		RPCAddress: getEnvOrDefault("WEMIXVISOR_RPC", "http://localhost:8545"),
	}

	// Create and start monitor
	monitor := governance.NewMonitor(cfg, logger)

	// Start monitoring
	if err := monitor.Start(); err != nil {
		log.Fatalf("Failed to start monitor: %v", err)
	}
	defer monitor.Stop()

	logger.Info("Governance monitor started successfully")

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Create ticker for periodic status checks
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Main monitoring loop
	for {
		select {
		case <-ctx.Done():
			logger.Info("Shutting down governance monitor")
			return

		case <-sigChan:
			logger.Info("Received shutdown signal")
			cancel()

		case <-ticker.C:
			// Print current status
			printStatus(monitor, logger)
		}
	}
}

func printStatus(monitor *governance.Monitor, logger *logger.Logger) {
	// Get all proposals
	proposals, err := monitor.GetProposals()
	if err != nil {
		logger.Error("Failed to get proposals", zap.Error(err))
		return
	}

	fmt.Println("\n========== Governance Status ==========")
	fmt.Printf("Total Proposals: %d\n", len(proposals))

	// Count by status
	statusCount := make(map[governance.ProposalStatus]int)
	for _, p := range proposals {
		statusCount[p.Status]++
	}

	fmt.Println("\nProposals by Status:")
	for status, count := range statusCount {
		fmt.Printf("  %s: %d\n", status, count)
	}

	// Show active proposals
	fmt.Println("\nActive Proposals:")
	for _, p := range proposals {
		if p.Status == governance.ProposalStatusVoting {
			fmt.Printf("  [%s] %s\n", p.ID, p.Title)
			if p.VotingStats != nil {
				fmt.Printf("    Turnout: %.2f%%, Quorum: %v\n",
					p.VotingStats.Turnout*100,
					p.VotingStats.QuorumReached)
			}
			fmt.Printf("    Voting ends: %s\n", p.VotingEndTime.Format(time.RFC3339))
		}
	}

	// Show scheduled upgrades
	upgrades, err := monitor.GetUpgradeQueue()
	if err != nil {
		logger.Error("Failed to get upgrades", zap.Error(err))
		return
	}

	if len(upgrades) > 0 {
		fmt.Println("\nScheduled Upgrades:")
		for _, u := range upgrades {
			fmt.Printf("  %s at height %d (Status: %s)\n",
				u.Name, u.Height, u.Status)
		}
	}

	fmt.Println("=======================================")
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}