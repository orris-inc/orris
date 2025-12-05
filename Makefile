# Build variables
BINARY_NAME=orris
CMD_PATH=cmd/orris/main.go
BUILD_DIR=bin
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION)"

.PHONY: build
build: ## Build the application binary for current platform
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_PATH)
	@echo "✅ Build completed: $(BUILD_DIR)/$(BINARY_NAME)"

.PHONY: build-linux
build-linux: ## Build for Linux AMD64
	@echo "Building $(BINARY_NAME) for Linux AMD64..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(CMD_PATH)
	@echo "✅ Linux AMD64 build completed: $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64"

.PHONY: build-linux-arm64
build-linux-arm64: ## Build for Linux ARM64
	@echo "Building $(BINARY_NAME) for Linux ARM64..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(CMD_PATH)
	@echo "✅ Linux ARM64 build completed: $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64"

.PHONY: build-windows
build-windows: ## Build for Windows AMD64
	@echo "Building $(BINARY_NAME) for Windows AMD64..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(CMD_PATH)
	@echo "✅ Windows AMD64 build completed: $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe"

.PHONY: build-darwin
build-darwin: ## Build for macOS (Intel and Apple Silicon)
	@echo "Building $(BINARY_NAME) for macOS..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(CMD_PATH)
	@GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(CMD_PATH)
	@echo "✅ macOS builds completed"

.PHONY: build-all
build-all: build-linux build-linux-arm64 build-windows build-darwin ## Build for all platforms
	@echo "✅ All platform builds completed"

.PHONY: compress
compress: ## Compress binaries with UPX
	@echo "Compressing binaries with UPX..."
	@if ! command -v upx >/dev/null 2>&1; then \
		echo "❌ Error: UPX is not installed"; \
		echo "Install with: brew install upx (macOS) or apt-get install upx (Linux)"; \
		exit 1; \
	fi
	@for binary in $(BUILD_DIR)/$(BINARY_NAME)*; do \
		if [ -f "$$binary" ]; then \
			echo "Compressing $$binary..."; \
			upx --best --lzma "$$binary" 2>/dev/null || upx --best "$$binary"; \
		fi \
	done
	@echo "✅ Compression completed"

.PHONY: compress-linux
compress-linux: build-linux ## Build and compress Linux binary
	@echo "Compressing Linux binary with UPX..."
	@if ! command -v upx >/dev/null 2>&1; then \
		echo "❌ Error: UPX is not installed"; \
		echo "Install with: brew install upx (macOS) or apt-get install upx (Linux)"; \
		exit 1; \
	fi
	@upx --best --lzma $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 2>/dev/null || upx --best $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64
	@echo "✅ Linux binary compressed"

.PHONY: release
release: build-all compress ## Build all platforms and compress with UPX
	@echo "📦 Creating release artifacts..."
	@mkdir -p release
	@for binary in $(BUILD_DIR)/$(BINARY_NAME)*; do \
		if [ -f "$$binary" ]; then \
			filename=$$(basename "$$binary"); \
			echo "Packaging $$filename..."; \
			tar -czf release/$$filename.tar.gz -C $(BUILD_DIR) $$filename; \
		fi \
	done
	@echo "✅ Release artifacts created in release/"
	@ls -lh release/

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
	@rm -rf bin/ release/ coverage.out coverage.html
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
	@docker build -t ghcr.io/orris-inc/orris:latest .
	@echo "✅ Docker image built: ghcr.io/orris-inc/orris:latest"

.PHONY: docker-build-multi
docker-build-multi: ## Build Docker image for multiple platforms
	@echo "Building Docker image for multiple platforms..."
	@docker buildx build --platform linux/amd64,linux/arm64 -t ghcr.io/orris-inc/orris:latest .
	@echo "✅ Multi-platform Docker image built"

.PHONY: docker-push
docker-push: docker-build ## Build and push Docker image to ghcr.io
	@echo "Pushing Docker image to ghcr.io..."
	@docker push ghcr.io/orris-inc/orris:latest
	@echo "✅ Docker image pushed: ghcr.io/orris-inc/orris:latest"

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