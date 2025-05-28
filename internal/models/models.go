package models

import (
	"time"

	"github.com/google/uuid"
)

type UserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// User представляет модель пользователя
type User struct {
	ID         uuid.UUID  `json:"id"`
	Username   string     `json:"username"`
	Password   string     `json:"password"`
	Points     int        `json:"points"`
	ReferrerID *uuid.UUID `json:"referrer_id,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// Task представляет модель задания
type Task struct {
	ID          uuid.UUID `json:"id"`
	UserID      uuid.UUID `json:"user_id"`
	TaskType    string    `json:"task_type"`
	Points      int       `json:"points"`
	CompletedAt time.Time `json:"completed_at"`
}

// TaskRequest представляет запрос на выполнение задания
type TaskRequest struct {
	TaskType string `json:"task_type"`
	Points   int    `json:"points"`
}

// ReferrerRequest представляет запрос на добавление реферального кода
type ReferrerRequest struct {
	ReferrerID string `json:"referrer_id"`
}

// ErrorResponse представляет ответ с ошибкой
type ErrorResponse struct {
	Error string `json:"error"`
}
