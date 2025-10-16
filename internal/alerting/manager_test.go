package alerting

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wemix/wemixvisor/internal/metrics"
	"github.com/wemix/wemixvisor/pkg/logger"
)

// MockNotificationChannel is a mock implementation of NotificationChannel
type MockNotificationChannel struct {
	name       string
	channelType string
	enabled    bool
	sendCalls  []*metrics.Alert
	sendError  error
}

func NewMockChannel(name, channelType string, enabled bool) *MockNotificationChannel {
	return &MockNotificationChannel{
		name:       name,
		channelType: channelType,
		enabled:    enabled,
		sendCalls:  make([]*metrics.Alert, 0),
	}
}

func (m *MockNotificationChannel) Send(alert *metrics.Alert) error {
	m.sendCalls = append(m.sendCalls, alert)
	return m.sendError
}

func (m *MockNotificationChannel) GetType() string {
	return m.channelType
}

func (m *MockNotificationChannel) GetName() string {
	return m.name
}

func (m *MockNotificationChannel) IsEnabled() bool {
	return m.enabled
}

func (m *MockNotificationChannel) GetSendCallCount() int {
	return len(m.sendCalls)
}

func (m *MockNotificationChannel) GetLastAlert() *metrics.Alert {
	if len(m.sendCalls) == 0 {
		return nil
	}
	return m.sendCalls[len(m.sendCalls)-1]
}

// TestNewAlertManager tests creating a new alert manager
func TestNewAlertManager(t *testing.T) {
	// Arrange
	config := &AlertConfig{
		Enabled:            true,
		EvaluationInterval: 30 * time.Second,
		AlertRetention:     24 * time.Hour,
		Channels:           []ChannelConfig{},
	}

	collectorConfig := &metrics.CollectorConfig{
		Enabled:             true,
		CollectionInterval:  1 * time.Second,
		EnableSystemMetrics: true,
	}
	logger := logger.NewTestLogger()
	collector := metrics.NewCollector(collectorConfig, logger)

	// Act
	manager := NewAlertManager(config, collector, logger)

	// Assert
	assert.NotNil(t, manager)
	assert.NotNil(t, manager.logger)
	assert.NotNil(t, manager.collector)
	assert.NotNil(t, manager.config)
	assert.Equal(t, 0, len(manager.rules))
	assert.Equal(t, 0, len(manager.channels))
	assert.Equal(t, 0, len(manager.alerts))
}

// TestAlertManagerStartStop tests starting and stopping the manager
func TestAlertManagerStartStop(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
	}{
		{
			name:    "start enabled manager",
			enabled: true,
		},
		{
			name:    "start disabled manager",
			enabled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			config := &AlertConfig{
				Enabled:            tt.enabled,
				EvaluationInterval: 100 * time.Millisecond,
				AlertRetention:     1 * time.Hour,
				Channels:           []ChannelConfig{},
			}

			collectorConfig := &metrics.CollectorConfig{
				Enabled:             true,
				CollectionInterval:  1 * time.Second,
				EnableSystemMetrics: true,
			}
			logger := logger.NewTestLogger()
			collector := metrics.NewCollector(collectorConfig, logger)
			manager := NewAlertManager(config, collector, logger)

			// Act - Start
			err := manager.Start()
			require.NoError(t, err)

			if tt.enabled {
				assert.NotNil(t, manager.ctx, "context should be initialized when enabled")
				assert.NotNil(t, manager.cancel, "cancel func should be initialized when enabled")
			}

			// Wait a moment
			time.Sleep(50 * time.Millisecond)

			// Act - Stop
			err = manager.Stop()
			assert.NoError(t, err)

			// Verify context is cancelled
			if tt.enabled && manager.ctx != nil {
				select {
				case <-manager.ctx.Done():
					// Context properly cancelled
				case <-time.After(100 * time.Millisecond):
					t.Error("context was not cancelled after Stop")
				}
			}
		})
	}
}

