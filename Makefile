.PHONY: build test clean run docker-build docker-run help lint deps

# Binary name
BINARY_NAME=rabbitmq-exporter

# Build flags
LDFLAGS=-ldflags "-X main.Version=$(shell git describe --tags --always --dirty) -X main.BuildTime=$(shell date -u '+%Y-%m-%d_%H:%M:%S')"

# Default target
all: build

# Build the application
build:
	@echo "Building $(BINARY_NAME)..."
	go build $(LDFLAGS) -o $(BINARY_NAME) .

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -cover ./...

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f $(BINARY_NAME)
	go clean

# Run the application locally
run: build
	@echo "Running $(BINARY_NAME)..."
	./$(BINARY_NAME)

# Build Docker image
docker-build:
	@echo "Building Docker image..."
	docker build -t $(BINARY_NAME):latest .

# Run Docker container
docker-run: docker-build
	@echo "Running Docker container..."
	docker run -p 9419:9419 \
		-e RABBITMQ_EXPORTER_RABBITMQ_URL=http://localhost:15672 \
		-e RABBITMQ_EXPORTER_RABBITMQ_USERNAME=guest \
		-e RABBITMQ_EXPORTER_RABBITMQ_PASSWORD=guest \
		$(BINARY_NAME):latest

# Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod tidy

# Lint code
lint:
	@echo "Linting code..."
	golangci-lint run

# Show help
help:
	@echo "Available targets:"
	@echo "  build         - Build the binary"
	@echo "  test          - Run tests"
	@echo "  test-coverage - Run tests with coverage"
	@echo "  clean         - Clean build artifacts"
	@echo "  run           - Build and run locally"
	@echo "  docker-build  - Build Docker image"
	@echo "  docker-run    - Build and run Docker container"
	@echo "  deps          - Install dependencies"
	@echo "  lint          - Lint code"
	@echo "  help          - Show this help" 