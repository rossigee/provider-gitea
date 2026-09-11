# Issue

**API Version**: `issue.gitea.m.crossplane.io/v1beta1`

Manages repository issues and tracking.

## Spec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `forProvider.repository` | string | yes | Repository name |
| `forProvider.owner` | string | yes | Repository owner |
| `forProvider.title` | string | yes | Issue title |
| `forProvider.body` | string | no | Issue body/description |
| `forProvider.assignees` | []string | no | Assigned usernames |
| `forProvider.milestone` | int64 | no | Milestone ID |
| `forProvider.labels` | []string | no | Issue labels |
| `forProvider.closed` | bool | no | Issue is closed |

## Example

```yaml
apiVersion: issue.gitea.m.crossplane.io/v1beta1
kind: Issue
metadata:
  name: new-issue
  namespace: production
spec:
  forProvider:
    repository: my-repo
    owner: myorg
    title: Fix authentication bug
    body: Users are unable to log in
    labels:
      - bug
      - priority
  providerConfigRef:
    name: default
```

## Behavior

- **Create**: Creates a new issue
- **Update**: Updates issue content, labels, or state
- **Delete**: Closes or deletes the issue

## Status Fields

- `status.atProvider.id` — Issue ID
- `status.atProvider.number` — Issue number
- `status.atProvider.state` — Issue state
- `status.atProvider.createdAt` — Creation timestamp
- `status.atProvider.updatedAt` — Last update timestamp
