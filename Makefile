.PHONY: dev prod clean build run deps test fmt lint proto proto-clean proto-lint proto-format gen

DATA_DIR := ./data
BIN := build/discordiance
BUF_IMAGE := bufbuild/buf:latest
BUF_RUN := docker run --rm \
	--volume "$(shell pwd):/workspace" \
	--workdir /workspace \
	--user "$(shell id -u):$(shell id -g)" \
	--env HOME=/tmp \
	$(BUF_IMAGE)

# Development mode
run:
	@echo "Starting development environment..."
	@mkdir -p $(DATA_DIR)
	go run cmd/discordiance/main.go

dev: clean run

# Build backend
build:
	@echo "Building backend..."
	go build -o $(BIN) ./cmd/discordiance


# Clean build artifacts and data
clean:
	@echo "Cleaning..."
	@rm -rf $(BIN)
	@echo "Clean complete!"

# Install dependencies
deps:
	@echo "Installing Go dependencies..."
	go mod download
	@echo "Updating buf dependencies (using Docker)..."
	$(BUF_RUN) dep update

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

gen: proto

# Proto generation
proto:
	@echo "Generating protocol buffer code (using Docker)..."
	$(BUF_RUN) generate
	@echo "Proto generation complete!"

proto-clean:
	@echo "Cleaning generated proto files..."
	rm -rf pkg/proto
	@echo "Proto files cleaned!"

proto-lint:
	@echo "Linting proto files (using Docker)..."
	$(BUF_RUN) lint || echo "Buf linting failed, but continuing."
	@echo "Proto linting complete!"

proto-format:
	@echo "Formatting proto files (using Docker)..."
	$(BUF_RUN) format -w
	@echo "Proto files formatted!"

