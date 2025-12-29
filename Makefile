.PHONY: help build test test-unit test-integration test-integration-nextcloud test-all lint clean docker-up docker-down docker-logs

# Variables
BINARY_NAME=gosynctasks
BUILD_DIR=.
GO_FILES=$(shell find . -name '*.go' -not -path "./vendor/*")
DOCKER_COMPOSE=docker-compose

# Default target
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the binary
	@echo "Building $(BINARY_NAME)..."
	go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/gosynctasks
	@echo "✓ Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

build-all: ## Build for all platforms
	@echo "Building for all platforms..."
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/gosynctasks
	GOOS=linux GOARCH=arm64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/gosynctasks
	GOOS=darwin GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/gosynctasks
	GOOS=darwin GOARCH=arm64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/gosynctasks
	GOOS=windows GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/gosynctasks
	@echo "✓ All platform builds complete"

test: test-unit ## Run unit tests

test-unit: ## Run unit tests only
	@echo "Running unit tests..."
	go test -v -race -coverprofile=coverage.out ./...
	@echo "✓ Unit tests complete"

test-coverage: test-unit ## Run unit tests and show coverage
	@echo "Generating coverage report..."
	go tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage report: coverage.html"

test-integration: docker-up ## Run mock backend integration tests
	@echo "Running mock backend integration tests..."
	@sleep 5
	go test -v -timeout 10m \
		./backend/integration_test.go \
		./backend/mockBackend.go \
		./backend/syncManager.go \
		./backend/taskManager.go
	@echo "✓ Mock integration tests complete"

test-integration-nextcloud: docker-up ## Run Nextcloud sync integration tests
	@echo "Waiting for Nextcloud to be ready..."
	@./scripts/wait-for-nextcloud.sh || (echo "ERROR: Nextcloud not ready" && exit 1)
	@echo "Running Nextcloud sync integration tests..."
	NEXTCLOUD_TEST_URL="nextcloud://admin:admin123@localhost:8080/" \
		go test -v -timeout 15m -tags=integration ./backend/sync_integration_test.go
	@echo "✓ Nextcloud integration tests complete"

test-all: test-unit test-integration test-integration-nextcloud ## Run all tests

lint: ## Run golangci-lint
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
		echo "✓ Linting complete"; \
	else \
		echo "ERROR: golangci-lint not installed"; \
		echo "Install: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		exit 1; \
	fi

fmt: ## Format code
	@echo "Formatting code..."
	gofmt -w $(GO_FILES)
	@echo "✓ Code formatted"

vet: ## Run go vet
	@echo "Running go vet..."
	go vet ./...
	@echo "✓ Vet complete"

docker-up: ## Start Nextcloud test server
	@echo "Starting Nextcloud test server..."
	@./scripts/start-test-server.sh
	@echo "✓ Nextcloud server started"

docker-down: ## Stop Nextcloud test server
	@echo "Stopping Nextcloud test server..."
	$(DOCKER_COMPOSE) down -v
	@echo "✓ Nextcloud server stopped"

docker-logs: ## Show Nextcloud logs
	$(DOCKER_COMPOSE) logs -f

docker-status: ## Show Docker container status
	@echo "Docker containers:"
	@docker ps -a --filter "name=nextcloud" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"

clean: ## Clean build artifacts
	@echo "Cleaning..."
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_NAME)-*
	rm -f coverage.out coverage.html
	rm -rf dist/
	@echo "✓ Clean complete"

clean-all: clean docker-down ## Clean everything including Docker volumes
	@echo "Removing Docker volumes..."
	docker volume rm gosynctasks_nextcloud_db gosynctasks_nextcloud_data 2>/dev/null || true
	@echo "✓ All cleaned"

install: build ## Install binary to $GOPATH/bin
	@echo "Installing $(BINARY_NAME) to $(GOPATH)/bin..."
	cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/
	@echo "✓ Installed to $(GOPATH)/bin/$(BINARY_NAME)"

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	go mod download
	go mod verify
	@echo "✓ Dependencies downloaded"

deps-update: ## Update dependencies
	@echo "Updating dependencies..."
	go get -u ./...
	go mod tidy
	@echo "✓ Dependencies updated"

security: ## Run security scan
	@echo "Running security scan..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo "Installing gosec..."; \
		go install github.com/securego/gosec/v2/cmd/gosec@latest; \
		gosec ./...; \
	fi
	@echo "Running vulnerability check..."
	@if command -v govulncheck >/dev/null 2>&1; then \
		govulncheck ./...; \
	else \
		echo "Installing govulncheck..."; \
		go install golang.org/x/vuln/cmd/govulncheck@latest; \
		govulncheck ./...; \
	fi
	@echo "✓ Security scan complete"

ci: lint test-unit build ## Run CI checks locally
	@echo "✓ All CI checks passed"

.DEFAULT_GOAL := help
