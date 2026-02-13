package services

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/zap"
)

// newTestPresenceService creates a PresenceService for testing.
// It returns the service and a cancel function that shuts it down.
func newTestPresenceService(t *testing.T) *PresenceService {
	t.Helper()
	logger := zap.NewNop()
	ps := NewPresenceService(logger)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = ps.Shutdown(ctx)
	})
	return ps
}

func TestPresenceService_ConnectMakesUserOnline(t *testing.T) {
	ps := newTestPresenceService(t)

	if ps.IsUserOnline(1) {
		t.Fatal("user should not be online before connecting")
	}

	ps.UserConnected(1)

	if !ps.IsUserOnline(1) {
		t.Fatal("user should be online after connecting")
	}
}

func TestPresenceService_DisconnectMakesUserOffline(t *testing.T) {
	ps := newTestPresenceService(t)

	ps.UserConnected(1)
	ps.UserDisconnected(1)

	if ps.IsUserOnline(1) {
		t.Fatal("user should be offline after disconnecting")
	}
}

func TestPresenceService_MultipleConnections(t *testing.T) {
	ps := newTestPresenceService(t)

	// Connect 3 times
	ps.UserConnected(1)
	ps.UserConnected(1)
	ps.UserConnected(1)

	if !ps.IsUserOnline(1) {
		t.Fatal("user should be online with 3 connections")
	}

	// Disconnect 2 times — still 1 connection
	ps.UserDisconnected(1)
	ps.UserDisconnected(1)

	if !ps.IsUserOnline(1) {
		t.Fatal("user should still be online with 1 remaining connection")
	}

	// Disconnect last
	ps.UserDisconnected(1)

	if ps.IsUserOnline(1) {
		t.Fatal("user should be offline after all connections closed")
	}
}

func TestPresenceService_CallbackOnFirstConnect(t *testing.T) {
	ps := newTestPresenceService(t)

	var calls int32
	var lastOnline bool
	var mu sync.Mutex
	done := make(chan struct{}, 2)

	ps.SetOnChangeCallback(func(userID uint, isOnline bool) {
		mu.Lock()
		atomic.AddInt32(&calls, 1)
		lastOnline = isOnline
		mu.Unlock()
		done <- struct{}{}
	})

	// First connect → callback fires
	ps.UserConnected(1)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for callback on first connect")
	}

	mu.Lock()
	if atomic.LoadInt32(&calls) != 1 {
		t.Errorf("expected 1 callback call, got %d", atomic.LoadInt32(&calls))
	}
	if !lastOnline {
		t.Error("callback should report online=true on first connect")
	}
	mu.Unlock()

	// Second connect → callback should NOT fire (already online)
	ps.UserConnected(1)

	// Give a short window for any spurious callback
	select {
	case <-done:
		t.Error("callback should not fire on second connect when already online")
	case <-time.After(100 * time.Millisecond):
		// Expected — no callback
	}
}

func TestPresenceService_CallbackOnLastDisconnect(t *testing.T) {
	ps := newTestPresenceService(t)

	var calls int32
	done := make(chan struct{}, 5)

	ps.SetOnChangeCallback(func(userID uint, isOnline bool) {
		atomic.AddInt32(&calls, 1)
		done <- struct{}{}
	})

	ps.UserConnected(1)
	<-done // consume connect callback
	atomic.StoreInt32(&calls, 0)

	ps.UserConnected(1) // second connection, no callback

	// Disconnect first — still one connection left, no offline callback
	ps.UserDisconnected(1)

	select {
	case <-done:
		t.Error("callback should not fire when connections remain")
	case <-time.After(100 * time.Millisecond):
	}

	// Disconnect last → callback fires with isOnline=false
	ps.UserDisconnected(1)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for callback on last disconnect")
	}

	if atomic.LoadInt32(&calls) != 1 {
		t.Errorf("expected 1 offline callback, got %d", atomic.LoadInt32(&calls))
	}
}

func TestPresenceService_GetUserStatus(t *testing.T) {
	ps := newTestPresenceService(t)

	// Unknown user returns zero-value status
	status := ps.GetUserStatus(99)
	if status.IsOnline {
		t.Error("unknown user should not be online")
	}
	if status.ConnCount != 0 {
		t.Errorf("unknown user ConnCount = %d, want 0", status.ConnCount)
	}

	// Connect and check
	ps.UserConnected(1)
	ps.UserConnected(1)
	status = ps.GetUserStatus(1)

	if !status.IsOnline {
		t.Error("connected user should be online")
	}
	if status.ConnCount != 2 {
		t.Errorf("ConnCount = %d, want 2", status.ConnCount)
	}
	if status.UserID != 1 {
		t.Errorf("UserID = %d, want 1", status.UserID)
	}
}

func TestPresenceService_GetOnlineUsers(t *testing.T) {
	ps := newTestPresenceService(t)

	ps.UserConnected(1)
	ps.UserConnected(2)
	ps.UserConnected(3)
	ps.UserDisconnected(2) // user 2 goes offline

	online := ps.GetOnlineUsers()
	onlineSet := make(map[uint]bool)
	for _, id := range online {
		onlineSet[id] = true
	}

	if !onlineSet[1] {
		t.Error("user 1 should be online")
	}
	if onlineSet[2] {
		t.Error("user 2 should be offline")
	}
	if !onlineSet[3] {
		t.Error("user 3 should be online")
	}
	if len(online) != 2 {
		t.Errorf("expected 2 online users, got %d", len(online))
	}
}

func TestPresenceService_DisconnectUnknownUser(t *testing.T) {
	ps := newTestPresenceService(t)

	// Should not panic
	ps.UserDisconnected(999)

	if ps.IsUserOnline(999) {
		t.Error("unknown disconnected user should not appear online")
	}
}

func TestPresenceService_UpdateActivity(t *testing.T) {
	ps := newTestPresenceService(t)

	ps.UserConnected(1)
	before := ps.GetUserStatus(1).LastSeen

	time.Sleep(10 * time.Millisecond)
	ps.UpdateActivity(1)

	after := ps.GetUserStatus(1).LastSeen
	if !after.After(before) {
		t.Error("UpdateActivity should advance LastSeen")
	}
}

func TestPresenceService_ConcurrentAccess(t *testing.T) {
	ps := newTestPresenceService(t)

	var wg sync.WaitGroup
	const goroutines = 50

	// Half connect, half disconnect the same users concurrently
	for i := uint(0); i < goroutines; i++ {
		wg.Add(1)
		go func(id uint) {
			defer wg.Done()
			userID := id%5 + 1
			ps.UserConnected(userID)
			ps.IsUserOnline(userID)
			ps.GetUserStatus(userID)
			ps.UpdateActivity(userID)
			ps.GetOnlineUsers()
			ps.UserDisconnected(userID)
		}(i)
	}

	wg.Wait()
	// No panics or race detector failures = pass
}
