package entity

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Task struct {
	ID          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (t *Task) Validate() error {
	if t.Title == "" {
		return fmt.Errorf("title cannot be empty")
	}
	validStatuses := []string{"todo", "in_progress", "done"}
	isValid := false
	for _, s := range validStatuses {
		if t.Status == s {
			isValid = true
			break
		}
	}
	if !isValid {
		return fmt.Errorf("status must be one of: todo, in_progress, done")
	}
	return nil
}
