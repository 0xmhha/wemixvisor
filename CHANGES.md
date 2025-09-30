# Changelog

All notable changes to Wemixvisor will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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