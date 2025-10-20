.PHONY: help build test test-unit test-integration clean docker-build docker-up docker-down migrate lint

# Build variables
BINARY_NAME=url-shortener
DOCKER_IMAGE=url-shortener:latest
GO_VERSION=1.21

# Default target
help:
	@echo "URL Shortener - Available targets:"
	@echo "  build       - Build the application"
	@echo "  test        - Run all tests"
	@echo "  test-unit   - Run unit tests only"
	@echo "  test-integration - Run integration tests only"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-up   - Start services with Docker Compose"
	@echo "  docker-down - Stop services"
	@echo "  migrate     - Run database migrations"
	@echo "  lint        - Run linter"
	@echo "  clean       - Clean build artifacts"

# Build the application
build:
	@echo "Building $(BINARY_NAME)..."
	go build -ldflags="-w -s" -o bin/$(BINARY_NAME) ./cmd/server

# Run tests
test: test-unit test-integration

test-unit:
	@echo "Running unit tests..."
	go test -v -race -short ./tests/unit/... -coverprofile=coverage-unit.out

test-integration:
	@echo "Running integration tests..."
	docker-compose -f docker/docker-compose.test.yml up --abort-on-container-exit --exit-code-from test-runner

test-coverage: test-unit
	@echo "Generating coverage report..."
	go tool cover -html=coverage-unit.out -o coverage.html

# Docker commands
docker-build:
	@echo "Building Docker image..."
	docker build -f docker/Dockerfile -t $(DOCKER_IMAGE) .

docker-up:
	@echo "Starting services..."
	docker-compose -f docker/docker-compose.yml up -d

docker-down:
	@echo "Stopping services..."
	docker-compose -f docker/docker-compose.yml down

docker-logs:
	docker-compose -f docker/docker-compose.yml logs -f

# Database migrations
migrate:
	@echo "Running migrations..."
	@if [ -f .env ]; then \
		source .env && \
		psql $$DB_URL -f migrations/001_create_urls_table.sql; \
	else \
		echo "Please create .env file first"; \
		exit 1; \
	fi

# Code quality
lint:
	@echo "Running linters..."
	golangci-lint run ./...

# Development
dev:
	@echo "Starting development server..."
	air -c .air.toml

# Clean up
clean:
	@echo "Cleaning up..."
	rm -rf bin/
	rm -f coverage*.out coverage.html
	go clean -testcache

# Database operations
db-seed:
	@echo "Seeding database..."
	go run scripts/seed.go

db-reset: docker-down docker-up
	@sleep 5
	@$(MAKE) migrate

# Release preparation
release: test lint docker-build
	@echo "Release build complete"

# Security scanning
security-scan:
	@echo "Running security scan..."
	gosec ./...
	trivy image $(DOCKER_IMAGE)

# Performance testing
benchmark:
	@echo "Running performance benchmarks..."
	go test -bench=. -benchmem ./tests/benchmarks/...

# Dependency management
deps-update:
	@echo "Updating dependencies..."
	go get -u ./...
	go mod tidy

deps-audit:
	@echo "Auditing dependencies..."
	go list -m all | nancy sleuth

# Generate documentation
docs:
	@echo "Generating API documentation..."
	swag init -g cmd/server/main.go

# Load testing (requires hey or ab)
load-test:
	@echo "Running load test..."
	hey -n 1000 -c 50 http://localhost:8080/health