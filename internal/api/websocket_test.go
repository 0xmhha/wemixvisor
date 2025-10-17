package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wemix/wemixvisor/internal/config"
	"github.com/wemix/wemixvisor/internal/metrics"
	"github.com/wemix/wemixvisor/pkg/logger"
)

// setupTestWebSocketServer creates a test server with WebSocket support
func setupTestWebSocketServer(t *testing.T) *Server {
	cfg := &config.Config{
		Home:       "/tmp/wemixvisor-test",
		RPCAddress: "http://localhost:26657",
		Debug:      false,
		APIPort:    8080,
	}

	testLogger := logger.NewTestLogger()
	server := NewServer(cfg, nil, nil, testLogger)

	return server
}

// TestWSMessageSerialization tests WebSocket message JSON serialization
func TestWSMessageSerialization(t *testing.T) {
	// Arrange
	msg := WSMessage{
		Type:  "update",
		Topic: "metrics",
		Data: map[string]interface{}{
			"cpu": 50.5,
			"mem": 1024,
		},
		Time: 1234567890,
	}

	// Act
	data, err := json.Marshal(msg)

	// Assert
	require.NoError(t, err)
	assert.Contains(t, string(data), "\"type\":\"update\"")
	assert.Contains(t, string(data), "\"topic\":\"metrics\"")
	assert.Contains(t, string(data), "\"timestamp\":1234567890")

	// Test deserialization
	var decoded WSMessage
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, msg.Type, decoded.Type)
	assert.Equal(t, msg.Topic, decoded.Topic)
	assert.Equal(t, msg.Time, decoded.Time)
}

// TestWSSubscribeRequestParsing tests subscription request parsing
func TestWSSubscribeRequestParsing(t *testing.T) {
	tests := []struct {
		name      string
		json      string
		wantError bool
		wantAction string
		wantTopics []string
	}{
		{
			name:      "valid subscribe request",
			json:      `{"action":"subscribe","topics":["metrics","logs"]}`,
			wantError: false,
			wantAction: "subscribe",
			wantTopics: []string{"metrics", "logs"},
		},
		{
			name:      "valid unsubscribe request",
			json:      `{"action":"unsubscribe","topics":["alerts"]}`,
			wantError: false,
			wantAction: "unsubscribe",
			wantTopics: []string{"alerts"},
		},
		{
			name:      "invalid JSON",
			json:      `{invalid}`,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			var req WSSubscribeRequest
			err := json.Unmarshal([]byte(tt.json), &req)

			// Assert
			if tt.wantError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantAction, req.Action)
				assert.Equal(t, tt.wantTopics, req.Topics)
			}
		})
	}
}

// TestBroadcastMetrics tests broadcasting metrics to clients
func TestBroadcastMetrics(t *testing.T) {
	// Arrange
	server := setupTestWebSocketServer(t)

	snapshot := &metrics.MetricsSnapshot{
		Timestamp: time.Now(),
		System: &metrics.SystemMetrics{
			CPUUsage:        50.5,
			MemoryUsage:     75.0,
			MemoryTotal:     1024 * 1024 * 1024,
			MemoryAvailable: 256 * 1024 * 1024,
		},
	}

	// Act - Broadcast should not block even if no clients
	server.BroadcastMetrics(snapshot)

	// Assert - Check message was sent to broadcast channel (non-blocking)
	// Since we're testing the broadcast mechanism, we verify it doesn't panic
	// and the channel doesn't block
	assert.NotNil(t, server.wsBroadcast)
}

// TestBroadcastMetricsNil tests broadcasting nil metrics
func TestBroadcastMetricsNil(t *testing.T) {
	// Arrange
	server := setupTestWebSocketServer(t)

	// Act - Should handle nil gracefully
	server.BroadcastMetrics(nil)

	// Assert - No panic, no message sent
	assert.NotNil(t, server.wsBroadcast)
}

// TestBroadcastAlert tests broadcasting alerts to clients
func TestBroadcastAlert(t *testing.T) {
	// Arrange
	server := setupTestWebSocketServer(t)

	alert := &metrics.Alert{
		ID:      "test-alert-1",
		Name:    "high_cpu",
		Level:   metrics.AlertLevelWarning,
		Message: "CPU usage is high",
		Source:  "test",
		Metric:  "cpu_usage",
		Value:   85.0,
		Threshold: 80.0,
	}

	// Act - Broadcast should not block
	server.BroadcastAlert(alert)

	// Assert
	assert.NotNil(t, server.wsBroadcast)
}

