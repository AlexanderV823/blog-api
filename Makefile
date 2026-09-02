# Makefile для Blog API

# Переменные
APP_NAME := blog-api
MAIN_PATH := cmd/api/main.go
BUILD_DIR := build
DOCKER_COMPOSE := docker-compose

# Цвета для вывода
GREEN := \033[0;32m
YELLOW := \033[0;33m
RED := \033[0;31m
NC := \033[0m # No Color

.PHONY: help
help: ## Показать справку
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "$(GREEN)%-15s$(NC) %s\n", $$1, $$2}'

.PHONY: run
run: ## Запустить приложение с детектором гонок (Режим разработки)
	@echo "$(GREEN)Starting application with race detector...$(NC)"
	go run -race $(MAIN_PATH)

.PHONY: build
build: ## Собрать оптимизированный бинарный файл для продакшна
	@echo "$(GREEN)Building production application...$(NC)"
	@mkdir -p $(BUILD_DIR)
	go build -ldflags="-s -w" -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_PATH)
	@echo "$(GREEN)Build completed: $(BUILD_DIR)/$(APP_NAME)$(NC)"

.PHONY: test
test: ## Запустить все тесты под контролем детектора гонок
	@echo "$(GREEN)Running concurrent tests with race detector...$(NC)"
	go test -v -race ./...

.PHONY: cover
cover: ## Запустить сквозной расчет точного покрытия кода тестами
	@echo "$(GREEN)Calculating explicit statement code coverage...$(NC)"
	go test -race -coverprofile=coverage.out -coverpkg=./internal/handler,./internal/middleware,./internal/service,./pkg/database ./...
	go tool cover -func=coverage.out

.PHONY: cover-html
cover-html: cover ## Открыть интерактивную карту покрытия строк (зеленый/красный) в браузере
	@echo "$(GREEN)Opening coverage HTML report in browser...$(NC)"
	go tool cover -html=coverage.out

.PHONY: fmt
fmt: ## Жестко форматировать код утилитой gofmt по стандартам сдачи
	@echo "$(GREEN)Formatting code with gofmt (including struct tags alignment)...$(NC)"
	gofmt -w .
	@echo "$(GREEN)Formatting completed$(NC)"

.PHONY: fmt-check
fmt-check: ## Проверить соответствие кода стандартам форматирования (должен быть пустой вывод)
	@echo "$(GREEN)Checking code formatting style...$(NC)"
	@if [ -n "$$(gofmt -d .)" ]; then \
		echo "$(RED)Ошибка: Обнаружены файлы с некорректным форматированием!$(NC)"; \
		gofmt -d .; \
		exit 1; \
	fi
	@echo "$(GREEN)Formatting check passed successfully!$(NC)"

.PHONY: deps
deps: ## Скачать и очистить зависимости
	@echo "$(GREEN)Downloading dependencies...$(NC)"
	go mod download
	go mod tidy
	@echo "$(GREEN)Dependencies downloaded$(NC)"

.PHONY: docker-up
docker-up: ## Запустить PostgreSQL в Docker
	@echo "$(GREEN)Starting PostgreSQL container...$(NC)"
	$(DOCKER_COMPOSE) up -d postgres
	@echo "$(GREEN)Waiting for PostgreSQL healthcheck...$(NC)"
	@sleep 3
	@echo "$(GREEN)PostgreSQL process isolated$(NC)"

.PHONY: docker-down
docker-down: ## Остановить контейнеры и очистить именованные тома
	@echo "$(YELLOW)Stopping Docker containers and cleaning volumes...$(NC)"
	$(DOCKER_COMPOSE) down -v
	@echo "$(GREEN)Docker containers and state cleaned$(NC)"

.PHONY: db-shell
db-shell: ## Подключиться к PostgreSQL через psql
	@echo "$(GREEN)Connecting to PostgreSQL shell...$(NC)"
	docker exec -it blog_postgres psql -U bloguser -d blogdb

.PHONY: dev
dev: docker-up ## Запустить среду разработки (БД + безопасное приложение)
	@echo "$(GREEN)Starting integrated development environment...$(NC)"
	@trap '$(MAKE) docker-down' INT TERM; \
	$(MAKE) run

.PHONY: clean
clean: ## Очистить артефакты сборки и файлы профилирования
	@echo "$(YELLOW)Cleaning build artifacts and cover profiles...$(NC)"
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out
	@echo "$(GREEN)Clean completed$(NC)"

# Цель по умолчанию
.DEFAULT_GOAL := help
