# TaskD Makefile

.PHONY: build clean test install dev fmt lint

# Variable definitions
BINARY_NAME=taskd
BUILD_DIR=build
MAIN_PATH=cmd/taskd/main.go

# Default target
all: build

# Build
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME).exe $(MAIN_PATH)

# Development build
dev:
	@echo "Development build..."
	go build -race -o $(BUILD_DIR)/$(BINARY_NAME)-dev.exe $(MAIN_PATH)

# Clean
clean:
	@echo "Cleaning build files..."
	@rm -rf $(BUILD_DIR)
	go clean

# Test
test:
	@echo "Running tests..."
	go test -v ./...

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Lint code
lint:
	@echo "Linting code..."
	golangci-lint run

# Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod tidy
	go mod download

# Cross compile
build-all:
	@echo "Cross compiling..."
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PATH)
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)
	GOOS=darwin GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)

# Run
run:
	go run $(MAIN_PATH)

# Help
help:
	@echo "Available commands:"
	@echo "  build     - Build executable"
	@echo "  dev       - Development build"
	@echo "  clean     - Clean build files"
	@echo "  test      - Run tests"
	@echo "  fmt       - Format code"
	@echo "  lint      - Lint code"
	@echo "  deps      - Install dependencies"
	@echo "  build-all - Cross compile"
	@echo "  run       - Run program"