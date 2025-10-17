package alerting

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wemix/wemixvisor/internal/metrics"
	"github.com/wemix/wemixvisor/pkg/logger"
)

// TestNewEmailChannel tests creating an email channel
func TestNewEmailChannel(t *testing.T) {
	tests := []struct {
		name           string
		config         map[string]interface{}
		expectedHost   string
		expectedPort   int
		expectedFrom   string
		expectedToLen  int
	}{
		{
			name: "complete configuration",
			config: map[string]interface{}{
				"smtp_host": "smtp.example.com",
				"smtp_port": 587.0,
				"username":  "user@example.com",
				"password":  "password123",
				"from":      "alerts@example.com",
				"to":        []interface{}{"admin1@example.com", "admin2@example.com"},
			},
			expectedHost:  "smtp.example.com",
			expectedPort:  587,
			expectedFrom:  "alerts@example.com",
			expectedToLen: 2,
		},
		{
			name: "minimal configuration",
			config: map[string]interface{}{
				"smtp_host": "smtp.test.com",
			},
			expectedHost:  "smtp.test.com",
			expectedPort:  0,
			expectedFrom:  "",
			expectedToLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			logger := logger.NewTestLogger()

			// Act
			channel, err := NewEmailChannel("test_email", tt.config, logger)

			// Assert
			require.NoError(t, err)
			assert.NotNil(t, channel)
			assert.Equal(t, "test_email", channel.GetName())
			assert.Equal(t, "email", channel.GetType())
			assert.True(t, channel.IsEnabled())
			assert.Equal(t, tt.expectedHost, channel.smtpHost)
			assert.Equal(t, tt.expectedPort, channel.smtpPort)
			assert.Equal(t, tt.expectedFrom, channel.from)
			assert.Equal(t, tt.expectedToLen, len(channel.to))
		})
	}
}

// TestNewSlackChannel tests creating a Slack channel
func TestNewSlackChannel(t *testing.T) {
	tests := []struct {
		name          string
		config        map[string]interface{}
		expectedURL   string
		expectedError bool
	}{
		{
			name: "complete configuration",
			config: map[string]interface{}{
				"webhook_url": "https://hooks.slack.com/services/TEST",
				"channel":     "#alerts",
				"username":    "Custom Bot",
				"icon_emoji":  ":fire:",
			},
			expectedURL:   "https://hooks.slack.com/services/TEST",
			expectedError: false,
		},
		{
			name: "minimal configuration",
			config: map[string]interface{}{
				"webhook_url": "https://hooks.slack.com/services/MINIMAL",
			},
			expectedURL:   "https://hooks.slack.com/services/MINIMAL",
			expectedError: false,
		},
		{
			name:          "missing webhook_url",
			config:        map[string]interface{}{},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			logger := logger.NewTestLogger()

			// Act
			channel, err := NewSlackChannel("test_slack", tt.config, logger)

			// Assert
			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, channel)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, channel)
				assert.Equal(t, "test_slack", channel.GetName())
				assert.Equal(t, "slack", channel.GetType())
				assert.True(t, channel.IsEnabled())
				assert.Equal(t, tt.expectedURL, channel.webhookURL)
			}
		})
	}
}

// TestNewWebhookChannel tests creating a webhook channel
func TestNewWebhookChannel(t *testing.T) {
	tests := []struct {
		name           string
		config         map[string]interface{}
		expectedURL    string
		expectedMethod string
		expectedError  bool
	}{
		{
			name: "complete configuration",
			config: map[string]interface{}{
				"url":    "https://example.com/webhook",
				"method": "PUT",
				"headers": map[string]interface{}{
					"Authorization": "Bearer token123",
					"X-Custom":      "value",
				},
				"timeout": 30.0,
			},
			expectedURL:    "https://example.com/webhook",
			expectedMethod: "PUT",
			expectedError:  false,
		},
		{
			name: "minimal configuration",
			config: map[string]interface{}{
				"url": "https://example.com/minimal",
			},
			expectedURL:    "https://example.com/minimal",
			expectedMethod: "POST",
			expectedError:  false,
		},
		{
			name:          "missing url",
			config:        map[string]interface{}{},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			logger := logger.NewTestLogger()

			// Act
			channel, err := NewWebhookChannel("test_webhook", tt.config, logger)

			// Assert
			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, channel)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, channel)
				assert.Equal(t, "test_webhook", channel.GetName())
				assert.Equal(t, "webhook", channel.GetType())
				assert.True(t, channel.IsEnabled())
				assert.Equal(t, tt.expectedURL, channel.url)
				assert.Equal(t, tt.expectedMethod, channel.method)
			}
		})
	}
}

