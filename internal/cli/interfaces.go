package cli

import (
	"github.com/wemix/wemixvisor/internal/node"
)

// NodeManager defines the interface for node management
type NodeManager interface {
	Start(args []string) error
	Stop() error
	Restart() error
	GetState() node.NodeState
	GetStatus() *node.Status
	GetVersion() string
	GetPID() int
	SetNodeArgs(args []string)
	Wait() <-chan struct{}
	IsHealthy() bool
	Close() error
}