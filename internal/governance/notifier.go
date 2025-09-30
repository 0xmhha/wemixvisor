package governance

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/wemix/wemixvisor/pkg/logger"
	"go.uber.org/zap"
)

// NotificationHandler represents a handler for notifications
type NotificationHandler interface {
	Handle(notification *Notification) error
	GetType() string
	IsEnabled() bool
}

// Notifier manages notifications for governance events
type Notifier struct {
	logger *logger.Logger

	// State management
	mu           sync.RWMutex
	notifications map[string]*Notification
	handlers      []NotificationHandler
	enabled       bool

	// Configuration
	maxNotifications int
	maxAge          time.Duration
	priorities      map[NotificationEvent]NotificationPriority
}

// NewNotifier creates a new notification manager
func NewNotifier(logger *logger.Logger) *Notifier {
	return &Notifier{
		logger:          logger,
		notifications:   make(map[string]*Notification),
		handlers:        make([]NotificationHandler, 0),
		enabled:         true,
		maxNotifications: 1000,
		maxAge:          30 * 24 * time.Hour, // 30 days
		priorities:      getDefaultPriorities(),
	}
}

// Start begins the notification system
func (n *Notifier) Start() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if !n.enabled {
		n.logger.Info("notification system is disabled")
		return nil
	}

	n.logger.Info("starting notification system")

	// Initialize handlers
	for _, handler := range n.handlers {
		if handler.IsEnabled() {
			n.logger.Info("notification handler enabled", zap.String("type", handler.GetType()))
		}
	}

	n.logger.Info("notification system started successfully")
	return nil
}

// Stop stops the notification system
func (n *Notifier) Stop() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.logger.Info("stopping notification system")
	n.enabled = false

	return nil
}

// NotifyNewProposal sends a notification for a new proposal
func (n *Notifier) NotifyNewProposal(proposal *Proposal) {
	title := fmt.Sprintf("New %s Proposal: %s", proposal.Type, proposal.Title)
	message := fmt.Sprintf("Proposal %s has been submitted and is now open for discussion.", proposal.ID)

	notification := &Notification{
		ID:        generateNotificationID(),
		Event:     EventNewProposal,
		Title:     title,
		Message:   message,
		Data:      proposal,
		Timestamp: time.Now(),
		Read:      false,
		Priority:  n.getPriority(EventNewProposal),
	}

	n.sendNotification(notification)
}

// NotifyProposalPassed sends a notification when a proposal passes
func (n *Notifier) NotifyProposalPassed(proposal *Proposal) {
	title := fmt.Sprintf("Proposal Passed: %s", proposal.Title)
	message := fmt.Sprintf("Proposal %s has passed and will take effect.", proposal.ID)

	notification := &Notification{
		ID:        generateNotificationID(),
		Event:     EventProposalPassed,
		Title:     title,
		Message:   message,
		Data:      proposal,
		Timestamp: time.Now(),
		Read:      false,
		Priority:  n.getPriority(EventProposalPassed),
	}

	n.sendNotification(notification)
}

// NotifyProposalRejected sends a notification when a proposal is rejected
func (n *Notifier) NotifyProposalRejected(proposal *Proposal) {
	title := fmt.Sprintf("Proposal Rejected: %s", proposal.Title)
	message := fmt.Sprintf("Proposal %s has been rejected.", proposal.ID)

	notification := &Notification{
		ID:        generateNotificationID(),
		Event:     EventProposalRejected,
		Title:     title,
		Message:   message,
		Data:      proposal,
		Timestamp: time.Now(),
		Read:      false,
		Priority:  n.getPriority(EventProposalRejected),
	}

	n.sendNotification(notification)
}