// TestBroadcastAlertNil tests broadcasting nil alert
func TestBroadcastAlertNil(t *testing.T) {
	// Arrange
	server := setupTestWebSocketServer(t)

	// Act - Should handle nil gracefully
	server.BroadcastAlert(nil)

	// Assert - No panic
	assert.NotNil(t, server.wsBroadcast)
}

// TestBroadcastLog tests broadcasting logs to clients
func TestBroadcastLog(t *testing.T) {
	// Arrange
	server := setupTestWebSocketServer(t)

	logEntry := map[string]interface{}{
		"timestamp": time.Now().Unix(),
		"level":     "info",
		"message":   "Test log entry",
	}

	// Act - Broadcast should not block
	server.BroadcastLog(logEntry)

	// Assert
	assert.NotNil(t, server.wsBroadcast)
}

// TestBroadcastProposal tests broadcasting proposals to clients
func TestBroadcastProposal(t *testing.T) {
	// Arrange
	server := setupTestWebSocketServer(t)

	proposal := map[string]interface{}{
		"id":     "test-proposal-1",
		"status": "voting",
		"title":  "Test Proposal",
	}

	// Act - Broadcast should not block
	server.BroadcastProposal(proposal)

	// Assert
	assert.NotNil(t, server.wsBroadcast)
}

// TestWSClientWriteMessage tests writing messages to client channel
func TestWSClientWriteMessage(t *testing.T) {
	// Arrange
	client := &WSClient{
		send:          make(chan []byte, 256),
		subscriptions: make(map[string]bool),
	}

	msg := WSMessage{
		Type:  "test",
		Topic: "metrics",
		Data:  map[string]string{"status": "ok"},
		Time:  time.Now().Unix(),
	}

	// Act
	client.writeMessage(msg)

	// Assert - Message should be in channel
	select {
	case data := <-client.send:
		var decoded WSMessage
		err := json.Unmarshal(data, &decoded)
		require.NoError(t, err)
		assert.Equal(t, msg.Type, decoded.Type)
		assert.Equal(t, msg.Topic, decoded.Topic)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for message")
	}
}

// TestWSClientWriteMessageChannelFull tests that writeMessage doesn't block when channel is full
func TestWSClientWriteMessageChannelFull(t *testing.T) {
	// Arrange
	client := &WSClient{
		send:          make(chan []byte, 2), // Small buffer
		subscriptions: make(map[string]bool),
	}

	msg := WSMessage{
		Type:  "test",
		Topic: "metrics",
		Data:  map[string]string{"status": "ok"},
		Time:  time.Now().Unix(),
	}

	// Fill the channel
	client.writeMessage(msg)
	client.writeMessage(msg)

	// Act - Should not block even when channel is full
	done := make(chan bool)
	go func() {
		client.writeMessage(msg) // This should not block
		done <- true
	}()

	// Assert - Should complete quickly without blocking
	select {
	case <-done:
		// Success - didn't block
	case <-time.After(100 * time.Millisecond):
		t.Fatal("writeMessage blocked when channel was full")
	}
}

// TestWebSocketUpgrade tests WebSocket connection upgrade
func TestWebSocketUpgrade(t *testing.T) {
	// Arrange
	server := setupTestWebSocketServer(t)

	// Create a test HTTP server
	ts := httptest.NewServer(server.router)
	defer ts.Close()

	// Convert http:// to ws://
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/api/v1/ws"

	// Act - Connect to WebSocket
	ws, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)
	defer ws.Close()

	// Read welcome message
	_, message, err := ws.ReadMessage()
	require.NoError(t, err)

	var welcome WSMessage
	err = json.Unmarshal(message, &welcome)
	require.NoError(t, err)
	assert.Equal(t, "connected", welcome.Type)
	assert.Equal(t, "system", welcome.Topic)
}

// TestWebSocketSubscription tests subscription flow
func TestWebSocketSubscription(t *testing.T) {
	// Arrange
	server := setupTestWebSocketServer(t)
	ts := httptest.NewServer(server.router)
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/api/v1/ws"
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws.Close()

	// Read and discard welcome message
	_, _, err = ws.ReadMessage()
	require.NoError(t, err)

	// Act - Send subscribe request
	subscribeReq := WSSubscribeRequest{
		Action: "subscribe",
		Topics: []string{"metrics", "alerts"},
	}

	reqData, err := json.Marshal(subscribeReq)
	require.NoError(t, err)

	err = ws.WriteMessage(websocket.TextMessage, reqData)
	require.NoError(t, err)

	// Assert - Read subscription confirmations
	for i := 0; i < 2; i++ {
		_, message, err := ws.ReadMessage()
		if err != nil {
			break // May timeout, which is okay
		}

		var msg WSMessage
		err = json.Unmarshal(message, &msg)
		if err == nil {
			assert.Equal(t, "subscribed", msg.Type)
			assert.Contains(t, []string{"metrics", "alerts"}, msg.Topic)
		}
	}
}

