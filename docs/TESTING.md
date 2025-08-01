# Testing Guide

This guide covers testing strategies and best practices for the Gitea provider.

## Test Structure

The provider uses a multi-layered testing approach:

- **Unit Tests**: Test individual components in isolation
- **Integration Tests**: Test the provider against a real Gitea instance
- **E2E Tests**: Test the complete provider lifecycle in a Kubernetes cluster

## Unit Tests

Unit tests are located alongside the code they test, following Go conventions.

### Running Unit Tests

```bash
# Run all unit tests
make test

# Run tests with coverage
make test-coverage

# Run specific package tests
go test ./internal/clients/... -v

# Run with race detection
go test -race ./...
```

### Current Coverage

The provider maintains high test coverage:

- **internal/clients**: 74.1% coverage
- **Overall**: 74.1% coverage (target: 80%+)

### Coverage Report

Generate detailed coverage reports:

```bash
# Generate coverage profile
go test -coverprofile=coverage.out ./...

# View coverage in browser
go tool cover -html=coverage.out

# Show coverage by function
go tool cover -func=coverage.out
```

## Client Layer Testing

The `internal/clients` package has comprehensive test coverage including:

### Tested Operations
- ✅ **Repository Management**: Get, Create, Update, Delete
- ✅ **Organization Management**: Get, Create, Update, Delete  
- ✅ **User Management**: Get, Create, Update, Delete
- ✅ **Webhook Management**: Get, Create, Update, Delete (both repo and org)
- ✅ **Deploy Key Management**: Get, Create, Delete
- ✅ **Authentication**: Token-based authentication
- ✅ **Error Handling**: HTTP error responses, network failures

### Test Patterns

```go
func TestRepositoryOperations(t *testing.T) {
    // Create mock HTTP server
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Handle different API endpoints
        switch {
        case r.Method == "GET" && r.URL.Path == "/api/v1/repos/owner/repo":
            // Return mock repository data
        }
    }))
    defer server.Close()

    // Create test client
    client := &giteaClient{
        httpClient: &http.Client{},
        baseURL:    server.URL + "/api/v1",
        token:      "test-token",
    }

    // Test operations
    t.Run("GetRepository", func(t *testing.T) {
        repo, err := client.GetRepository(ctx, "owner", "repo")
        require.NoError(t, err)
        assert.Equal(t, "repo", repo.Name)
    })
}
```

## Integration Tests

Integration tests run against a live Gitea instance to ensure real-world compatibility.

### Setup

1. Start a local Gitea instance:
```bash
docker run -d \
  --name gitea-test \
  -p 3000:3000 \
  -e INSTALL_LOCK=true \
  -e SECRET_KEY=test-secret \
  -e DISABLE_REGISTRATION=false \
  gitea/gitea:latest
```

2. Create a test user and generate an access token via the web UI

3. Run integration tests:
```bash
GITEA_URL=http://localhost:3000 \
GITEA_TOKEN=your-token-here \
go test -v ./test/integration/... -tags=integration
```

### CI Integration

GitHub Actions automatically runs integration tests with a Gitea service:

```yaml
services:
  gitea:
    image: gitea/gitea:latest
    ports:
      - 3000:3000
    env:
      INSTALL_LOCK: true
      SECRET_KEY: test-secret
```

## E2E Tests

End-to-end tests validate the complete provider lifecycle in a Kubernetes environment.

### Prerequisites

- kind cluster
- Crossplane installed
- Provider built and loaded

### Running E2E Tests

```bash
# Set up test environment
make e2e-setup

# Run E2E tests
make e2e-test

# Clean up
make e2e-cleanup
```

## Test Development Guidelines

### Writing Good Tests

1. **Test Naming**: Use descriptive test names that explain what is being tested
2. **Test Structure**: Follow Arrange-Act-Assert pattern
3. **Test Isolation**: Each test should be independent and not rely on other tests
4. **Mock External Dependencies**: Use httptest.Server for HTTP clients
5. **Error Testing**: Test both success and failure scenarios

### Example Test Structure

```go
func TestClientOperation(t *testing.T) {
    // Arrange: Set up test dependencies
    server := setupMockServer()
    client := createTestClient(server.URL)
    
    // Define test cases
    tests := []struct {
        name    string
        input   interface{}
        want    interface{}
        wantErr bool
    }{
        {
            name: "successful operation",
            input: validInput,
            want: expectedOutput,
            wantErr: false,
        },
        {
            name: "error condition",
            input: invalidInput,
            want: nil,
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Act: Execute the operation
            got, err := client.Operation(tt.input)
            
            // Assert: Verify results
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                assert.Equal(t, tt.want, got)
            }
        })
    }
}
```

### Coverage Goals

- **Unit Tests**: 80%+ coverage for core functionality
- **Integration Tests**: Cover all major user workflows
- **E2E Tests**: Cover provider installation and basic resource lifecycle

### Test Data Management

- Use realistic but anonymized test data
- Store complex test fixtures in separate files
- Use builders or factories for creating test objects

## Debugging Tests

### Common Issues

1. **Flaky Tests**: Use deterministic test data and proper cleanup
2. **Timing Issues**: Use context with timeouts, avoid sleep statements
3. **Resource Leaks**: Ensure proper cleanup in test teardown

### Debug Commands

```bash
# Run single test with verbose output
go test -v -run TestSpecificTest ./internal/clients/

# Run with race detection
go test -race ./...

# Generate test coverage profile
go test -coverprofile=cover.out ./...
go tool cover -html=cover.out -o coverage.html
```

## Continuous Integration

The CI pipeline includes:

1. **Lint**: Code quality checks with golangci-lint
2. **Unit Tests**: Full test suite with coverage reporting
3. **Integration Tests**: Tests against live Gitea instance
4. **Build**: Verify all artifacts build successfully
5. **Security**: Vulnerability scanning and secret detection

### Coverage Reporting

Coverage results are automatically uploaded to Codecov for tracking trends and ensuring quality standards.