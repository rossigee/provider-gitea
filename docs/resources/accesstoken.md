# AccessToken

**API Version**: `accesstoken.gitea.m.crossplane.io/v1beta1`

Manages scoped API tokens for automation.

## Spec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `forProvider.name` | string | yes | Token name |
| `forProvider.scopes` | []string | no | Token scopes |
| `forProvider.username` | string | yes | Token owner username |

## Example

```yaml
apiVersion: accesstoken.gitea.m.crossplane.io/v1beta1
kind: AccessToken
metadata:
  name: ci-token
  namespace: production
spec:
  forProvider:
    name: ci-automation
    scopes:
      - repo
      - write:user
    username: ci-bot
  providerConfigRef:
    name: default
```

## Behavior

- **Create**: Generates a new API token
- **Update**: Token name update (may regenerate)
- **Delete**: Revokes the token

## Status Fields

- `status.atProvider.id` — Token ID
- `status.atProvider.token` — Token value (write-only)
- `status.atProvider.sha1` — Token hash
- `status.atProvider.lastEight` — Last 8 characters
