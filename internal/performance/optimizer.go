package performance

import (
	"context"
	"fmt"
	"net"
	"runtime"
	"runtime/debug"
	"sync"
	"time"

	"github.com/wemix/wemixvisor/pkg/logger"
)

// Optimizer manages performance optimization for the system
type Optimizer struct {
	logger            *logger.Logger
	config            *OptimizerConfig
	cache             *Cache
	connectionPool    *ConnectionPool
	workerPool        *WorkerPool
	mu                sync.RWMutex
	ctx               context.Context
	cancel            context.CancelFunc
	gcTuner           *GCTuner
	profileCollector  *ProfileCollector
	profiler          *Profiler
}

// OptimizerConfig represents optimizer configuration
type OptimizerConfig struct {
	EnableCaching       bool          `json:"enable_caching"`
	CacheSize           int           `json:"cache_size"`
	CacheTTL            time.Duration `json:"cache_ttl"`
	EnablePooling       bool          `json:"enable_pooling"`
	MaxConnections      int           `json:"max_connections"`
	MaxWorkers          int           `json:"max_workers"`
	EnableGCTuning      bool          `json:"enable_gc_tuning"`
	GCPercent           int           `json:"gc_percent"`
	EnableProfiling     bool          `json:"enable_profiling"`
	ProfileInterval     time.Duration `json:"profile_interval"`
	ProfileDir          string        `json:"profile_dir"`
}

// NewOptimizer creates a new performance optimizer
func NewOptimizer(config *OptimizerConfig, logger *logger.Logger) *Optimizer {
	return &Optimizer{
		logger: logger,
		config: config,
	}
}

// Start starts the performance optimizer
func (o *Optimizer) Start() error {
	o.ctx, o.cancel = context.WithCancel(context.Background())

	// Initialize cache
	if o.config.EnableCaching {
		o.cache = NewCache(o.config.CacheSize, o.config.CacheTTL, o.logger)
		if err := o.cache.Start(); err != nil {
			return err
		}
		o.logger.Info("Cache initialized")
	}

	// Initialize connection pool
	if o.config.EnablePooling {
		o.connectionPool = NewConnectionPool(o.config.MaxConnections, o.logger)
		// Set a default factory if not already set
		o.connectionPool.SetFactory(func() (net.Conn, error) {
			// Default to TCP connection to localhost
			return net.Dial("tcp", "localhost:8080")
		})
		if err := o.connectionPool.Start(); err != nil {
			return err
		}
		o.logger.Info("Connection pool initialized")

		// Initialize worker pool
		o.workerPool = NewWorkerPool(o.config.MaxWorkers, o.logger)
		if err := o.workerPool.Start(); err != nil {
			return err
		}
		o.logger.Info("Worker pool initialized")
	}

	// Initialize GC tuning
	if o.config.EnableGCTuning {
		o.gcTuner = NewGCTuner(o.config.GCPercent, o.logger)
		o.gcTuner.Start()
		o.logger.Info("GC tuning enabled")
	}

	// Initialize profiling
	if o.config.EnableProfiling {
		// Set default profile directory if not specified
		profileDir := o.config.ProfileDir
		if profileDir == "" {
			profileDir = "./profiles"
		}

		o.profiler = NewProfiler(profileDir, o.logger)
		if err := o.profiler.Start(); err != nil {
			return fmt.Errorf("failed to start profiler: %w", err)
		}

		o.profileCollector = NewProfileCollector(o.config.ProfileInterval, o.logger)
		o.profileCollector.profiler = o.profiler // Link profiler to collector
		o.profileCollector.Start()
		o.logger.Info("Profiling enabled")
	}

	// Start optimization loop
	go o.optimizationLoop()

	o.logger.Info("Performance optimizer started")
	return nil
}

// Stop stops the performance optimizer
func (o *Optimizer) Stop() error {
	if o.cancel != nil {
		o.cancel()
	}

	if o.cache != nil {
		o.cache.Stop()
	}

	if o.connectionPool != nil {
		o.connectionPool.Stop()
	}

	if o.workerPool != nil {
		o.workerPool.Stop()
	}

	if o.gcTuner != nil {
		o.gcTuner.Stop()
	}

	if o.profileCollector != nil {
		o.profileCollector.Stop()
	}

	if o.profiler != nil {
		if err := o.profiler.Stop(); err != nil {
			o.logger.Error("Failed to stop profiler", "error", err.Error())
		}
	}

	o.logger.Info("Performance optimizer stopped")
	return nil
}

// optimizationLoop runs periodic optimization tasks
func (o *Optimizer) optimizationLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-o.ctx.Done():
			return
		case <-ticker.C:
			o.performOptimizations()
		}
	}
}

// performOptimizations executes optimization tasks
func (o *Optimizer) performOptimizations() {
	// Optimize memory usage
	o.optimizeMemory()

	// Optimize goroutines
	o.optimizeGoroutines()

	// Optimize cache
	if o.cache != nil {
		o.cache.Optimize()
	}

	// Optimize pools
	if o.connectionPool != nil {
		o.connectionPool.Optimize()
	}
	if o.workerPool != nil {
		o.workerPool.Optimize()
	}
}

// optimizeMemory performs memory optimization
func (o *Optimizer) optimizeMemory() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Log current memory usage
	o.logger.Debug("Memory stats collected")

	// Force GC if memory usage is high
	if m.Alloc > 500*1024*1024 { // 500MB threshold
		runtime.GC()
		debug.FreeOSMemory()
		o.logger.Info("Forced garbage collection due to high memory usage")
	}
}

