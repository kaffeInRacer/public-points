#!/bin/bash

# Test Runner Script for Online Shop Application
# This script provides comprehensive testing capabilities including unit tests,
# integration tests, performance tests, and test reporting.

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
TEST_CONFIG="$PROJECT_ROOT/tests/config/test_config.yaml"
REPORTS_DIR="$PROJECT_ROOT/test_reports"
COVERAGE_DIR="$REPORTS_DIR/coverage"
BENCHMARK_DIR="$REPORTS_DIR/benchmarks"

# Default values
TEST_TYPE="all"
VERBOSE=false
COVERAGE=true
BENCHMARKS=false
INTEGRATION=true
CLEANUP=true
PARALLEL=true
TIMEOUT="30m"
TAGS=""

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to show usage
show_usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Test Runner for Online Shop Application

OPTIONS:
    -t, --type TYPE         Test type: unit, integration, performance, all (default: all)
    -v, --verbose          Enable verbose output
    -c, --coverage         Enable coverage reporting (default: true)
    -b, --benchmarks       Run benchmark tests
    -i, --integration      Run integration tests (default: true)
    --no-cleanup          Skip cleanup after tests
    --no-parallel         Disable parallel test execution
    --timeout DURATION    Test timeout (default: 30m)
    --tags TAGS           Build tags for tests
    -h, --help            Show this help message

EXAMPLES:
    $0                              # Run all tests with default settings
    $0 -t unit -v                   # Run only unit tests with verbose output
    $0 -t integration --no-cleanup  # Run integration tests without cleanup
    $0 -b --timeout 60m             # Run benchmarks with 60 minute timeout
    $0 --tags "integration,redis"   # Run tests with specific build tags

TEST TYPES:
    unit         - Unit tests only
    integration  - Integration tests only
    performance  - Performance and load tests
    all          - All test types

EOF
}

# Function to parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -t|--type)
                TEST_TYPE="$2"
                shift 2
                ;;
            -v|--verbose)
                VERBOSE=true
                shift
                ;;
            -c|--coverage)
                COVERAGE=true
                shift
                ;;
            -b|--benchmarks)
                BENCHMARKS=true
                shift
                ;;
            -i|--integration)
                INTEGRATION=true
                shift
                ;;
            --no-cleanup)
                CLEANUP=false
                shift
                ;;
            --no-parallel)
                PARALLEL=false
                shift
                ;;
            --timeout)
                TIMEOUT="$2"
                shift 2
                ;;
            --tags)
                TAGS="$2"
                shift 2
                ;;
            -h|--help)
                show_usage
                exit 0
                ;;
            *)
                print_error "Unknown option: $1"
                show_usage
                exit 1
                ;;
        esac
    done
}

# Function to check prerequisites
check_prerequisites() {
    print_status "Checking prerequisites..."
    
    # Check if Go is installed
    if ! command -v go &> /dev/null; then
        print_error "Go is not installed or not in PATH"
        exit 1
    fi
    
    # Check Go version
    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    print_status "Go version: $GO_VERSION"
    
    # Check if Docker is available for integration tests
    if [[ "$INTEGRATION" == "true" || "$TEST_TYPE" == "integration" || "$TEST_TYPE" == "all" ]]; then
        if ! command -v docker &> /dev/null; then
            print_warning "Docker is not available. Integration tests will be skipped."
            INTEGRATION=false
        else
            print_status "Docker is available for integration tests"
        fi
    fi
    
    # Check if test configuration exists
    if [[ ! -f "$TEST_CONFIG" ]]; then
        print_warning "Test configuration not found at $TEST_CONFIG"
    fi
    
    print_success "Prerequisites check completed"
}