// TestAddRemoveRule tests adding and removing alert rules
func TestAddRemoveRule(t *testing.T) {
	// Arrange
	config := &AlertConfig{
		Enabled:            true,
		EvaluationInterval: 1 * time.Second,
		AlertRetention:     1 * time.Hour,
	}

	collectorConfig := &metrics.CollectorConfig{
		Enabled:             true,
		CollectionInterval:  1 * time.Second,
		EnableSystemMetrics: true,
	}
	logger := logger.NewTestLogger()
	collector := metrics.NewCollector(collectorConfig, logger)
	manager := NewAlertManager(config, collector, logger)

	rule1 := &AlertRule{
		Name:      "test_rule_1",
		Metric:    "cpu_usage",
		Condition: ">",
		Threshold: 80,
		Level:     metrics.AlertLevelWarning,
		Message:   "High CPU usage",
		Enabled:   true,
	}

	rule2 := &AlertRule{
		Name:      "test_rule_2",
		Metric:    "memory_usage",
		Condition: ">",
		Threshold: 90,
		Level:     metrics.AlertLevelError,
		Message:   "High memory usage",
		Enabled:   true,
	}

	// Act - Add rules
	manager.AddRule(rule1)
	manager.AddRule(rule2)

	// Assert - Rules added
	rules := manager.GetRules()
	assert.Equal(t, 2, len(rules))
	assert.Equal(t, "test_rule_1", rules[0].Name)
	assert.Equal(t, "test_rule_2", rules[1].Name)

	// Act - Remove rule
	err := manager.RemoveRule("test_rule_1")

	// Assert - Rule removed
	assert.NoError(t, err)
	rules = manager.GetRules()
	assert.Equal(t, 1, len(rules))
	assert.Equal(t, "test_rule_2", rules[0].Name)

	// Act - Try to remove non-existent rule
	err = manager.RemoveRule("non_existent")

	// Assert - Error returned
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "rule not found")
}

// TestGetMetricValue tests metric value extraction
func TestGetMetricValue(t *testing.T) {
	// Arrange
	config := &AlertConfig{
		Enabled:            true,
		EvaluationInterval: 1 * time.Second,
		AlertRetention:     1 * time.Hour,
	}

	collectorConfig := &metrics.CollectorConfig{
		Enabled:             true,
		CollectionInterval:  1 * time.Second,
		EnableSystemMetrics: true,
	}
	logger := logger.NewTestLogger()
	collector := metrics.NewCollector(collectorConfig, logger)
	manager := NewAlertManager(config, collector, logger)

	snapshot := &metrics.MetricsSnapshot{
		System: &metrics.SystemMetrics{
			CPUUsage:    75.5,
			MemoryUsage: 85.2,
			DiskUsage:   60.1,
		},
		Application: &metrics.ApplicationMetrics{
			NodeHeight: 12345,
		},
		Governance: &metrics.GovernanceMetrics{
			ProposalVoting: 5,
		},
	}

	tests := []struct {
		name          string
		metric        string
		expectedValue float64
	}{
		{"cpu_usage", "cpu_usage", 75.5},
		{"memory_usage", "memory_usage", 85.2},
		{"disk_usage", "disk_usage", 60.1},
		{"node_height", "node_height", 12345.0},
		{"proposals_voting", "proposals_voting", 5.0},
		{"unknown_metric", "unknown", 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			value := manager.getMetricValue(tt.metric, snapshot)

			// Assert
			assert.Equal(t, tt.expectedValue, value)
		})
	}
}

