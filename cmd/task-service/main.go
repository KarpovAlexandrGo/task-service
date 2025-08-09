package main

import (
	"net/http"

	_ "github.com/KarpovAlexandrGo/task-service/docs" // Для Swagger (сгенерируется swag)
	"github.com/KarpovAlexandrGo/task-service/internal/app"
	"github.com/KarpovAlexandrGo/task-service/pkg/logger"
	"github.com/go-chi/chi"

	httpSwagger "github.com/swaggo/http-swagger"
)

// @title           Task Service API
// @version         1.0
// @description     This is a sample server for managing tasks.
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8080
// @BasePath  /v1

func main() {
	a, err := app.NewApp()
	if err != nil {
		logger.Log.WithError(err).Fatal("Failed to initialize app")
	}

	// Регистрация Swagger
	a.Server.Handler = setupSwagger(a.Server.Handler)

	if err := a.Run(); err != nil {
		logger.Log.WithError(err).Fatal("Failed to run app")
	}
}

// setupSwagger настраивает маршруты для Swagger UI.
func setupSwagger(handler http.Handler) http.Handler {
	r := chi.NewRouter()

	// Убрал wildcard из основного маршрута задач
	r.Mount("/v1/tasks", handler)

	// Исправил маршрут для Swagger UI
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"), // Указывает путь к вашему swagger.json
	))

	return r
}
