package governance

import (
	"encoding/json"
	"time"
)

// ProposalType represents the type of governance proposal
type ProposalType string

const (
	ProposalTypeUpgrade     ProposalType = "upgrade"
	ProposalTypeParameter   ProposalType = "parameter"
	ProposalTypeText        ProposalType = "text"
	ProposalTypeCommunity   ProposalType = "community"
	ProposalTypeEmergency   ProposalType = "emergency"
)

// ProposalStatus represents the current status of a proposal
type ProposalStatus string

const (
	ProposalStatusSubmitted ProposalStatus = "submitted"
	ProposalStatusVoting    ProposalStatus = "voting"
	ProposalStatusPassed    ProposalStatus = "passed"
	ProposalStatusRejected  ProposalStatus = "rejected"
	ProposalStatusFailed    ProposalStatus = "failed"
	ProposalStatusExpired   ProposalStatus = "expired"
)

// UpgradeStatus represents the status of an upgrade
type UpgradeStatus string

const (
	UpgradeStatusScheduled  UpgradeStatus = "scheduled"
	UpgradeStatusInProgress UpgradeStatus = "in_progress"
	UpgradeStatusCompleted  UpgradeStatus = "completed"
	UpgradeStatusFailed     UpgradeStatus = "failed"
	UpgradeStatusCancelled  UpgradeStatus = "cancelled"
)

// Proposal represents a governance proposal
type Proposal struct {
	ID             string                 `json:"id"`
	Title          string                 `json:"title"`
	Description    string                 `json:"description"`
	Type           ProposalType           `json:"type"`
	Status         ProposalStatus         `json:"status"`
	Proposer       string                 `json:"proposer"`
	SubmitTime     time.Time              `json:"submit_time"`
	VotingEndTime  time.Time              `json:"voting_end_time"`
	UpgradeHeight  int64                  `json:"upgrade_height,omitempty"`
	UpgradeInfo    *UpgradeInfo           `json:"upgrade_info,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`

	// Voting statistics
	VotingStats *VotingStats `json:"voting_stats,omitempty"`
}

// VotingStats contains voting statistics for a proposal
type VotingStats struct {
	YesVotes        int64   `json:"yes_votes"`
	NoVotes         int64   `json:"no_votes"`
	AbstainVotes    int64   `json:"abstain_votes"`
	NoWithVetoVotes int64   `json:"no_with_veto_votes"`
	TotalVotes      int64   `json:"total_votes"`
	Turnout         float64 `json:"turnout"`
	QuorumReached   bool    `json:"quorum_reached"`
	ThresholdMet    bool    `json:"threshold_met"`
}

