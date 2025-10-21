package height

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wemix/wemixvisor/pkg/logger"
)

// MockHeightProvider is a mock implementation of HeightProvider for testing.
// It tracks the number of calls and allows controlling the returned height and error.
type MockHeightProvider struct {
	mu     sync.Mutex
	height int64
	err    error
	calls  int
	// callDelay simulates network latency
	callDelay time.Duration
}

// NewMockHeightProvider creates a new MockHeightProvider with the given initial height.
func NewMockHeightProvider(height int64) *MockHeightProvider {
	return &MockHeightProvider{
		height: height,
	}
}

// GetCurrentHeight implements HeightProvider interface.
func (m *MockHeightProvider) GetCurrentHeight() (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.calls++

	// Simulate network delay if configured
	if m.callDelay > 0 {
		time.Sleep(m.callDelay)
	}

	return m.height, m.err
}

// SetHeight updates the mock height to return.
func (m *MockHeightProvider) SetHeight(height int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.height = height
}

// SetError sets the error to return from GetCurrentHeight.
func (m *MockHeightProvider) SetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.err = err
}

// SetCallDelay sets a delay to simulate network latency.
func (m *MockHeightProvider) SetCallDelay(delay time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callDelay = delay
}

// GetCalls returns the number of times GetCurrentHeight was called.
func (m *MockHeightProvider) GetCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls
}

// ResetCalls resets the call counter to zero.
func (m *MockHeightProvider) ResetCalls() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = 0
}

// Test helper function to create a test logger
func newTestLogger() *logger.Logger {
	log, err := logger.New(false, false, "")
	if err != nil {
		panic(err)
	}
	return log
}

// =============================================================================
// Task 8.1.2.2: Test NewHeightMonitor creation
// =============================================================================

func TestNewHeightMonitor_Success(t *testing.T) {
	// Arrange
	provider := NewMockHeightProvider(1000)
	interval := 1 * time.Second
	log := newTestLogger()

	// Act
	monitor := NewHeightMonitor(provider, interval, log)

	// Assert
	require.NotNil(t, monitor, "NewHeightMonitor should return non-nil monitor")
	assert.NotNil(t, monitor.provider, "provider should be set")
	assert.NotNil(t, monitor.logger, "logger should be set")
	assert.Equal(t, interval, monitor.pollInterval, "pollInterval should match provided value")
	assert.NotNil(t, monitor.ctx, "context should be initialized")
	assert.NotNil(t, monitor.cancel, "cancel function should be initialized")
	assert.NotNil(t, monitor.subscribers, "subscribers slice should be initialized")
	assert.Equal(t, int64(0), monitor.currentHeight, "initial height should be 0")
}

func TestNewHeightMonitor_DefaultInterval(t *testing.T) {
	// Arrange
	provider := NewMockHeightProvider(1000)
	log := newTestLogger()

	// Act - pass 0 or negative interval
	monitor := NewHeightMonitor(provider, 0, log)

	// Assert
	require.NotNil(t, monitor, "NewHeightMonitor should return non-nil monitor")
	assert.Equal(t, DefaultPollInterval, monitor.pollInterval,
		"should use DefaultPollInterval when 0 is provided")

	// Act - negative interval
	monitor2 := NewHeightMonitor(provider, -1*time.Second, log)

	// Assert
	assert.Equal(t, DefaultPollInterval, monitor2.pollInterval,
		"should use DefaultPollInterval when negative is provided")
}

func TestNewHeightMonitor_NilProvider(t *testing.T) {
	// Arrange
	log := newTestLogger()

	// Act
	monitor := NewHeightMonitor(nil, DefaultPollInterval, log)

	// Assert
	require.NotNil(t, monitor, "NewHeightMonitor should return non-nil monitor even with nil provider")
	// Note: The actual Start() call should handle nil provider gracefully
}

func TestNewHeightMonitor_MultipleInstances(t *testing.T) {
	// Arrange
	provider := NewMockHeightProvider(1000)
	log := newTestLogger()

	// Act - create multiple monitors
	monitor1 := NewHeightMonitor(provider, 1*time.Second, log)
	monitor2 := NewHeightMonitor(provider, 2*time.Second, log)

	// Assert - each should have independent state
	require.NotNil(t, monitor1, "first monitor should be created")
	require.NotNil(t, monitor2, "second monitor should be created")
	assert.NotEqual(t, monitor1.pollInterval, monitor2.pollInterval, "monitors should have independent intervals")

	// Start both and verify they work independently
	err1 := monitor1.Start()
	err2 := monitor2.Start()
	assert.NoError(t, err1, "first monitor should start")
	assert.NoError(t, err2, "second monitor should start")

	// Cleanup
	monitor1.Stop()
	monitor2.Stop()
}

