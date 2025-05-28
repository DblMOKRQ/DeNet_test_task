package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/DblMOKRQ/DeNet_test_task/internal/models"
	"github.com/DblMOKRQ/DeNet_test_task/internal/service"
	"github.com/DblMOKRQ/DeNet_test_task/pkg/jwt"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// UserHandler обрабатывает запросы, связанные с пользователями
type UserHandler struct {
	userService *service.UserService
	jwtService  *jwt.Service
	log         *zap.Logger
}

// NewUserHandler создает новый экземпляр UserHandler
func NewUserHandler(userService *service.UserService, jwtService *jwt.Service, log *zap.Logger) *UserHandler {
	return &UserHandler{
		userService: userService,
		jwtService:  jwtService,
		log:         log.Named("user_handler"),
	}
}

// LoginUser регистрирует нового пользователя и возвращает JWT токен
func (h *UserHandler) LoginUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.log.Warn("Invalid request method", zap.String("path", r.URL.Path), zap.String("method", r.Method))
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}
	h.log.Info("Handling register user request", zap.String("path", r.URL.Path), zap.String("method", r.Method))

	// Извлечение данных из запроса
	var userReq models.UserRequest
	if err := json.NewDecoder(r.Body).Decode(&userReq); err != nil {
		h.log.Warn("Invalid request body", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Валидация данных
	if userReq.Username == "" || userReq.Password == "" {
		h.log.Warn("Username and password are required")
		http.Error(w, "Username and password are required", http.StatusBadRequest)
		return
	}

	// Регистрация пользователя
	user, err := h.userService.LoginUser(r.Context(), userReq.Username, userReq.Password)
	if err != nil {
		h.log.Error("Failed to register user",
			zap.String("username", userReq.Username),
			zap.Error(err))
		http.Error(w, fmt.Sprintf("Failed to register user: %v", err), http.StatusInternalServerError)
		return
	}

	// Генерация JWT токена
	token, err := h.jwtService.GenerateToken(user.ID.String())
	if err != nil {
		h.log.Error("Failed to generate token",
			zap.String("user_id", user.ID.String()),
			zap.Error(err))
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	// Установка токена в заголовок
	w.Header().Set("Authorization", token)

	// Сериализация ответа в JSON
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	response := map[string]interface{}{
		"user":  user,
		"token": token,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.log.Error("Failed to encode response", zap.Error(err))
		return
	}

	h.log.Info("Successfully registered user",
		zap.String("user_id", user.ID.String()),
		zap.String("username", user.Username))
}

// GetUserStatus возвращает информацию о пользователе
func (h *UserHandler) GetUserStatus(w http.ResponseWriter, r *http.Request) {
	h.log.Info("Handling get user status request", zap.String("path", r.URL.Path), zap.String("method", r.Method))

	// Извлечение ID пользователя из токена
	claims, err := h.jwtService.ValidateToken(r.Header.Get("Authorization"))
	if err != nil {
		h.log.Warn("Invalid token", zap.Error(err))
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	userIDStr := claims.UserID
	h.log.Debug("Extracted user ID from URL", zap.String("user_id", userIDStr))

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		h.log.Warn("Invalid user ID format", zap.String("user_id", userIDStr), zap.Error(err))
		http.Error(w, "Invalid user ID format", http.StatusBadRequest)
		return
	}

	h.log.Debug("Getting user by ID", zap.String("user_id", userID.String()))
	user, err := h.userService.GetUserByID(r.Context(), userID)
	if err != nil {
		h.log.Error("Failed to get user",
			zap.String("user_id", userID.String()),
			zap.Error(err))
		http.Error(w, fmt.Sprintf("Failed to get user: %v", err), http.StatusInternalServerError)
		return
	}

	if user == nil {
		h.log.Warn("User not found", zap.String("user_id", userID.String()))
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Сериализация ответа в JSON
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(user); err != nil {
		h.log.Error("Failed to encode response", zap.Error(err))
		return
	}

	h.log.Info("Successfully returned user status", zap.String("user_id", userID.String()))
}

// GetLeaderboard возвращает список пользователей с наибольшим балансом
func (h *UserHandler) GetLeaderboard(w http.ResponseWriter, r *http.Request) {
	h.log.Info("Handling get leaderboard request", zap.String("path", r.URL.Path), zap.String("method", r.Method))

	// Получение параметра limit из query string
	limitStr := r.URL.Query().Get("limit")
	limit := 10 // По умолчанию 10 пользователей

	if limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err == nil && parsedLimit > 0 {
			limit = parsedLimit
		} else if err != nil {
			h.log.Warn("Invalid limit parameter", zap.String("limit", limitStr), zap.Error(err))
		}
	}

	h.log.Debug("Getting leaderboard", zap.Int("limit", limit))
	users, err := h.userService.GetLeaderboard(r.Context(), limit)
	if err != nil {
		h.log.Error("Failed to get leaderboard", zap.Int("limit", limit), zap.Error(err))
		http.Error(w, fmt.Sprintf("Failed to get leaderboard: %v", err), http.StatusInternalServerError)
		return
	}

	// Сериализация ответа в JSON
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(users); err != nil {
		h.log.Error("Failed to encode response", zap.Error(err))
		return
	}

	h.log.Info("Successfully returned leaderboard", zap.Int("users_count", len(users)))
}

// CompleteTask отмечает задание как выполненное и начисляет баллы
func (h *UserHandler) CompleteTask(w http.ResponseWriter, r *http.Request) {
	h.log.Info("Handling complete task request", zap.String("path", r.URL.Path), zap.String("method", r.Method))

	// Извлечение ID пользователя из токена
	claims, err := h.jwtService.ValidateToken(r.Header.Get("Authorization"))
	if err != nil {
		h.log.Warn("Invalid token", zap.Error(err))
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	userIDStr := claims.UserID
	h.log.Debug("Extracted user ID from URL", zap.String("user_id", userIDStr))

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		h.log.Warn("Invalid user ID format", zap.String("user_id", userIDStr), zap.Error(err))
		http.Error(w, "Invalid user ID format", http.StatusBadRequest)
		return
	}

	// Десериализация запроса
	var taskRequest models.TaskRequest
	if err := json.NewDecoder(r.Body).Decode(&taskRequest); err != nil {
		h.log.Warn("Invalid request body", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	h.log.Debug("Received task request",
		zap.String("user_id", userID.String()),
		zap.String("task_type", taskRequest.TaskType),
		zap.Int("points", taskRequest.Points))

	// Валидация запроса
	if taskRequest.TaskType == "" {
		h.log.Warn("Task type is required", zap.String("user_id", userID.String()))
		http.Error(w, "Task type is required", http.StatusBadRequest)
		return
	}

	if taskRequest.Points <= 0 {
		h.log.Warn("Points must be positive",
			zap.String("user_id", userID.String()),
			zap.Int("points", taskRequest.Points))
		http.Error(w, "Points must be positive", http.StatusBadRequest)
		return
	}

	task, err := h.userService.CompleteTask(r.Context(), userID, taskRequest)
	if err != nil {
		h.log.Error("Failed to complete task",
			zap.String("user_id", userID.String()),
			zap.String("task_type", taskRequest.TaskType),
			zap.Int("points", taskRequest.Points),
			zap.Error(err))
		http.Error(w, fmt.Sprintf("Failed to complete task: %v", err), http.StatusInternalServerError)
		return
	}

	// Сериализация ответа в JSON
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(task); err != nil {
		h.log.Error("Failed to encode response", zap.Error(err))
		return
	}

	h.log.Info("Successfully completed task",
		zap.String("user_id", userID.String()),
		zap.String("task_id", task.ID.String()),
		zap.String("task_type", task.TaskType),
		zap.Int("points", task.Points))
}

// AddReferrer добавляет реферальный код
func (h *UserHandler) AddReferrer(w http.ResponseWriter, r *http.Request) {
	h.log.Info("Handling add referrer request", zap.String("path", r.URL.Path), zap.String("method", r.Method))

	// Извлечение ID пользователя из токена
	claims, err := h.jwtService.ValidateToken(r.Header.Get("Authorization"))
	if err != nil {
		h.log.Warn("Invalid token", zap.Error(err))
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	userIDStr := claims.UserID
	h.log.Debug("Extracted user ID from URL", zap.String("user_id", userIDStr))

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		h.log.Warn("Invalid user ID format", zap.String("user_id", userIDStr), zap.Error(err))
		http.Error(w, "Invalid user ID format", http.StatusBadRequest)
		return
	}

	// Десериализация запроса
	var referrerRequest models.ReferrerRequest
	if err := json.NewDecoder(r.Body).Decode(&referrerRequest); err != nil {
		h.log.Warn("Invalid request body", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	h.log.Debug("Received referrer request",
		zap.String("user_id", userID.String()),
		zap.String("referrer_id", referrerRequest.ReferrerID))

	// Валидация запроса
	if referrerRequest.ReferrerID == "" {
		h.log.Warn("Referrer ID is required", zap.String("user_id", userID.String()))
		http.Error(w, "Referrer ID is required", http.StatusBadRequest)
		return
	}

	referrerID, err := uuid.Parse(referrerRequest.ReferrerID)
	if err != nil {
		h.log.Warn("Invalid referrer ID format",
			zap.String("user_id", userID.String()),
			zap.String("referrer_id", referrerRequest.ReferrerID),
			zap.Error(err))
		http.Error(w, "Invalid referrer ID format", http.StatusBadRequest)
		return
	}

	// Проверка, что пользователь не добавляет сам себя как реферера
	if userID == referrerID {
		h.log.Warn("User cannot add themselves as referrer",
			zap.String("user_id", userID.String()),
			zap.String("referrer_id", referrerID.String()))
		http.Error(w, "User cannot add themselves as referrer", http.StatusBadRequest)
		return
	}

	user, err := h.userService.AddReferrer(r.Context(), userID, referrerID)
	if err != nil {
		h.log.Error("Failed to add referrer",
			zap.String("user_id", userID.String()),
			zap.String("referrer_id", referrerID.String()),
			zap.Error(err))
		http.Error(w, fmt.Sprintf("Failed to add referrer: %v", err), http.StatusInternalServerError)
		return
	}

	// Сериализация ответа в JSON
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(user); err != nil {
		h.log.Error("Failed to encode response", zap.Error(err))
		return
	}

	h.log.Info("Successfully added referrer",
		zap.String("user_id", userID.String()),
		zap.String("referrer_id", referrerID.String()))
}
