package config

import (
	"os"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	// Clear any existing env vars
	os.Unsetenv("PORT")
	os.Unsetenv("ALLOWED_ORIGINS")
	os.Unsetenv("REDIS_URL")
	os.Unsetenv("AUTH_TOKEN")
	os.Unsetenv("RATE_LIMIT")
	os.Unsetenv("MAX_MESSAGE_SIZE")

	cfg := Load()

	if cfg.Port != "8080" {
		t.Errorf("expected default port 8080, got %s", cfg.Port)
	}
	if len(cfg.AllowedOrigins) != 1 || cfg.AllowedOrigins[0] != "*" {
		t.Errorf("expected default allowed origins [*], got %v", cfg.AllowedOrigins)
	}
	if cfg.RateLimit != 60 {
		t.Errorf("expected default rate limit 60, got %d", cfg.RateLimit)
	}
	if cfg.MaxMessageSize != 4096 {
		t.Errorf("expected default max message size 4096, got %d", cfg.MaxMessageSize)
	}
}

func TestLoad_CustomValues(t *testing.T) {
	os.Setenv("PORT", "3000")
	os.Setenv("ALLOWED_ORIGINS", "https://example.com,https://app.example.com")
	os.Setenv("REDIS_URL", "redis://localhost:6379")
	os.Setenv("AUTH_TOKEN", "secret123")
	os.Setenv("RATE_LIMIT", "100")
	os.Setenv("MAX_MESSAGE_SIZE", "8192")
	defer func() {
		os.Unsetenv("PORT")
		os.Unsetenv("ALLOWED_ORIGINS")
		os.Unsetenv("REDIS_URL")
		os.Unsetenv("AUTH_TOKEN")
		os.Unsetenv("RATE_LIMIT")
		os.Unsetenv("MAX_MESSAGE_SIZE")
	}()

	cfg := Load()

	if cfg.Port != "3000" {
		t.Errorf("expected port 3000, got %s", cfg.Port)
	}
	if len(cfg.AllowedOrigins) != 2 {
		t.Errorf("expected 2 allowed origins, got %d", len(cfg.AllowedOrigins))
	}
	if cfg.AllowedOrigins[0] != "https://example.com" {
		t.Errorf("expected first origin https://example.com, got %s", cfg.AllowedOrigins[0])
	}
	if cfg.RedisURL != "redis://localhost:6379" {
		t.Errorf("expected redis URL, got %s", cfg.RedisURL)
	}
	if cfg.AuthToken != "secret123" {
		t.Errorf("expected auth token, got %s", cfg.AuthToken)
	}
	if cfg.RateLimit != 100 {
		t.Errorf("expected rate limit 100, got %d", cfg.RateLimit)
	}
	if cfg.MaxMessageSize != 8192 {
		t.Errorf("expected max message size 8192, got %d", cfg.MaxMessageSize)
	}
}

func TestIsOriginAllowed_Wildcard(t *testing.T) {
	cfg := &Config{AllowedOrigins: []string{"*"}}

	if !cfg.IsOriginAllowed("https://example.com") {
		t.Error("wildcard should allow any origin")
	}
	if !cfg.IsOriginAllowed("http://localhost:3000") {
		t.Error("wildcard should allow localhost")
	}
}

func TestIsOriginAllowed_Whitelist(t *testing.T) {
	cfg := &Config{AllowedOrigins: []string{"https://example.com", "https://app.example.com"}}

	if !cfg.IsOriginAllowed("https://example.com") {
		t.Error("should allow whitelisted origin")
	}
	if !cfg.IsOriginAllowed("https://app.example.com") {
		t.Error("should allow whitelisted origin")
	}
	if cfg.IsOriginAllowed("https://evil.com") {
		t.Error("should not allow non-whitelisted origin")
	}
	if cfg.IsOriginAllowed("http://example.com") {
		t.Error("should not allow http when https is whitelisted")
	}
}

func TestAuthEnabled(t *testing.T) {
	cfg := &Config{AuthToken: ""}
	if cfg.AuthEnabled() {
		t.Error("auth should be disabled when token is empty")
	}

	cfg.AuthToken = "secret"
	if !cfg.AuthEnabled() {
		t.Error("auth should be enabled when token is set")
	}
}
