# Makefile for Online Shop Application
# Provides convenient commands for building, testing, and running the application

.PHONY: help build test test-unit test-integration test-performance test-coverage clean deps lint format run docker-build docker-run setup

# Default target
.DEFAULT_GOAL := help

# Variables
APP_NAME := online-shop
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GO_VERSION := $(shell go version | awk '{print $$3}')
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME) -X main.goVersion=$(GO_VERSION)"

# Directories
BUILD_DIR := ./build
REPORTS_DIR := ./test_reports
COVERAGE_DIR := $(REPORTS_DIR)/coverage
DOCS_DIR := ./docs

# Test configuration
TEST_TIMEOUT := 30m
TEST_TAGS := 
COVERAGE_THRESHOLD := 80

# Docker configuration
DOCKER_IMAGE := $(APP_NAME):$(VERSION)
DOCKER_REGISTRY := 
DOCKER_TAG := latest

# Colors for output
RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[1;33m
BLUE := \033[0;34m
NC := \033[0m # No Color

## help: Show this help message
help:
	@echo "$(BLUE)Online Shop Application - Makefile Commands$(NC)"
	@echo ""
	@echo "$(YELLOW)Build Commands:$(NC)"
	@echo "  build          Build the application"
	@echo "  build-all      Build for all platforms"
	@echo "  clean          Clean build artifacts"
	@echo ""
	@echo "$(YELLOW)Development Commands:$(NC)"
	@echo "  deps           Download and tidy dependencies"
	@echo "  lint           Run linters"
	@echo "  format         Format code"
	@echo "  run            Run the application locally"
	@echo "  setup          Setup development environment"
	@echo ""
	@echo "$(YELLOW)Test Commands:$(NC)"
	@echo "  test           Run all tests"
	@echo "  test-unit      Run unit tests only"
	@echo "  test-integration Run integration tests only"
	@echo "  test-performance Run performance tests"
	@echo "  test-coverage  Run tests with coverage report"
	@echo "  test-bench     Run benchmark tests"
	@echo "  test-race      Run tests with race detection"
	@echo "  test-verbose   Run tests with verbose output"
	@echo ""
	@echo "$(YELLOW)Docker Commands:$(NC)"
	@echo "  docker-build   Build Docker image"
	@echo "  docker-run     Run application in Docker"
	@echo "  docker-up      Start all services with docker-compose"
	@echo "  docker-down    Stop all services"

## setup: Setup development environment
setup:
	@echo "$(BLUE)Setting up development environment...$(NC)"
	@go mod download
	@go mod tidy
	@mkdir -p $(BUILD_DIR) $(REPORTS_DIR) $(COVERAGE_DIR) $(DOCS_DIR)
	@echo "$(GREEN)Development environment setup complete$(NC)"

## deps: Download and tidy dependencies
deps:
	@echo "$(BLUE)Downloading dependencies...$(NC)"
	@go mod download
	@go mod tidy
	@go mod verify
	@echo "$(GREEN)Dependencies updated$(NC)"

## build: Build the application
build: deps
	@echo "$(BLUE)Building $(APP_NAME)...$(NC)"
	@mkdir -p $(BUILD_DIR)
	@go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME) ./cmd/server
	@echo "$(GREEN)Build complete: $(BUILD_DIR)/$(APP_NAME)$(NC)"

## clean: Clean build artifacts
clean:
	@echo "$(BLUE)Cleaning build artifacts...$(NC)"
	@rm -rf $(BUILD_DIR)
	@rm -rf $(REPORTS_DIR)
	@rm -rf bin/
	@go clean -cache -testcache -modcache
	@echo "$(GREEN)Clean complete$(NC)"

## lint: Run linters
lint:
	@echo "$(BLUE)Running linters...$(NC)"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "$(YELLOW)golangci-lint not installed, running basic checks...$(NC)"; \
		go vet ./...; \
		go fmt ./...; \
	fi
	@echo "$(GREEN)Linting complete$(NC)"

## format: Format code
format:
	@echo "$(BLUE)Formatting code...$(NC)"
	@go fmt ./...
	@echo "$(GREEN)Code formatting complete$(NC)"

## run: Run the application locally
run: build
	@echo "$(BLUE)Starting $(APP_NAME)...$(NC)"
	@$(BUILD_DIR)/$(APP_NAME)

## test: Run all tests
test:
	@echo "$(BLUE)Running all tests...$(NC)"
	@./scripts/test_runner.sh -t all -v
	@echo "$(GREEN)All tests complete$(NC)"

## test-unit: Run unit tests only
test-unit:
	@echo "$(BLUE)Running unit tests...$(NC)"
	@./scripts/test_runner.sh -t unit -v
	@echo "$(GREEN)Unit tests complete$(NC)"

## test-integration: Run integration tests only
test-integration:
	@echo "$(BLUE)Running integration tests...$(NC)"
	@./scripts/test_runner.sh -t integration -v
	@echo "$(GREEN)Integration tests complete$(NC)"

## test-coverage: Run tests with coverage report
test-coverage:
	@echo "$(BLUE)Running tests with coverage...$(NC)"
	@./scripts/test_runner.sh -t all -v -c
	@if [ -f $(COVERAGE_DIR)/coverage.html ]; then \
		echo "$(GREEN)Coverage report generated: $(COVERAGE_DIR)/coverage.html$(NC)"; \
	fi

## docker-build: Build Docker image
docker-build:
	@echo "$(BLUE)Building Docker image...$(NC)"
	@docker build -t $(DOCKER_IMAGE) .
	@docker tag $(DOCKER_IMAGE) $(APP_NAME):$(DOCKER_TAG)
	@echo "$(GREEN)Docker image built: $(DOCKER_IMAGE)$(NC)"

## docker-run: Run application in Docker
docker-run: docker-build
	@echo "$(BLUE)Running $(APP_NAME) in Docker...$(NC)"
	@docker run --rm -p 8080:8080 $(DOCKER_IMAGE)

## docker-up: Start all services with docker-compose
docker-up:
	@echo "$(BLUE)Starting services with docker-compose...$(NC)"
	@docker-compose up -d
	@echo "$(GREEN)Services started$(NC)"

## docker-down: Stop all services
docker-down:
	@echo "$(BLUE)Stopping services...$(NC)"
	@docker-compose down
	@echo "$(GREEN)Services stopped$(NC)"

# Legacy targets for backward compatibility
run-api: run
run-grpc: run

# Additional legacy targets
docker-infra:
	@echo "$(BLUE)Starting infrastructure services...$(NC)"
	@docker-compose up -d postgres redis elasticsearch
	@echo "$(GREEN)Infrastructure services started$(NC)"

docker-logs:
	@echo "$(BLUE)Viewing logs...$(NC)"
	@docker-compose logs -f

proto:
	@echo "$(BLUE)Generating protobuf files...$(NC)"
	@protoc --go_out=. --go-grpc_out=. proto/*.proto
	@echo "$(GREEN)Protobuf files generated$(NC)"

migrate:
	@echo "$(BLUE)Running database migration...$(NC)"
	@go run cmd/migrate/main.go
	@echo "$(GREEN)Migration completed$(NC)"

dev:
	@echo "$(BLUE)Starting development server...$(NC)"
	@if command -v air >/dev/null 2>&1; then \
		air; \
	else \
		echo "$(YELLOW)air not installed, running normally...$(NC)"; \
		make run; \
	fi

build-prod:
	@echo "$(BLUE)Building for production...$(NC)"
	@CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/api cmd/api/main.go
	@echo "$(GREEN)Production build complete$(NC)"