// NotifyUpgradeScheduled sends a notification when an upgrade is scheduled
func (n *Notifier) NotifyUpgradeScheduled(proposal *Proposal) {
	title := fmt.Sprintf("Upgrade Scheduled: %s", proposal.UpgradeInfo.Name)
	message := fmt.Sprintf("Upgrade %s has been scheduled for block height %d.",
		proposal.UpgradeInfo.Name, proposal.UpgradeHeight)

	notification := &Notification{
		ID:        generateNotificationID(),
		Event:     EventUpgradeScheduled,
		Title:     title,
		Message:   message,
		Data:      proposal.UpgradeInfo,
		Timestamp: time.Now(),
		Read:      false,
		Priority:  n.getPriority(EventUpgradeScheduled),
	}

	n.sendNotification(notification)
}

// NotifyUpgradeTriggered sends a notification when an upgrade is triggered
func (n *Notifier) NotifyUpgradeTriggered(upgrade *UpgradeInfo) {
	title := fmt.Sprintf("Upgrade Started: %s", upgrade.Name)
	message := fmt.Sprintf("Upgrade %s has been triggered at block height %d.",
		upgrade.Name, upgrade.Height)

	notification := &Notification{
		ID:        generateNotificationID(),
		Event:     EventUpgradeTriggered,
		Title:     title,
		Message:   message,
		Data:      upgrade,
		Timestamp: time.Now(),
		Read:      false,
		Priority:  n.getPriority(EventUpgradeTriggered),
	}

	n.sendNotification(notification)
}

// NotifyUpgradeCompleted sends a notification when an upgrade is completed
func (n *Notifier) NotifyUpgradeCompleted(upgrade *UpgradeInfo) {
	title := fmt.Sprintf("Upgrade Completed: %s", upgrade.Name)
	message := fmt.Sprintf("Upgrade %s has been completed successfully.", upgrade.Name)

	notification := &Notification{
		ID:        generateNotificationID(),
		Event:     EventUpgradeCompleted,
		Title:     title,
		Message:   message,
		Data:      upgrade,
		Timestamp: time.Now(),
		Read:      false,
		Priority:  n.getPriority(EventUpgradeCompleted),
	}

	n.sendNotification(notification)
}

// NotifyUpgradeFailed sends a notification when an upgrade fails
func (n *Notifier) NotifyUpgradeFailed(upgrade *UpgradeInfo, err error) {
	title := fmt.Sprintf("Upgrade Failed: %s", upgrade.Name)
	message := fmt.Sprintf("Upgrade %s has failed: %v", upgrade.Name, err)

	notification := &Notification{
		ID:        generateNotificationID(),
		Event:     EventUpgradeFailed,
		Title:     title,
		Message:   message,
		Data:      map[string]interface{}{
			"upgrade": upgrade,
			"error":   err.Error(),
		},
		Timestamp: time.Now(),
		Read:      false,
		Priority:  n.getPriority(EventUpgradeFailed),
	}

	n.sendNotification(notification)
}

// NotifyVotingStarted sends a notification when voting starts
func (n *Notifier) NotifyVotingStarted(proposal *Proposal) {
	title := fmt.Sprintf("Voting Started: %s", proposal.Title)
	message := fmt.Sprintf("Voting has started for proposal %s. Voting ends at %s.",
		proposal.ID, proposal.VotingEndTime.Format(time.RFC3339))

	notification := &Notification{
		ID:        generateNotificationID(),
		Event:     EventVotingStarted,
		Title:     title,
		Message:   message,
		Data:      proposal,
		Timestamp: time.Now(),
		Read:      false,
		Priority:  n.getPriority(EventVotingStarted),
	}

	n.sendNotification(notification)
}

// NotifyVotingEnded sends a notification when voting ends
func (n *Notifier) NotifyVotingEnded(proposal *Proposal) {
	title := fmt.Sprintf("Voting Ended: %s", proposal.Title)
	message := fmt.Sprintf("Voting has ended for proposal %s. Final result: %s.",
		proposal.ID, proposal.Status)

	notification := &Notification{
		ID:        generateNotificationID(),
		Event:     EventVotingEnded,
		Title:     title,
		Message:   message,
		Data:      proposal,
		Timestamp: time.Now(),
		Read:      false,
		Priority:  n.getPriority(EventVotingEnded),
	}

	n.sendNotification(notification)
}

