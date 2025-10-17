package alerting

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/smtp"
	"strings"
	"time"

	"github.com/wemix/wemixvisor/internal/metrics"
	"github.com/wemix/wemixvisor/pkg/logger"
)

// EmailChannel sends notifications via email
type EmailChannel struct {
	name       string
	enabled    bool
	smtpHost   string
	smtpPort   int
	username   string
	password   string
	from       string
	to         []string
	logger     *logger.Logger
}

// NewEmailChannel creates a new email notification channel
func NewEmailChannel(name string, config map[string]interface{}, logger *logger.Logger) (*EmailChannel, error) {
	ch := &EmailChannel{
		name:    name,
		enabled: true,
		logger:  logger,
	}

	// Parse configuration
	if host, ok := config["smtp_host"].(string); ok {
		ch.smtpHost = host
	}
	if port, ok := config["smtp_port"].(float64); ok {
		ch.smtpPort = int(port)
	}
	if username, ok := config["username"].(string); ok {
		ch.username = username
	}
	if password, ok := config["password"].(string); ok {
		ch.password = password
	}
	if from, ok := config["from"].(string); ok {
		ch.from = from
	}
	if to, ok := config["to"].([]interface{}); ok {
		for _, addr := range to {
			if email, ok := addr.(string); ok {
				ch.to = append(ch.to, email)
			}
		}
	}

	return ch, nil
}

// Send sends an email notification
func (e *EmailChannel) Send(alert *metrics.Alert) error {
	subject := fmt.Sprintf("[%s] Alert: %s", alert.Level, alert.Name)

	body := fmt.Sprintf(`
Alert Details:
--------------
Name: %s
Level: %s
Message: %s
Description: %s
Metric: %s
Value: %.2f
Threshold: %.2f
Time: %s

This is an automated alert from Wemixvisor monitoring system.
	`, alert.Name, alert.Level, alert.Message, alert.Description,
		alert.Metric, alert.Value, alert.Threshold,
		alert.Timestamp.Format(time.RFC3339))

	// Construct email message
	message := fmt.Sprintf("From: %s\r\n", e.from)
	message += fmt.Sprintf("To: %s\r\n", strings.Join(e.to, ", "))
	message += fmt.Sprintf("Subject: %s\r\n", subject)
	message += "\r\n" + body

	// Send email
	auth := smtp.PlainAuth("", e.username, e.password, e.smtpHost)
	addr := fmt.Sprintf("%s:%d", e.smtpHost, e.smtpPort)

	err := smtp.SendMail(addr, auth, e.from, e.to, []byte(message))
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

// GetType returns the channel type
func (e *EmailChannel) GetType() string {
	return "email"
}

// GetName returns the channel name
func (e *EmailChannel) GetName() string {
	return e.name
}

// IsEnabled returns whether the channel is enabled
func (e *EmailChannel) IsEnabled() bool {
	return e.enabled
}

// SlackChannel sends notifications to Slack
type SlackChannel struct {
	name       string
	enabled    bool
	webhookURL string
	channel    string
	username   string
	iconEmoji  string
	logger     *logger.Logger
}

// NewSlackChannel creates a new Slack notification channel
func NewSlackChannel(name string, config map[string]interface{}, logger *logger.Logger) (*SlackChannel, error) {
	ch := &SlackChannel{
		name:      name,
		enabled:   true,
		username:  "Wemixvisor",
		iconEmoji: ":robot_face:",
		logger:    logger,
	}

	// Parse configuration
	if url, ok := config["webhook_url"].(string); ok {
		ch.webhookURL = url
	} else {
		return nil, fmt.Errorf("webhook_url is required for Slack channel")
	}

	if channel, ok := config["channel"].(string); ok {
		ch.channel = channel
	}
	if username, ok := config["username"].(string); ok {
		ch.username = username
	}
	if icon, ok := config["icon_emoji"].(string); ok {
		ch.iconEmoji = icon
	}

	return ch, nil
}

// Send sends a Slack notification
func (s *SlackChannel) Send(alert *metrics.Alert) error {
	// Determine color based on alert level
	color := "#808080" // gray for info
	switch alert.Level {
	case metrics.AlertLevelWarning:
		color = "#FFA500" // orange
	case metrics.AlertLevelError:
		color = "#FF0000" // red
	case metrics.AlertLevelCritical:
		color = "#8B0000" // dark red
	}

	// Create Slack message payload
	payload := map[string]interface{}{
		"username":   s.username,
		"icon_emoji": s.iconEmoji,
		"attachments": []map[string]interface{}{
			{
				"color":      color,
				"title":      fmt.Sprintf("%s Alert: %s", alert.Level, alert.Name),
				"text":       alert.Message,
				"fallback":   alert.Message,
				"fields": []map[string]interface{}{
					{
						"title": "Description",
						"value": alert.Description,
						"short": false,
					},
					{
						"title": "Metric",
						"value": alert.Metric,
						"short": true,
					},
					{
						"title": "Value",
						"value": fmt.Sprintf("%.2f", alert.Value),
						"short": true,
					},
					{
						"title": "Threshold",
						"value": fmt.Sprintf("%.2f", alert.Threshold),
						"short": true,
					},
					{
						"title": "Time",
						"value": alert.Timestamp.Format(time.RFC3339),
						"short": true,
					},
				},
				"footer":      "Wemixvisor Alert System",
				"footer_icon": "https://platform.slack-edge.com/img/default_application_icon.png",
				"ts":          alert.Timestamp.Unix(),
			},
		},
	}

	if s.channel != "" {
		payload["channel"] = s.channel
	}

	// Send to Slack
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Slack payload: %w", err)
	}

	resp, err := http.Post(s.webhookURL, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to send Slack notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Slack webhook returned status %d", resp.StatusCode)
	}

	return nil
}

