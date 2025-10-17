package performance

import (
	"context"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wemix/wemixvisor/pkg/logger"
)

// Mock connection for testing
type mockConn struct {
	closed bool
	mu     sync.Mutex
}

func (m *mockConn) Read(b []byte) (n int, err error)   { return 0, nil }
func (m *mockConn) Write(b []byte) (n int, err error)  { return len(b), nil }
func (m *mockConn) Close() error                       { m.mu.Lock(); m.closed = true; m.mu.Unlock(); return nil }
func (m *mockConn) LocalAddr() net.Addr                { return nil }
func (m *mockConn) RemoteAddr() net.Addr               { return nil }
func (m *mockConn) SetDeadline(t time.Time) error      { return nil }
func (m *mockConn) SetReadDeadline(t time.Time) error  { return nil }
func (m *mockConn) SetWriteDeadline(t time.Time) error { return nil }

func TestCache(t *testing.T) {
	log, err := logger.New(false, false, "iso8601")
	require.NoError(t, err)

	cache := NewCache(100, 1*time.Hour, log)
	err = cache.Start()
	require.NoError(t, err)
	defer cache.Stop()

	// Test Set and Get
	cache.Set("key1", "value1", 1)
	val, found := cache.Get("key1")
	assert.True(t, found)
	assert.Equal(t, "value1", val)

	// Test cache miss
	_, found = cache.Get("nonexistent")
	assert.False(t, found)

	// Test Delete
	cache.Delete("key1")
	_, found = cache.Get("key1")
	assert.False(t, found)

	// Test Clear
	cache.Set("key2", "value2", 1)
	cache.Set("key3", "value3", 1)
	cache.Clear()
	_, found = cache.Get("key2")
	assert.False(t, found)

	// Test stats
	stats := cache.GetStats()
	assert.NotNil(t, stats)
	assert.GreaterOrEqual(t, stats.Misses, int64(1))
}

func TestCacheLRU(t *testing.T) {
	log, err := logger.New(false, false, "iso8601")
	require.NoError(t, err)

	cache := NewCache(3, 1*time.Hour, log)
	err = cache.Start()
	require.NoError(t, err)
	defer cache.Stop()

	// Fill cache to max size
	cache.Set("key1", "value1", 1)
	cache.Set("key2", "value2", 1)
	cache.Set("key3", "value3", 1)

	// Access key1 to make it most recently used
	cache.Get("key1")

	// Add new item, should evict key2 (least recently used)
	cache.Set("key4", "value4", 1)

	// Check key2 was evicted
	_, found := cache.Get("key2")
	assert.False(t, found)

	// Check key1 still exists
	val, found := cache.Get("key1")
	assert.True(t, found)
	assert.Equal(t, "value1", val)
}

func TestCacheTTL(t *testing.T) {
	log, err := logger.New(false, false, "iso8601")
	require.NoError(t, err)

	cache := NewCache(100, 100*time.Millisecond, log)
	err = cache.Start()
	require.NoError(t, err)
	defer cache.Stop()

	cache.Set("key1", "value1", 1)

	// Item should exist initially
	val, found := cache.Get("key1")
	assert.True(t, found)
	assert.Equal(t, "value1", val)

	// Wait for TTL to expire
	time.Sleep(150 * time.Millisecond)

	// Item should be expired
	_, found = cache.Get("key1")
	assert.False(t, found)
}

func TestConnectionPool(t *testing.T) {
	log, err := logger.New(false, false, "iso8601")
	require.NoError(t, err)

	pool := NewConnectionPool(5, log)
	pool.SetFactory(func() (net.Conn, error) {
		return &mockConn{}, nil
	})

	err = pool.Start()
	require.NoError(t, err)
	defer pool.Stop()

	// Get connection
	ctx := context.Background()
	conn, err := pool.Get(ctx)
	require.NoError(t, err)
	assert.NotNil(t, conn)

	// Return connection
	pool.Put(conn)

	// Get stats
	stats := pool.GetStats()
	assert.NotNil(t, stats)
	assert.GreaterOrEqual(t, stats.Created, int64(1))
}

func TestConnectionPoolMaxSize(t *testing.T) {
	log, err := logger.New(false, false, "iso8601")
	require.NoError(t, err)

	pool := NewConnectionPool(2, log)
	pool.SetFactory(func() (net.Conn, error) {
		return &mockConn{}, nil
	})

	err = pool.Start()
	require.NoError(t, err)
	defer pool.Stop()

	ctx := context.Background()

	// Get max connections
	conn1, err := pool.Get(ctx)
	require.NoError(t, err)
	conn2, err := pool.Get(ctx)
	require.NoError(t, err)

	// Try to get one more with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err = pool.Get(ctx)
	assert.Error(t, err) // Should timeout

	// Return one connection
	pool.Put(conn1)

	// Now should be able to get another
	conn3, err := pool.Get(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, conn3)

	pool.Put(conn2)
	pool.Put(conn3)
}

