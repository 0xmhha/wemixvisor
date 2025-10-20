package height

// HeightProvider defines the interface for querying blockchain height.
// This abstraction allows for different implementations (RPC client, mock, etc.)
// and follows the Dependency Inversion Principle (DIP).
//
// Implementations must be thread-safe as they may be called concurrently
// by the height monitoring goroutine.
type HeightProvider interface {
	// GetCurrentHeight returns the current blockchain height.
	// Returns an error if the height cannot be retrieved (e.g., RPC timeout, network error).
	//
	// Thread-safe: This method may be called concurrently.
	GetCurrentHeight() (int64, error)
}
