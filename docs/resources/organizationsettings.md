# OrganizationSettings

**API Version**: `organizationsettings.gitea.m.crossplane.io/v2`

Manages organization-wide policies and settings.

## Spec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `forProvider.organization` | string | yes | Organization name |
| `forProvider.fullName` | string | no | Organization display name |
| `forProvider.description` | string | no | Organization description |
| `forProvider.website` | string | no | Organization website |
| `forProvider.location` | string | no | Organization location |
| `forProvider.visibility` | string | no | Default visibility (public, limited, private) |
| `forProvider.maxRepoCreation` | int32 | no | Maximum repositories |
| `forProvider.repoAdminChangeTeamAccess` | bool | no | Allow repo admins to change team access |

## Example

```yaml
apiVersion: organizationsettings.gitea.m.crossplane.io/v2
kind: OrganizationSettings
metadata:
  name: org-settings
  namespace: production
spec:
  forProvider:
    organization: myorg
    fullName: My Organization
    description: Enterprise Git infrastructure
    website: https://myorg.com
    location: San Francisco
    visibility: private
    maxRepoCreation: 100
  providerConfigRef:
    name: default
```

## Behavior

- **Create**: Creates organization settings
- **Update**: Updates organization policies
- **Delete**: Removes custom settings (resets to defaults)

## Status Fields

- `status.atProvider.updatedAt` — Last update timestamp