// NotifyQuorumReached sends a notification when quorum is reached
func (n *Notifier) NotifyQuorumReached(proposal *Proposal) {
	title := fmt.Sprintf("Quorum Reached: %s", proposal.Title)
	message := fmt.Sprintf("Proposal %s has reached quorum with %.2f%% turnout.",
		proposal.ID, proposal.VotingStats.Turnout*100)

	notification := &Notification{
		ID:        generateNotificationID(),
		Event:     EventQuorumReached,
		Title:     title,
		Message:   message,
		Data:      proposal,
		Timestamp: time.Now(),
		Read:      false,
		Priority:  n.getPriority(EventQuorumReached),
	}

	n.sendNotification(notification)
}

// NotifyEmergencyProposal sends a notification for emergency proposals
func (n *Notifier) NotifyEmergencyProposal(proposal *Proposal) {
	title := fmt.Sprintf("EMERGENCY PROPOSAL: %s", proposal.Title)
	message := fmt.Sprintf("Emergency proposal %s requires immediate attention.", proposal.ID)

	notification := &Notification{
		ID:        generateNotificationID(),
		Event:     EventEmergencyProposal,
		Title:     title,
		Message:   message,
		Data:      proposal,
		Timestamp: time.Now(),
		Read:      false,
		Priority:  n.getPriority(EventEmergencyProposal),
	}

	n.sendNotification(notification)
}

// AddHandler adds a notification handler
func (n *Notifier) AddHandler(handler NotificationHandler) {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.handlers = append(n.handlers, handler)
	n.logger.Info("notification handler added", zap.String("type", handler.GetType()))
}

// RemoveHandler removes a notification handler
func (n *Notifier) RemoveHandler(handlerType string) {
	n.mu.Lock()
	defer n.mu.Unlock()

	for i, handler := range n.handlers {
		if handler.GetType() == handlerType {
			n.handlers = append(n.handlers[:i], n.handlers[i+1:]...)
			n.logger.Info("notification handler removed", zap.String("type", handlerType))
			break
		}
	}
}

// GetNotifications returns all notifications
func (n *Notifier) GetNotifications() []*Notification {
	n.mu.RLock()
	defer n.mu.RUnlock()

	notifications := make([]*Notification, 0, len(n.notifications))
	for _, notification := range n.notifications {
		notifications = append(notifications, notification)
	}

	// Sort by timestamp (newest first)
	sort.Slice(notifications, func(i, j int) bool {
		return notifications[i].Timestamp.After(notifications[j].Timestamp)
	})

	return notifications
}

// GetUnreadNotifications returns unread notifications
func (n *Notifier) GetUnreadNotifications() []*Notification {
	n.mu.RLock()
	defer n.mu.RUnlock()

	var unread []*Notification
	for _, notification := range n.notifications {
		if !notification.Read {
			unread = append(unread, notification)
		}
	}

	// Sort by timestamp (newest first)
	sort.Slice(unread, func(i, j int) bool {
		return unread[i].Timestamp.After(unread[j].Timestamp)
	})

	return unread
}

// MarkAsRead marks a notification as read
func (n *Notifier) MarkAsRead(notificationID string) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	notification, exists := n.notifications[notificationID]
	if !exists {
		return fmt.Errorf("notification not found: %s", notificationID)
	}

	notification.Read = true
	return nil
}

// MarkAllAsRead marks all notifications as read
func (n *Notifier) MarkAllAsRead() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	for _, notification := range n.notifications {
		notification.Read = true
	}

	return nil
}

