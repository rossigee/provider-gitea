# OrganizationMember

**API Version**: `organizationmember.gitea.m.crossplane.io/v1beta1`

Manages organization membership and roles.

## Spec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `forProvider.organization` | string | yes | Organization name |
| `forProvider.username` | string | yes | Member username |
| `forProvider.role` | string | no | Member role (member, admin, owner) |

## Example

```yaml
apiVersion: organizationmember.gitea.m.crossplane.io/v1beta1
kind: OrganizationMember
metadata:
  name: member
  namespace: production
spec:
  forProvider:
    organization: myorg
    username: developer
    role: member
  providerConfigRef:
    name: default
```

## Behavior

- **Create**: Adds member to organization
- **Update**: Updates member role
- **Delete**: Removes member from organization

## Status Fields

- `status.atProvider.state` — Membership state
- `status.atProvider.url` — Membership URL
- `status.atProvider.organizationUrl` — Organization URL
