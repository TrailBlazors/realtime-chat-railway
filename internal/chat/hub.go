package chat

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/TrailBlazors/realtime-chat-railway/internal/store"
)

type Message struct {
	Type     string `json:"type"`
	Username string `json:"username"`
	Content  string `json:"content"`
	Room     string `json:"room"`
	Time     string `json:"time"`
}

type Hub struct {
	rooms      map[string]map[*Client]bool
	broadcast  chan Message
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
	store      store.Store
}

func NewHub(s store.Store) *Hub {
	return &Hub{
		rooms:      make(map[string]map[*Client]bool),
		broadcast:  make(chan Message, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		store:      s,
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if h.rooms[client.room] == nil {
				h.rooms[client.room] = make(map[*Client]bool)
				slog.Info("room created", "room", client.room)
			}
			h.rooms[client.room][client] = true
			clientCount := len(h.rooms[client.room])
			h.mu.Unlock()

			slog.Debug("client registered",
				"room", client.room,
				"username", client.username,
				"clients_in_room", clientCount,
			)

		case client := <-h.unregister:
			h.mu.Lock()
			if clients, ok := h.rooms[client.room]; ok {
				if _, exists := clients[client]; exists {
					delete(clients, client)
					close(client.send)
					if len(clients) == 0 {
						delete(h.rooms, client.room)
						slog.Info("room deleted (empty)", "room", client.room)
					}
				}
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			// Persist message to store
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			if err := h.store.SaveMessage(ctx, store.Message(message)); err != nil {
				slog.Warn("failed to persist message", "error", err, "room", message.Room)
			}
			cancel()

			h.mu.RLock()
			clients := h.rooms[message.Room]
			h.mu.RUnlock()

			messageBytes, _ := json.Marshal(message)
			for client := range clients {
				select {
				case client.send <- messageBytes:
				default:
					close(client.send)
					h.mu.Lock()
					delete(h.rooms[message.Room], client)
					h.mu.Unlock()
				}
			}

			if message.Type == "message" {
				slog.Debug("message broadcast",
					"room", message.Room,
					"username", message.Username,
					"recipients", len(clients),
				)
			}
		}
	}
}

func (h *Hub) BroadcastMessage(msg Message) {
	h.broadcast <- msg
}

func (h *Hub) GetRoomCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.rooms)
}

func (h *Hub) GetClientCount(room string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if clients, ok := h.rooms[room]; ok {
		return len(clients)
	}
	return 0
}
