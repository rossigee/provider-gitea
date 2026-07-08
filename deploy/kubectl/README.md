# provider-gitea - Direct kubectl Deployment

Install provider-gitea using Kubernetes manifests without Helm or external tools.

## Quick Start

```bash
# Apply all manifests
kubectl apply -k deploy/kubectl/

# Verify installation
kubectl get provider provider-gitea
kubectl get pods -n crossplane-system -l app=provider-gitea
```

## Files

- `00-namespace.yaml` - Creates `crossplane-system` namespace
- `10-serviceaccount.yaml` - ServiceAccount for provider
- `20-clusterrole.yaml` - ClusterRole with required permissions
- `30-clusterrolebinding.yaml` - Binds ClusterRole to ServiceAccount
- `40-provider.yaml` - Provider resource
- `50-providerconfig.yaml` - ProviderConfig for Gitea API access

## Configuration

Edit `50-providerconfig.yaml` to set your Gitea API endpoint and authentication:

```yaml
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
      key: credentials
```

Create the secret:

```bash
kubectl create secret generic gitea-credentials \
  -n crossplane-system \
  --from-literal=credentials='baseURL=https://git.example.com&token=YOUR_TOKEN'
```

## Examples

See `examples/` directory for usage examples.
