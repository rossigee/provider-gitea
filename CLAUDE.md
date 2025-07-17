# Gitea Provider for Crossplane

## Overview
This provider enables declarative management of Gitea resources through Kubernetes using Crossplane.

## Development Status
- **Created**: 2024-01-17
- **Status**: Complete implementation
- **Version**: v0.1.0 (pre-release)

## Implemented Resources
1. **Repository** - Full CRUD operations for Git repositories
2. **Organization** - Organization management
3. **User** - User account management (admin only)
4. **Webhook** - Repository and organization webhooks
5. **DeployKey** - SSH deploy keys for repositories

## Architecture
- Built on Crossplane Runtime v1.15.0
- Uses Gitea API v1
- Follows standard Crossplane provider patterns
- Comprehensive test coverage

## Key Files
- `cmd/provider/main.go` - Provider entry point
- `internal/clients/gitea.go` - Gitea API client
- `internal/controller/` - Resource controllers
- `apis/*/v1alpha1/types.go` - Resource definitions

## Build & Test
```bash
make generate    # Generate code
make build       # Build provider
make test        # Run tests
make docker-build # Build container
make xpkg.build  # Build package
```

## Deployment
1. Install Crossplane
2. Create provider config with Gitea credentials
3. Apply resource manifests

## Notes
- All resources support full lifecycle management
- Provider supports multiple Gitea instances
- Comprehensive examples in `examples/` directory
- CI/CD pipeline configured for GitHub Actions