# Team

**API Version**: `team.gitea.m.crossplane.io/v1beta1`

Manages organization teams and permissions.

## Spec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `forProvider.name` | string | yes | Team name |
| `forProvider.organization` | string | yes | Parent organization |
| `forProvider.description` | string | no | Team description |
| `forProvider.permission` | string | no | Default permission (read, write, admin) |
| `forProvider.canCreateOrgRepo` | bool | no | Can create organization repos |
| `forProvider.includesAllRepositories` | bool | no | Access all repositories |
| `forProvider.units` | []string | no | Repository access units |

## Example

```yaml
apiVersion: team.gitea.m.crossplane.io/v1beta1
kind: Team
metadata:
  name: developers
  namespace: production
spec:
  forProvider:
    name: developers
    organization: myorg
    description: Development team
    permission: write
    includesAllRepositories: true
  providerConfigRef:
    name: default
```

## Behavior

- **Create**: Creates a new team
- **Update**: Updates team settings and permissions
- **Delete**: Deletes the team

## Status Fields

- `status.atProvider.id` — Team ID
- `status.atProvider.slug` — Team slug
- `status.atProvider.numMembers` — Member count
- `status.atProvider.numRepos` — Repository count
