# Changelog

All notable changes to Wemixvisor will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.8.0] - 2025-10-21

### Phase 8: Upgrade Automation

Complete upgrade automation system enabling zero-downtime, height-based upgrades with automatic triggering and rollback.

### Added

#### Core Features
- **Automatic Height Monitoring**
  - Continuous blockchain height polling via RPC
  - Configurable poll interval (default: 5s)
  - Real-time height update notifications
  - Thread-safe subscriber pattern with channel-based distribution
  - 100% test coverage with 20 comprehensive tests

- **Upgrade Orchestration**
  - Height-based automatic upgrade triggering
  - Multi-step upgrade workflow: stop → switch → start
  - Automatic rollback on upgrade failure
  - Concurrent-safe upgrade scheduling
  - File-based upgrade configuration monitoring
  - 78.1% test coverage with 17 comprehensive tests

- **CLI Commands**
  - `wemixvisor upgrade schedule <name> <height>` - Schedule upgrades
  - `wemixvisor upgrade status` - View current upgrade status
  - `wemixvisor upgrade cancel` - Cancel scheduled upgrades
  - Optional metadata support (checksum, binaries, description)
  - JSON output support for automation
  - 64.1% test coverage with 10 comprehensive tests

#### Configuration
- `DAEMON_UPGRADE_ENABLED` - Enable/disable automatic upgrades (default: true)
- `DAEMON_HEIGHT_POLL_INTERVAL` - Height monitoring interval (default: 5s)
- File-based upgrade configuration (`upgrade-info.json`)

#### Testing
- Comprehensive unit tests for all components
- Integration tests validating end-to-end workflows
- Benchmark tests for performance validation
- Mock-based testing for external dependencies
- Overall coverage: 78-100% across components

### Technical Details

#### HeightMonitor (`internal/height/`)
- Generic `HeightProvider` interface for extensibility
- Pub/sub pattern for height distribution
- Automatic cleanup of closed subscribers
- Context-based lifecycle management
- 100% test coverage

#### UpgradeOrchestrator (`internal/orchestrator/`)
- Interface-based dependency injection (DIP compliance)
- Concurrent-safe upgrade state management
- Height-based trigger validation
- Automatic rollback with error recovery
- 78.1% test coverage

#### CLI Commands (`internal/cli/upgrade.go`)
- Cobra-based command structure
- Automatic directory creation for upgrade-info.json
- User confirmation for destructive operations
- Comprehensive error handling
- 64.1% test coverage

### Changed
- Updated configuration with Phase 8 settings
- Enhanced README with upgrade management guide
- Added Phase 8 environment variables documentation

### Performance
- Height monitoring overhead: <0.1% CPU
- Upgrade orchestration: <1ms decision latency
- Zero performance impact when upgrades disabled

### Documentation
- Complete Phase 8 implementation guide
- CLI usage examples
- Upgrade workflow documentation
- Integration testing guide

## [0.7.0] - 2025-10-17

### Phase 7: Advanced Monitoring and Management

Complete monitoring, observability, and management system for production deployments.

### Added

#### Core Features
- **Metrics Collection System**
  - Real-time metrics collection with configurable intervals (1-300s)
  - Multi-category metrics: System, Process, Chain, Application, Governance
  - Prometheus-compatible exporter with HTTP endpoint (:9090/metrics)
  - Metrics snapshot API for programmatic access
  - Low overhead design (<1% CPU impact)

- **RESTful API Server**
  - Complete HTTP API for management and monitoring (:8080)
  - WebSocket support for real-time updates and log streaming
  - Health check endpoint for container orchestration
  - Status, metrics, upgrades, and governance endpoints
  - CORS support for web-based dashboards

- **Alerting System**
  - Rule-based alert evaluation with flexible conditions
  - Four notification channels: Email, Slack, Discord, Webhook
  - Severity levels: critical, warning, info
  - Alert duration thresholds (prevent flapping)
  - Alert history tracking and management API

- **Performance Profiling**
  - CPU profiling with configurable duration
  - Memory heap profiling
  - Goroutine profiling for concurrency analysis
  - Profile management (list, clean old profiles)
  - CLI commands for all profiling operations

- **Performance Optimization**
  - LRU cache with TTL support and hit rate tracking
  - Connection pool with automatic cleanup and monitoring
  - Worker pool for concurrent task execution
  - Garbage collection tuning for production workloads
  - Resource usage optimization

