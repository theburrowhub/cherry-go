# Cherry-go Makefile

.PHONY: build test clean install demo help

# Build the binary
build:
	@echo "Building cherry-go..."
	@go build -o cherry-go

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -f cherry-go
	@rm -rf .cherry-go/

# Install dependencies
deps:
	@echo "Installing dependencies..."
	@go mod tidy
	@go mod download

# Run demo
demo: build
	@echo "Running demo..."
	@./examples/demo.sh

# Install to GOPATH/bin
install:
	@echo "Installing cherry-go..."
	@go install

# Install locally using installation script
install-local:
	@echo "Installing cherry-go locally..."
	@./install.sh

# Install locally (quick)
install-quick:
	@echo "Quick install cherry-go locally..."
	@./scripts/quick-install.sh

# Uninstall local installation
uninstall:
	@echo "Uninstalling cherry-go..."
	@./scripts/uninstall.sh

# Run linting
lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found, running go vet instead"; \
		go vet ./...; \
	fi

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Run all checks
check: fmt lint test
	@echo "All checks passed!"

# Development setup
dev-setup: deps
	@echo "Setting up development environment..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Show help
help:
	@echo "Available targets:"
	@echo "  build         - Build the binary"
	@echo "  test          - Run tests"
	@echo "  clean         - Clean build artifacts"
	@echo "  deps          - Install dependencies"
	@echo "  demo          - Run demo script"
	@echo "  install       - Install to GOPATH/bin"
	@echo "  install-local - Install locally using installation script"
	@echo "  install-quick - Quick local install"
	@echo "  uninstall     - Uninstall local installation"
	@echo "  lint          - Run linter"
	@echo "  fmt           - Format code"
	@echo "  check         - Run all checks (fmt, lint, test)"
	@echo "  dev-setup     - Setup development environment"
	@echo "  help          - Show this help"

