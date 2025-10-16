package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/wemix/wemixvisor/internal/config"
	"github.com/wemix/wemixvisor/internal/governance"
	"github.com/wemix/wemixvisor/internal/metrics"
	"github.com/wemix/wemixvisor/pkg/logger"
)

// Server represents the API server
type Server struct {
	router    *gin.Engine
	server    *http.Server
	logger    *logger.Logger
	config    *config.Config
	monitor   *governance.Monitor
	collector *metrics.Collector
	auth      *AuthMiddleware
	port      int

	// WebSocket management
	wsClients   map[*websocket.Conn]*WSClient
	wsClientsMu sync.RWMutex
	wsBroadcast chan WSMessage
	wsUpgrader  websocket.Upgrader
}

// WSClient represents a WebSocket client connection
type WSClient struct {
	conn          *websocket.Conn
	send          chan []byte
	subscriptions map[string]bool // topic -> subscribed
	mu            sync.RWMutex
}

// WSMessage represents a WebSocket message
type WSMessage struct {
	Type    string      `json:"type"`
	Topic   string      `json:"topic"`
	Data    interface{} `json:"data"`
	Time    int64       `json:"timestamp"`
}

// WSSubscribeRequest represents a subscription request
type WSSubscribeRequest struct {
	Action string   `json:"action"` // "subscribe" or "unsubscribe"
	Topics []string `json:"topics"` // ["metrics", "logs", "alerts", "proposals"]
}

// NewServer creates a new API server
func NewServer(cfg *config.Config, monitor *governance.Monitor, collector *metrics.Collector, logger *logger.Logger) *Server {
	// Set Gin mode based on environment
	if cfg.Debug {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Add recovery middleware
	router.Use(gin.Recovery())

	// Add logging middleware
	router.Use(ginLogger(logger))

	// Add CORS middleware
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	port := 8080
	if cfg.APIPort > 0 {
		port = cfg.APIPort
	}

	server := &Server{
		router:      router,
		logger:      logger,
		config:      cfg,
		monitor:     monitor,
		collector:   collector,
		port:        port,
		wsClients:   make(map[*websocket.Conn]*WSClient),
		wsBroadcast: make(chan WSMessage, 256),
		wsUpgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins in development
			},
		},
	}

	// Setup routes
	server.setupRoutes()

	// Start WebSocket broadcast handler
	go server.handleWSBroadcast()

	return server
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	// Health check
	s.router.GET("/health", s.healthHandler)
	s.router.GET("/ready", s.readyHandler)

	// API v1 routes
	v1 := s.router.Group("/api/v1")

	// Public routes (no auth required)
	v1.GET("/status", s.getStatus)
	v1.GET("/version", s.getVersion)

	// Protected routes (auth required)
	// Note: Auth middleware would be added here in production
	// v1.Use(s.auth.Authenticate())

	// System routes
	v1.GET("/metrics", s.getMetrics)
	v1.GET("/metrics/snapshot", s.getMetricsSnapshot)

	// Upgrade routes
	v1.GET("/upgrades", s.getUpgrades)
	v1.GET("/upgrades/:id", s.getUpgrade)
	v1.POST("/upgrades", s.scheduleUpgrade)
	v1.DELETE("/upgrades/:id", s.cancelUpgrade)

	// Governance routes
	v1.GET("/governance/proposals", s.getProposals)
	v1.GET("/governance/proposals/:id", s.getProposal)
	v1.GET("/governance/votes", s.getVotes)
	v1.GET("/governance/validators", s.getValidators)

	// Configuration routes
	v1.GET("/config", s.getConfig)
	v1.PUT("/config", s.updateConfig)
	v1.GET("/config/validate", s.validateConfig)

	// Logs routes
	v1.GET("/logs", s.getLogs)
	v1.GET("/logs/stream", s.streamLogs)

	// WebSocket route
	v1.GET("/ws", s.handleWebSocket)
}

// Start starts the API server
func (s *Server) Start() error {
	s.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.port),
		Handler:      s.router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		s.logger.Info("Starting API server", "port", s.port)
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("API server error", "error", err.Error())
		}
	}()

	return nil
}

