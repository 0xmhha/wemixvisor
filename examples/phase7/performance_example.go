//go:build ignore
// +build ignore

package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net"
	"runtime"
	"sync"
	"time"

	"github.com/wemix/wemixvisor/internal/performance"
	"github.com/wemix/wemixvisor/pkg/logger"
)

// Example demonstrating performance optimization features
func main() {
	// Create logger
	logger, err := logger.New(false, false, "iso8601")
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}

	fmt.Println("=== Wemixvisor Performance Optimization Example ===\n")

	// Example 1: Cache Usage
	fmt.Println("1. Cache Example:")
	demonstrateCache(logger)

	// Example 2: Connection Pool
	fmt.Println("\n2. Connection Pool Example:")
	demonstrateConnectionPool(logger)

	// Example 3: Worker Pool
	fmt.Println("\n3. Worker Pool Example:")
	demonstrateWorkerPool(logger)

	// Example 4: Full Optimizer
	fmt.Println("\n4. Full Optimizer Example:")
	demonstrateOptimizer(logger)

	// Example 5: Memory Management
	fmt.Println("\n5. Memory Management Example:")
	demonstrateMemoryManagement()

	fmt.Println("\n=== Examples Completed ===")
}

// demonstrateCache shows cache usage
func demonstrateCache(logger *logger.Logger) {
	cache := performance.NewCache(100, 5*time.Minute, logger)
	if err := cache.Start(); err != nil {
		log.Fatalf("Failed to start cache: %v", err)
	}
	defer cache.Stop()

	// Store some data
	cache.Set("user:1", map[string]string{"name": "Alice", "role": "admin"}, 1)
	cache.Set("user:2", map[string]string{"name": "Bob", "role": "user"}, 1)
	cache.Set("config:app", map[string]interface{}{"version": "1.0", "debug": false}, 1)

	// Retrieve data
	if val, found := cache.Get("user:1"); found {
		fmt.Printf("  Retrieved from cache: %v\n", val)
	}

	// Simulate cache misses
	for i := 0; i < 5; i++ {
		cache.Get(fmt.Sprintf("nonexistent:%d", i))
	}

	// Get cache statistics
	stats := cache.GetStats()
	fmt.Printf("  Cache Stats - Hits: %d, Misses: %d, Hit Rate: %.2f%%\n",
		stats.Hits, stats.Misses, stats.HitRate*100)
}

// demonstrateConnectionPool shows connection pool usage
func demonstrateConnectionPool(logger *logger.Logger) {
	pool := performance.NewConnectionPool(5, logger)

	// Set up mock connection factory
	pool.SetFactory(func() (net.Conn, error) {
		// In real usage, this would create actual network connections
		return &mockConn{id: rand.Intn(1000)}, nil
	})

	if err := pool.Start(); err != nil {
		log.Fatalf("Failed to start connection pool: %v", err)
	}
	defer pool.Stop()

	// Simulate concurrent connection usage
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			ctx := context.Background()
			conn, err := pool.Get(ctx)
			if err != nil {
				fmt.Printf("  Worker %d: Failed to get connection: %v\n", id, err)
				return
			}

			// Simulate work
			time.Sleep(100 * time.Millisecond)
			fmt.Printf("  Worker %d: Using connection\n", id)

			// Return connection to pool
			pool.Put(conn)
		}(i)
	}

	wg.Wait()

	// Get pool statistics
	stats := pool.GetStats()
	fmt.Printf("  Pool Stats - Active: %d, Idle: %d, Created: %d\n",
		stats.Active, stats.Idle, stats.Created)
}

// demonstrateWorkerPool shows worker pool usage
func demonstrateWorkerPool(logger *logger.Logger) {
	pool := performance.NewWorkerPool(3, logger)
	if err := pool.Start(); err != nil {
		log.Fatalf("Failed to start worker pool: %v", err)
	}
	defer pool.Stop()

	// Submit tasks
	var results sync.Map
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		taskID := fmt.Sprintf("task-%d", i)
		taskNum := i

		err := pool.SubmitFunc(taskID, 1, func() error {
			defer wg.Done()

			// Simulate work
			time.Sleep(50 * time.Millisecond)
			result := taskNum * taskNum
			results.Store(taskID, result)
			fmt.Printf("  Task %s completed: %d^2 = %d\n", taskID, taskNum, result)
			return nil
		})

		if err != nil {
			wg.Done()
			fmt.Printf("  Failed to submit task %s: %v\n", taskID, err)
		}
	}

	// Wait for completion
	wg.Wait()

	// Wait for all tasks to complete
	if err := pool.WaitForCompletion(5 * time.Second); err != nil {
		fmt.Printf("  Warning: %v\n", err)
	}

	// Get pool statistics
	stats := pool.GetStats()
	fmt.Printf("  Worker Pool Stats - Processed: %d, Failed: %d, Avg Time: %dms\n",
		stats.Processed, stats.Failed, stats.AvgTime)
}

