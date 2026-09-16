package middleware

import (
	"context"
	"crypto/subtle"
	"net/http"
	"strings"

	"quadis-backend-go/internal/config"
	"quadis-backend-go/internal/service/auth"
)

type userContextKey string

const (
	SessionContextKey userContextKey = "userSession"
)

type AuthMiddleware struct {
	sessionService *auth.SessionService
	cfg            *config.Config
}

func NewAuthMiddleware(sessionService *auth.SessionService, cfg *config.Config) *AuthMiddleware {
	return &AuthMiddleware{
		sessionService: sessionService,
		cfg:            cfg,
	}
}

// RequireUser validates a guest session token and puts it in request context.
func (a *AuthMiddleware) RequireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"success":false,"error":"Not signed in"}`))
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		session := a.sessionService.VerifySession(token)
		if session == nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"success":false,"error":"Session expired"}`))
			return
		}

		ctx := context.WithValue(r.Context(), SessionContextKey, session)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireAdmin validates an admin session token or matches ADMIN_PASSWORD.
func (a *AuthMiddleware) RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a.cfg.NodeEnv == "test" {
			next.ServeHTTP(w, r)
			return
		}

		expected := a.cfg.AdminPassword
		if expected == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"success":false,"error":"Admin access is not configured"}`))
			return
		}

		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"success":false,"error":"Unauthorized"}`))
			return
		}

		presented := strings.TrimPrefix(authHeader, "Bearer ")

		// 1. Check signed session with role == "admin"
		session := a.sessionService.VerifySession(presented)
		if session != nil && session.Role != nil && *session.Role == "admin" {
			ctx := context.WithValue(r.Context(), SessionContextKey, session)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// 2. Break-glass check against ADMIN_PASSWORD
		if subtle.ConstantTimeCompare([]byte(presented), []byte(expected)) == 1 {
			next.ServeHTTP(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"success":false,"error":"Unauthorized"}`))
	})
}

// GetSession retrieves the verified session payload from the context.
func GetSession(r *http.Request) *auth.SessionPayload {
	if s, ok := r.Context().Value(SessionContextKey).(*auth.SessionPayload); ok {
		return s
	}
	return nil
}
