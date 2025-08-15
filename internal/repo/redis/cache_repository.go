package redis

import (
	"context"
	"encoding/json"
	"time"

	"github.com/KarpovAlexandrGo/task-service/internal/entity"
	"github.com/redis/go-redis/v9"
)

type CacheRepository struct {
	client *redis.Client
}

func NewCacheRepository(addr, password string, db int) *CacheRepository {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	return &CacheRepository{client: client}
}

func (c *CacheRepository) SetTasks(ctx context.Context, tasks []entity.Task, ttl time.Duration) error {
	data, err := json.Marshal(tasks)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, "tasks", data, ttl).Err()
}

func (c *CacheRepository) GetTasks(ctx context.Context) ([]entity.Task, error) {
	data, err := c.client.Get(ctx, "tasks").Result()
	if err == redis.Nil {
		return nil, nil // Кэш пуст
	} else if err != nil {
		return nil, err
	}

	var tasks []entity.Task
	if err := json.Unmarshal([]byte(data), &tasks); err != nil {
		return nil, err
	}
	return tasks, nil
}

func (c *CacheRepository) Invalidate(ctx context.Context) error {
	return c.client.Del(ctx, "tasks").Err()
}

// Ping проверяет подключение к Redis
func (c *CacheRepository) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}