// =============================================================================
// Task 8.1.2.3: Test Start and Stop lifecycle
// =============================================================================

func TestHeightMonitor_Start_Success(t *testing.T) {
	// Arrange
	provider := NewMockHeightProvider(1000)
	log := newTestLogger()
	monitor := NewHeightMonitor(provider, 100*time.Millisecond, log)

	// Act
	err := monitor.Start()

	// Assert
	assert.NoError(t, err, "Start should succeed on first call")

	// Cleanup
	monitor.Stop()
}

func TestHeightMonitor_Start_AlreadyStarted(t *testing.T) {
	// Arrange
	provider := NewMockHeightProvider(1000)
	log := newTestLogger()
	monitor := NewHeightMonitor(provider, 100*time.Millisecond, log)

	// Act
	err1 := monitor.Start()
	err2 := monitor.Start() // Second call should fail

	// Assert
	assert.NoError(t, err1, "first Start should succeed")
	assert.Error(t, err2, "second Start should return error")
	assert.Contains(t, err2.Error(), "already started", "error should indicate monitor is already started")

	// Cleanup
	monitor.Stop()
}

func TestHeightMonitor_Stop_Idempotent(t *testing.T) {
	// Arrange
	provider := NewMockHeightProvider(1000)
	log := newTestLogger()
	monitor := NewHeightMonitor(provider, 100*time.Millisecond, log)

	// Act
	err := monitor.Start()
	require.NoError(t, err, "Start should succeed")

	// Stop multiple times
	monitor.Stop()
	monitor.Stop()
	monitor.Stop()

	// Assert - should not panic or cause issues
	// If we get here without panic, test passes
}

func TestHeightMonitor_Stop_WithoutStart(t *testing.T) {
	// Arrange
	provider := NewMockHeightProvider(1000)
	log := newTestLogger()
	monitor := NewHeightMonitor(provider, 100*time.Millisecond, log)

	// Act & Assert - calling Stop without Start should not panic
	monitor.Stop()
}

func TestHeightMonitor_Lifecycle_StartStopStart(t *testing.T) {
	// Arrange
	provider := NewMockHeightProvider(1000)
	log := newTestLogger()
	monitor := NewHeightMonitor(provider, 100*time.Millisecond, log)

	// Act - Start, Stop, then Start again
	err1 := monitor.Start()
	require.NoError(t, err1, "first Start should succeed")

	monitor.Stop()
	time.Sleep(150 * time.Millisecond) // Wait for goroutine to fully stop

	err2 := monitor.Start()

	// Assert
	assert.Error(t, err2, "Starting again after Stop should fail (context is cancelled)")
	// Note: Current design does not support restart. This is intentional.
	// If restart is needed, a new HeightMonitor instance should be created.

	// Cleanup
	monitor.Stop()
}

// =============================================================================
// Task 8.1.2.4: Test height monitoring
// =============================================================================

func TestHeightMonitor_GetCurrentHeight_Initial(t *testing.T) {
	// Arrange
	provider := NewMockHeightProvider(1000)
	log := newTestLogger()
	monitor := NewHeightMonitor(provider, 100*time.Millisecond, log)

	// Act
	height := monitor.GetCurrentHeight()

	// Assert
	assert.Equal(t, int64(0), height, "initial height should be 0 before monitoring starts")
}

func TestHeightMonitor_MonitorsHeight_Success(t *testing.T) {
	// Arrange
	provider := NewMockHeightProvider(1000)
	log := newTestLogger()
	monitor := NewHeightMonitor(provider, 50*time.Millisecond, log)

	// Act
	err := monitor.Start()
	require.NoError(t, err, "Start should succeed")
	defer monitor.Stop()

	// Wait for at least one poll cycle
	time.Sleep(150 * time.Millisecond)

	// Assert
	height := monitor.GetCurrentHeight()
	assert.Equal(t, int64(1000), height, "height should be updated from provider")
	assert.GreaterOrEqual(t, provider.GetCalls(), 1, "provider should be called at least once")
}

func TestHeightMonitor_MonitorsHeight_ProviderError(t *testing.T) {
	// Arrange
	provider := NewMockHeightProvider(1000)
	provider.SetError(assert.AnError)
	log := newTestLogger()
	monitor := NewHeightMonitor(provider, 50*time.Millisecond, log)

	// Act
	err := monitor.Start()
	require.NoError(t, err, "Start should succeed")
	defer monitor.Stop()

	// Wait for multiple poll cycles
	time.Sleep(150 * time.Millisecond)

	// Assert
	height := monitor.GetCurrentHeight()
	assert.Equal(t, int64(0), height, "height should remain 0 when provider returns errors")
	assert.GreaterOrEqual(t, provider.GetCalls(), 1, "provider should still be called despite errors")
	// Monitor should continue running despite errors (resilience)
}

