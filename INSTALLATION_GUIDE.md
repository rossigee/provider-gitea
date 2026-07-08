# provider-gitea Installation Guide

Three distribution and installation methods for provider-gitea supporting Crossplane v2.2+.

## Quick Start

Choose your installation method:

### 1. kubectl (Minimal Setup) - Recommended for Development

Best for: Local development, testing, learning

```bash
# Clone the repository
git clone https://github.com/rossigee/provider-gitea.git
cd provider-gitea

# Install
kubectl apply -k deploy/kubectl/

# Verify
kubectl get provider provider-gitea
```

**Details**: See [deploy/kubectl/README.md](deploy/kubectl/README.md)

---

### 2. Helm Chart (Production Ready) - Recommended for Production

Best for: Kubernetes-native deployments, version management, upgrades

```bash
# Add repository
helm repo add rossigee https://charts.example.com
helm repo update

# Install
helm install provider-gitea rossigee/provider-gitea \
  --namespace crossplane-system \
  --set gitea.baseURL=https://git.example.com \
  --set gitea.token=YOUR_API_TOKEN

# Verify
helm list -n crossplane-system
kubectl get provider provider-gitea
```

**Details**: See [deploy/helm/provider-gitea/README.md](deploy/helm/provider-gitea/README.md)

---

### 3. Upbound `up` CLI - Enterprise/Cloud

Best for: Upbound control planes, enterprise features, marketplace discovery

```bash
# Install up CLI: https://docs.upbound.io/getting-started/

# Install provider
up ctp provider install xpkg.upbound.io/rossigee/provider-gitea:v0.8.9

# Or via kubectl
kubectl apply -f - <<EOF
apiVersion: pkg.crossplane.io/v1
kind: Provider
metadata:
  name: provider-gitea
spec:
  package: xpkg.upbound.io/rossigee/provider-gitea:v0.8.9
EOF
```

**Details**: See [deploy/upbound/README.md](deploy/upbound/README.md)

---

## Configuration

After installation, configure Gitea API access:

```bash
kubectl apply -f - <<EOF
apiVersion: gitea.crossplane.io/v1alpha1
kind: ProviderConfig
metadata:
  name: default
  namespace: crossplane-system
spec:
  credentials:
    source: Secret
    secretRef:
      namespace: crossplane-system
      name: gitea-credentials
      key: config

---
apiVersion: v1
kind: Secret
metadata:
  name: gitea-credentials
  namespace: crossplane-system
type: Opaque
stringData:
  config: |
    {
      "baseURL": "https://git.example.com",
      "token": "your-gitea-personal-access-token"
    }
EOF
```

## Usage Examples

### Create an Organization

```bash
kubectl apply -f - <<EOF
apiVersion: organization.gitea.crossplane.io/v1alpha1
kind: Organization
metadata:
  name: example-org
spec:
  forProvider:
    username: example-org
    fullName: Example Organization
  providerConfigRef:
    name: default
  deletionPolicy: Orphan
EOF
```

### Create a Repository

```bash
kubectl apply -f - <<EOF
apiVersion: repository.gitea.crossplane.io/v1alpha1
kind: Repository
metadata:
  name: example-repo
spec:
  forProvider:
    owner: example-org
    name: example-repo
    description: "Example repository"
    autoInit: true
  providerConfigRef:
    name: default
  deletionPolicy: Orphan
EOF
```

### Create a User

```bash
kubectl apply -f - <<EOF
apiVersion: user.gitea.crossplane.io/v1alpha1
kind: User
metadata:
  name: example-user
spec:
  forProvider:
    loginName: example-user
    fullName: Example User
    email: user@example.com
  providerConfigRef:
    name: default
  deletionPolicy: Orphan
EOF
```

## Version Support Matrix

| Method | v2.2.x | v2.3.x | v2.4.x | v2.5.x |
|--------|--------|--------|--------|--------|
| kubectl | ✅ | ✅ | ✅ | ✅ |
| Helm | ✅ | ✅ | ✅ | ✅ |
| Upbound | ⚠️ | ✅ | ✅ | ✅ |

**Note**: xpkg format was removed in Crossplane v2.4.0-rc.0+. All three methods use OCI image directly.

## Troubleshooting

### Check provider status

```bash
kubectl describe provider provider-gitea
kubectl logs -n crossplane-system -l app=provider-gitea
```

### Verify ProviderConfig is working

```bash
kubectl get providerconfig default
kubectl describe providerconfig default
```

### Check Gitea API credentials

```bash
kubectl get secret -n crossplane-system gitea-credentials
```

## Uninstallation

### kubectl

```bash
kubectl delete -k deploy/kubectl/
```

### Helm

```bash
helm uninstall provider-gitea -n crossplane-system
```

### Upbound

```bash
kubectl delete provider provider-gitea
```

## Support

- **GitHub**: https://github.com/rossigee/provider-gitea
- **Issues**: https://github.com/rossigee/provider-gitea/issues
- **Crossplane Docs**: https://docs.crossplane.io
- **Upbound Docs**: https://docs.upbound.io

## Version History

| Version | Release Date | Notes |
|---------|--------------|-------|
| v0.8.9 | 2026-07-08 | Fixed release workflow, added three distribution methods |
| v0.8.8 | 2026-06-09 | Last version with attempted xpkg support |
| Pre-v0.8.8 | Earlier | xpkg format incompatible with Crossplane v2.3.3+ |

## Key Differences Between Methods

### kubectl
- **Pros**: Simple, no external tools, easy to inspect and modify
- **Cons**: Manual secret management, no built-in upgrades
- **Best for**: Development, testing, DIY deployments

### Helm
- **Pros**: Templating, values management, standard K8s packaging, Artifact Hub discoverability
- **Cons**: Requires Helm CLI
- **Best for**: Production, enterprise, repeatable deployments

### Upbound
- **Pros**: Enterprise features, marketplace integration, managed control planes, dependency resolution
- **Cons**: Requires Upbound account/CLI
- **Best for**: SaaS deployments, enterprise workflows, cloud-native teams
