package governance

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/wemix/wemixvisor/pkg/logger"
)

// MockNotificationHandler is a mock implementation of NotificationHandler for testing
type MockNotificationHandler struct {
	mock.Mock
	handlerType string
	enabled     bool
}

func (m *MockNotificationHandler) Handle(notification *Notification) error {
	args := m.Called(notification)
	return args.Error(0)
}

func (m *MockNotificationHandler) GetType() string {
	return m.handlerType
}

func (m *MockNotificationHandler) IsEnabled() bool {
	return m.enabled
}

func TestNewNotifier(t *testing.T) {
	testLogger := logger.NewTestLogger()

	notifier := NewNotifier(testLogger)

	assert.NotNil(t, notifier)
	assert.Equal(t, testLogger, notifier.logger)
	assert.True(t, notifier.enabled)
	assert.Equal(t, 1000, notifier.maxNotifications)
	assert.Equal(t, 30*24*time.Hour, notifier.maxAge)
	assert.Empty(t, notifier.notifications)
	assert.Empty(t, notifier.handlers)
	assert.NotEmpty(t, notifier.priorities)
}

func TestNotifier_NotifyNewProposal(t *testing.T) {
	testLogger := logger.NewTestLogger()
	notifier := NewNotifier(testLogger)

	proposal := &Proposal{
		ID:    "1",
		Title: "Test Proposal",
		Type:  ProposalTypeText,
	}

	notifier.NotifyNewProposal(proposal)

	assert.Len(t, notifier.notifications, 1)

	// Get the notification
	var notification *Notification
	for _, n := range notifier.notifications {
		notification = n
		break
	}

	assert.NotNil(t, notification)
	assert.Equal(t, EventNewProposal, notification.Event)
	assert.Contains(t, notification.Title, "Test Proposal")
	assert.Equal(t, proposal, notification.Data)
	assert.False(t, notification.Read)
}

func TestNotifier_NotifyProposalPassed(t *testing.T) {
	testLogger := logger.NewTestLogger()
	notifier := NewNotifier(testLogger)

	proposal := &Proposal{
		ID:    "1",
		Title: "Test Proposal",
		Type:  ProposalTypeText,
	}

	notifier.NotifyProposalPassed(proposal)

	assert.Len(t, notifier.notifications, 1)

	var notification *Notification
	for _, n := range notifier.notifications {
		notification = n
		break
	}

	assert.Equal(t, EventProposalPassed, notification.Event)
	assert.Contains(t, notification.Title, "Proposal Passed")
}

func TestNotifier_NotifyUpgradeScheduled(t *testing.T) {
	testLogger := logger.NewTestLogger()
	notifier := NewNotifier(testLogger)

	proposal := &Proposal{
		ID:            "1",
		UpgradeHeight: 1000,
		UpgradeInfo: &UpgradeInfo{
			Name:   "test-upgrade",
			Height: 1000,
		},
	}

	notifier.NotifyUpgradeScheduled(proposal)

	assert.Len(t, notifier.notifications, 1)

	var notification *Notification
	for _, n := range notifier.notifications {
		notification = n
		break
	}

	assert.Equal(t, EventUpgradeScheduled, notification.Event)
	assert.Contains(t, notification.Title, "test-upgrade")
	assert.Contains(t, notification.Message, "1000")
}

func TestNotifier_NotifyUpgradeTriggered(t *testing.T) {
	testLogger := logger.NewTestLogger()
	notifier := NewNotifier(testLogger)

	upgrade := &UpgradeInfo{
		Name:   "test-upgrade",
		Height: 1000,
	}

	notifier.NotifyUpgradeTriggered(upgrade)

	assert.Len(t, notifier.notifications, 1)

	var notification *Notification
	for _, n := range notifier.notifications {
		notification = n
		break
	}

	assert.Equal(t, EventUpgradeTriggered, notification.Event)
	assert.Contains(t, notification.Title, "Upgrade Started")
	assert.Equal(t, upgrade, notification.Data)
}

func TestNotifier_NotifyUpgradeFailed(t *testing.T) {
	testLogger := logger.NewTestLogger()
	notifier := NewNotifier(testLogger)

	upgrade := &UpgradeInfo{
		Name: "test-upgrade",
	}
	testError := assert.AnError

	notifier.NotifyUpgradeFailed(upgrade, testError)

	assert.Len(t, notifier.notifications, 1)

	var notification *Notification
	for _, n := range notifier.notifications {
		notification = n
		break
	}

	assert.Equal(t, EventUpgradeFailed, notification.Event)
	assert.Contains(t, notification.Title, "Upgrade Failed")

	// Check that error is included in data
	data, ok := notification.Data.(map[string]interface{})
	assert.True(t, ok)
	assert.Contains(t, data, "upgrade")
	assert.Contains(t, data, "error")
}