// TestCheckCondition tests condition evaluation
func TestCheckCondition(t *testing.T) {
	// Arrange
	config := &AlertConfig{
		Enabled:            true,
		EvaluationInterval: 1 * time.Second,
		AlertRetention:     1 * time.Hour,
	}

	collectorConfig := &metrics.CollectorConfig{
		Enabled:             true,
		CollectionInterval:  1 * time.Second,
		EnableSystemMetrics: true,
	}
	logger := logger.NewTestLogger()
	collector := metrics.NewCollector(collectorConfig, logger)
	manager := NewAlertManager(config, collector, logger)

	tests := []struct {
		name      string
		condition string
		value     float64
		threshold float64
		expected  bool
	}{
		{"greater than - true", ">", 85, 80, true},
		{"greater than - false", ">", 75, 80, false},
		{"greater than alias - true", "gt", 85, 80, true},
		{"greater or equal - true (greater)", ">=", 85, 80, true},
		{"greater or equal - true (equal)", ">=", 80, 80, true},
		{"greater or equal - false", ">=", 75, 80, false},
		{"less than - true", "<", 75, 80, true},
		{"less than - false", "<", 85, 80, false},
		{"less or equal - true (less)", "<=", 75, 80, true},
		{"less or equal - true (equal)", "<=", 80, 80, true},
		{"less or equal - false", "<=", 85, 80, false},
		{"equal - true", "==", 80, 80, true},
		{"equal - false", "==", 85, 80, false},
		{"not equal - true", "!=", 85, 80, true},
		{"not equal - false", "!=", 80, 80, false},
		{"unknown condition", "???", 85, 80, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			result := manager.checkCondition(tt.condition, tt.value, tt.threshold)

			// Assert
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestAlertTriggering tests alert triggering and resolution
func TestAlertTriggering(t *testing.T) {
	// Arrange
	config := &AlertConfig{
		Enabled:            true,
		EvaluationInterval: 100 * time.Millisecond,
		AlertRetention:     1 * time.Hour,
	}

	collectorConfig := &metrics.CollectorConfig{
		Enabled:             true,
		CollectionInterval:  100 * time.Millisecond,
		EnableSystemMetrics: true,
	}
	logger := logger.NewTestLogger()
	collector := metrics.NewCollector(collectorConfig, logger)

	// Start collector to generate snapshots
	err := collector.Start()
	require.NoError(t, err)
	defer collector.Stop()

	manager := NewAlertManager(config, collector, logger)

	// Add mock channel
	mockChannel := NewMockChannel("test", "mock", true)
	manager.channels["test"] = mockChannel

	// Add rule that should trigger
	rule := &AlertRule{
		Name:      "high_cpu",
		Metric:    "cpu_usage",
		Condition: ">",
		Threshold: 0, // Will always trigger since CPU usage is > 0
		Level:     metrics.AlertLevelWarning,
		Message:   "High CPU detected",
		Channels:  []string{"test"},
		Enabled:   true,
	}
	manager.AddRule(rule)

	// Start manager
	err = manager.Start()
	require.NoError(t, err)
	defer manager.Stop()

	// Wait for collection and evaluation
	time.Sleep(2500 * time.Millisecond)

	// Assert - Alert should be triggered
	activeAlerts := manager.GetActiveAlerts()
	assert.GreaterOrEqual(t, len(activeAlerts), 1, "at least one alert should be active")

	// Verify notification was sent
	assert.GreaterOrEqual(t, mockChannel.GetSendCallCount(), 1, "notification should be sent")
}

// TestInitializeDefaultRules tests initializing default rules
func TestInitializeDefaultRules(t *testing.T) {
	// Arrange
	config := &AlertConfig{
		Enabled:            true,
		EvaluationInterval: 1 * time.Second,
		AlertRetention:     1 * time.Hour,
	}

	collectorConfig := &metrics.CollectorConfig{
		Enabled:             true,
		CollectionInterval:  1 * time.Second,
		EnableSystemMetrics: true,
	}
	logger := logger.NewTestLogger()
	collector := metrics.NewCollector(collectorConfig, logger)
	manager := NewAlertManager(config, collector, logger)

	// Act
	manager.InitializeDefaultRules()

	// Assert
	rules := manager.GetRules()
	assert.Equal(t, 4, len(rules), "should have 4 default rules")

	// Verify rule names
	ruleNames := make(map[string]bool)
	for _, rule := range rules {
		ruleNames[rule.Name] = true
	}

	assert.True(t, ruleNames["high_cpu_usage"])
	assert.True(t, ruleNames["high_memory_usage"])
	assert.True(t, ruleNames["disk_space_low"])
	assert.True(t, ruleNames["node_not_syncing"])
}

// TestGetActiveAlerts tests getting active alerts
func TestGetActiveAlerts(t *testing.T) {
	// Arrange
	config := &AlertConfig{
		Enabled:            true,
		EvaluationInterval: 1 * time.Second,
		AlertRetention:     1 * time.Hour,
	}

	collectorConfig := &metrics.CollectorConfig{
		Enabled:             true,
		CollectionInterval:  1 * time.Second,
		EnableSystemMetrics: true,
	}
	logger := logger.NewTestLogger()
	collector := metrics.NewCollector(collectorConfig, logger)
	manager := NewAlertManager(config, collector, logger)

	now := time.Now()
	resolved := now.Add(-10 * time.Minute)

	// Add test alerts
	manager.alerts["alert1"] = &metrics.Alert{
		ID:   "alert1",
		Name: "Test Alert 1",
	}

	manager.alerts["alert2"] = &metrics.Alert{
		ID:         "alert2",
		Name:       "Test Alert 2",
		ResolvedAt: &resolved,
	}

	manager.alerts["alert3"] = &metrics.Alert{
		ID:   "alert3",
		Name: "Test Alert 3",
	}

	// Act
	activeAlerts := manager.GetActiveAlerts()

	// Assert
	assert.Equal(t, 2, len(activeAlerts), "should have 2 active alerts")

	// Verify only unresolved alerts are returned
	for _, alert := range activeAlerts {
		assert.Nil(t, alert.ResolvedAt, "active alerts should not have ResolvedAt set")
	}
}

// TestGetAlertHistory tests getting alert history
func TestGetAlertHistory(t *testing.T) {
	// Arrange
	config := &AlertConfig{
		Enabled:            true,
		EvaluationInterval: 1 * time.Second,
		AlertRetention:     1 * time.Hour,
	}

	collectorConfig := &metrics.CollectorConfig{
		Enabled:             true,
		CollectionInterval:  1 * time.Second,
		EnableSystemMetrics: true,
	}
	logger := logger.NewTestLogger()
	collector := metrics.NewCollector(collectorConfig, logger)
	manager := NewAlertManager(config, collector, logger)

	// Add test alerts
	for i := 0; i < 5; i++ {
		manager.alerts[string(rune('a'+i))] = &metrics.Alert{
			ID:   string(rune('a' + i)),
			Name: "Test Alert",
		}
	}

	tests := []struct {
		name          string
		limit         int
		expectedCount int
	}{
		{"no limit", 0, 5},
		{"limit 3", 3, 3},
		{"limit exceeds count", 10, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			history := manager.GetAlertHistory(tt.limit)

			// Assert
			assert.Equal(t, tt.expectedCount, len(history))
		})
	}
}

// TestCreateChannel tests creating notification channels
func TestCreateChannel(t *testing.T) {
	// Arrange
	config := &AlertConfig{
		Enabled:            true,
		EvaluationInterval: 1 * time.Second,
		AlertRetention:     1 * time.Hour,
	}

	collectorConfig := &metrics.CollectorConfig{
		Enabled:             true,
		CollectionInterval:  1 * time.Second,
		EnableSystemMetrics: true,
	}
	logger := logger.NewTestLogger()
	collector := metrics.NewCollector(collectorConfig, logger)
	manager := NewAlertManager(config, collector, logger)

	tests := []struct {
		name        string
		channelType string
		config      map[string]interface{}
		expectError bool
	}{
		{
			name:        "slack channel",
			channelType: "slack",
			config: map[string]interface{}{
				"webhook_url": "https://hooks.slack.com/services/test",
			},
			expectError: false,
		},
		{
			name:        "webhook channel",
			channelType: "webhook",
			config: map[string]interface{}{
				"url": "https://example.com/webhook",
			},
			expectError: false,
		},
		{
			name:        "discord channel",
			channelType: "discord",
			config: map[string]interface{}{
				"webhook_url": "https://discord.com/api/webhooks/test",
			},
			expectError: false,
		},
		{
			name:        "email channel",
			channelType: "email",
			config: map[string]interface{}{
				"smtp_host": "smtp.example.com",
				"smtp_port": 587.0,
				"from":      "alerts@example.com",
				"to":        []interface{}{"admin@example.com"},
			},
			expectError: false,
		},
		{
			name:        "unknown channel type",
			channelType: "unknown",
			config:      map[string]interface{}{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			cfg := ChannelConfig{
				Type:   tt.channelType,
				Name:   tt.name,
				Config: tt.config,
			}

			// Act
			channel, err := manager.createChannel(cfg)

			// Assert
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, channel)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, channel)
				assert.Equal(t, tt.channelType, channel.GetType())
			}
		})
	}
}

