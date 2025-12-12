# Cherry-go Makefile

.PHONY: build test clean install demo help \
        docker-build docker-build-local docker-run docker-push docker-clean \
        goreleaser-check goreleaser-build goreleaser-release-dry

# Variables
DOCKER_IMAGE := cherry-go
DOCKER_TAG := dev
DOCKER_REGISTRY := ghcr.io/theburrowhub

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
	@rm -rf dist/

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

# ========== DOCKER TARGETS ==========

# Build Docker image (requires binary to exist)
docker-build: build
	@echo "Building Docker image $(DOCKER_IMAGE):$(DOCKER_TAG)..."
	@docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

# Build Docker image for local testing with multi-arch support
docker-build-multiarch: build
	@echo "Building Docker image with buildx for current platform..."
	@docker buildx build --load -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

# Run cherry-go in Docker container
# Usage: make docker-run ARGS="status" or make docker-run ARGS="sync --all"
docker-run:
	@docker run --rm -v "$$(pwd)":/workspace $(DOCKER_IMAGE):$(DOCKER_TAG) $(ARGS)

# Run cherry-go in Docker with SSH keys mounted
# Usage: make docker-run-ssh ARGS="sync myrepo"
docker-run-ssh:
	@docker run --rm \
		-v "$$(pwd)":/workspace \
		-v "$$HOME/.ssh":/root/.ssh:ro \
		$(DOCKER_IMAGE):$(DOCKER_TAG) $(ARGS)

# Run cherry-go in Docker with GitHub token
# Usage: GITHUB_TOKEN=xxx make docker-run-token ARGS="sync myrepo"
docker-run-token:
	@docker run --rm \
		-v "$$(pwd)":/workspace \
		-e GITHUB_TOKEN \
		$(DOCKER_IMAGE):$(DOCKER_TAG) $(ARGS)

# Tag and push Docker image to registry
docker-push: docker-build
	@echo "Pushing Docker image to $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):$(DOCKER_TAG)..."
	@docker tag $(DOCKER_IMAGE):$(DOCKER_TAG) $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):$(DOCKER_TAG)
	@docker push $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):$(DOCKER_TAG)

# Remove local Docker images
docker-clean:
	@echo "Removing Docker images..."
	@docker rmi $(DOCKER_IMAGE):$(DOCKER_TAG) 2>/dev/null || true
	@docker rmi $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):$(DOCKER_TAG) 2>/dev/null || true

# ========== GORELEASER TARGETS ==========

# Check goreleaser configuration
goreleaser-check:
	@echo "Checking goreleaser configuration..."
	@goreleaser check

# Build snapshot with goreleaser (no publish)
goreleaser-build:
	@echo "Building with goreleaser (snapshot)..."
	@goreleaser build --snapshot --clean

# Simulate full release with goreleaser (no publish)
goreleaser-release-dry:
	@echo "Simulating release with goreleaser..."
	@goreleaser release --snapshot --clean

# Show help
help:
	@echo "Available targets:"
	@echo ""
	@echo "  Build & Test:"
	@echo "    build                   - Build the binary"
	@echo "    test                    - Run tests"
	@echo "    clean                   - Clean build artifacts (including dist/)"
	@echo "    deps                    - Install dependencies"
	@echo "    lint                    - Run linter"
	@echo "    fmt                     - Format code"
	@echo "    check                   - Run all checks (fmt, lint, test)"
	@echo ""
	@echo "  Installation:"
	@echo "    install                 - Install to GOPATH/bin"
	@echo "    install-local           - Install locally using installation script"
	@echo "    install-quick           - Quick local install"
	@echo "    uninstall               - Uninstall local installation"
	@echo ""
	@echo "  Docker:"
	@echo "    docker-build            - Build Docker image"
	@echo "    docker-build-multiarch  - Build with buildx for current platform"
	@echo "    docker-run ARGS=        - Run command in Docker (e.g., ARGS=\"status\")"
	@echo "    docker-run-ssh ARGS=    - Run with SSH keys mounted"
	@echo "    docker-run-token ARGS=  - Run with GITHUB_TOKEN"
	@echo "    docker-push             - Push image to ghcr.io"
	@echo "    docker-clean            - Remove local Docker images"
	@echo ""
	@echo "  GoReleaser:"
	@echo "    goreleaser-check        - Validate goreleaser config"
	@echo "    goreleaser-build        - Build snapshot (no publish)"
	@echo "    goreleaser-release-dry  - Simulate full release"
	@echo ""
	@echo "  Other:"
	@echo "    demo                    - Run demo script"
	@echo "    dev-setup               - Setup development environment"
	@echo "    help                    - Show this help"

