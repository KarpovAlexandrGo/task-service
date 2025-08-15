package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/KarpovAlexandrGo/task-service/internal/entity"
	"github.com/KarpovAlexandrGo/task-service/pkg/logger"
	"github.com/google/uuid"
)

var (
	ErrTaskNotFound = errors.New("task not found")
)

type Logger interface {
	Error(msg string, fields map[string]interface{})
	Warn(msg string, fields map[string]interface{})
}

type TaskUseCase interface {
	Create(ctx context.Context, task entity.Task) (entity.Task, error)
	Get(ctx context.Context, id string) (entity.Task, error)
	List(ctx context.Context, page, limit int) ([]entity.Task, error)
	Update(ctx context.Context, task entity.Task) (entity.Task, error)
	Delete(ctx context.Context, id string) error
}

type TaskUseCaseImpl struct {
	taskRepo  TaskRepository
	cacheRepo CacheRepository
}

func NewTaskUseCase(taskRepo TaskRepository, cacheRepo CacheRepository) *TaskUseCaseImpl {
	return &TaskUseCaseImpl{
		taskRepo:  taskRepo,
		cacheRepo: cacheRepo,
	}
}

func (uc *TaskUseCaseImpl) Create(ctx context.Context, task entity.Task) (entity.Task, error) {
	logger.Log.Info("Starting task creation", "title", task.Title)

	if err := task.Validate(); err != nil {
		logger.Log.WithError(err).Error("Task validation failed")
		return entity.Task{}, err
	}

	if task.ID == uuid.Nil {
		task.ID = uuid.New()
	}
	task.CreatedAt = time.Now()
	task.UpdatedAt = task.CreatedAt

	createdTask, err := uc.taskRepo.Create(ctx, task)
	if err != nil {
		logger.Log.WithError(err).Error("Failed to create task")
		return entity.Task{}, err
	}

	if err := uc.cacheRepo.Invalidate(ctx); err != nil {
		logger.Log.WithError(err).Error("Failed to invalidate cache")
	}

	logger.Log.Info("Task created successfully", "task_id", createdTask.ID)
	return createdTask, nil
}

func (uc *TaskUseCaseImpl) Get(ctx context.Context, id string) (entity.Task, error) {
	logger.Log.Info("Getting task", "id", id)
	task, err := uc.taskRepo.Get(ctx, id)
	if err != nil {
		logger.Log.WithError(err).Error("Failed to get task from repository")
		return entity.Task{}, err
	}
	return task, nil
}

func (uc *TaskUseCaseImpl) List(ctx context.Context, page, limit int) ([]entity.Task, error) {
	logger.Log.Info("Listing tasks")

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	tasks, err := uc.cacheRepo.GetTasks(ctx)
	if err == nil {
		logger.Log.Info("Tasks retrieved from cache")
		return tasks[limit*(page-1) : limit*page], nil // Простая пагинация
	}

	logger.Log.Info("Cache miss, retrieving from repository")

	tasks, err = uc.taskRepo.List(ctx)
	if err != nil {
		logger.Log.WithError(err).Error("Failed to list tasks from repository")
		return nil, err
	}

	if err := uc.cacheRepo.SetTasks(ctx, tasks, 5*time.Minute); err != nil {
		logger.Log.WithError(err).Error("Failed to set tasks in cache")
	}

	logger.Log.Info("Tasks listed successfully", "count", len(tasks))
	return tasks[limit*(page-1) : limit*page], nil // Простая пагинация
}

func (uc *TaskUseCaseImpl) Update(ctx context.Context, task entity.Task) (entity.Task, error) {
	logger.Log.Info("Starting task update", "id", task.ID.String())

	if err := task.Validate(); err != nil {
		logger.Log.WithError(err).Error("Validation failed during task update")
		return entity.Task{}, err
	}

	task.UpdatedAt = time.Now()
	updatedTask, err := uc.taskRepo.Update(ctx, task)
	if err != nil {
		logger.Log.WithError(err).Error("Failed to update task in repository")
		return entity.Task{}, err
	}

	if err := uc.cacheRepo.Invalidate(ctx); err != nil {
		logger.Log.WithError(err).Error("Failed to invalidate cache after task update")
	}

	logger.Log.Info("Task updated successfully", "id", updatedTask.ID.String())
	return updatedTask, nil
}

func (uc *TaskUseCaseImpl) Delete(ctx context.Context, id string) error {
	logger.Log.Info("Deleting task", "id", id)

	if err := uc.taskRepo.Delete(ctx, id); err != nil {
		logger.Log.WithError(err).Error("Failed to delete task from repository")
		return err
	}

	if err := uc.cacheRepo.Invalidate(ctx); err != nil {
		logger.Log.WithError(err).Error("Failed to invalidate cache after task deletion")
	}

	logger.Log.Info("Task deleted successfully", "id", id)
	return nil
}

type TaskRepository interface {
	Create(ctx context.Context, task entity.Task) (entity.Task, error)
	Get(ctx context.Context, id string) (entity.Task, error)
	List(ctx context.Context) ([]entity.Task, error)
	Update(ctx context.Context, task entity.Task) (entity.Task, error)
	Delete(ctx context.Context, id string) error
}

type CacheRepository interface {
	SetTasks(ctx context.Context, tasks []entity.Task, ttl time.Duration) error
	GetTasks(ctx context.Context) ([]entity.Task, error)
	Invalidate(ctx context.Context) error
}
