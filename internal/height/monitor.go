package height

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/wemix/wemixvisor/pkg/logger"
)

const (
	// DefaultPollInterval is the default interval for polling blockchain height
	DefaultPollInterval = 5 * time.Second

	// DefaultSubscriberBufferSize is the buffer size for subscriber channels
	DefaultSubscriberBufferSize = 10
)

// HeightMonitor continuously monitors blockchain height and notifies subscribers
// when the height changes.
//
// The monitor follows these design principles:
// - Single Responsibility: Only monitors blockchain height
// - Open/Closed: Extensible via HeightProvider interface
// - Dependency Inversion: Depends on HeightProvider abstraction
//
// Thread-safety: All public methods are thread-safe and can be called concurrently.
type HeightMonitor struct {
	// Core dependencies (injected)
	provider HeightProvider
	logger   *logger.Logger

	// State (protected by mu)
	currentHeight int64
	started       bool
	mu            sync.RWMutex

	// Configuration
	pollInterval time.Duration

	// Subscriber management (protected by subMu)
	subscribers []chan<- int64
	subMu       sync.RWMutex

	// Lifecycle management
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewHeightMonitor creates a new HeightMonitor instance.
//
// Parameters:
//   - provider: The HeightProvider implementation for querying blockchain height
//   - interval: The polling interval (use 0 for default)
//   - logger: Logger instance for structured logging
//
// Returns a configured HeightMonitor ready to be started.
//
// The monitor is created in a stopped state. Call Start() to begin monitoring.
func NewHeightMonitor(provider HeightProvider, interval time.Duration, logger *logger.Logger) *HeightMonitor {
	ctx, cancel := context.WithCancel(context.Background())

	// Use default interval if not specified
	if interval <= 0 {
		interval = DefaultPollInterval
	}

	return &HeightMonitor{
		provider:     provider,
		logger:       logger,
		pollInterval: interval,
		subscribers:  make([]chan<- int64, 0),
		ctx:          ctx,
		cancel:       cancel,
	}
}

// Start begins monitoring blockchain height.
//
// The monitor will poll the HeightProvider at the configured interval
// and notify subscribers when the height changes.
//
// Returns an error if the monitor is already started.
//
// Thread-safe: Can be called concurrently, but starting an already-started
// monitor will return an error.
func (hm *HeightMonitor) Start() error {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	// Check if already started
	if hm.started {
		return fmt.Errorf("monitor already started")
	}

	// Mark as started
	hm.started = true

	// Start the monitoring goroutine
	hm.wg.Add(1)
	go hm.monitorLoop()

	return nil
}

// Stop stops the height monitor and waits for cleanup.
//
// This method is idempotent - calling it multiple times is safe.
// It will block until the monitoring goroutine has fully stopped.
//
// Thread-safe: Can be called concurrently.
func (hm *HeightMonitor) Stop() {
	// Cancel the context to signal goroutine to stop
	hm.cancel()

	// Wait for monitoring goroutine to finish
	hm.wg.Wait()
}

// Subscribe returns a channel that receives height updates.
//
// The channel has a buffer of DefaultSubscriberBufferSize to prevent
// blocking the monitor if a subscriber is slow.
//
// If the channel buffer fills up, the oldest update is dropped and
// a warning is logged.
//
// The returned channel is read-only. Subscribers should read from it
// in a loop until the monitor is stopped.
//
// Thread-safe: Can be called concurrently.
func (hm *HeightMonitor) Subscribe() <-chan int64 {
	hm.subMu.Lock()
	defer hm.subMu.Unlock()

	// Create a buffered channel for this subscriber
	ch := make(chan int64, DefaultSubscriberBufferSize)

	// Add to subscribers list
	hm.subscribers = append(hm.subscribers, ch)

	return ch
}

// GetCurrentHeight returns the last observed blockchain height.
//
// Returns 0 if the monitor has not yet successfully queried the height.
//
// Thread-safe: Can be called concurrently.
func (hm *HeightMonitor) GetCurrentHeight() int64 {
	hm.mu.RLock()
	defer hm.mu.RUnlock()
	return hm.currentHeight
}

// monitorLoop is the main monitoring goroutine.
//
// It continuously polls the HeightProvider and notifies subscribers
// when the height changes.
//
// The loop exits when the context is cancelled (via Stop).
func (hm *HeightMonitor) monitorLoop() {
	defer hm.wg.Done()

	ticker := time.NewTicker(hm.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-hm.ctx.Done():
			// Context cancelled, exit monitoring loop
			return

		case <-ticker.C:
			// Poll for current height
			height, err := hm.provider.GetCurrentHeight()
			if err != nil {
				hm.logger.Error("failed to get current height", "error", err)
				continue
			}

			// Check if height has changed
			hm.mu.Lock()
			if height != hm.currentHeight {
				oldHeight := hm.currentHeight
				hm.currentHeight = height
				hm.mu.Unlock()

				// Log height change
				hm.logger.Info("blockchain height updated",
					"old_height", oldHeight,
					"new_height", height)

				// Notify all subscribers
				hm.notifySubscribers(height)
			} else {
				hm.mu.Unlock()
			}
		}
	}
}

// notifySubscribers sends a height update to all subscribers.
//
// Uses a non-blocking send to prevent slow subscribers from blocking
// the monitor. If a subscriber's channel is full, the update is dropped
// and a warning is logged.
//
// Thread-safe: Protected by subMu lock.
func (hm *HeightMonitor) notifySubscribers(height int64) {
	hm.subMu.RLock()
	defer hm.subMu.RUnlock()

	for i, ch := range hm.subscribers {
		select {
		case ch <- height:
			// Successfully sent
		default:
			// Channel is full, drop this update and log a warning
			hm.logger.Warn("subscriber channel full, dropping update",
				"subscriber_index", i,
				"height", height)
		}
	}
}
