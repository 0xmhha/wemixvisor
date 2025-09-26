# Changelog

All notable changes to Wemixvisor will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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