# Organization

**API Version**: `organization.gitea.m.crossplane.io/v2`

Manages organizations with comprehensive policy controls.

## Spec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `forProvider.username` | string | yes | Organization username |
| `forProvider.name` | string | no | Display name |
| `forProvider.fullName` | string | no | Full name |
| `forProvider.description` | string | no | Organization description |
| `forProvider.website` | string | no | Organization website |
| `forProvider.location` | string | no | Organization location |
| `forProvider.visibility` | string | no | Visibility (public, limited, private) |
| `forProvider.repoAdminChangeTeamAccess` | bool | no | Allow repo admins to change team access |

## Example

```yaml
apiVersion: organization.gitea.m.crossplane.io/v2
kind: Organization
metadata:
  name: myorg
  namespace: production
spec:
  forProvider:
    name: My Organization
    fullName: My Organization Ltd
    description: Enterprise Git infrastructure
    website: https://myorg.com
    location: San Francisco, CA
    visibility: private
  providerConfigRef:
    name: default
```

## Behavior

- **Create**: Creates a new organization
- **Update**: Updates organization settings and profile
- **Delete**: Deletes the organization and all owned repositories

## Status Fields

- `status.atProvider.id` — Organization ID
- `status.atProvider.email` — Organization email
- `status.atProvider.avatarUrl` — Avatar URL
