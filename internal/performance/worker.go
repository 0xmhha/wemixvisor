package performance

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wemix/wemixvisor/pkg/logger"
)

// WorkerPool manages a pool of workers for concurrent task execution
type WorkerPool struct {
	maxWorkers   int
	taskQueue    chan Task
	workers      []*Worker
	wg           sync.WaitGroup
	mu           sync.RWMutex
	logger       *logger.Logger
	stats        *WorkerStats
	ctx          context.Context
	cancel       context.CancelFunc
	started      bool
}

// Worker represents a single worker in the pool
type Worker struct {
	id       int
	pool     *WorkerPool
	taskChan chan Task
	quit     chan struct{}
	active   atomic.Bool
}

// Task represents a unit of work
type Task interface {
	Execute() error
	GetID() string
	GetPriority() int
}

// WorkerStats holds worker pool statistics
type WorkerStats struct {
	Active     int   `json:"active"`
	Idle       int   `json:"idle"`
	Total      int   `json:"total"`
	Queued     int   `json:"queued"`
	Processed  int64 `json:"processed"`
	Failed     int64 `json:"failed"`
	AvgTime    int64 `json:"avg_time_ms"`
	mu         sync.RWMutex
	totalTime  int64
}

// SimpleTask is a basic implementation of Task
type SimpleTask struct {
	ID       string
	Priority int
	Func     func() error
}

// Execute runs the task
func (t *SimpleTask) Execute() error {
	if t.Func != nil {
		return t.Func()
	}
	return nil
}

// GetID returns the task ID
func (t *SimpleTask) GetID() string {
	return t.ID
}

// GetPriority returns the task priority
func (t *SimpleTask) GetPriority() int {
	return t.Priority
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(maxWorkers int, logger *logger.Logger) *WorkerPool {
	return &WorkerPool{
		maxWorkers: maxWorkers,
		taskQueue:  make(chan Task, maxWorkers*10),
		workers:    make([]*Worker, 0, maxWorkers),
		logger:     logger,
		stats:      &WorkerStats{},
	}
}

// Start starts the worker pool
func (wp *WorkerPool) Start() error {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if wp.started {
		return errors.New("worker pool already started")
	}

	wp.ctx, wp.cancel = context.WithCancel(context.Background())

	// Create and start workers
	for i := 0; i < wp.maxWorkers; i++ {
		worker := &Worker{
			id:       i,
			pool:     wp,
			taskChan: make(chan Task, 1),
			quit:     make(chan struct{}),
		}
		wp.workers = append(wp.workers, worker)
		wp.wg.Add(1)
		go worker.run()
	}

	// Start dispatcher
	go wp.dispatch()

	wp.started = true
	wp.logger.Info("Worker pool started")
	return nil
}

// Stop stops the worker pool
func (wp *WorkerPool) Stop() {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if !wp.started {
		return
	}

	// Cancel context
	if wp.cancel != nil {
		wp.cancel()
	}

	// Stop all workers
	for _, worker := range wp.workers {
		close(worker.quit)
	}

	// Wait for all workers to finish
	wp.wg.Wait()

	// Close task queue
	close(wp.taskQueue)

	wp.started = false
	wp.logger.Info("Worker pool stopped")
}

// Submit submits a task to the worker pool
func (wp *WorkerPool) Submit(task Task) error {
	wp.mu.RLock()
	if !wp.started {
		wp.mu.RUnlock()
		return errors.New("worker pool not started")
	}
	wp.mu.RUnlock()

	select {
	case wp.taskQueue <- task:
		wp.updateQueuedCount(1)
		return nil
	case <-time.After(5 * time.Second):
		return errors.New("timeout submitting task")
	}
}

// SubmitFunc submits a function as a task
func (wp *WorkerPool) SubmitFunc(id string, priority int, fn func() error) error {
	task := &SimpleTask{
		ID:       id,
		Priority: priority,
		Func:     fn,
	}
	return wp.Submit(task)
}

// dispatch distributes tasks to available workers
func (wp *WorkerPool) dispatch() {
	for {
		select {
		case <-wp.ctx.Done():
			return
		case task := <-wp.taskQueue:
			if task == nil {
				continue
			}

			wp.updateQueuedCount(-1)

			// Find an available worker
			assigned := false
			for _, worker := range wp.workers {
				if !worker.active.Load() {
					select {
					case worker.taskChan <- task:
						assigned = true
						break
					default:
						continue
					}
				}
				if assigned {
					break
				}
			}

			// If no worker available, put back in queue
			if !assigned {
				select {
				case wp.taskQueue <- task:
					wp.updateQueuedCount(1)
				case <-time.After(1 * time.Second):
					wp.logger.Warn("Failed to reassign task")
				}
			}
		}
	}
}

// run is the main loop for a worker
func (w *Worker) run() {
	defer w.pool.wg.Done()

	for {
		select {
		case <-w.quit:
			return
		case task := <-w.taskChan:
			if task == nil {
				continue
			}

			w.active.Store(true)
			w.pool.updateActiveCount(1)

			startTime := time.Now()
			err := w.executeTask(task)
			duration := time.Since(startTime)

			w.pool.recordTaskCompletion(duration, err)

			w.active.Store(false)
			w.pool.updateActiveCount(-1)
		}
	}
}

// executeTask executes a single task
func (w *Worker) executeTask(task Task) error {
	defer func() {
		if r := recover(); r != nil {
			w.pool.logger.Error("Task panic occurred")
		}
	}()

	err := task.Execute()
	if err != nil {
		w.pool.logger.Debug("Task failed")
		return err
	}

	w.pool.logger.Debug("Task completed")
	return nil
}

// Optimize optimizes the worker pool
func (wp *WorkerPool) Optimize() {
	stats := wp.GetStats()

	// Check if we need more workers
	if stats.Queued > wp.maxWorkers*2 && stats.Active == wp.maxWorkers {
		wp.logger.Warn("Worker pool saturated")
	}

	// Check average processing time
	if stats.AvgTime > 1000 { // > 1 second
		wp.logger.Warn("Slow task processing detected")
	}
}

// GetStats returns worker pool statistics
func (wp *WorkerPool) GetStats() *WorkerStats {
	wp.stats.mu.RLock()
	defer wp.stats.mu.RUnlock()

	activeCount := 0
	for _, worker := range wp.workers {
		if worker.active.Load() {
			activeCount++
		}
	}

	avgTime := int64(0)
	if wp.stats.Processed > 0 {
		avgTime = wp.stats.totalTime / wp.stats.Processed
	}

	return &WorkerStats{
		Active:    activeCount,
		Idle:      len(wp.workers) - activeCount,
		Total:     len(wp.workers),
		Queued:    len(wp.taskQueue),
		Processed: wp.stats.Processed,
		Failed:    wp.stats.Failed,
		AvgTime:   avgTime,
	}
}

// WaitForCompletion waits for all queued tasks to complete
func (wp *WorkerPool) WaitForCompletion(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return errors.New("timeout waiting for completion")
		case <-ticker.C:
			stats := wp.GetStats()
			if stats.Queued == 0 && stats.Active == 0 {
				return nil
			}
		}
	}
}