// GetType returns the channel type
func (s *SlackChannel) GetType() string {
	return "slack"
}

// GetName returns the channel name
func (s *SlackChannel) GetName() string {
	return s.name
}

// IsEnabled returns whether the channel is enabled
func (s *SlackChannel) IsEnabled() bool {
	return s.enabled
}

// WebhookChannel sends notifications to a generic webhook
type WebhookChannel struct {
	name      string
	enabled   bool
	url       string
	method    string
	headers   map[string]string
	timeout   time.Duration
	logger    *logger.Logger
}

// NewWebhookChannel creates a new webhook notification channel
func NewWebhookChannel(name string, config map[string]interface{}, logger *logger.Logger) (*WebhookChannel, error) {
	ch := &WebhookChannel{
		name:    name,
		enabled: true,
		method:  "POST",
		headers: make(map[string]string),
		timeout: 10 * time.Second,
		logger:  logger,
	}

	// Parse configuration
	if url, ok := config["url"].(string); ok {
		ch.url = url
	} else {
		return nil, fmt.Errorf("url is required for webhook channel")
	}

	if method, ok := config["method"].(string); ok {
		ch.method = method
	}

	if headers, ok := config["headers"].(map[string]interface{}); ok {
		for key, value := range headers {
			if strValue, ok := value.(string); ok {
				ch.headers[key] = strValue
			}
		}
	}

	if timeout, ok := config["timeout"].(float64); ok {
		ch.timeout = time.Duration(timeout) * time.Second
	}

	return ch, nil
}

