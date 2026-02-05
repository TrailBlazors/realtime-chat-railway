package middleware

import (
	"log/slog"
	"net/http"
	"strings"
)

type Auth struct {
	token string
}

func NewAuth(token string) *Auth {
	if token != "" {
		slog.Info("authentication enabled")
	} else {
		slog.Info("authentication disabled (no AUTH_TOKEN set)")
	}
	return &Auth{token: token}
}

func (a *Auth) Enabled() bool {
	return a.token != ""
}

func (a *Auth) ValidateRequest(r *http.Request) bool {
	if !a.Enabled() {
		return true
	}

	// Check query parameter
	if token := r.URL.Query().Get("token"); token == a.token {
		return true
	}

	// Check Authorization header (Bearer token)
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == a.token {
			return true
		}
	}

	return false
}

func (a *Auth) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !a.ValidateRequest(r) {
			slog.Warn("unauthorized request", "ip", r.RemoteAddr, "path", r.URL.Path)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *Auth) MiddlewareFunc(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !a.ValidateRequest(r) {
			slog.Warn("unauthorized request", "ip", r.RemoteAddr, "path", r.URL.Path)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}
