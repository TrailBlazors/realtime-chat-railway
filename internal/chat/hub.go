package chat

import (
	"encoding/json"
	"log"
	"sync"
)

type Message struct {
	Type     string `json:"type"` // "message", "join", "leave"
	Username string `json:"username"`
	Content  string `json:"content"`
	Room     string `json:"room"`
	Time     string `json:"time"`
}

type Hub struct {
	// Registered clients per room
	rooms map[string]map[*Client]bool

	// Inbound messages from clients
	broadcast chan Message

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	mu sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		rooms:      make(map[string]map[*Client]bool),
		broadcast:  make(chan Message, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if h.rooms[client.room] == nil {
				h.rooms[client.room] = make(map[*Client]bool)
			}
			h.rooms[client.room][client] = true
			h.mu.Unlock()
			log.Printf("Client registered to room: %s", client.room)

		case client := <-h.unregister:
			h.mu.Lock()
			if clients, ok := h.rooms[client.room]; ok {
				if _, exists := clients[client]; exists {
					delete(clients, client)
					close(client.send)
					if len(clients) == 0 {
						delete(h.rooms, client.room)
					}
				}
			}
			h.mu.Unlock()
			log.Printf("Client unregistered from room: %s", client.room)

		case message := <-h.broadcast:
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
		}
	}
}

func (h *Hub) BroadcastMessage(msg Message) {
	h.broadcast <- msg
}
