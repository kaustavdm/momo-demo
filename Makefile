# Build configuration
BINARY_NAME=momo
BUILD_DIR=build
MAIN_FILE=main.go

# Go commands
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod
GOGET=$(GOCMD) get
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet

# Version information
VERSION?=0.0.1
BUILD_TIME=$(shell date +%FT%T%z)
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Build flags
LDFLAGS=-ldflags "-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME} -X main.GitCommit=${GIT_COMMIT}"

.PHONY: all build clean test lint deps help

all: clean lint test build

# Build the application
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) ${LDFLAGS} -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_FILE)
	@echo "Build complete. Binary: $(BUILD_DIR)/$(BINARY_NAME)"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@echo "Clean complete"

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...
	@echo "Tests complete"

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out
	@echo "Coverage analysis complete"

# Lint the code
lint:
	@echo "Running linters..."
	$(GOFMT) ./...
	$(GOVET) ./...
	@echo "Lint complete"

# Install dependencies
deps:
	@echo "Installing dependencies..."
	$(GOMOD) download
	@echo "Dependencies installed"

# Run the application
run: build
	@echo "Running $(BINARY_NAME)..."
	./$(BUILD_DIR)/$(BINARY_NAME)

# Run with specific modules
run-modules:
	@echo "Running specific modules..."
	./$(BUILD_DIR)/$(BINARY_NAME) -modules=$(modules)

# Build for multiple platforms
build-all: clean
	@echo "Building for multiple platforms..."
	GOOS=linux GOARCH=amd64 $(GOBUILD) ${LDFLAGS} -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_FILE)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) ${LDFLAGS} -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_FILE)
	GOOS=windows GOARCH=amd64 $(GOBUILD) ${LDFLAGS} -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_FILE)
	@echo "Multi-platform build complete"

# Show help
help:
	@echo "Available commands:"
	@echo "  make build         - Build the application"
	@echo "  make clean         - Remove build artifacts"
	@echo "  make test          - Run tests"
	@echo "  make test-coverage - Run tests with coverage analysis"
	@echo "  make lint          - Run linters"
	@echo "  make deps          - Install dependencies"
	@echo "  make run           - Build and run the application"
	@echo "  make run-modules   - Run specific modules (use: make run-modules modules=scheduler,worker)"
	@echo "  make build-all     - Build for multiple platforms"
	@echo "  make help          - Show this help message"