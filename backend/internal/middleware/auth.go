package middleware

import (
	"context"
	"database/sql"
	"net/http"
	"strings"

	"gemini-clone/backend/internal/auth"
	"gemini-clone/backend/internal/repository"
)

type contextKey string

const userIDContextKey contextKey = "auth_user_id"

func ParseAuthUserID(r *http.Request, jwtSecret string) int64 {
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if header == "" {
		return 0
	}
	if !strings.HasPrefix(strings.ToLower(header), "bearer ") {
		return 0
	}
	token := strings.TrimSpace(header[7:])
	if token == "" {
		return 0
	}
	claims, err := auth.ParseToken(jwtSecret, token)
	if err != nil {
		return 0
	}
	return claims.UserID
}

func RequireAuth(jwtSecret string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := ParseAuthUserID(r, jwtSecret)
		if userID <= 0 {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":{"message":"请先登录"}}`))
			return
		}
		ctx := context.WithValue(r.Context(), userIDContextKey, userID)
		next(w, r.WithContext(ctx))
	}
}

func UserIDFromContext(ctx context.Context) int64 {
	v := ctx.Value(userIDContextKey)
	if id, ok := v.(int64); ok {
		return id
	}
	return 0
}

func RequireAdmin(
	jwtSecret string,
	adminEmail string,
	userRepo *repository.UserRepository,
	next http.HandlerFunc,
) http.HandlerFunc {
	return RequireAuth(jwtSecret, func(w http.ResponseWriter, r *http.Request) {
		userID := UserIDFromContext(r.Context())
		if userID <= 0 {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":{"message":"请先登录"}}`))
			return
		}
		user, err := userRepo.GetByID(r.Context(), userID)
		if err != nil {
			status := http.StatusForbidden
			if err == sql.ErrNoRows {
				status = http.StatusUnauthorized
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(status)
			_, _ = w.Write([]byte(`{"error":{"message":"无后台管理权限"}}`))
			return
		}
		if !strings.EqualFold(strings.TrimSpace(user.Email), strings.TrimSpace(adminEmail)) {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"error":{"message":"无后台管理权限"}}`))
			return
		}
		next(w, r)
	})
}
