# Contributing to Provider Gitea

Thank you for your interest in contributing to the Gitea provider for Crossplane! This document provides guidelines and information for contributors.

## Code of Conduct

This project adheres to the [Crossplane Code of Conduct](https://github.com/crossplane/crossplane/blob/master/CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code.

## Getting Started

### Prerequisites

- Go 1.21+
- Docker
- kubectl
- kind (for local testing)
- pre-commit (recommended)

### Development Setup

1. **Fork and Clone**:
   ```bash
   git clone https://github.com/YOUR_USERNAME/provider-gitea.git
   cd provider-gitea
   # upstream: https://github.com/mosabastion/provider-gitea
   ```

2. **Set up pre-commit hooks** (recommended):
   ```bash
   pip install pre-commit
   pre-commit install
   ```

3. **Install dependencies**:
   ```bash
   go mod download
   make submodules
   ```

4. **Generate code**:
   ```bash
   make generate
   ```

5. **Run tests**:
   ```bash
   make test
   ```

## Development Workflow

### Making Changes

1. **Create a feature branch**:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes** following the coding standards below

3. **Run quality checks**:
   ```bash
   # Run unit tests
   make test
   
   # Run the full CI gate (build, lint, generate-diff, unit tests)
   make validate
   
   # Run pre-commit hooks
   pre-commit run --all-files
   ```

4. **Commit your changes**:
   ```bash
   git add .
   git commit -m "feat: add new repository webhook feature"
   ```

   Use [Conventional Commits](https://www.conventionalcommits.org/) format:
   - `feat:` for new features
   - `fix:` for bug fixes
   - `docs:` for documentation changes
   - `test:` for test additions/changes
   - `refactor:` for code refactoring
   - `chore:` for maintenance tasks

5. **Push and create PR**:
   ```bash
   git push origin feature/your-feature-name
   ```

### Pull Request Process

1. **Ensure your PR**:
   - [ ] Has a clear, descriptive title
   - [ ] Includes a detailed description of changes
   - [ ] References any related issues
   - [ ] Includes tests for new functionality
   - [ ] Updates documentation if needed
   - [ ] Passes all CI checks

2. **PR Review Process**:
   - At least one maintainer review is required
   - Address any feedback promptly
   - Keep the PR up to date with the main branch

3. **After Approval**:
   - PRs will be merged by maintainers
   - Your branch will be deleted automatically

## Coding Standards

### Go Code Style

- Follow standard Go conventions
- Use `gofmt` and `goimports` (handled by pre-commit hooks)
- Maintain high test coverage (target: 80%+)
- Write clear, self-documenting code
- Add comments for complex logic

### API Design

When adding new resources:

1. **Follow Crossplane patterns**:
   - Embed `xpv1.ResourceSpec` and `xpv1.ResourceStatus`
   - Implement required managed resource interfaces
   - Use proper Crossplane annotations

2. **API Conventions**:
   - Use the v2 namespaced API style (`<kind>.gitea.m.crossplane.io/v2`) for new resources
   - Use descriptive field names
   - Add proper validation tags
   - Include detailed field documentation

3. **Controller Patterns**:
   - Implement proper reconciliation logic
   - Handle errors gracefully with proper wrapping
   - Use appropriate requeue strategies
   - Add proper logging with context

### Testing Requirements

1. **Unit Tests**:
   - Test all public functions
   - Mock external dependencies
   - Use table-driven tests for multiple scenarios
   - Include both success and error cases

2. **Integration Tests**:
   - Test against real Gitea instance
   - Cover major user workflows
   - Use build tags to separate from unit tests

3. **Test Organization**:
   ```go
   func TestClientFunction(t *testing.T) {
       tests := []struct {
           name    string
           setup   func()
           input   interface{}
           want    interface{}
           wantErr bool
       }{
           // Test cases
       }
       
       for _, tt := range tests {
           t.Run(tt.name, func(t *testing.T) {
               // Test implementation
           })
       }
   }
   ```

### Documentation

- Update relevant documentation for any changes
- Include examples for new features
- Use clear, concise language
- Follow markdown formatting standards

## Testing

### Running Tests Locally

```bash
# Unit tests
make test

# Full CI gate (build, lint, generate-diff, unit tests)
make validate

# End-to-end tests (uptest on kind against a real Gitea, via scripts/e2e.sh)
make e2e

# End-to-end tests, keeping the cluster + Gitea afterwards
make e2e-keep
```

### Test Coverage

Maintain high unit-test coverage. Areas that typically need attention:
- Error handling paths in the HTTP client
- Controller reconciliation logic
- Edge cases in API validation

## Release Process

Releases are handled by maintainers:

1. Version bump in relevant files
2. Update CHANGELOG.md
3. Create release tag (e.g., `git tag v0.1.0`)
4. Push tag to trigger automated build and publish via GitHub Actions
5. Artifacts are published to:
   - GitHub Container Registry (GHCR) for Docker images and packages
   - GitHub Releases for binaries and artifacts

Note: Currently using GitHub Container Registry. Upbound registry integration planned for future.

## Getting Help

- **Questions**: Open a [Discussion](https://github.com/mosabastion/provider-gitea/discussions)
- **Bugs**: Open an [Issue](https://github.com/mosabastion/provider-gitea/issues/new/choose)
- **Security**: Report privately via the [security advisory page](https://github.com/mosabastion/provider-gitea/security/advisories/new)

## Recognition

Contributors will be recognized:
- In the CHANGELOG.md
- In release notes
- As code authors in git history

Thank you for contributing to making Crossplane better! 🎉
