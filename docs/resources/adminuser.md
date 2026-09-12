# AdminUser

**API Version**: `adminuser.gitea.m.crossplane.io/v2`

Manages administrative users and service accounts.

## Spec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `forProvider.username` | string | yes | Username |
| `forProvider.email` | string | yes | Email address |
| `forProvider.fullName` | string | no | Full name |
| `forProvider.loginName` | string | no | Login name |
| `forProvider.password` | string | no* | Direct password |
| `forProvider.passwordRef` | object | no* | Kubernetes secret reference for password |
| `forProvider.mustChangePassword` | bool | no | Force password change on first login |
| `forProvider.sendNotify` | bool | no | Send welcome email |
| `forProvider.admin` | bool | no | Grant admin privileges |
| `forProvider.restricted` | bool | no | Restricted account |

*Either `password` or `passwordRef` must be specified.

## Example

```yaml
apiVersion: adminuser.gitea.m.crossplane.io/v2
kind: AdminUser
metadata:
  name: service-account
  namespace: production
spec:
  forProvider:
    username: service-account
    email: sa@example.com
    fullName: Service Account
    password: secure-password
    admin: false
    restricted: true
  providerConfigRef:
    name: default
```

## Behavior

- **Create**: Creates an admin user account
- **Update**: Updates user settings
- **Delete**: Deletes the user account

## Status Fields

- `status.atProvider.id` — User ID
- `status.atProvider.avatarUrl` — Avatar URL
- `status.atProvider.created` — Creation timestamp
- `status.atProvider.lastLogin` — Last login timestamp
