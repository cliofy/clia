.PHONY: build clean test lint fmt vet tidy run install help

# Version information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

# Build flags
LDFLAGS = -X github.com/yourusername/clia/internal/version.Version=$(VERSION) \
          -X github.com/yourusername/clia/internal/version.GitCommit=$(COMMIT) \
          -X github.com/yourusername/clia/internal/version.BuildTime=$(BUILD_TIME)

# Build directory
BUILD_DIR = bin

# Binary name
BINARY_NAME = clia

# Go commands
GO_BUILD = go build -ldflags "$(LDFLAGS)"
GO_TEST = go test -v
GO_CLEAN = go clean
GO_FMT = go fmt
GO_VET = go vet
GO_MOD_TIDY = go mod tidy

## build: Build the binary
build: clean tidy
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GO_BUILD) -o $(BUILD_DIR)/$(BINARY_NAME) cmd/clia/main.go
	@echo "Binary built: $(BUILD_DIR)/$(BINARY_NAME)"

## build-all: Build for all platforms
build-all: clean tidy
	@echo "Building for all platforms..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GO_BUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 cmd/clia/main.go
	GOOS=darwin GOARCH=amd64 $(GO_BUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 cmd/clia/main.go
	GOOS=darwin GOARCH=arm64 $(GO_BUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 cmd/clia/main.go
	GOOS=windows GOARCH=amd64 $(GO_BUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe cmd/clia/main.go
	@echo "Cross-platform binaries built in $(BUILD_DIR)/"

## test: Run tests
test: tidy
	@echo "Running tests..."
	$(GO_TEST) ./...

## test-cover: Run tests with coverage
test-cover: tidy
	@echo "Running tests with coverage..."
	$(GO_TEST) -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## bench: Run benchmarks
bench: tidy
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./...

## lint: Run golangci-lint
lint:
	@echo "Running golangci-lint..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not found. Install it from https://golangci-lint.run/usage/install/" && exit 1)
	golangci-lint run

## fmt: Format code
fmt:
	@echo "Formatting code..."
	$(GO_FMT) ./...

## vet: Run go vet
vet:
	@echo "Running go vet..."
	$(GO_VET) ./...

## tidy: Tidy go.mod
tidy:
	@echo "Tidying go.mod..."
	$(GO_MOD_TIDY)

## clean: Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GO_CLEAN)
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

## run: Run the application
run: build
	@echo "Running $(BINARY_NAME)..."
	./$(BUILD_DIR)/$(BINARY_NAME)

## install: Install the binary to $GOPATH/bin
install: tidy
	@echo "Installing $(BINARY_NAME) to $$GOPATH/bin..."
	go install -ldflags "$(LDFLAGS)" cmd/clia/main.go

## dev: Run in development mode with live reload
dev:
	@echo "Starting development server..."
	@which air > /dev/null || go install github.com/cosmtrek/air@latest
	air

## check: Run all checks (fmt, vet, lint, test)
check: fmt vet lint test
	@echo "All checks passed!"

## help: Show this help
help: Makefile
	@echo "Available targets:"
	@sed -n 's/^##//p' $< | sort

# Default target
.DEFAULT_GOAL := help