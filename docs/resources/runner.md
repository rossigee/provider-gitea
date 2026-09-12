# Runner

**API Version**: `runner.gitea.m.crossplane.io/v2`

Manages self-hosted runners for CI/CD execution.

## Spec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `forProvider.name` | string | yes | Runner name |
| `forProvider.repository` | string | no* | Repository name (for repo runners) |
| `forProvider.owner` | string | no* | Repository owner (for repo runners) |
| `forProvider.organization` | string | no* | Organization name (for org runners) |
| `forProvider.token` | string | yes | Runner registration token |
| `forProvider.labels` | []string | no | Runner labels |
| `forProvider.description` | string | no | Runner description |

*Specify either `repository`+`owner` for repo runners or `organization` for org runners.

## Example

```yaml
apiVersion: runner.gitea.m.crossplane.io/v2
kind: Runner
metadata:
  name: ci-runner
  namespace: production
spec:
  forProvider:
    name: ci-runner
    organization: myorg
    token: runner-registration-token
    labels:
      - ubuntu-latest
      - docker
    description: CI runner for myorg
  providerConfigRef:
    name: default
```

## Behavior

- **Create**: Registers a new runner
- **Update**: Updates runner labels and description
- **Delete**: Removes the runner registration

## Status Fields

- `status.atProvider.id` — Runner ID
- `status.atProvider.uuid` — Runner UUID
- `status.atProvider.status` — Runner status
- `status.atProvider.lastOnline` — Last online timestamp
