package redis

import (
	"context"
	"time"

	"github.com/KarpovAlexandrGo/task-service/internal/entity"
)

// Placeholder для CacheRepository (реализуем позже)
type CacheRepository struct{}

func (c *CacheRepository) SetTasks(ctx context.Context, tasks []entity.Task, ttl time.Duration) error {
	return nil
}

func (c *CacheRepository) GetTasks(ctx context.Context) ([]entity.Task, error) {
	return nil, nil
}

func (c *CacheRepository) Invalidate(ctx context.Context) error {
	return nil
}
