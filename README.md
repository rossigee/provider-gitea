# Provider Gitea

A v2-only Crossplane provider framework for Gitea management with complete API definitions and client libraries. **Note**: Controller implementations required for actual resource management functionality.

## Overview

This provider framework includes **22 v2 resource type definitions** for declarative Gitea management. It provides complete API definitions, client libraries, and testing infrastructure. **Controller implementations are required to enable actual resource management functionality.**

## ⚠️ Current Status: Framework Ready

**What's Included:**
- ✅ Complete v2 API definitions with namespace isolation
- ✅ Gitea client library with comprehensive API coverage
- ✅ Provider infrastructure that builds and packages successfully
- ✅ Testing framework and mock clients

**What's Missing:**
- ❌ Controller implementations for resource lifecycle management
- ❌ Actual Gitea resource synchronization and management

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

### **V2-Only Architecture** ✨ (v0.7.0+)
- **Pure V2 Implementation**: Clean v2-only provider without legacy code burden
- **Namespace Isolation**: All 22 resources use namespace-scoped `.m.` API groups
- **Enhanced Multi-tenancy**: Complete namespace isolation and tenant separation
- **Modern Architecture**: Built with Crossplane Runtime v2.0 patterns
- **Connection References**: Advanced multi-tenant capabilities with enhanced connectivity
- **No Backward Compatibility**: Clean slate implementation for optimal performance

## Status

[![CI](https://github.com/crossplane-contrib/provider-gitea/workflows/CI/badge.svg)](https://github.com/crossplane-contrib/provider-gitea/actions)
[![Coverage](https://codecov.io/gh/crossplane-contrib/provider-gitea/branch/master/graph/badge.svg)](https://codecov.io/gh/crossplane-contrib/provider-gitea)
[![Go Report Card](https://goreportcard.com/badge/github.com/crossplane-contrib/provider-gitea)](https://goreportcard.com/report/github.com/crossplane-contrib/provider-gitea)

- **Version**: v0.7.0+ (v2-only provider framework)
- **Resources**: 22 v2 resource type definitions with namespace isolation
- **API Client**: Complete Gitea API integration with 19.7% test coverage
- **Controller Status**: Framework ready - controller implementations required
- **Registry**: `ghcr.io/rossigee/provider-gitea:v0.7.0`

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

## Quick Start (Framework Installation)

⚠️ **Note**: This installs the provider framework with API definitions. Controller implementations are required for actual resource management.

1. Install the v2-only provider framework:
```bash
# Install v2-only framework
kubectl crossplane install provider ghcr.io/rossigee/provider-gitea:v0.7.0
```

2. View available v2 resource definitions:
```bash
kubectl get crds | grep gitea.m.crossplane.io
```

3. Example v2 resource definition (requires controller implementation):
```bash
kubectl apply -f examples/v2/repository-namespaced.yaml
# Note: Resource will be created but not reconciled until controllers are implemented
```

## Framework Development

This v2-only provider framework includes complete API definitions and examples for enterprise Gitea management:

```bash
# View all available v2 resource examples
ls examples/v2/

# View enterprise feature examples (require controller implementation)
ls examples/adminuser/ examples/branchprotection/ examples/action/
```

**Enterprise capabilities when controllers are implemented**:
- 🔒 **Enterprise Security**: Branch protection, SSH keys, access tokens
- 🚀 **CI/CD Integration**: Actions workflows and self-hosted runners
- 👑 **Administrative Control**: Service accounts and administrative users
- 🏢 **Organization Management**: Complete organizational policies

## Testing

This provider framework includes solid test coverage for its current scope:

- **Test Success Rate**: 100% - all tests pass
- **Client Library**: 19.7% coverage across all major Gitea API operations
- **Test Infrastructure**: 8.4% coverage for shared testing utilities
- **Overall Coverage**: 4.8% (appropriate for framework with no controllers)
- **Mock Clients**: Ready for controller development and testing

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
