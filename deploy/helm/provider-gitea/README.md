# provider-gitea Helm Chart

A Helm chart for deploying provider-gitea - the Crossplane provider for Gitea.

## Prerequisites

- Kubernetes 1.20+
- Crossplane 2.2+
- Helm 3+

## Installation

### Add the repository

```bash
helm repo add rossigee https://charts.example.com
helm repo update
```

### Install with default values

```bash
helm install provider-gitea rossigee/provider-gitea \
  --namespace crossplane-system
```

### Install with custom values

```bash
helm install provider-gitea rossigee/provider-gitea \
  --namespace crossplane-system \
  --values values-prod.yaml
```

### Install with Gitea credentials

```bash
helm install provider-gitea rossigee/provider-gitea \
  --namespace crossplane-system \
  --set gitea.baseURL=https://git.example.com \
  --set gitea.token=YOUR_API_TOKEN
```

## Values

See `values.yaml` for all available configuration options:

- `image.repository` - Docker image repository
- `image.tag` - Docker image tag
- `image.pullPolicy` - Image pull policy (Always/IfNotPresent/Never)
- `gitea.baseURL` - Gitea API base URL
- `gitea.token` - Gitea API token
- `resources` - Pod resource limits and requests
- `nodeSelector` - Node selector for pod placement
- `tolerations` - Pod tolerations
- `affinity` - Pod affinity rules

## Uninstallation

```bash
helm uninstall provider-gitea --namespace crossplane-system
```

## Examples

### Create an Organization

```bash
kubectl apply -f - <<EOF
apiVersion: organization.gitea.crossplane.io/v1alpha1
kind: Organization
metadata:
  name: my-org
spec:
  forProvider:
    username: my-org
    fullName: My Organization
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
  name: my-repo
spec:
  forProvider:
    owner: my-org
    name: my-repo
    description: My repository
    autoInit: true
  providerConfigRef:
    name: default
  deletionPolicy: Orphan
EOF
```

## Support

For issues, feature requests, or contributions, visit:
https://github.com/rossigee/provider-gitea
