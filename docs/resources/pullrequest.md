# PullRequest

**API Version**: `pullrequest.gitea.m.crossplane.io/v2`

Manages pull requests and code review workflows.

## Spec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `forProvider.repository` | string | yes | Repository name |
| `forProvider.owner` | string | yes | Repository owner |
| `forProvider.title` | string | yes | Pull request title |
| `forProvider.body` | string | no | Pull request description |
| `forProvider.head` | string | yes | Head branch |
| `forProvider.base` | string | yes | Base branch |
| `forProvider.assignees` | []string | no | Assigned reviewers |
| `forProvider.milestone` | int64 | no | Milestone ID |
| `forProvider.labels` | []string | no | Pull request labels |

## Example

```yaml
apiVersion: pullrequest.gitea.m.crossplane.io/v2
kind: PullRequest
metadata:
  name: new-pr
  namespace: production
spec:
  forProvider:
    repository: my-repo
    owner: myorg
    title: Add new feature
    body: Implements the new feature
    head: feature-branch
    base: main
  providerConfigRef:
    name: default
```

## Behavior

- **Create**: Creates a new pull request
- **Update**: Updates PR content, labels, or target branch
- **Delete**: Closes or deletes the pull request

## Status Fields

- `status.atProvider.id` — PR ID
- `status.atProvider.number` — PR number
- `status.atProvider.state` — PR state
- `status.atProvider.mergeable` — Mergeability status
- `status.atProvider.createdAt` — Creation timestamp
- `status.atProvider.updatedAt` — Last update timestamp
