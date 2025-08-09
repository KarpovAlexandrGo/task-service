package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/KarpovAlexandrGo/task-service/internal/entity"
	"github.com/KarpovAlexandrGo/task-service/internal/usecase"
	"github.com/KarpovAlexandrGo/task-service/pkg/logger"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// TaskHandler обрабатывает HTTP-запросы для работы с задачами.
type TaskHandler struct {
	taskUseCase usecase.TaskUseCase
}

// NewTaskHandler создает новый экземпляр TaskHandler.
func NewTaskHandler(taskUseCase usecase.TaskUseCase) *TaskHandler {
	return &TaskHandler{
		taskUseCase: taskUseCase,
	}
}

// RegisterRoutes регистрирует маршруты для обработки задач.
func (h *TaskHandler) RegisterRoutes(r chi.Router) {
	r.Route("/v1/tasks", func(r chi.Router) {
		r.Post("/", h.CreateTask)
		r.Get("/", h.ListTasks)
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", h.GetTask)
			r.Put("/", h.UpdateTask)
			r.Delete("/", h.DeleteTask)
		})
	})
}

// CreateTask обрабатывает создание новой задачи.
// @Summary      Создать задачу
// @Description  Создает новую задачу в системе
// @Tags         tasks
// @Accept       json
// @Produce      json
// @Param        task body     entity.Task true "Данные задачи"
// @Success      201  {object} entity.Task
// @Failure      400  {string} string "Неверный формат данных"
// @Failure      422  {string} string "Ошибка валидации"
// @Failure      500  {string} string "Внутренняя ошибка сервера"
// @Router       /v1/tasks [post]
func (h *TaskHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
	var task entity.Task
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		logger.Log.Error("Failed to decode request body", "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := validateTask(&task); err != nil {
		logger.Log.Warn("Task validation failed", "error", err)
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	createdTask, err := h.taskUseCase.Create(r.Context(), task)
	if err != nil {
		logger.Log.Error("Failed to create task", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(createdTask)
}

// GetTask обрабатывает получение задачи по ID.
// @Summary      Получить задачу
// @Description  Возвращает задачу по её ID
// @Tags         tasks
// @Accept       json
// @Produce      json
// @Param        id   path     string true "ID задачи"
// @Success      200  {object} entity.Task
// @Failure      400  {string} string "Неверный формат ID"
// @Failure      404  {string} string "Задача не найдена"
// @Router       /v1/tasks/{id} [get]
func (h *TaskHandler) GetTask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if _, err := uuid.Parse(id); err != nil {
		logger.Log.Warn("Invalid task ID format", "id", id, "error", err)
		http.Error(w, "Invalid task ID format", http.StatusBadRequest)
		return
	}

	task, err := h.taskUseCase.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, usecase.ErrTaskNotFound) {
			logger.Log.Warn("Task not found", "id", id)
			http.Error(w, "Task not found", http.StatusNotFound)
		} else {
			logger.Log.Error("Failed to get task", "id", id, "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

// ListTasks обрабатывает получение списка задач.
// @Summary      Список задач
// @Description  Возвращает список задач с пагинацией
// @Tags         tasks
// @Accept       json
// @Produce      json
// @Param        page   query    int false "Номер страницы" default(1)
// @Param        limit  query    int false "Количество элементов на странице" default(20)
// @Success      200    {array}  entity.Task
// @Failure      500    {string} string "Внутренняя ошибка сервера"
// @Router       /v1/tasks [get]
func (h *TaskHandler) ListTasks(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	tasks, err := h.taskUseCase.List(r.Context(), page, limit)
	if err != nil {
		logger.Log.Error("Failed to list tasks", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}

// UpdateTask обрабатывает обновление задачи.
// @Summary      Обновить задачу
// @Description  Обновляет существующую задачу
// @Tags         tasks
// @Accept       json
// @Produce      json
// @Param        id   path     string true "ID задачи"
// @Param        task body     entity.Task true "Обновленные данные задачи"
// @Success      200  {object} entity.Task
// @Failure      400  {string} string "Неверный формат ID или данных"
// @Failure      404  {string} string "Задача не найдена"
// @Failure      422  {string} string "Ошибка валидации"
// @Router       /v1/tasks/{id} [put]
func (h *TaskHandler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if _, err := uuid.Parse(id); err != nil {
		logger.Log.Warn("Invalid task ID format", "id", id, "error", err)
		http.Error(w, "Invalid task ID format", http.StatusBadRequest)
		return
	}

	var task entity.Task
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		logger.Log.Error("Failed to decode request body", "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	task.ID, _ = uuid.Parse(id) // Устанавливаем ID из пути

	if err := validateTask(&task); err != nil {
		logger.Log.Warn("Task validation failed", "error", err)
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	updatedTask, err := h.taskUseCase.Update(r.Context(), task)
	if err != nil {
		if errors.Is(err, usecase.ErrTaskNotFound) {
			logger.Log.Warn("Task not found for update", "id", id)
			http.Error(w, "Task not found", http.StatusNotFound)
		} else {
			logger.Log.Error("Failed to update task", "id", id, "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedTask)
}

// DeleteTask обрабатывает удаление задачи.
// @Summary      Удалить задачу
// @Description  Удаляет задачу по её ID
// @Tags         tasks
// @Accept       json
// @Produce      json
// @Param        id   path     string true "ID задачи"
// @Success      204
// @Failure      400  {string} string "Неверный формат ID"
// @Failure      404  {string} string "Задача не найдена"
// @Router       /v1/tasks/{id} [delete]
func (h *TaskHandler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if _, err := uuid.Parse(id); err != nil {
		logger.Log.Warn("Invalid task ID format", "id", id, "error", err)
		http.Error(w, "Invalid task ID format", http.StatusBadRequest)
		return
	}

	if err := h.taskUseCase.Delete(r.Context(), id); err != nil {
		if errors.Is(err, usecase.ErrTaskNotFound) {
			logger.Log.Warn("Task not found for deletion", "id", id)
			http.Error(w, "Task not found", http.StatusNotFound)
		} else {
			logger.Log.Error("Failed to delete task", "id", id, "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// validateTask проверяет валидность данных задачи.
func validateTask(task *entity.Task) error {
	if task.Title == "" {
		return errors.New("title is required")
	}
	if len(task.Title) > 255 {
		return errors.New("title must be less than 255 characters")
	}
	if task.Status == "" {
		return errors.New("status is required")
	}
	return nil
}
