.PHONY: dev prod clean build build-frontend run deps test fmt lint help kill-dev image proto proto-clean proto-lint proto-format gen restore

DATA_DIR := ./data
BIN := build/discordiance
FRONTEND_DIR := web/discordiance
BUF_IMAGE := bufbuild/buf:latest
BUF_RUN := docker run --rm \
	--volume "$(shell pwd):/workspace" \
	--workdir /workspace \
	--user "$(shell id -u):$(shell id -g)" \
	--env HOME=/tmp \
	$(BUF_IMAGE)

# Development mode - runs backend and frontend concurrently
run:
	@echo "Starting development environment..."
	@mkdir -p $(DATA_DIR)
	@trap 'echo "Stopping all processes..."; kill $$(jobs -p) 2>/dev/null; wait; exit' INT TERM; \
	cd $(FRONTEND_DIR) && npm run dev & \
	FRONTEND_PID=$$!; \
	go run cmd/discordiance/main.go & \
	BACKEND_PID=$$!; \
	wait $$BACKEND_PID $$FRONTEND_PID

restore:
	@echo "Restoring saved dev db"
	cp ./dev/discordiance.db ./data/discordiance.db || echo "Couldnt resore, moving on"

dev: clean restore run

# Production build
prod: build-frontend
	@echo "Building for production..."
	@mkdir -p $(DATA_DIR)
	go build -o $(BIN) ./cmd/discordiance

# Build frontend for production
build-frontend:
	@echo "Building frontend..."
	cd $(FRONTEND_DIR) && npm run build

# Build backend with embedded frontend
build: build-frontend
	@echo "Building backend with embedded frontend..."
	go build -o $(BIN) ./cmd/discordiance

# Build Docker image
image:
	@echo "Building Docker image..."
	docker compose build

# Clean build artifacts and data
clean:
	@echo "Cleaning..."
	@rm -rf $(DATA_DIR) $(BIN) discordiance.db
	@echo "Clean complete!"

# Kill any orphaned dev processes
kill-dev:
	@echo "Killing orphaned development processes..."
	@pkill -f "npm run dev" || true
	@pkill -f "vite" || true
	@pkill -f "go run cmd/discordiance/main.go" || true
	@echo "Cleanup complete!"

# Install dependencies
deps:
	@echo "Installing Go dependencies..."
	go mod download
	@echo "Updating buf dependencies (using Docker)..."
	$(BUF_RUN) dep update
	@echo "Installing frontend dependencies..."
	cd $(FRONTEND_DIR) && npm install

# Run tests
test:
	@echo "Running tests..."
	go test ./...

# Format code
fmt:
	@echo "Formatting Go code..."
	go fmt ./...

# Lint code
lint: proto-lint
	@echo "Running vet..."
	go vet ./...

# Proto generation
proto:
	@echo "Generating protocol buffer code (using Docker)..."
	$(BUF_RUN) generate
	@echo "Proto generation complete!"

proto-clean:
	@echo "Cleaning generated proto files..."
	rm -rf pkg/proto
	rm -rf web/discordiance/src/lib/proto
	@echo "Proto files cleaned!"

proto-lint:
	@echo "Linting proto files (using Docker)..."
	$(BUF_RUN) lint || echo "Buf linting failed, but continuing."
	@echo "Proto linting complete!"

proto-format:
	@echo "Formatting proto files (using Docker)..."
	$(BUF_RUN) format -w
	@echo "Proto files formatted!"

gen: proto-clean proto

# Help
help:
	@echo "Available commands:"
	@echo "  make dev            - Clean and run in development mode (frontend + backend)"
	@echo "  make run            - Run frontend + backend concurrently"
	@echo "  make build          - Build standalone binary with embedded frontend"
	@echo "  make prod           - Build for production"
	@echo "  make image          - Build Docker image"
	@echo "  make clean          - Remove data and build artifacts"
	@echo "  make kill-dev       - Kill orphaned dev processes"
	@echo "  make deps           - Install all dependencies (Go + buf + npm)"
	@echo "  make test           - Run tests"
	@echo "  make fmt            - Format code"
	@echo "  make lint           - Lint code"
	@echo "  make gen            - Clean and regenerate proto code (via Docker)"
	@echo "  make proto          - Generate Go and TypeScript code from proto files (via Docker)"
	@echo "  make proto-clean    - Remove all generated proto files"
	@echo "  make proto-lint     - Lint proto files (via Docker)"
	@echo "  make proto-format   - Format proto files (via Docker)"
	@echo "  make help           - Show this help message"