func TestNotifier_AddHandler(t *testing.T) {
	testLogger := logger.NewTestLogger()
	notifier := NewNotifier(testLogger)

	handler := &MockNotificationHandler{
		handlerType: "test-handler",
		enabled:     true,
	}

	notifier.AddHandler(handler)

	assert.Len(t, notifier.handlers, 1)
	assert.Equal(t, handler, notifier.handlers[0])
}

func TestNotifier_RemoveHandler(t *testing.T) {
	testLogger := logger.NewTestLogger()
	notifier := NewNotifier(testLogger)

	handler := &MockNotificationHandler{
		handlerType: "test-handler",
		enabled:     true,
	}

	notifier.AddHandler(handler)
	assert.Len(t, notifier.handlers, 1)

	notifier.RemoveHandler("test-handler")
	assert.Len(t, notifier.handlers, 0)

	// Test removing non-existing handler
	notifier.RemoveHandler("non-existing")
	assert.Len(t, notifier.handlers, 0)
}

func TestNotifier_GetNotifications(t *testing.T) {
	testLogger := logger.NewTestLogger()
	notifier := NewNotifier(testLogger)

	// Add test notifications
	now := time.Now()
	notifier.notifications["1"] = &Notification{
		ID:        "1",
		Timestamp: now,
	}
	notifier.notifications["2"] = &Notification{
		ID:        "2",
		Timestamp: now.Add(-time.Hour),
	}

	notifications := notifier.GetNotifications()

	assert.Len(t, notifications, 2)
	// Should be sorted by timestamp (newest first)
	assert.Equal(t, "1", notifications[0].ID)
	assert.Equal(t, "2", notifications[1].ID)
}

func TestNotifier_GetUnreadNotifications(t *testing.T) {
	testLogger := logger.NewTestLogger()
	notifier := NewNotifier(testLogger)

	// Add test notifications
	notifier.notifications["1"] = &Notification{
		ID:   "1",
		Read: false,
	}
	notifier.notifications["2"] = &Notification{
		ID:   "2",
		Read: true,
	}
	notifier.notifications["3"] = &Notification{
		ID:   "3",
		Read: false,
	}

	unread := notifier.GetUnreadNotifications()

	assert.Len(t, unread, 2)
	// Check that only unread notifications are returned
	for _, notification := range unread {
		assert.False(t, notification.Read)
	}
}

func TestNotifier_MarkAsRead(t *testing.T) {
	testLogger := logger.NewTestLogger()
	notifier := NewNotifier(testLogger)

	notifier.notifications["1"] = &Notification{
		ID:   "1",
		Read: false,
	}

	err := notifier.MarkAsRead("1")

	assert.NoError(t, err)
	assert.True(t, notifier.notifications["1"].Read)

	// Test marking non-existing notification
	err = notifier.MarkAsRead("999")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "notification not found")
}

func TestNotifier_MarkAllAsRead(t *testing.T) {
	testLogger := logger.NewTestLogger()
	notifier := NewNotifier(testLogger)

	notifier.notifications["1"] = &Notification{ID: "1", Read: false}
	notifier.notifications["2"] = &Notification{ID: "2", Read: false}
	notifier.notifications["3"] = &Notification{ID: "3", Read: true}

	err := notifier.MarkAllAsRead()

	assert.NoError(t, err)
	for _, notification := range notifier.notifications {
		assert.True(t, notification.Read)
	}
}

func TestNotifier_CleanupOld(t *testing.T) {
	testLogger := logger.NewTestLogger()
	notifier := NewNotifier(testLogger)

	// Add old notifications
	oldTime := time.Now().Add(-40 * 24 * time.Hour) // 40 days ago
	recentTime := time.Now().Add(-5 * 24 * time.Hour) // 5 days ago

	notifier.notifications["old"] = &Notification{
		ID:        "old",
		Timestamp: oldTime,
	}
	notifier.notifications["recent"] = &Notification{
		ID:        "recent",
		Timestamp: recentTime,
	}

	err := notifier.CleanupOld()

	assert.NoError(t, err)
	assert.NotContains(t, notifier.notifications, "old")
	assert.Contains(t, notifier.notifications, "recent")
}

