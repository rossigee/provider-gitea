# GitHook

**API Version**: `githook.gitea.m.crossplane.io/v1beta1`

Manages server-side Git hooks for policy enforcement.

## Spec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `forProvider.repository` | string | yes | Repository name |
| `forProvider.owner` | string | yes | Repository owner |
| `forProvider.hookType` | string | yes | Hook type (pre-receive, post-receive, update) |
| `forProvider.content` | string | yes | Hook script content |

## Example

```yaml
apiVersion: githook.gitea.m.crossplane.io/v1beta1
kind: GitHook
metadata:
  name: pre-receive-hook
  namespace: production
spec:
  forProvider:
    repository: my-repo
    owner: myorg
    hookType: pre-receive
    content: |
      #!/bin/bash
      echo "Running pre-receive hook"
      exit 0
  providerConfigRef:
    name: default
```

## Behavior

- **Create**: Creates a server-side Git hook
- **Update**: Updates hook script content
- **Delete**: Removes the hook

## Status Fields

- `status.atProvider.lastUpdated` — Last update timestamp