// TestNewDiscordChannel tests creating a Discord channel
func TestNewDiscordChannel(t *testing.T) {
	tests := []struct {
		name          string
		config        map[string]interface{}
		expectedURL   string
		expectedError bool
	}{
		{
			name: "complete configuration",
			config: map[string]interface{}{
				"webhook_url": "https://discord.com/api/webhooks/TEST",
				"username":    "Custom Bot",
				"avatar_url":  "https://example.com/avatar.png",
			},
			expectedURL:   "https://discord.com/api/webhooks/TEST",
			expectedError: false,
		},
		{
			name: "minimal configuration",
			config: map[string]interface{}{
				"webhook_url": "https://discord.com/api/webhooks/MINIMAL",
			},
			expectedURL:   "https://discord.com/api/webhooks/MINIMAL",
			expectedError: false,
		},
		{
			name:          "missing webhook_url",
			config:        map[string]interface{}{},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			logger := logger.NewTestLogger()

			// Act
			channel, err := NewDiscordChannel("test_discord", tt.config, logger)

			// Assert
			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, channel)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, channel)
				assert.Equal(t, "test_discord", channel.GetName())
				assert.Equal(t, "discord", channel.GetType())
				assert.True(t, channel.IsEnabled())
				assert.Equal(t, tt.expectedURL, channel.webhookURL)
			}
		})
	}
}

// TestSlackChannelSend tests sending Slack notifications
func TestSlackChannelSend(t *testing.T) {
	tests := []struct {
		name           string
		alertLevel     metrics.AlertLevel
		serverStatus   int
		expectedError  bool
		expectedColor  string
	}{
		{
			name:          "info alert - success",
			alertLevel:    metrics.AlertLevelInfo,
			serverStatus:  http.StatusOK,
			expectedError: false,
			expectedColor: "#808080",
		},
		{
			name:          "warning alert - success",
			alertLevel:    metrics.AlertLevelWarning,
			serverStatus:  http.StatusOK,
			expectedError: false,
			expectedColor: "#FFA500",
		},
		{
			name:          "error alert - success",
			alertLevel:    metrics.AlertLevelError,
			serverStatus:  http.StatusOK,
			expectedError: false,
			expectedColor: "#FF0000",
		},
		{
			name:          "critical alert - success",
			alertLevel:    metrics.AlertLevelCritical,
			serverStatus:  http.StatusOK,
			expectedError: false,
			expectedColor: "#8B0000",
		},
		{
			name:          "webhook error",
			alertLevel:    metrics.AlertLevelWarning,
			serverStatus:  http.StatusInternalServerError,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange - Create mock HTTP server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.serverStatus)
			}))
			defer server.Close()

			logger := logger.NewTestLogger()
			config := map[string]interface{}{
				"webhook_url": server.URL,
				"channel":     "#test",
			}

			channel, err := NewSlackChannel("test", config, logger)
			require.NoError(t, err)

			alert := &metrics.Alert{
				ID:          "test-001",
				Name:        "Test Alert",
				Level:       tt.alertLevel,
				Message:     "This is a test alert",
				Description: "Testing Slack notifications",
				Metric:      "test_metric",
				Value:       85.5,
				Threshold:   80.0,
				Timestamp:   time.Now(),
			}

			// Act
			err = channel.Send(alert)

			// Assert
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestWebhookChannelSend tests sending webhook notifications
func TestWebhookChannelSend(t *testing.T) {
	tests := []struct {
		name          string
		method        string
		serverStatus  int
		expectedError bool
	}{
		{
			name:          "POST request - success",
			method:        "POST",
			serverStatus:  http.StatusOK,
			expectedError: false,
		},
		{
			name:          "PUT request - success",
			method:        "PUT",
			serverStatus:  http.StatusCreated,
			expectedError: false,
		},
		{
			name:          "server error",
			method:        "POST",
			serverStatus:  http.StatusBadRequest,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange - Create mock HTTP server
			requestReceived := false
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requestReceived = true
				assert.Equal(t, tt.method, r.Method)
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
				w.WriteHeader(tt.serverStatus)
			}))
			defer server.Close()

			logger := logger.NewTestLogger()
			config := map[string]interface{}{
				"url":    server.URL,
				"method": tt.method,
			}

			channel, err := NewWebhookChannel("test", config, logger)
			require.NoError(t, err)

			alert := &metrics.Alert{
				ID:          "test-001",
				Name:        "Test Alert",
				Level:       metrics.AlertLevelWarning,
				Message:     "This is a test alert",
				Description: "Testing webhook notifications",
				Metric:      "test_metric",
				Value:       85.5,
				Threshold:   80.0,
				Timestamp:   time.Now(),
			}

			// Act
			err = channel.Send(alert)

			// Assert
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.True(t, requestReceived, "webhook request should be received")
			}
		})
	}
}

