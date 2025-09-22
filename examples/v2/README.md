# Gitea Provider v2 APIs Examples

This directory contains examples demonstrating the v2 namespaced APIs for the Gitea Crossplane provider. The v2 APIs provide enhanced features including namespace isolation, improved observability, and multi-tenant support.

## Overview

The Gitea provider now supports dual-scope operation:
- **v1alpha1 APIs**: Cluster-scoped (backward compatible)
- **v2 APIs**: Namespaced with `.m.` API group pattern

## Key v2 Features

### 🏠 **Namespace Isolation**
- Resources are scoped to namespaces
- Team-specific configurations and credentials
- Better RBAC and access control

### 🔗 **Enhanced Connectivity**
- `providerConfigRef`: Namespace-scoped provider configurations
- `connectionRef`: Multi-tenant connection management
- Flexible credential management

### 📊 **Improved Observability**
- Enhanced status fields for better monitoring
- Rich metadata and annotations
- Better debugging capabilities

### 🛡️ **Multi-Tenancy**
- Multiple teams can use the same provider
- Isolated configurations per namespace
- No resource conflicts between teams

## Examples

### Basic Examples
- [`repository-namespaced.yaml`](repository-namespaced.yaml) - Simple namespaced repository
- [`organization-namespaced.yaml`](organization-namespaced.yaml) - Organization with enhanced observability

### Advanced Examples
- [`multi-tenant-setup.yaml`](multi-tenant-setup.yaml) - Multi-team namespace isolation
- [`migration-guide.yaml`](migration-guide.yaml) - v1alpha1 to v2 migration examples

## Quick Start

### 1. Create a Namespace
```bash
kubectl create namespace my-team
```

### 2. Create Credentials Secret
```bash
kubectl create secret generic gitea-credentials \
  --namespace=my-team \
  --from-literal=token=your_gitea_token
```

### 3. Apply v2 Resources
```bash
kubectl apply -f repository-namespaced.yaml
```

### 4. Verify Resources
```bash
# List v2 repositories in namespace
kubectl get repositories.repository.gitea.m.crossplane.io -n my-team

# Check status
kubectl describe repository example-repo-v2 -n my-team
```

## API Reference

### v2 API Groups
All v2 APIs use the `.m.` pattern in their API groups:

- `repository.gitea.m.crossplane.io/v2`
- `organization.gitea.m.crossplane.io/v2`
- `user.gitea.m.crossplane.io/v2`
- `webhook.gitea.m.crossplane.io/v2`
- And 18 more resource types...

### Enhanced Fields in v2

#### Common Enhancements
All v2 resources include:
```yaml
spec:
  forProvider:
    # Standard resource fields...

    # v2 Enhancements
    providerConfigRef:
      name: namespace-scoped-config
    connectionRef:  # Optional
      name: connection-name
```

#### Enhanced Observability
Many v2 resources include additional status fields:
```yaml
status:
  atProvider:
    # Standard observed fields...

    # v2 Enhanced observability
    createdAt: "2024-01-01T00:00:00Z"
    updatedAt: "2024-01-01T00:00:00Z"
    # Additional metrics and metadata
```

## Migration from v1alpha1

### Backward Compatibility
- v1alpha1 resources continue to work unchanged
- No breaking changes to existing deployments
- Gradual migration is supported

### Migration Steps
1. Create namespace and credentials
2. Create namespace-scoped ProviderConfig
3. Create v2 resources in namespace
4. Test thoroughly
5. Migrate existing resources when ready

See [`migration-guide.yaml`](migration-guide.yaml) for detailed examples.

## Best Practices

### Namespace Organization
```
company-frontend/     # Frontend team namespace
├── ProviderConfig: frontend-gitea
├── Repository: web-app
└── Secret: gitea-credentials

company-backend/      # Backend team namespace
├── ProviderConfig: backend-gitea
├── Repository: api-service
└── Secret: gitea-credentials

company-platform/     # Platform team namespace
├── Organization: platform-team
└── ProviderConfig: platform-gitea
```

### Security
- Use namespace-scoped secrets for credentials
- Apply RBAC per namespace
- Separate configurations per team/environment
- Regular credential rotation

### Monitoring
- Monitor resources per namespace
- Use enhanced observability fields
- Set up alerts for resource status changes
- Track resource lifecycle events

## Troubleshooting

### Common Issues

1. **Resource not found**
   ```bash
   # Make sure you're looking in the right namespace
   kubectl get repositories.repository.gitea.m.crossplane.io -A
   ```

2. **ProviderConfig not found**
   ```bash
   # Ensure ProviderConfig is in the same namespace as resources
   kubectl get providerconfigs -n my-team
   ```

3. **Authentication failures**
   ```bash
   # Check secret exists and has correct token
   kubectl get secret gitea-credentials -n my-team -o yaml
   ```

### Debug Commands
```bash
# List all v2 resources across namespaces
kubectl get repositories.repository.gitea.m.crossplane.io -A

# Check provider logs
kubectl logs -n crossplane-system deployment/provider-gitea

# Describe resource for detailed status
kubectl describe repository my-repo -n my-team
```

## Additional Resources

- [Crossplane Documentation](https://docs.crossplane.io/)
- [Provider Configuration](../../README.md)
- [API Reference](../../package/crds/)
- [Original v1alpha1 Examples](../)