// UpgradeInfo contains information about a planned upgrade
type UpgradeInfo struct {
	Name        string                 `json:"name"`
	Height      int64                  `json:"height"`
	Info        string                 `json:"info"`
	Binaries    map[string]*BinaryInfo `json:"binaries,omitempty"`
	UpgradeURL  string                 `json:"upgrade_url,omitempty"`
	ChecksumURL string                 `json:"checksum_url,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Status      UpgradeStatus          `json:"status"`

	// Timing information
	ScheduledTime time.Time `json:"scheduled_time"`
	StartedTime   *time.Time `json:"started_time,omitempty"`
	CompletedTime *time.Time `json:"completed_time,omitempty"`
}

// BinaryInfo contains information about a binary for a specific platform
type BinaryInfo struct {
	URL      string `json:"url"`
	Checksum string `json:"checksum"`
	Platform string `json:"platform"`
	Arch     string `json:"arch"`
}

// Vote represents a single vote on a proposal
type Vote struct {
	ProposalID string    `json:"proposal_id"`
	Voter      string    `json:"voter"`
	Option     VoteOption `json:"option"`
	Weight     string    `json:"weight"`
	Timestamp  time.Time `json:"timestamp"`
}

// VoteOption represents voting options
type VoteOption string

const (
	VoteOptionYes        VoteOption = "yes"
	VoteOptionNo         VoteOption = "no"
	VoteOptionAbstain    VoteOption = "abstain"
	VoteOptionNoWithVeto VoteOption = "no_with_veto"
)

// GovernanceParams contains governance parameters
type GovernanceParams struct {
	VotingPeriod         time.Duration `json:"voting_period"`
	MinDeposit           string        `json:"min_deposit"`
	QuorumThreshold      string        `json:"quorum_threshold"`
	PassThreshold        string        `json:"pass_threshold"`
	VetoThreshold        string        `json:"veto_threshold"`
	MaxUpgradeHeight     int64         `json:"max_upgrade_height"`
	MinUpgradeDelay      time.Duration `json:"min_upgrade_delay"`
	EmergencyVotePeriod  time.Duration `json:"emergency_vote_period"`
}

// ValidatorInfo contains information about a validator
type ValidatorInfo struct {
	OperatorAddress string `json:"operator_address"`
	ConsensusPubkey string `json:"consensus_pubkey"`
	Jailed          bool   `json:"jailed"`
	Status          string `json:"status"`
	Tokens          string `json:"tokens"`
	VotingPower     int64  `json:"voting_power"`
	Commission      string `json:"commission"`
	Moniker         string `json:"moniker"`
}

// BlockInfo contains information about a block
type BlockInfo struct {
	Height    int64     `json:"height"`
	Hash      string    `json:"hash"`
	Time      time.Time `json:"time"`
	Proposer  string    `json:"proposer"`
	TxCount   int       `json:"tx_count"`
	Validator string    `json:"validator"`
}

// NotificationEvent represents different types of events that can trigger notifications
type NotificationEvent string

const (
	EventNewProposal       NotificationEvent = "new_proposal"
	EventProposalPassed    NotificationEvent = "proposal_passed"
	EventProposalRejected  NotificationEvent = "proposal_rejected"
	EventUpgradeScheduled  NotificationEvent = "upgrade_scheduled"
	EventUpgradeTriggered  NotificationEvent = "upgrade_triggered"
	EventUpgradeCompleted  NotificationEvent = "upgrade_completed"
	EventUpgradeFailed     NotificationEvent = "upgrade_failed"
	EventVotingStarted     NotificationEvent = "voting_started"
	EventVotingEnded       NotificationEvent = "voting_ended"
	EventQuorumReached     NotificationEvent = "quorum_reached"
	EventEmergencyProposal NotificationEvent = "emergency_proposal"
)

// Notification represents a notification about a governance event
type Notification struct {
	ID        string            `json:"id"`
	Event     NotificationEvent `json:"event"`
	Title     string            `json:"title"`
	Message   string            `json:"message"`
	Data      interface{}       `json:"data"`
	Timestamp time.Time         `json:"timestamp"`
	Read      bool              `json:"read"`
	Priority  NotificationPriority `json:"priority"`
}

// NotificationPriority represents the priority level of a notification
type NotificationPriority string

const (
	PriorityLow      NotificationPriority = "low"
	PriorityMedium   NotificationPriority = "medium"
	PriorityHigh     NotificationPriority = "high"
	PriorityCritical NotificationPriority = "critical"
)

// SyncStatus represents the synchronization status with the blockchain
type SyncStatus struct {
	LastSyncHeight    int64     `json:"last_sync_height"`
	LastSyncTime      time.Time `json:"last_sync_time"`
	CurrentHeight     int64     `json:"current_height"`
	IsSyncing         bool      `json:"is_syncing"`
	SyncProgress      float64   `json:"sync_progress"`
	ActiveProposals   int       `json:"active_proposals"`
	PendingUpgrades   int       `json:"pending_upgrades"`
	LastErrorMessage  string    `json:"last_error_message,omitempty"`
	LastErrorTime     *time.Time `json:"last_error_time,omitempty"`
}

// writeUpgradeInfo writes upgrade information to a JSON file
func writeUpgradeInfo(path string, data map[string]interface{}) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return writeFile(path, jsonData)
}

// writeFile is a placeholder for file writing functionality
// This should be implemented based on the project's file handling patterns
func writeFile(path string, data []byte) error {
	// This will be implemented when we add file operations
	// For now, return nil to satisfy the interface
	return nil
}