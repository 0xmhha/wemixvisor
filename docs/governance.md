# Governance Integration

## Overview

Wemixvisor Phase 6 introduces comprehensive governance integration for WBFT-based blockchain networks. This feature enables automatic monitoring of governance proposals, upgrade scheduling, and notification systems for node operators.

## Architecture

### Core Components

#### 1. Monitor (`internal/governance/monitor.go`)
The central orchestrator that coordinates all governance-related activities.

**Key Features:**
- Real-time proposal monitoring
- Voting progress tracking
- Automatic upgrade scheduling
- Event-driven notifications

**Configuration:**
```go
type Monitor struct {
    cfg          *config.Config
    rpcClient    WBFTClient
    tracker      *ProposalTracker
    scheduler    *UpgradeScheduler
    notifier     *Notifier
    pollInterval time.Duration
}
```

#### 2. WBFT Client (`internal/governance/client.go`)
RPC client for interacting with WBFT blockchain nodes.

**Supported Methods:**
- `GetCurrentHeight()` - Get current blockchain height
- `GetBlock(height)` - Fetch specific block information
- `GetGovernanceProposals(status)` - Query proposals by status
- `GetProposal(id)` - Get specific proposal details
- `GetValidators()` - List active validators
- `GetGovernanceParams()` - Fetch governance parameters

#### 3. Proposal Tracker (`internal/governance/proposals.go`)
Manages and tracks governance proposals throughout their lifecycle.

**Proposal Lifecycle:**
```
Submitted → Voting → Passed/Rejected → Executed
```

**Features:**
- Automatic proposal discovery
- Status tracking and updates
- Voting statistics monitoring
- Historical proposal management

#### 4. Upgrade Scheduler (`internal/governance/scheduler.go`)
Manages software upgrade scheduling based on passed proposals.

**Capabilities:**
- Automatic upgrade scheduling
- Height-based trigger management
- Binary download coordination
- Upgrade validation

#### 5. Notifier (`internal/governance/notifier.go`)
Event notification system for governance activities.

**Event Types:**
- New proposal submitted
- Voting started/ended
- Quorum reached
- Proposal passed/rejected
- Upgrade scheduled/triggered/completed

## API Reference

### Monitor API

#### Starting the Monitor
```go
monitor := governance.NewMonitor(cfg, logger)
err := monitor.Start()
```

#### Stopping the Monitor
```go
err := monitor.Stop()
```

#### Querying Proposals
```go
// Get all proposals
proposals, err := monitor.GetProposals()

// Get proposals by status
proposals, err := monitor.GetProposalsByStatus(governance.ProposalStatusVoting)

// Get specific proposal
proposal, err := monitor.GetProposalByID("1")
```

#### Managing Upgrades
```go
// Get scheduled upgrades
upgrades, err := monitor.GetUpgradeQueue()

// Force sync with blockchain
err := monitor.ForceSync()
```

### Proposal Types

```go
type ProposalType string

const (
    ProposalTypeUpgrade   ProposalType = "upgrade"
    ProposalTypeParameter ProposalType = "parameter"
    ProposalTypeText      ProposalType = "text"
    ProposalTypeCommunity ProposalType = "community"
)
```

### Proposal Status

```go
type ProposalStatus string

const (
    ProposalStatusSubmitted ProposalStatus = "submitted"
    ProposalStatusVoting    ProposalStatus = "voting"
    ProposalStatusPassed    ProposalStatus = "passed"
    ProposalStatusRejected  ProposalStatus = "rejected"
    ProposalStatusExecuted  ProposalStatus = "executed"
)
```

## Configuration

### Environment Variables

```bash
# RPC endpoint for WBFT node
WEMIXVISOR_RPC_ADDRESS="http://localhost:8545"

# Governance monitoring poll interval
WEMIXVISOR_GOVERNANCE_POLL_INTERVAL="30s"

# Enable/disable governance monitoring
WEMIXVISOR_GOVERNANCE_ENABLED="true"

# Maximum proposal age to track
WEMIXVISOR_MAX_PROPOSAL_AGE="720h"  # 30 days
```

### Configuration File

```yaml
# config.yaml
governance:
  enabled: true
  rpc_address: "http://localhost:8545"
  poll_interval: "30s"
  max_proposal_age: "720h"
  notifications:
    enabled: true
    handlers:
      - type: "console"
        enabled: true
      - type: "webhook"
        enabled: false
        url: "https://example.com/webhook"
```

## Usage Examples

### Basic Monitoring Setup

```go
package main

import (
    "log"
    "github.com/wemix/wemixvisor/internal/config"
    "github.com/wemix/wemixvisor/internal/governance"
    "github.com/wemix/wemixvisor/pkg/logger"
)

func main() {
    // Load configuration
    cfg, err := config.Load()
    if err != nil {
        log.Fatal(err)
    }

    // Create logger
    logger := logger.New("governance")

    // Create and start monitor
    monitor := governance.NewMonitor(cfg, logger)

    if err := monitor.Start(); err != nil {
        log.Fatal(err)
    }
    defer monitor.Stop()

    // Monitor will run in background
    select {}
}
```

### Custom Notification Handler

