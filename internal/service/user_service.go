package service

import (
	"context"

	"github.com/DblMOKRQ/DeNet_test_task/internal/models"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// UserRepository интерфейс для доступа к данным пользователей
type UserRepository interface {
	GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetLeaderboard(ctx context.Context, limit int) ([]*models.User, error)
	CompleteTask(ctx context.Context, userID uuid.UUID, taskRequest models.TaskRequest) (*models.Task, error)
	AddReferrer(ctx context.Context, userID, referrerID uuid.UUID) (*models.User, error)
	LoginUser(ctx context.Context, username string, password string) (*models.User, error)
}

// UserService предоставляет методы для работы с пользователями
type UserService struct {
	repo UserRepository
	log  *zap.Logger
}

// NewUserService создает новый экземпляр UserService
func NewUserService(repo UserRepository, log *zap.Logger) *UserService {
	return &UserService{
		repo: repo,
		log:  log.Named("user_service"),
	}
}

// LoginUser регистрирует пользователя
func (s *UserService) LoginUser(context context.Context, username string, password string) (*models.User, error) {
	s.log.Info("Logging in user", zap.String("username", username))
	user, err := s.repo.LoginUser(context, username, password)
	if err != nil {
		s.log.Error("Failed to login user", zap.String("username", username), zap.Error(err))
		return nil, err
	}

	return user, nil
}

// GetUserByID возвращает пользователя по ID
func (s *UserService) GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	s.log.Info("Getting user by ID", zap.String("user_id", id.String()))

	user, err := s.repo.GetUserByID(ctx, id)
	if err != nil {
		s.log.Error("Failed to get user by ID",
			zap.String("user_id", id.String()),
			zap.Error(err))
		return nil, err
	}

	if user == nil {
		s.log.Warn("User not found", zap.String("user_id", id.String()))
		return nil, nil
	}

	s.log.Debug("User retrieved successfully",
		zap.String("user_id", id.String()),
		zap.String("username", user.Username),
		zap.Int("points", user.Points))
	return user, nil
}

// GetLeaderboard возвращает список пользователей с наибольшим балансом
func (s *UserService) GetLeaderboard(ctx context.Context, limit int) ([]*models.User, error) {
	s.log.Info("Getting leaderboard", zap.Int("limit", limit))

	users, err := s.repo.GetLeaderboard(ctx, limit)
	if err != nil {
		s.log.Error("Failed to get leaderboard",
			zap.Int("limit", limit),
			zap.Error(err))
		return nil, err
	}

	s.log.Debug("Leaderboard retrieved successfully",
		zap.Int("limit", limit),
		zap.Int("users_count", len(users)))
	return users, nil
}

// CompleteTask отмечает задание как выполненное и начисляет баллы
func (s *UserService) CompleteTask(ctx context.Context, userID uuid.UUID, taskRequest models.TaskRequest) (*models.Task, error) {
	s.log.Info("Completing task",
		zap.String("user_id", userID.String()),
		zap.String("task_type", taskRequest.TaskType),
		zap.Int("points", taskRequest.Points))

	task, err := s.repo.CompleteTask(ctx, userID, taskRequest)
	if err != nil {
		s.log.Error("Failed to complete task",
			zap.String("user_id", userID.String()),
			zap.String("task_type", taskRequest.TaskType),
			zap.Int("points", taskRequest.Points),
			zap.Error(err))
		return nil, err
	}

	s.log.Info("Task completed successfully",
		zap.String("user_id", userID.String()),
		zap.String("task_id", task.ID.String()),
		zap.String("task_type", task.TaskType),
		zap.Int("points", task.Points))
	return task, nil
}

// AddReferrer добавляет реферальный код
func (s *UserService) AddReferrer(ctx context.Context, userID, referrerID uuid.UUID) (*models.User, error) {
	s.log.Info("Adding referrer",
		zap.String("user_id", userID.String()),
		zap.String("referrer_id", referrerID.String()))

	user, err := s.repo.AddReferrer(ctx, userID, referrerID)
	if err != nil {
		s.log.Error("Failed to add referrer",
			zap.String("user_id", userID.String()),
			zap.String("referrer_id", referrerID.String()),
			zap.Error(err))
		return nil, err
	}

	s.log.Info("Referrer added successfully",
		zap.String("user_id", userID.String()),
		zap.String("referrer_id", referrerID.String()),
		zap.Int("user_points", user.Points))
	return user, nil
}