func TestHeightMonitor_MonitorsHeight_HeightChanges(t *testing.T) {
	// Arrange
	provider := NewMockHeightProvider(1000)
	log := newTestLogger()
	monitor := NewHeightMonitor(provider, 50*time.Millisecond, log)

	// Act
	err := monitor.Start()
	require.NoError(t, err, "Start should succeed")
	defer monitor.Stop()

	// Wait for first poll
	time.Sleep(100 * time.Millisecond)
	height1 := monitor.GetCurrentHeight()

	// Change provider height
	provider.SetHeight(2000)

	// Wait for next poll
	time.Sleep(100 * time.Millisecond)
	height2 := monitor.GetCurrentHeight()

	// Assert
	assert.Equal(t, int64(1000), height1, "first poll should get initial height")
	assert.Equal(t, int64(2000), height2, "second poll should get updated height")
}

// =============================================================================
// Task 8.1.2.5: Test subscriber pattern
// =============================================================================

func TestHeightMonitor_Subscribe_ReceivesUpdates(t *testing.T) {
	// Arrange
	provider := NewMockHeightProvider(1000)
	log := newTestLogger()
	monitor := NewHeightMonitor(provider, 50*time.Millisecond, log)

	// Subscribe before starting
	sub := monitor.Subscribe()
	require.NotNil(t, sub, "Subscribe should return non-nil channel")

	// Act
	err := monitor.Start()
	require.NoError(t, err, "Start should succeed")
	defer monitor.Stop()

	// Wait for first update
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	select {
	case height := <-sub:
		// Assert
		assert.Equal(t, int64(1000), height, "subscriber should receive initial height")
	case <-ctx.Done():
		t.Fatal("timeout waiting for height update")
	}
}

func TestHeightMonitor_Subscribe_MultipleSubscribers(t *testing.T) {
	// Arrange
	provider := NewMockHeightProvider(1000)
	log := newTestLogger()
	monitor := NewHeightMonitor(provider, 50*time.Millisecond, log)

	// Create multiple subscribers
	sub1 := monitor.Subscribe()
	sub2 := monitor.Subscribe()
	sub3 := monitor.Subscribe()

	// Act
	err := monitor.Start()
	require.NoError(t, err, "Start should succeed")
	defer monitor.Stop()

	// Collect updates from all subscribers
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	heights := make([]int64, 3)
	var wg sync.WaitGroup
	wg.Add(3)

	// Subscriber 1
	go func() {
		defer wg.Done()
		select {
		case h := <-sub1:
			heights[0] = h
		case <-ctx.Done():
		}
	}()

	// Subscriber 2
	go func() {
		defer wg.Done()
		select {
		case h := <-sub2:
			heights[1] = h
		case <-ctx.Done():
		}
	}()

	// Subscriber 3
	go func() {
		defer wg.Done()
		select {
		case h := <-sub3:
			heights[2] = h
		case <-ctx.Done():
		}
	}()

	wg.Wait()

	// Assert - all subscribers should receive the update
	assert.Equal(t, int64(1000), heights[0], "subscriber 1 should receive height")
	assert.Equal(t, int64(1000), heights[1], "subscriber 2 should receive height")
	assert.Equal(t, int64(1000), heights[2], "subscriber 3 should receive height")
}

func TestHeightMonitor_Subscribe_OnlyNotifiesOnChange(t *testing.T) {
	// Arrange
	provider := NewMockHeightProvider(1000)
	log := newTestLogger()
	monitor := NewHeightMonitor(provider, 30*time.Millisecond, log)

	sub := monitor.Subscribe()

	// Act
	err := monitor.Start()
	require.NoError(t, err, "Start should succeed")
	defer monitor.Stop()

	// Receive first update
	ctx1, cancel1 := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel1()

	var firstHeight int64
	select {
	case firstHeight = <-sub:
		// Got first update
	case <-ctx1.Done():
		t.Fatal("timeout waiting for first height update")
	}

	assert.Equal(t, int64(1000), firstHeight, "should receive initial height")

	// Wait for multiple poll cycles without height change
	time.Sleep(150 * time.Millisecond)

	// Try to receive another update (should not get one since height hasn't changed)
	ctx2, cancel2 := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel2()

	select {
	case h := <-sub:
		t.Fatalf("should not receive update when height hasn't changed, got: %d", h)
	case <-ctx2.Done():
		// Expected - no update received
	}

	// Now change the height
	provider.SetHeight(2000)

	// Should receive new update
	ctx3, cancel3 := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel3()

	select {
	case h := <-sub:
		assert.Equal(t, int64(2000), h, "should receive updated height")
	case <-ctx3.Done():
		t.Fatal("timeout waiting for height change update")
	}
}

