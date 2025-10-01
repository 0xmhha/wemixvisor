package performance

import (
	"context"
	"errors"
	"net"
	"sync"
	"time"

	"github.com/wemix/wemixvisor/pkg/logger"
)

// ConnectionPool manages a pool of reusable connections
type ConnectionPool struct {
	maxSize      int
	idleTimeout  time.Duration
	connections  chan *PooledConnection
	factory      ConnectionFactory
	mu           sync.RWMutex
	logger       *logger.Logger
	stats        *PoolStats
	closed       bool
	activeCount  int32
}

// PooledConnection represents a pooled connection
type PooledConnection struct {
	conn      net.Conn
	pool      *ConnectionPool
	createdAt time.Time
	lastUsed  time.Time
	inUse     bool
}

// ConnectionFactory creates new connections
type ConnectionFactory func() (net.Conn, error)

// PoolStats holds connection pool statistics
type PoolStats struct {
	Active    int   `json:"active"`
	Idle      int   `json:"idle"`
	Total     int   `json:"total"`
	Created   int64 `json:"created"`
	Destroyed int64 `json:"destroyed"`
	Timeouts  int64 `json:"timeouts"`
	mu        sync.RWMutex
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(maxSize int, logger *logger.Logger) *ConnectionPool {
	return &ConnectionPool{
		maxSize:     maxSize,
		idleTimeout: 5 * time.Minute,
		connections: make(chan *PooledConnection, maxSize),
		logger:      logger,
		stats:       &PoolStats{},
	}
}

// SetFactory sets the connection factory
func (p *ConnectionPool) SetFactory(factory ConnectionFactory) {
	p.factory = factory
}

// Start starts the connection pool
func (p *ConnectionPool) Start() error {
	if p.factory == nil {
		return errors.New("connection factory not set")
	}

	// Start cleanup goroutine
	go p.cleanupLoop()

	p.logger.Info("Connection pool started")
	return nil
}

// Stop stops the connection pool
func (p *ConnectionPool) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return
	}

	p.closed = true
	close(p.connections)

	// Close all connections
	for conn := range p.connections {
		if conn.conn != nil {
			conn.conn.Close()
		}
	}

	p.logger.Info("Connection pool stopped")
}

// Get retrieves a connection from the pool
func (p *ConnectionPool) Get(ctx context.Context) (*PooledConnection, error) {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return nil, errors.New("pool is closed")
	}
	p.mu.RUnlock()

	// Try to get an existing connection
	select {
	case conn := <-p.connections:
		if conn != nil && p.isConnectionValid(conn) {
			conn.inUse = true
			conn.lastUsed = time.Now()
			p.updateStats(1, -1)
			return conn, nil
		}
		// Connection is invalid, destroy it
		if conn != nil && conn.conn != nil {
			conn.conn.Close()
			p.recordDestroyed()
		}
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		// No connections available
	}

	// Create a new connection
	if p.getActiveCount() < p.maxSize {
		netConn, err := p.factory()
		if err != nil {
			return nil, err
		}

		conn := &PooledConnection{
			conn:      netConn,
			pool:      p,
			createdAt: time.Now(),
			lastUsed:  time.Now(),
			inUse:     true,
		}

		p.updateStats(1, 0)
		p.recordCreated()
		return conn, nil
	}

	// Wait for a connection to become available
	select {
	case conn := <-p.connections:
		if conn != nil && p.isConnectionValid(conn) {
			conn.inUse = true
			conn.lastUsed = time.Now()
			p.updateStats(1, -1)
			return conn, nil
		}
		// Connection is invalid, recurse to try again
		return p.Get(ctx)
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(30 * time.Second):
		p.recordTimeout()
		return nil, errors.New("timeout waiting for connection")
	}
}

