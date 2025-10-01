package alerting

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/wemix/wemixvisor/internal/metrics"
	"github.com/wemix/wemixvisor/pkg/logger"
)

// AlertManager manages alerts and notifications
type AlertManager struct {
	logger    *logger.Logger
	collector *metrics.Collector
	rules     []*AlertRule
	channels  map[string]NotificationChannel
	alerts    map[string]*metrics.Alert
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
	config    *AlertConfig
}

// AlertConfig represents alerting configuration
type AlertConfig struct {
	Enabled         bool          `json:"enabled"`
	EvaluationInterval time.Duration `json:"evaluation_interval"`
	AlertRetention  time.Duration `json:"alert_retention"`
	Channels        []ChannelConfig `json:"channels"`
}

// ChannelConfig represents notification channel configuration
type ChannelConfig struct {
	Type    string                 `json:"type"`
	Name    string                 `json:"name"`
	Enabled bool                   `json:"enabled"`
	Config  map[string]interface{} `json:"config"`
}

// AlertRule represents a rule for generating alerts
type AlertRule struct {
	Name        string              `json:"name"`
	Metric      string              `json:"metric"`
	Condition   string              `json:"condition"`
	Threshold   float64             `json:"threshold"`
	Duration    time.Duration       `json:"duration"`
	Level       metrics.AlertLevel  `json:"level"`
	Message     string              `json:"message"`
	Description string              `json:"description"`
	Channels    []string            `json:"channels"`
	Labels      map[string]string   `json:"labels"`
	Enabled     bool                `json:"enabled"`
	LastEval    *time.Time          `json:"last_eval,omitempty"`
	LastFired   *time.Time          `json:"last_fired,omitempty"`
}

// NotificationChannel represents a channel for sending notifications
type NotificationChannel interface {
	Send(alert *metrics.Alert) error
	GetType() string
	GetName() string
	IsEnabled() bool
}

// NewAlertManager creates a new alert manager
func NewAlertManager(config *AlertConfig, collector *metrics.Collector, logger *logger.Logger) *AlertManager {
	return &AlertManager{
		logger:    logger,
		collector: collector,
		rules:     make([]*AlertRule, 0),
		channels:  make(map[string]NotificationChannel),
		alerts:    make(map[string]*metrics.Alert),
		config:    config,
	}
}

// Start starts the alert manager
func (am *AlertManager) Start() error {
	if !am.config.Enabled {
		am.logger.Info("Alert manager is disabled")
		return nil
	}

	am.ctx, am.cancel = context.WithCancel(context.Background())

	// Initialize notification channels
	if err := am.initChannels(); err != nil {
		return fmt.Errorf("failed to initialize notification channels: %w", err)
	}

	// Start alert evaluation loop
	go am.evaluationLoop()

	am.logger.Info("Alert manager started",
		"evaluation_interval", am.config.EvaluationInterval,
		"channels", len(am.channels),
		"rules", len(am.rules))

	return nil
}

// Stop stops the alert manager
func (am *AlertManager) Stop() error {
	if am.cancel != nil {
		am.cancel()
	}

	am.logger.Info("Alert manager stopped")
	return nil
}

// initChannels initializes notification channels
func (am *AlertManager) initChannels() error {
	for _, cfg := range am.config.Channels {
		if !cfg.Enabled {
			continue
		}

		channel, err := am.createChannel(cfg)
		if err != nil {
			am.logger.Error("Failed to create notification channel",
				"type", cfg.Type,
				"name", cfg.Name,
				"error", err.Error())
			continue
		}

		am.channels[cfg.Name] = channel
		am.logger.Info("Notification channel initialized",
			"type", cfg.Type,
			"name", cfg.Name)
	}

	return nil
}

// createChannel creates a notification channel based on configuration
func (am *AlertManager) createChannel(cfg ChannelConfig) (NotificationChannel, error) {
	switch cfg.Type {
	case "email":
		return NewEmailChannel(cfg.Name, cfg.Config, am.logger)
	case "slack":
		return NewSlackChannel(cfg.Name, cfg.Config, am.logger)
	case "webhook":
		return NewWebhookChannel(cfg.Name, cfg.Config, am.logger)
	case "discord":
		return NewDiscordChannel(cfg.Name, cfg.Config, am.logger)
	default:
		return nil, fmt.Errorf("unknown channel type: %s", cfg.Type)
	}
}

// AddRule adds an alert rule
func (am *AlertManager) AddRule(rule *AlertRule) {
	am.mu.Lock()
	defer am.mu.Unlock()

	am.rules = append(am.rules, rule)
	am.logger.Info("Alert rule added",
		"name", rule.Name,
		"metric", rule.Metric,
		"level", rule.Level)
}

