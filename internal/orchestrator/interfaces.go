package orchestrator

import (
	"github.com/wemix/wemixvisor/internal/config"
	"github.com/wemix/wemixvisor/internal/node"
	"github.com/wemix/wemixvisor/pkg/types"
)

// NodeManager defines the interface for managing node lifecycle.
// This abstraction follows the Dependency Inversion Principle (DIP),
// allowing UpgradeOrchestrator to work with any node management implementation.
//
// Implementations must be thread-safe as they may be called concurrently
// during upgrade operations.
type NodeManager interface {
	// Start starts the node with the given arguments.
	// Returns an error if the node cannot be started.
	//
	// Thread-safe: This method may be called concurrently.
	Start(args []string) error

	// Stop stops the running node.
	// Returns an error if the node cannot be stopped gracefully.
	//
	// Thread-safe: This method may be called concurrently.
	Stop() error

	// GetState returns the current state of the node.
	//
	// Thread-safe: This method may be called concurrently.
	GetState() node.NodeState

	// GetStatus returns the current status of the node.
	//
	// Thread-safe: This method may be called concurrently.
	GetStatus() *node.Status
}

// ConfigManager defines the interface for accessing configuration.
// This abstraction allows for different configuration sources and
// follows the Dependency Inversion Principle (DIP).
//
// Implementations must be thread-safe as configuration may be accessed
// concurrently during upgrade operations.
type ConfigManager interface {
	// GetConfig returns the current configuration.
	//
	// Thread-safe: This method may be called concurrently.
	GetConfig() *config.Config
}

// UpgradeWatcher defines the interface for monitoring upgrade plans.
// This abstraction allows for different upgrade detection mechanisms
// (file-based, governance-based, etc.) and follows the DIP.
//
// Implementations must be thread-safe as they may be called concurrently
// by the upgrade monitoring goroutine.
type UpgradeWatcher interface {
	// GetCurrentUpgrade returns the current pending upgrade info.
	// Returns nil if no upgrade is pending.
	//
	// Thread-safe: This method may be called concurrently.
	GetCurrentUpgrade() *types.UpgradeInfo

	// NeedsUpdate returns true if a new upgrade has been detected
	// and needs to be processed.
	//
	// Thread-safe: This method may be called concurrently.
	NeedsUpdate() bool

	// ClearUpdateFlag clears the update flag after an upgrade
	// has been processed.
	//
	// Thread-safe: This method may be called concurrently.
	ClearUpdateFlag()
}
