package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuth_Disabled(t *testing.T) {
	auth := NewAuth("")

	if auth.Enabled() {
		t.Error("auth should be disabled when token is empty")
	}

	req := httptest.NewRequest("GET", "/", nil)
	if !auth.ValidateRequest(req) {
		t.Error("should allow all requests when auth is disabled")
	}
}

func TestAuth_QueryParam(t *testing.T) {
	auth := NewAuth("secret123")

	// Valid token
	req := httptest.NewRequest("GET", "/?token=secret123", nil)
	if !auth.ValidateRequest(req) {
		t.Error("should allow valid token in query param")
	}

	// Invalid token
	req = httptest.NewRequest("GET", "/?token=wrong", nil)
	if auth.ValidateRequest(req) {
		t.Error("should reject invalid token")
	}

	// No token
	req = httptest.NewRequest("GET", "/", nil)
	if auth.ValidateRequest(req) {
		t.Error("should reject missing token")
	}
}

func TestAuth_BearerToken(t *testing.T) {
	auth := NewAuth("secret123")

	// Valid bearer token
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer secret123")
	if !auth.ValidateRequest(req) {
		t.Error("should allow valid bearer token")
	}

	// Invalid bearer token
	req = httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer wrong")
	if auth.ValidateRequest(req) {
		t.Error("should reject invalid bearer token")
	}

	// Wrong auth scheme
	req = httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Basic secret123")
	if auth.ValidateRequest(req) {
		t.Error("should reject non-bearer auth")
	}
}

func TestAuth_Middleware(t *testing.T) {
	auth := NewAuth("secret123")

	handler := auth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	// Valid token
	req := httptest.NewRequest("GET", "/?token=secret123", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	// Invalid token
	req = httptest.NewRequest("GET", "/", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}
