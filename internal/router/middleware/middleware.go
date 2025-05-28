package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/DblMOKRQ/DeNet_test_task/pkg/jwt"
)

// Middleware представляет функцию middleware
type Middleware func(http.Handler) http.Handler

// Chain объединяет несколько middleware в одну цепочку
func Chain(h http.Handler, middlewares ...Middleware) http.Handler {
	for _, m := range middlewares {
		h = m(h)
	}
	return h
}

// JWTAuth проверяет JWT токен в заголовке Authorization
func JWTAuth(jwtService *jwt.Service) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Получение заголовка Authorization
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Authorization header is required", http.StatusUnauthorized)
				return
			}

			// Проверка формата "Bearer <token>"
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, "Invalid authorization format", http.StatusUnauthorized)
				return
			}

			// Валидация токена
			claims, err := jwtService.ValidateToken(parts[1])
			if err != nil {
				if err == jwt.ErrExpiredToken {
					http.Error(w, "Token expired", http.StatusUnauthorized)
				} else {
					http.Error(w, "Invalid token", http.StatusUnauthorized)
				}
				return
			}

			// Сохранение данных пользователя в контексте
			ctx := context.WithValue(r.Context(), "userID", claims.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ContentTypeJSON устанавливает Content-Type: application/json
func ContentTypeJSON(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

// Logger логирует информацию о запросе
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// Здесь можно добавить логирование запроса
		next.ServeHTTP(w, r)
	})
}

// Recover обрабатывает панику в обработчиках
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
