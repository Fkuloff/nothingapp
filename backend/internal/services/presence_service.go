package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// UserStatus represents online status of a user
type UserStatus struct {
	UserID    uint
	ConnCount int
	IsOnline  bool
	LastSeen  time.Time
}

// PresenceService manages user online/offline status
type PresenceService struct {
	users    map[uint]*UserStatus
	onChange func(userID uint, isOnline bool)
	logger   *zap.Logger
	stopChan chan struct{}
	doneChan chan struct{}
	mu       sync.RWMutex
}

// NewPresenceService creates a new presence service
func NewPresenceService(logger *zap.Logger) *PresenceService {
	ps := &PresenceService{
		users:    make(map[uint]*UserStatus),
		logger:   logger,
		stopChan: make(chan struct{}),
		doneChan: make(chan struct{}),
	}

	// Start cleanup goroutine to mark inactive users as offline
	go ps.cleanupInactiveUsers()

	return ps
}

// SetOnChangeCallback sets a callback that fires when user status changes
func (ps *PresenceService) SetOnChangeCallback(callback func(userID uint, isOnline bool)) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.onChange = callback
}

// UserConnected marks a user as online when they connect
func (ps *PresenceService) UserConnected(userID uint) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	status, exists := ps.users[userID]
	if !exists {
		ps.users[userID] = &UserStatus{
			UserID:    userID,
			IsOnline:  true,
			LastSeen:  time.Now(),
			ConnCount: 1,
		}
		// Trigger callback for new online user
		if ps.onChange != nil {
			go ps.onChange(userID, true)
		}
		return
	}

	// User already online, increment connection count
	wasOnline := status.IsOnline
	status.ConnCount++
	status.IsOnline = true
	status.LastSeen = time.Now()

	// Trigger callback if user just came online
	if !wasOnline && ps.onChange != nil {
		go ps.onChange(userID, true)
	}
}

// UserDisconnected marks a user as offline when they disconnect
func (ps *PresenceService) UserDisconnected(userID uint) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	status, exists := ps.users[userID]
	if !exists {
		return
	}

	status.ConnCount--
	status.LastSeen = time.Now()

	// Only mark as offline if no connections remain
	if status.ConnCount <= 0 {
		status.IsOnline = false
		status.ConnCount = 0

		// Trigger callback for offline user
		if ps.onChange != nil {
			go ps.onChange(userID, false)
		}
	}
}

// UpdateActivity updates the last seen time for a user
func (ps *PresenceService) UpdateActivity(userID uint) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if status, exists := ps.users[userID]; exists {
		status.LastSeen = time.Now()
	}
}

// IsUserOnline checks if a user is currently online
func (ps *PresenceService) IsUserOnline(userID uint) bool {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	if status, exists := ps.users[userID]; exists {
		return status.IsOnline
	}
	return false
}

// GetUserStatus returns the full status of a user
func (ps *PresenceService) GetUserStatus(userID uint) *UserStatus {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	if status, exists := ps.users[userID]; exists {
		// Return a copy to avoid race conditions
		return &UserStatus{
			UserID:    status.UserID,
			IsOnline:  status.IsOnline,
			LastSeen:  status.LastSeen,
			ConnCount: status.ConnCount,
		}
	}
	return &UserStatus{
		UserID:   userID,
		IsOnline: false,
		LastSeen: time.Time{},
	}
}

// GetOnlineUsers returns a list of all currently online user IDs
func (ps *PresenceService) GetOnlineUsers() []uint {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	var onlineUsers []uint
	for userID, status := range ps.users {
		if status.IsOnline {
			onlineUsers = append(onlineUsers, userID)
		}
	}
	return onlineUsers
}

// cleanupInactiveUsers periodically checks for users with stale connections
// and marks them as offline if they haven't been active
func (ps *PresenceService) cleanupInactiveUsers() {
	defer close(ps.doneChan)

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ps.stopChan:
			ps.logger.Info("presence cleanup stopped")
			return
		case <-ticker.C:
			ps.mu.Lock()
			now := time.Now()

			for userID, status := range ps.users {
				// Cleanup thresholds:
				// - 2 minutes: User considered offline if no ping/activity received
				//   This accounts for network issues and provides reasonable grace period
				if status.IsOnline && now.Sub(status.LastSeen) > 2*time.Minute {
					status.IsOnline = false
					status.ConnCount = 0

					// Trigger callback
					if ps.onChange != nil {
						go ps.onChange(userID, false)
					}
				}

				// - 1 hour: Remove stale offline entries to prevent unbounded memory growth
				//   This balances memory efficiency with reasonable presence history retention
				if !status.IsOnline && now.Sub(status.LastSeen) > time.Hour {
					delete(ps.users, userID)
				}
			}

			ps.mu.Unlock()
		}
	}
}

// Shutdown gracefully stops the presence service
func (ps *PresenceService) Shutdown(ctx context.Context) error {
	close(ps.stopChan)

	select {
	case <-ps.doneChan:
		ps.logger.Info("presence service stopped gracefully")
		return nil
	case <-ctx.Done():
		return fmt.Errorf("presence shutdown timeout: %w", ctx.Err())
	}
}