// CleanupOld removes old notifications
func (n *Notifier) CleanupOld() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	cutoff := time.Now().Add(-n.maxAge)
	removed := 0

	for id, notification := range n.notifications {
		if notification.Timestamp.Before(cutoff) {
			delete(n.notifications, id)
			removed++
		}
	}

	// Also limit by count
	if len(n.notifications) > n.maxNotifications {
		// Convert to slice and sort by timestamp
		notifications := make([]*Notification, 0, len(n.notifications))
		for _, notification := range n.notifications {
			notifications = append(notifications, notification)
		}

		sort.Slice(notifications, func(i, j int) bool {
			return notifications[i].Timestamp.After(notifications[j].Timestamp)
		})

		// Keep only the newest maxNotifications
		n.notifications = make(map[string]*Notification)
		for i := 0; i < n.maxNotifications && i < len(notifications); i++ {
			n.notifications[notifications[i].ID] = notifications[i]
		}

		removed += len(notifications) - n.maxNotifications
	}

	if removed > 0 {
		n.logger.Info("cleaned up old notifications", zap.Int("removed", removed))
	}

	return nil
}

// SetEnabled enables or disables the notification system
func (n *Notifier) SetEnabled(enabled bool) {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.enabled = enabled
	n.logger.Info("notification system enabled status changed", zap.Bool("enabled", enabled))
}

// GetStats returns notification statistics
func (n *Notifier) GetStats() map[string]interface{} {
	n.mu.RLock()
	defer n.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["total_notifications"] = len(n.notifications)
	stats["enabled_handlers"] = n.countEnabledHandlers()

	// Count by event type
	eventCounts := make(map[string]int)
	priorityCounts := make(map[string]int)
	unreadCount := 0

	for _, notification := range n.notifications {
		eventCounts[string(notification.Event)]++
		priorityCounts[string(notification.Priority)]++
		if !notification.Read {
			unreadCount++
		}
	}

	stats["unread_notifications"] = unreadCount
	stats["by_event"] = eventCounts
	stats["by_priority"] = priorityCounts

	return stats
}

// sendNotification sends a notification to all handlers
func (n *Notifier) sendNotification(notification *Notification) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if !n.enabled {
		return
	}

	// Store the notification
	n.notifications[notification.ID] = notification

	n.logger.Info("sending notification",
		zap.String("id", notification.ID),
		zap.String("event", string(notification.Event)),
		zap.String("title", notification.Title),
		zap.String("priority", string(notification.Priority)))

	// Send to all enabled handlers
	for _, handler := range n.handlers {
		if handler.IsEnabled() {
			go func(h NotificationHandler, notif *Notification) {
				if err := h.Handle(notif); err != nil {
					n.logger.Error("notification handler failed",
						zap.String("handler", h.GetType()),
						zap.String("notification_id", notif.ID),
						zap.Error(err))
				}
			}(handler, notification)
		}
	}
}

// getPriority returns the priority for an event
func (n *Notifier) getPriority(event NotificationEvent) NotificationPriority {
	if priority, exists := n.priorities[event]; exists {
		return priority
	}
	return PriorityMedium
}

// countEnabledHandlers returns the number of enabled handlers
func (n *Notifier) countEnabledHandlers() int {
	count := 0
	for _, handler := range n.handlers {
		if handler.IsEnabled() {
			count++
		}
	}
	return count
}

// generateNotificationID generates a unique notification ID
func generateNotificationID() string {
	return fmt.Sprintf("notif_%d", time.Now().UnixNano())
}

// getDefaultPriorities returns default priority mappings for events
func getDefaultPriorities() map[NotificationEvent]NotificationPriority {
	return map[NotificationEvent]NotificationPriority{
		EventNewProposal:       PriorityMedium,
		EventProposalPassed:    PriorityHigh,
		EventProposalRejected:  PriorityMedium,
		EventUpgradeScheduled:  PriorityHigh,
		EventUpgradeTriggered:  PriorityCritical,
		EventUpgradeCompleted:  PriorityHigh,
		EventUpgradeFailed:     PriorityCritical,
		EventVotingStarted:     PriorityMedium,
		EventVotingEnded:       PriorityMedium,
		EventQuorumReached:     PriorityHigh,
		EventEmergencyProposal: PriorityCritical,
	}
}