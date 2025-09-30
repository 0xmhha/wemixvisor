# Governance API Reference

## Table of Contents
- [Client API](#client-api)
- [Monitor API](#monitor-api)
- [Proposal Tracker API](#proposal-tracker-api)
- [Upgrade Scheduler API](#upgrade-scheduler-api)
- [Notifier API](#notifier-api)
- [Data Types](#data-types)

## Client API

### NewWBFTClient
Creates a new WBFT RPC client.

```go
func NewWBFTClient(rpcURL string, logger logger.Logger) (*WBFTClient, error)
```

**Parameters:**
- `rpcURL`: The RPC endpoint URL
- `logger`: Logger instance

**Returns:**
- `*WBFTClient`: Client instance
- `error`: Error if initialization fails

**Example:**
```go
client, err := governance.NewWBFTClient("http://localhost:8545", logger)
```

### GetCurrentHeight
Retrieves the current blockchain height.

```go
func (c *WBFTClient) GetCurrentHeight() (int64, error)
```

**Returns:**
- `int64`: Current block height
- `error`: Error if request fails

### GetBlock
Fetches a specific block by height.

```go
func (c *WBFTClient) GetBlock(height int64) (*BlockInfo, error)
```

**Parameters:**
- `height`: Block height to fetch

**Returns:**
- `*BlockInfo`: Block information
- `error`: Error if request fails

### GetGovernanceProposals
Queries proposals by status.

```go
func (c *WBFTClient) GetGovernanceProposals(status ProposalStatus) ([]*Proposal, error)
```

**Parameters:**
- `status`: Proposal status filter

**Returns:**
- `[]*Proposal`: List of proposals
- `error`: Error if request fails

### GetProposal
Gets details of a specific proposal.

```go
func (c *WBFTClient) GetProposal(proposalID string) (*Proposal, error)
```

**Parameters:**
- `proposalID`: Proposal identifier

**Returns:**
- `*Proposal`: Proposal details
- `error`: Error if request fails

### GetValidators
Lists active validators.

```go
func (c *WBFTClient) GetValidators() ([]*ValidatorInfo, error)
```

**Returns:**
- `[]*ValidatorInfo`: List of validators
- `error`: Error if request fails

### GetGovernanceParams
Retrieves governance parameters.

```go
func (c *WBFTClient) GetGovernanceParams() (*GovernanceParams, error)
```

**Returns:**
- `*GovernanceParams`: Governance parameters
- `error`: Error if request fails

## Monitor API

### NewMonitor
Creates a new governance monitor instance.

```go
func NewMonitor(cfg *config.Config, logger logger.Logger) *Monitor
```

**Parameters:**
- `cfg`: Configuration instance
- `logger`: Logger instance

**Returns:**
- `*Monitor`: Monitor instance

### Start
Starts the governance monitoring.

```go
func (m *Monitor) Start() error
```

**Returns:**
- `error`: Error if start fails

### Stop
Stops the governance monitoring.

```go
func (m *Monitor) Stop() error
```

**Returns:**
- `error`: Error if stop fails

### GetProposals
Retrieves all tracked proposals.

```go
func (m *Monitor) GetProposals() ([]*Proposal, error)
```

**Returns:**
- `[]*Proposal`: All proposals
- `error`: Error if retrieval fails

### GetProposalsByStatus
Gets proposals filtered by status.

```go
func (m *Monitor) GetProposalsByStatus(status ProposalStatus) ([]*Proposal, error)
```

**Parameters:**
- `status`: Status filter

**Returns:**
- `[]*Proposal`: Filtered proposals
- `error`: Error if retrieval fails

### GetUpgradeQueue
Returns scheduled upgrades.

```go
func (m *Monitor) GetUpgradeQueue() ([]*UpgradeInfo, error)
```

**Returns:**
- `[]*UpgradeInfo`: Scheduled upgrades
- `error`: Error if retrieval fails

### ForceSync
Forces synchronization with blockchain.

```go
func (m *Monitor) ForceSync() error
```

**Returns:**
- `error`: Error if sync fails

### SetEnabled
Enables or disables monitoring.

```go
func (m *Monitor) SetEnabled(enabled bool)
```

**Parameters:**
- `enabled`: Enable/disable flag

### SetPollInterval
Sets the polling interval.

```go
func (m *Monitor) SetPollInterval(interval time.Duration)
```

**Parameters:**
- `interval`: Poll interval duration

## Proposal Tracker API

### NewProposalTracker
Creates a new proposal tracker.

```go
func NewProposalTracker(client WBFTClient, logger logger.Logger) *ProposalTracker
```

**Parameters:**
- `client`: WBFT client instance
- `logger`: Logger instance

**Returns:**
- `*ProposalTracker`: Tracker instance

### Start
Starts proposal tracking.

```go
func (pt *ProposalTracker) Start() error
```

**Returns:**
- `error`: Error if start fails

### Stop
Stops proposal tracking.

```go
func (pt *ProposalTracker) Stop() error
```

**Returns:**
- `error`: Error if stop fails

### FetchLatest
Fetches latest proposals from blockchain.

```go
func (pt *ProposalTracker) FetchLatest() ([]*Proposal, error)
```

**Returns:**
- `[]*Proposal`: New proposals
- `error`: Error if fetch fails

### GetAll
Returns all tracked proposals.

```go
func (pt *ProposalTracker) GetAll() ([]*Proposal, error)
```

**Returns:**
- `[]*Proposal`: All proposals
- `error`: Error if retrieval fails

### GetByID
Gets proposal by ID.

```go
func (pt *ProposalTracker) GetByID(proposalID string) (*Proposal, error)
```

**Parameters:**
- `proposalID`: Proposal ID

**Returns:**
- `*Proposal`: Proposal details
- `error`: Error if not found

### GetByStatus
Gets proposals by status.

```go
func (pt *ProposalTracker) GetByStatus(status ProposalStatus) ([]*Proposal, error)
```

**Parameters:**
- `status`: Status filter

**Returns:**
- `[]*Proposal`: Filtered proposals
- `error`: Error if retrieval fails

### GetByType
Gets proposals by type.

```go
func (pt *ProposalTracker) GetByType(proposalType ProposalType) ([]*Proposal, error)
```

**Parameters:**
- `proposalType`: Type filter

**Returns:**
- `[]*Proposal`: Filtered proposals
- `error`: Error if retrieval fails

### UpdateVotingStatus
Updates voting statistics for a proposal.

```go
func (pt *ProposalTracker) UpdateVotingStatus(proposalID string) error
```

**Parameters:**
- `proposalID`: Proposal to update

**Returns:**
- `error`: Error if update fails

### CleanupOld
Removes old proposals based on age.

```go
func (pt *ProposalTracker) CleanupOld() error
```

**Returns:**
- `error`: Error if cleanup fails

### GetProposalStats
Returns proposal statistics.

```go
func (pt *ProposalTracker) GetProposalStats() *ProposalStats
```

**Returns:**
- `*ProposalStats`: Statistics summary

## Upgrade Scheduler API

### NewUpgradeScheduler
Creates a new upgrade scheduler.

```go
func NewUpgradeScheduler(cfg *config.Config, logger logger.Logger) *UpgradeScheduler
```

**Parameters:**
- `cfg`: Configuration
- `logger`: Logger instance

**Returns:**
- `*UpgradeScheduler`: Scheduler instance

### Start
Starts the upgrade scheduler.

```go
func (us *UpgradeScheduler) Start() error
```

**Returns:**
- `error`: Error if start fails

### Stop
Stops the upgrade scheduler.

```go
func (us *UpgradeScheduler) Stop() error
```

**Returns:**
- `error`: Error if stop fails

### ScheduleUpgrade
Schedules an upgrade from a proposal.

```go
func (us *UpgradeScheduler) ScheduleUpgrade(proposal *Proposal) error
```

**Parameters:**
- `proposal`: Upgrade proposal

**Returns:**
- `error`: Error if scheduling fails

### GetQueue
Returns the upgrade queue.

```go
func (us *UpgradeScheduler) GetQueue() ([]*UpgradeInfo, error)
```

**Returns:**
- `[]*UpgradeInfo`: Scheduled upgrades
- `error`: Error if retrieval fails

### GetNextUpgrade
Gets the next scheduled upgrade.

```go
func (us *UpgradeScheduler) GetNextUpgrade() (*UpgradeInfo, error)
```

**Returns:**
- `*UpgradeInfo`: Next upgrade
- `error`: Error if none scheduled

### GetCompletedUpgrades
Returns completed upgrades.

```go
func (us *UpgradeScheduler) GetCompletedUpgrades() ([]*UpgradeInfo, error)
```

**Returns:**
- `[]*UpgradeInfo`: Completed upgrades
- `error`: Error if retrieval fails

### IsUpgradeReady
Checks if upgrade is ready at height.

```go
func (us *UpgradeScheduler) IsUpgradeReady(currentHeight int64) (*UpgradeInfo, bool)
```

**Parameters:**
- `currentHeight`: Current block height

**Returns:**
- `*UpgradeInfo`: Ready upgrade
- `bool`: Ready flag

### UpdateStatus
Updates upgrade status.

```go
func (us *UpgradeScheduler) UpdateStatus(upgrade *UpgradeInfo) error
```

**Parameters:**
- `upgrade`: Upgrade to update

**Returns:**
- `error`: Error if update fails

### CancelUpgrade
Cancels a scheduled upgrade.

```go
func (us *UpgradeScheduler) CancelUpgrade(upgradeName string) error
```

**Parameters:**
- `upgradeName`: Upgrade to cancel

**Returns:**
- `error`: Error if cancellation fails

## Notifier API

### NewNotifier
Creates a new notifier instance.

```go
func NewNotifier(logger logger.Logger) *Notifier
```

**Parameters:**
- `logger`: Logger instance

**Returns:**
- `*Notifier`: Notifier instance

### Start
Starts the notification system.

```go
func (n *Notifier) Start() error
```

**Returns:**
- `error`: Error if start fails

### Stop
Stops the notification system.

```go
func (n *Notifier) Stop() error
```

**Returns:**
- `error`: Error if stop fails

### AddHandler
Adds a notification handler.

```go
func (n *Notifier) AddHandler(handler NotificationHandler)
```

**Parameters:**
- `handler`: Handler to add

### RemoveHandler
Removes a notification handler.

```go
func (n *Notifier) RemoveHandler(handlerType string) error
```

**Parameters:**
- `handlerType`: Handler type to remove

**Returns:**
- `error`: Error if not found

### NotifyNewProposal
Sends notification for new proposal.

```go
func (n *Notifier) NotifyNewProposal(proposal *Proposal)
```

**Parameters:**
- `proposal`: New proposal

### NotifyVotingStarted
Sends notification for voting start.

```go
func (n *Notifier) NotifyVotingStarted(proposal *Proposal)
```

**Parameters:**
- `proposal`: Proposal in voting

### NotifyVotingEnded
Sends notification for voting end.

```go
func (n *Notifier) NotifyVotingEnded(proposal *Proposal)
```

**Parameters:**
- `proposal`: Proposal after voting

### NotifyQuorumReached
Sends notification for quorum reached.

```go
func (n *Notifier) NotifyQuorumReached(proposal *Proposal)
```

**Parameters:**
- `proposal`: Proposal with quorum

### NotifyProposalPassed
Sends notification for passed proposal.

```go
func (n *Notifier) NotifyProposalPassed(proposal *Proposal)
```

**Parameters:**
- `proposal`: Passed proposal

### NotifyProposalRejected
Sends notification for rejected proposal.

```go
func (n *Notifier) NotifyProposalRejected(proposal *Proposal)
```

**Parameters:**
- `proposal`: Rejected proposal

### NotifyUpgradeScheduled
Sends notification for scheduled upgrade.

```go
func (n *Notifier) NotifyUpgradeScheduled(proposal *Proposal)
```

**Parameters:**
- `proposal`: Upgrade proposal

### NotifyUpgradeTriggered
Sends notification for triggered upgrade.

```go
func (n *Notifier) NotifyUpgradeTriggered(upgrade *UpgradeInfo)
```

**Parameters:**
- `upgrade`: Triggered upgrade

### NotifyUpgradeCompleted
Sends notification for completed upgrade.

```go
func (n *Notifier) NotifyUpgradeCompleted(upgrade *UpgradeInfo)
```

**Parameters:**
- `upgrade`: Completed upgrade

## Data Types

### Proposal
Represents a governance proposal.

```go
type Proposal struct {
    ID             string
    Title          string
    Description    string
    Type           ProposalType
    Status         ProposalStatus
    Proposer       string
    SubmitTime     time.Time
    VotingEndTime  time.Time
    UpgradeHeight  int64
    UpgradeInfo    *UpgradeInfo
    VotingStats    *VotingStats
}
```

### ProposalType
Proposal type enumeration.

```go
type ProposalType string

const (
    ProposalTypeUpgrade   ProposalType = "upgrade"
    ProposalTypeParameter ProposalType = "parameter"
    ProposalTypeText      ProposalType = "text"
    ProposalTypeCommunity ProposalType = "community"
)
```

### ProposalStatus
Proposal status enumeration.

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

### VotingStats
Voting statistics for a proposal.

```go
type VotingStats struct {
    YesVotes        int64
    NoVotes         int64
    AbstainVotes    int64
    NoWithVetoVotes int64
    QuorumReached   bool
    ThresholdMet    bool
    Turnout         float64
}
```

### UpgradeInfo
Upgrade information.

```go
type UpgradeInfo struct {
    Name        string
    Height      int64
    Info        string
    Status      UpgradeStatus
    Binaries    map[string]*BinaryInfo
    ProposalID  string
}
```

### UpgradeStatus
Upgrade status enumeration.

```go
type UpgradeStatus string

const (
    UpgradeStatusScheduled  UpgradeStatus = "scheduled"
    UpgradeStatusInProgress UpgradeStatus = "in_progress"
    UpgradeStatusCompleted  UpgradeStatus = "completed"
    UpgradeStatusFailed     UpgradeStatus = "failed"
    UpgradeStatusCancelled  UpgradeStatus = "cancelled"
)
```

### BinaryInfo
Binary download information.

```go
type BinaryInfo struct {
    URL      string
    Checksum string
    Platform string
    Arch     string
}
```

### ValidatorInfo
Validator information.

```go
type ValidatorInfo struct {
    OperatorAddress string
    ConsensusPubkey string
    Jailed          bool
    Status          string
    Tokens          string
    VotingPower     int64
    Commission      string
}
```

### GovernanceParams
Governance parameters.

```go
type GovernanceParams struct {
    VotingPeriod      time.Duration
    MinDeposit        string
    QuorumThreshold   string
    PassThreshold     string
    VetoThreshold     string
    MinUpgradeDelay   time.Duration
}
```

### BlockInfo
Block information.

```go
type BlockInfo struct {
    Height    int64
    Hash      string
    Time      time.Time
    Proposer  string
    TxCount   int
}
```

### Notification
Notification structure.

```go
type Notification struct {
    ID        string
    Event     NotificationEvent
    Title     string
    Message   string
    Data      interface{}
    Timestamp time.Time
    Read      bool
    Priority  NotificationPriority
}
```

### NotificationEvent
Notification event type.

```go
type NotificationEvent string

const (
    EventNewProposal      NotificationEvent = "new_proposal"
    EventVotingStarted    NotificationEvent = "voting_started"
    EventVotingEnded      NotificationEvent = "voting_ended"
    EventQuorumReached    NotificationEvent = "quorum_reached"
    EventProposalPassed   NotificationEvent = "proposal_passed"
    EventProposalRejected NotificationEvent = "proposal_rejected"
    EventUpgradeScheduled NotificationEvent = "upgrade_scheduled"
    EventUpgradeTriggered NotificationEvent = "upgrade_triggered"
    EventUpgradeCompleted NotificationEvent = "upgrade_completed"
)
```

### NotificationPriority
Notification priority level.

```go
type NotificationPriority int

const (
    PriorityLow    NotificationPriority = 0
    PriorityMedium NotificationPriority = 1
    PriorityHigh   NotificationPriority = 2
    PriorityCritical NotificationPriority = 3
)
```

### NotificationHandler
Handler interface for notifications.

```go
type NotificationHandler interface {
    Handle(notification *Notification) error
    IsEnabled() bool
    GetType() string
}
```

### ProposalStats
Statistics summary for proposals.

```go
type ProposalStats struct {
    TotalProposals    int
    ActiveProposals   int
    PassedProposals   int
    RejectedProposals int
    PendingUpgrades   int
    CompletedUpgrades int
}
```

## Error Types

### Common Errors

```go
var (
    ErrProposalNotFound  = errors.New("proposal not found")
    ErrUpgradeNotFound   = errors.New("upgrade not found")
    ErrInvalidStatus     = errors.New("invalid status")
    ErrInvalidProposal   = errors.New("invalid proposal")
    ErrAlreadyScheduled  = errors.New("upgrade already scheduled")
    ErrNotReady          = errors.New("upgrade not ready")
    ErrRPCConnection     = errors.New("RPC connection failed")
)
```