// RemoveRule removes an alert rule
func (am *AlertManager) RemoveRule(name string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	for i, rule := range am.rules {
		if rule.Name == name {
			am.rules = append(am.rules[:i], am.rules[i+1:]...)
			am.logger.Info("Alert rule removed", "name", name)
			return nil
		}
	}

	return fmt.Errorf("rule not found: %s", name)
}

// evaluationLoop runs the periodic alert evaluation
func (am *AlertManager) evaluationLoop() {
	ticker := time.NewTicker(am.config.EvaluationInterval)
	defer ticker.Stop()

	for {
		select {
		case <-am.ctx.Done():
			return
		case <-ticker.C:
			am.evaluateRules()
		}
	}
}

// evaluateRules evaluates all alert rules
func (am *AlertManager) evaluateRules() {
	am.mu.RLock()
	rules := make([]*AlertRule, len(am.rules))
	copy(rules, am.rules)
	am.mu.RUnlock()

	snapshot := am.collector.GetSnapshot()
	if snapshot == nil {
		return
	}

	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}

		am.evaluateRule(rule, snapshot)
	}

	// Clean up old alerts
	am.cleanupAlerts()
}

// evaluateRule evaluates a single alert rule
func (am *AlertManager) evaluateRule(rule *AlertRule, snapshot *metrics.MetricsSnapshot) {
	now := time.Now()
	rule.LastEval = &now

	value := am.getMetricValue(rule.Metric, snapshot)
	shouldAlert := am.checkCondition(rule.Condition, value, rule.Threshold)

	alertID := fmt.Sprintf("%s_%d", rule.Name, now.Unix())

	if shouldAlert {
		// Check if alert already exists
		if _, exists := am.alerts[rule.Name]; exists {
			return // Alert already active
		}

		// Create new alert
		alert := &metrics.Alert{
			ID:          alertID,
			Name:        rule.Name,
			Level:       rule.Level,
			Message:     rule.Message,
			Description: rule.Description,
			Source:      "alert_manager",
			Metric:      rule.Metric,
			Value:       value,
			Threshold:   rule.Threshold,
			Labels:      rule.Labels,
			Timestamp:   now,
		}

		// Store alert
		am.mu.Lock()
		am.alerts[rule.Name] = alert
		am.mu.Unlock()

		// Update last fired time
		rule.LastFired = &now

		// Send notifications
		am.sendNotifications(alert, rule.Channels)

		am.logger.Warn("Alert triggered",
			"name", rule.Name,
			"level", rule.Level,
			"value", value,
			"threshold", rule.Threshold)
	} else {
		// Check if alert should be resolved
		am.mu.Lock()
		if existingAlert, exists := am.alerts[rule.Name]; exists {
			existingAlert.ResolvedAt = &now
			delete(am.alerts, rule.Name)
			am.mu.Unlock()

			// Send resolution notification
			am.sendResolutionNotification(existingAlert, rule.Channels)

			am.logger.Info("Alert resolved",
				"name", rule.Name,
				"duration", now.Sub(existingAlert.Timestamp).String())
		} else {
			am.mu.Unlock()
		}
	}
}

// getMetricValue gets the value of a metric from the snapshot
func (am *AlertManager) getMetricValue(metric string, snapshot *metrics.MetricsSnapshot) float64 {
	// Simplified metric value extraction
	// In production, this would use proper metric lookup
	switch metric {
	case "cpu_usage":
		if snapshot.System != nil {
			return snapshot.System.CPUUsage
		}
	case "memory_usage":
		if snapshot.System != nil {
			return snapshot.System.MemoryUsage
		}
	case "disk_usage":
		if snapshot.System != nil {
			return snapshot.System.DiskUsage
		}
	case "node_height":
		if snapshot.Application != nil {
			return float64(snapshot.Application.NodeHeight)
		}
	case "proposals_voting":
		if snapshot.Governance != nil {
			return float64(snapshot.Governance.ProposalVoting)
		}
	}
	return 0
}

// checkCondition checks if a condition is met
func (am *AlertManager) checkCondition(condition string, value, threshold float64) bool {
	switch condition {
	case ">", "gt":
		return value > threshold
	case ">=", "gte":
		return value >= threshold
	case "<", "lt":
		return value < threshold
	case "<=", "lte":
		return value <= threshold
	case "==", "eq":
		return value == threshold
	case "!=", "ne":
		return value != threshold
	default:
		return false
	}
}