// TestDiscordChannelSend tests sending Discord notifications
func TestDiscordChannelSend(t *testing.T) {
	tests := []struct {
		name          string
		alertLevel    metrics.AlertLevel
		serverStatus  int
		expectedError bool
		expectedColor int
	}{
		{
			name:          "info alert - success",
			alertLevel:    metrics.AlertLevelInfo,
			serverStatus:  http.StatusNoContent,
			expectedError: false,
			expectedColor: 0x808080,
		},
		{
			name:          "warning alert - success",
			alertLevel:    metrics.AlertLevelWarning,
			serverStatus:  http.StatusOK,
			expectedError: false,
			expectedColor: 0xFFA500,
		},
		{
			name:          "error alert - success",
			alertLevel:    metrics.AlertLevelError,
			serverStatus:  http.StatusNoContent,
			expectedError: false,
			expectedColor: 0xFF0000,
		},
		{
			name:          "webhook error",
			alertLevel:    metrics.AlertLevelWarning,
			serverStatus:  http.StatusBadRequest,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange - Create mock HTTP server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.serverStatus)
			}))
			defer server.Close()

			logger := logger.NewTestLogger()
			config := map[string]interface{}{
				"webhook_url": server.URL,
				"username":    "Test Bot",
			}

			channel, err := NewDiscordChannel("test", config, logger)
			require.NoError(t, err)

			alert := &metrics.Alert{
				ID:          "test-001",
				Name:        "Test Alert",
				Level:       tt.alertLevel,
				Message:     "This is a test alert",
				Description: "Testing Discord notifications",
				Metric:      "test_metric",
				Value:       85.5,
				Threshold:   80.0,
				Timestamp:   time.Now(),
			}

			// Act
			err = channel.Send(alert)

			// Assert
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestEmailChannelSend tests email sending (error case only)
func TestEmailChannelSend(t *testing.T) {
	// Arrange
	logger := logger.NewTestLogger()
	config := map[string]interface{}{
		"smtp_host": "invalid.example.com",
		"smtp_port": 9999.0,
		"from":      "test@example.com",
		"to":        []interface{}{"admin@example.com"},
	}

	channel, err := NewEmailChannel("test", config, logger)
	require.NoError(t, err)

	alert := &metrics.Alert{
		ID:          "test-001",
		Name:        "Test Alert",
		Level:       metrics.AlertLevelWarning,
		Message:     "This is a test alert",
		Description: "Testing email notifications",
		Metric:      "test_metric",
		Value:       85.5,
		Threshold:   80.0,
		Timestamp:   time.Now(),
	}

	// Act
	err = channel.Send(alert)

	// Assert - should fail since we're using invalid SMTP server
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to send email")
}

// TestWebhookChannelTimeout tests webhook timeout configuration
// Note: httptest.NewServer doesn't properly simulate network timeouts,
// so we test that timeout is correctly configured instead
func TestWebhookChannelTimeout(t *testing.T) {
	// Arrange
	logger := logger.NewTestLogger()
	config := map[string]interface{}{
		"url":     "https://example.com/webhook",
		"timeout": 5.0, // 5 seconds
	}

	// Act
	channel, err := NewWebhookChannel("test", config, logger)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, 5*time.Second, channel.timeout, "timeout should be configured correctly")
}

// TestWebhookChannelCustomHeaders tests custom headers
func TestWebhookChannelCustomHeaders(t *testing.T) {
	// Arrange - Create server that checks headers
	headerChecked := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer token123", r.Header.Get("Authorization"))
		assert.Equal(t, "custom-value", r.Header.Get("X-Custom-Header"))
		headerChecked = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	logger := logger.NewTestLogger()
	config := map[string]interface{}{
		"url": server.URL,
		"headers": map[string]interface{}{
			"Authorization":   "Bearer token123",
			"X-Custom-Header": "custom-value",
		},
	}

	channel, err := NewWebhookChannel("test", config, logger)
	require.NoError(t, err)

	alert := &metrics.Alert{
		ID:        "test-001",
		Name:      "Test Alert",
		Level:     metrics.AlertLevelWarning,
		Message:   "Test headers",
		Timestamp: time.Now(),
	}

	// Act
	err = channel.Send(alert)

	// Assert
	assert.NoError(t, err)
	assert.True(t, headerChecked, "custom headers should be sent")
}

