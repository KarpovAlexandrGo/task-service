# Task Service

A simple task management microservice written in Go, following Clean Architecture and Go Standard Project Layout.

## Features
- CRUD operations for tasks
- REST API with Swagger documentation
- PostgreSQL for storage
- Redis for caching
- Dockerized for local development

## Setup
1. Clone the repository: `git clone git@github.com:KarpovAlexandrGo/task-service.git`
2. Install dependencies: `go mod tidy`
3. Set up environment variables in `.env` (see `.env.example`)
4. Run with Docker: `docker-compose up`

## License
MIT