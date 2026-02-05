package chat

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/TrailBlazors/realtime-chat-railway/internal/config"
	"github.com/gorilla/websocket"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
)

var (
	cfg      *config.Config
	upgrader websocket.Upgrader
)

func InitClient(c *config.Config) {
	cfg = c
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			if origin == "" {
				return true // Allow requests without Origin header (e.g., from same origin)
			}
			allowed := cfg.IsOriginAllowed(origin)
			if !allowed {
				slog.Warn("rejected connection from disallowed origin", "origin", origin)
			}
			return allowed
		},
	}
}

type Client struct {
	hub      *Hub
	conn     *websocket.Conn
	send     chan []byte
	room     string
	username string
}

func ServeWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("websocket upgrade failed", "error", err)
		return
	}

	room := r.URL.Query().Get("room")
	username := r.URL.Query().Get("username")

	if room == "" {
		room = "general"
	}
	if username == "" {
		username = "anonymous"
	}

	client := &Client{
		hub:      hub,
		conn:     conn,
		send:     make(chan []byte, 256),
		room:     room,
		username: username,
	}

	client.hub.register <- client

	slog.Info("client connected",
		"username", username,
		"room", room,
		"remote_addr", r.RemoteAddr,
	)

	// Send recent messages from history
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	messages, err := hub.store.GetRecentMessages(ctx, room, 50)
	cancel()

	if err != nil {
		slog.Warn("failed to load message history", "error", err, "room", room)
	} else {
		for _, msg := range messages {
			data, _ := json.Marshal(msg)
			select {
			case client.send <- data:
			default:
			}
		}
	}

	// Send join message
	joinMsg := Message{
		Type:     "join",
		Username: username,
		Content:  username + " joined the room",
		Room:     room,
		Time:     time.Now().Format(time.RFC3339),
	}
	hub.BroadcastMessage(joinMsg)

	go client.writePump()
	go client.readPump()
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()

		slog.Info("client disconnected",
			"username", c.username,
			"room", c.room,
		)

		leaveMsg := Message{
			Type:     "leave",
			Username: c.username,
			Content:  c.username + " left the room",
			Room:     c.room,
			Time:     time.Now().Format(time.RFC3339),
		}
		c.hub.BroadcastMessage(leaveMsg)
	}()

	c.conn.SetReadLimit(cfg.MaxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				slog.Warn("unexpected websocket close",
					"error", err,
					"username", c.username,
				)
			}
			break
		}

		var msg Message
		if err := json.Unmarshal(message, &msg); err != nil {
			slog.Warn("failed to unmarshal message",
				"error", err,
				"username", c.username,
			)
			continue
		}

		msg.Username = c.username
		msg.Room = c.room
		msg.Time = time.Now().Format(time.RFC3339)
		msg.Type = "message"

		c.hub.BroadcastMessage(msg)
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
