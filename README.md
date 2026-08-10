# provider-gitea

[![CI](https://img.shields.io/github/actions/workflow/status/rossigee/provider-gitea/ci.yml?branch=master)][build]
[![Version](https://img.shields.io/github/v/release/rossigee/provider-gitea)][releases]
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

[build]: https://github.com/rossigee/provider-gitea/actions/workflows/ci.yml
[releases]: https://github.com/rossigee/provider-gitea/releases

## Overview

A [Crossplane](https://crossplane.io/) provider for declarative Gitea repository, organization, and user management. All 22 resource types have complete v2 (namespaced, `.m.` API group) type definitions and a Gitea API client, but only **6 have controllers wired up and actually reconciling** — see [Resource Types](#resource-types) below for which.

## Container Registry

- **Primary**: `ghcr.io/rossigee/provider-gitea:v0.10.2`

## Features

- **Repository management**: full Git repository lifecycle
- **Repository keys & secrets**: deploy keys and CI/CD secrets
- **Organization & user management**: organizations and user accounts
- **Webhooks**: webhook configuration for repository events
- **16 additional resource types defined but not yet reconciling** (see below) — branch protection, access tokens, teams, labels, CI/CD actions and runners, admin users, and more

## Getting Started

### Prerequisites

- Kubernetes with Crossplane installed
- A Gitea instance and an API access token

### Installation

```bash
kubectl crossplane install provider ghcr.io/rossigee/provider-gitea:v0.10.2
```

### Configuration

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: gitea-creds
  namespace: crossplane-system
type: Opaque
stringData:
  token: "your-gitea-api-token"
---
apiVersion: gitea.crossplane.io/v1beta1
kind: ProviderConfig
metadata:
  name: default
spec:
  credentials:
    source: Secret
    secretRef:
      namespace: crossplane-system
      name: gitea-creds
      key: token
```

## Usage

```yaml
apiVersion: repository.gitea.m.crossplane.io/v2
kind: Repository
metadata:
  name: my-repo
  namespace: default
spec:
  forProvider:
    owner: my-org
    name: my-repo
    private: true
  providerConfigRef:
    name: default
```

## Resource Types

All groups are `<resource>.gitea.m.crossplane.io/v2` (namespaced).

**Controllers active:**

| Resource | Description |
|----------|-------------|
| Repository | Git repository lifecycle |
| RepositoryKey | Repository SSH deploy keys |
| RepositorySecret | Repository CI/CD secrets |
| Organization | Organization lifecycle |
| User | User account management |
| Webhook | Webhook configuration |

**Types and client defined, no controller yet** (CRDs install and API types exist, but resources will not reconcile):

Issue, UserKey, RepositoryCollaborator, OrganizationSecret, AdminUser, BranchProtection, GitHook, Action, Team, PullRequest, Runner, AccessToken, OrganizationMember, Release, OrganizationSettings, DeployKey, Label

See `examples/` for a working manifest per resource type (including the not-yet-reconciling ones, for reference).

## Development

**Important**: install the git hooks after cloning, to prevent large file commits:

```bash
./scripts/install-hooks.sh
```

```bash
# Build
make build

# Test
make test

# Lint
make lint

# Generate
make generate
```

Further documentation:

- [Configuration Guide](docs/CONFIGURATION.md)
- [Development Guide](docs/DEVELOPMENT.md)
- [Resource Reference](docs/RESOURCES.md)
- [Test Infrastructure](internal/controller/testing/README.md)

## Contributing

Issues and pull requests are welcome at [github.com/rossigee/provider-gitea](https://github.com/rossigee/provider-gitea).

## License

provider-gitea is under the Apache 2.0 license.
