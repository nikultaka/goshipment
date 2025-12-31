.PHONY: help run build test clean migrate migrate-reset dev install

# Variables
BINARY_NAME=goapi
CMD_DIR=./cmd/api
BUILD_DIR=./bin

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

install: ## Install dependencies
	@echo "📦 Installing dependencies..."
	@go mod download
	@go mod tidy
	@echo "✓ Dependencies installed"

run: ## Run the application
	@echo "🚀 Starting server..."
	@go run $(CMD_DIR)/main.go

dev: ## Run in development mode with live reload (requires air)
	@if command -v air > /dev/null; then \
		air; \
	else \
		echo "⚠️  air not installed. Install with: go install github.com/air-verse/air@latest"; \
		echo "Running without live reload..."; \
		make run; \
	fi

build: ## Build the application
	@echo "🔨 Building..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)/main.go
	@echo "✓ Built: $(BUILD_DIR)/$(BINARY_NAME)"

test: ## Run tests
	@echo "🧪 Running tests..."
	@go test -v ./...

test-coverage: ## Run tests with coverage
	@echo "🧪 Running tests with coverage..."
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage report: coverage.html"

migrate: ## Run database migrations
	@echo "🔄 Running migrations..."
	@./scripts/migrate.sh up

migrate-reset: ## Reset database (drop all tables and remigrate)
	@./scripts/migrate.sh reset

clean: ## Clean build artifacts
	@echo "🧹 Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@echo "✓ Cleaned"

fmt: ## Format code
	@echo "🎨 Formatting code..."
	@go fmt ./...
	@echo "✓ Code formatted"

lint: ## Run linter (requires golangci-lint)
	@if command -v golangci-lint > /dev/null; then \
		echo "🔍 Running linter..."; \
		golangci-lint run; \
	else \
		echo "⚠️  golangci-lint not installed"; \
		echo "Install: https://golangci-lint.run/usage/install/"; \
	fi

docker-build: ## Build Docker image
	@echo "🐳 Building Docker image..."
	@docker build -t $(BINARY_NAME):latest .

docker-run: ## Run Docker container
	@echo "🐳 Running Docker container..."
	@docker run -p 8080:8080 --env-file .env $(BINARY_NAME):latest

.DEFAULT_GOAL := help
