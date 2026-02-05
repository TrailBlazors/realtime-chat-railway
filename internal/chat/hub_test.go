package chat

import (
	"context"
	"testing"
	"time"

	"github.com/TrailBlazors/realtime-chat-railway/internal/store"
)

func TestHub_RoomManagement(t *testing.T) {
	s := store.NewNoOpStore()
	hub := NewHub(s)
	go hub.Run()

	// Give hub time to start
	time.Sleep(10 * time.Millisecond)

	// Initially no rooms
	if hub.GetRoomCount() != 0 {
		t.Errorf("expected 0 rooms, got %d", hub.GetRoomCount())
	}

	// Create mock clients
	client1 := &Client{
		hub:      hub,
		send:     make(chan []byte, 256),
		room:     "test-room",
		username: "user1",
	}

	client2 := &Client{
		hub:      hub,
		send:     make(chan []byte, 256),
		room:     "test-room",
		username: "user2",
	}

	// Register first client
	hub.register <- client1
	time.Sleep(10 * time.Millisecond)

	if hub.GetRoomCount() != 1 {
		t.Errorf("expected 1 room, got %d", hub.GetRoomCount())
	}
	if hub.GetClientCount("test-room") != 1 {
		t.Errorf("expected 1 client in room, got %d", hub.GetClientCount("test-room"))
	}

	// Register second client
	hub.register <- client2
	time.Sleep(10 * time.Millisecond)

	if hub.GetClientCount("test-room") != 2 {
		t.Errorf("expected 2 clients in room, got %d", hub.GetClientCount("test-room"))
	}

	// Unregister first client
	hub.unregister <- client1
	time.Sleep(10 * time.Millisecond)

	if hub.GetClientCount("test-room") != 1 {
		t.Errorf("expected 1 client in room after unregister, got %d", hub.GetClientCount("test-room"))
	}

	// Unregister second client (room should be deleted)
	hub.unregister <- client2
	time.Sleep(10 * time.Millisecond)

	if hub.GetRoomCount() != 0 {
		t.Errorf("expected 0 rooms after all clients left, got %d", hub.GetRoomCount())
	}
}

func TestHub_BroadcastMessage(t *testing.T) {
	s := store.NewNoOpStore()
	hub := NewHub(s)
	go hub.Run()

	time.Sleep(10 * time.Millisecond)

	client1 := &Client{
		hub:      hub,
		send:     make(chan []byte, 256),
		room:     "test-room",
		username: "user1",
	}

	client2 := &Client{
		hub:      hub,
		send:     make(chan []byte, 256),
		room:     "test-room",
		username: "user2",
	}

	client3 := &Client{
		hub:      hub,
		send:     make(chan []byte, 256),
		room:     "other-room",
		username: "user3",
	}

	hub.register <- client1
	hub.register <- client2
	hub.register <- client3
	time.Sleep(10 * time.Millisecond)

	// Broadcast to test-room
	msg := Message{
		Type:     "message",
		Username: "user1",
		Content:  "Hello!",
		Room:     "test-room",
		Time:     time.Now().Format(time.RFC3339),
	}
	hub.BroadcastMessage(msg)

	// Wait for broadcast
	time.Sleep(50 * time.Millisecond)

	// Check client1 and client2 received the message
	select {
	case <-client1.send:
		// OK
	default:
		t.Error("client1 should have received the message")
	}

	select {
	case <-client2.send:
		// OK
	default:
		t.Error("client2 should have received the message")
	}

	// client3 should NOT have received the message (different room)
	select {
	case <-client3.send:
		t.Error("client3 should not have received the message")
	default:
		// OK
	}
}

// Mock store for testing
type mockStore struct {
	messages []store.Message
}

func (m *mockStore) SaveMessage(ctx context.Context, msg store.Message) error {
	// Only persist actual messages, not join/leave (matching RedisStore behavior)
	if msg.Type != "message" {
		return nil
	}
	m.messages = append(m.messages, msg)
	return nil
}

func (m *mockStore) GetRecentMessages(ctx context.Context, room string, limit int) ([]store.Message, error) {
	return m.messages, nil
}

func (m *mockStore) Close() error {
	return nil
}

func TestHub_MessagePersistence(t *testing.T) {
	ms := &mockStore{}
	hub := NewHub(ms)
	go hub.Run()

	time.Sleep(10 * time.Millisecond)

	client := &Client{
		hub:      hub,
		send:     make(chan []byte, 256),
		room:     "test-room",
		username: "user1",
	}

	hub.register <- client
	time.Sleep(10 * time.Millisecond)

	// Broadcast a message
	msg := Message{
		Type:     "message",
		Username: "user1",
		Content:  "Hello!",
		Room:     "test-room",
		Time:     time.Now().Format(time.RFC3339),
	}
	hub.BroadcastMessage(msg)
	time.Sleep(50 * time.Millisecond)

	// Check message was persisted
	if len(ms.messages) != 1 {
		t.Errorf("expected 1 persisted message, got %d", len(ms.messages))
	}

	// Join/leave messages should not be persisted
	joinMsg := Message{
		Type:     "join",
		Username: "user2",
		Content:  "user2 joined",
		Room:     "test-room",
		Time:     time.Now().Format(time.RFC3339),
	}
	hub.BroadcastMessage(joinMsg)
	time.Sleep(50 * time.Millisecond)

	if len(ms.messages) != 1 {
		t.Errorf("join message should not be persisted, got %d messages", len(ms.messages))
	}
}
