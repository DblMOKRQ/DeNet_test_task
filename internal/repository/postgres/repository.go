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
	"go.uber.org/zap"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// Repository представляет слой доступа к данным PostgreSQL
type Repository struct {
	db  *sql.DB
	log *zap.Logger
}

// NewRepository создает новый экземпляр репозитория
func NewRepository(user string, password string, host string, port string, dbname string, sslmode string, log *zap.Logger) (*Repository, error) {
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", user, password, host, port, dbname, sslmode)

	log.Info("Connecting to PostgreSQL database",
		zap.String("dbname", dbname),
		zap.String("user", user),
		zap.String("sslmode", sslmode))

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Error("Failed to open database connection", zap.Error(err))
		return nil, err
	}

	// Проверка соединения
	log.Debug("Testing database connection")
	if err := db.Ping(); err != nil {
		log.Error("Failed to ping database", zap.Error(err))
		return nil, err
	}

	log.Info("Successfully connected to database")

	log.Info("Starting database migrations")

	if err := migrations(connStr); err != nil {
		log.Error("Failed to run database migrations", zap.Error(err))
		return nil, err
	}

	return &Repository{
		db:  db,
		log: log.Named("postgres_repository"),
	}, nil
}

func migrations(connStr string) error {

	m, err := migrate.New("file://../migrations", connStr)

	if err != nil {
		return fmt.Errorf("start migrations error %v", err)
	}

	if err := m.Up(); err != nil {
		if err == migrate.ErrNoChange {
			return nil
		}

		return fmt.Errorf("migration up error: %v", err)
	}

	return nil

}

// Close закрывает соединение с базой данных
func (r *Repository) Close() error {
	r.log.Info("Closing database connection")
	return r.db.Close()
}

// LoginUser регистрирует пользователя
func (r *Repository) LoginUser(ctx context.Context, username string, password string) (*models.User, error) {
	query := `
		INSERT INTO users (username, passw)
		VALUES ($1, $2)
	`
	var user models.User
	_, err := r.db.ExecContext(ctx, query, username, password)

	if err != nil {
		r.log.Error("Failed to register user", zap.Error(err))
		return nil, fmt.Errorf("failed to register user: %w", err)
	}
	res := r.db.QueryRowContext(ctx, "SELECT id, username, passw, created_at, updated_at FROM users WHERE username = $1", username)
	err = res.Scan(&user.ID, &user.Username, &user.Password, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		r.log.Error("Failed to register user", zap.Error(err))
		return nil, fmt.Errorf("failed to register user: %w", err)
	}

	return &user, nil
}

// GetUserByID возвращает пользователя по ID
func (r *Repository) GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	r.log.Debug("Getting user by ID", zap.String("user_id", id.String()))

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
		&user.Password,
		&user.Points,
		&referrerID,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.log.Warn("User not found", zap.String("user_id", id.String()))
			return nil, nil
		}
		r.log.Error("Failed to get user",
			zap.String("user_id", id.String()),
			zap.Error(err))
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if referrerID.Valid {
		refID, err := uuid.Parse(referrerID.String)
		if err == nil {
			user.ReferrerID = &refID
			r.log.Debug("User has referrer",
				zap.String("user_id", id.String()),
				zap.String("referrer_id", refID.String()))
		} else {
			r.log.Warn("Invalid referrer ID format",
				zap.String("user_id", id.String()),
				zap.String("raw_referrer_id", referrerID.String),
				zap.Error(err))
		}
	}

	r.log.Debug("User retrieved successfully",
		zap.String("user_id", id.String()),
		zap.String("username", user.Username))
	return &user, nil
}

