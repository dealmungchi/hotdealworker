.PHONY: build test unit-test integration-test clean run deps lint fmt coverage docker-build docker-run help

# Default target
all: deps fmt lint test build

# Build configuration
BINARY_NAME=hotdealworker
BUILD_DIR=build
GO_FILES=$(shell find . -name '*.go' -type f -not -path "./vendor/*")

# Build targets
build:
	@echo "Building $(BINARY_NAME)..."
	@go build -o $(BINARY_NAME) -v

build-linux:
	@echo "Building for Linux..."
	@GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 -v

build-all: build build-linux

# Test targets
test: unit-test integration-test

unit-test:
	@echo "Running unit tests..."
	@go test -v ./... -run "^Test[^I]" -race -short

integration-test:
	@echo "Running integration tests..."
	@go test -v ./... -run "^TestIntegration" -race

coverage:
	@echo "Running tests with coverage..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Clean targets
clean:
	@echo "Cleaning..."
	@rm -f $(BINARY_NAME)
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@rm -f error.old
	@go clean

# Run targets
run:
	@echo "Running $(BINARY_NAME)..."
	@go run main.go

run-dev:
	@echo "Running in development mode..."
	@HOTDEAL_ENVIRONMENT=development LOG_LEVEL=debug go run main.go

# Development targets
deps:
	@echo "Installing dependencies..."
	@go mod tidy
	@go mod download

fmt:
	@echo "Formatting code..."
	@gofmt -s -w $(GO_FILES)
	@goimports -w $(GO_FILES) 2>/dev/null || true

lint:
	@echo "Running linter..."
	@golangci-lint run --deadline=5m || true

vet:
	@echo "Running go vet..."
	@go vet ./...

# Docker targets
docker-build:
	@echo "Building Docker image..."
	@docker build -t $(BINARY_NAME):latest .

docker-run:
	@echo "Running Docker container..."
	@docker-compose up

docker-down:
	@echo "Stopping Docker containers..."
	@docker-compose down

# Utility targets
env-setup:
	@echo "Setting up environment..."
	@cp -n .env.example .env || true
	@echo "Created .env file from .env.example"

check-env:
	@echo "Checking environment variables..."
	@[ -f .env ] || (echo "No .env file found. Run 'make env-setup' first." && exit 1)
	@echo "Environment file exists"

# Database targets
redis-cli:
	@echo "Connecting to Redis..."
	@redis-cli

memcache-stats:
	@echo "Getting Memcache stats..."
	@echo stats | nc localhost 11211

# Help target
help:
	@echo "Available targets:"
	@echo "  make build          - Build the binary"
	@echo "  make build-linux    - Build for Linux"
	@echo "  make test           - Run all tests"
	@echo "  make unit-test      - Run unit tests only"
	@echo "  make integration-test - Run integration tests"
	@echo "  make coverage       - Generate test coverage report"
	@echo "  make run            - Run the application"
	@echo "  make run-dev        - Run in development mode"
	@echo "  make deps           - Install dependencies"
	@echo "  make fmt            - Format code"
	@echo "  make lint           - Run linter"
	@echo "  make clean          - Clean build artifacts"
	@echo "  make docker-build   - Build Docker image"
	@echo "  make docker-run     - Run with Docker Compose"
	@echo "  make env-setup      - Setup environment file"
	@echo "  make help           - Show this help message"