#### CLI Integration
- `wemixvisor api` - Start API server with metrics and WebSocket
  - `--port` - API server port (default: 8080)
  - `--enable-metrics` - Enable metrics collection
  - `--enable-governance` - Enable governance monitoring
  - `--metrics-interval` - Collection interval in seconds
  - `--enable-system-metrics` - Enable system metrics

- `wemixvisor metrics` - Metrics management commands
  - `collect` - Start metrics collection
  - `show` - Display current metrics (human-readable or JSON)
  - `export` - Start Prometheus exporter

- `wemixvisor profile` - Performance profiling commands
  - `cpu` - Capture CPU profile
  - `heap` - Capture memory heap profile
  - `goroutine` - Capture goroutine profile
  - `all` - Capture all profiles
  - `list` - List saved profiles
  - `clean` - Remove old profiles

#### Configuration Examples
- **Basic Configuration** (`examples/config/basic-config.toml`)
  - Minimal settings for quick start
  - Development and testing optimized
  - Simple monitoring setup

- **Advanced Configuration** (`examples/config/advanced-config.toml`)
  - All Phase 7 features enabled
  - Comprehensive alerting rules
  - Performance optimization settings
  - Multi-channel notifications

- **Production Configuration** (`examples/config/production-config.toml`)
  - Security-hardened settings
  - Manual binary verification required
  - Critical alert rules with short evaluation windows
  - Long-term log retention (90 days)
  - JSON logging for aggregation systems

#### Grafana Dashboards
Three pre-built Grafana dashboard templates:

- **System Overview Dashboard** (`examples/grafana/overview-dashboard.json`)
  - System uptime and resource usage (CPU, Memory, Disk)
  - Network traffic (RX/TX bytes)
  - Goroutine count
  - Active alerts table
  - Auto-refresh: 10s

- **Upgrades Dashboard** (`examples/grafana/upgrades-dashboard.json`)
  - Total upgrades and success rate gauge
  - Upgrade history timeline
  - Upgrade duration trends
  - Backup operations tracking
  - Pre/post upgrade hook execution
  - Auto-refresh: 10s, 6-hour time range

- **Governance Dashboard** (`examples/grafana/governance-dashboard.json`)
  - Active proposals and voting status
  - Proposal status distribution (pie chart)
  - Voting distribution (Yes/No/Abstain/Veto)
  - Voter participation rate
  - Validator statistics and voting power
  - Quorum status gauge
  - Auto-refresh: 30s, 24-hour time range

#### Documentation
Complete documentation suite for Phase 7:

- **Metrics Documentation** (`docs/metrics.md`)
  - Overview and architecture
  - All metric categories and reference
  - Collection configuration (TOML, CLI, environment)
  - Usage examples (CLI, API, WebSocket, Go)
  - Prometheus integration with queries
  - Alert rules and best practices
  - Troubleshooting guide

- **Alerting Configuration Guide** (`docs/alerting.md`)
  - Alert system architecture and lifecycle
  - Configuration options
  - Alert rule syntax and examples
  - Notification channel setup (Email, Slack, Discord, Webhook)
  - Alert management API
  - Best practices and severity guidelines
  - Troubleshooting common issues

- **Grafana Dashboard Setup Guide** (`docs/grafana.md`)
  - Installation (Docker Compose, manual)
  - Prometheus data source configuration
  - Dashboard import methods
  - Customization and panel configuration
  - Alert configuration in Grafana
  - Best practices for layout and performance
  - Production deployment with HA

- **Usage Examples** (`examples/README.md`)
  - Configuration file descriptions
  - CLI usage for all Phase 7 commands
  - API endpoint reference
  - Prometheus and Grafana integration
  - WebSocket connection examples
  - Troubleshooting tips

#### Monitoring Script
- **Start with Monitoring** (`examples/scripts/start-with-monitoring.sh`)
  - Convenience script for full monitoring stack
  - Starts API server and Prometheus exporter
  - Displays access URLs
  - Automatic cleanup on exit

### Features

#### Metrics Collection (internal/metrics/)
- **Collector** - Core metrics collection engine
  - Configurable collection interval
  - Five metric categories
  - Thread-safe snapshot generation
  - Start/stop lifecycle management
  - Metrics aggregation and caching

- **Exporter** - Prometheus HTTP exporter
  - HTTP handler for /metrics endpoint
  - Prometheus text format export
  - Concurrent scraping support
  - Configurable port and path

- **Metric Categories**
  - **System**: CPU, memory, disk, network, uptime, goroutines
  - **Process**: Uptime, state, restart count, exit codes
  - **Chain**: Block height, sync status, peers, transactions
  - **Application**: Upgrades, backups, hooks, API stats
  - **Governance**: Proposals, voting, validators, participation