// sendNotifications sends notifications to specified channels
func (am *AlertManager) sendNotifications(alert *metrics.Alert, channelNames []string) {
	for _, channelName := range channelNames {
		channel, exists := am.channels[channelName]
		if !exists {
			am.logger.Warn("Notification channel not found", "channel", channelName)
			continue
		}

		if !channel.IsEnabled() {
			continue
		}

		go func(ch NotificationChannel, a *metrics.Alert) {
			if err := ch.Send(a); err != nil {
				am.logger.Error("Failed to send notification",
					"channel", ch.GetName(),
					"alert", a.Name,
					"error", err.Error())
			} else {
				am.logger.Info("Notification sent",
					"channel", ch.GetName(),
					"alert", a.Name)
			}
		}(channel, alert)
	}
}

// sendResolutionNotification sends a resolution notification
func (am *AlertManager) sendResolutionNotification(alert *metrics.Alert, channelNames []string) {
	// Modify alert message for resolution
	resolvedAlert := *alert
	resolvedAlert.Message = fmt.Sprintf("[RESOLVED] %s", alert.Message)

	am.sendNotifications(&resolvedAlert, channelNames)
}

// cleanupAlerts removes old alerts
func (am *AlertManager) cleanupAlerts() {
	am.mu.Lock()
	defer am.mu.Unlock()

	now := time.Now()
	retention := am.config.AlertRetention

	for id, alert := range am.alerts {
		if alert.ResolvedAt != nil && now.Sub(*alert.ResolvedAt) > retention {
			delete(am.alerts, id)
		}
	}
}

// GetActiveAlerts returns all active alerts
func (am *AlertManager) GetActiveAlerts() []*metrics.Alert {
	am.mu.RLock()
	defer am.mu.RUnlock()

	alerts := make([]*metrics.Alert, 0, len(am.alerts))
	for _, alert := range am.alerts {
		if alert.ResolvedAt == nil {
			alerts = append(alerts, alert)
		}
	}

	return alerts
}

// GetAlertHistory returns alert history
func (am *AlertManager) GetAlertHistory(limit int) []*metrics.Alert {
	am.mu.RLock()
	defer am.mu.RUnlock()

	alerts := make([]*metrics.Alert, 0, len(am.alerts))
	for _, alert := range am.alerts {
		alerts = append(alerts, alert)
	}

	// Sort by timestamp (newest first)
	// Simplified for demo - would use proper sorting in production

	if limit > 0 && len(alerts) > limit {
		alerts = alerts[:limit]
	}

	return alerts
}

// GetRules returns all alert rules
func (am *AlertManager) GetRules() []*AlertRule {
	am.mu.RLock()
	defer am.mu.RUnlock()

	rules := make([]*AlertRule, len(am.rules))
	copy(rules, am.rules)
	return rules
}

// InitializeDefaultRules initializes default alert rules
func (am *AlertManager) InitializeDefaultRules() {
	// System alerts
	am.AddRule(&AlertRule{
		Name:        "high_cpu_usage",
		Metric:      "cpu_usage",
		Condition:   ">",
		Threshold:   80,
		Duration:    5 * time.Minute,
		Level:       metrics.AlertLevelWarning,
		Message:     "CPU usage is above 80%",
		Description: "System CPU usage has exceeded the warning threshold",
		Channels:    []string{"slack"},
		Enabled:     true,
	})

	am.AddRule(&AlertRule{
		Name:        "high_memory_usage",
		Metric:      "memory_usage",
		Condition:   ">",
		Threshold:   90,
		Duration:    5 * time.Minute,
		Level:       metrics.AlertLevelWarning,
		Message:     "Memory usage is above 90%",
		Description: "System memory usage has exceeded the warning threshold",
		Channels:    []string{"slack"},
		Enabled:     true,
	})

	am.AddRule(&AlertRule{
		Name:        "disk_space_low",
		Metric:      "disk_usage",
		Condition:   ">",
		Threshold:   85,
		Duration:    10 * time.Minute,
		Level:       metrics.AlertLevelWarning,
		Message:     "Disk usage is above 85%",
		Description: "Disk space is running low",
		Channels:    []string{"email", "slack"},
		Enabled:     true,
	})

	// Node alerts
	am.AddRule(&AlertRule{
		Name:        "node_not_syncing",
		Metric:      "node_syncing",
		Condition:   "==",
		Threshold:   0,
		Duration:    15 * time.Minute,
		Level:       metrics.AlertLevelError,
		Message:     "Node is not syncing",
		Description: "The node has stopped syncing with the network",
		Channels:    []string{"slack", "email"},
		Enabled:     true,
	})

	am.logger.Info("Default alert rules initialized", "count", 4)
}