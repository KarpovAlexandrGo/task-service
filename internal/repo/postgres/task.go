package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/KarpovAlexandrGo/task-service/internal/entity"
	"github.com/KarpovAlexandrGo/task-service/pkg/logger"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"
)

var (
	ErrTaskNotFound = errors.New("task not found")
	ErrInvalidUUID  = errors.New("invalid UUID format")
)

type TaskRepository struct {
	db     *pgxpool.Pool
	logger *logrus.Logger
}

type CacheRepository struct {
	db *pgxpool.Pool
}

func NewTaskRepository(db *pgxpool.Pool) *TaskRepository {
	return &TaskRepository{
		db:     db,
		logger: logger.Log,
	}
}

// Добавьте этот метод в CacheRepository
func (r *CacheRepository) GetTasks(ctx context.Context) ([]entity.Task, error) {
	// Реализация метода или временный заглушка
	return nil, nil
}

func (r *CacheRepository) SetTasks(ctx context.Context, tasks []entity.Task, ttl time.Duration) error {
	// Реализация метода или временный заглушка
	return nil
}

func (r *CacheRepository) Invalidate(ctx context.Context) error {
	// Реализация метода или временный заглушка
	return nil
}

func NewCacheRepository(db *pgxpool.Pool) *CacheRepository {
	return &CacheRepository{db: db}
}

func (r *TaskRepository) Create(ctx context.Context, task entity.Task) (entity.Task, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `
		INSERT INTO tasks (id, title, description, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, title, description, status, created_at, updated_at`

	now := time.Now()
	err := r.db.QueryRow(ctx, query,
		task.ID,
		task.Title,
		task.Description,
		task.Status,
		now,
		now,
	).Scan(&task.ID, &task.Title, &task.Description, &task.Status, &task.CreatedAt, &task.UpdatedAt)

	if err != nil {
		r.logger.WithFields(logrus.Fields{
			"method":  "Create",
			"task_id": task.ID.String(),
			"title":   task.Title,
		}).WithError(err).Error("Failed to create task")
		return entity.Task{}, fmt.Errorf("failed to create task: %w", err)
	}

	return task, nil
}

func (r *TaskRepository) Get(ctx context.Context, id string) (entity.Task, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	parsedID, err := uuid.Parse(id)
	if err != nil {
		r.logger.WithFields(logrus.Fields{
			"method":  "Get",
			"task_id": id,
		}).WithError(err).Warn("Invalid task ID format")
		return entity.Task{}, ErrInvalidUUID
	}

	query := `
		SELECT id, title, description, status, created_at, updated_at
		FROM tasks WHERE id = $1`

	var task entity.Task
	err = r.db.QueryRow(ctx, query, parsedID).Scan(
		&task.ID,
		&task.Title,
		&task.Description,
		&task.Status,
		&task.CreatedAt,
		&task.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			r.logger.WithFields(logrus.Fields{
				"method":  "Get",
				"task_id": id,
			}).Warn("Task not found")
			return entity.Task{}, ErrTaskNotFound
		}
		r.logger.WithFields(logrus.Fields{
			"method":  "Get",
			"task_id": id,
		}).WithError(err).Error("Failed to get task")
		return entity.Task{}, fmt.Errorf("failed to get task: %w", err)
	}

	return task, nil
}

func (r *TaskRepository) List(ctx context.Context) ([]entity.Task, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `
		SELECT id, title, description, status, created_at, updated_at
		FROM tasks`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		r.logger.WithFields(logrus.Fields{
			"method": "List",
		}).WithError(err).Error("Failed to list tasks")
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}
	defer rows.Close()

	var tasks []entity.Task
	for rows.Next() {
		var task entity.Task
		if err := rows.Scan(
			&task.ID,
			&task.Title,
			&task.Description,
			&task.Status,
			&task.CreatedAt,
			&task.UpdatedAt,
		); err != nil {
			r.logger.WithFields(logrus.Fields{
				"method": "List",
			}).WithError(err).Error("Failed to scan task row")
			return nil, fmt.Errorf("failed to scan task row: %w", err)
		}
		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		r.logger.WithFields(logrus.Fields{
			"method": "List",
		}).WithError(err).Error("Error after scanning rows")
		return nil, fmt.Errorf("error after scanning rows: %w", err)
	}

	return tasks, nil
}

func (r *TaskRepository) Update(ctx context.Context, task entity.Task) (entity.Task, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `
		UPDATE tasks
		SET title = $2, description = $3, status = $4, updated_at = $5
		WHERE id = $1
		RETURNING id, title, description, status, created_at, updated_at`

	err := r.db.QueryRow(ctx, query,
		task.ID,
		task.Title,
		task.Description,
		task.Status,
		time.Now(),
	).Scan(&task.ID, &task.Title, &task.Description, &task.Status, &task.CreatedAt, &task.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			r.logger.WithFields(logrus.Fields{
				"method":  "Update",
				"task_id": task.ID.String(),
			}).Warn("Task not found for update")
			return entity.Task{}, ErrTaskNotFound
		}
		r.logger.WithFields(logrus.Fields{
			"method":  "Update",
			"task_id": task.ID.String(),
			"title":   task.Title,
		}).WithError(err).Error("Failed to update task")
		return entity.Task{}, fmt.Errorf("failed to update task: %w", err)
	}

	return task, nil
}

func (r *TaskRepository) Delete(ctx context.Context, id string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	parsedID, err := uuid.Parse(id)
	if err != nil {
		r.logger.WithFields(logrus.Fields{
			"method":  "Delete",
			"task_id": id,
		}).WithError(err).Warn("Invalid task ID format")
		return ErrInvalidUUID
	}

	query := `DELETE FROM tasks WHERE id = $1`
	result, err := r.db.Exec(ctx, query, parsedID)
	if err != nil {
		r.logger.WithFields(logrus.Fields{
			"method":  "Delete",
			"task_id": id,
		}).WithError(err).Error("Failed to delete task")
		return fmt.Errorf("failed to delete task: %w", err)
	}

	if result.RowsAffected() == 0 {
		r.logger.WithFields(logrus.Fields{
			"method":  "Delete",
			"task_id": id,
		}).Warn("Task not found for deletion")
		return ErrTaskNotFound
	}

	return nil
}