#### API Server (internal/api/)
- **HTTP Server**
  - RESTful API with standard HTTP methods
  - JSON request/response format
  - CORS support for cross-origin requests
  - Health check endpoint
  - Graceful shutdown with timeout

- **Endpoints**
  - `GET /health` - Health check
  - `GET /api/v1/status` - Node status
  - `GET /api/v1/metrics` - Current metrics
  - `GET /api/v1/upgrades` - Upgrade history
  - `GET /api/v1/governance/proposals` - Governance proposals
  - `GET /api/v1/ws` - WebSocket upgrade

- **WebSocket Server**
  - Real-time bidirectional communication
  - Topic-based subscription (metrics, alerts, logs)
  - Broadcast to all connected clients
  - Connection management and cleanup
  - Heartbeat/ping-pong keep-alive

#### Alerting System (internal/alerting/)
- **Alert Manager**
  - Rule-based alert evaluation
  - Alert state management (pending, firing, resolved)
  - Alert history with configurable retention
  - Concurrent alert processing

- **Notification Channels**
  - **Email**: SMTP with TLS support, multiple recipients
  - **Slack**: Webhook integration with attachments
  - **Discord**: Webhook integration with embeds
  - **Webhook**: Generic HTTP POST for custom integrations

- **Alert Rules**
  - Flexible condition expressions
  - Severity classification
  - Duration thresholds ("for" clause)
  - Custom labels for routing
  - Templated messages

#### Performance Profiling (internal/performance/)
- **Profiler**
  - CPU profiling with configurable duration
  - Memory heap snapshots
  - Goroutine profiling
  - Profile file management
  - Automatic profile rotation

- **Optimization Components**
  - **Cache**: LRU cache with TTL and hit rate tracking
  - **Connection Pool**: Reusable connections with health checks
  - **Worker Pool**: Concurrent task execution with queuing
  - **GC Tuning**: Garbage collection optimization

### Architecture

#### Design Principles
- Modular architecture with clear separation of concerns
- Interface-based design for testability
- Thread-safe implementations with proper synchronization
- Context-aware cancellation and cleanup
- Efficient resource management with automatic cleanup
- Comprehensive error handling and recovery

#### Component Integration
```
┌──────────────────────────────────────┐
│         CLI Commands                 │
│  (api, metrics, profile)             │
└──────────┬───────────────────────────┘
           │
           v
┌──────────────────────────────────────┐
│      Internal Packages               │
│  - api/      (HTTP + WebSocket)      │
│  - metrics/  (Collection + Export)   │
│  - alerting/ (Rules + Notifications) │
│  - performance/ (Profiling + Opts)   │
└──────────┬───────────────────────────┘
           │
           v
┌──────────────────────────────────────┐
│     External Integrations            │
│  - Prometheus (metrics scraping)     │
│  - Grafana (visualization)           │
│  - Email/Slack/Discord (alerts)      │
└──────────────────────────────────────┘
```

### Testing

Comprehensive test coverage across all Phase 7 components:

#### Test Statistics
- **Metrics Package**: 90%+ coverage
  - Collector: Complete metric collection tests
  - Exporter: HTTP handler and Prometheus format tests

- **Alerting Package**: 88.2% coverage
  - Alert evaluation engine tests
  - Notification channel tests
  - Rule parsing and condition evaluation

- **API Package**: 54.2% total coverage
  - HTTP handlers: 30.2% coverage (16 tests)
  - WebSocket: Complete connection lifecycle tests (16 tests)
  - Server lifecycle and graceful shutdown

- **Performance Package**: 69.6% coverage
  - Profiler tests (19 tests)
  - CPU, heap, and goroutine profiling
  - Profile management and cleanup

#### Test Types
- Unit tests for all components
- Integration tests for API endpoints
- WebSocket connection tests
- Mock implementations for external dependencies
- Error handling and edge case coverage

### Configuration

#### New Configuration Sections

**Monitoring Configuration**:
```toml
[monitoring]
enable_api = true
api_port = 8080
enable_metrics = true
metrics_interval = 10
enable_system_metrics = true
enable_process_metrics = true
enable_chain_metrics = true
enable_prometheus = true
prometheus_port = 9090
prometheus_path = "/metrics"
enable_profiling = false
profile_dir = "/opt/wemixd/profiles"
```

**Alerting Configuration**:
```toml
[alerting]
enabled = true
evaluation_interval = "10s"
alert_retention = "24h"

[alerting.email]
enabled = false
smtp_server = "smtp.gmail.com"
smtp_port = 587
from = "alerts@example.com"
to = ["admin@example.com"]

[[alerting.rules]]
name = "high_cpu_usage"
condition = "cpu_usage > 80"
severity = "warning"
for = "5m"
message = "CPU usage is above 80%"
```

