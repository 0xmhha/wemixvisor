//go:build ignore
// +build ignore

// Package main demonstrates custom notification handler implementation
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"go.uber.org/zap"
	"github.com/wemix/wemixvisor/internal/config"
	"github.com/wemix/wemixvisor/internal/governance"
	"github.com/wemix/wemixvisor/pkg/logger"
)

// WebhookHandler sends notifications to a webhook endpoint
type WebhookHandler struct {
	url     string
	enabled bool
	client  *http.Client
}

// NewWebhookHandler creates a new webhook handler
func NewWebhookHandler(url string) *WebhookHandler {
	return &WebhookHandler{
		url:     url,
		enabled: true,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Handle processes a notification
func (h *WebhookHandler) Handle(notification *governance.Notification) error {
	// Create webhook payload
	payload := map[string]interface{}{
		"event":     string(notification.Event),
		"title":     notification.Title,
		"message":   notification.Message,
		"timestamp": notification.Timestamp.Format(time.RFC3339),
		"priority":  notification.Priority,
	}

	// Add proposal data if available
	if proposal, ok := notification.Data.(*governance.Proposal); ok {
		payload["proposal"] = map[string]interface{}{
			"id":     proposal.ID,
			"title":  proposal.Title,
			"type":   string(proposal.Type),
			"status": string(proposal.Status),
		}
	}

	// Marshal payload
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Send webhook
	resp, err := h.client.Post(h.url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}

// IsEnabled returns whether the handler is enabled
func (h *WebhookHandler) IsEnabled() bool {
	return h.enabled
}

// GetType returns the handler type
func (h *WebhookHandler) GetType() string {
	return "webhook"
}

// ConsoleHandler prints notifications to console with formatting
type ConsoleHandler struct {
	enabled bool
}

// NewConsoleHandler creates a new console handler
func NewConsoleHandler() *ConsoleHandler {
	return &ConsoleHandler{enabled: true}
}

// Handle prints the notification
func (h *ConsoleHandler) Handle(notification *governance.Notification) error {
	// Color codes for terminal
	const (
		colorReset  = "\033[0m"
		colorRed    = "\033[31m"
		colorGreen  = "\033[32m"
		colorYellow = "\033[33m"
		colorBlue   = "\033[34m"
		colorPurple = "\033[35m"
		colorCyan   = "\033[36m"
	)

	// Select color based on event
	var color string
	switch notification.Event {
	case governance.EventProposalPassed:
		color = colorGreen
	case governance.EventProposalRejected:
		color = colorRed
	case governance.EventVotingStarted:
		color = colorBlue
	case governance.EventUpgradeScheduled:
		color = colorPurple
	case governance.EventUpgradeTriggered:
		color = colorYellow
	default:
		color = colorCyan
	}

	// Print formatted notification
	fmt.Printf("\n%s========== GOVERNANCE NOTIFICATION ==========%s\n", color, colorReset)
	fmt.Printf("Time:     %s\n", notification.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Printf("Event:    %s%s%s\n", color, notification.Event, colorReset)
	fmt.Printf("Title:    %s\n", notification.Title)
	fmt.Printf("Message:  %s\n", notification.Message)

	// Print proposal details if available
	if proposal, ok := notification.Data.(*governance.Proposal); ok {
		fmt.Printf("\nProposal Details:\n")
		fmt.Printf("  ID:     %s\n", proposal.ID)
		fmt.Printf("  Type:   %s\n", proposal.Type)
		fmt.Printf("  Status: %s\n", proposal.Status)

		if proposal.VotingStats != nil {
			fmt.Printf("  Voting:\n")
			fmt.Printf("    Yes:     %d\n", proposal.VotingStats.YesVotes)
			fmt.Printf("    No:      %d\n", proposal.VotingStats.NoVotes)
			fmt.Printf("    Abstain: %d\n", proposal.VotingStats.AbstainVotes)
			fmt.Printf("    Turnout: %.2f%%\n", proposal.VotingStats.Turnout*100)
		}
	}

	fmt.Printf("%s=============================================%s\n", color, colorReset)
	return nil
}

// IsEnabled returns whether the handler is enabled
func (h *ConsoleHandler) IsEnabled() bool {
	return h.enabled
}

// GetType returns the handler type
func (h *ConsoleHandler) GetType() string {
	return "console"
}

// FileHandler writes notifications to a file
type FileHandler struct {
	filepath string
	enabled  bool
}

// NewFileHandler creates a new file handler
func NewFileHandler(filepath string) *FileHandler {
	return &FileHandler{
		filepath: filepath,
		enabled:  true,
	}
}

// Handle writes the notification to file
func (h *FileHandler) Handle(notification *governance.Notification) error {
	// Open file in append mode
	file, err := os.OpenFile(h.filepath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Create log entry
	entry := map[string]interface{}{
		"timestamp": notification.Timestamp.Format(time.RFC3339),
		"event":     string(notification.Event),
		"title":     notification.Title,
		"message":   notification.Message,
		"priority":  notification.Priority,
	}

	// Write as JSON line
	jsonData, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal entry: %w", err)
	}

	if _, err := file.Write(append(jsonData, '\n')); err != nil {
		return fmt.Errorf("failed to write to file: %w", err)
	}

	return nil
}

// IsEnabled returns whether the handler is enabled
func (h *FileHandler) IsEnabled() bool {
	return h.enabled
}

// GetType returns the handler type
func (h *FileHandler) GetType() string {
	return "file"
}

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

	// Create monitor
	monitor := governance.NewMonitor(cfg, logger)

	// Create notifier with custom handlers
	notifier := governance.NewNotifier(logger)

	// Add console handler
	notifier.AddHandler(NewConsoleHandler())

	// Add file handler
	logFile := getEnvOrDefault("GOVERNANCE_LOG_FILE", "/tmp/governance.log")
	notifier.AddHandler(NewFileHandler(logFile))

	// Add webhook handler if URL is provided
	if webhookURL := os.Getenv("GOVERNANCE_WEBHOOK_URL"); webhookURL != "" {
		notifier.AddHandler(NewWebhookHandler(webhookURL))
		logger.Info("Webhook handler enabled", zap.String("url", webhookURL))
	}

	// Set notifier on monitor
	// Note: In real implementation, this would be done internally
	// This is just for demonstration

	// Start notifier
	if err := notifier.Start(); err != nil {
		log.Fatalf("Failed to start notifier: %v", err)
	}
	defer notifier.Stop()

	// Start monitor
	if err := monitor.Start(); err != nil {
		log.Fatalf("Failed to start monitor: %v", err)
	}
	defer monitor.Stop()

	logger.Info("Custom handlers configured and running")
	logger.Info("Console output: enabled")
	logger.Info("File output", zap.String("path", logFile))

	// Keep running
	select {}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