// GetLeaderboard возвращает список пользователей с наибольшим балансом
func (r *Repository) GetLeaderboard(ctx context.Context, limit int) ([]*models.User, error) {
	r.log.Debug("Getting leaderboard", zap.Int("limit", limit))

	query := `
		SELECT id, username, points, referrer_id, created_at, updated_at
		FROM users
		ORDER BY points DESC
		LIMIT $1
	`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		r.log.Error("Failed to query leaderboard",
			zap.Int("limit", limit),
			zap.Error(err))
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
			r.log.Error("Failed to scan user", zap.Error(err))
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}

		if referrerID.Valid {
			refID, err := uuid.Parse(referrerID.String)
			if err == nil {
				user.ReferrerID = &refID
			} else {
				r.log.Warn("Invalid referrer ID format",
					zap.String("user_id", user.ID.String()),
					zap.String("raw_referrer_id", referrerID.String),
					zap.Error(err))
			}
		}

		users = append(users, &user)
	}

	if err := rows.Err(); err != nil {
		r.log.Error("Error iterating rows", zap.Error(err))
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	r.log.Debug("Leaderboard retrieved successfully",
		zap.Int("limit", limit),
		zap.Int("users_count", len(users)))
	return users, nil
}

// CompleteTask отмечает задание как выполненное и начисляет баллы
func (r *Repository) CompleteTask(ctx context.Context, userID uuid.UUID, taskRequest models.TaskRequest) (*models.Task, error) {
	r.log.Info("Completing task",
		zap.String("user_id", userID.String()),
		zap.String("task_type", taskRequest.TaskType),
		zap.Int("points", taskRequest.Points))

	// Начало транзакции
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		r.log.Error("Failed to begin transaction", zap.Error(err))
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Проверка существования пользователя
	var exists bool
	r.log.Debug("Checking user existence", zap.String("user_id", userID.String()))

	err = tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)", userID).Scan(&exists)
	if err != nil {
		r.log.Error("Failed to check user existence",
			zap.String("user_id", userID.String()),
			zap.Error(err))
		return nil, fmt.Errorf("failed to check user existence: %w", err)
	}

	if !exists {
		r.log.Warn("User not found", zap.String("user_id", userID.String()))
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
	r.log.Debug("Inserting task record",
		zap.String("task_id", task.ID.String()),
		zap.String("user_id", userID.String()))

	_, err = tx.ExecContext(ctx,
		"INSERT INTO tasks (id, user_id, task_type, points, completed_at) VALUES ($1, $2, $3, $4, $5)",
		task.ID, task.UserID, task.TaskType, task.Points, task.CompletedAt,
	)
	if err != nil {
		r.log.Error("Failed to insert task",
			zap.String("user_id", userID.String()),
			zap.String("task_type", taskRequest.TaskType),
			zap.Error(err))
		return nil, fmt.Errorf("failed to insert task: %w", err)
	}

	// Обновление баланса пользователя
	r.log.Debug("Updating user points",
		zap.String("user_id", userID.String()),
		zap.Int("points_to_add", task.Points))

	_, err = tx.ExecContext(ctx,
		"UPDATE users SET points = points + $1, updated_at = NOW() WHERE id = $2",
		task.Points, task.UserID,
	)
	if err != nil {
		r.log.Error("Failed to update user points",
			zap.String("user_id", userID.String()),
			zap.Int("points_to_add", task.Points),
			zap.Error(err))
		return nil, fmt.Errorf("failed to update user points: %w", err)
	}

	// Фиксация транзакции
	if err = tx.Commit(); err != nil {
		r.log.Error("Failed to commit transaction", zap.Error(err))
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	r.log.Info("Task completed successfully",
		zap.String("task_id", task.ID.String()),
		zap.String("user_id", userID.String()),
		zap.Int("points", task.Points))
	return task, nil
}

// AddReferrer добавляет реферальный код
func (r *Repository) AddReferrer(ctx context.Context, userID, referrerID uuid.UUID) (*models.User, error) {
	r.log.Info("Adding referrer",
		zap.String("user_id", userID.String()),
		zap.String("referrer_id", referrerID.String()))

	// Начало транзакции
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		r.log.Error("Failed to begin transaction", zap.Error(err))
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Проверка существования реферера
	var exists bool
	r.log.Debug("Checking referrer existence", zap.String("referrer_id", referrerID.String()))

	err = tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)", referrerID).Scan(&exists)
	if err != nil {
		r.log.Error("Failed to check referrer existence",
			zap.String("referrer_id", referrerID.String()),
			zap.Error(err))
		return nil, fmt.Errorf("failed to check referrer existence: %w", err)
	}

	if !exists {
		r.log.Warn("Referrer not found", zap.String("referrer_id", referrerID.String()))
		return nil, errors.New("referrer not found")
	}

	// Проверка, что пользователь не имеет реферера
	var hasReferrer bool
	r.log.Debug("Checking if user already has referrer", zap.String("user_id", userID.String()))

	err = tx.QueryRowContext(ctx, "SELECT referrer_id IS NOT NULL FROM users WHERE id = $1", userID).Scan(&hasReferrer)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.log.Warn("User not found", zap.String("user_id", userID.String()))
			return nil, errors.New("user not found")
		}
		r.log.Error("Failed to check user referrer",
			zap.String("user_id", userID.String()),
			zap.Error(err))
		return nil, fmt.Errorf("failed to check user referrer: %w", err)
	}

	if hasReferrer {
		r.log.Warn("User already has a referrer", zap.String("user_id", userID.String()))
		return nil, errors.New("user already has a referrer")
	}

	// Обновление реферального кода пользователя
	r.log.Debug("Updating user referrer",
		zap.String("user_id", userID.String()),
		zap.String("referrer_id", referrerID.String()))

	_, err = tx.ExecContext(ctx,
		"UPDATE users SET referrer_id = $1, updated_at = NOW() WHERE id = $2",
		referrerID, userID,
	)
	if err != nil {
		r.log.Error("Failed to update user referrer",
			zap.String("user_id", userID.String()),
			zap.String("referrer_id", referrerID.String()),
			zap.Error(err))
		return nil, fmt.Errorf("failed to update user referrer: %w", err)
	}

	// Начисление бонусных баллов рефереру
	bonusPoints := 10 // Бонус за реферала
	r.log.Debug("Adding bonus points to referrer",
		zap.String("referrer_id", referrerID.String()),
		zap.Int("bonus_points", bonusPoints))

	_, err = tx.ExecContext(ctx,
		"UPDATE users SET points = points + $1, updated_at = NOW() WHERE id = $2",
		bonusPoints, referrerID,
	)
	if err != nil {
		r.log.Error("Failed to update referrer points",
			zap.String("referrer_id", referrerID.String()),
			zap.Int("bonus_points", bonusPoints),
			zap.Error(err))
		return nil, fmt.Errorf("failed to update referrer points: %w", err)
	}

	// Получение обновленных данных пользователя
	var user models.User
	var refID sql.NullString

	r.log.Debug("Getting updated user data", zap.String("user_id", userID.String()))

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
		r.log.Error("Failed to get updated user",
			zap.String("user_id", userID.String()),
			zap.Error(err))
		return nil, fmt.Errorf("failed to get updated user: %w", err)
	}

	// Преобразование sql.NullString в *uuid.UUID
	if refID.Valid {
		parsedRefID, err := uuid.Parse(refID.String)
		if err == nil {
			user.ReferrerID = &parsedRefID
		} else {
			r.log.Warn("Invalid referrer ID format",
				zap.String("user_id", userID.String()),
				zap.String("raw_referrer_id", refID.String),
				zap.Error(err))
		}
	}

	// Фиксация транзакции
	if err = tx.Commit(); err != nil {
		r.log.Error("Failed to commit transaction", zap.Error(err))
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	r.log.Info("Referrer added successfully",
		zap.String("user_id", userID.String()),
		zap.String("referrer_id", referrerID.String()))
	return &user, nil
}
