# Governance Integration Examples

This directory contains example implementations demonstrating how to use the Wemixvisor governance integration features.

## Examples

### 1. Basic Monitor (`basic_monitor.go`)
Demonstrates basic governance monitoring setup with periodic status reporting.

**Features:**
- Monitor initialization and startup
- Periodic status checks
- Proposal counting and filtering
- Upgrade queue display
- Graceful shutdown handling

**Usage:**
```bash
# Set environment variables
export WEMIXVISOR_HOME=/path/to/wemixvisor
export WEMIXVISOR_RPC=http://localhost:8545

# Run the example
go run basic_monitor.go
```

### 2. Custom Notification Handlers (`custom_handler.go`)
Shows how to implement custom notification handlers for governance events.

**Handlers Implemented:**
- **Console Handler**: Colored terminal output
- **File Handler**: JSON line logging to file
- **Webhook Handler**: HTTP webhook notifications

**Features:**
- Custom handler interface implementation
- Event-based notification processing
- Multiple handler registration
- Priority-based notification handling

**Usage:**
```bash
# Optional: Set webhook URL
export GOVERNANCE_WEBHOOK_URL=https://your-webhook.com/notify
export GOVERNANCE_LOG_FILE=/var/log/governance.log

# Run the example
go run custom_handler.go
```

### 3. Upgrade Manager (`upgrade_manager.go`)
Demonstrates comprehensive upgrade management with automatic scheduling and execution.

**Features:**
- Automatic upgrade detection
- Binary download coordination
- Upgrade preparation and validation
- Status tracking and reporting
- Schedule visualization

**Usage:**
```bash
# Run the upgrade manager
go run upgrade_manager.go
```

## Configuration

All examples support the following environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `WEMIXVISOR_HOME` | Wemixvisor home directory | `/tmp/wemixvisor` |
| `WEMIXVISOR_RPC` | WBFT RPC endpoint | `http://localhost:8545` |
| `GOVERNANCE_WEBHOOK_URL` | Webhook URL for notifications | (optional) |
| `GOVERNANCE_LOG_FILE` | Log file path | `/tmp/governance.log` |

## Building Examples

To build all examples:

```bash
# Build all examples
go build -o bin/basic_monitor basic_monitor.go
go build -o bin/custom_handler custom_handler.go
go build -o bin/upgrade_manager upgrade_manager.go
```

## Testing with Mock Data

For testing without a real blockchain, you can use the mock client:

```go
import "github.com/wemix/wemixvisor/internal/governance"

// Create mock client
mockClient := &governance.MockWBFTClient{}

// Setup mock responses
mockClient.On("GetGovernanceProposals", governance.ProposalStatusVoting).
    Return([]*governance.Proposal{
        {
            ID:     "1",
            Title:  "Test Proposal",
            Status: governance.ProposalStatusVoting,
        },
    }, nil)
```

## Integration Patterns

### Pattern 1: Event-Driven Architecture
```go
// React to specific governance events
func handleGovernanceEvent(event governance.NotificationEvent, data interface{}) {
    switch event {
    case governance.EventProposalPassed:
        // Handle passed proposal
    case governance.EventUpgradeScheduled:
        // Prepare for upgrade
    }
}
```

### Pattern 2: Polling-Based Monitoring
```go
// Periodic state checking
ticker := time.NewTicker(30 * time.Second)
for range ticker.C {
    proposals, _ := monitor.GetProposals()
    // Process proposals
}
```

### Pattern 3: Webhook Integration
```go
// Send governance events to external systems
type WebhookPayload struct {
    Event     string      `json:"event"`
    Timestamp time.Time   `json:"timestamp"`
    Data      interface{} `json:"data"`
}

func sendWebhook(url string, payload WebhookPayload) error {
    data, _ := json.Marshal(payload)
    resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
    // Handle response
}
```

## Common Use Cases

### 1. Validator Notification System
Monitor governance proposals and notify validators when their participation is required.

### 2. Automated Upgrade Management
Automatically schedule and execute node upgrades based on passed proposals.

### 3. Governance Analytics
Collect and analyze voting patterns, proposal success rates, and participation metrics.

### 4. Compliance Reporting
Generate reports on governance activities for regulatory compliance.

### 5. Multi-Node Coordination
Coordinate upgrades across multiple nodes in a validator infrastructure.

## Troubleshooting

### Issue: Cannot connect to RPC
**Solution**: Verify the RPC endpoint is accessible and the node is running.

### Issue: No proposals detected
**Solution**: Check if governance module is enabled on the chain.

### Issue: Notifications not received
**Solution**: Verify handler is enabled and properly configured.

## Additional Resources

- [Governance Documentation](../../docs/governance.md)
- [API Reference](../../docs/governance-api.md)
- [WBFT Documentation](https://github.com/wemix/go-wemix)

## Support

For issues or questions:
- Open an issue on [GitHub](https://github.com/wemix/wemixvisor/issues)
- Contact the Wemix development team