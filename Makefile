.PHONY: test test-integration test-verbose coverage clean build run help

# Default target
help:
	@echo "Available targets:"
	@echo "  test           - Run unit tests"
	@echo "  test-verbose   - Run tests with verbose output"
	@echo "  test-integration - Run integration tests"
	@echo "  coverage       - Generate test coverage report"
	@echo "  build          - Build the application"
	@echo "  clean          - Clean build artifacts"
	@echo "  run            - Run the application"

# Run tests
test:
	@echo "Running tests..."
	@gotestsum --format testname

# Run tests with verbose output
test-verbose:
	@echo "Running tests with verbose output..."
	@go test -v -cover -count=1 ./...

# Run integration tests
test-integration:
	@echo "Running integration tests..."
	@go test -v -tags=integration ./...

# Generate coverage report
coverage:
	@echo "Generating coverage report..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Build the application
build:
	@echo "Building clia..."
	@go build -o bin/clia cmd/clia/main.go
	@echo "Build complete: bin/clia"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf bin/ coverage.out coverage.html
	@go clean
	@echo "Clean complete"

# Run the application
run:
	@go run cmd/clia/main.go

# Run tests and show coverage in terminal
test-cover:
	@go test -cover ./...