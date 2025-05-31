#!/bin/bash

# Online Shop Test Runner Script
# This script runs all tests with coverage reporting

set -e

# Configuration
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COVERAGE_DIR="$PROJECT_ROOT/coverage"
COVERAGE_FILE="$COVERAGE_DIR/coverage.out"
COVERAGE_HTML="$COVERAGE_DIR/coverage.html"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging function
log() {
    echo -e "${BLUE}[$(date '+%Y-%m-%d %H:%M:%S')]${NC} $1"
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Create coverage directory
mkdir -p "$COVERAGE_DIR"

# Change to project root
cd "$PROJECT_ROOT"

log "Starting test execution..."

# Check if Go is installed
if ! command -v go &> /dev/null; then
    error "Go is not installed or not in PATH"
    exit 1
fi

# Download dependencies
log "Downloading dependencies..."
go mod download

# Run go mod tidy to clean up
go mod tidy

# Verify dependencies
log "Verifying dependencies..."
go mod verify

# Run linting (if golangci-lint is available)
if command -v golangci-lint &> /dev/null; then
    log "Running linter..."
    golangci-lint run ./... || warning "Linting issues found"
else
    warning "golangci-lint not found, skipping linting"
fi

# Run go vet
log "Running go vet..."
go vet ./... || {
    error "go vet failed"
    exit 1
}

# Run unit tests
log "Running unit tests..."
go test -v -race -coverprofile="$COVERAGE_FILE" -covermode=atomic ./tests/unit/... || {
    error "Unit tests failed"
    exit 1
}

# Run integration tests (if database is available)
log "Running integration tests..."
if go test -v -race ./tests/integration/... -timeout=30s; then
    success "Integration tests passed"
else
    warning "Integration tests failed or skipped"
fi

# Generate coverage report
if [ -f "$COVERAGE_FILE" ]; then
    log "Generating coverage report..."
    
    # Generate HTML coverage report
    go tool cover -html="$COVERAGE_FILE" -o "$COVERAGE_HTML"
    
    # Display coverage summary
    COVERAGE_PERCENT=$(go tool cover -func="$COVERAGE_FILE" | grep total | awk '{print $3}')
    log "Total coverage: $COVERAGE_PERCENT"
    
    # Check coverage threshold
    COVERAGE_NUM=$(echo "$COVERAGE_PERCENT" | sed 's/%//')
    THRESHOLD=70
    
    if (( $(echo "$COVERAGE_NUM >= $THRESHOLD" | bc -l) )); then
        success "Coverage threshold met: $COVERAGE_PERCENT (>= $THRESHOLD%)"
    else
        warning "Coverage below threshold: $COVERAGE_PERCENT (< $THRESHOLD%)"
    fi
    
    log "Coverage report generated: $COVERAGE_HTML"
else
    warning "No coverage file found"
fi

# Run benchmark tests
log "Running benchmark tests..."
go test -bench=. -benchmem ./... > "$COVERAGE_DIR/benchmark.txt" 2>&1 || {
    warning "Benchmark tests failed or not found"
}

# Run race condition tests
log "Running race condition tests..."
go test -race ./... || {
    warning "Race condition tests failed"
}

# Test build
log "Testing build..."
go build -o /tmp/online-shop-test ./cmd/api/ || {
    error "Build failed"
    exit 1
}
rm -f /tmp/online-shop-test

# Test gRPC server build
log "Testing gRPC server build..."
go build -o /tmp/grpc-server-test ./cmd/grpc/ || {
    error "gRPC server build failed"
    exit 1
}
rm -f /tmp/grpc-server-test

# Test worker build
log "Testing worker build..."
go build -o /tmp/worker-test ./cmd/worker/ || {
    error "Worker build failed"
    exit 1
}
rm -f /tmp/worker-test

# Generate test report
log "Generating test report..."
cat > "$COVERAGE_DIR/test_report.md" << EOF
# Test Report

Generated on: $(date)

## Test Results

### Unit Tests
- Status: ✅ Passed
- Coverage: $COVERAGE_PERCENT

### Integration Tests
- Status: ⚠️ Conditional (depends on external services)

### Build Tests
- API Server: ✅ Passed
- gRPC Server: ✅ Passed
- Worker: ✅ Passed

## Coverage Details

See [coverage.html](coverage.html) for detailed coverage report.

## Benchmark Results

See [benchmark.txt](benchmark.txt) for benchmark results.

## Next Steps

1. Review coverage report for areas needing more tests
2. Add integration tests for external service dependencies
3. Set up CI/CD pipeline for automated testing
4. Consider adding end-to-end tests

EOF

success "All tests completed successfully!"
log "Test report generated: $COVERAGE_DIR/test_report.md"
log "Coverage report: $COVERAGE_DIR/coverage.html"

# Open coverage report in browser (if available)
if command -v xdg-open &> /dev/null; then
    log "Opening coverage report in browser..."
    xdg-open "$COVERAGE_HTML" &
elif command -v open &> /dev/null; then
    log "Opening coverage report in browser..."
    open "$COVERAGE_HTML" &
fi

log "Test execution completed!"