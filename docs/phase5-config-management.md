# Phase 5: Configuration Management System

## Overview

Phase 5 introduces a comprehensive configuration management system with hot-reload capability, templates, validation, and migration support. This system provides a robust foundation for managing wemixvisor configurations across different environments.

## Features

### 1. Configuration Manager
- **Hot-reload Support**: Automatic configuration reloading using fsnotify
- **Multi-format Support**: TOML, YAML, and JSON configuration formats
- **Merged Configuration**: Separate management of Wemixvisor and Node configs
- **Update Notifications**: Real-time configuration change notifications

### 2. Template System
Built-in network templates for quick setup:
- **mainnet**: WEMIX3.0 Mainnet configuration
- **testnet**: WEMIX3.0 Testnet configuration
- **devnet**: Local development network
- **validator**: Validator node configuration
- **archive**: Archive node configuration
- **rpc**: RPC node configuration

### 3. Validation Framework
Comprehensive validation rules:
- **PathValidation**: Ensures required paths exist
- **BinaryValidation**: Validates binary names
- **PortValidation**: Port range validation (1-65535)
- **NetworkValidation**: Network configuration validation
- **ResourceValidation**: Resource limit validation
- **SecurityValidation**: Security configuration checks
- **CompatibilityValidation**: Cross-setting compatibility

### 4. Migration System
Version migration support:
- **Version Tracking**: Semantic versioning (v0.1.0 to v0.5.0)
- **Automatic Migration**: Step-by-step migration between versions
- **Backup Support**: Automatic backup before migration
- **Rollback Capability**: Restore previous configuration on failure

### 5. CLI Commands
New configuration management commands:

```bash
# Configuration Commands
wemixvisor config show              # Display current configuration
wemixvisor config show --format json # JSON format output
wemixvisor config set <key> <value> # Update configuration value
wemixvisor config validate          # Validate configuration
wemixvisor config template list     # List available templates
wemixvisor config template mainnet  # Apply mainnet template
wemixvisor config migrate           # Migrate configuration to latest version

# Backup Commands
wemixvisor backup create            # Create manual backup
wemixvisor backup list              # List available backups
wemixvisor backup restore <name>    # Restore from backup
wemixvisor backup clean             # Clean old backups

# Core Commands
wemixvisor init <binary-path>       # Initialize wemixvisor
wemixvisor start                    # Start node
wemixvisor run                      # Run node in foreground
wemixvisor stop                     # Stop node
wemixvisor restart                  # Restart node
wemixvisor status                   # Show node status
wemixvisor version                  # Show version information
```

## Architecture

### Configuration Structure
```go
type Config struct {
    // Wemixvisor settings
    Home                  string
    Name                  string
    RestartAfterUpgrade   bool
    AllowDownloadBinaries bool
    UnsafeSkipBackup      bool

    // Node settings
    NetworkID     uint64
    ChainID       string
    RPCPort       int
    WSPort        int
    P2PPort       int
    ValidatorMode bool

    // Management settings
    MaxRestarts         int
    HealthCheckInterval time.Duration
    MetricsInterval     time.Duration
    ShutdownGrace       time.Duration

    // Version tracking
    ConfigVersion string
}
```

### Manager Architecture
```
ConfigManager
├── WemixvisorConfig  # Core wemixvisor settings
├── NodeConfig        # Node-specific settings
├── MergedConfig      # Combined configuration
├── TemplateManager   # Template management
├── Validator         # Configuration validation
├── Migrator          # Version migration
└── FileWatcher       # Hot-reload support
```

## Usage Examples

### Initialize with Template
```bash
# Initialize with mainnet configuration
wemixvisor init /path/to/geth --template mainnet

# Initialize with validator configuration
wemixvisor init /path/to/geth --template validator
```

### Configuration Management
```bash
# View current configuration
wemixvisor config show

# Update configuration
wemixvisor config set max_restarts 10
wemixvisor config set validator_mode true

# Validate configuration
wemixvisor config validate

# Apply template
wemixvisor config template testnet
```

