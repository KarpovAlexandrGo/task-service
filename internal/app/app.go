package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/KarpovAlexandrGo/task-service/internal/entity"
	"github.com/KarpovAlexandrGo/task-service/internal/repo/postgres"
	"github.com/KarpovAlexandrGo/task-service/internal/usecase"
	"github.com/KarpovAlexandrGo/task-service/pkg/logger"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/viper"
)

type App struct {
	Server      *http.Server
	wg          sync.WaitGroup
	dbPool      *pgxpool.Pool
	taskUseCase usecase.TaskUseCase
	cacheRepo   usecase.CacheRepository
}

func NewApp() (*App, error) {
	if err := loadConfig(); err != nil {
		return nil, err
	}

	dbPool, err := initDB()
	if err != nil {
		return nil, err
	}

	// Инициализация репозиториев
	taskRepo := postgres.NewTaskRepository(dbPool)
	cacheRepo := postgres.NewCacheRepository(dbPool) // или другая реализация кэша

	// Инициализация use case
	taskUseCase := usecase.NewTaskUseCase(taskRepo, cacheRepo)

	router := setupRouter(taskUseCase)

	server := &http.Server{
		Addr:    ":" + viper.GetString("HTTP_PORT"),
		Handler: router,
	}

	return &App{
		Server:      server,
		dbPool:      dbPool,
		taskUseCase: taskUseCase,
		cacheRepo:   cacheRepo,
	}, nil
}

func loadConfig() error {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")
	viper.AutomaticEnv()
	viper.SetDefault("HTTP_PORT", "8080")
	viper.SetDefault("POSTGRES_DSN", "postgres://user:password@localhost:5432/tasks?sslmode=disable")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("error reading config file: %w", err)
		}
		logger.Log.Info("Using default configuration")
	}
	return nil
}

func initDB() (*pgxpool.Pool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dbPool, err := pgxpool.New(ctx, viper.GetString("POSTGRES_DSN"))
	if err != nil {
		return nil, fmt.Errorf("failed to create pgx pool: %w", err)
	}

	if err := dbPool.Ping(ctx); err != nil {
		dbPool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Log.Info("Connected to database successfully")
	return dbPool, nil
}

func setupRouter(taskUC usecase.TaskUseCase) *chi.Mux {
	router := chi.NewRouter()

	router.Use(
		middleware.RequestID,
		middleware.Logger,
		middleware.Recoverer,
		middleware.Heartbeat("/health"),
		middleware.Timeout(60*time.Second),
	)

	router.Route("/api/v1", func(r chi.Router) {
		r.Route("/tasks", func(r chi.Router) {
			r.Post("/", createTaskHandler(taskUC))
			r.Get("/", listTasksHandler(taskUC))
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", getTaskHandler(taskUC))
				r.Put("/", updateTaskHandler(taskUC))
				r.Delete("/", deleteTaskHandler(taskUC))
			})
		})
	})

	router.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("pong"))
	})

	return router
}

func createTaskHandler(uc usecase.TaskUseCase) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var task entity.Task
		if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid request payload")
			return
		}

		createdTask, err := uc.Create(r.Context(), task)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondWithJSON(w, http.StatusCreated, createdTask)
	}
}

func getTaskHandler(uc usecase.TaskUseCase) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		task, err := uc.Get(r.Context(), id)
		if err != nil {
			if errors.Is(err, usecase.ErrTaskNotFound) {
				respondWithError(w, http.StatusNotFound, "Task not found")
			} else {
				respondWithError(w, http.StatusInternalServerError, err.Error())
			}
			return
		}

		respondWithJSON(w, http.StatusOK, task)
	}
}

func listTasksHandler(uc usecase.TaskUseCase) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page := 1
		limit := 20
		// Здесь можно добавить парсинг query-параметров для page и limit

		tasks, err := uc.List(r.Context(), page, limit)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondWithJSON(w, http.StatusOK, tasks)
	}
}

func updateTaskHandler(uc usecase.TaskUseCase) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		var task entity.Task
		if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid request payload")
			return
		}
		task.ID = uuid.MustParse(id) // предполагается, что entity.Task.ID имеет тип uuid.UUID

		updatedTask, err := uc.Update(r.Context(), task)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondWithJSON(w, http.StatusOK, updatedTask)
	}
}

func deleteTaskHandler(uc usecase.TaskUseCase) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		if err := uc.Delete(r.Context(), id); err != nil {
			respondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondWithJSON(w, http.StatusNoContent, nil)
	}
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if payload != nil {
		json.NewEncoder(w).Encode(payload)
	}
}

func (a *App) Run() error {
	defer a.dbPool.Close()

	serverCtx, serverStopCtx := context.WithCancel(context.Background())

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		<-sig
		logger.Log.Info("Shutdown signal received")

		shutdownCtx, cancel := context.WithTimeout(serverCtx, 30*time.Second)
		defer cancel()

		go func() {
			<-shutdownCtx.Done()
			if shutdownCtx.Err() == context.DeadlineExceeded {
				logger.Log.Error("Graceful shutdown timed out")
			}
		}()

		if err := a.Server.Shutdown(shutdownCtx); err != nil {
			logger.Log.WithError(err).Error("HTTP server shutdown failed")
		}
		serverStopCtx()
	}()

	logger.Log.Info("Starting server on " + a.Server.Addr)
	if err := a.Server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server failed: %w", err)
	}

	a.wg.Wait()
	logger.Log.Info("Server stopped gracefully")
	return nil
}
