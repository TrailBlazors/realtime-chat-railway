package store

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

type Message struct {
	Type     string `json:"type"`
	Username string `json:"username"`
	Content  string `json:"content"`
	Room     string `json:"room"`
	Time     string `json:"time"`
}

type Store interface {
	SaveMessage(ctx context.Context, msg Message) error
	GetRecentMessages(ctx context.Context, room string, limit int) ([]Message, error)
	Close() error
}

// RedisStore implements Store using Redis
type RedisStore struct {
	client      *redis.Client
	ttl         time.Duration
	maxMessages int64
}

func NewRedisStore(redisURL string, ttlHours int, maxMessages int) (*RedisStore, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}

	client := redis.NewClient(opt)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	slog.Info("connected to Redis", "url", redisURL)

	return &RedisStore{
		client:      client,
		ttl:         time.Duration(ttlHours) * time.Hour,
		maxMessages: int64(maxMessages),
	}, nil
}

func (s *RedisStore) roomKey(room string) string {
	return "chat:room:" + room + ":messages"
}

func (s *RedisStore) SaveMessage(ctx context.Context, msg Message) error {
	if msg.Type != "message" {
		return nil // Only persist actual messages, not join/leave
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	key := s.roomKey(msg.Room)

	pipe := s.client.Pipeline()
	pipe.LPush(ctx, key, data)
	pipe.LTrim(ctx, key, 0, s.maxMessages-1)
	pipe.Expire(ctx, key, s.ttl)

	_, err = pipe.Exec(ctx)
	return err
}

func (s *RedisStore) GetRecentMessages(ctx context.Context, room string, limit int) ([]Message, error) {
	key := s.roomKey(room)

	data, err := s.client.LRange(ctx, key, 0, int64(limit-1)).Result()
	if err != nil {
		return nil, err
	}

	messages := make([]Message, 0, len(data))
	// Reverse order since we use LPUSH (newest first in list)
	for i := len(data) - 1; i >= 0; i-- {
		var msg Message
		if err := json.Unmarshal([]byte(data[i]), &msg); err != nil {
			slog.Warn("failed to unmarshal message from Redis", "error", err)
			continue
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

func (s *RedisStore) Close() error {
	return s.client.Close()
}

// NoOpStore is a fallback when Redis is not configured
type NoOpStore struct{}

func NewNoOpStore() *NoOpStore {
	slog.Info("Redis not configured, message persistence disabled")
	return &NoOpStore{}
}

func (s *NoOpStore) SaveMessage(ctx context.Context, msg Message) error {
	return nil
}

func (s *NoOpStore) GetRecentMessages(ctx context.Context, room string, limit int) ([]Message, error) {
	return []Message{}, nil
}

func (s *NoOpStore) Close() error {
	return nil
}