// Stop stops the API server
func (s *Server) Stop() error {
	if s.server == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.server.Shutdown(ctx); err != nil {
		s.logger.Error("Failed to shutdown API server gracefully", "error", err.Error())
		return err
	}

	s.logger.Info("API server stopped")
	return nil
}

// healthHandler handles health check requests
func (s *Server) healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
	})
}

// readyHandler handles readiness check requests
func (s *Server) readyHandler(c *gin.Context) {
	// Check if monitor is running
	if s.monitor == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  "not_ready",
			"message": "Monitor not initialized",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "ready",
		"timestamp": time.Now().Unix(),
	})
}

// getStatus returns the system status
func (s *Server) getStatus(c *gin.Context) {
	status := gin.H{
		"status":     "running",
		"version":    "0.7.0",
		"uptime":     time.Now().Unix(), // Simplified for demo
		"governance": gin.H{
			"enabled": s.monitor != nil,
		},
		"metrics": gin.H{
			"enabled": s.collector != nil,
		},
	}

	c.JSON(http.StatusOK, status)
}

// getVersion returns version information
func (s *Server) getVersion(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"version":    "0.7.0",
		"api":        "v1",
		"build_time": "2024-01-01T00:00:00Z", // Simplified for demo
		"git_commit": "abc123",               // Simplified for demo
	})
}

// getMetrics returns current metrics
func (s *Server) getMetrics(c *gin.Context) {
	if s.collector == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Metrics collector not available",
		})
		return
	}

	snapshot := s.collector.GetSnapshot()
	if snapshot == nil {
		c.JSON(http.StatusNoContent, gin.H{
			"message": "No metrics available yet",
		})
		return
	}

	c.JSON(http.StatusOK, snapshot)
}

// getMetricsSnapshot returns the latest metrics snapshot
func (s *Server) getMetricsSnapshot(c *gin.Context) {
	if s.collector == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Metrics collector not available",
		})
		return
	}

	snapshot := s.collector.GetSnapshot()
	c.JSON(http.StatusOK, snapshot)
}

// getUpgrades returns the list of upgrades
func (s *Server) getUpgrades(c *gin.Context) {
	if s.monitor == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Monitor not available",
		})
		return
	}

	upgrades, err := s.monitor.GetUpgradeQueue()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"upgrades": upgrades,
		"count":    len(upgrades),
	})
}

// getUpgrade returns a specific upgrade
func (s *Server) getUpgrade(c *gin.Context) {
	upgradeID := c.Param("id")

	// Implementation would fetch specific upgrade
	c.JSON(http.StatusOK, gin.H{
		"id":     upgradeID,
		"status": "scheduled",
		"height": 100000,
	})
}

