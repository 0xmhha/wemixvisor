# Phase 3: Advanced Features

## Overview

Phase 3 introduces advanced features for Wemixvisor including automatic binary downloads, batch upgrade support, and WBFT consensus integration. These features enable sophisticated upgrade orchestration and improved operational efficiency.

## Features

### 1. Automatic Binary Download

The automatic binary download feature allows Wemixvisor to fetch and verify upgrade binaries from configured URLs.

#### Configuration

```yaml
daemon_allow_download_binaries: true
unsafe_skip_checksum: false  # Always verify checksums in production
download_urls:
  v2.0.0: "https://example.com/releases/v2.0.0/wemixd"
  v3.0.0: "https://example.com/releases/v3.0.0/wemixd"
  default: "https://example.com/releases/{version}/wemixd"
```

#### Features
- HTTP/HTTPS download support
- SHA256/SHA512 checksum verification
- Retry mechanism with exponential backoff
- Progress reporting during downloads
- Atomic file operations for safety

#### Usage

When an upgrade is triggered, Wemixvisor will:
1. Check if the binary exists locally
2. If not found and downloads are enabled, fetch from configured URL
3. Download the checksum file (appends .sha256 or .sha512 to URL)
4. Verify the downloaded binary against the checksum
5. Make the binary executable
6. Proceed with the upgrade

### 2. Batch Upgrade Support

Batch upgrades allow planning and executing multiple upgrades at predetermined block heights.

#### Creating a Batch Plan

```go
// Example batch plan structure
plan := &UpgradePlan{
    Name:        "q4-2025-upgrades",
    Description: "Q4 2025 scheduled upgrades",
    Upgrades: []UpgradeInfo{
        {Name: "v2.0.0", Height: 1000000},
        {Name: "v2.1.0", Height: 1500000},
        {Name: "v3.0.0", Height: 2000000},
    },
}
```

#### Plan Management

- **Create Plan**: Define multiple upgrades with their target heights
- **Save Plan**: Store plan to disk for persistence
- **Load Plan**: Retrieve saved plans for execution
- **Validate Plan**: Ensure plan integrity and ordering
- **Execute Plan**: Apply upgrades at scheduled heights

#### Plan Files

Plans are stored in JSON format in `$HOME/wemixvisor/plans/`:
```json
{
  "version": "1.0",
  "name": "q4-2025-upgrades",
  "description": "Q4 2025 scheduled upgrades",
  "created_at": "2025-09-26T10:00:00Z",
  "upgrades": [
    {
      "name": "v2.0.0",
      "height": 1000000
    },
    {
      "name": "v2.1.0",
      "height": 1500000
    }
  ]
}
```

#### Status Tracking

Monitor plan progress:
```json
{
  "name": "q4-2025-upgrades",
  "total_upgrades": 3,
  "completed": 1,
  "pending": 2,
  "active": "v2.1.0",
  "current_height": 1200000,
  "progress": 33.33
}
```

### 3. WBFT Consensus Integration

WBFT (Wemix Byzantine Fault Tolerance) integration enables consensus-aware upgrades.

#### Configuration

```yaml
daemon_rpc_address: "localhost:8545"
validator_mode: true  # Enable for validator nodes
```

#### Consensus Monitoring

The WBFT client monitors:
- Current block height
- Consensus round and step
- Validator participation
- Sync status
- Network state

#### Validator Coordination

For validator nodes:
1. Monitor consensus participation
2. Wait for stable consensus rounds
3. Coordinate upgrade timing
4. Ensure minimal disruption

#### Height-Based Coordination

```go
// Wait for specific upgrade height
coordinator.WaitForUpgradeHeight(ctx, &UpgradeInfo{
    Name:   "v2.0.0",
    Height: 1000000,
})
```

#### Consensus State

Monitor real-time consensus state:
```go
state := &ConsensusState{
    Height:         1000000,
    Round:          0,
    Step:           "commit",
    IsValidator:    true,
    ValidatorPower: 100,
    TotalPower:     1000,
}
```

## Architecture

### Component Integration

```
┌─────────────────────────────────────────┐
│           Process Manager               │
│                                         │
│  ┌───────────┐  ┌──────────────────┐  │
│  │Downloader │  │  Batch Manager    │  │
│  └───────────┘  └──────────────────┘  │
│        │               │               │
│        └───────┬───────┘               │
│                │                       │
│         ┌──────────────┐               │
│         │WBFT Coordinator│             │
│         └──────────────┘               │
│                │                       │
│         ┌──────────────┐               │
│         │ WBFT Client │                │
│         └──────────────┘               │
└─────────────────────────────────────────┘
```

### Data Flow

1. **Upgrade Detection**
   - File watcher detects upgrade-info.json
   - Or batch plan schedules upgrade at height

2. **Binary Preparation**
   - Check local binary existence
   - Download if necessary
   - Verify checksum

3. **Consensus Coordination**
   - Monitor block height
   - Check validator status
   - Wait for consensus participation

4. **Upgrade Execution**
   - Graceful shutdown
   - Backup data
   - Run pre-upgrade hooks
   - Switch binary
   - Restart process

## Testing

### Unit Tests

All Phase 3 features include comprehensive unit tests:

```bash
# Test automatic downloads
go test ./internal/download/...

# Test batch upgrades
go test ./internal/batch/...

# Test WBFT integration
go test ./internal/wbft/...
```

### Integration Testing

Test complete upgrade workflow:
1. Create batch plan
2. Prepare binaries
3. Monitor consensus
4. Execute upgrades

## Security Considerations

### Download Security
- Always verify checksums
- Use HTTPS for downloads
- Validate binary permissions
- Atomic file operations

### Consensus Security
- Validate RPC responses
- Handle network failures gracefully
- Implement timeout mechanisms
- Secure validator operations

### Batch Upgrade Security
- Validate plan integrity
- Ensure height ordering
- Prevent duplicate upgrades
- Secure plan storage

## Configuration Reference

### Download Configuration

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `daemon_allow_download_binaries` | bool | false | Enable automatic downloads |
| `unsafe_skip_checksum` | bool | false | Skip checksum verification |
| `download_urls` | map | {} | URL mappings for binaries |

### Batch Configuration

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| Plans stored in | string | `$HOME/wemixvisor/plans/` | Plan storage directory |

### WBFT Configuration

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `daemon_rpc_address` | string | localhost:8545 | RPC endpoint |
| `validator_mode` | bool | false | Enable validator features |

## Best Practices

### For Downloads
1. Always use checksums in production
2. Configure multiple download URLs for redundancy
3. Test download URLs before deployment
4. Monitor download progress and errors

### For Batch Upgrades
1. Validate plans before execution
2. Test upgrades in staging first
3. Keep backup plans ready
4. Monitor plan progress actively

### For WBFT Integration
1. Ensure stable RPC connection
2. Monitor consensus continuously
3. Coordinate with other validators
4. Plan upgrades during low-activity periods

## Troubleshooting

### Download Issues
- Check network connectivity
- Verify URL accessibility
- Ensure checksum files exist
- Check disk space

### Batch Upgrade Issues
- Validate plan format
- Check height ordering
- Verify binary availability
- Monitor execution logs

### WBFT Issues
- Verify RPC endpoint
- Check node sync status
- Monitor consensus participation
- Review validator configuration

## Future Enhancements

Potential improvements for future phases:
- Multi-region download CDN support
- Encrypted binary downloads
- Distributed upgrade coordination
- Automated rollback mechanisms
- Performance metrics collection
- Enhanced validator coordination protocols