# Metrics Documentation

Complete guide to wemixvisor metrics collection, monitoring, and analysis.

## Table of Contents

- [Overview](#overview)
- [Metrics Categories](#metrics-categories)
- [Collection Configuration](#collection-configuration)
- [Metrics Reference](#metrics-reference)
- [Usage Examples](#usage-examples)
- [Prometheus Integration](#prometheus-integration)
- [Best Practices](#best-practices)

## Overview

Wemixvisor provides comprehensive metrics collection for monitoring blockchain node operations, system resources, and application performance. Metrics are collected in real-time and can be exposed via Prometheus or accessed through the API.

### Key Features

- **Real-time Collection**: Configurable collection intervals (1-300 seconds)
- **Multi-category Metrics**: System, process, chain, application, and governance metrics
- **Prometheus Export**: Native Prometheus exporter with HTTP endpoint
- **Low Overhead**: <1% CPU overhead with optimized collection
- **Flexible Configuration**: Enable/disable specific metric categories
- **Historical Tracking**: Track metrics over time for trend analysis

### Architecture

```
┌─────────────────┐
│  Metric Sources │
│  - System       │
│  - Process      │
│  - Chain        │
│  - Application  │
│  - Governance   │
└────────┬────────┘
         │
         v
┌─────────────────┐
│   Collector     │
│  - Aggregation  │
│  - Sampling     │
│  - Caching      │
└────────┬────────┘
         │
         ├──────────────┐
         v              v
┌─────────────┐  ┌──────────────┐
│ Prometheus  │  │  API Server  │
│  Exporter   │  │  /metrics    │
└─────────────┘  └──────────────┘
```

## Metrics Categories

### System Metrics

Monitor host system resources and performance.

**Collected Metrics**:
- CPU usage percentage
- Memory usage (bytes and percentage)
- Disk usage (bytes and percentage)
- Network I/O (bytes sent/received)
- System uptime
- Goroutine count

**Use Cases**:
- Resource planning and capacity management
- Performance bottleneck identification
- System health monitoring
- Alert threshold configuration

### Process Metrics

Track wemixvisor process-specific metrics.

**Collected Metrics**:
- Process uptime
- Process state (running, stopped, restarting)
- Restart count
- Process exit codes
- Memory usage by process
- CPU usage by process

**Use Cases**:
- Process stability monitoring
- Crash detection and analysis
- Resource allocation optimization
- Upgrade impact assessment

### Chain Metrics

Monitor blockchain-specific metrics.

**Collected Metrics**:
- Block height (current and latest)
- Sync status (syncing/synced)
- Peer count
- Transaction count
- Block time
- Consensus state

**Use Cases**:
- Node synchronization monitoring
- Network health assessment
- Performance benchmarking
- Consensus issue detection

### Application Metrics

Track wemixvisor application-level operations.

**Collected Metrics**:
- Upgrade operations (total, success, failed)
- Upgrade duration
- Backup operations (total, success, failed)
- Hook execution (pre/post upgrade)
- API request count and latency
- WebSocket connections

**Use Cases**:
- Upgrade success rate tracking
- Operation performance analysis
- Backup validation
- API performance monitoring

### Governance Metrics

Monitor governance proposals and voting.

**Collected Metrics**:
- Proposal count (total, voting, passed, rejected)
- Vote distribution (yes, no, abstain, veto)
- Voter participation rate
- Validator count and status
- Voting power distribution
- Quorum status

**Use Cases**:
- Governance participation tracking
- Proposal outcome prediction
- Validator activity monitoring
- Community engagement analysis

## Collection Configuration

### TOML Configuration

```toml
[monitoring]
# Enable metrics collection
enable_metrics = true

# Collection interval in seconds (1-300)
metrics_interval = 10

# Enable specific metric categories
enable_system_metrics = true
enable_process_metrics = true
enable_chain_metrics = true

# Prometheus export
enable_prometheus = true
prometheus_port = 9090
prometheus_path = "/metrics"
```

### Environment Variables

```bash
# Override collection interval
export WEMIXVISOR_METRICS_INTERVAL=15

# Disable specific categories
export WEMIXVISOR_ENABLE_SYSTEM_METRICS=false
```

### CLI Configuration

```bash
# Start with metrics enabled
wemixvisor api --enable-metrics --metrics-interval 10

# Start metrics collector
wemixvisor metrics collect --interval 10 --duration 60

# Start Prometheus exporter
wemixvisor metrics export --port 9090
```

## Metrics Reference

### Metric Naming Convention

All metrics follow the pattern: `wemixvisor_<category>_<metric>_<unit>`

**Categories**: `system`, `process`, `chain`, `application`, `governance`
**Units**: `total`, `bytes`, `seconds`, `percent`

### System Metrics

| Metric Name | Type | Unit | Description |
|-------------|------|------|-------------|
| `wemixvisor_system_uptime_seconds` | Gauge | seconds | System uptime |
| `wemixvisor_system_cpu_usage_percent` | Gauge | percent | CPU usage percentage |
| `wemixvisor_system_memory_usage_bytes` | Gauge | bytes | Memory usage in bytes |
| `wemixvisor_system_memory_usage_percent` | Gauge | percent | Memory usage percentage |
| `wemixvisor_system_disk_usage_bytes` | Gauge | bytes | Disk usage in bytes |
| `wemixvisor_system_disk_usage_percent` | Gauge | percent | Disk usage percentage |
| `wemixvisor_system_network_rx_bytes_total` | Counter | bytes | Network bytes received |
| `wemixvisor_system_network_tx_bytes_total` | Counter | bytes | Network bytes transmitted |
| `wemixvisor_system_goroutines_total` | Gauge | count | Number of goroutines |

### Process Metrics

| Metric Name | Type | Unit | Description |
|-------------|------|------|-------------|
| `wemixvisor_process_uptime_seconds` | Gauge | seconds | Process uptime |
| `wemixvisor_process_state` | Gauge | enum | Process state (0=stopped, 1=running, 2=restarting) |
| `wemixvisor_process_restarts_total` | Counter | count | Total process restarts |
| `wemixvisor_process_memory_bytes` | Gauge | bytes | Process memory usage |
| `wemixvisor_process_cpu_percent` | Gauge | percent | Process CPU usage |

### Chain Metrics

| Metric Name | Type | Unit | Description |
|-------------|------|------|-------------|
| `wemixvisor_chain_block_height` | Gauge | count | Current block height |
| `wemixvisor_chain_latest_block_height` | Gauge | count | Latest network block height |
| `wemixvisor_chain_syncing` | Gauge | boolean | Sync status (0=synced, 1=syncing) |
| `wemixvisor_chain_peers_total` | Gauge | count | Connected peer count |
| `wemixvisor_chain_transactions_total` | Counter | count | Total transactions processed |
| `wemixvisor_chain_block_time_seconds` | Gauge | seconds | Average block time |

### Application Metrics

| Metric Name | Type | Unit | Description |
|-------------|------|------|-------------|
| `wemixvisor_upgrades_total` | Counter | count | Total upgrade operations |
| `wemixvisor_upgrades_success` | Counter | count | Successful upgrades |
| `wemixvisor_upgrades_failed` | Counter | count | Failed upgrades |
| `wemixvisor_upgrade_duration_seconds` | Histogram | seconds | Upgrade duration |
| `wemixvisor_backup_total` | Counter | count | Total backup operations |
| `wemixvisor_backup_success` | Counter | count | Successful backups |
| `wemixvisor_backup_failed` | Counter | count | Failed backups |
| `wemixvisor_hooks_executed_total` | Counter | count | Hooks executed (labeled by type) |
| `wemixvisor_api_requests_total` | Counter | count | Total API requests |
| `wemixvisor_api_request_duration_seconds` | Histogram | seconds | API request latency |
| `wemixvisor_websocket_connections` | Gauge | count | Active WebSocket connections |

### Governance Metrics

| Metric Name | Type | Unit | Description |
|-------------|------|------|-------------|
| `wemixvisor_governance_proposals_total` | Counter | count | Total proposals |
| `wemixvisor_governance_proposals_voting` | Gauge | count | Proposals in voting period |
| `wemixvisor_governance_proposals_passed` | Counter | count | Passed proposals |
| `wemixvisor_governance_proposals_rejected` | Counter | count | Rejected proposals |
| `wemixvisor_governance_votes_yes` | Gauge | count | Yes votes |
| `wemixvisor_governance_votes_no` | Gauge | count | No votes |
| `wemixvisor_governance_votes_abstain` | Gauge | count | Abstain votes |
| `wemixvisor_governance_votes_veto` | Gauge | count | No with veto votes |
| `wemixvisor_governance_voter_participation_percent` | Gauge | percent | Voter participation rate |
| `wemixvisor_governance_validators_total` | Gauge | count | Total validators |
| `wemixvisor_governance_validators_active` | Gauge | count | Active validators |
| `wemixvisor_governance_voting_power_total` | Gauge | count | Total voting power |
| `wemixvisor_governance_quorum_reached_percent` | Gauge | percent | Quorum percentage |

## Usage Examples

### CLI Usage

#### View Current Metrics

```bash
# Display all metrics in human-readable format
wemixvisor metrics show

# Display in JSON format
wemixvisor metrics show --json

# Watch metrics in real-time (refresh every 5 seconds)
wemixvisor metrics show --watch --interval 5
```

**Example Output**:
```
=== System Metrics ===
Uptime:               3h 42m 15s
CPU Usage:            45.2%
Memory Usage:         2.3 GB (23.4%)
Disk Usage:          125.8 GB (62.1%)
Network RX:           1.2 GB
Network TX:           892.5 MB
Goroutines:           147

=== Chain Metrics ===
Block Height:         1,234,567
Latest Height:        1,234,567
Syncing:              false
Peers:                42
Transactions:         5,678,901
Block Time:           5.2s

=== Application Metrics ===
Upgrades (Total):     12
Upgrades (Success):   11
Upgrades (Failed):    1
Success Rate:         91.7%
API Requests:         45,892
WebSocket Conns:      8
```

#### Collect Metrics

```bash
# Collect metrics every 10 seconds for 60 seconds
wemixvisor metrics collect --interval 10 --duration 60

# Collect continuously (until interrupted)
wemixvisor metrics collect --interval 5

# Collect with specific categories
wemixvisor metrics collect --system --chain --interval 10
```

#### Export to Prometheus

```bash
# Start Prometheus exporter on default port (9090)
wemixvisor metrics export

# Custom port and path
wemixvisor metrics export --port 9100 --path /custom-metrics

# With TLS
wemixvisor metrics export --port 9090 --tls --cert-file /path/to/cert.pem --key-file /path/to/key.pem
```

### API Usage

#### HTTP Endpoints

**Get Current Metrics**:
```bash
curl http://localhost:8080/api/v1/metrics
```

**Response**:
```json
{
  "timestamp": "2025-10-17T10:30:00Z",
  "system": {
    "uptime_seconds": 13335,
    "cpu_usage_percent": 45.2,
    "memory_usage_bytes": 2469606195,
    "memory_usage_percent": 23.4,
    "disk_usage_bytes": 135107092275,
    "disk_usage_percent": 62.1,
    "network_rx_bytes": 1288490188,
    "network_tx_bytes": 935985152,
    "goroutines": 147
  },
  "chain": {
    "block_height": 1234567,
    "latest_block_height": 1234567,
    "syncing": false,
    "peers": 42,
    "transactions": 5678901,
    "block_time_seconds": 5.2
  },
  "application": {
    "upgrades_total": 12,
    "upgrades_success": 11,
    "upgrades_failed": 1,
    "api_requests_total": 45892,
    "websocket_connections": 8
  }
}
```

**Prometheus Format**:
```bash
curl http://localhost:9090/metrics
```

**Response**:
```
# HELP wemixvisor_system_cpu_usage_percent CPU usage percentage
# TYPE wemixvisor_system_cpu_usage_percent gauge
wemixvisor_system_cpu_usage_percent 45.2

# HELP wemixvisor_system_memory_usage_bytes Memory usage in bytes
# TYPE wemixvisor_system_memory_usage_bytes gauge
wemixvisor_system_memory_usage_bytes 2469606195

# HELP wemixvisor_chain_block_height Current block height
# TYPE wemixvisor_chain_block_height gauge
wemixvisor_chain_block_height 1234567
```

#### WebSocket Streaming

```javascript
// Connect to WebSocket
const ws = new WebSocket('ws://localhost:8080/api/v1/ws');

// Subscribe to metrics updates
ws.onopen = () => {
  ws.send(JSON.stringify({
    action: 'subscribe',
    topics: ['metrics']
  }));
};

// Handle metrics updates
ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  if (data.topic === 'metrics') {
    console.log('Metrics update:', data.data);
    // Update UI with new metrics
    updateDashboard(data.data);
  }
};
```

### Programmatic Usage (Go)

```go
import (
    "github.com/wemix/wemixvisor/internal/metrics"
    "github.com/wemix/wemixvisor/pkg/logger"
    "time"
)

// Create collector
cfg := &metrics.CollectorConfig{
    Enabled:             true,
    CollectionInterval:  10 * time.Second,
    EnableSystemMetrics: true,
    EnableProcessMetrics: true,
    EnableChainMetrics:  true,
}

log := logger.NewLogger("info", false, "")
collector := metrics.NewCollector(cfg, log)

// Start collection
if err := collector.Start(); err != nil {
    log.Error("Failed to start collector", "error", err)
    return
}
defer collector.Stop()

// Get current snapshot
snapshot := collector.GetSnapshot()
fmt.Printf("CPU Usage: %.2f%%\n", snapshot.System.CPUUsage)
fmt.Printf("Block Height: %d\n", snapshot.Chain.BlockHeight)

// Create Prometheus exporter
exporter := metrics.NewExporter(collector, 9090, "/metrics", log)
if err := exporter.Start(); err != nil {
    log.Error("Failed to start exporter", "error", err)
    return
}
defer exporter.Stop()
```

## Prometheus Integration

### Configuration

Add to `prometheus.yml`:

```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'wemixvisor'
    static_configs:
      - targets: ['localhost:9090']
    scrape_interval: 10s
    scrape_timeout: 5s
```

### Query Examples

**System Resource Usage**:
```promql
# Average CPU usage over 5 minutes
avg_over_time(wemixvisor_system_cpu_usage_percent[5m])

# Memory usage trend
rate(wemixvisor_system_memory_usage_bytes[1h])

# Network I/O rate
rate(wemixvisor_system_network_rx_bytes_total[5m])
```

**Chain Synchronization**:
```promql
# Blocks behind network
wemixvisor_chain_latest_block_height - wemixvisor_chain_block_height

# Sync speed (blocks per second)
rate(wemixvisor_chain_block_height[1m])

# Peer connectivity
wemixvisor_chain_peers_total
```

**Upgrade Operations**:
```promql
# Upgrade success rate
100 * wemixvisor_upgrades_success / wemixvisor_upgrades_total

# Recent upgrade failures
increase(wemixvisor_upgrades_failed[1h])

# Average upgrade duration
histogram_quantile(0.95, rate(wemixvisor_upgrade_duration_seconds_bucket[1h]))
```

**API Performance**:
```promql
# Request rate
rate(wemixvisor_api_requests_total[5m])

# P95 latency
histogram_quantile(0.95, rate(wemixvisor_api_request_duration_seconds_bucket[5m]))

# Active connections
wemixvisor_websocket_connections
```

**Governance Activity**:
```promql
# Proposal success rate
100 * wemixvisor_governance_proposals_passed / wemixvisor_governance_proposals_total

# Voting participation
wemixvisor_governance_voter_participation_percent

# Active validators
wemixvisor_governance_validators_active
```

### Alert Rules

```yaml
groups:
  - name: wemixvisor_alerts
    interval: 30s
    rules:
      # High CPU usage
      - alert: HighCPUUsage
        expr: wemixvisor_system_cpu_usage_percent > 80
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High CPU usage detected"
          description: "CPU usage is {{ $value }}% for 5 minutes"

      # Low disk space
      - alert: LowDiskSpace
        expr: wemixvisor_system_disk_usage_percent > 90
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Low disk space"
          description: "Disk usage is {{ $value }}%"

      # Node not syncing
      - alert: NodeNotSyncing
        expr: wemixvisor_chain_syncing == 1 and (wemixvisor_chain_latest_block_height - wemixvisor_chain_block_height) > 100
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "Node falling behind"
          description: "Node is {{ $value }} blocks behind"

      # Upgrade failure
      - alert: UpgradeFailed
        expr: increase(wemixvisor_upgrades_failed[5m]) > 0
        labels:
          severity: critical
        annotations:
          summary: "Upgrade operation failed"
          description: "{{ $value }} upgrade failures in last 5 minutes"

      # Low peer count
      - alert: LowPeerCount
        expr: wemixvisor_chain_peers_total < 5
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Low peer count"
          description: "Only {{ $value }} peers connected"
```

## Best Practices

### Collection Intervals

**Recommended Settings**:
- **Development**: 5-10 seconds for rapid feedback
- **Production**: 10-15 seconds for balance
- **High-traffic**: 15-30 seconds to reduce overhead
- **Low-priority**: 30-60 seconds for minimal impact

### Performance Optimization

1. **Selective Collection**: Enable only needed metric categories
2. **Caching**: Use built-in caching for expensive operations
3. **Sampling**: Consider sampling for high-frequency metrics
4. **Aggregation**: Pre-aggregate metrics where possible

```toml
[monitoring]
enable_metrics = true
metrics_interval = 10

# Enable only required categories
enable_system_metrics = true
enable_process_metrics = true
enable_chain_metrics = true
enable_application_metrics = false  # Disable if not needed
enable_governance_metrics = false   # Disable if not needed
```

### Storage and Retention

**Prometheus Configuration**:
```yaml
global:
  # Keep data for 30 days
  storage.tsdb.retention.time: 30d

  # Limit storage size
  storage.tsdb.retention.size: 50GB
```

**Data Management**:
- Archive old metrics to long-term storage
- Use recording rules for aggregated metrics
- Implement tiered storage (hot/warm/cold)

### Monitoring Coverage

**Essential Metrics**:
1. System: CPU, Memory, Disk
2. Chain: Block height, Sync status
3. Application: Upgrade success rate
4. API: Request rate, Latency

**Optional Metrics**:
1. Network: Peer count, Bandwidth
2. Governance: Proposals, Voting
3. Process: Restart count, Uptime

### Alerting Strategy

**Alert Levels**:
- **Critical**: Immediate action required (page on-call)
- **Warning**: Attention needed (email/Slack)
- **Info**: Informational only (log/dashboard)

**Alert Configuration**:
```toml
[[alerting.rules]]
name = "critical_disk_space"
condition = "disk_usage > 95"
severity = "critical"
for = "1m"
message = "CRITICAL: Disk almost full"

[[alerting.rules]]
name = "high_memory_usage"
condition = "memory_usage > 85"
severity = "warning"
for = "5m"
message = "WARNING: High memory usage"

[[alerting.rules]]
name = "upgrade_available"
condition = "upgrade_pending > 0"
severity = "info"
for = "0s"
message = "INFO: Upgrade available"
```

### Security Considerations

1. **Access Control**: Protect metrics endpoints with authentication
2. **TLS Encryption**: Use TLS for metrics transmission
3. **Rate Limiting**: Prevent metrics endpoint abuse
4. **Sensitive Data**: Never include secrets in metric labels

```toml
[security]
enable_auth = true
api_key = "${WEMIXVISOR_API_KEY}"

enable_tls = true
tls_cert = "/path/to/cert.pem"
tls_key = "/path/to/key.pem"

enable_rate_limit = true
rate_limit_requests = 100
rate_limit_window = "1m"
```

### Troubleshooting

**Metrics Not Collecting**:
1. Check configuration: `enable_metrics = true`
2. Verify collection interval is reasonable (1-300s)
3. Review logs for collection errors
4. Ensure required permissions for system metrics

**High Overhead**:
1. Increase collection interval
2. Disable unnecessary metric categories
3. Reduce Prometheus scrape frequency
4. Check for network issues

**Missing Metrics**:
1. Verify category is enabled in configuration
2. Check if node is properly configured
3. Review API connectivity for chain metrics
4. Ensure governance features are enabled

**Prometheus Issues**:
1. Verify exporter is running: `curl http://localhost:9090/metrics`
2. Check Prometheus targets: Prometheus UI → Status → Targets
3. Review Prometheus logs for scrape errors
4. Validate network connectivity and firewall rules

## See Also

- [Alerting Configuration Guide](alerting.md)
- [Grafana Dashboard Setup](grafana.md)
- [API Documentation](api.md)
- [Configuration Reference](configuration.md)
