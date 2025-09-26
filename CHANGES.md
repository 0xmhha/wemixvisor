# Changelog

All notable changes to Wemixvisor will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.2.0] - 2024-12-26

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

## [0.1.0] - 2024-12-26

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

## [0.2.0] - TBD

### Planned
- Data backup functionality
- Pre-upgrade hooks
- Graceful shutdown with timeout
- Custom pre-upgrade scripts
- Improved error handling

## [0.3.0] - TBD

### Planned
- WBFT consensus integration
- Validator state monitoring
- Network-wide coordination
- Batch upgrade support
- Automatic binary download with checksum verification