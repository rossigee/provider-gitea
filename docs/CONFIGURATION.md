# Configuration Guide

This guide explains how to configure the Gitea provider for use with your Gitea instance.

## Provider Configuration

The provider requires a `ProviderConfig` that specifies how to connect to your Gitea instance.

### Basic Configuration

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: gitea-secret
  namespace: crossplane-system
type: Opaque
stringData:
  token: "your-gitea-access-token"
---
apiVersion: gitea.crossplane.io/v1beta1
kind: ProviderConfig
metadata:
  name: default
spec:
  baseURL: "https://gitea.example.com"
  credentials:
    source: Secret
    secretRef:
      namespace: crossplane-system
      name: gitea-secret
      key: token
```

### Configuration Options

- **baseURL**: The base URL of your Gitea instance (required)
- **insecure**: Set to `true` to skip SSL certificate verification (default: `false`)
- **credentials.source**: Must be `Secret` (other sources not yet supported)
- **credentials.secretRef**: Reference to a Kubernetes secret containing the access token

### Access Token Requirements

The access token must have appropriate permissions for the resources you want to manage:

- **Repository management**: `repo` scope
- **Organization management**: `admin:org` scope  
- **User management**: `admin:user` scope (admin users only)
- **Webhook management**: `admin:repo_hook` scope

### Multiple Environments

You can create multiple ProviderConfigs for different Gitea instances:

```yaml
apiVersion: gitea.crossplane.io/v1beta1
kind: ProviderConfig
metadata:
  name: production
spec:
  baseURL: "https://git.company.com"
  credentials:
    source: Secret
    secretRef:
      namespace: crossplane-system
      name: gitea-prod-secret
      key: token
---
apiVersion: gitea.crossplane.io/v1beta1
kind: ProviderConfig
metadata:
  name: staging
spec:
  baseURL: "https://staging-git.company.com"
  credentials:
    source: Secret
    secretRef:
      namespace: crossplane-system
      name: gitea-staging-secret
      key: token
```

Then reference the appropriate config in your managed resources:

```yaml
apiVersion: repository.gitea.crossplane.io/v1alpha1
kind: Repository
metadata:
  name: my-repo
spec:
  forProvider:
    name: my-repo
    owner: myorg
  providerConfigRef:
    name: production  # or staging
```