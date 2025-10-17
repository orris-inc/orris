.PHONY: build
build: ## Build the application binary
	@echo "Building orris..."
	@go build -o bin/orris cmd/orris/main.go
	@echo "✅ Build completed: bin/orris"

.PHONY: swagger
swagger: ## Generate Swagger documentation
	@echo "Generating Swagger docs..."
	@~/go/bin/swag init -g cmd/orris/main.go --output docs --parseDependency --parseInternal
	@echo "✅ Swagger docs generated"

.PHONY: run
run: build ## Run the server
	@./bin/orris server

.PHONY: migrate-up
migrate-up: build ## Run database migrations up
	@./bin/orris migrate up

.PHONY: migrate-down
migrate-down: build ## Rollback database migrations
	@./bin/orris migrate down

.PHONY: migrate-status
migrate-status: build ## Check migration status
	@./bin/orris migrate status

.PHONY: migrate-create
migrate-create: build ## Create a new migration (use NAME=migration_name)
	@if [ -z "$(NAME)" ]; then \
		echo "❌ Error: NAME is required. Usage: make migrate-create NAME=your_migration_name"; \
		exit 1; \
	fi
	@./bin/orris migrate create --name=$(NAME)

.PHONY: test
test: ## Run tests
	@echo "Running tests..."
	@go test ./...

.PHONY: test-coverage
test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report generated: coverage.html"

.PHONY: clean
clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf bin/ coverage.out coverage.html
	@echo "✅ Clean completed"

.PHONY: deps
deps: ## Download dependencies
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy
	@echo "✅ Dependencies updated"

.PHONY: lint
lint: ## Run linters
	@echo "Running linters..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

.PHONY: fmt
fmt: ## Format code
	@echo "Formatting code..."
	@go fmt ./...
	@gofmt -s -w .
	@echo "✅ Code formatted"

.PHONY: dev
dev: ## Run server in development mode with auto-reload (requires air)
	@if command -v air >/dev/null 2>&1; then \
		air; \
	else \
		echo "air not installed. Run: go install github.com/air-verse/air@latest"; \
		echo "Falling back to regular run..."; \
		make run; \
	fi

.PHONY: docker-build
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	@docker build -t orris:latest .
	@echo "✅ Docker image built: orris:latest"

.PHONY: docker-up
docker-up: ## Start services with docker-compose
	@docker-compose up -d
	@echo "✅ Services started"

.PHONY: docker-down
docker-down: ## Stop services with docker-compose
	@docker-compose down
	@echo "✅ Services stopped"

.PHONY: docker-logs
docker-logs: ## View docker-compose logs
	@docker-compose logs -f

.PHONY: db-reset
db-reset: ## Reset database (drop and recreate)
	@echo "⚠️  Warning: This will delete all data!"
	@read -p "Are you sure? [y/N] " confirm && \
	if [ "$$confirm" = "y" ]; then \
		./bin/orris migrate down --steps=999; \
		./bin/orris migrate up; \
		echo "✅ Database reset completed"; \
	else \
		echo "❌ Operation cancelled"; \
	fi

.PHONY: install
install: deps build ## Install the application
	@echo "Installing orris to /usr/local/bin..."
	@sudo cp bin/orris /usr/local/bin/
	@echo "✅ Installation completed"

.PHONY: help
help: ## Display this help message
	@echo "Orris - Makefile Commands"
	@echo ""
	@echo "Usage: make [command]"
	@echo ""
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

# Default target
.DEFAULT_GOAL := help