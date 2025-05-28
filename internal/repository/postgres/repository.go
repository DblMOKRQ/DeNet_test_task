package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/DblMOKRQ/DeNet_test_task/internal/models"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// Repository представляет слой доступа к данным PostgreSQL
type Repository struct {
	db *sql.DB
}

// NewRepository создает новый экземпляр репозитория
func NewRepository(user string, password string, dbname string, sslmode string) (*Repository, error) {
	connStr := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=%s", user, password, dbname, sslmode)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	// Проверка соединения
	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &Repository{
		db: db,
	}, nil
}

// Close закрывает соединение с базой данных
func (r *Repository) Close() error {
	return r.db.Close()
}

// GetUserByID возвращает пользователя по ID
func (r *Repository) GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	query := `
		SELECT id, username, points, referrer_id, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	var user models.User
	var referrerID sql.NullString

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.Username,
		&user.Points,
		&referrerID,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if referrerID.Valid {
		refID, err := uuid.Parse(referrerID.String)
		if err == nil {
			user.ReferrerID = &refID
		}
	}

	return &user, nil
}

// GetLeaderboard возвращает список пользователей с наибольшим балансом
func (r *Repository) GetLeaderboard(ctx context.Context, limit int) ([]*models.User, error) {
	query := `
		SELECT id, username, points, referrer_id, created_at, updated_at
		FROM users
		ORDER BY points DESC
		LIMIT $1
	`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query leaderboard: %w", err)
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		var user models.User
		var referrerID sql.NullString

		err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.Points,
			&referrerID,
			&user.CreatedAt,
			&user.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}

		if referrerID.Valid {
			refID, err := uuid.Parse(referrerID.String)
			if err == nil {
				user.ReferrerID = &refID
			}
		}

		users = append(users, &user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return users, nil
}

// CompleteTask отмечает задание как выполненное и начисляет баллы
func (r *Repository) CompleteTask(ctx context.Context, userID uuid.UUID, taskRequest models.TaskRequest) (*models.Task, error) {
	// Начало транзакции
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Проверка существования пользователя
	var exists bool
	err = tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)", userID).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("failed to check user existence: %w", err)
	}

	if !exists {
		return nil, errors.New("user not found")
	}

	// Создание задания
	task := &models.Task{
		ID:          uuid.New(),
		UserID:      userID,
		TaskType:    taskRequest.TaskType,
		Points:      taskRequest.Points,
		CompletedAt: time.Now(),
	}

	// Вставка записи о выполненном задании
	_, err = tx.ExecContext(ctx,
		"INSERT INTO tasks (id, user_id, task_type, points, completed_at) VALUES ($1, $2, $3, $4, $5)",
		task.ID, task.UserID, task.TaskType, task.Points, task.CompletedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to insert task: %w", err)
	}

	// Обновление баланса пользователя
	_, err = tx.ExecContext(ctx,
		"UPDATE users SET points = points + $1, updated_at = NOW() WHERE id = $2",
		task.Points, task.UserID,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to update user points: %w", err)
	}

	// Фиксация транзакции
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return task, nil
}

// AddReferrer добавляет реферальный код
func (r *Repository) AddReferrer(ctx context.Context, userID, referrerID uuid.UUID) (*models.User, error) {
	// Начало транзакции
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Проверка существования реферера
	var exists bool
	err = tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)", referrerID).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("failed to check referrer existence: %w", err)
	}

	if !exists {
		return nil, errors.New("referrer not found")
	}

	// Проверка, что пользователь не имеет реферера
	var hasReferrer bool
	err = tx.QueryRowContext(ctx, "SELECT referrer_id IS NOT NULL FROM users WHERE id = $1", userID).Scan(&hasReferrer)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("failed to check user referrer: %w", err)
	}

	// Обновление реферального кода пользователя
	_, err = tx.ExecContext(ctx,
		"UPDATE users SET referrer_id = $1, updated_at = NOW() WHERE id = $2",
		referrerID, userID,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to update user referrer: %w", err)
	}

	// Начисление бонусных баллов рефереру
	bonusPoints := 10 // Бонус за реферала
	_, err = tx.ExecContext(ctx,
		"UPDATE users SET points = points + $1, updated_at = NOW() WHERE id = $2",
		bonusPoints, referrerID,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to update referrer points: %w", err)
	}

	// Получение обновленных данных пользователя
	var user models.User
	var refID sql.NullString

	err = tx.QueryRowContext(ctx,
		"SELECT id, username, points, referrer_id, created_at, updated_at FROM users WHERE id = $1",
		userID,
	).Scan(
		&user.ID,
		&user.Username,
		&user.Points,
		&refID,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get updated user: %w", err)
	}

	// Преобразование sql.NullString в *uuid.UUID
	if refID.Valid {
		parsedRefID, err := uuid.Parse(refID.String)
		if err == nil {
			user.ReferrerID = &parsedRefID
		}
	}

	// Фиксация транзакции
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &user, nil
}