func TestWorkerPool(t *testing.T) {
	log, err := logger.New(false, false, "iso8601")
	require.NoError(t, err)

	pool := NewWorkerPool(3, log)
	err = pool.Start()
	require.NoError(t, err)
	defer pool.Stop()

	// Submit tasks
	var wg sync.WaitGroup
	results := make([]int, 5)

	for i := 0; i < 5; i++ {
		wg.Add(1)
		idx := i
		err := pool.SubmitFunc(fmt.Sprintf("task-%d", i), 1, func() error {
			results[idx] = idx * 2
			wg.Done()
			return nil
		})
		require.NoError(t, err)
	}

	// Wait for completion
	wg.Wait()

	// Check results
	for i := 0; i < 5; i++ {
		assert.Equal(t, i*2, results[i])
	}

	// Get stats
	stats := pool.GetStats()
	assert.NotNil(t, stats)
	assert.Equal(t, 3, stats.Total)
	assert.GreaterOrEqual(t, stats.Processed, int64(5))
}

func TestWorkerPoolPriority(t *testing.T) {
	log, err := logger.New(false, false, "iso8601")
	require.NoError(t, err)

	pool := NewWorkerPool(1, log)
	err = pool.Start()
	require.NoError(t, err)
	defer pool.Stop()

	// Create priority queue
	pq := &PriorityQueue{}

	// Add tasks with different priorities
	pq.Push(&SimpleTask{ID: "low", Priority: 1})
	pq.Push(&SimpleTask{ID: "high", Priority: 10})
	pq.Push(&SimpleTask{ID: "medium", Priority: 5})

	// Pop should return in priority order
	task1 := pq.Pop()
	assert.Equal(t, "high", task1.GetID())

	task2 := pq.Pop()
	assert.Equal(t, "medium", task2.GetID())

	task3 := pq.Pop()
	assert.Equal(t, "low", task3.GetID())
}

func TestWorkerPoolWaitForCompletion(t *testing.T) {
	log, err := logger.New(false, false, "iso8601")
	require.NoError(t, err)

	pool := NewWorkerPool(2, log)
	err = pool.Start()
	require.NoError(t, err)
	defer pool.Stop()

	// Submit tasks that take some time
	for i := 0; i < 5; i++ {
		err := pool.SubmitFunc(fmt.Sprintf("task-%d", i), 1, func() error {
			time.Sleep(50 * time.Millisecond)
			return nil
		})
		require.NoError(t, err)
	}

	// Wait for completion
	err = pool.WaitForCompletion(5 * time.Second)
	assert.NoError(t, err)

	// All tasks should be processed
	stats := pool.GetStats()
	assert.Equal(t, 0, stats.Queued)
	assert.Equal(t, 0, stats.Active)
}

func TestOptimizer(t *testing.T) {
	log, err := logger.New(false, false, "iso8601")
	require.NoError(t, err)

	config := &OptimizerConfig{
		EnableCaching:    true,
		CacheSize:        100,
		CacheTTL:         1 * time.Hour,
		EnablePooling:    true,
		MaxConnections:   10,
		MaxWorkers:       5,
		EnableGCTuning:   true,
		GCPercent:        100,
		EnableProfiling:  false,
		ProfileInterval:  30 * time.Second,
	}

	optimizer := NewOptimizer(config, log)
	err = optimizer.Start()
	require.NoError(t, err)
	defer optimizer.Stop()

	// Test cache access
	cache := optimizer.GetCache()
	assert.NotNil(t, cache)

	// Test connection pool access
	connPool := optimizer.GetConnectionPool()
	assert.NotNil(t, connPool)

	// Test worker pool access
	workerPool := optimizer.GetWorkerPool()
	assert.NotNil(t, workerPool)

	// Get optimizer stats
	stats := optimizer.GetStats()
	assert.NotNil(t, stats)
	assert.NotZero(t, stats.Timestamp)
}

func TestGCTuner(t *testing.T) {
	log, err := logger.New(false, false, "iso8601")
	require.NoError(t, err)

	tuner := NewGCTuner(50, log)
	tuner.Start()
	defer tuner.Stop()

	// GC tuning should be applied
	// (actual GC percent check would require runtime introspection)
}