// TestChannelInterface tests that all channels implement the interface correctly
func TestChannelInterface(t *testing.T) {
	logger := logger.NewTestLogger()

	// Test each channel type
	channels := []struct {
		name         string
		channel      NotificationChannel
		expectedType string
		expectedName string
	}{
		{
			name: "Email Channel",
			channel: func() NotificationChannel {
				ch, _ := NewEmailChannel("email-test", map[string]interface{}{
					"smtp_host": "smtp.example.com",
				}, logger)
				return ch
			}(),
			expectedType: "email",
			expectedName: "email-test",
		},
		{
			name: "Slack Channel",
			channel: func() NotificationChannel {
				ch, _ := NewSlackChannel("slack-test", map[string]interface{}{
					"webhook_url": "https://hooks.slack.com/test",
				}, logger)
				return ch
			}(),
			expectedType: "slack",
			expectedName: "slack-test",
		},
		{
			name: "Webhook Channel",
			channel: func() NotificationChannel {
				ch, _ := NewWebhookChannel("webhook-test", map[string]interface{}{
					"url": "https://example.com/webhook",
				}, logger)
				return ch
			}(),
			expectedType: "webhook",
			expectedName: "webhook-test",
		},
		{
			name: "Discord Channel",
			channel: func() NotificationChannel {
				ch, _ := NewDiscordChannel("discord-test", map[string]interface{}{
					"webhook_url": "https://discord.com/api/webhooks/test",
				}, logger)
				return ch
			}(),
			expectedType: "discord",
			expectedName: "discord-test",
		},
	}

	for _, tt := range channels {
		t.Run(tt.name, func(t *testing.T) {
			// Assert interface methods
			assert.Equal(t, tt.expectedType, tt.channel.GetType())
			assert.Equal(t, tt.expectedName, tt.channel.GetName())
			assert.True(t, tt.channel.IsEnabled())

			// Verify Send method exists (signature check)
			assert.NotPanics(t, func() {
				// Create dummy alert
				alert := &metrics.Alert{
					ID:        "test",
					Name:      "Test",
					Level:     metrics.AlertLevelInfo,
					Timestamp: time.Now(),
				}
				// Send will fail but shouldn't panic
				_ = tt.channel.Send(alert)
			})
		})
	}
}

// TestSlackChannelWithChannel tests Slack channel field
func TestSlackChannelWithChannel(t *testing.T) {
	// Arrange
	channelReceived := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// We can't easily parse JSON here, so just verify request was received
		channelReceived = "received"
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	logger := logger.NewTestLogger()
	config := map[string]interface{}{
		"webhook_url": server.URL,
		"channel":     "#custom-channel",
	}

	channel, err := NewSlackChannel("test", config, logger)
	require.NoError(t, err)
	assert.Equal(t, "#custom-channel", channel.channel)

	alert := &metrics.Alert{
		ID:        "test-001",
		Name:      "Test",
		Level:     metrics.AlertLevelInfo,
		Timestamp: time.Now(),
	}

	// Act
	err = channel.Send(alert)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, "received", channelReceived)
}

// TestDiscordChannelWithAvatar tests Discord avatar URL
func TestDiscordChannelWithAvatar(t *testing.T) {
	// Arrange
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	logger := logger.NewTestLogger()
	config := map[string]interface{}{
		"webhook_url": server.URL,
		"avatar_url":  "https://example.com/avatar.png",
	}

	channel, err := NewDiscordChannel("test", config, logger)
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/avatar.png", channel.avatarURL)

	alert := &metrics.Alert{
		ID:        "test-001",
		Name:      "Test",
		Level:     metrics.AlertLevelInfo,
		Timestamp: time.Now(),
	}

	// Act
	err = channel.Send(alert)

	// Assert
	assert.NoError(t, err)
}

// TestInvalidWebhookURL tests invalid webhook URLs
func TestInvalidWebhookURL(t *testing.T) {
	logger := logger.NewTestLogger()

	tests := []struct {
		name        string
		channelType string
		url         string
	}{
		{"Slack invalid URL", "slack", "not-a-valid-url"},
		{"Discord invalid URL", "discord", "not-a-valid-url"},
		{"Webhook invalid URL", "webhook", "not-a-valid-url"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var channel NotificationChannel
			var err error

			switch tt.channelType {
			case "slack":
				channel, err = NewSlackChannel("test", map[string]interface{}{
					"webhook_url": tt.url,
				}, logger)
			case "discord":
				channel, err = NewDiscordChannel("test", map[string]interface{}{
					"webhook_url": tt.url,
				}, logger)
			case "webhook":
				channel, err = NewWebhookChannel("test", map[string]interface{}{
					"url": tt.url,
				}, logger)
			}

			require.NoError(t, err) // Channel creation should succeed

			alert := &metrics.Alert{
				ID:        "test",
				Name:      "Test",
				Level:     metrics.AlertLevelInfo,
				Timestamp: time.Now(),
			}

			// Act - Send should fail
			err = channel.Send(alert)

			// Assert
			assert.Error(t, err, fmt.Sprintf("%s should fail with invalid URL", tt.channelType))
		})
	}
}