func TestHeightMonitor_Subscribe_BufferOverflow(t *testing.T) {
	// Arrange
	provider := NewMockHeightProvider(1000)
	log := newTestLogger()
	monitor := NewHeightMonitor(provider, 20*time.Millisecond, log)

	sub := monitor.Subscribe()

	// Act
	err := monitor.Start()
	require.NoError(t, err, "Start should succeed")
	defer monitor.Stop()

	// Don't read from subscriber channel to fill the buffer
	// Change height multiple times to trigger multiple notifications
	for i := int64(1000); i <= 1000+DefaultSubscriberBufferSize+5; i++ {
		provider.SetHeight(i)
		time.Sleep(25 * time.Millisecond) // Wait for poll cycle
	}

	// Assert - monitor should still be running (not blocked)
	// Try to get current height (if monitor is blocked, this would timeout)
	height := monitor.GetCurrentHeight()
	assert.Greater(t, height, int64(1000), "monitor should continue running despite slow subscriber")

	// Drain the channel to verify it has updates (but may have dropped some)
	updateCount := 0
	draining := true
	for draining {
		select {
		case <-sub:
			updateCount++
		case <-time.After(10 * time.Millisecond):
			draining = false
		}
	}

	assert.LessOrEqual(t, updateCount, DefaultSubscriberBufferSize+1,
		"channel should not have more than buffer size + 1 updates")
}

// =============================================================================
// Task 8.1.2.6: Test concurrent access safety
// =============================================================================

func TestHeightMonitor_ConcurrentGetCurrentHeight(t *testing.T) {
	// Arrange
	provider := NewMockHeightProvider(1000)
	log := newTestLogger()
	monitor := NewHeightMonitor(provider, 50*time.Millisecond, log)

	err := monitor.Start()
	require.NoError(t, err, "Start should succeed")
	defer monitor.Stop()

	// Act - multiple goroutines reading height concurrently
	var wg sync.WaitGroup
	const numReaders = 50

	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				_ = monitor.GetCurrentHeight()
				time.Sleep(1 * time.Millisecond)
			}
		}()
	}

	// Concurrently change height
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := int64(1000); i < 1100; i++ {
			provider.SetHeight(i)
			time.Sleep(5 * time.Millisecond)
		}
	}()

	// Assert - should complete without deadlock or panic
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success - all goroutines completed
	case <-time.After(5 * time.Second):
		t.Fatal("test timed out - possible deadlock")
	}
}

func TestHeightMonitor_ConcurrentSubscribe(t *testing.T) {
	// Arrange
	provider := NewMockHeightProvider(1000)
	log := newTestLogger()
	monitor := NewHeightMonitor(provider, 50*time.Millisecond, log)

	err := monitor.Start()
	require.NoError(t, err, "Start should succeed")
	defer monitor.Stop()

	// Act - multiple goroutines subscribing concurrently
	var wg sync.WaitGroup
	const numSubscribers = 20

	for i := 0; i < numSubscribers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ch := monitor.Subscribe()
			require.NotNil(t, ch, "Subscribe should return non-nil channel")

			// Try to receive one update
			ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
			defer cancel()

			select {
			case <-ch:
				// Received update
			case <-ctx.Done():
				// Timeout is acceptable in this test
			}
		}()
	}

	// Assert - should complete without deadlock or panic
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success - all goroutines completed
	case <-time.After(2 * time.Second):
		t.Fatal("test timed out - possible deadlock in Subscribe")
	}
}

func TestHeightMonitor_ConcurrentStartStop(t *testing.T) {
	// Arrange
	provider := NewMockHeightProvider(1000)
	log := newTestLogger()

	// Act - attempt concurrent Start/Stop operations
	// Note: This is testing that concurrent calls don't cause panics or deadlocks
	// Actual behavior is that only one Start should succeed
	var wg sync.WaitGroup
	const numAttempts = 10

	for i := 0; i < numAttempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			monitor := NewHeightMonitor(provider, 50*time.Millisecond, log)

			// Rapid Start/Stop cycles
			_ = monitor.Start()
			time.Sleep(10 * time.Millisecond)
			monitor.Stop()
		}()
	}

	// Assert - should complete without deadlock or panic
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success - all goroutines completed
	case <-time.After(3 * time.Second):
		t.Fatal("test timed out - possible deadlock in Start/Stop")
	}
}
