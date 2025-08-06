# Test Infrastructure

This package provides shared testing utilities for Crossplane Gitea Provider controller tests. It includes fixtures, builders, and orchestration frameworks to reduce code duplication and improve test maintainability.

## Overview

The test infrastructure consists of 6 core components:

1. **[TestFixtures](fixtures.go)** - Common test data and response builders
2. **[MockClientBuilder](fixtures.go)** - Gitea mock client with fluent interface  
3. **[K8sSecretBuilder](fixtures.go)** - Kubernetes secret creation utilities
4. **[K8sClientBuilder](fixtures.go)** - Fake Kubernetes client setup
5. **[ParameterBuilders](parameters.go)** - Consistent test parameter generation
6. **[TestSuite](suite.go)** - Test orchestration and assertion helpers

## Quick Start

```go
import "github.com/rossigee/provider-gitea/internal/controller/testing"

func TestMyController(t *testing.T) {
    // Create test fixtures
    fixtures := testing.NewTestFixtures()
    
    // Create mock client with expectations
    mockClient := testing.NewMockClient().
        ExpectMethod("CreateRepository", fixtures.RepositoryResponse(), nil).
        Build()
    
    // Create Kubernetes client with secrets
    kubeClient := testing.NewK8sClient().
        WithSecret(testing.NewSecret("test-secret", "default").
            WithPasswordData("testpass123").
            Build()).
        Build()
    
    // Use in your tests...
}
```

## Components

### TestFixtures

Provides common test data and response builders:

```go
fixtures := testing.NewTestFixtures()

// Access common test data
fixtures.TestUser      // "testuser"
fixtures.TestOrg       // "testorg" 
fixtures.TestRepo      // "testrepo"
fixtures.TestEmail     // "testuser@example.com"

// Generate response objects
repo := fixtures.RepositoryResponse()     // *clients.Repository
user := fixtures.UserResponse()           // *clients.User
org := fixtures.OrganizationResponse()    // *clients.Organization
```

### MockClientBuilder

Fluent interface for creating Gitea mock clients:

```go
mockClient := testing.NewMockClient().
    ExpectMethod("CreateRepository", expectedResponse, nil).
    ExpectMethod("GetRepository", existingResponse, nil).
    Build()
```

### Secret Builders

Create Kubernetes secrets for testing controllers that need secret access:

```go
// Password secret
passwordSecret := testing.NewSecret("user-password", "default").
    WithPasswordData("supersecret123").
    Build()

// Value secret  
valueSecret := testing.NewSecret("api-secret", "default").
    WithValueData("apikey123").
    Build()

// Custom data secret
customSecret := testing.NewSecret("custom-secret", "default").
    WithData("token", "token123").
    WithData("url", "https://api.example.com").
    Build()
```

### K8sClientBuilder  

Create fake Kubernetes clients with pre-loaded secrets:

```go
kubeClient := testing.NewK8sClient().
    WithSecret(passwordSecret).
    WithSecret(valueSecret).
    Build()
```

### Parameter Builders

Generate consistent test parameters:

```go
fixtures := testing.NewTestFixtures()

// Repository parameters
repoParams := fixtures.RepositoryParameters()
// Returns v1alpha1.RepositoryParameters with test data
```

### TestSuite

Test orchestration and helper functions:

```go
suite := testing.NewTestSuite(t).WithFixtures(fixtures)

// Assertion helpers
suite.AssertNoError(err)
suite.AssertError(err)  
suite.AssertErrorContains(err, "expected text")
```

## Examples

See [example_test.go](example_test.go) for complete working examples demonstrating:

- Basic test fixture usage
- Secret builder patterns
- Kubernetes client setup
- Mock client expectations

## Benefits

### Reduced Duplication
- Shared fixtures eliminate repetitive test setup code
- Common builders provide consistent test data across controllers

### Improved Maintainability  
- Centralized test infrastructure makes updates easier
- Changes to client types only require updates in one place

### Enhanced Readability
- Fluent interfaces provide clean, readable test setup
- Self-documenting builder patterns

### Comprehensive Coverage
- Supports all 23 controller types with their unique patterns
- Built-in support for both Gitea and Kubernetes client mocking

## Integration with Controller Tests

This infrastructure integrates seamlessly with existing controller tests:

```go
func TestRepository_Create_Successful(t *testing.T) {
    fixtures := testing.NewTestFixtures()
    
    // Create mock with expectations
    mockClient := testing.NewMockClient().
        ExpectMethod("CreateRepository", fixtures.RepositoryResponse(), nil).
        Build()
        
    // Create external client
    external := &external{client: mockClient}
    
    // Create test resource
    repo := &repositoryv1alpha1.Repository{
        Spec: repositoryv1alpha1.RepositorySpec{
            ForProvider: fixtures.RepositoryParameters(),
        },
    }
    
    // Test the operation
    result, err := external.Create(context.Background(), repo)
    
    // Verify results
    assert.NoError(t, err)
    assert.NotNil(t, result)
    mockClient.AssertExpectations(t)
}
```

## Development Guidelines

When adding new functionality to the test infrastructure:

1. **Keep it Simple** - Avoid over-engineering, focus on common use cases
2. **Maintain Consistency** - Follow established patterns for builders and fixtures
3. **Document Changes** - Update this README when adding new components
4. **Test the Infrastructure** - The test infrastructure itself should have tests
5. **Consider All Controllers** - Ensure new features work across all 23 controller types

## Architecture Decisions

### Why Builder Patterns?
- Fluent interfaces are more readable than large parameter lists
- Easy to add new options without breaking existing code
- Self-documenting through method names

### Why Separate Files?
- Clear separation of concerns (fixtures vs parameters vs orchestration)
- Easier to find and maintain specific functionality
- Avoids single large file that's difficult to navigate

### Why Simplified Types?
- Avoids complex struct field dependencies that cause lint errors
- Focuses on testing behavior rather than exact data structure matches
- Easier to maintain as client types evolve

This test infrastructure provides a solid foundation for maintainable, readable, and comprehensive controller tests across the entire Crossplane Gitea Provider codebase.