# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt
GOLINT=golangci-lint

# Binary name
BINARY_NAME=api
BINARY_UNIX=$(BINARY_NAME)_unix

# Build directories
BUILD_DIR=build
COVERAGE_DIR=coverage

# Docker
DOCKER_COMPOSE=docker-compose
DOCKERFILE=Dockerfile
DOCKERFILE_DEV=Dockerfile.dev

# Main package path
MAIN_PACKAGE=.

# Migration command
MIGRATE_CMD=go run cmd/migrate/main.go

.PHONY: all build clean test coverage lint fmt deps run dev docker-build docker-up docker-down migrate seed help

# Default target
all: clean deps lint test build

## Build:
build: ## Build the binary
	@echo "Building..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) -v $(MAIN_PACKAGE)

build-linux: ## Build for Linux
	@echo "Building for Linux..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_UNIX) -v $(MAIN_PACKAGE)

## Run:
run: ## Run the application
	$(GOCMD) run $(MAIN_PACKAGE)

dev: ## Run with hot reload (requires air)
	@if command -v air > /dev/null; then \
		air; \
	else \
		echo "air is not installed. Install with: go install github.com/cosmtrek/air@latest"; \
		echo "Falling back to go run..."; \
		$(GOCMD) run $(MAIN_PACKAGE); \
	fi

## Dependencies:
deps: ## Download dependencies
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

deps-update: ## Update dependencies
	@echo "Updating dependencies..."
	$(GOMOD) tidy
	$(GOCMD) get -u ./...
	$(GOMOD) tidy

## Testing:
test: ## Run tests
	@echo "Running tests..."
	$(GOTEST) -v ./...

test-short: ## Run short tests
	$(GOTEST) -v -short ./...

test-race: ## Run tests with race detector
	$(GOTEST) -v -race ./...

coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	@mkdir -p $(COVERAGE_DIR)
	$(GOTEST) -v -coverprofile=$(COVERAGE_DIR)/coverage.out ./...
	$(GOCMD) tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@echo "Coverage report generated at $(COVERAGE_DIR)/coverage.html"

coverage-func: ## Show coverage by function
	$(GOTEST) -coverprofile=$(COVERAGE_DIR)/coverage.out ./...
	$(GOCMD) tool cover -func=$(COVERAGE_DIR)/coverage.out

## Code Quality:
lint: ## Run linter
	@if command -v $(GOLINT) > /dev/null; then \
		$(GOLINT) run ./...; \
	else \
		echo "golangci-lint is not installed. Install with:"; \
		echo "  curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b \$$(go env GOPATH)/bin"; \
	fi

lint-fix: ## Run linter and fix issues
	$(GOLINT) run --fix ./...

fmt: ## Format code
	@echo "Formatting code..."
	$(GOFMT) -s -w .

fmt-check: ## Check code formatting
	@echo "Checking code formatting..."
	@test -z "$$($(GOFMT) -l .)" || (echo "Code is not formatted. Run 'make fmt'" && exit 1)

vet: ## Run go vet
	$(GOCMD) vet ./...

## Database:
migrate: ## Run database migrations
	@echo "Running migrations..."
	$(MIGRATE_CMD) -up

migrate-down: ## Rollback database migrations
	@echo "Rolling back migrations..."
	$(MIGRATE_CMD) -down

migrate-reset: ## Reset database (DANGEROUS!)
	@echo "Resetting database..."
	$(MIGRATE_CMD) -reset

seed: ## Run database seeders
	@echo "Running seeders..."
	$(MIGRATE_CMD) -seed

seed-only: ## Run specific seeders (usage: make seed-only SEEDERS=roles,users)
	@echo "Running specific seeders: $(SEEDERS)"
	$(MIGRATE_CMD) -seed-only=$(SEEDERS)

list-seeders: ## List available seeders
	$(MIGRATE_CMD) -list-seeders

## Docker:
docker-build: ## Build Docker image
	docker build -t $(BINARY_NAME) -f $(DOCKERFILE) .

docker-build-dev: ## Build development Docker image
	docker build -t $(BINARY_NAME)-dev -f $(DOCKERFILE_DEV) .

docker-up: ## Start Docker containers
	$(DOCKER_COMPOSE) up -d

docker-down: ## Stop Docker containers
	$(DOCKER_COMPOSE) down

docker-logs: ## Show Docker logs
	$(DOCKER_COMPOSE) logs -f

docker-restart: ## Restart Docker containers
	$(DOCKER_COMPOSE) restart

docker-clean: ## Remove Docker containers and volumes
	$(DOCKER_COMPOSE) down -v --remove-orphans

## Utilities:
clean: ## Clean build artifacts
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -rf $(COVERAGE_DIR)

env-setup: ## Create .env from .env.example
	@if [ ! -f .env ]; then \
		cp .env.example .env; \
		echo ".env file created from .env.example"; \
	else \
		echo ".env file already exists"; \
	fi

generate: ## Run go generate
	$(GOCMD) generate ./...

docs: ## Generate API documentation (requires swag)
	@if command -v swag > /dev/null; then \
		swag init -g main.go; \
	else \
		echo "swag is not installed. Install with: go install github.com/swaggo/swag/cmd/swag@latest"; \
	fi

## Security:
security-check: ## Run security checks (requires gosec)
	@if command -v gosec > /dev/null; then \
		gosec ./...; \
	else \
		echo "gosec is not installed. Install with: go install github.com/securego/gosec/v2/cmd/gosec@latest"; \
	fi

## Help:
help: ## Display this help
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Default target when just running 'make'
.DEFAULT_GOAL := help
