package store

import (
	"context"
	"testing"
)

func TestNoOpStore(t *testing.T) {
	s := NewNoOpStore()

	ctx := context.Background()

	// SaveMessage should not error
	err := s.SaveMessage(ctx, Message{
		Type:     "message",
		Username: "test",
		Content:  "hello",
		Room:     "test-room",
		Time:     "2024-01-01T00:00:00Z",
	})
	if err != nil {
		t.Errorf("SaveMessage should not error: %v", err)
	}

	// GetRecentMessages should return empty slice
	messages, err := s.GetRecentMessages(ctx, "test-room", 10)
	if err != nil {
		t.Errorf("GetRecentMessages should not error: %v", err)
	}
	if len(messages) != 0 {
		t.Errorf("expected 0 messages, got %d", len(messages))
	}

	// Close should not error
	err = s.Close()
	if err != nil {
		t.Errorf("Close should not error: %v", err)
	}
}

func TestMessage_Structure(t *testing.T) {
	msg := Message{
		Type:     "message",
		Username: "testuser",
		Content:  "Hello, World!",
		Room:     "general",
		Time:     "2024-01-01T12:00:00Z",
	}

	if msg.Type != "message" {
		t.Errorf("expected type 'message', got %s", msg.Type)
	}
	if msg.Username != "testuser" {
		t.Errorf("expected username 'testuser', got %s", msg.Username)
	}
	if msg.Content != "Hello, World!" {
		t.Errorf("expected content 'Hello, World!', got %s", msg.Content)
	}
	if msg.Room != "general" {
		t.Errorf("expected room 'general', got %s", msg.Room)
	}
	if msg.Time != "2024-01-01T12:00:00Z" {
		t.Errorf("expected time '2024-01-01T12:00:00Z', got %s", msg.Time)
	}
}
