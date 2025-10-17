# Alerting Configuration Guide

Complete guide to configuring and managing the wemixvisor alerting system.

## Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Configuration](#configuration)
- [Alert Rules](#alert-rules)
- [Notification Channels](#notification-channels)
- [Alert Management](#alert-management)
- [Best Practices](#best-practices)
- [Troubleshooting](#troubleshooting)

## Overview

The wemixvisor alerting system provides real-time monitoring and notifications for critical events, resource issues, and operational anomalies. It supports multiple notification channels and flexible rule-based alerting.

### Key Features

- **Rule-Based Alerting**: Define custom alert rules with flexible conditions
- **Multi-Channel Notifications**: Email, Slack, Discord, and generic webhooks
- **Severity Levels**: Critical, warning, and info severity classifications
- **Alert Aggregation**: Prevent alert flooding with intelligent deduplication
- **Alert History**: Track and audit alert history
- **Conditional Evaluation**: Fire alerts only when conditions persist
- **Template Support**: Customizable alert messages

### Alert Lifecycle

```
┌─────────────┐
│ Condition   │
│ Evaluation  │
└──────┬──────┘
       │
       ├─ False ──> No Alert
       │
       └─ True ───> Wait (for duration)
                    │
                    └─> Still True?
                        │
                        ├─ Yes ──> Fire Alert
                        │          │
                        │          ├─> Email
                        │          ├─> Slack
                        │          ├─> Discord
                        │          └─> Webhook
                        │
                        └─ No ───> Reset
```

## Architecture

### Components

```
┌──────────────────────────────────────────────┐
│           Metrics Collector                  │
│  (System, Process, Chain, Application)       │
└─────────────┬────────────────────────────────┘
              │ Metrics Stream
              v
┌──────────────────────────────────────────────┐
│          Alert Evaluator                     │
│  - Load rules                                │
│  - Evaluate conditions                       │
│  - Track alert state                         │
│  - Apply "for" duration                      │
└─────────────┬────────────────────────────────┘
              │ Triggered Alerts
              v
┌──────────────────────────────────────────────┐
│        Notification Manager                  │
│  - Route to channels                         │
│  - Format messages                           │
│  - Handle retries                            │
│  - Track delivery                            │
└─────────────┬────────────────────────────────┘
              │
      ┌───────┴───────┬───────────┬───────────┐
      v               v           v           v
┌─────────┐   ┌────────────┐ ┌──────────┐ ┌─────────┐
│  Email  │   │   Slack    │ │ Discord  │ │ Webhook │
└─────────┘   └────────────┘ └──────────┘ └─────────┘
```

### Alert States

- **Pending**: Condition met but waiting for "for" duration
- **Firing**: Alert actively firing, notifications sent
- **Resolved**: Condition no longer met
- **Acknowledged**: Alert acknowledged by operator
- **Silenced**: Alert temporarily suppressed

## Configuration

### Basic Configuration

```toml
[alerting]
# Enable alerting system
enabled = true

# How often to evaluate alert rules
evaluation_interval = "10s"

# How long to keep alert history
alert_retention = "24h"

# Maximum alerts per minute (rate limiting)
max_alerts_per_minute = 60
```

### Environment Variables

```bash
# Enable alerting
export WEMIXVISOR_ALERTING_ENABLED=true

# Override evaluation interval
export WEMIXVISOR_ALERTING_EVALUATION_INTERVAL=15s

# Alert history retention
export WEMIXVISOR_ALERTING_RETENTION=48h
```

## Alert Rules

### Rule Structure

```toml
[[alerting.rules]]
name = "rule_name"           # Unique identifier
condition = "expression"      # Boolean expression to evaluate
severity = "warning"          # critical, warning, info
for = "5m"                    # Duration condition must be true
message = "Alert message"     # Human-readable description
labels = { key = "value" }    # Optional labels for routing/filtering
```

### Condition Expressions

Alert conditions use simple comparison expressions with metrics:

**Operators**:
- `>` greater than
- `<` less than
- `>=` greater than or equal
- `<=` less than or equal
- `==` equal
- `!=` not equal
- `&&` logical AND
- `||` logical OR

**Available Metrics**:
- `cpu_usage` - CPU usage percentage
- `memory_usage` - Memory usage percentage
- `disk_usage` - Disk usage percentage
- `block_height` - Current block height
- `expected_height` - Expected network height
- `syncing` - Sync status (true/false)
- `peers` - Peer count
- `upgrade_pending` - Pending upgrades count
- `process_restarts` - Process restart count
- `api_error_rate` - API error percentage

### Example Rules

#### System Resource Alerts

**High CPU Usage**:
```toml
[[alerting.rules]]
name = "high_cpu_usage"
condition = "cpu_usage > 80"
severity = "warning"
for = "5m"
message = "CPU usage is above 80% for 5 minutes"
labels = { component = "system", category = "resources" }
```

**Critical CPU Usage**:
```toml
[[alerting.rules]]
name = "critical_cpu_usage"
condition = "cpu_usage > 90"
severity = "critical"
for = "2m"
message = "CRITICAL: CPU usage above 90% for 2 minutes"
labels = { component = "system", category = "resources" }
```

**High Memory Usage**:
```toml
[[alerting.rules]]
name = "high_memory_usage"
condition = "memory_usage > 85"
severity = "warning"
for = "5m"
message = "Memory usage is above 85%"
labels = { component = "system", category = "resources" }
```

**Critical Memory Usage**:
```toml
[[alerting.rules]]
name = "critical_memory_usage"
condition = "memory_usage > 95"
severity = "critical"
for = "1m"
message = "CRITICAL: Memory usage above 95%"
labels = { component = "system", category = "resources" }
```

**Low Disk Space**:
```toml
[[alerting.rules]]
name = "low_disk_space"
condition = "disk_usage > 90"
severity = "critical"
for = "1m"
message = "Disk space is critically low (>90%)"
labels = { component = "system", category = "storage" }
```

#### Chain Monitoring Alerts

**Node Not Syncing**:
```toml
[[alerting.rules]]
name = "node_not_syncing"
condition = "syncing == false && block_height < expected_height - 10"
severity = "critical"
for = "5m"
message = "Node stopped syncing or significantly behind"
labels = { component = "chain", category = "sync" }
```

**Low Peer Count**:
```toml
[[alerting.rules]]
name = "low_peer_count"
condition = "peers < 5"
severity = "warning"
for = "5m"
message = "Low peer count detected"
labels = { component = "chain", category = "network" }
```

**No Peers Connected**:
```toml
[[alerting.rules]]
name = "no_peers"
condition = "peers == 0"
severity = "critical"
for = "1m"
message = "CRITICAL: No peers connected"
labels = { component = "chain", category = "network" }
```

#### Application Alerts

**Upgrade Available**:
```toml
[[alerting.rules]]
name = "upgrade_available"
condition = "upgrade_pending > 0"
severity = "info"
for = "0s"
message = "New upgrade available - review required"
labels = { component = "upgrade", category = "operation" }
```

**Frequent Process Restarts**:
```toml
[[alerting.rules]]
name = "process_restarts"
condition = "process_restarts > 3"
severity = "warning"
for = "1h"
message = "Multiple process restarts detected"
labels = { component = "process", category = "stability" }
```

**High API Error Rate**:
```toml
[[alerting.rules]]
name = "high_api_error_rate"
condition = "api_error_rate > 10"
severity = "warning"
for = "5m"
message = "API error rate is above 10%"
labels = { component = "api", category = "reliability" }
```

#### Complex Conditions

**Combined Resource Pressure**:
```toml
[[alerting.rules]]
name = "resource_pressure"
condition = "cpu_usage > 70 && memory_usage > 80"
severity = "warning"
for = "10m"
message = "High CPU and memory usage detected"
labels = { component = "system", category = "resources" }
```

**Sync Issues with Low Peers**:
```toml
[[alerting.rules]]
name = "sync_and_peer_issues"
condition = "syncing == false && peers < 10 && block_height < expected_height - 50"
severity = "critical"
for = "10m"
message = "Sync issues combined with low peer count"
labels = { component = "chain", category = "critical" }
```

## Notification Channels

### Email

**Configuration**:
```toml
[alerting.email]
enabled = true
smtp_server = "smtp.gmail.com"
smtp_port = 587
from = "alerts@example.com"
to = ["admin@example.com", "oncall@example.com"]
username = "alerts@example.com"
# Password via environment: WEMIXVISOR_ALERT_EMAIL_PASSWORD
```

**Environment Variables**:
```bash
export WEMIXVISOR_ALERT_EMAIL_PASSWORD="your-secure-password"
```

**Email Template**:
```
Subject: [${SEVERITY}] Wemixvisor Alert: ${ALERT_NAME}

Alert: ${ALERT_NAME}
Severity: ${SEVERITY}
Time: ${TIMESTAMP}
Node: ${NODE_NAME}

Message:
${MESSAGE}

Details:
${DETAILS}

---
Wemixvisor Monitoring System
```

**Gmail Configuration**:
```toml
[alerting.email]
enabled = true
smtp_server = "smtp.gmail.com"
smtp_port = 587
from = "your-email@gmail.com"
to = ["recipient@example.com"]
username = "your-email@gmail.com"
# Use App Password, not regular password
```

**AWS SES Configuration**:
```toml
[alerting.email]
enabled = true
smtp_server = "email-smtp.us-east-1.amazonaws.com"
smtp_port = 587
from = "alerts@yourdomain.com"
to = ["team@yourdomain.com"]
username = "YOUR_SMTP_USERNAME"
# Password from AWS SES SMTP credentials
```

### Slack

**Configuration**:
```toml
[alerting.slack]
enabled = true
# Webhook URL via environment: WEMIXVISOR_ALERT_SLACK_WEBHOOK
channel = "#alerts"  # Optional, override webhook default
username = "Wemixvisor"  # Optional bot name
icon_emoji = ":warning:"  # Optional icon
```

**Setup Steps**:

1. Create Slack App: https://api.slack.com/apps
2. Enable Incoming Webhooks
3. Create webhook for your channel
4. Set environment variable:
   ```bash
   export WEMIXVISOR_ALERT_SLACK_WEBHOOK="https://hooks.slack.com/services/YOUR/WEBHOOK/URL"
   ```

**Message Format**:
```json
{
  "channel": "#alerts",
  "username": "Wemixvisor",
  "icon_emoji": ":warning:",
  "attachments": [{
    "color": "danger",
    "title": "[CRITICAL] Node Not Syncing",
    "text": "Node stopped syncing or significantly behind",
    "fields": [
      {"title": "Severity", "value": "critical", "short": true},
      {"title": "Node", "value": "validator-1", "short": true},
      {"title": "Block Height", "value": "1234567", "short": true},
      {"title": "Expected Height", "value": "1234600", "short": true}
    ],
    "footer": "Wemixvisor",
    "ts": 1634567890
  }]
}
```

**Severity Colors**:
- Critical: `danger` (red)
- Warning: `warning` (yellow)
- Info: `good` (green)

### Discord

**Configuration**:
```toml
[alerting.discord]
enabled = true
# Webhook URL via environment: WEMIXVISOR_ALERT_DISCORD_WEBHOOK
username = "Wemixvisor"  # Optional bot name
avatar_url = ""  # Optional avatar URL
```

**Setup Steps**:

1. Server Settings → Integrations → Webhooks
2. Create New Webhook
3. Set name and channel
4. Copy webhook URL
5. Set environment variable:
   ```bash
   export WEMIXVISOR_ALERT_DISCORD_WEBHOOK="https://discord.com/api/webhooks/YOUR/WEBHOOK/URL"
   ```

**Message Format**:
```json
{
  "username": "Wemixvisor",
  "embeds": [{
    "title": "[CRITICAL] Node Not Syncing",
    "description": "Node stopped syncing or significantly behind",
    "color": 15158332,
    "fields": [
      {"name": "Severity", "value": "critical", "inline": true},
      {"name": "Node", "value": "validator-1", "inline": true},
      {"name": "Block Height", "value": "1234567", "inline": true}
    ],
    "timestamp": "2025-10-17T10:30:00Z",
    "footer": {"text": "Wemixvisor Alerting"}
  }]
}
```

**Severity Colors** (decimal):
- Critical: `15158332` (red)
- Warning: `16776960` (yellow)
- Info: `3066993` (green)

### Webhook

Generic webhook integration for custom services (PagerDuty, Opsgenie, custom endpoints).

**Configuration**:
```toml
[alerting.webhook]
enabled = true
# URL via environment: WEMIXVISOR_ALERT_WEBHOOK_URL
timeout = 10  # Seconds
headers = { "Authorization" = "Bearer TOKEN" }  # Optional custom headers
```

**Environment Variable**:
```bash
export WEMIXVISOR_ALERT_WEBHOOK_URL="https://your-service.com/webhooks/alerts"
```

**Payload Format**:
```json
{
  "alert_name": "high_cpu_usage",
  "severity": "warning",
  "message": "CPU usage is above 80% for 5 minutes",
  "timestamp": "2025-10-17T10:30:00Z",
  "node_name": "validator-1",
  "labels": {
    "component": "system",
    "category": "resources"
  },
  "metrics": {
    "cpu_usage": 85.2,
    "memory_usage": 67.3,
    "disk_usage": 45.1
  },
  "state": "firing"
}
```

**PagerDuty Integration**:
```toml
[alerting.webhook]
enabled = true
# URL: https://events.pagerduty.com/v2/enqueue
headers = { "Authorization" = "Token token=YOUR_ROUTING_KEY" }
```

**Opsgenie Integration**:
```toml
[alerting.webhook]
enabled = true
# URL: https://api.opsgenie.com/v2/alerts
headers = { "Authorization" = "GenieKey YOUR_API_KEY" }
```

## Alert Management

### API Endpoints

**List Active Alerts**:
```bash
curl http://localhost:8080/api/v1/alerts
```

**Response**:
```json
{
  "alerts": [
    {
      "name": "high_cpu_usage",
      "severity": "warning",
      "state": "firing",
      "message": "CPU usage is above 80% for 5 minutes",
      "started_at": "2025-10-17T10:25:00Z",
      "labels": {
        "component": "system",
        "category": "resources"
      }
    }
  ],
  "total": 1
}
```

**Get Alert Details**:
```bash
curl http://localhost:8080/api/v1/alerts/high_cpu_usage
```

**Acknowledge Alert**:
```bash
curl -X POST http://localhost:8080/api/v1/alerts/high_cpu_usage/acknowledge
```

**Silence Alert**:
```bash
curl -X POST http://localhost:8080/api/v1/alerts/high_cpu_usage/silence \
  -H "Content-Type: application/json" \
  -d '{"duration": "1h"}'
```

**Get Alert History**:
```bash
curl http://localhost:8080/api/v1/alerts/history?limit=100
```

### WebSocket Streaming

```javascript
const ws = new WebSocket('ws://localhost:8080/api/v1/ws');

ws.onopen = () => {
  // Subscribe to alert updates
  ws.send(JSON.stringify({
    action: 'subscribe',
    topics: ['alerts']
  }));
};

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  if (data.topic === 'alerts') {
    console.log('Alert update:', data.data);
    // Display alert notification
    showAlertNotification(data.data);
  }
};
```

## Best Practices

### Alert Design

**DO**:
- ✅ Alert on symptoms, not causes
- ✅ Make alerts actionable (include remediation steps)
- ✅ Use appropriate severity levels
- ✅ Include relevant context in messages
- ✅ Test alerts before deploying

**DON'T**:
- ❌ Alert on everything (alert fatigue)
- ❌ Use vague messages ("Something is wrong")
- ❌ Set thresholds too low (false positives)
- ❌ Ignore "for" duration (flapping alerts)
- ❌ Send all alerts to everyone

### Severity Guidelines

**Critical** (Immediate Action):
- Service completely down
- Data loss risk
- Security breach
- <5% disk space remaining

**Warning** (Action Required):
- Degraded performance
- High resource usage
- Approaching limits
- Sync issues

**Info** (Awareness):
- Upgrades available
- Configuration changes
- Routine maintenance
- Non-urgent notifications

### Alert Thresholds

**Resource Alerts**:
- CPU: Warning at 80%, Critical at 90%
- Memory: Warning at 85%, Critical at 95%
- Disk: Warning at 80%, Critical at 95%

**Duration Settings**:
- Critical alerts: 1-2 minutes
- Warning alerts: 5-10 minutes
- Info alerts: Immediate (0s)

**Rate Limiting**:
```toml
[alerting]
# Prevent alert flooding
max_alerts_per_minute = 60
deduplication_window = "5m"
```

### Notification Routing

**By Severity**:
```toml
# Critical: PagerDuty + Slack
[[alerting.rules]]
name = "critical_alert"
severity = "critical"
labels = { notify = "pagerduty,slack" }

# Warning: Slack + Email
[[alerting.rules]]
name = "warning_alert"
severity = "warning"
labels = { notify = "slack,email" }

# Info: Email only
[[alerting.rules]]
name = "info_alert"
severity = "info"
labels = { notify = "email" }
```

**By Time of Day**:
- Business hours: Slack + Email
- After hours: PagerDuty (critical only)
- Weekends: Email + Discord

### Alert Fatigue Prevention

1. **Tune Thresholds**: Adjust based on actual patterns
2. **Use "For" Duration**: Prevent flapping
3. **Implement Silencing**: Temporary suppression during maintenance
4. **Group Related Alerts**: Use labels for grouping
5. **Regular Review**: Audit and refine alerts quarterly

### Testing Alerts

**Dry Run Mode**:
```bash
# Test alert configuration without sending notifications
wemixvisor alerts test --dry-run --config /path/to/config.toml
```

**Manual Trigger**:
```bash
# Manually trigger specific alert for testing
wemixvisor alerts trigger --name high_cpu_usage --test
```

**Configuration Validation**:
```bash
# Validate alert rule syntax
wemixvisor config validate --section alerting
```

## Troubleshooting

### Alerts Not Firing

**Check Configuration**:
```bash
# Verify alerting is enabled
grep "enabled = true" /path/to/config.toml

# Check evaluation interval
grep "evaluation_interval" /path/to/config.toml
```

**Check Logs**:
```bash
# Search for alert evaluation in logs
tail -f /var/log/wemixvisor/wemixvisor.log | grep -i alert

# Check for evaluation errors
grep "alert.*error" /var/log/wemixvisor/wemixvisor.log
```

**Verify Conditions**:
```bash
# Check current metrics
wemixvisor metrics show

# Test alert condition manually
wemixvisor alerts evaluate --name high_cpu_usage
```

### Notifications Not Sending

**Email Issues**:
```bash
# Test SMTP connection
telnet smtp.gmail.com 587

# Check password environment variable
echo $WEMIXVISOR_ALERT_EMAIL_PASSWORD

# Review email logs
grep "email" /var/log/wemixvisor/wemixvisor.log
```

**Slack/Discord Issues**:
```bash
# Test webhook manually
curl -X POST $WEMIXVISOR_ALERT_SLACK_WEBHOOK \
  -H "Content-Type: application/json" \
  -d '{"text": "Test alert"}'

# Check webhook URL format
echo $WEMIXVISOR_ALERT_SLACK_WEBHOOK

# Review notification logs
grep "slack\|discord" /var/log/wemixvisor/wemixvisor.log
```

**Webhook Issues**:
```bash
# Test webhook endpoint
curl -X POST $WEMIXVISOR_ALERT_WEBHOOK_URL \
  -H "Content-Type: application/json" \
  -d '{"test": "alert"}'

# Check timeout settings
grep "timeout" /path/to/config.toml

# Review webhook logs
grep "webhook" /var/log/wemixvisor/wemixvisor.log
```

### Alert Flapping

**Increase "For" Duration**:
```toml
[[alerting.rules]]
name = "flapping_alert"
condition = "cpu_usage > 80"
for = "10m"  # Increase from 5m to reduce flapping
```

**Adjust Thresholds**:
```toml
# Instead of exact threshold
condition = "cpu_usage > 80"

# Use wider margin
condition = "cpu_usage > 85"
```

**Add Hysteresis**:
```toml
# Fire alert
[[alerting.rules]]
name = "high_cpu"
condition = "cpu_usage > 85"
severity = "warning"
for = "5m"

# Resolve only when significantly lower
[[alerting.rules]]
name = "high_cpu_resolved"
condition = "cpu_usage < 75"
severity = "info"
for = "5m"
```

### High Alert Volume

**Implement Aggregation**:
```toml
[alerting]
# Group alerts within 5-minute window
deduplication_window = "5m"

# Rate limit to 60 alerts per minute
max_alerts_per_minute = 60
```

**Use Alert Grouping**:
```toml
[[alerting.rules]]
labels = { group = "resources", component = "system" }
```

**Review and Prune**:
```bash
# Analyze alert frequency
wemixvisor alerts stats --last 7d

# Identify noisy alerts
wemixvisor alerts top --by frequency --last 24h
```

## Production Configuration Example

```toml
[alerting]
enabled = true
evaluation_interval = "10s"
alert_retention = "7d"
max_alerts_per_minute = 60
deduplication_window = "5m"

# Email for all alerts
[alerting.email]
enabled = true
smtp_server = "smtp.company.com"
smtp_port = 587
from = "wemixvisor-alerts@company.com"
to = ["devops@company.com"]
username = "wemixvisor-alerts@company.com"

# Slack for warnings and critical
[alerting.slack]
enabled = true
channel = "#wemixvisor-alerts"
username = "Wemixvisor"

# PagerDuty for critical only (via webhook)
[alerting.webhook]
enabled = true
timeout = 10

# Critical Alerts
[[alerting.rules]]
name = "critical_disk_space"
condition = "disk_usage > 95"
severity = "critical"
for = "30s"
message = "CRITICAL: Disk almost full"
labels = { notify = "pagerduty,slack,email" }

[[alerting.rules]]
name = "node_offline"
condition = "process_state == 0"
severity = "critical"
for = "1m"
message = "CRITICAL: Node process stopped"
labels = { notify = "pagerduty,slack" }

# Warning Alerts
[[alerting.rules]]
name = "high_cpu_usage"
condition = "cpu_usage > 80"
severity = "warning"
for = "5m"
message = "High CPU usage detected"
labels = { notify = "slack,email" }

[[alerting.rules]]
name = "sync_falling_behind"
condition = "block_height < expected_height - 50"
severity = "warning"
for = "10m"
message = "Node falling behind sync"
labels = { notify = "slack,email" }

# Info Alerts
[[alerting.rules]]
name = "upgrade_available"
condition = "upgrade_pending > 0"
severity = "info"
for = "0s"
message = "New upgrade available"
labels = { notify = "email" }
```

## See Also

- [Metrics Documentation](metrics.md)
- [Grafana Dashboard Setup](grafana.md)
- [Configuration Reference](configuration.md)
- [API Documentation](api.md)
