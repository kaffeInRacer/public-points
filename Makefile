# Online Shop Makefile

.PHONY: build test clean run-api run-grpc docker-build docker-up docker-down

# Build the application
build:
	go build -o bin/api cmd/api/main.go

# Run tests
test:
	go test ./...

# Clean build artifacts
clean:
	rm -rf bin/

# Run the API server
run-api:
	go run cmd/api/main.go

# Run the gRPC server
run-grpc:
	go run cmd/grpc/main.go

# Build Docker image
docker-build:
	docker build -t online-shop:latest .

# Start all services with Docker Compose
docker-up:
	docker-compose up -d

# Stop all services
docker-down:
	docker-compose down

# Start only infrastructure services
docker-infra:
	docker-compose up -d postgres redis elasticsearch

# View logs
docker-logs:
	docker-compose logs -f

# Install dependencies
deps:
	go mod download
	go mod tidy

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golangci-lint run

# Generate protobuf files
proto:
	protoc --go_out=. --go-grpc_out=. proto/*.proto

# Database migration
migrate:
	go run cmd/migrate/main.go

# Run development server with hot reload
dev:
	air

# Build for production
build-prod:
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/api cmd/api/main.go

# Help
help:
	@echo "Available commands:"
	@echo "  build       - Build the application"
	@echo "  test        - Run tests"
	@echo "  clean       - Clean build artifacts"
	@echo "  run-api     - Run the API server"
	@echo "  run-grpc    - Run the gRPC server"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-up   - Start all services with Docker Compose"
	@echo "  docker-down - Stop all services"
	@echo "  docker-infra - Start only infrastructure services"
	@echo "  docker-logs - View logs"
	@echo "  deps        - Install dependencies"
	@echo "  fmt         - Format code"
	@echo "  lint        - Lint code"
	@echo "  proto       - Generate protobuf files"
	@echo "  migrate     - Run database migration"
	@echo "  dev         - Run development server with hot reload"
	@echo "  build-prod  - Build for production"