func TestNotifier_CleanupOld_ByCount(t *testing.T) {
	testLogger := logger.NewTestLogger()
	notifier := NewNotifier(testLogger)
	notifier.maxNotifications = 2

	// Add 3 notifications
	now := time.Now()
	notifier.notifications["1"] = &Notification{ID: "1", Timestamp: now.Add(-3 * time.Hour)}
	notifier.notifications["2"] = &Notification{ID: "2", Timestamp: now.Add(-2 * time.Hour)}
	notifier.notifications["3"] = &Notification{ID: "3", Timestamp: now.Add(-1 * time.Hour)}

	err := notifier.CleanupOld()

	assert.NoError(t, err)
	assert.Len(t, notifier.notifications, 2)
	// Should keep the 2 newest
	assert.Contains(t, notifier.notifications, "2")
	assert.Contains(t, notifier.notifications, "3")
	assert.NotContains(t, notifier.notifications, "1")
}

func TestNotifier_SetEnabled(t *testing.T) {
	testLogger := logger.NewTestLogger()
	notifier := NewNotifier(testLogger)

	notifier.SetEnabled(false)
	assert.False(t, notifier.enabled)

	notifier.SetEnabled(true)
	assert.True(t, notifier.enabled)
}

func TestNotifier_GetStats(t *testing.T) {
	testLogger := logger.NewTestLogger()
	notifier := NewNotifier(testLogger)

	// Add test notifications
	notifier.notifications["1"] = &Notification{
		Event:    EventNewProposal,
		Priority: PriorityHigh,
		Read:     false,
	}
	notifier.notifications["2"] = &Notification{
		Event:    EventUpgradeScheduled,
		Priority: PriorityHigh,
		Read:     true,
	}

	// Add test handler
	handler := &MockNotificationHandler{enabled: true}
	notifier.handlers = []NotificationHandler{handler}

	stats := notifier.GetStats()

	assert.Equal(t, 2, stats["total_notifications"])
	assert.Equal(t, 1, stats["unread_notifications"])
	assert.Equal(t, 1, stats["enabled_handlers"])

	byEvent := stats["by_event"].(map[string]int)
	assert.Equal(t, 1, byEvent["new_proposal"])
	assert.Equal(t, 1, byEvent["upgrade_scheduled"])

	byPriority := stats["by_priority"].(map[string]int)
	assert.Equal(t, 2, byPriority["high"])
}

func TestNotifier_SendNotificationToHandlers(t *testing.T) {
	testLogger := logger.NewTestLogger()
	notifier := NewNotifier(testLogger)

	// Add enabled handler
	enabledHandler := &MockNotificationHandler{
		handlerType: "enabled",
		enabled:     true,
	}
	enabledHandler.On("Handle", mock.AnythingOfType("*governance.Notification")).Return(nil)

	// Add disabled handler
	disabledHandler := &MockNotificationHandler{
		handlerType: "disabled",
		enabled:     false,
	}
	// Disabled handler should not be called

	notifier.AddHandler(enabledHandler)
	notifier.AddHandler(disabledHandler)

	proposal := &Proposal{
		ID:    "1",
		Title: "Test Proposal",
		Type:  ProposalTypeText,
	}

	notifier.NotifyNewProposal(proposal)

	// Give time for goroutines to complete
	time.Sleep(10 * time.Millisecond)

	enabledHandler.AssertExpectations(t)
	disabledHandler.AssertNotCalled(t, "Handle")
}

func TestNotifier_DisabledNotifier(t *testing.T) {
	testLogger := logger.NewTestLogger()
	notifier := NewNotifier(testLogger)
	notifier.SetEnabled(false)

	handler := &MockNotificationHandler{
		handlerType: "test",
		enabled:     true,
	}
	notifier.AddHandler(handler)

	proposal := &Proposal{
		ID:    "1",
		Title: "Test Proposal",
		Type:  ProposalTypeText,
	}

	notifier.NotifyNewProposal(proposal)

	// No notifications should be created when disabled
	assert.Empty(t, notifier.notifications)
	handler.AssertNotCalled(t, "Handle")
}

func TestGetDefaultPriorities(t *testing.T) {
	priorities := getDefaultPriorities()

	assert.NotEmpty(t, priorities)
	assert.Equal(t, PriorityCritical, priorities[EventUpgradeTriggered])
	assert.Equal(t, PriorityCritical, priorities[EventUpgradeFailed])
	assert.Equal(t, PriorityCritical, priorities[EventEmergencyProposal])
	assert.Equal(t, PriorityHigh, priorities[EventProposalPassed])
	assert.Equal(t, PriorityMedium, priorities[EventNewProposal])
}

func TestGenerateNotificationID(t *testing.T) {
	id1 := generateNotificationID()
	time.Sleep(1 * time.Nanosecond) // Ensure different timestamps
	id2 := generateNotificationID()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2)
	assert.Contains(t, id1, "notif_")
}