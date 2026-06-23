# Development Guide

This guide covers how to develop and contribute to the Gitea provider with 15 managed resource types and comprehensive testing infrastructure.

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
git clone https://github.com/mosabastion/provider-gitea.git
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

### Run the full CI gate:
```bash
make validate
```

### Run end-to-end tests (uptest on kind against a real Gitea):
```bash
make e2e          # spin up, run, tear down
make e2e-keep     # keep the kind cluster + Gitea running afterwards
```

These targets are driven by `scripts/e2e.sh`.

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
├── apis/                           # API definitions (15 resource types, v2 namespaced)
│   ├── accesstoken/                # API token management
│   ├── branchprotection/           # Branch protection rules
│   ├── githook/                    # Server-side Git hooks
│   ├── label/                      # Issue/PR labels
│   ├── organization/               # Organization management
│   ├── organizationsecret/         # Organization-wide CI/CD secrets
│   ├── organizationsettings/       # Organization settings
│   ├── repository/                 # Repository management
│   ├── repositorycollaborator/     # Repository collaborators
│   ├── repositorykey/              # Repository deploy keys
│   ├── repositorysecret/           # Repository CI/CD secrets
│   ├── team/                       # Organization teams
│   ├── user/                       # User management
│   ├── webhook/                    # Webhook configuration
│   └── v1beta1/                    # Provider configuration (cluster-scoped)
├── cmd/provider/                   # Main entry point
├── internal/
│   ├── clients/                    # Gitea API clients
│   ├── controller/                 # Resource controllers (one per kind)
│   │   ├── testing/                # Shared test infrastructure
│   │   ├── accesstoken/
│   │   ├── branchprotection/
│   │   ├── githook/
│   │   ├── label/
│   │   ├── organization/
│   │   ├── organizationsecret/
│   │   ├── organizationsettings/
│   │   ├── repository/
│   │   ├── repositorycollaborator/
│   │   ├── repositorykey/
│   │   ├── repositorysecret/
│   │   ├── team/
│   │   ├── user/
│   │   └── webhook/
│   └── features/                   # Feature flags
├── package/                        # Crossplane package manifests
├── examples/                       # Example manifests
├── docs/                           # Documentation
├── test/                           # Test utilities and mocks
└── scripts/                        # Development scripts
```

## Adding New Resources

1. Create the API definition in `apis/<resource>/` (v2 namespaced, API group `<resource>.gitea.m.crossplane.io`):
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
- Every controller has unit tests covering its CRUD operations
- **Mock integration** for both Gitea API and Kubernetes clients
- **Systematic patterns** for CRUD operations testing

See [Test Infrastructure README](../internal/controller/testing/README.md) for detailed usage.

## Testing Checklist

Before submitting a PR:

- [ ] Code builds successfully
- [ ] Unit tests pass (`make test`)
- [ ] Full CI gate passes (`make validate`)
- [ ] Generated code is up to date (`make generate`)
- [ ] End-to-end tests pass (`make e2e`)
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
