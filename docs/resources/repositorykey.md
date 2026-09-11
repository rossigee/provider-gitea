# RepositoryKey

**API Version**: `repositorykey.gitea.m.crossplane.io/v1beta1`

Manages SSH deployment keys for repositories.

## Spec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `forProvider.repository` | string | yes | Repository name |
| `forProvider.owner` | string | yes | Repository owner |
| `forProvider.title` | string | yes | Key title/name |
| `forProvider.key` | string | yes | SSH public key content |
| `forProvider.readOnly` | bool | no | Read-only access (default: true) |

## Example

```yaml
apiVersion: repositorykey.gitea.m.crossplane.io/v1beta1
kind: RepositoryKey
metadata:
  name: deploy-key
  namespace: production
spec:
  forProvider:
    repository: my-repo
    owner: myorg
    title: deploy-key
    key: "ssh-rsa AAAAB3NzaC1..."
    readOnly: true
  providerConfigRef:
    name: default
```

## Behavior

- **Create**: Adds SSH deploy key to repository
- **Update**: Updates key title
- **Delete**: Removes the deploy key

## Status Fields

- `status.atProvider.id` — Key ID
- `status.atProvider.fingerprint` — Key fingerprint
- `status.atProvider.createdAt` — Creation timestamp
