.PHONY: build run test clean docker-build docker-up docker-down lint fmt

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet

# Binary name
BINARY_NAME=GopherSentinel
BINARY_PATH=bin/$(BINARY_NAME)

# Docker
DOCKER_IMAGE=gophersentinel
DOCKER_TAG=latest

# Build the application
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p bin
	$(GOBUILD) -o $(BINARY_PATH) ./cmd/cli/

# Run the application
run: build
	@echo "Running $(BINARY_NAME)..."
	$(BINARY_PATH) ask "hello"

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v -race -cover ./...

# Run tests with coverage report
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Clean build files
clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

# Format code
fmt:
	@echo "Formatting code..."
	$(GOFMT) ./...

# Lint code
lint:
	@echo "Linting code..."
	$(GOVET) ./...

# Build Docker image
docker-build:
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

# Push Docker image
docker-push:
	@echo "Pushing Docker image..."
	docker push $(DOCKER_IMAGE):$(DOCKER_TAG)

# Start services with docker-compose
docker-up:
	@echo "Starting services..."
	docker-compose up -d

# Stop services
docker-down:
	@echo "Stopping services..."
	docker-compose down

# View logs
docker-logs:
	docker-compose logs -f

# Run end-to-end tests
e2e:
	@echo "Running e2e tests..."
	$(GOTEST) -v -tags=e2e ./e2e/...

# Generate mocks (if using mockgen)
mocks:
	@echo "Generating mocks..."
	@find . -name "*_test.go" -exec grep -l "mock" {} \; | head -5

# Show help
help:
	@echo "Available targets:"
	@echo "  build          - Build the application"
	@echo "  run            - Build and run the application"
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Run tests with coverage report"
	@echo "  clean          - Clean build files"
	@echo "  deps           - Download dependencies"
	@echo "  fmt            - Format code"
	@echo "  lint           - Lint code"
	@echo "  docker-build   - Build Docker image"
	@echo "  docker-up      - Start services"
	@echo "  docker-down    - Stop services"
	@echo "  e2e            - Run e2e tests"
