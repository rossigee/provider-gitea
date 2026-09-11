# UserKey

**API Version**: `userkey.gitea.m.crossplane.io/v1beta1`

Manages SSH keys for user accounts.

## Spec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `forProvider.title` | string | yes | Key title/name |
| `forProvider.key` | string | yes | SSH public key content |
| `forProvider.username` | string | yes | Key owner username |
| `forProvider.readOnly` | bool | no | Read-only access |

## Example

```yaml
apiVersion: userkey.gitea.m.crossplane.io/v1beta1
kind: UserKey
metadata:
  name: ssh-key
  namespace: production
spec:
  forProvider:
    title: work-laptop
    key: "ssh-rsa AAAAB3NzaC1..."
    username: developer
  providerConfigRef:
    name: default
```

## Behavior

- **Create**: Adds SSH key to user account
- **Update**: Updates key title
- **Delete**: Removes the SSH key

## Status Fields

- `status.atProvider.id` — Key ID
- `status.atProvider.fingerprint` — Key fingerprint
- `status.atProvider.createdAt` — Creation timestamp
