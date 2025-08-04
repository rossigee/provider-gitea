# Provider Gitea

A comprehensive Crossplane provider for enterprise-grade Gitea management with integrated CI/CD automation, security enforcement, and administrative capabilities.

## Overview

This provider enables declarative management of Gitea instances through Kubernetes custom resources with **22 managed resource types**. It provides complete enterprise functionality including repository management, security enforcement, CI/CD integration, and administrative automation using Crossplane's managed resource lifecycle.

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

## Status

[![CI](https://github.com/crossplane-contrib/provider-gitea/workflows/CI/badge.svg)](https://github.com/crossplane-contrib/provider-gitea/actions)
[![Coverage](https://codecov.io/gh/crossplane-contrib/provider-gitea/branch/master/graph/badge.svg)](https://codecov.io/gh/crossplane-contrib/provider-gitea)
[![Go Report Card](https://goreportcard.com/badge/github.com/crossplane-contrib/provider-gitea)](https://goreportcard.com/report/github.com/crossplane-contrib/provider-gitea)

- **Version**: v0.2.0 (Enterprise features complete)
- **Resources**: 22 managed resource types
- **API Client**: Complete Gitea API integration with enterprise features
- **Controller Status**: Production ready with comprehensive test coverage
- **Registry**: `ghcr.io/rossigee/provider-gitea:v0.2.0`

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

1. Install the provider from GitHub Container Registry:
```bash
# Install latest enterprise version
kubectl crossplane install provider ghcr.io/rossigee/provider-gitea:v0.2.0

# Or install latest stable
kubectl crossplane install provider ghcr.io/rossigee/provider-gitea:latest
```

Alternatively, use the install manifest:
```bash
kubectl apply -f examples/install.yaml
```

2. Create a provider configuration:
```bash
kubectl apply -f examples/provider/config.yaml
```

3. Create your first repository:
```bash
kubectl apply -f examples/repository/basic-repo.yaml
```

## Enterprise Setup

For complete enterprise-grade setup with CI/CD integration, security policies, and administrative automation:

```bash
# Complete enterprise configuration
kubectl apply -f examples/enterprise-complete-setup.yaml

# Or step-by-step setup:
kubectl apply -f examples/adminuser/service-accounts.yaml
kubectl apply -f examples/branchprotection/enterprise-branch-protection.yaml  
kubectl apply -f examples/runner/organization-runners.yaml
kubectl apply -f examples/action/ci-pipeline.yaml
```

This provides:
- 🔒 **Enterprise Security**: Branch protection, SSH keys, access tokens
- 🚀 **CI/CD Integration**: Actions workflows and self-hosted runners  
- 👑 **Administrative Control**: Service accounts and administrative users
- 🏢 **Organization Management**: Complete organizational policies

## Documentation

- [Configuration Guide](docs/CONFIGURATION.md)
- [Development Guide](docs/DEVELOPMENT.md)
- [Resource Reference](docs/RESOURCES.md)

## Development

See [DEVELOPMENT.md](docs/DEVELOPMENT.md) for development setup and contribution guidelines.