// scheduleUpgrade schedules a new upgrade
func (s *Server) scheduleUpgrade(c *gin.Context) {
	var req struct {
		Name   string `json:"name" binding:"required"`
		Height int64  `json:"height" binding:"required"`
		Info   string `json:"info"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Implementation would schedule the upgrade
	c.JSON(http.StatusCreated, gin.H{
		"message": "Upgrade scheduled successfully",
		"upgrade": gin.H{
			"name":   req.Name,
			"height": req.Height,
		},
	})
}

// cancelUpgrade cancels a scheduled upgrade
func (s *Server) cancelUpgrade(c *gin.Context) {
	upgradeID := c.Param("id")

	// Implementation would cancel the upgrade
	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Upgrade %s cancelled", upgradeID),
	})
}

// getProposals returns governance proposals
func (s *Server) getProposals(c *gin.Context) {
	if s.monitor == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Monitor not available",
		})
		return
	}

	proposals, err := s.monitor.GetProposals()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"proposals": proposals,
		"count":     len(proposals),
	})
}

// getProposal returns a specific proposal
func (s *Server) getProposal(c *gin.Context) {
	proposalID := c.Param("id")

	if s.monitor == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Monitor not available",
		})
		return
	}

	proposals, err := s.monitor.GetProposals()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Find proposal by ID
	for _, proposal := range proposals {
		if proposal.ID == proposalID {
			c.JSON(http.StatusOK, proposal)
			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{
		"error": "Proposal not found",
	})
}

// getVotes returns voting statistics
func (s *Server) getVotes(c *gin.Context) {
	// Implementation would fetch voting stats
	c.JSON(http.StatusOK, gin.H{
		"total_proposals": 10,
		"voting_active":   2,
		"participation":   "67.5%",
	})
}

// getValidators returns validator information
func (s *Server) getValidators(c *gin.Context) {
	// Implementation would fetch validator info
	c.JSON(http.StatusOK, gin.H{
		"validators": []gin.H{
			{
				"address": "wemix1abc...",
				"status":  "active",
				"power":   1000000,
			},
		},
		"count": 1,
	})
}

// getConfig returns current configuration
func (s *Server) getConfig(c *gin.Context) {
	// Simplified config response
	config := gin.H{
		"home":        s.config.Home,
		"rpc_address": s.config.RPCAddress,
		"debug":       s.config.Debug,
	}

	c.JSON(http.StatusOK, config)
}

// updateConfig updates configuration
func (s *Server) updateConfig(c *gin.Context) {
	var req map[string]interface{}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Implementation would update config
	c.JSON(http.StatusOK, gin.H{
		"message": "Configuration updated successfully",
	})
}

// validateConfig validates configuration
func (s *Server) validateConfig(c *gin.Context) {
	var req map[string]interface{}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Implementation would validate config
	c.JSON(http.StatusOK, gin.H{
		"valid":   true,
		"message": "Configuration is valid",
	})
}

// getLogs returns system logs
func (s *Server) getLogs(c *gin.Context) {
	limit := c.DefaultQuery("limit", "100")
	level := c.DefaultQuery("level", "info")

	// Implementation would fetch logs
	c.JSON(http.StatusOK, gin.H{
		"logs": []gin.H{
			{
				"timestamp": time.Now().Unix(),
				"level":     level,
				"message":   "Sample log entry",
			},
		},
		"limit": limit,
	})
}

// streamLogs streams logs via SSE
func (s *Server) streamLogs(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	// Implementation would stream logs
	c.SSEvent("log", gin.H{
		"timestamp": time.Now().Unix(),
		"level":     "info",
		"message":   "Streaming log entry",
	})
}

// handleWebSocket handles WebSocket connections
func (s *Server) handleWebSocket(c *gin.Context) {
	// Upgrade HTTP connection to WebSocket
	conn, err := s.wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		s.logger.Error("Failed to upgrade WebSocket connection", "error", err.Error())
		return
	}

	// Create new client
	client := &WSClient{
		conn:          conn,
		send:          make(chan []byte, 256),
		subscriptions: make(map[string]bool),
	}

	// Register client
	s.wsClientsMu.Lock()
	s.wsClients[conn] = client
	s.wsClientsMu.Unlock()

	s.logger.Info("WebSocket client connected", "remote_addr", conn.RemoteAddr().String())

	// Send welcome message
	welcome := WSMessage{
		Type:  "connected",
		Topic: "system",
		Data: map[string]interface{}{
			"message": "Connected to Wemixvisor API",
			"version": "0.7.0",
			"topics":  []string{"metrics", "logs", "alerts", "proposals"},
		},
		Time: time.Now().Unix(),
	}
	client.writeMessage(welcome)

	// Start client handlers
	go s.handleWSRead(client)
	go s.handleWSWrite(client)
}

// handleWSRead reads messages from WebSocket client
func (s *Server) handleWSRead(client *WSClient) {
	defer func() {
		s.removeWSClient(client)
		client.conn.Close()
	}()

	client.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	client.conn.SetPongHandler(func(string) error {
		client.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := client.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				s.logger.Error("WebSocket read error", "error", err.Error())
			}
			break
		}

		// Parse subscription request
		var req WSSubscribeRequest
		if err := json.Unmarshal(message, &req); err != nil {
			s.logger.Warn("Invalid WebSocket message", "error", err.Error())
			continue
		}

		// Handle subscription
		s.handleWSSubscription(client, &req)
	}
}

// handleWSWrite writes messages to WebSocket client
func (s *Server) handleWSWrite(client *WSClient) {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		client.conn.Close()
	}()

	for {
		select {
		case message, ok := <-client.send:
			client.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				client.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := client.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			client.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleWSBroadcast handles broadcasting messages to subscribed clients
func (s *Server) handleWSBroadcast() {
	for msg := range s.wsBroadcast {
		s.wsClientsMu.RLock()
		for _, client := range s.wsClients {
			client.mu.RLock()
			subscribed := client.subscriptions[msg.Topic]
			client.mu.RUnlock()

			if subscribed {
				client.writeMessage(msg)
			}
		}
		s.wsClientsMu.RUnlock()
	}
}

// handleWSSubscription handles subscription requests
func (s *Server) handleWSSubscription(client *WSClient, req *WSSubscribeRequest) {
	client.mu.Lock()
	defer client.mu.Unlock()

	validTopics := map[string]bool{
		"metrics":   true,
		"logs":      true,
		"alerts":    true,
		"proposals": true,
	}

	for _, topic := range req.Topics {
		if !validTopics[topic] {
			s.logger.Warn("Invalid subscription topic", "topic", topic)
			continue
		}

		if req.Action == "subscribe" {
			client.subscriptions[topic] = true
			s.logger.Info("Client subscribed", "topic", topic, "remote_addr", client.conn.RemoteAddr().String())

			// Send confirmation
			msg := WSMessage{
				Type:  "subscribed",
				Topic: topic,
				Data:  map[string]string{"status": "subscribed"},
				Time:  time.Now().Unix(),
			}
			client.writeMessage(msg)

		} else if req.Action == "unsubscribe" {
			delete(client.subscriptions, topic)
			s.logger.Info("Client unsubscribed", "topic", topic, "remote_addr", client.conn.RemoteAddr().String())

			// Send confirmation
			msg := WSMessage{
				Type:  "unsubscribed",
				Topic: topic,
				Data:  map[string]string{"status": "unsubscribed"},
				Time:  time.Now().Unix(),
			}
			client.writeMessage(msg)
		}
	}
}

// removeWSClient removes a client from the server
func (s *Server) removeWSClient(client *WSClient) {
	s.wsClientsMu.Lock()
	defer s.wsClientsMu.Unlock()

	if _, ok := s.wsClients[client.conn]; ok {
		delete(s.wsClients, client.conn)
		close(client.send)
		s.logger.Info("WebSocket client disconnected", "remote_addr", client.conn.RemoteAddr().String())
	}
}

// writeMessage writes a message to the client
func (client *WSClient) writeMessage(msg WSMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	select {
	case client.send <- data:
	default:
		// Channel full, skip message
	}
}

// BroadcastMetrics broadcasts metrics update to all subscribed clients
func (s *Server) BroadcastMetrics(snapshot *metrics.MetricsSnapshot) {
	if snapshot == nil {
		return
	}

	msg := WSMessage{
		Type:  "update",
		Topic: "metrics",
		Data:  snapshot,
		Time:  time.Now().Unix(),
	}

	select {
	case s.wsBroadcast <- msg:
	default:
		// Broadcast channel full, skip
	}
}

// BroadcastAlert broadcasts alert to all subscribed clients
func (s *Server) BroadcastAlert(alert *metrics.Alert) {
	if alert == nil {
		return
	}

	msg := WSMessage{
		Type:  "alert",
		Topic: "alerts",
		Data:  alert,
		Time:  time.Now().Unix(),
	}

	select {
	case s.wsBroadcast <- msg:
	default:
		// Broadcast channel full, skip
	}
}

// BroadcastLog broadcasts log entry to all subscribed clients
func (s *Server) BroadcastLog(logEntry map[string]interface{}) {
	msg := WSMessage{
		Type:  "log",
		Topic: "logs",
		Data:  logEntry,
		Time:  time.Now().Unix(),
	}

	select {
	case s.wsBroadcast <- msg:
	default:
		// Broadcast channel full, skip
	}
}

// BroadcastProposal broadcasts proposal update to all subscribed clients
func (s *Server) BroadcastProposal(proposal interface{}) {
	msg := WSMessage{
		Type:  "proposal",
		Topic: "proposals",
		Data:  proposal,
		Time:  time.Now().Unix(),
	}

	select {
	case s.wsBroadcast <- msg:
	default:
		// Broadcast channel full, skip
	}
}

// ginLogger creates a Gin logging middleware
func ginLogger(logger *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Log request details
		latency := time.Since(start)
		if raw != "" {
			path = path + "?" + raw
		}

		logger.Info("API request",
			"method", c.Request.Method,
			"path", path,
			"status", c.Writer.Status(),
			"latency", latency.String(),
			"ip", c.ClientIP(),
		)
	}
}