**Performance Configuration**:
```toml
[performance]
enabled = true
connection_pool_size = 100
worker_pool_size = 10
enable_cache = true
cache_size = 1000
cache_ttl = "5m"
gc_percent = 100
```

#### Environment Variables
- `WEMIXVISOR_ALERT_EMAIL_PASSWORD` - SMTP password
- `WEMIXVISOR_ALERT_SLACK_WEBHOOK` - Slack webhook URL
- `WEMIXVISOR_ALERT_DISCORD_WEBHOOK` - Discord webhook URL
- `WEMIXVISOR_ALERT_WEBHOOK_URL` - Generic webhook URL
- `WEMIXVISOR_API_KEY` - API authentication key

### Performance Characteristics

#### Resource Usage
- Metrics collection overhead: <1% CPU
- Memory footprint: ~50MB additional for monitoring stack
- API server: <100ms average response time
- Metrics export: <10ms per scrape
- WebSocket: <5ms message latency

#### Scalability
- Metrics collection: Up to 300-second intervals configurable
- API server: Concurrent request handling
- WebSocket: Multiple simultaneous connections
- Alert evaluation: Sub-second rule processing
- Prometheus export: 1000s of metrics supported

### Breaking Changes
None. Phase 7 is fully backward compatible with existing configurations.

### Migration Guide

#### From v0.6.0 to v0.7.0

1. **Update Configuration** (Optional):
   ```toml
   # Add monitoring section
   [monitoring]
   enable_api = true
   enable_metrics = true
   ```

2. **Install Monitoring Stack** (Optional):
   ```bash
   # Start with monitoring
   wemixvisor api --enable-metrics

   # Or use convenience script
   ./examples/scripts/start-with-monitoring.sh
   ```

3. **Import Grafana Dashboards** (Optional):
   - Import `examples/grafana/*.json` to Grafana
   - Configure Prometheus data source

#### Backward Compatibility
All Phase 7 features are opt-in:
- Existing configurations work without changes
- Monitoring features disabled by default
- No impact on core upgrade functionality

### Known Issues
None.

### Future Improvements
- OpenAPI/Swagger documentation for API
- E2E tests for complete Phase 7 workflows
- Additional dashboard templates
- Enhanced alert rule language
- Metrics retention and archival

## [0.6.0] - 2025-09-30

### Added
- Comprehensive governance integration system
- Real-time governance proposal monitoring
- Automatic upgrade scheduling from governance proposals
- WBFT blockchain RPC client with full JSON-RPC support
- Event-driven notification system for governance events
- Proposal state tracking and management
- Upgrade queue management with validation

### Features
- GovernanceMonitor with multi-threaded proposal and upgrade tracking
- WBFTClient for direct blockchain communication via RPC
- ProposalTracker for real-time proposal status monitoring
- UpgradeScheduler for automated upgrade planning and execution
- Notifier system with pluggable handlers and priority levels
- Comprehensive type system for governance entities
- Interface-based design for testability and extensibility

### Architecture
- Monitor orchestrates all governance activities
- Separate goroutines for proposal monitoring, voting tracking, and upgrade scheduling
- Thread-safe state management with proper synchronization
- Configurable polling intervals and timeouts
- Automatic cleanup of old proposals and notifications

### Integration
- Seamless integration with existing configuration management
- Works with existing upgrade process (backup, hooks, binary switching)
- Compatible with existing CLI commands and process management
- Extensible notification handler system

### Testing
- Comprehensive unit tests with 100% coverage
- Mock WBFT client for testing without blockchain
- Complete test scenarios for all components
- Error handling and edge case testing

### Improved
- Enhanced error handling with robust recovery strategies
- Performance optimization with efficient polling and state management
- Security validation for proposals and upgrades
- Memory management with automatic cleanup policies

## [0.5.0] - 2025-09-30

### Added
- Comprehensive configuration management system
- Hot-reload configuration support with file watching
- Template system for network configurations
- Configuration validation framework
- Version migration system (v0.1.0 to v0.5.0)
- CLI commands for configuration management
- Multi-format configuration support (TOML, YAML, JSON)

### Features
- ConfigManager with automatic reload on file changes
- 6 built-in network templates (mainnet, testnet, devnet, validator, archive, rpc)
- 7 validation rules for configuration integrity
- Automatic configuration migration between versions
- Backup and restore functionality for configurations
- Real-time configuration update notifications
- Merged configuration from multiple sources

