# Provider Gitea

A v2-only Crossplane provider for declarative Gitea management: **23 namespaced
resource kinds**, each with a working reconciler.

## Overview

This provider manages Gitea resources (repositories, organizations, teams,
labels, webhooks, secrets, CI runners, branch protection, and more) as
Kubernetes custom resources. Every kind has a full create/observe/update/delete
controller, a unit test, and is exercised end-to-end on a kind cluster against a
mock Gitea backend.

## Status

**Implemented:**
- ✅ v2 API definitions, namespace-isolated (`*.gitea.m.crossplane.io`)
- ✅ Comprehensive Gitea client library (typed HTTP-status error classification)
- ✅ A reconciler for **all 23 resource kinds**, each unit-tested
- ✅ Correct packaging: one multi-arch xpkg with the runtime embedded and CRDs
  verified present before publish (`make xpkg-verify`)
- ✅ Self-contained e2e: `scripts/e2e.sh` drives apply→Ready→delete on kind
  against an in-cluster mock Gitea, wired into CI (`.github/workflows/e2e.yml`)

The controllers bake in the correctness lessons distilled in
[`crossplane-provider-template`](https://github.com/mosabastion/crossplane-provider-template)
`dev/docs/09-lessons-learned.md` — `Available()` set in `Observe`, not-found
classified off the typed HTTP status, real drift detection, external-name as the
authoritative identity for Observe/Update/Delete, a non-nil rate limiter, and a
package that can't ship Healthy-but-CRD-less.

## Core Features

### **Repository & Organization Management**
- **Repository Management**: Full lifecycle management of Git repositories
- **Organization Management**: Organizations, settings, and membership control  
- **Team Management**: Advanced team collaboration and access control
- **Label Management**: Issue and PR labeling automation
- **Collaborator Management**: Repository collaboration workflows

### **Enterprise Security** 🔒
- **Branch Protection**: Enterprise-grade branch protection with approval workflows
- **SSH Key Management**: User and repository SSH key lifecycle management  
- **Access Token Management**: Scoped API token management with security controls
- **Repository Secrets**: Secure CI/CD secret management with Kubernetes integration
- **Organization Secrets**: Centralized secret management for enterprise workflows
- **Git Hooks**: Server-side Git hook management for policy enforcement

### **CI/CD Integration** 🚀
- **Actions Workflows**: Declarative CI/CD pipeline management
- **Self-hosted Runners**: Multi-scope runner management (repository, organization, system)
- **Complete DevOps Automation**: End-to-end development and deployment workflows
- **Integrated Secret Management**: Seamless CI/CD secrets with Kubernetes

### **Administrative Features** 👑
- **Administrative Users**: Service account and admin user lifecycle management
- **User Management**: Complete user lifecycle with privilege controls
- **Organization Settings**: Enterprise-grade organizational policies
- **Multi-tenant Support**: Organization and user isolation with proper access controls

### **V2-Only Architecture** ✨ (v0.8.2)
- **Pure V2 Implementation**: Clean v2-only provider without legacy code burden
- **Namespace Isolation**: All 23 resources use namespace-scoped `.m.` API groups
- **Enhanced Multi-tenancy**: Complete namespace isolation and tenant separation
- **Modern Architecture**: Built with Crossplane Runtime v2.0 patterns
- **Connection References**: Advanced multi-tenant capabilities with enhanced connectivity
- **No Backward Compatibility**: Clean slate implementation for optimal performance

## Status

[![CI](https://github.com/crossplane-contrib/provider-gitea/workflows/CI/badge.svg)](https://github.com/crossplane-contrib/provider-gitea/actions)
[![Coverage](https://codecov.io/gh/crossplane-contrib/provider-gitea/branch/master/graph/badge.svg)](https://codecov.io/gh/crossplane-contrib/provider-gitea)
[![Go Report Card](https://goreportcard.com/badge/github.com/crossplane-contrib/provider-gitea)](https://goreportcard.com/report/github.com/crossplane-contrib/provider-gitea)

- **Resources**: 23 v2 resource kinds (namespace-isolated `.m.` API groups)
- **Controllers**: a working reconciler for every kind, each unit-tested
- **API Client**: complete Gitea API integration with typed error classification
- **e2e**: all 23 kinds driven apply→Ready→delete on kind against a mock Gitea
- **Registry**: `ghcr.io/rossigee/provider-gitea`

## Complete Resource Catalog

| **Resource Type** | **Purpose** | **Examples** |
|-------------------|-------------|--------------|
| `Repository` | Git repository management | [basic-repo.yaml](examples/repository/basic-repo.yaml) |
| `Organization` | Organization lifecycle | [basic-org.yaml](examples/organization/basic-org.yaml) |
| `User` | User account management | [basic-user.yaml](examples/user/basic-user.yaml) |
| `Webhook` | Webhook configuration | [repo-webhook.yaml](examples/webhook/repo-webhook.yaml) |
| `DeployKey` | SSH deploy key management | [basic-deploykey.yaml](examples/deploykey/basic-deploykey.yaml) |
| `Team` | Team collaboration | [basic-team.yaml](examples/team/basic-team.yaml) |
| `Label` | Issue/PR labels | [basic-labels.yaml](examples/label/basic-labels.yaml) |
| `RepositoryCollaborator` | Repository access | [basic-collaborators.yaml](examples/repositorycollaborator/basic-collaborators.yaml) |
| `OrganizationSettings` | Organization policies | [organizationsettings.yaml](examples/organizationsettings/organizationsettings.yaml) |
| `GitHook` | Server-side Git hooks | [post-receive-hook.yaml](examples/githook/post-receive-hook.yaml) |
| **Security Resources** | | |
| `BranchProtection` | Branch protection rules | [enterprise-branch-protection.yaml](examples/branchprotection/enterprise-branch-protection.yaml) |
| `RepositoryKey` | SSH key management | [deployment-key.yaml](examples/repositorykey/deployment-key.yaml) |
| `AccessToken` | API token management | [ci-token.yaml](examples/accesstoken/ci-token.yaml) |
| `RepositorySecret` | CI/CD secrets | [docker-registry-secret.yaml](examples/repositorysecret/docker-registry-secret.yaml) |
| `UserKey` | User SSH keys | [developer-ssh-key.yaml](examples/userkey/developer-ssh-key.yaml) |
| `OrganizationMember` | Organization membership | [team-membership.yaml](examples/organizationmember/team-membership.yaml) |
| `OrganizationSecret` | Organization-wide secrets | [harbor-integration.yaml](examples/organizationsecret/harbor-integration.yaml) |
| **CI/CD Resources** | | |
| `Action` | CI/CD workflows | [ci-pipeline.yaml](examples/action/ci-pipeline.yaml) |
| `Runner` | Self-hosted runners | [repository-runner.yaml](examples/runner/repository-runner.yaml) |
| **Administrative Resources** | | |
| `AdminUser` | Administrative users | [service-accounts.yaml](examples/adminuser/service-accounts.yaml) |

## Development Setup

**Important**: After cloning this repository, install the git hooks to prevent large file commits:

```bash
./scripts/install-hooks.sh
```

This installs a pre-commit hook that prevents:
- Files larger than 10MB
- Binary artifacts (*.xpkg, *.tar.gz, etc.)
- Build artifacts (provider binaries, cache files)

## Quick Start

1. Install the provider:
```bash
kubectl crossplane install provider ghcr.io/rossigee/provider-gitea:<tag>
```

2. Confirm the CRDs registered:
```bash
kubectl get crds | grep gitea.m.crossplane.io
```

3. Create a ProviderConfig + apply a resource (see `examples/`):
```bash
kubectl apply -f examples/e2e/repository.yaml
kubectl get repository.repository.gitea.m.crossplane.io -n <ns>
```

## Testing

```bash
make test          # unit tests (offline, table-driven per controller)
make e2e           # self-contained kind + mock-Gitea e2e (apply->Ready->delete)
make xpkg-verify   # assert the built package carries the Provider meta + all CRDs
```

- Every controller has a unit test asserting the correctness invariants
  (Available on the exists path, typed not-found, drift, external-name identity,
  idempotent delete).
- `make e2e` (and CI `e2e.yml`) drives all 23 kinds through their full lifecycle
  on a throwaway kind cluster against an in-cluster mock Gitea — no external
  dependency.

### Test Infrastructure

The provider includes a shared test infrastructure at [`internal/controller/testing/`](internal/controller/testing/) that provides:

- **TestFixtures** - Common test data and response builders
- **MockClientBuilder** - Fluent interface for Gitea mock clients
- **K8sSecretBuilder** - Kubernetes secret creation utilities
- **TestSuite** - Test orchestration and assertion helpers

See the [Test Infrastructure README](internal/controller/testing/README.md) for detailed usage examples.

### Running Tests

```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Run client library tests
go test ./internal/clients/...

# Lint code
make lint
```

## Documentation

- [Configuration Guide](docs/CONFIGURATION.md)
- [Development Guide](docs/DEVELOPMENT.md)
- [Resource Reference](docs/RESOURCES.md)
- [Test Infrastructure](internal/controller/testing/README.md)

## Development

See [DEVELOPMENT.md](docs/DEVELOPMENT.md) for development setup and contribution guidelines.
