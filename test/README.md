# Provider Gitea Test Suite

This directory contains comprehensive test suites for the Gitea Crossplane provider, covering unit tests, integration tests, and end-to-end testing scenarios.

## Test Structure

```
test/
├── integration/          # Integration tests for example validation and security
│   ├── examples_test.go   # Validates all example manifests
│   └── security_test.go   # Security validation tests
├── e2e/                  # End-to-end workflow tests
│   └── complete_workflow_test.go  # Complete enterprise setup workflow
└── README.md             # This file
```

## Running Tests

### Unit Tests
Unit tests are located alongside the source code in `internal/controller/*/` directories:

```bash
# Run all unit tests
make test

# Run specific controller tests
go test ./internal/controller/action/...
go test ./internal/controller/runner/...
go test ./internal/controller/adminuser/...
```

### Integration Tests
Integration tests validate example manifests and security configurations:

```bash
# Run integration tests
go test ./test/integration/...

# Run specific integration test suites
go test ./test/integration/ -run TestExampleManifests
go test ./test/integration/ -run TestBranchProtectionSecurity
go test ./test/integration/ -run TestSSHKeySecurity
```

### End-to-End Tests
E2E tests require a Kubernetes cluster and validate complete workflows:

```bash
# Run E2E tests (requires Kubernetes cluster)
go test ./test/e2e/... -timeout 10m

# Skip E2E tests in short mode
go test ./test/e2e/... -short
```

## Test Categories

### 1. Unit Tests (`internal/controller/*/`)

**Purpose**: Test individual controller logic and external client interactions.

**Coverage**:
- Controller Observe/Create/Update/Delete operations
- External name parsing and validation
- Error handling and edge cases
- Mock client interactions

**Files**:
- `action_test.go` - Action controller tests
- `runner_test.go` - Runner controller tests  
- `adminuser_test.go` - AdminUser controller tests

### 2. Integration Tests (`test/integration/`)

**Purpose**: Validate example manifests and security configurations.

**Coverage**:
- Example manifest syntax validation
- Security configuration assessment
- SSH key format validation
- Access token scope security
- Secret handling validation

**Key Test Suites**:

#### Example Validation (`examples_test.go`)
- **TestExampleManifests**: Validates all example YAML files can be parsed
- **TestExampleCompleteness**: Ensures all resources have examples
- **TestExampleSecretRefs**: Validates secret reference formats

#### Security Validation (`security_test.go`)
- **TestBranchProtectionSecurity**: Evaluates branch protection security settings
- **TestSSHKeySecurity**: Validates SSH key formats and security
- **TestAccessTokenSecurity**: Assesses access token scope security
- **TestOrganizationMemberSecurity**: Validates membership configurations
- **TestSecretHandling**: Ensures proper secret encoding
- **TestRunnerSecurity**: Evaluates runner security configurations

### 3. End-to-End Tests (`test/e2e/`)

**Purpose**: Test complete workflows and resource dependencies.

**Coverage**:
- Complete enterprise setup workflow
- Resource creation order and dependencies
- Example manifest application to real cluster
- Performance scenarios under load

**Key Test Suites**:

#### Complete Workflow (`complete_workflow_test.go`)
- **TestCompleteEnterpriseWorkflow**: Tests full enterprise setup in phases:
  1. Create admin users and service accounts
  2. Setup security (branch protection, SSH keys, tokens)
  3. Configure CI/CD (runners, actions, secrets)
  4. Validate complete setup
- **TestWorkflowDependencies**: Documents and validates resource dependencies
- **TestExampleValidation**: Applies examples to test cluster
- **TestPerformanceScenarios**: Tests resource creation under load

## Test Data and Fixtures

### Mock Implementations
Each controller test includes mock client implementations:
- `MockActionClient` - Mocks Gitea Actions API
- `MockRunnerClient` - Mocks Gitea Runner API
- `MockAdminUserClient` - Mocks Gitea Admin API

### Test Examples
Tests use realistic example data:
- Valid SSH keys for key validation tests
- Secure branch protection configurations
- Enterprise-grade CI/CD pipelines
- Proper secret references and encoding

## Security Testing

### Branch Protection Security
Evaluates protection rules based on:
- Required approvals (minimum 1, recommended 2+)
- Status check requirements
- Signed commit requirements
- Protected file patterns
- Push restrictions

### SSH Key Security
Validates:
- Key format (RSA, Ed25519, ECDSA)
- Key length and strength
- Proper encoding
- Email attribution

### Access Token Security
Assesses:
- Scope minimization (read vs write)
- Admin permission restrictions
- Token lifecycle management
- Secret storage

### Runner Security
Evaluates:
- Scope restrictions (repository > organization > system)
- Label-based risk assessment
- Privileged access detection

## Test Configuration

### Environment Variables
```bash
# Skip E2E tests
TEST_SHORT=true

# Gitea test instance (for integration tests)
GITEA_URL=http://localhost:3000
GITEA_TOKEN=test-token

# Kubernetes test cluster (for E2E tests)
KUBECONFIG=/path/to/test/kubeconfig
```

### Test Timeouts
- Unit tests: 10 seconds per test
- Integration tests: 30 seconds per test
- E2E tests: 10 minutes total

### Test Dependencies
```go
// Required for all tests
github.com/crossplane/crossplane-runtime/pkg/test
github.com/google/go-cmp/cmp

// Integration tests
k8s.io/apimachinery/pkg/util/yaml
k8s.io/apimachinery/pkg/apis/meta/v1/unstructured

// E2E tests
sigs.k8s.io/controller-runtime/pkg/envtest
sigs.k8s.io/controller-runtime/pkg/client
```

## Adding New Tests

### For New Controllers
1. Create `*_test.go` file alongside controller
2. Implement mock client interface
3. Test all CRUD operations and error cases
4. Add external name parsing tests

### For New Examples
1. Add example to `test/integration/examples_test.go`
2. Include validation function for resource-specific checks
3. Add to `TestExampleCompleteness` required examples list

### For New Security Features
1. Add security evaluation function to `security_test.go`
2. Include test cases for secure and insecure configurations
3. Document security criteria and scoring

### For New E2E Scenarios
1. Add test function to `complete_workflow_test.go`
2. Include setup, execution, and validation phases
3. Add resource dependencies to dependency map

## Test Maintenance

### Regular Tasks
- Update mock responses when API changes
- Add new security test cases for emerging threats
- Refresh example data with current best practices
- Update dependency documentation

### Performance Monitoring
- Track test execution times
- Monitor memory usage during E2E tests
- Validate resource cleanup after tests

### CI/CD Integration
Tests are designed for automated execution:
- Unit tests run on every commit
- Integration tests run on PR validation
- E2E tests run on release candidates
- Security tests run on schedule for compliance