// Helper methods for stats tracking
func (wp *WorkerPool) updateActiveCount(delta int) {
	wp.stats.mu.Lock()
	wp.stats.Active += delta
	if delta > 0 {
		wp.stats.Idle--
	} else {
		wp.stats.Idle++
	}
	wp.stats.mu.Unlock()
}

func (wp *WorkerPool) updateQueuedCount(delta int) {
	wp.stats.mu.Lock()
	wp.stats.Queued += delta
	wp.stats.mu.Unlock()
}

func (wp *WorkerPool) recordTaskCompletion(duration time.Duration, err error) {
	wp.stats.mu.Lock()
	wp.stats.Processed++
	if err != nil {
		wp.stats.Failed++
	}
	wp.stats.totalTime += duration.Milliseconds()
	wp.stats.mu.Unlock()
}

// PriorityTask wraps a task with priority
type PriorityTask struct {
	Task
	priority int
}

// PriorityQueue implements a priority queue for tasks
type PriorityQueue struct {
	tasks []PriorityTask
	mu    sync.Mutex
}

// Push adds a task to the priority queue
func (pq *PriorityQueue) Push(task Task) {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	pt := PriorityTask{
		Task:     task,
		priority: task.GetPriority(),
	}

	// Insert in priority order
	inserted := false
	for i, existing := range pq.tasks {
		if pt.priority > existing.priority {
			pq.tasks = append(pq.tasks[:i], append([]PriorityTask{pt}, pq.tasks[i:]...)...)
			inserted = true
			break
		}
	}

	if !inserted {
		pq.tasks = append(pq.tasks, pt)
	}
}

// Pop removes and returns the highest priority task
func (pq *PriorityQueue) Pop() Task {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	if len(pq.tasks) == 0 {
		return nil
	}

	task := pq.tasks[0]
	pq.tasks = pq.tasks[1:]
	return task.Task
}

// Len returns the number of tasks in the queue
func (pq *PriorityQueue) Len() int {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	return len(pq.tasks)
}