// demonstrateOptimizer shows full optimizer usage
func demonstrateOptimizer(logger *logger.Logger) {
	config := &performance.OptimizerConfig{
		EnableCaching:   true,
		CacheSize:       100,
		CacheTTL:        5 * time.Minute,
		EnablePooling:   true,
		MaxConnections:  10,
		MaxWorkers:      5,
		EnableGCTuning:  true,
		GCPercent:       100,
		EnableProfiling: false,
	}

	optimizer := performance.NewOptimizer(config, logger)
	if err := optimizer.Start(); err != nil {
		log.Fatalf("Failed to start optimizer: %v", err)
	}
	defer optimizer.Stop()

	// Use cache through optimizer
	cache := optimizer.GetCache()
	if cache != nil {
		cache.Set("optimized:1", "value1", 1)
		if val, found := cache.Get("optimized:1"); found {
			fmt.Printf("  Cache working: %v\n", val)
		}
	}

	// Use worker pool through optimizer
	workerPool := optimizer.GetWorkerPool()
	if workerPool != nil {
		err := workerPool.SubmitFunc("optimize-task", 1, func() error {
			fmt.Println("  Task executed through optimizer")
			return nil
		})
		if err != nil {
			fmt.Printf("  Failed to submit task: %v\n", err)
		}

		// Wait for task completion
		time.Sleep(100 * time.Millisecond)
	}

	// Get optimization statistics
	stats := optimizer.GetStats()
	fmt.Printf("  Optimizer Stats:\n")
	fmt.Printf("    Memory Alloc: %d MB\n", stats.MemoryAlloc/1024/1024)
	fmt.Printf("    Goroutines: %d\n", stats.NumGoroutine)
	fmt.Printf("    Cache Hit Rate: %.2f%%\n", stats.CacheHitRate*100)
	fmt.Printf("    Workers Active: %d\n", stats.WorkersActive)
}

// demonstrateMemoryManagement shows memory management techniques
func demonstrateMemoryManagement() {
	// Force GC to get baseline
	runtime.GC()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	baseline := m.Alloc

	// Allocate memory
	data := make([][]byte, 100)
	for i := range data {
		data[i] = make([]byte, 1024*10) // 10KB each
	}

	runtime.ReadMemStats(&m)
	allocated := m.Alloc - baseline
	fmt.Printf("  Allocated: %d KB\n", allocated/1024)

	// Clear references and force GC
	data = nil
	runtime.GC()
	runtime.ReadMemStats(&m)
	afterGC := m.Alloc

	fmt.Printf("  After GC: %d KB\n", (afterGC-baseline)/1024)
	fmt.Printf("  Number of GC cycles: %d\n", m.NumGC)

	// Demonstrate GC tuning
	oldPercent := runtime.SetGCPercent(50) // More aggressive GC
	fmt.Printf("  Changed GC percent from %d to 50\n", oldPercent)

	// Restore original GC percent
	runtime.SetGCPercent(oldPercent)
}

// mockConn is a mock connection for testing
type mockConn struct {
	id     int
	closed bool
	mu     sync.Mutex
}

func (m *mockConn) Read(b []byte) (n int, err error)  { return 0, nil }
func (m *mockConn) Write(b []byte) (n int, err error) { return len(b), nil }
func (m *mockConn) Close() error {
	m.mu.Lock()
	m.closed = true
	m.mu.Unlock()
	return nil
}
func (m *mockConn) LocalAddr() net.Addr                { return nil }
func (m *mockConn) RemoteAddr() net.Addr               { return nil }
func (m *mockConn) SetDeadline(t time.Time) error      { return nil }
func (m *mockConn) SetReadDeadline(t time.Time) error  { return nil }
func (m *mockConn) SetWriteDeadline(t time.Time) error { return nil }
