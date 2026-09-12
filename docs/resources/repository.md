# Repository

**API Version**: `repository.gitea.m.crossplane.io/v2`

Manages Git repositories with comprehensive configuration options.

## Spec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `forProvider.name` | string | yes | Repository name |
| `forProvider.owner` | string | no | Repository owner (user or organization) |
| `forProvider.description` | string | no | Repository description |
| `forProvider.private` | bool | no | Whether the repository is private (default: false) |
| `forProvider.autoInit` | bool | no | Initialize with README (default: false) |
| `forProvider.template` | bool | no | Mark as template repository (default: false) |
| `forProvider.defaultBranch` | string | no | Default branch name (default: "main") |
| `forProvider.website` | string | no | Repository website URL |
| `forProvider.hasIssues` | bool | no | Enable issue tracker |
| `forProvider.hasWiki` | bool | no | Enable wiki |
| `forProvider.hasPullRequests` | bool | no | Enable pull requests |
| `forProvider.allowMergeCommits` | bool | no | Allow merge commits |
| `forProvider.allowRebase` | bool | no | Allow rebase merging |
| `forProvider.allowSquashMerge` | bool | no | Allow squash merging |

## Example

```yaml
apiVersion: repository.gitea.m.crossplane.io/v2
kind: Repository
metadata:
  name: my-repo
  namespace: production
spec:
  forProvider:
    owner: myorg
    description: My repository
    private: true
    defaultBranch: main
    hasIssues: true
    hasWiki: true
    hasPullRequests: true
  providerConfigRef:
    name: default
```

## Behavior

- **Create**: Creates a new repository with specified configuration
- **Update**: Updates repository settings, visibility, and features
- **Delete**: Deletes the repository (may be irreversible)

## Status Fields

- `status.atProvider.id` — Repository ID
- `status.atProvider.fullName` — Full repository name (owner/repo)
- `status.atProvider.htmlUrl` — Web URL
- `status.atProvider.sshUrl` — SSH clone URL
- `status.atProvider.cloneUrl` — HTTPS clone URL
