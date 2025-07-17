# Resource Reference

This document provides detailed information about all the managed resources provided by the Gitea provider.

## Repository

Manages Git repositories in a Gitea instance.

### Spec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Repository name |
| `owner` | string | No | Repository owner (user or organization) |
| `description` | string | No | Repository description |
| `private` | bool | No | Whether the repository is private (default: false) |
| `autoInit` | bool | No | Initialize with README (default: false) |
| `template` | bool | No | Mark as template repository (default: false) |
| `defaultBranch` | string | No | Default branch name (default: "main") |
| `website` | string | No | Repository website URL |
| `hasIssues` | bool | No | Enable issue tracker |
| `hasWiki` | bool | No | Enable wiki |
| `hasPullRequests` | bool | No | Enable pull requests |
| `allowMergeCommits` | bool | No | Allow merge commits |
| `allowRebase` | bool | No | Allow rebase merging |
| `allowSquashMerge` | bool | No | Allow squash merging |

### Status

| Field | Type | Description |
|-------|------|-------------|
| `id` | int64 | Repository ID |
| `fullName` | string | Full repository name (owner/name) |
| `htmlUrl` | string | Repository HTML URL |
| `sshUrl` | string | Repository SSH URL |
| `cloneUrl` | string | Repository clone URL |

## Organization

Manages organizations in a Gitea instance.

### Spec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `username` | string | Yes | Organization username |
| `name` | string | No | Display name |
| `fullName` | string | No | Full name |
| `description` | string | No | Organization description |
| `website` | string | No | Organization website |
| `location` | string | No | Organization location |
| `visibility` | string | No | Visibility (public, limited, private) |
| `repoAdminChangeTeamAccess` | bool | No | Allow repo admins to change team access |

### Status

| Field | Type | Description |
|-------|------|-------------|
| `id` | int64 | Organization ID |
| `email` | string | Organization email |
| `avatarUrl` | string | Avatar URL |

## User

Manages user accounts in a Gitea instance (admin only).

### Spec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `username` | string | Yes | Username |
| `email` | string | Yes | Email address |
| `password` | string | Yes | User password |
| `fullName` | string | No | Full name |
| `restricted` | bool | No | Restricted user account |
| `admin` | bool | No | Admin privileges |
| `active` | bool | No | Account is active |
| `website` | string | No | User website |
| `location` | string | No | User location |
| `description` | string | No | User description/bio |

### Status

| Field | Type | Description |
|-------|------|-------------|
| `id` | int64 | User ID |
| `avatarUrl` | string | Avatar URL |
| `isAdmin` | bool | Has admin privileges |
| `created` | string | Creation timestamp |

## Webhook

Manages webhooks for repositories or organizations.

### Spec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `repository` | string | No* | Repository name (for repo webhooks) |
| `owner` | string | No* | Repository owner |
| `organization` | string | No* | Organization name (for org webhooks) |
| `url` | string | Yes | Webhook payload URL |
| `type` | string | No | Webhook type (default: "gitea") |
| `contentType` | string | No | Content type (json, form) |
| `secret` | string | No | Webhook secret |
| `active` | bool | No | Webhook is active (default: true) |
| `events` | []string | No | Trigger events (default: ["push"]) |
| `sslVerification` | bool | No | Verify SSL certificates (default: true) |

*Either `repository`+`owner` or `organization` must be specified.

### Status

| Field | Type | Description |
|-------|------|-------------|
| `id` | int64 | Webhook ID |
| `createdAt` | string | Creation timestamp |
| `updatedAt` | string | Last update timestamp |

## Common Fields

All managed resources support these common fields:

### ResourceSpec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `providerConfigRef` | object | No | Reference to ProviderConfig |
| `deletionPolicy` | string | No | Deletion policy (Delete, Orphan) |
| `managementPolicies` | []string | No | Management policies |

### ResourceStatus

| Field | Type | Description |
|-------|------|-------------|
| `conditions` | []Condition | Resource conditions |
| `observedGeneration` | int64 | Last observed generation |

## Examples

See the `examples/` directory for complete working examples of each resource type.