// optimizeGoroutines monitors and optimizes goroutine usage
func (o *Optimizer) optimizeGoroutines() {
	numGoroutines := runtime.NumGoroutine()

	if numGoroutines > 1000 {
		o.logger.Warn("High number of goroutines detected")
	}
}

// GetCache returns the cache instance
func (o *Optimizer) GetCache() *Cache {
	return o.cache
}

// GetConnectionPool returns the connection pool
func (o *Optimizer) GetConnectionPool() *ConnectionPool {
	return o.connectionPool
}

// GetWorkerPool returns the worker pool
func (o *Optimizer) GetWorkerPool() *WorkerPool {
	return o.workerPool
}

// GetProfiler returns the profiler instance
func (o *Optimizer) GetProfiler() *Profiler {
	return o.profiler
}

// GetStats returns optimization statistics
func (o *Optimizer) GetStats() *OptimizationStats {
	stats := &OptimizationStats{
		Timestamp: time.Now(),
	}

	// Memory stats
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	stats.MemoryAlloc = m.Alloc
	stats.MemoryTotalAlloc = m.TotalAlloc
	stats.MemorySys = m.Sys
	stats.NumGC = m.NumGC
	stats.NumGoroutine = runtime.NumGoroutine()

	// Cache stats
	if o.cache != nil {
		cacheStats := o.cache.GetStats()
		stats.CacheHits = cacheStats.Hits
		stats.CacheMisses = cacheStats.Misses
		stats.CacheSize = cacheStats.Size
		stats.CacheHitRate = cacheStats.HitRate
	}

	// Pool stats
	if o.connectionPool != nil {
		poolStats := o.connectionPool.GetStats()
		stats.PoolActive = poolStats.Active
		stats.PoolIdle = poolStats.Idle
		stats.PoolTotal = poolStats.Total
	}

	if o.workerPool != nil {
		workerStats := o.workerPool.GetStats()
		stats.WorkersActive = workerStats.Active
		stats.WorkersIdle = workerStats.Idle
		stats.WorkersTotal = workerStats.Total
		stats.TasksQueued = workerStats.Queued
		stats.TasksProcessed = workerStats.Processed
	}

	return stats
}

// OptimizationStats holds optimization statistics
type OptimizationStats struct {
	Timestamp        time.Time `json:"timestamp"`
	MemoryAlloc      uint64    `json:"memory_alloc"`
	MemoryTotalAlloc uint64    `json:"memory_total_alloc"`
	MemorySys        uint64    `json:"memory_sys"`
	NumGC            uint32    `json:"num_gc"`
	NumGoroutine     int       `json:"num_goroutine"`
	CacheHits        int64     `json:"cache_hits"`
	CacheMisses      int64     `json:"cache_misses"`
	CacheSize        int       `json:"cache_size"`
	CacheHitRate     float64   `json:"cache_hit_rate"`
	PoolActive       int       `json:"pool_active"`
	PoolIdle         int       `json:"pool_idle"`
	PoolTotal        int       `json:"pool_total"`
	WorkersActive    int       `json:"workers_active"`
	WorkersIdle      int       `json:"workers_idle"`
	WorkersTotal     int       `json:"workers_total"`
	TasksQueued      int       `json:"tasks_queued"`
	TasksProcessed   int64     `json:"tasks_processed"`
}

// GCTuner manages garbage collection tuning
type GCTuner struct {
	gcPercent int
	logger    *logger.Logger
}

// NewGCTuner creates a new GC tuner
func NewGCTuner(gcPercent int, logger *logger.Logger) *GCTuner {
	return &GCTuner{
		gcPercent: gcPercent,
		logger:    logger,
	}
}

// Start starts GC tuning
func (g *GCTuner) Start() {
	// Set GC percentage
	debug.SetGCPercent(g.gcPercent)

	// Set memory limit if supported
	// debug.SetMemoryLimit(1 << 30) // 1GB limit (Go 1.19+)

	g.logger.Info("GC tuning applied")
}

// Stop stops GC tuning
func (g *GCTuner) Stop() {
	// Reset to default
	debug.SetGCPercent(100)
}

// ProfileCollector collects performance profiles
type ProfileCollector struct {
	interval time.Duration
	logger   *logger.Logger
	stop     chan struct{}
	profiler *Profiler
}

// NewProfileCollector creates a new profile collector
func NewProfileCollector(interval time.Duration, logger *logger.Logger) *ProfileCollector {
	return &ProfileCollector{
		interval: interval,
		logger:   logger,
		stop:     make(chan struct{}),
	}
}

// Start starts profile collection
func (p *ProfileCollector) Start() {
	go p.collectLoop()
}

// Stop stops profile collection
func (p *ProfileCollector) Stop() {
	close(p.stop)
}

// collectLoop runs the profile collection loop
func (p *ProfileCollector) collectLoop() {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-p.stop:
			return
		case <-ticker.C:
			p.collect()
		}
	}
}

// collect performs profile collection
func (p *ProfileCollector) collect() {
	if p.profiler == nil {
		p.logger.Warn("Profiler not initialized")
		return
	}

	p.logger.Debug("Collecting performance profiles")

	// Write all available profiles
	if err := p.profiler.WriteAllProfiles(); err != nil {
		p.logger.Error("Failed to collect profiles", "error", err.Error())
		return
	}

	// Clean old profiles (keep only last 7 days)
	if err := p.profiler.CleanOldProfiles(7 * 24 * time.Hour); err != nil {
		p.logger.Warn("Failed to clean old profiles", "error", err.Error())
	}
}