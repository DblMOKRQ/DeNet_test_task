package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/DblMOKRQ/DeNet_test_task/internal/models"
	"github.com/DblMOKRQ/DeNet_test_task/internal/service"
	"github.com/google/uuid"
)

// UserHandler обрабатывает запросы, связанные с пользователями
type UserHandler struct {
	userService *service.UserService
}

// NewUserHandler создает новый экземпляр UserHandler
func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

// GetUserStatus возвращает информацию о пользователе
func (h *UserHandler) GetUserStatus(w http.ResponseWriter, r *http.Request) {
	// Извлечение ID пользователя из URL
	path := strings.Split(r.URL.Path, "/")
	if len(path) < 4 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	userID, err := uuid.Parse(path[2])
	if err != nil {
		http.Error(w, "Invalid user ID format", http.StatusBadRequest)
		return
	}

	user, err := h.userService.GetUserByID(r.Context(), userID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get user: %v", err), http.StatusInternalServerError)
		return
	}

	if user == nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Сериализация ответа в JSON
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(user)
}

// GetLeaderboard возвращает список пользователей с наибольшим балансом
func (h *UserHandler) GetLeaderboard(w http.ResponseWriter, r *http.Request) {
	// Получение параметра limit из query string
	limitStr := r.URL.Query().Get("limit")
	limit := 10 // По умолчанию 10 пользователей

	if limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	users, err := h.userService.GetLeaderboard(r.Context(), limit)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get leaderboard: %v", err), http.StatusInternalServerError)
		return
	}

	// Сериализация ответа в JSON
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(users)
}

// CompleteTask отмечает задание как выполненное и начисляет баллы
func (h *UserHandler) CompleteTask(w http.ResponseWriter, r *http.Request) {
	// Извлечение ID пользователя из URL
	path := strings.Split(r.URL.Path, "/")
	if len(path) < 5 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	userID, err := uuid.Parse(path[2])
	if err != nil {
		http.Error(w, "Invalid user ID format", http.StatusBadRequest)
		return
	}

	// Десериализация запроса
	var taskRequest models.TaskRequest
	if err := json.NewDecoder(r.Body).Decode(&taskRequest); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Валидация запроса
	if taskRequest.TaskType == "" {
		http.Error(w, "Task type is required", http.StatusBadRequest)
		return
	}

	if taskRequest.Points <= 0 {
		http.Error(w, "Points must be positive", http.StatusBadRequest)
		return
	}

	task, err := h.userService.CompleteTask(r.Context(), userID, taskRequest)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to complete task: %v", err), http.StatusInternalServerError)
		return
	}

	// Сериализация ответа в JSON
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(task)
}

// AddReferrer добавляет реферальный код
func (h *UserHandler) AddReferrer(w http.ResponseWriter, r *http.Request) {
	// Извлечение ID пользователя из URL
	path := strings.Split(r.URL.Path, "/")
	if len(path) < 4 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	userID, err := uuid.Parse(path[2])
	if err != nil {
		http.Error(w, "Invalid user ID format", http.StatusBadRequest)
		return
	}

	// Десериализация запроса
	var referrerRequest models.ReferrerRequest
	if err := json.NewDecoder(r.Body).Decode(&referrerRequest); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Валидация запроса
	if referrerRequest.ReferrerID == "" {
		http.Error(w, "Referrer ID is required", http.StatusBadRequest)
		return
	}

	referrerID, err := uuid.Parse(referrerRequest.ReferrerID)
	if err != nil {
		http.Error(w, "Invalid referrer ID format", http.StatusBadRequest)
		return
	}

	user, err := h.userService.AddReferrer(r.Context(), userID, referrerID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to add referrer: %v", err), http.StatusInternalServerError)
		return
	}

	// Сериализация ответа в JSON
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(user)
}
