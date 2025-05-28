package service

import (
	"context"

	"github.com/DblMOKRQ/DeNet_test_task/internal/models"
	"github.com/google/uuid"
)

// UserRepository интерфейс для доступа к данным пользователей
type UserRepository interface {
	GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetLeaderboard(ctx context.Context, limit int) ([]*models.User, error)
	CompleteTask(ctx context.Context, userID uuid.UUID, taskRequest models.TaskRequest) (*models.Task, error)
	AddReferrer(ctx context.Context, userID, referrerID uuid.UUID) (*models.User, error)
}

// UserService предоставляет методы для работы с пользователями
type UserService struct {
	repo UserRepository
}

// NewUserService создает новый экземпляр UserService
func NewUserService(repo UserRepository) *UserService {
	return &UserService{
		repo: repo,
	}
}

// GetUserByID возвращает пользователя по ID
func (s *UserService) GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	return s.repo.GetUserByID(ctx, id)
}

// GetLeaderboard возвращает список пользователей с наибольшим балансом
func (s *UserService) GetLeaderboard(ctx context.Context, limit int) ([]*models.User, error) {
	return s.repo.GetLeaderboard(ctx, limit)
}

// CompleteTask отмечает задание как выполненное и начисляет баллы
func (s *UserService) CompleteTask(ctx context.Context, userID uuid.UUID, taskRequest models.TaskRequest) (*models.Task, error) {
	return s.repo.CompleteTask(ctx, userID, taskRequest)
}

// AddReferrer добавляет реферальный код
func (s *UserService) AddReferrer(ctx context.Context, userID, referrerID uuid.UUID) (*models.User, error) {
	return s.repo.AddReferrer(ctx, userID, referrerID)
}
