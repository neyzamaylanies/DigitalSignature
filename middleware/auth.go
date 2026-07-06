package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const UserIDKey contextKey = "user_id"
const UserRoleKey contextKey = "user_role"

func AuthMiddleware(secret string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "Invalid authorization format", http.StatusUnauthorized)
			return
		}

		tokenStr := parts[1]

		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrTokenSignatureInvalid
			}
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			http.Error(w, "Invalid token claims", http.StatusUnauthorized)
			return
		}

		userIDFloat, ok := claims["user_id"].(float64)
		if !ok {
			http.Error(w, "Invalid user ID in token", http.StatusUnauthorized)
			return
		}

		userRole, ok := claims["user_role"].(string)
		if !ok {
			http.Error(w, "Invalid user role in token", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), UserIDKey, int(userIDFloat))
		ctx = context.WithValue(ctx, UserRoleKey, userRole)

		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func AdminMiddleware(secret string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		AuthMiddleware(secret, func(w http.ResponseWriter, r *http.Request) {
			role, ok := r.Context().Value(UserRoleKey).(string)
			if !ok {
				http.Error(w, "Invalid role context", http.StatusUnauthorized)
				return
			}

			if role != "admin" {
				http.Error(w, "Forbidden: admin only", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})(w, r)
	}
}
