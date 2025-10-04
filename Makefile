.PHONY: help proto build run clean docker test lint

# Variables
PROTO_DIR := api/proto
GO_OUT_DIR := api/proto
SERVICES := common gateway router message user

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

proto: ## Generate protobuf code
	@echo "🔨 Generating protobuf code..."
	@bash scripts/generate_proto.sh

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy

build: ## Build all services
	@echo "Building services..."
	@for service in gateway router message user file; do \
		echo "Building $$service..."; \
		CGO_ENABLED=0 go build -o bin/$$service cmd/$$service/main.go; \
	done

build-user: ## Build user service
	@echo "Building user service..."
	@CGO_ENABLED=0 go build -o bin/user cmd/user/main.go

build-router: ## Build router service
	@echo "Building router service..."
	@CGO_ENABLED=0 go build -o bin/router cmd/router/main.go

build-message: ## Build message service
	@echo "Building message service..."
	@CGO_ENABLED=0 go build -o bin/message cmd/message/main.go

build-gateway: ## Build gateway service
	@echo "Building gateway service..."
	@CGO_ENABLED=0 go build -o bin/gateway cmd/gateway/main.go

build-file: ## Build file service
	@echo "Building file service..."
	@CGO_ENABLED=0 go build -o bin/file cmd/file/main.go

run-user: ## Run user service
	@./bin/user

run-router: ## Run router service
	@./bin/router

run-message: ## Run message service
	@./bin/message

run-gateway: ## Run gateway service
	@./bin/gateway

run-file: ## Run file service
	@./bin/file

test: ## Run tests
	@echo "Running tests..."
	@go test -v -race -coverprofile=coverage.out ./...

test-coverage: test ## Run tests and show coverage
	@go tool cover -html=coverage.out

lint: ## Run linter
	@echo "Running linter..."
	@golangci-lint run ./...

fmt: ## Format code
	@echo "Formatting code..."
	@go fmt ./...
	@goimports -w .

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -f coverage.out

docker-build: ## Build Docker images
	@echo "Building Docker images..."
	@docker-compose build

docker-up: ## Start services with docker-compose
	@echo "Starting services..."
	@docker-compose up -d

docker-down: ## Stop services
	@echo "Stopping services..."
	@docker-compose down

docker-logs: ## Show docker logs
	@docker-compose logs -f

db-migrate: ## Run database migrations
	@echo "Running migrations..."
	@psql $(DATABASE_URL) -f migrations/001_init_schema.sql

db-reset: ## Reset database (WARNING: destructive)
	@echo "Resetting database..."
	@psql -U $(POSTGRES_USER) -c "DROP DATABASE IF EXISTS $(POSTGRES_DB);"
	@psql -U $(POSTGRES_USER) -c "CREATE DATABASE $(POSTGRES_DB);"
	@make db-migrate

install-tools: ## Install development tools
	@echo "Installing tools..."
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install golang.org/x/tools/cmd/goimports@latest

.DEFAULT_GOAL := help