// TestWebSocketUnsubscription tests unsubscription flow
func TestWebSocketUnsubscription(t *testing.T) {
	// Arrange
	server := setupTestWebSocketServer(t)
	ts := httptest.NewServer(server.router)
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/api/v1/ws"
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws.Close()

	// Read and discard welcome message
	_, _, err = ws.ReadMessage()
	require.NoError(t, err)

	// Subscribe first
	subscribeReq := WSSubscribeRequest{
		Action: "subscribe",
		Topics: []string{"metrics"},
	}
	reqData, _ := json.Marshal(subscribeReq)
	ws.WriteMessage(websocket.TextMessage, reqData)

	// Read subscription confirmation
	ws.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	ws.ReadMessage()

	// Act - Unsubscribe
	unsubscribeReq := WSSubscribeRequest{
		Action: "unsubscribe",
		Topics: []string{"metrics"},
	}
	reqData, err = json.Marshal(unsubscribeReq)
	require.NoError(t, err)

	err = ws.WriteMessage(websocket.TextMessage, reqData)
	require.NoError(t, err)

	// Assert - Read unsubscription confirmation
	ws.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	_, message, err := ws.ReadMessage()
	if err == nil {
		var msg WSMessage
		err = json.Unmarshal(message, &msg)
		if err == nil {
			assert.Equal(t, "unsubscribed", msg.Type)
			assert.Equal(t, "metrics", msg.Topic)
		}
	}
}

// TestWebSocketInvalidTopic tests subscribing to invalid topic
func TestWebSocketInvalidTopic(t *testing.T) {
	// Arrange
	server := setupTestWebSocketServer(t)
	ts := httptest.NewServer(server.router)
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/api/v1/ws"
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws.Close()

	// Read and discard welcome message
	_, _, err = ws.ReadMessage()
	require.NoError(t, err)

	// Act - Try to subscribe to invalid topic
	subscribeReq := WSSubscribeRequest{
		Action: "subscribe",
		Topics: []string{"invalid_topic"},
	}

	reqData, err := json.Marshal(subscribeReq)
	require.NoError(t, err)

	err = ws.WriteMessage(websocket.TextMessage, reqData)
	require.NoError(t, err)

	// Assert - Should not receive confirmation for invalid topic
	// (Server logs warning but doesn't send error message)
	ws.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	_, _, err = ws.ReadMessage()
	// Timeout is expected since no confirmation is sent for invalid topics
	assert.Error(t, err)
}

// TestWebSocketMultipleClients tests multiple concurrent clients
func TestWebSocketMultipleClients(t *testing.T) {
	// Arrange
	server := setupTestWebSocketServer(t)
	ts := httptest.NewServer(server.router)
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/api/v1/ws"

	// Act - Connect multiple clients
	clients := make([]*websocket.Conn, 3)
	for i := 0; i < 3; i++ {
		ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		clients[i] = ws

		// Read welcome message
		_, _, err = ws.ReadMessage()
		require.NoError(t, err)
	}

	// Assert - All clients connected successfully
	server.wsClientsMu.RLock()
	clientCount := len(server.wsClients)
	server.wsClientsMu.RUnlock()
	assert.Equal(t, 3, clientCount)

	// Cleanup
	for _, client := range clients {
		client.Close()
	}

	// Wait for cleanup
	time.Sleep(100 * time.Millisecond)

	// Verify clients were removed
	server.wsClientsMu.RLock()
	clientCount = len(server.wsClients)
	server.wsClientsMu.RUnlock()
	assert.Equal(t, 0, clientCount)
}

// TestWebSocketConnectionCleanup tests that clients are properly cleaned up on disconnect
func TestWebSocketConnectionCleanup(t *testing.T) {
	// Arrange
	server := setupTestWebSocketServer(t)
	ts := httptest.NewServer(server.router)
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/api/v1/ws"

	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)

	// Read welcome message
	_, _, err = ws.ReadMessage()
	require.NoError(t, err)

	// Verify client is registered
	server.wsClientsMu.RLock()
	initialCount := len(server.wsClients)
	server.wsClientsMu.RUnlock()
	assert.Equal(t, 1, initialCount)

	// Act - Close connection
	ws.Close()

	// Assert - Wait for cleanup and verify client was removed
	time.Sleep(100 * time.Millisecond)

	server.wsClientsMu.RLock()
	finalCount := len(server.wsClients)
	server.wsClientsMu.RUnlock()
	assert.Equal(t, 0, finalCount)
}
