# Configuration Guide

This guide explains how to configure the Gitea provider for enterprise-grade infrastructure management with comprehensive security and CI/CD integration.

## Provider Configuration

The provider requires a `ProviderConfig` that specifies how to connect to your Gitea instance with appropriate security credentials.

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

## Enterprise Access Token Requirements

For managing all 22 resource types, the access token must have comprehensive permissions:

### **Core Repository & Organization Management**
- **Repository management**: `repo` scope - Full repository access
- **Organization management**: `admin:org` scope - Organization administration  
- **User management**: `admin:user` scope - User lifecycle management (admin users only)
- **Webhook management**: `admin:repo_hook`, `admin:org_hook` - Webhook configuration

### **Enterprise Security Features**
- **SSH Key management**: `admin:public_key` scope - User and repository SSH keys
- **Access Token management**: `admin:application` scope - API token lifecycle
- **Branch Protection**: `repo` scope with admin privileges - Protection rule enforcement
- **Repository Secrets**: `repo` scope - CI/CD secret management
- **Organization Secrets**: `admin:org` scope - Organization-wide secrets

### **CI/CD Integration**
- **Actions Workflows**: `repo`, `workflow` scopes - Workflow management
- **Self-hosted Runners**: `admin:org`, `admin:user` scopes - Runner registration
- **Repository Keys**: `repo` scope - Deployment key management

### **Administrative Features**
- **Administrative Users**: `admin:user` scope - Service account management
- **Organization Settings**: `admin:org` scope - Policy enforcement
- **Git Hooks**: `admin:repo_hook` scope - Server-side hook management
- **Team Management**: `admin:org` scope - Team and membership control

### **Recommended Token Scopes for Enterprise Setup**
```
repo, admin:org, admin:user, admin:public_key, admin:repo_hook, admin:org_hook, admin:application, workflow
```

## Multi-Environment Enterprise Setup

For enterprise deployments, configure separate ProviderConfigs for different environments with appropriate security isolation:

### Production Environment
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

### Environment-Specific Resource Configuration

```yaml
apiVersion: repository.gitea.crossplane.io/v1alpha1
kind: Repository
metadata:
  name: enterprise-app
spec:
  forProvider:
    name: enterprise-app
    owner: production-org
    private: true
  providerConfigRef:
    name: production
---
apiVersion: branchprotection.gitea.crossplane.io/v1alpha1
kind: BranchProtection
metadata:
  name: main-branch-protection
spec:
  forProvider:
    repository: enterprise-app
    owner: production-org
    branch: main
    dismissStaleReviews: true
    requireCodeOwnerReviews: true
    requiredApprovingReviewCount: 2
    restrictPushes: true
  providerConfigRef:
    name: production
```

## Security Best Practices

### 1. **Token Rotation**
```yaml
# Use external-secrets operator for automated token rotation
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: gitea-token
  namespace: crossplane-system
spec:
  secretStoreRef:
    name: vault-secret-store
    kind: SecretStore
  target:
    name: gitea-secret
  data:
  - secretKey: token
    remoteRef:
      key: gitea/production
      property: access-token
```

### 2. **Network Security**
```yaml
apiVersion: gitea.crossplane.io/v1beta1
kind: ProviderConfig
metadata:
  name: secure-production
spec:
  baseURL: "https://git.company.com"
  insecure: false  # Always validate SSL certificates
  credentials:
    source: Secret
    secretRef:
      namespace: crossplane-system
      name: gitea-prod-secret
      key: token
```

### 3. **RBAC Integration**
```yaml
# Limit provider access with RBAC
apiVersion: v1
kind: ServiceAccount
metadata:
  name: gitea-provider
  namespace: crossplane-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: gitea-provider
rules:
- apiGroups: ["gitea.crossplane.io"]
  resources: ["*"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: gitea-provider
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: gitea-provider
subjects:
- kind: ServiceAccount
  name: gitea-provider
  namespace: crossplane-system
```

## Advanced Configuration Patterns

### CI/CD Integration
```yaml
# Repository with full CI/CD setup
apiVersion: repository.gitea.crossplane.io/v1alpha1
kind: Repository
metadata:
  name: microservice-app
spec:
  forProvider:
    name: microservice-app
    owner: devops-org
    private: true
    hasIssues: true
    hasPullRequests: true
---
apiVersion: action.gitea.crossplane.io/v1alpha1
kind: Action
metadata:
  name: ci-pipeline
spec:
  forProvider:
    repository: microservice-app
    owner: devops-org
    filename: ".gitea/workflows/ci.yaml"
    active: true
    content: |
      name: CI Pipeline
      on: [push, pull_request]
      jobs:
        test:
          runs-on: ubuntu-latest
          steps:
          - uses: actions/checkout@v3
          - name: Run tests
            run: make test
---
apiVersion: runner.gitea.crossplane.io/v1alpha1
kind: Runner
metadata:
  name: org-runner
spec:
  forProvider:
    organization: devops-org
    name: "Enterprise Runner"
    token: "runner-registration-token"
    labels: ["enterprise", "docker"]
```

## Troubleshooting

### Common Configuration Issues

1. **Token Permissions**
   - Error: `403 Forbidden` → Check token scopes
   - Error: `401 Unauthorized` → Verify token validity
   - Error: `404 Not Found` → Check baseURL and resource existence

2. **SSL Certificate Issues**
   ```yaml
   # For development only - disable SSL verification
   spec:
     baseURL: "https://gitea.example.com"
     insecure: true  # NOT recommended for production
   ```

3. **Network Connectivity**
   ```bash
   # Test connectivity from provider pod
   kubectl exec -n crossplane-system deployment/provider-gitea -- \
     curl -H "Authorization: token YOUR_TOKEN" \
     https://gitea.example.com/api/v1/user
   ```

### Debugging Configuration

```bash
# Check provider logs
kubectl logs -f deployment/provider-gitea -n crossplane-system

# Describe ProviderConfig
kubectl describe providerconfig default

# Check secret exists
kubectl get secret gitea-secret -n crossplane-system -o yaml
```

## Migration Guide

### From Basic to Enterprise Setup

1. **Update Token Scopes**: Expand token permissions for enterprise features
2. **Add Security Resources**: Implement branch protection, SSH keys, access tokens
3. **Configure CI/CD**: Set up actions, runners, and repository secrets
4. **Enable Administrative Features**: Configure organization settings and admin users

### Upgrade Checklist
- [ ] Token has all required scopes
- [ ] SSL certificates properly configured
- [ ] RBAC permissions updated
- [ ] Network connectivity verified
- [ ] Provider logs show successful authentication
- [ ] Test resource creation works