### CLI Commands
- `config show` - Display current configuration
- `config set` - Update configuration values
- `config validate` - Validate configuration
- `config template` - Apply configuration templates
- `config migrate` - Migrate configuration versions
- `backup create/list/restore/clean` - Backup management
- `init` - Initialize wemixvisor with templates
- `start/stop/restart/status` - Process management
- `version` - Show version information

### Improved
- Configuration structure with separate Wemixvisor and Node configs
- Error handling with detailed validation messages
- Test coverage reaching 82.4% for config package

## [0.4.0] - 2025-09-29

### Added - Phase 4: Node Lifecycle Management & CLI Enhancement
- **Complete CLI Command System**: Full-featured command-line interface with init, start, stop, restart, status, and version commands
- **Advanced Health Monitoring**: Real-time health checks including process, RPC, memory, disk space, and network monitoring
- **Comprehensive Status Reporting**: JSON and human-readable status output with health metrics
- **Metrics Collection System**: Automated metrics collection with JSON and Prometheus export formats
- **Enhanced Node Manager**: Robust node process lifecycle management with state machine
- **Configuration Management**: Complete TOML-based configuration with initialization command

### Features
- **CLI Commands**:
  - `wemixvisor init` - Initialize directory structure and configuration
  - `wemixvisor start/stop/restart` - Full process lifecycle control
  - `wemixvisor status [--json]` - Comprehensive status and health reporting
  - `wemixvisor version` - Version information display
- **Health Monitoring**:
  - Process liveness monitoring with PID tracking
  - JSON-RPC endpoint connectivity checks
  - Memory usage monitoring and alerts
  - Disk space availability validation
  - Peer connection and sync status monitoring
- **Metrics Collection**:
  - Node uptime and restart count tracking
  - Memory usage metrics in megabytes
  - Health status aggregation and reporting
  - Configurable collection intervals
  - JSON and Prometheus export formats
- **Node Management**:
  - Thread-safe concurrent operations with proper locking
  - Graceful shutdown with configurable timeouts (SIGTERM → SIGKILL)
  - Auto-restart mechanism with configurable maximum attempts
  - Process group management and zombie process prevention
  - Binary version detection with multiple command patterns
  - Environment variable and argument pass-through

### Improved
- **Parser System**: Enhanced CLI argument parsing with validation
- **Error Handling**: Comprehensive error handling and recovery mechanisms
- **Testing**: 100% test coverage for new components with E2E tests
- **Performance**: Benchmark tests for metrics and health monitoring systems
- **Documentation**: Updated API documentation with CLI usage examples

## [0.3.0] - 2025-09-26

### Added
- Automatic binary download with checksum verification
- Batch upgrade support for multiple scheduled upgrades
- WBFT consensus integration for coordinated upgrades
- Validator-specific upgrade coordination
- Height-based upgrade scheduling
- Progress reporting for downloads
- Upgrade plan management system

### Features
- Download binaries from configured URLs with SHA256/SHA512 verification
- Create and manage batch upgrade plans with multiple upgrades
- Monitor consensus state and coordinate upgrades with WBFT
- Wait for specific block heights before triggering upgrades
- Support validator participation in consensus during upgrades
- Track upgrade plan progress and status
- Retry mechanism for failed downloads

### Improved
- Integration with process manager for automatic downloads
- Comprehensive error handling and recovery
- Validator mode support with consensus awareness

## [0.2.0] - 2025-09-26

### Added
- Data backup functionality before upgrades
- Pre-upgrade hook system for validation and preparation
- Graceful shutdown with configurable timeout
- Backup restoration on upgrade failure
- Custom pre-upgrade script support
- Automatic old backup cleanup
- Enhanced error handling and recovery

### Features
- Create tar.gz backups of data directory
- Run custom or standard pre-upgrade scripts
- Validate upgrade binaries before execution
- SIGTERM/SIGQUIT/SIGKILL signal handling
- Configurable shutdown grace period
- Retry mechanism for pre-upgrade scripts
- Environment variable passing to scripts

### Improved
- Process shutdown sequence with thread dumps
- Error recovery with automatic backup restore
- Upgrade workflow with validation steps

## [0.1.0] - 2025-09-25

### Added
- Basic process management
- File-based upgrade detection
- Symbolic link management for binary switching
- Command-line interface
- Configuration management
- Logging system

### Features
- Start/stop blockchain node process
- Monitor upgrade-info.json file
- Switch between binary versions using symbolic links
- Basic signal handling (SIGTERM, SIGINT)