```go
type CustomHandler struct {
    enabled bool
}

func (h *CustomHandler) Handle(notification *governance.Notification) error {
    log.Printf("Event: %s - %s", notification.Event, notification.Message)
    // Custom handling logic
    return nil
}

func (h *CustomHandler) IsEnabled() bool {
    return h.enabled
}

func (h *CustomHandler) GetType() string {
    return "custom"
}

// Add to notifier
notifier := governance.NewNotifier(logger)
notifier.AddHandler(&CustomHandler{enabled: true})
```

### Querying Governance State

```go
// Get all active proposals
proposals, err := monitor.GetProposals()
if err != nil {
    log.Fatal(err)
}

for _, proposal := range proposals {
    log.Printf("Proposal %s: %s (Status: %s)",
        proposal.ID,
        proposal.Title,
        proposal.Status)

    if proposal.Type == governance.ProposalTypeUpgrade {
        log.Printf("  Upgrade Height: %d", proposal.UpgradeHeight)
    }

    if proposal.VotingStats != nil {
        log.Printf("  Voting Progress: %.2f%% turnout",
            proposal.VotingStats.Turnout * 100)
    }
}
```

## Testing

The governance package includes comprehensive tests with 92.7% code coverage.

### Running Tests

```bash
# Run all tests
go test ./internal/governance/ -v

# Check coverage
go test ./internal/governance/ -cover

# Generate coverage report
go test ./internal/governance/ -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Test Categories

1. **Unit Tests**: Individual component testing
2. **Integration Tests**: Multi-component interaction testing
3. **Mock Tests**: Client and handler mocking
4. **Error Handling Tests**: Edge cases and error paths

## Security Considerations

### RPC Security
- Always use HTTPS for RPC connections in production
- Implement authentication for RPC endpoints
- Use read-only RPC endpoints when possible

### Upgrade Validation
- Verify binary checksums before upgrades
- Validate upgrade heights and timing
- Implement rollback mechanisms

### Notification Security
- Authenticate webhook endpoints
- Use TLS for all external communications
- Sanitize notification content

## Performance Optimization

### Polling Strategy
- Adjust poll interval based on network activity
- Use exponential backoff for failed requests
- Cache frequently accessed data

### Resource Management
- Limit concurrent RPC requests
- Implement connection pooling
- Set appropriate timeouts

### Memory Optimization
- Limit proposal history retention
- Implement periodic cleanup
- Use efficient data structures

## Troubleshooting

### Common Issues

#### RPC Connection Failed
```
Error: failed to initialize components: failed to create RPC client
```
**Solution**: Verify RPC endpoint is accessible and correct.

#### No Proposals Found
```
Info: No new proposals detected
```
**Solution**: Check if governance is active on the network.

#### Upgrade Not Triggered
```
Warning: Upgrade height reached but not triggered
```
**Solution**: Verify wemixvisor has write permissions to upgrade directory.

### Debug Logging

Enable debug logging for detailed information:
```bash
export WEMIXVISOR_LOG_LEVEL=debug
```

## Migration Guide

### From Manual Governance

If migrating from manual governance monitoring:

1. **Export Current State**: Document all active proposals
2. **Configure Wemixvisor**: Set up governance configuration
3. **Initial Sync**: Run force sync to populate state
4. **Verify State**: Compare with manual records
5. **Enable Automation**: Activate automatic monitoring

### Upgrade from Previous Versions

When upgrading from earlier wemixvisor versions:

1. **Backup Configuration**: Save current config files
2. **Stop Current Version**: Gracefully shutdown
3. **Update Binary**: Install new version
4. **Migrate Config**: Update configuration format if needed
5. **Restart Service**: Start with new version

## Best Practices

### Production Deployment

1. **High Availability**
   - Run multiple monitor instances (active/passive)
   - Use load balancer for RPC endpoints
   - Implement health checks

2. **Monitoring**
   - Set up metrics collection
   - Configure alerting for critical events
   - Log aggregation for troubleshooting

3. **Backup Strategy**
   - Regular state backups
   - Configuration versioning
   - Disaster recovery plan

### Development Guidelines

1. **Error Handling**
   - Always check errors
   - Provide context in error messages
   - Implement retry logic for transient failures

2. **Testing**
   - Write tests for new features
   - Maintain >90% code coverage
   - Test error scenarios

3. **Documentation**
   - Update docs with API changes
   - Include usage examples
   - Document configuration options

## Roadmap

### Phase 7 Plans
- [ ] Multi-chain governance support
- [ ] Advanced voting analytics
- [ ] Automated proposal creation
- [ ] Governance dashboard UI
- [ ] Enhanced notification channels

### Future Enhancements
- GraphQL API for governance queries
- Machine learning for proposal analysis
- Automated governance participation
- Cross-chain governance coordination

## Contributing

We welcome contributions to the governance integration system. Please follow these guidelines:

1. **Code Style**: Follow Go best practices and existing patterns
2. **Testing**: Include tests for new features
3. **Documentation**: Update relevant documentation
4. **Commit Messages**: Use conventional commit format

## License

Copyright 2024 Wemix Foundation

Licensed under the MIT License.