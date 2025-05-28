package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/DblMOKRQ/DeNet_test_task/pkg/jwt"
	"go.uber.org/zap"
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
func JWTAuth(jwtService *jwt.Service, log *zap.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Debug("Checking JWT token",
				zap.String("path", r.URL.Path),
				zap.String("method", r.Method))

			// Получение заголовка Authorization
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				log.Warn("Missing Authorization header",
					zap.String("path", r.URL.Path),
					zap.String("remote_addr", r.RemoteAddr))
				http.Error(w, "Authorization header is required", http.StatusUnauthorized)
				return
			}

			// Валидация токена
			claims, err := jwtService.ValidateToken(authHeader)
			if err != nil {
				if err == jwt.ErrExpiredToken {
					log.Warn("Token expired",
						zap.String("path", r.URL.Path),
						zap.String("remote_addr", r.RemoteAddr))
					http.Error(w, "Token expired", http.StatusUnauthorized)
				} else {
					log.Warn("Invalid token",
						zap.String("path", r.URL.Path),
						zap.String("remote_addr", r.RemoteAddr),
						zap.Error(err))
					http.Error(w, "Invalid token", http.StatusUnauthorized)
				}
				return
			}

			log.Debug("JWT token validated successfully",
				zap.String("user_id", claims.UserID),
				zap.String("path", r.URL.Path))

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
func Logger(log *zap.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Создаем ResponseWriter, который отслеживает статус ответа
			rw := newResponseWriter(w)

			// Логируем входящий запрос
			log.Info("Request started",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.String("remote_addr", r.RemoteAddr),
				zap.String("user_agent", r.UserAgent()))

			// Обрабатываем запрос
			next.ServeHTTP(rw, r)

			// Логируем результат запроса
			duration := time.Since(start)
			log.Info("Request completed",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.Int("status", rw.status),
				zap.Duration("duration", duration),
				zap.Int("size", rw.size))
		})
	}
}

// Recover обрабатывает панику в обработчиках
func Recover(log *zap.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					log.Error("Panic recovered in HTTP handler",
						zap.Any("error", err),
						zap.String("path", r.URL.Path),
						zap.String("method", r.Method),
						zap.String("remote_addr", r.RemoteAddr))

					http.Error(w, "Internal server error", http.StatusInternalServerError)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

// responseWriter - обертка для http.ResponseWriter для отслеживания статуса и размера ответа
type responseWriter struct {
	http.ResponseWriter
	status int
	size   int
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{
		ResponseWriter: w,
		status:         http.StatusOK, // По умолчанию 200 OK
	}
}

func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(b)
	rw.size += size
	return size, err
}