// TestGetActiveAlertsFiltered tests filtering active alerts by rule name
func TestGetActiveAlertsFiltered(t *testing.T) {
	// Arrange
	config := &AlertConfig{
		Enabled:            true,
		EvaluationInterval: 1 * time.Second,
		AlertRetention:     1 * time.Hour,
	}

	collectorConfig := &metrics.CollectorConfig{
		Enabled:             true,
		CollectionInterval:  1 * time.Second,
		EnableSystemMetrics: true,
	}
	logger := logger.NewTestLogger()
	collector := metrics.NewCollector(collectorConfig, logger)
	manager := NewAlertManager(config, collector, logger)

	// Add alerts for different rules
	alert1 := &metrics.Alert{
		ID:        "alert-1",
		Name:      "rule1",
		Level:     metrics.AlertLevelWarning,
		Timestamp: time.Now(),
	}
	alert2 := &metrics.Alert{
		ID:        "alert-2",
		Name:      "rule2",
		Level:     metrics.AlertLevelError,
		Timestamp: time.Now(),
	}
	alert3 := &metrics.Alert{
		ID:        "alert-3",
		Name:      "rule1",
		Level:     metrics.AlertLevelCritical,
		Timestamp: time.Now(),
	}

	manager.alerts["alert-1"] = alert1
	manager.alerts["alert-2"] = alert2
	manager.alerts["alert-3"] = alert3

	// Act - get all alerts
	all := manager.GetActiveAlerts()

	// Assert
	assert.Equal(t, 3, len(all), "should return all 3 alerts")

	// Manually filter by rule1 to test the alert structure
	rule1Alerts := make([]*metrics.Alert, 0)
	for _, alert := range all {
		if alert.Name == "rule1" {
			rule1Alerts = append(rule1Alerts, alert)
		}
	}

	assert.Equal(t, 2, len(rule1Alerts), "should have 2 alerts for rule1")
	for _, alert := range rule1Alerts {
		assert.Equal(t, "rule1", alert.Name)
	}

	// Manually filter by rule2
	rule2Alerts := make([]*metrics.Alert, 0)
	for _, alert := range all {
		if alert.Name == "rule2" {
			rule2Alerts = append(rule2Alerts, alert)
		}
	}

	assert.Equal(t, 1, len(rule2Alerts), "should have 1 alert for rule2")
	assert.Equal(t, "rule2", rule2Alerts[0].Name)
}

