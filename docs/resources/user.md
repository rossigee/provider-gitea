# User

**API Version**: `user.gitea.m.crossplane.io/v1beta1`

Manages user accounts (admin privileges required).

## Spec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `forProvider.username` | string | yes | Username |
| `forProvider.email` | string | yes | Email address |
| `forProvider.password` | string | yes | User password |
| `forProvider.fullName` | string | no | Full name |
| `forProvider.restricted` | bool | no | Restricted user account |
| `forProvider.admin` | bool | no | Admin privileges |
| `forProvider.active` | bool | no | Account is active |
| `forProvider.website` | string | no | User website |
| `forProvider.location` | string | no | User location |
| `forProvider.description` | string | no | User description/bio |

## Example

```yaml
apiVersion: user.gitea.m.crossplane.io/v1beta1
kind: User
metadata:
  name: developer
  namespace: production
spec:
  forProvider:
    username: developer
    email: dev@example.com
    password: secure-password-123
    fullName: Developer User
    active: true
  providerConfigRef:
    name: default
```

## Behavior

- **Create**: Creates a new user account
- **Update**: Updates user profile and settings
- **Delete**: Deletes the user account

## Status Fields

- `status.atProvider.id` — User ID
- `status.atProvider.avatarUrl` — Avatar URL
- `status.atProvider.isAdmin` — Admin flag
- `status.atProvider.created` — Creation timestamp
