package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"

	"github.com/TrailBlazors/realtime-chat-railway/internal/chat"
	"github.com/TrailBlazors/realtime-chat-railway/internal/config"
	"github.com/TrailBlazors/realtime-chat-railway/internal/middleware"
	"github.com/TrailBlazors/realtime-chat-railway/internal/store"
	"github.com/gorilla/mux"
)

func main() {
	// Setup structured logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Load configuration
	cfg := config.Load()
	chat.InitClient(cfg)

	// Initialize store
	var messageStore store.Store
	if cfg.RedisURL != "" {
		redisStore, err := store.NewRedisStore(cfg.RedisURL, cfg.MessageTTL, cfg.MaxMessages)
		if err != nil {
			slog.Warn("failed to connect to Redis, falling back to no-op store", "error", err)
			messageStore = store.NewNoOpStore()
		} else {
			messageStore = redisStore
		}
	} else {
		messageStore = store.NewNoOpStore()
	}
	defer messageStore.Close()

	// Initialize hub with store
	hub := chat.NewHub(messageStore)
	go hub.Run()

	// Initialize middleware
	rateLimiter := middleware.NewRateLimiter(cfg.RateLimit)
	auth := middleware.NewAuth(cfg.AuthToken)

	// Setup router
	r := mux.NewRouter()

	// WebSocket endpoint (with rate limiting and auth)
	r.HandleFunc("/ws", rateLimiter.MiddlewareFunc(auth.MiddlewareFunc(
		func(w http.ResponseWriter, r *http.Request) {
			chat.ServeWs(hub, w, r)
		},
	)))

	// Health check (no auth required)
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "ok",
			"rooms":  hub.GetRoomCount(),
		})
	})

	// Static files (apply rate limiting)
	r.PathPrefix("/").Handler(rateLimiter.Middleware(
		http.FileServer(http.Dir("./web/static")),
	))

	slog.Info("server starting",
		"port", cfg.Port,
		"auth_enabled", cfg.AuthEnabled(),
		"rate_limit", cfg.RateLimit,
		"allowed_origins", cfg.AllowedOrigins,
	)

	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}
