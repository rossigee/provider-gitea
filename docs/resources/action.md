# Action

**API Version**: `action.gitea.m.crossplane.io/v1beta1`

Manages CI/CD workflows and pipeline automation.

## Spec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `forProvider.repository` | string | yes | Repository name |
| `forProvider.owner` | string | yes | Repository owner |
| `forProvider.filename` | string | yes | Workflow filename |
| `forProvider.content` | string | yes | Workflow YAML content |
| `forProvider.active` | bool | no | Workflow is active (default: true) |

## Example

```yaml
apiVersion: action.gitea.m.crossplane.io/v1beta1
kind: Action
metadata:
  name: ci-workflow
  namespace: production
spec:
  forProvider:
    repository: my-repo
    owner: myorg
    filename: .gitea/workflows/ci.yaml
    content: |
      name: CI
      on: [push, pull_request]
      jobs:
        build:
          runs-on: ubuntu-latest
          steps:
            - uses: actions/checkout@v3
  providerConfigRef:
    name: default
```

## Behavior

- **Create**: Creates or updates a workflow file
- **Update**: Updates workflow content
- **Delete**: Removes the workflow file

## Status Fields

- `status.atProvider.id` — Workflow ID
- `status.atProvider.state` — Workflow state
- `status.atProvider.createdAt` — Creation timestamp
- `status.atProvider.updatedAt` — Last update timestamp
