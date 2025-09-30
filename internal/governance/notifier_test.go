package governance

import (
	"fmt"
	"sync"
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

// Advanced Scenario Tests

func TestNotifier_Start(t *testing.T) {
	testLogger := logger.NewTestLogger()
	notifier := NewNotifier(testLogger)

	// Add some handlers
	handler1 := &MockNotificationHandler{handlerType: "enabled", enabled: true}
	handler2 := &MockNotificationHandler{handlerType: "disabled", enabled: false}
	notifier.AddHandler(handler1)
	notifier.AddHandler(handler2)

	err := notifier.Start()
	assert.NoError(t, err)
}

func TestNotifier_Start_Disabled(t *testing.T) {
	testLogger := logger.NewTestLogger()
	notifier := NewNotifier(testLogger)
	notifier.SetEnabled(false)

	err := notifier.Start()
	assert.NoError(t, err)
}

func TestNotifier_Stop(t *testing.T) {
	testLogger := logger.NewTestLogger()
	notifier := NewNotifier(testLogger)

	err := notifier.Stop()
	assert.NoError(t, err)
	assert.False(t, notifier.enabled)
}

func TestNotifier_NotifyProposalRejected(t *testing.T) {
	testLogger := logger.NewTestLogger()
	notifier := NewNotifier(testLogger)

	proposal := &Proposal{
		ID:    "1",
		Title: "Test Proposal",
		Type:  ProposalTypeText,
	}

	notifier.NotifyProposalRejected(proposal)

	assert.Len(t, notifier.notifications, 1)

	var notification *Notification
	for _, n := range notifier.notifications {
		notification = n
		break
	}

	assert.Equal(t, EventProposalRejected, notification.Event)
	assert.Contains(t, notification.Title, "Proposal Rejected")
	assert.Equal(t, PriorityMedium, notification.Priority)
}

func TestNotifier_NotifyUpgradeCompleted(t *testing.T) {
	testLogger := logger.NewTestLogger()
	notifier := NewNotifier(testLogger)

	upgrade := &UpgradeInfo{
		Name:   "test-upgrade",
		Height: 1000,
	}

	notifier.NotifyUpgradeCompleted(upgrade)

	assert.Len(t, notifier.notifications, 1)

	var notification *Notification
	for _, n := range notifier.notifications {
		notification = n
		break
	}

	assert.Equal(t, EventUpgradeCompleted, notification.Event)
	assert.Contains(t, notification.Title, "Upgrade Completed")
	assert.Equal(t, PriorityHigh, notification.Priority)
}

func TestNotifier_NotifyVotingStarted(t *testing.T) {
	testLogger := logger.NewTestLogger()
	notifier := NewNotifier(testLogger)

	votingEndTime := time.Now().Add(24 * time.Hour)
	proposal := &Proposal{
		ID:             "1",
		Title:          "Test Proposal",
		Type:           ProposalTypeText,
		VotingEndTime:  votingEndTime,
	}

	notifier.NotifyVotingStarted(proposal)

	assert.Len(t, notifier.notifications, 1)

	var notification *Notification
	for _, n := range notifier.notifications {
		notification = n
		break
	}

	assert.Equal(t, EventVotingStarted, notification.Event)
	assert.Contains(t, notification.Title, "Voting Started")
	assert.Contains(t, notification.Message, votingEndTime.Format(time.RFC3339))
}

func TestNotifier_NotifyVotingEnded(t *testing.T) {
	testLogger := logger.NewTestLogger()
	notifier := NewNotifier(testLogger)

	proposal := &Proposal{
		ID:     "1",
		Title:  "Test Proposal",
		Type:   ProposalTypeText,
		Status: ProposalStatusPassed,
	}

	notifier.NotifyVotingEnded(proposal)

	assert.Len(t, notifier.notifications, 1)

	var notification *Notification
	for _, n := range notifier.notifications {
		notification = n
		break
	}

	assert.Equal(t, EventVotingEnded, notification.Event)
	assert.Contains(t, notification.Title, "Voting Ended")
	assert.Contains(t, notification.Message, string(ProposalStatusPassed))
}

func TestNotifier_NotifyQuorumReached(t *testing.T) {
	testLogger := logger.NewTestLogger()
	notifier := NewNotifier(testLogger)

	proposal := &Proposal{
		ID:    "1",
		Title: "Test Proposal",
		Type:  ProposalTypeText,
		VotingStats: &VotingStats{
			Turnout: 0.75, // 75% turnout
		},
	}

	notifier.NotifyQuorumReached(proposal)

	assert.Len(t, notifier.notifications, 1)

	var notification *Notification
	for _, n := range notifier.notifications {
		notification = n
		break
	}

	assert.Equal(t, EventQuorumReached, notification.Event)
	assert.Contains(t, notification.Title, "Quorum Reached")
	assert.Contains(t, notification.Message, "75.00%")
	assert.Equal(t, PriorityHigh, notification.Priority)
}

func TestNotifier_NotifyEmergencyProposal(t *testing.T) {
	testLogger := logger.NewTestLogger()
	notifier := NewNotifier(testLogger)

	proposal := &Proposal{
		ID:    "1",
		Title: "Emergency Fix",
		Type:  ProposalTypeUpgrade,
	}

	notifier.NotifyEmergencyProposal(proposal)

	assert.Len(t, notifier.notifications, 1)

	var notification *Notification
	for _, n := range notifier.notifications {
		notification = n
		break
	}

	assert.Equal(t, EventEmergencyProposal, notification.Event)
	assert.Contains(t, notification.Title, "EMERGENCY PROPOSAL")
	assert.Equal(t, PriorityCritical, notification.Priority)
}

func TestNotifier_HandlerError(t *testing.T) {
	testLogger := logger.NewTestLogger()
	notifier := NewNotifier(testLogger)

	// Add handler that will fail
	failingHandler := &MockNotificationHandler{
		handlerType: "failing",
		enabled:     true,
	}
	failingHandler.On("Handle", mock.AnythingOfType("*governance.Notification")).Return(assert.AnError)

	notifier.AddHandler(failingHandler)

	proposal := &Proposal{
		ID:    "1",
		Title: "Test Proposal",
		Type:  ProposalTypeText,
	}

	notifier.NotifyNewProposal(proposal)

	// Give time for goroutines to complete
	time.Sleep(50 * time.Millisecond)

	// Should have created notification despite handler failure
	assert.Len(t, notifier.notifications, 1)
	failingHandler.AssertExpectations(t)
}

func TestNotifier_MultipleHandlersWithPriorities(t *testing.T) {
	testLogger := logger.NewTestLogger()
	notifier := NewNotifier(testLogger)

	// Add multiple handlers
	handler1 := &MockNotificationHandler{handlerType: "handler1", enabled: true}
	handler1.On("Handle", mock.AnythingOfType("*governance.Notification")).Return(nil)

	handler2 := &MockNotificationHandler{handlerType: "handler2", enabled: true}
	handler2.On("Handle", mock.AnythingOfType("*governance.Notification")).Return(nil)

	notifier.AddHandler(handler1)
	notifier.AddHandler(handler2)

	// Test different priority notifications
	proposal := &Proposal{ID: "1", Title: "Test", Type: ProposalTypeText}
	upgrade := &UpgradeInfo{Name: "test-upgrade", Height: 1000}

	notifier.NotifyNewProposal(proposal)          // Medium priority
	notifier.NotifyUpgradeTriggered(upgrade)      // Critical priority
	notifier.NotifyEmergencyProposal(proposal)    // Critical priority

	// Give time for goroutines to complete
	time.Sleep(50 * time.Millisecond)

	assert.Len(t, notifier.notifications, 3)

	// Verify both handlers were called for each notification (3 times each)
	handler1.AssertNumberOfCalls(t, "Handle", 3)
	handler2.AssertNumberOfCalls(t, "Handle", 3)
}

func TestNotifier_ConcurrentNotifications(t *testing.T) {
	testLogger := logger.NewTestLogger()
	notifier := NewNotifier(testLogger)

	handler := &MockNotificationHandler{handlerType: "test", enabled: true}
	handler.On("Handle", mock.AnythingOfType("*governance.Notification")).Return(nil)
	notifier.AddHandler(handler)

	// Send multiple notifications concurrently with different proposals
	numNotifications := 10
	var wg sync.WaitGroup
	wg.Add(numNotifications)

	for i := 0; i < numNotifications; i++ {
		go func(index int) {
			defer wg.Done()
			proposal := &Proposal{
				ID:    fmt.Sprintf("%d", index),
				Title: fmt.Sprintf("Test %d", index),
				Type:  ProposalTypeText,
			}
			notifier.NotifyNewProposal(proposal)
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	time.Sleep(50 * time.Millisecond) // Give handlers time to complete

	// Should have created all notifications
	assert.Len(t, notifier.notifications, numNotifications)
	handler.AssertNumberOfCalls(t, "Handle", numNotifications)
}

func TestNotifier_MaxNotificationsLimit(t *testing.T) {
	testLogger := logger.NewTestLogger()
	notifier := NewNotifier(testLogger)
	notifier.maxNotifications = 3 // Set low limit for testing

	proposal := &Proposal{ID: "1", Title: "Test", Type: ProposalTypeText}

	// Add notifications beyond the limit
	for i := 0; i < 5; i++ {
		notifier.NotifyNewProposal(proposal)
		time.Sleep(1 * time.Millisecond) // Ensure different timestamps
	}

	// Trigger cleanup
	notifier.CleanupOld()

	// Should only keep the maximum allowed
	assert.Len(t, notifier.notifications, notifier.maxNotifications)
}

func TestNotifier_GetPriorityForUnknownEvent(t *testing.T) {
	testLogger := logger.NewTestLogger()
	notifier := NewNotifier(testLogger)

	// Test with unknown event
	unknownEvent := NotificationEvent("unknown_event")
	priority := notifier.getPriority(unknownEvent)

	assert.Equal(t, PriorityMedium, priority) // Should default to medium
}

func TestNotifier_CountEnabledHandlers(t *testing.T) {
	testLogger := logger.NewTestLogger()
	notifier := NewNotifier(testLogger)

	// Add mix of enabled and disabled handlers
	handler1 := &MockNotificationHandler{handlerType: "enabled1", enabled: true}
	handler2 := &MockNotificationHandler{handlerType: "disabled", enabled: false}
	handler3 := &MockNotificationHandler{handlerType: "enabled2", enabled: true}

	notifier.AddHandler(handler1)
	notifier.AddHandler(handler2)
	notifier.AddHandler(handler3)

	count := notifier.countEnabledHandlers()
	assert.Equal(t, 2, count)
}