# Function to setup test environment
setup_test_env() {
    print_status "Setting up test environment..."
    
    # Create reports directory
    mkdir -p "$REPORTS_DIR"
    mkdir -p "$COVERAGE_DIR"
    mkdir -p "$BENCHMARK_DIR"
    
    # Set environment variables
    export GO_ENV=test
    export CONFIG_PATH="$TEST_CONFIG"
    
    # Clean previous test artifacts
    if [[ "$CLEANUP" == "true" ]]; then
        rm -rf "$REPORTS_DIR"/*.xml
        rm -rf "$REPORTS_DIR"/*.json
        rm -rf "$REPORTS_DIR"/*.html
        rm -rf "$COVERAGE_DIR"/*
        rm -rf "$BENCHMARK_DIR"/*
    fi
    
    print_success "Test environment setup completed"
}

# Function to build test flags
build_test_flags() {
    local flags=""
    
    if [[ "$VERBOSE" == "true" ]]; then
        flags="$flags -v"
    fi
    
    if [[ "$PARALLEL" == "true" ]]; then
        flags="$flags -parallel 4"
    fi
    
    if [[ "$COVERAGE" == "true" ]]; then
        flags="$flags -coverprofile=$COVERAGE_DIR/coverage.out"
        flags="$flags -covermode=atomic"
    fi
    
    if [[ -n "$TAGS" ]]; then
        flags="$flags -tags=$TAGS"
    fi
    
    flags="$flags -timeout=$TIMEOUT"
    flags="$flags -race"
    
    echo "$flags"
}

# Function to run unit tests
run_unit_tests() {
    print_status "Running unit tests..."
    
    local flags=$(build_test_flags)
    local test_pattern="./tests/unit/..."
    
    if [[ "$COVERAGE" == "true" ]]; then
        go test $flags -outputdir="$COVERAGE_DIR" $test_pattern
    else
        go test $flags $test_pattern
    fi
    
    if [[ $? -eq 0 ]]; then
        print_success "Unit tests passed"
    else
        print_error "Unit tests failed"
        return 1
    fi
}

# Function to run integration tests
run_integration_tests() {
    if [[ "$INTEGRATION" != "true" ]]; then
        print_warning "Integration tests skipped"
        return 0
    fi
    
    print_status "Running integration tests..."
    
    local flags=$(build_test_flags)
    local test_pattern="./tests/integration/..."
    
    # Set integration test specific environment
    export TESTCONTAINERS_RYUK_DISABLED=true
    
    if [[ "$COVERAGE" == "true" ]]; then
        go test $flags -outputdir="$COVERAGE_DIR" $test_pattern
    else
        go test $flags $test_pattern
    fi
    
    if [[ $? -eq 0 ]]; then
        print_success "Integration tests passed"
    else
        print_error "Integration tests failed"
        return 1
    fi
}

# Function to run performance tests
run_performance_tests() {
    print_status "Running performance tests..."
    
    local flags="-v -timeout=$TIMEOUT"
    if [[ -n "$TAGS" ]]; then
        flags="$flags -tags=$TAGS"
    fi
    
    # Run performance tests with specific pattern
    go test $flags -run="Performance" ./tests/...
    
    if [[ $? -eq 0 ]]; then
        print_success "Performance tests passed"
    else
        print_error "Performance tests failed"
        return 1
    fi
}

# Function to run benchmark tests
run_benchmark_tests() {
    if [[ "$BENCHMARKS" != "true" ]]; then
        return 0
    fi
    
    print_status "Running benchmark tests..."
    
    local flags="-v -timeout=$TIMEOUT -bench=. -benchmem"
    if [[ -n "$TAGS" ]]; then
        flags="$flags -tags=$TAGS"
    fi
    
    # Run benchmarks and save results
    go test $flags ./tests/... > "$BENCHMARK_DIR/benchmarks.txt" 2>&1
    
    if [[ $? -eq 0 ]]; then
        print_success "Benchmark tests completed"
    else
        print_error "Benchmark tests failed"
        return 1
    fi
}

# Function to generate test reports
generate_reports() {
    print_status "Generating test reports..."
    
    # Generate coverage report if coverage was enabled
    if [[ "$COVERAGE" == "true" && -f "$COVERAGE_DIR/coverage.out" ]]; then
        print_status "Generating coverage reports..."
        
        # Generate HTML coverage report
        go tool cover -html="$COVERAGE_DIR/coverage.out" -o "$COVERAGE_DIR/coverage.html"
        
        # Generate coverage summary
        go tool cover -func="$COVERAGE_DIR/coverage.out" > "$COVERAGE_DIR/coverage.txt"
        
        # Extract coverage percentage
        local coverage_pct=$(go tool cover -func="$COVERAGE_DIR/coverage.out" | grep "total:" | awk '{print $3}')
        print_status "Total coverage: $coverage_pct"
        
        # Check coverage threshold
        local threshold="80.0%"
        local coverage_num=$(echo "$coverage_pct" | sed 's/%//')
        local threshold_num=$(echo "$threshold" | sed 's/%//')
        
        if (( $(echo "$coverage_num >= $threshold_num" | bc -l) )); then
            print_success "Coverage threshold met: $coverage_pct >= $threshold"
        else
            print_warning "Coverage below threshold: $coverage_pct < $threshold"
        fi
    fi
    
    # Generate test summary
    cat > "$REPORTS_DIR/test_summary.json" << EOF
{
    "timestamp": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
    "test_type": "$TEST_TYPE",
    "coverage_enabled": $COVERAGE,
    "benchmarks_enabled": $BENCHMARKS,
    "integration_enabled": $INTEGRATION,
    "parallel_enabled": $PARALLEL,
    "timeout": "$TIMEOUT",
    "tags": "$TAGS",
    "reports_directory": "$REPORTS_DIR"
}
EOF
    
    print_success "Test reports generated in $REPORTS_DIR"
}

# Function to cleanup test environment
cleanup_test_env() {
    if [[ "$CLEANUP" != "true" ]]; then
        return 0
    fi
    
    print_status "Cleaning up test environment..."
    
    # Clean up Docker containers if integration tests were run
    if [[ "$INTEGRATION" == "true" ]]; then
        print_status "Cleaning up test containers..."
        docker container prune -f --filter "label=org.testcontainers=true" 2>/dev/null || true
        docker network prune -f --filter "label=org.testcontainers=true" 2>/dev/null || true
        docker volume prune -f --filter "label=org.testcontainers=true" 2>/dev/null || true
    fi
    
    # Clean up temporary files
    rm -rf /tmp/test_* 2>/dev/null || true
    
    print_success "Cleanup completed"
}

# Function to run all tests
run_all_tests() {
    local exit_code=0
    
    # Run unit tests
    if ! run_unit_tests; then
        exit_code=1
    fi
    
    # Run integration tests
    if ! run_integration_tests; then
        exit_code=1
    fi
    
    # Run performance tests
    if ! run_performance_tests; then
        exit_code=1
    fi
    
    # Run benchmark tests
    if ! run_benchmark_tests; then
        exit_code=1
    fi
    
    return $exit_code
}

# Main function
main() {
    local start_time=$(date +%s)
    
    print_status "Starting test runner for Online Shop Application"
    print_status "Test type: $TEST_TYPE"
    print_status "Configuration: $TEST_CONFIG"
    
    # Setup
    check_prerequisites
    setup_test_env
    
    # Run tests based on type
    local exit_code=0
    case "$TEST_TYPE" in
        "unit")
            if ! run_unit_tests; then
                exit_code=1
            fi
            ;;
        "integration")
            if ! run_integration_tests; then
                exit_code=1
            fi
            ;;
        "performance")
            if ! run_performance_tests; then
                exit_code=1
            fi
            if ! run_benchmark_tests; then
                exit_code=1
            fi
            ;;
        "all")
            if ! run_all_tests; then
                exit_code=1
            fi
            ;;
        *)
            print_error "Unknown test type: $TEST_TYPE"
            show_usage
            exit 1
            ;;
    esac
    
    # Generate reports
    generate_reports
    
    # Cleanup
    cleanup_test_env
    
    # Summary
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    if [[ $exit_code -eq 0 ]]; then
        print_success "All tests completed successfully in ${duration}s"
        print_status "Reports available in: $REPORTS_DIR"
    else
        print_error "Some tests failed. Check the output above for details."
        print_status "Reports available in: $REPORTS_DIR"
    fi
    
    exit $exit_code
}

# Parse arguments and run main function
parse_args "$@"
main