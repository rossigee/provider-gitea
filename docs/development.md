# Development Guide

This guide covers how to develop and contribute to the enterprise-grade Gitea provider with 22 managed resource types and comprehensive testing infrastructure.

## Prerequisites

- Go 1.21 or later
- Docker
- kubectl
- kind (for local testing)
- Crossplane CLI
- Git with pre-commit hooks support

## Local Development Setup

1. Clone the repository:
```bash
git clone https://github.com/crossplane-contrib/provider-gitea.git
cd provider-gitea
```

2. Install Git hooks (prevents large file commits):
```bash
./scripts/install-hooks.sh
```

3. Initialize git submodules:
```bash
make submodules
```

4. Install dependencies:
```bash
go mod download
```

5. Generate code:
```bash
make generate
```

## Building

### Build the provider binary:
```bash
make build
```

### Build the Docker image:
```bash
make docker-build
```

### Build the Crossplane package:
```bash
make xpkg.build
```

## Testing

### Run unit tests:
```bash
make test
```

### Run with a local Gitea instance:

1. Start a local Gitea instance with Docker:
```bash
docker run -d \
  --name gitea \
  -p 3000:3000 \
  -e INSTALL_LOCK=true \
  -e SECRET_KEY=test \
  gitea/gitea:latest
```

2. Create a test user and access token in the Gitea web UI (http://localhost:3000)

3. Run the provider out-of-cluster:
```bash
make run
```

## Code Structure

```
provider-gitea/
├── apis/                           # API definitions (22 resource types)
│   ├── repository/v1alpha1/        # Repository management
│   ├── organization/v1alpha1/      # Organization management  
│   ├── user/v1alpha1/              # User management
│   ├── webhook/v1alpha1/           # Webhook configuration
│   ├── branchprotection/v1alpha1/  # Branch protection rules
│   ├── accesstoken/v1alpha1/       # API token management
│   ├── repositorysecret/v1alpha1/  # CI/CD secrets
│   ├── action/v1alpha1/            # Actions/workflows
│   ├── runner/v1alpha1/            # Self-hosted runners
│   ├── adminuser/v1alpha1/         # Administrative users
│   ├── [+12 more enterprise resources]
│   └── v1beta1/                    # Provider configuration
├── cmd/provider/                   # Main entry point
├── internal/
│   ├── clients/                    # Gitea API clients (60+ methods)
│   ├── controller/                 # Resource controllers (22 controllers)
│   │   ├── testing/                # Shared test infrastructure
│   │   ├── repository/             # Repository controller
│   │   ├── organization/           # Organization controller
│   │   └── [+20 more controllers]
│   └── features/                   # Feature flags
├── package/                        # Crossplane package manifests
├── examples/                       # Example manifests (100+ examples)
├── docs/                           # Documentation
├── test/                           # Test utilities and mocks
└── scripts/                        # Development scripts
```

## Adding New Resources

1. Create the API definition in `apis/<resource>/v1alpha1/`:
   - `doc.go` - Package documentation
   - `register.go` - Scheme registration
   - `types.go` - Resource types

2. Add client methods in `internal/clients/`:
   - Create a new file for the resource's API calls
   - Implement CRUD operations

3. Create the controller in `internal/controller/<resource>/`:
   - Implement the managed resource reconciler
   - Handle Create, Update, Delete, Observe operations

4. Register the controller in `cmd/provider/main.go`

5. Add the new API to `apis/apis.go`

6. Generate code:
```bash
make generate
```

## Testing Infrastructure

The provider includes comprehensive testing infrastructure at `internal/controller/testing/`:

### Shared Test Components
- **TestFixtures**: Common test data and response builders
- **MockClientBuilder**: Fluent interface for Gitea mock clients
- **K8sSecretBuilder**: Kubernetes secret creation utilities  
- **K8sClientBuilder**: Fake Kubernetes client setup
- **TestSuite**: Test orchestration and assertion helpers

### Test Coverage
- **23/23 controllers** with 100% test success rate
- **184 passing tests** across all resource types
- **Mock integration** for both Gitea API and Kubernetes clients
- **Systematic patterns** for CRUD operations testing

See [Test Infrastructure README](../internal/controller/testing/README.md) for detailed usage.

## Testing Checklist

Before submitting a PR:

- [ ] Code builds successfully
- [ ] All 184 tests pass (`make test`)
- [ ] Lint checks pass (`make lint`)
- [ ] Generated code is up to date (`make generate`)
- [ ] Examples work with a real Gitea instance
- [ ] Test infrastructure patterns followed for new controllers
- [ ] Documentation is updated
- [ ] Git hooks pass (no large files committed)
- [ ] Commit follows conventional commit format

## Release Process

1. Update version in relevant files
2. Update CHANGELOG.md
3. Create a git tag (e.g., `git tag v0.1.0`)
4. Push the tag (`git push origin v0.1.0`)
5. GitHub Actions will automatically:
   - Build and push Docker images to GHCR
   - Create GitHub release with artifacts
   - Upload Crossplane packages to GHCR OCI registry

Note: Currently using GitHub Container Registry (GHCR). Upbound registry integration is planned for future releases.

## Debugging

### Enable debug logging:
```bash
make run -- --debug
```

### View controller logs in cluster:
```bash
kubectl logs -f deployment/provider-gitea -n crossplane-system
```

### Inspect managed resources:
```bash
kubectl describe repository my-repo
kubectl get events --field-selector involvedObject.name=my-repo
```
