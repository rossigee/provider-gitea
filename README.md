# Provider Gitea

A Crossplane provider for managing Gitea repositories, organizations, users, and related resources.

## Overview

This provider enables declarative management of Gitea instances through Kubernetes custom resources. It supports managing repositories, organizations, users, webhooks, deploy keys, and access tokens using Crossplane's managed resource lifecycle.

## Features

- **Repository Management**: Create and manage Git repositories
- **Organization Management**: Manage organizations and their settings
- **User Management**: User account management and configuration
- **Webhook Management**: Configure repository and organization webhooks
- **Deploy Key Management**: Manage SSH deploy keys for repositories

## Status

[![CI](https://github.com/crossplane-contrib/provider-gitea/workflows/CI/badge.svg)](https://github.com/crossplane-contrib/provider-gitea/actions)
[![Coverage](https://codecov.io/gh/crossplane-contrib/provider-gitea/branch/master/graph/badge.svg)](https://codecov.io/gh/crossplane-contrib/provider-gitea)
[![Go Report Card](https://goreportcard.com/badge/github.com/crossplane-contrib/provider-gitea)](https://goreportcard.com/report/github.com/crossplane-contrib/provider-gitea)

- **Test Coverage**: 74.1% (target: 80%+)
- **API Client**: Comprehensive Gitea API integration
- **Controller Status**: In development (API types need Crossplane integration)

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
# Install latest version
kubectl crossplane install provider ghcr.io/crossplane-contrib/provider-gitea:latest

# Or install specific version
kubectl crossplane install provider ghcr.io/crossplane-contrib/provider-gitea:v0.1.0
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

## Documentation

- [Configuration Guide](docs/CONFIGURATION.md)
- [Development Guide](docs/DEVELOPMENT.md)
- [Resource Reference](docs/RESOURCES.md)

## Development

See [DEVELOPMENT.md](docs/DEVELOPMENT.md) for development setup and contribution guidelines.