// TestInitializeDefaultRulesManual tests manual initialization of default rules
func TestInitializeDefaultRulesManual(t *testing.T) {
	// Arrange
	config := &AlertConfig{
		Enabled:            true,
		EvaluationInterval: 1 * time.Second,
		AlertRetention:     1 * time.Hour,
	}

	collectorConfig := &metrics.CollectorConfig{
		Enabled:             true,
		CollectionInterval:  1 * time.Second,
		EnableSystemMetrics: true,
	}

	logger := logger.NewTestLogger()
	collector := metrics.NewCollector(collectorConfig, logger)
	manager := NewAlertManager(config, collector, logger)

	// Verify no rules initially
	initialRules := manager.GetRules()
	assert.Equal(t, 0, len(initialRules), "should have no rules initially")

	// Act - call InitializeDefaultRules()
	manager.InitializeDefaultRules()

	// Assert - verify default rules were initialized
	rules := manager.GetRules()
	assert.Equal(t, 4, len(rules), "should have 4 default rules")

	// Verify specific rules exist
	ruleNames := make(map[string]bool)
	for _, rule := range rules {
		ruleNames[rule.Name] = true
		assert.True(t, rule.Enabled, "default rules should be enabled")
	}

	assert.True(t, ruleNames["high_cpu_usage"], "should have high_cpu_usage rule")
	assert.True(t, ruleNames["high_memory_usage"], "should have high_memory_usage rule")
	assert.True(t, ruleNames["disk_space_low"], "should have disk_space_low rule")
	assert.True(t, ruleNames["node_not_syncing"], "should have node_not_syncing rule")
}

