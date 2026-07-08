# Upbound Package Distribution

Installing provider-gitea via Upbound's `up` CLI and xpkg.upbound.io registry.

## Prerequisites

- Crossplane 2.2+
- Upbound `up` CLI (https://docs.upbound.io/getting-started/)
- (Optional) Upbound account at https://app.upbound.io

## Installation via `up` CLI

### Install from xpkg.upbound.io

```bash
up ctp provider install xpkg.upbound.io/rossigee/provider-gitea:v0.8.9
```

### Or via kubectl

If you have access to xpkg.upbound.io, you can also use kubectl directly:

```bash
kubectl apply -f - <<EOF
apiVersion: pkg.crossplane.io/v1
kind: Provider
metadata:
  name: provider-gitea
spec:
  package: xpkg.upbound.io/rossigee/provider-gitea:v0.8.9
  runtimeConfigRef:
    name: default
EOF
```

## Package Contents

The xpkg package includes:

- Provider binary (Crossplane controller)
- Custom Resource Definitions (CRDs) for:
  - Organizations
  - Repositories
  - Users
  - Teams
  - Webhooks
  - And more...

## Configuration

After installation, configure the provider with Gitea API credentials:

```bash
kubectl apply -f - <<EOF
apiVersion: gitea.crossplane.io/v1alpha1
kind: ProviderConfig
metadata:
  name: default
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
      "token": "your-gitea-api-token"
    }
EOF
```

## Upbound Control Plane Integration

If using Upbound's managed control plane:

1. Log in to https://app.upbound.io
2. Navigate to your control plane
3. Go to Packages → Providers
4. Search for `provider-gitea`
5. Click Install

Or use `up` CLI:

```bash
up login
up ctp provider install xpkg.upbound.io/rossigee/provider-gitea:v0.8.9 \
  --account YOUR_UPBOUND_ACCOUNT \
  --control-plane YOUR_CONTROL_PLANE
```

## Package Versions

All versions are published to: `xpkg.upbound.io/rossigee/provider-gitea`

List available versions:

```bash
up registry pull xpkg.upbound.io/rossigee/provider-gitea --all
```

## Troubleshooting

### Package not found

Ensure you have access to xpkg.upbound.io. If private, check your Upbound account credentials:

```bash
up login
```

### Installation fails

Check provider logs:

```bash
kubectl logs -n crossplane-system -l pkg.crossplane.io/provider=provider-gitea
```

## Support

For issues and support:
- GitHub: https://github.com/rossigee/provider-gitea
- Upbound Marketplace: https://marketplace.upbound.io
