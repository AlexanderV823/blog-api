package middleware

import (
	"blog-api/pkg/auth"
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

// contextKey — пользовательский тип ключей контекста для предотвращения конфликтов
type contextKey string

const (
	UserIDKey    contextKey = "userID"
	UserEmailKey contextKey = "userEmail"
	UserNameKey  contextKey = "username"
)

// AuthMiddleware provides JWT authentication
type AuthMiddleware struct {
	jwtManager *auth.JWTManager
}

// NewAuthMiddleware creates a new auth middleware instance
func NewAuthMiddleware(jwtManager *auth.JWTManager) *AuthMiddleware {
	return &AuthMiddleware{
		jwtManager: jwtManager,
	}
}

// RequireAuth requires valid JWT token
func (m *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return m.Handler(next)
}

// Handler адаптирует middleware для работы со стандартными интерфейсами net/http
func (m *AuthMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenStr := extractToken(r)
		if tokenStr == "" {
			writeJSONError(w, "Authorization required metadata missing", http.StatusUnauthorized)
			return
		}

		claims, err := m.jwtManager.ValidateToken(tokenStr)
		if err != nil {
			writeJSONError(w, "Authorization configuration layout exception: "+err.Error(), http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
		ctx = context.WithValue(ctx, UserEmailKey, claims.Email)
		ctx = context.WithValue(ctx, UserNameKey, claims.Username)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// OptionalAuth извлекает JWT-токен при наличии, если он передан, но допускает запросы без него
func (m *AuthMiddleware) OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenStr := extractToken(r)
		if tokenStr != "" {
			if claims, err := m.jwtManager.ValidateToken(tokenStr); err == nil {
				ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
				ctx = context.WithValue(ctx, UserEmailKey, claims.Email)
				ctx = context.WithValue(ctx, UserNameKey, claims.Username)
				r = r.WithContext(ctx)
			}
		}
		next.ServeHTTP(w, r)
	})
}

// extractToken извлекает JWT токен из заголовка Authorization
func extractToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) == 2 && parts[0] == "Bearer" {
		return parts[1]
	}
	return ""
}

// GetUserIDFromContext извлекает ID пользователя из контекста
func GetUserIDFromContext(ctx context.Context) (int, bool) {
	if val := ctx.Value(UserIDKey); val != nil {
		if id, ok := val.(int); ok {
			return id, true
		}
	}
	return 0, false
}

// GetUserEmailFromContext извлекает email пользователя из контекста
func GetUserEmailFromContext(ctx context.Context) (string, bool) {
	if val := ctx.Value(UserEmailKey); val != nil {
		if email, ok := val.(string); ok {
			return email, true
		}
	}
	return "", false
}

// GetUsernameFromContext извлекает username из контекста
func GetUsernameFromContext(ctx context.Context) (string, bool) {
	if val := ctx.Value(UserNameKey); val != nil {
		if name, ok := val.(string); ok {
			return name, true
		}
	}
	return "", false
}

// writeJSONError отправляет ошибку в формате JSON
func writeJSONError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// Chain позволяет объединить несколько middleware в цепочку
func Chain(handler http.HandlerFunc, middlewares ...func(http.HandlerFunc) http.HandlerFunc) http.HandlerFunc {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}
