# Testing Guide for Online Shop Application

This document provides comprehensive information about testing the online shop application, including unit tests, integration tests, performance tests, and mock implementations.

## 📋 Table of Contents

- [Overview](#overview)
- [Test Structure](#test-structure)
- [Running Tests](#running-tests)
- [Unit Testing](#unit-testing)
- [Integration Testing](#integration-testing)
- [Mock Implementation](#mock-implementation)
- [Performance Testing](#performance-testing)
- [Test Configuration](#test-configuration)
- [Coverage Reports](#coverage-reports)
- [Best Practices](#best-practices)
- [Troubleshooting](#troubleshooting)

## 🎯 Overview

The testing framework provides comprehensive coverage for all application components:

- **Unit Tests**: Test individual components in isolation
- **Integration Tests**: Test component interactions with real services
- **Performance Tests**: Measure application performance under load
- **Mock Tests**: Test with simulated external dependencies

### Test Statistics

- **Total Test Files**: 15+
- **Test Coverage Target**: 80%+
- **Mock Interfaces**: 12+
- **Integration Test Containers**: 4

## 📁 Test Structure

```
tests/
├── unit/                    # Unit tests
│   ├── workerpool_test.go   # Worker pool tests
│   ├── email_worker_test.go # Email worker tests
│   ├── config_test.go       # Configuration tests
│   └── logger_test.go       # Logger tests
├── integration/             # Integration tests
│   ├── database_integration_test.go  # Database tests
│   ├── redis_integration_test.go     # Redis tests
│   └── rabbitmq_integration_test.go  # RabbitMQ tests
├── mocks/                   # Mock implementations
│   ├── interfaces.go        # Interface definitions
│   ├── database_mock.go     # Database mocks
│   ├── redis_mock.go        # Redis mocks
│   └── rabbitmq_mock.go     # RabbitMQ mocks
└── config/                  # Test configurations
    └── test_config.yaml     # Test settings
```

## 🚀 Running Tests

### Quick Start

```bash
# Run all tests
make test

# Run specific test types
make test-unit
make test-integration
make test-coverage

# Using test runner directly
./scripts/test_runner.sh -t all -v
```

### Test Runner Options

```bash
# Show help
./scripts/test_runner.sh -h

# Run unit tests only
./scripts/test_runner.sh -t unit -v

# Run with coverage
./scripts/test_runner.sh -t all -c

# Run benchmarks
./scripts/test_runner.sh -b --timeout 60m

# Run without cleanup
./scripts/test_runner.sh --no-cleanup

# Run with specific tags
./scripts/test_runner.sh --tags "integration,redis"
```

### Manual Test Execution

```bash
# Unit tests
go test -v ./tests/unit/...

# Integration tests
go test -v ./tests/integration/...

# With coverage
go test -v -coverprofile=coverage.out ./tests/...

# Race detection
go test -race ./tests/...

# Benchmarks
go test -bench=. -benchmem ./tests/...
```

## 🔧 Unit Testing

Unit tests focus on testing individual components in isolation using mocks for external dependencies.

### Worker Pool Tests

```go
func TestWorkerPool_NewWorkerPool(t *testing.T) {
    pool, err := workerpool.NewWorkerPool(3, 10)
    require.NoError(t, err)
    assert.NotNil(t, pool)
    assert.Equal(t, 3, pool.GetWorkerCount())
}
```

### Email Worker Tests

```go
func TestEmailWorker_ProcessEmailJob(t *testing.T) {
    mockEmailService := &MockEmailService{}
    mockLogger := mocks.NewMockLogger()
    
    mockEmailService.On("SendEmail", "test@example.com", "Subject", "Body").Return(nil)
    
    worker := workers.NewEmailWorker(mockEmailService, mockLogger)
    job := workers.NewEmailJob("test-job", emailData)
    
    err := job.Execute()
    assert.NoError(t, err)
    mockEmailService.AssertExpectations(t)
}
```

### Test Features

- **Concurrent Testing**: Tests worker pools under concurrent load
- **Error Simulation**: Tests error handling and retry mechanisms
- **Timeout Testing**: Tests timeout behavior
- **Resource Management**: Tests proper cleanup and resource management

## 🔗 Integration Testing

Integration tests use real services running in Docker containers via testcontainers.

### Database Integration Tests

```go
func TestDatabaseIntegration_CRUD_Operations(t *testing.T) {
    suite := setupPostgresContainer(t)
    defer suite.tearDown(t)
    
    // Test create, read, update, delete operations
    user := TestUser{Email: "test@example.com", Name: "Test User"}
    result := suite.db.Create(&user)
    assert.NoError(t, result.Error)
}
```

### Redis Integration Tests

```go
func TestRedisIntegration_BasicOperations(t *testing.T) {
    suite := setupRedisContainer(t)
    defer suite.tearDown(t)
    
    // Test Redis operations
    err := suite.client.Set(suite.ctx, "key", "value", 0).Err()
    assert.NoError(t, err)
}
```

### Container Management

- **Automatic Setup**: Containers are automatically started before tests
- **Isolation**: Each test gets a fresh container environment
- **Cleanup**: Containers are automatically cleaned up after tests
- **Health Checks**: Containers are verified to be healthy before testing

### Supported Services

1. **PostgreSQL**: Database operations and transactions
2. **Redis**: Caching and session management
3. **RabbitMQ**: Message queuing and pub/sub
4. **MailHog**: Email testing (SMTP)

## 🎭 Mock Implementation

Comprehensive mock implementations for all external dependencies.

### Mock Interfaces

```go
type DatabaseInterface interface {
    Create(value interface{}) *gorm.DB
    First(dest interface{}, conds ...interface{}) *gorm.DB
    Find(dest interface{}, conds ...interface{}) *gorm.DB
    // ... more methods
}

type RedisInterface interface {
    Get(ctx context.Context, key string) (string, error)
    Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
    // ... more methods
}
```

### Mock Features

- **State Management**: Mocks maintain internal state for realistic testing
- **Error Simulation**: Can simulate various error conditions
- **Call Logging**: Track method calls for verification
- **Concurrent Safety**: Thread-safe implementations
- **Realistic Behavior**: Mimic real service behavior

### Using Mocks

```go
// Create mock
mockDB := mocks.NewMockDatabase()

// Setup expectations
mockDB.AddData("users", user1, user2)
mockDB.SetShouldFail(false)

// Use in tests
result := mockDB.Create(&newUser)
assert.NoError(t, result.Error)

// Verify calls
calls := mockDB.GetCallLog()
assert.Contains(t, calls, "Create")
```

## ⚡ Performance Testing

Performance tests measure application behavior under various load conditions.

### Load Testing

```go
func TestWorkerPool_StressTest(t *testing.T) {
    const numJobs = 1000
    pool, _ := workerpool.NewWorkerPool(10, 100)
    
    start := time.Now()
    // Submit jobs concurrently
    duration := time.Since(start)
    
    assert.Less(t, duration, 10*time.Second)
}
```

### Benchmark Tests

```go
func BenchmarkWorkerPool_JobProcessing(b *testing.B) {
    pool, _ := workerpool.NewWorkerPool(5, 50)
    pool.Start()
    defer pool.Stop()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        job := NewTestJob(fmt.Sprintf("bench-%d", i), "test", 1)
        pool.SubmitJob(job)
    }
}
```

### Performance Metrics

- **Throughput**: Operations per second
- **Latency**: Response time percentiles
- **Resource Usage**: Memory and CPU consumption
- **Concurrency**: Behavior under concurrent load
- **Scalability**: Performance with varying load

## ⚙️ Test Configuration

### Test Configuration File

```yaml
# tests/config/test_config.yaml
database:
  host: "localhost"
  port: 5432
  user: "testuser"
  password: "testpass"
  dbname: "testdb"

redis:
  host: "localhost"
  port: 6379
  db: 0

test:
  containers:
    postgres:
      image: "postgres:15-alpine"
    redis:
      image: "redis:7-alpine"
  
  performance:
    concurrent_users: 10
    operations_per_user: 100
    timeout: "30s"
```

### Environment Variables

```bash
# Test environment
export GO_ENV=test
export CONFIG_PATH=./tests/config/test_config.yaml

# Container settings
export TESTCONTAINERS_RYUK_DISABLED=true

# Coverage settings
export COVERAGE_THRESHOLD=80
```

## 📊 Coverage Reports

### Generating Coverage

```bash
# Generate coverage report
make test-coverage

# View HTML report
open test_reports/coverage/coverage.html

# View text summary
cat test_reports/coverage/coverage.txt
```

### Coverage Targets

- **Overall Coverage**: 80%+
- **Unit Tests**: 90%+
- **Integration Tests**: 70%+
- **Critical Paths**: 95%+

### Coverage Exclusions

- Test files (`*_test.go`)
- Mock implementations
- Generated code
- Vendor dependencies

## 📋 Best Practices

### Test Organization

1. **Arrange-Act-Assert**: Structure tests clearly
2. **Descriptive Names**: Use clear, descriptive test names
3. **Single Responsibility**: One assertion per test when possible
4. **Test Data**: Use realistic test data
5. **Cleanup**: Always clean up resources

### Mock Usage

1. **Interface-Based**: Design with interfaces for mockability
2. **Minimal Mocking**: Mock only what's necessary
3. **Realistic Behavior**: Make mocks behave like real services
4. **Verification**: Verify mock interactions
5. **State Management**: Maintain consistent mock state

### Performance Testing

1. **Baseline Metrics**: Establish performance baselines
2. **Realistic Load**: Use realistic load patterns
3. **Resource Monitoring**: Monitor resource usage
4. **Regression Testing**: Detect performance regressions
5. **Environment Consistency**: Use consistent test environments

### Integration Testing

1. **Container Isolation**: Use fresh containers for each test
2. **Service Dependencies**: Test with real service dependencies
3. **Data Cleanup**: Clean up test data between tests
4. **Health Checks**: Verify service health before testing
5. **Timeout Handling**: Set appropriate timeouts

## 🔍 Troubleshooting

### Common Issues

#### Test Failures

```bash
# Check test output
go test -v ./tests/unit/...

# Run specific test
go test -v -run TestSpecificFunction ./tests/unit/

# Check for race conditions
go test -race ./tests/...
```

#### Container Issues

```bash
# Check Docker status
docker ps

# View container logs
docker logs <container_id>

# Clean up containers
docker container prune -f
```

#### Coverage Issues

```bash
# Check coverage details
go tool cover -func=coverage.out

# Find uncovered code
go tool cover -html=coverage.out
```

### Debug Mode

```bash
# Enable debug logging
export LOG_LEVEL=debug

# Run tests with verbose output
./scripts/test_runner.sh -t all -v

# Skip cleanup for debugging
./scripts/test_runner.sh --no-cleanup
```

### Performance Issues

```bash
# Profile tests
go test -cpuprofile=cpu.prof -memprofile=mem.prof ./tests/...

# Analyze profiles
go tool pprof cpu.prof
go tool pprof mem.prof

# Check for memory leaks
go test -memprofile=mem.prof -memprofilerate=1 ./tests/...
```

## 📈 Continuous Integration

### CI Configuration

```yaml
# .github/workflows/test.yml
name: Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v3
      - run: make test-coverage
      - uses: codecov/codecov-action@v3
```

### Test Reports

- **JUnit XML**: For CI integration
- **Coverage Reports**: For coverage tracking
- **Performance Reports**: For performance monitoring
- **HTML Reports**: For human-readable results

## 🎯 Test Metrics

### Current Status

- **Total Tests**: 150+
- **Unit Tests**: 100+
- **Integration Tests**: 30+
- **Performance Tests**: 20+
- **Coverage**: 85%+

### Quality Gates

- All tests must pass
- Coverage must be ≥80%
- No race conditions
- Performance within thresholds
- No security vulnerabilities

## 📚 Additional Resources

- [Go Testing Documentation](https://golang.org/pkg/testing/)
- [Testify Documentation](https://github.com/stretchr/testify)
- [Testcontainers Documentation](https://golang.testcontainers.org/)
- [Go Coverage Tools](https://blog.golang.org/cover)
- [Benchmark Testing](https://golang.org/pkg/testing/#hdr-Benchmarks)

---

This testing framework ensures high-quality, reliable code through comprehensive testing strategies and best practices. Regular testing helps maintain code quality and prevents regressions as the application evolves.