// Put returns a connection to the pool
func (p *ConnectionPool) Put(conn *PooledConnection) {
	if conn == nil {
		return
	}

	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		if conn.conn != nil {
			conn.conn.Close()
		}
		return
	}
	p.mu.RUnlock()

	conn.inUse = false
	conn.lastUsed = time.Now()

	// Try to return to pool
	select {
	case p.connections <- conn:
		p.updateStats(-1, 1)
	default:
		// Pool is full, close the connection
		if conn.conn != nil {
			conn.conn.Close()
			p.recordDestroyed()
		}
		p.updateStats(-1, 0)
	}
}

// Close closes a pooled connection
func (pc *PooledConnection) Close() error {
	pc.pool.Put(pc)
	return nil
}

// cleanupLoop periodically removes idle connections
func (p *ConnectionPool) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		p.mu.RLock()
		if p.closed {
			p.mu.RUnlock()
			return
		}
		p.mu.RUnlock()

		p.cleanup()
	}
}

// cleanup removes expired connections
func (p *ConnectionPool) cleanup() {
	now := time.Now()
	cleaned := 0

	// Temporarily store valid connections
	validConns := make([]*PooledConnection, 0)

	// Drain the channel and check each connection
	for {
		select {
		case conn := <-p.connections:
			if conn != nil && now.Sub(conn.lastUsed) < p.idleTimeout {
				validConns = append(validConns, conn)
			} else {
				if conn != nil && conn.conn != nil {
					conn.conn.Close()
					p.recordDestroyed()
				}
				cleaned++
			}
		default:
			// No more connections to check
			goto done
		}
	}

done:
	// Return valid connections to the pool
	for _, conn := range validConns {
		select {
		case p.connections <- conn:
			// Successfully returned to pool
		default:
			// Pool is full, close the connection
			if conn.conn != nil {
				conn.conn.Close()
				p.recordDestroyed()
			}
		}
	}

	if cleaned > 0 {
		p.logger.Debug("Cleaned up idle connections")
	}
}

// isConnectionValid checks if a connection is still valid
func (p *ConnectionPool) isConnectionValid(conn *PooledConnection) bool {
	if conn == nil || conn.conn == nil {
		return false
	}

	// Check if connection has been idle too long
	if time.Since(conn.lastUsed) > p.idleTimeout {
		return false
	}

	// Try to set a deadline to test if connection is alive
	if err := conn.conn.SetDeadline(time.Now().Add(1 * time.Millisecond)); err != nil {
		return false
	}

	// Reset deadline
	conn.conn.SetDeadline(time.Time{})
	return true
}

// Optimize optimizes the connection pool
func (p *ConnectionPool) Optimize() {
	p.cleanup()

	stats := p.GetStats()
	if stats.Active > p.maxSize*8/10 {
		p.logger.Warn("Connection pool near capacity")
	}
}

// GetStats returns pool statistics
func (p *ConnectionPool) GetStats() *PoolStats {
	p.stats.mu.RLock()
	defer p.stats.mu.RUnlock()

	idle := len(p.connections)
	active := p.getActiveCount()

	return &PoolStats{
		Active:    active,
		Idle:      idle,
		Total:     active + idle,
		Created:   p.stats.Created,
		Destroyed: p.stats.Destroyed,
		Timeouts:  p.stats.Timeouts,
	}
}

// Helper methods for stats tracking
func (p *ConnectionPool) updateStats(activeDelta, idleDelta int) {
	p.stats.mu.Lock()
	p.stats.Active += activeDelta
	p.stats.Idle += idleDelta
	p.stats.mu.Unlock()
}

func (p *ConnectionPool) recordCreated() {
	p.stats.mu.Lock()
	p.stats.Created++
	p.stats.mu.Unlock()
}

func (p *ConnectionPool) recordDestroyed() {
	p.stats.mu.Lock()
	p.stats.Destroyed++
	p.stats.mu.Unlock()
}

func (p *ConnectionPool) recordTimeout() {
	p.stats.mu.Lock()
	p.stats.Timeouts++
	p.stats.mu.Unlock()
}

func (p *ConnectionPool) getActiveCount() int {
	p.stats.mu.RLock()
	defer p.stats.mu.RUnlock()
	return p.stats.Active
}