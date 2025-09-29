# Changelog

All notable changes to Wemixvisor will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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