// Send sends a webhook notification
func (w *WebhookChannel) Send(alert *metrics.Alert) error {
	// Create webhook payload
	payload := map[string]interface{}{
		"alert": map[string]interface{}{
			"id":          alert.ID,
			"name":        alert.Name,
			"level":       string(alert.Level),
			"message":     alert.Message,
			"description": alert.Description,
			"metric":      alert.Metric,
			"value":       alert.Value,
			"threshold":   alert.Threshold,
			"timestamp":   alert.Timestamp.Format(time.RFC3339),
			"labels":      alert.Labels,
		},
		"source": "wemixvisor",
		"time":   time.Now().Format(time.RFC3339),
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	client := &http.Client{
		Timeout: w.timeout,
	}

	req, err := http.NewRequest(w.method, w.url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to create webhook request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	for key, value := range w.headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}

// GetType returns the channel type
func (w *WebhookChannel) GetType() string {
	return "webhook"
}

// GetName returns the channel name
func (w *WebhookChannel) GetName() string {
	return w.name
}

// IsEnabled returns whether the channel is enabled
func (w *WebhookChannel) IsEnabled() bool {
	return w.enabled
}

// DiscordChannel sends notifications to Discord
type DiscordChannel struct {
	name       string
	enabled    bool
	webhookURL string
	username   string
	avatarURL  string
	logger     *logger.Logger
}

// NewDiscordChannel creates a new Discord notification channel
func NewDiscordChannel(name string, config map[string]interface{}, logger *logger.Logger) (*DiscordChannel, error) {
	ch := &DiscordChannel{
		name:     name,
		enabled:  true,
		username: "Wemixvisor Bot",
		logger:   logger,
	}

	// Parse configuration
	if url, ok := config["webhook_url"].(string); ok {
		ch.webhookURL = url
	} else {
		return nil, fmt.Errorf("webhook_url is required for Discord channel")
	}

	if username, ok := config["username"].(string); ok {
		ch.username = username
	}
	if avatar, ok := config["avatar_url"].(string); ok {
		ch.avatarURL = avatar
	}

	return ch, nil
}

// Send sends a Discord notification
func (d *DiscordChannel) Send(alert *metrics.Alert) error {
	// Determine color based on alert level
	var color int
	switch alert.Level {
	case metrics.AlertLevelInfo:
		color = 0x808080 // gray
	case metrics.AlertLevelWarning:
		color = 0xFFA500 // orange
	case metrics.AlertLevelError:
		color = 0xFF0000 // red
	case metrics.AlertLevelCritical:
		color = 0x8B0000 // dark red
	default:
		color = 0x808080 // gray
	}

	// Create Discord embed
	embed := map[string]interface{}{
		"title":       fmt.Sprintf("%s Alert: %s", alert.Level, alert.Name),
		"description": alert.Message,
		"color":       color,
		"fields": []map[string]interface{}{
			{
				"name":   "Description",
				"value":  alert.Description,
				"inline": false,
			},
			{
				"name":   "Metric",
				"value":  alert.Metric,
				"inline": true,
			},
			{
				"name":   "Value",
				"value":  fmt.Sprintf("%.2f", alert.Value),
				"inline": true,
			},
			{
				"name":   "Threshold",
				"value":  fmt.Sprintf("%.2f", alert.Threshold),
				"inline": true,
			},
		},
		"footer": map[string]interface{}{
			"text": "Wemixvisor Alert System",
		},
		"timestamp": alert.Timestamp.Format(time.RFC3339),
	}

	// Create Discord message payload
	payload := map[string]interface{}{
		"username": d.username,
		"embeds":   []interface{}{embed},
	}

	if d.avatarURL != "" {
		payload["avatar_url"] = d.avatarURL
	}

	// Send to Discord
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Discord payload: %w", err)
	}

	resp, err := http.Post(d.webhookURL, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to send Discord notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("Discord webhook returned status %d", resp.StatusCode)
	}

	return nil
}

// GetType returns the channel type
func (d *DiscordChannel) GetType() string {
	return "discord"
}

// GetName returns the channel name
func (d *DiscordChannel) GetName() string {
	return d.name
}

// IsEnabled returns whether the channel is enabled
func (d *DiscordChannel) IsEnabled() bool {
	return d.enabled
}