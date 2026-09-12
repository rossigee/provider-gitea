# RepositorySecret

**API Version**: `repositorysecret.gitea.m.crossplane.io/v2`

Manages CI/CD secrets with Kubernetes integration.

## Spec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `forProvider.repository` | string | yes | Repository name |
| `forProvider.owner` | string | yes | Repository owner |
| `forProvider.secretName` | string | yes | Secret name |
| `forProvider.data` | string | no* | Direct secret value |
| `forProvider.secretRef` | object | no* | Kubernetes secret reference |

*Either `data` or `secretRef` must be specified.

## Example

```yaml
apiVersion: repositorysecret.gitea.m.crossplane.io/v2
kind: RepositorySecret
metadata:
  name: api-key
  namespace: production
spec:
  forProvider:
    repository: my-repo
    owner: myorg
    secretName: API_KEY
    secretRef:
      name: k8s-secret
      namespace: production
      key: api-key
  providerConfigRef:
    name: default
```

## Behavior

- **Create**: Creates a repository secret
- **Update**: Updates secret value
- **Delete**: Removes the secret

## Status Fields

- `status.atProvider.createdAt` — Creation timestamp
- `status.atProvider.updatedAt` — Last update timestamp
