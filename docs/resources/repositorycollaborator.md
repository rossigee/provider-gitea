# RepositoryCollaborator

**API Version**: `repositorycollaborator.gitea.m.crossplane.io/v2`

Manages repository collaboration and access control.

## Spec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `forProvider.repository` | string | yes | Repository name |
| `forProvider.owner` | string | yes | Repository owner |
| `forProvider.username` | string | yes | Collaborator username |
| `forProvider.permission` | string | no | Permission level (read, write, admin) |

## Example

```yaml
apiVersion: repositorycollaborator.gitea.m.crossplane.io/v2
kind: RepositoryCollaborator
metadata:
  name: collab
  namespace: production
spec:
  forProvider:
    repository: my-repo
    owner: myorg
    username: developer
    permission: write
  providerConfigRef:
    name: default
```

## Behavior

- **Create**: Adds a collaborator to the repository
- **Update**: Updates collaborator permission level
- **Delete**: Removes collaborator from repository

## Status Fields

- `status.atProvider.id` — Collaborator ID
- `status.atProvider.fullName` — Full name
- `status.atProvider.email` — Email address