// TestDisabledRuleEvaluation tests that disabled rules are not evaluated
func TestDisabledRuleEvaluation(t *testing.T) {
	// Arrange
	config := &AlertConfig{
		Enabled:            true,
		EvaluationInterval: 100 * time.Millisecond,
		AlertRetention:     1 * time.Hour,
	}

	collectorConfig := &metrics.CollectorConfig{
		Enabled:             true,
		CollectionInterval:  50 * time.Millisecond,
		EnableSystemMetrics: true,
	}

	logger := logger.NewTestLogger()
	collector := metrics.NewCollector(collectorConfig, logger)
	err := collector.Start()
	require.NoError(t, err)
	defer collector.Stop()

	manager := NewAlertManager(config, collector, logger)
	mockChannel := NewMockChannel("test", "mock", true)
	manager.channels["test"] = mockChannel

	// Add disabled rule
	rule := &AlertRule{
		Name:      "disabled_alert",
		Metric:    "cpu_usage",
		Condition: ">",
		Threshold: 0, // Would trigger if enabled
		Level:     metrics.AlertLevelWarning,
		Channels:  []string{"test"},
		Enabled:   false, // Disabled
	}
	manager.AddRule(rule)
	err = manager.Start()
	require.NoError(t, err)
	defer manager.Stop()

	// Wait for potential evaluation
	time.Sleep(300 * time.Millisecond)

	// Assert - no alerts should be triggered
	assert.Equal(t, 0, mockChannel.GetSendCallCount(), "disabled rule should not trigger notifications")
	activeAlerts := manager.GetActiveAlerts()
	disabledAlertCount := 0
	for _, alert := range activeAlerts {
		if alert.Name == "disabled_alert" {
			disabledAlertCount++
		}
	}
	assert.Equal(t, 0, disabledAlertCount, "disabled rule should not create alerts")
}

// TestAlertManagerConcurrency tests concurrent access to alert manager
func TestAlertManagerConcurrency(t *testing.T) {
	// Arrange
	config := &AlertConfig{
		Enabled:            true,
		EvaluationInterval: 50 * time.Millisecond,
		AlertRetention:     1 * time.Hour,
	}

	collectorConfig := &metrics.CollectorConfig{
		Enabled:             true,
		CollectionInterval:  50 * time.Millisecond,
		EnableSystemMetrics: true,
	}
	logger := logger.NewTestLogger()
	collector := metrics.NewCollector(collectorConfig, logger)

	err := collector.Start()
	require.NoError(t, err)
	defer collector.Stop()

	manager := NewAlertManager(config, collector, logger)

	err = manager.Start()
	require.NoError(t, err)
	defer manager.Stop()

	// Act - concurrent operations
	done := make(chan bool)
	goroutines := 10

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			for j := 0; j < 10; j++ {
				// Add rule
				rule := &AlertRule{
					Name:      string(rune('a' + id)),
					Metric:    "cpu_usage",
					Condition: ">",
					Threshold: 50,
					Enabled:   true,
				}
				manager.AddRule(rule)

				// Get rules
				_ = manager.GetRules()

				// Get active alerts
				_ = manager.GetActiveAlerts()

				// Get history
				_ = manager.GetAlertHistory(10)

				time.Sleep(10 * time.Millisecond)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < goroutines; i++ {
		<-done
	}

	// Assert - no race conditions
	rules := manager.GetRules()
	assert.NotNil(t, rules)
}
