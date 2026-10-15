GOCMD=go
BINARY_NAME=api
BUILD_DIR=build
COVERAGE_DIR=coverage
MIGRATE_CMD=$(GOCMD) run ./cmd/migrate
DOCKER_COMPOSE=docker compose

.PHONY: help build build-linux run dev deps test test-race coverage lint fmt fmt-check vet \
	migrate migrate-status seed seed-only list-seeders \
	docker-build docker-up docker-down docker-logs docker-clean clean env-setup check

.DEFAULT_GOAL := help

## Build:
build: ## Build the binary
	@mkdir -p $(BUILD_DIR)
	$(GOCMD) build -o $(BUILD_DIR)/$(BINARY_NAME) .

build-linux: ## Build a static Linux binary
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOCMD) build -trimpath -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME)_linux .

## Run:
run: ## Run the application
	$(GOCMD) run .

dev: ## Run with hot reload (requires air)
	@if command -v air > /dev/null; then \
		air; \
	else \
		echo "air is not installed. Install with: go install github.com/air-verse/air@latest"; \
		$(GOCMD) run .; \
	fi

## Dependencies:
deps: ## Download and tidy dependencies
	$(GOCMD) mod download
	$(GOCMD) mod tidy

## Quality:
test: ## Run tests
	$(GOCMD) test ./...

test-race: ## Run tests with the race detector
	$(GOCMD) test -race ./...

coverage: ## Generate an HTML coverage report
	@mkdir -p $(COVERAGE_DIR)
	$(GOCMD) test -coverprofile=$(COVERAGE_DIR)/coverage.out ./...
	$(GOCMD) tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@echo "Coverage report: $(COVERAGE_DIR)/coverage.html"

lint: ## Run golangci-lint
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint is not installed: https://golangci-lint.run/welcome/install/"; \
	fi

fmt: ## Format code
	gofmt -s -w .

fmt-check: ## Fail if code is not formatted
	@test -z "$$(gofmt -l .)" || (echo "Code is not formatted. Run 'make fmt'" && gofmt -l . && exit 1)

vet: ## Run go vet
	$(GOCMD) vet ./...

check: fmt-check vet test ## Run all local checks

## Database:
migrate: ## Apply pending migrations
	$(MIGRATE_CMD) -up

migrate-status: ## Show applied/pending migrations
	$(MIGRATE_CMD) -status

seed: ## Run all seeders
	$(MIGRATE_CMD) -seed

seed-only: ## Run specific seeders (make seed-only SEEDERS=roles,users)
	$(MIGRATE_CMD) -seed-only=$(SEEDERS)

list-seeders: ## List available seeders
	$(MIGRATE_CMD) -list-seeders

## Docker:
docker-build: ## Build the production image
	docker build -t $(BINARY_NAME) .

docker-up: ## Start the dev stack (api + postgres + redis)
	$(DOCKER_COMPOSE) up -d --build

docker-down: ## Stop the dev stack
	$(DOCKER_COMPOSE) down

docker-logs: ## Tail dev stack logs
	$(DOCKER_COMPOSE) logs -f

docker-clean: ## Stop the dev stack and remove volumes
	$(DOCKER_COMPOSE) down -v --remove-orphans

## Utilities:
clean: ## Remove build artifacts
	$(GOCMD) clean
	rm -rf $(BUILD_DIR) $(COVERAGE_DIR) tmp

env-setup: ## Create .env from .env.example
	@if [ ! -f .env ]; then cp .env.example .env && echo ".env created from .env.example"; else echo ".env already exists"; fi

help: ## Show this help
	@echo "Usage: make [target]"
	@echo ""
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