### Backup Operations
```bash
# Create backup before upgrade
wemixvisor backup create

# List backups
wemixvisor backup list

# Restore from backup
wemixvisor backup restore backup-20240101-120000.tar.gz

# Clean old backups (keep last 7 days)
wemixvisor backup clean --max-age-days 7
```

## Configuration File Formats

### TOML (default)
```toml
# wemixvisor.toml
home = "/home/user/.wemixvisor"
name = "wemixd"
network_id = 1111
chain_id = "1111"
rpc_port = 8588
validator_mode = true
max_restarts = 10

[node]
bootnodes = [
    "enode://abc123...@node1.wemix.com:8589",
    "enode://def456...@node2.wemix.com:8589"
]
```

### YAML
```yaml
# wemixvisor.yaml
home: /home/user/.wemixvisor
name: wemixd
network_id: 1111
chain_id: "1111"
rpc_port: 8588
validator_mode: true
max_restarts: 10
node:
  bootnodes:
    - enode://abc123...@node1.wemix.com:8589
    - enode://def456...@node2.wemix.com:8589
```

### JSON
```json
{
  "home": "/home/user/.wemixvisor",
  "name": "wemixd",
  "network_id": 1111,
  "chain_id": "1111",
  "rpc_port": 8588,
  "validator_mode": true,
  "max_restarts": 10,
  "node": {
    "bootnodes": [
      "enode://abc123...@node1.wemix.com:8589",
      "enode://def456...@node2.wemix.com:8589"
    ]
  }
}
```

## Testing

Phase 5 includes comprehensive test coverage:

```bash
# Run all tests
go test ./...

# Run config package tests with coverage
go test ./internal/config -cover

# Current coverage: 82.4%
```

Test coverage includes:
- Configuration loading/saving (TOML, YAML, JSON)
- Template application and management
- Validation rules and error cases
- Migration between versions
- Hot-reload functionality
- Backup and restore operations

## Migration Guide

### From v0.4.0 to v0.5.0
The migration system automatically handles:
1. Configuration structure updates
2. New field initialization with defaults
3. Version tracking
4. Backup creation

```bash
# Automatic migration
wemixvisor config migrate

# Migration with custom versions
wemixvisor config migrate --from 0.4.0 --to 0.5.0

# Migration with backup
wemixvisor config migrate --backup
```

## Best Practices

1. **Always Validate**: Run `config validate` after changes
2. **Use Templates**: Start with templates for standard configurations
3. **Create Backups**: Backup before major configuration changes
4. **Version Control**: Track configuration files in git
5. **Environment Separation**: Use different configs for different environments
6. **Monitor Changes**: Use hot-reload for dynamic updates
7. **Test Migrations**: Test configuration migrations in development first

## Troubleshooting

### Common Issues

1. **Hot-reload not working**
   - Check file permissions
   - Verify fsnotify is working on your OS
   - Check logs for watcher errors

2. **Validation failures**
   - Review validation error messages
   - Check port ranges (1-65535)
   - Verify paths exist
   - Ensure network configuration is complete

3. **Migration failures**
   - Check backup was created
   - Review migration logs
   - Restore from backup if needed
   - Manual migration may be required for custom configs

## Next Steps

Phase 5 provides the foundation for advanced configuration management. Future enhancements could include:

- Remote configuration management
- Configuration encryption
- Multi-node configuration synchronization
- Configuration drift detection
- Automated configuration optimization
- Integration with configuration management tools

## Summary

Phase 5 successfully implements a comprehensive configuration management system with:
- ✅ Hot-reload configuration support
- ✅ Multi-format configuration files
- ✅ Template system for quick setup
- ✅ Validation framework
- ✅ Version migration system
- ✅ CLI commands for management
- ✅ 82.4% test coverage

This provides a solid foundation for managing wemixvisor configurations across different environments and use cases.