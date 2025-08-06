# Resource Reference

This document provides detailed information about all 22 managed resources provided by the enterprise-grade Gitea provider.

## Overview

The Gitea provider supports comprehensive enterprise Git infrastructure management through four main categories:

- **[Core Resources](#core-resources)** - Basic Git infrastructure (7 resources)
- **[Security Resources](#security-resources)** - Enterprise security features (7 resources)
- **[CI/CD Resources](#cicd-resources)** - DevOps automation (2 resources)
- **[Administrative Resources](#administrative-resources)** - Enterprise administration (6 resources)

## Core Resources

### Repository
Manages Git repositories with comprehensive configuration options.

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

**Status Fields**: `id`, `fullName`, `htmlUrl`, `sshUrl`, `cloneUrl`

### Organization
Manages organizations with comprehensive policy controls.

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

**Status Fields**: `id`, `email`, `avatarUrl`

### User
Manages user accounts (admin privileges required).

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

**Status Fields**: `id`, `avatarUrl`, `isAdmin`, `created`

### Webhook
Manages webhooks for repositories and organizations.

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

**Status Fields**: `id`, `createdAt`, `updatedAt`

### Team
Manages organization teams and permissions.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Team name |
| `organization` | string | Yes | Parent organization |
| `description` | string | No | Team description |
| `permission` | string | No | Default permission (read, write, admin) |
| `canCreateOrgRepo` | bool | No | Can create organization repos |
| `includesAllRepositories` | bool | No | Access all repositories |
| `units` | []string | No | Repository access units |

**Status Fields**: `id`, `slug`, `numMembers`, `numRepos`

### Label
Manages issue and pull request labels.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Label name |
| `repository` | string | Yes | Repository name |
| `owner` | string | Yes | Repository owner |
| `color` | string | Yes | Label color (hex format) |
| `description` | string | No | Label description |
| `exclusive` | bool | No | Mutually exclusive label |

**Status Fields**: `id`, `url`

### RepositoryCollaborator
Manages repository collaboration and access control.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `repository` | string | Yes | Repository name |
| `owner` | string | Yes | Repository owner |
| `username` | string | Yes | Collaborator username |
| `permission` | string | No | Permission level (read, write, admin) |

**Status Fields**: `id`, `fullName`, `email`

## Security Resources

### BranchProtection
Enterprise-grade branch protection with approval workflows.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `repository` | string | Yes | Repository name |
| `owner` | string | Yes | Repository owner |
| `branch` | string | Yes | Branch name or pattern |
| `enablePush` | bool | No | Allow direct pushes |
| `enablePushWhitelist` | bool | No | Use push whitelist |
| `pushWhitelistUsernames` | []string | No | Users allowed to push |
| `pushWhitelistTeams` | []string | No | Teams allowed to push |
| `enableMergeWhitelist` | bool | No | Use merge whitelist |
| `mergeWhitelistUsernames` | []string | No | Users allowed to merge |
| `mergeWhitelistTeams` | []string | No | Teams allowed to merge |
| `enableStatusCheck` | bool | No | Require status checks |
| `statusCheckContexts` | []string | No | Required status contexts |
| `requiredApprovingReviewCount` | int32 | No | Required approvals |
| `enableApprovingReviewWhitelist` | bool | No | Use approval whitelist |
| `approvingWhitelistUsernames` | []string | No | Users who can approve |
| `approvingWhitelistTeams` | []string | No | Teams who can approve |
| `blockOnRejectedReviews` | bool | No | Block on rejected reviews |
| `dismissStaleReviews` | bool | No | Dismiss stale reviews |
| `requireCodeOwnerReviews` | bool | No | Require code owner reviews |
| `blockOnOutdatedBranch` | bool | No | Block on outdated branch |

**Status Fields**: `id`, `createdAt`, `updatedAt`

### RepositoryKey
Manages SSH deployment keys for repositories.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `repository` | string | Yes | Repository name |
| `owner` | string | Yes | Repository owner |
| `title` | string | Yes | Key title/name |
| `key` | string | Yes | SSH public key content |
| `readOnly` | bool | No | Read-only access (default: true) |

**Status Fields**: `id`, `fingerprint`, `createdAt`

### AccessToken
Manages scoped API tokens for automation.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Token name |
| `scopes` | []string | No | Token scopes |
| `username` | string | Yes | Token owner username |

**Status Fields**: `id`, `token`, `sha1`, `lastEight`

### RepositorySecret
Manages CI/CD secrets with Kubernetes integration.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `repository` | string | Yes | Repository name |
| `owner` | string | Yes | Repository owner |
| `secretName` | string | Yes | Secret name |
| `data` | string | No* | Direct secret value |
| `secretRef` | SecretRef | No* | Kubernetes secret reference |

*Either `data` or `secretRef` must be specified.

**Status Fields**: `createdAt`, `updatedAt`

### UserKey
Manages SSH keys for user accounts.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `title` | string | Yes | Key title/name |
| `key` | string | Yes | SSH public key content |
| `username` | string | Yes | Key owner username |
| `readOnly` | bool | No | Read-only access |

**Status Fields**: `id`, `fingerprint`, `createdAt`

### OrganizationMember
Manages organization membership and roles.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `organization` | string | Yes | Organization name |
| `username` | string | Yes | Member username |
| `role` | string | No | Member role (member, admin, owner) |

**Status Fields**: `state`, `url`, `organizationUrl`

### OrganizationSecret
Manages organization-wide CI/CD secrets.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `organization` | string | Yes | Organization name |
| `secretName` | string | Yes | Secret name |
| `data` | string | No* | Direct secret value |
| `secretRef` | SecretRef | No* | Kubernetes secret reference |
| `visibility` | string | No | Secret visibility (all, private, selected) |
| `selectedRepositoryNames` | []string | No | Selected repositories (for selected visibility) |

*Either `data` or `secretRef` must be specified.

**Status Fields**: `createdAt`, `updatedAt`

## CI/CD Resources

### Action
Manages CI/CD workflows and pipeline automation.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `repository` | string | Yes | Repository name |
| `owner` | string | Yes | Repository owner |
| `filename` | string | Yes | Workflow filename (e.g., ".gitea/workflows/ci.yaml") |
| `content` | string | Yes | Workflow YAML content |
| `active` | bool | No | Workflow is active (default: true) |

**Status Fields**: `id`, `state`, `createdAt`, `updatedAt`

### Runner
Manages self-hosted runners for CI/CD execution.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Runner name |
| `repository` | string | No* | Repository name (for repo runners) |
| `owner` | string | No* | Repository owner (for repo runners) |
| `organization` | string | No* | Organization name (for org runners) |
| `token` | string | Yes | Runner registration token |
| `labels` | []string | No | Runner labels |
| `description` | string | No | Runner description |

*Specify either `repository`+`owner` for repo runners or `organization` for org runners.

**Status Fields**: `id`, `uuid`, `status`, `lastOnline`

## Administrative Resources

### AdminUser
Manages administrative users and service accounts.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `username` | string | Yes | Username |
| `email` | string | Yes | Email address |
| `fullName` | string | No | Full name |
| `loginName` | string | No | Login name |
| `password` | string | No* | Direct password |
| `passwordRef` | SecretRef | No* | Kubernetes secret reference for password |
| `mustChangePassword` | bool | No | Force password change on first login |
| `sendNotify` | bool | No | Send welcome email |
| `admin` | bool | No | Grant admin privileges |
| `restricted` | bool | No | Restricted account |

*Either `password` or `passwordRef` must be specified.

**Status Fields**: `id`, `avatarUrl`, `created`, `lastLogin`

### OrganizationSettings
Manages organization-wide policies and settings.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `organization` | string | Yes | Organization name |
| `fullName` | string | No | Organization display name |
| `description` | string | No | Organization description |
| `website` | string | No | Organization website |
| `location` | string | No | Organization location |
| `visibility` | string | No | Default visibility (public, limited, private) |
| `maxRepoCreation` | int32 | No | Maximum repositories |
| `repoAdminChangeTeamAccess` | bool | No | Allow repo admins to change team access |

**Status Fields**: `updatedAt`

### GitHook
Manages server-side Git hooks for policy enforcement.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `repository` | string | Yes | Repository name |
| `owner` | string | Yes | Repository owner |
| `hookType` | string | Yes | Hook type (pre-receive, post-receive, update) |
| `content` | string | Yes | Hook script content |

**Status Fields**: `lastUpdated`

### Issue
Manages repository issues and tracking.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `repository` | string | Yes | Repository name |
| `owner` | string | Yes | Repository owner |
| `title` | string | Yes | Issue title |
| `body` | string | No | Issue body/description |
| `assignees` | []string | No | Assigned usernames |
| `milestone` | int64 | No | Milestone ID |
| `labels` | []string | No | Issue labels |
| `closed` | bool | No | Issue is closed |

**Status Fields**: `id`, `number`, `state`, `createdAt`, `updatedAt`

### PullRequest
Manages pull requests and code review workflows.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `repository` | string | Yes | Repository name |
| `owner` | string | Yes | Repository owner |
| `title` | string | Yes | Pull request title |
| `body` | string | No | Pull request description |
| `head` | string | Yes | Head branch |
| `base` | string | Yes | Base branch |
| `assignees` | []string | No | Assigned reviewers |
| `milestone` | int64 | No | Milestone ID |
| `labels` | []string | No | Pull request labels |

**Status Fields**: `id`, `number`, `state`, `mergeable`, `createdAt`, `updatedAt`

### Release
Manages repository releases and version tags.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `repository` | string | Yes | Repository name |
| `owner` | string | Yes | Repository owner |
| `tagName` | string | Yes | Git tag name |
| `name` | string | No | Release name |
| `body` | string | No | Release notes |
| `draft` | bool | No | Draft release |
| `prerelease` | bool | No | Pre-release version |
| `targetCommitish` | string | No | Target branch or commit |

**Status Fields**: `id`, `url`, `assetsUrl`, `tarballUrl`, `zipballUrl`, `createdAt`, `publishedAt`

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

See the `examples/` directory for complete working examples of each resource type:

- [Repository Examples](../examples/repository/)
- [Security Examples](../examples/branchprotection/, ../examples/accesstoken/)
- [CI/CD Examples](../examples/action/, ../examples/runner/)
- [Administrative Examples](../examples/adminuser/, ../examples/organizationsettings/)

## Resource Relationships

### Enterprise Security Workflow
```
Organization -> OrganizationSettings -> OrganizationMember -> Team
     |              |                       |                  |
     v              v                       v                  v
Repository -> BranchProtection -> RepositoryCollaborator -> Label
     |              |                       |                  |
     v              v                       v                  v
AccessToken -> RepositoryKey -> RepositorySecret -> GitHook
```

### CI/CD Integration Workflow
```
Repository -> Action -> Runner
    |           |        |
    v           v        v
OrganizationSecret -> RepositorySecret
```

### Administrative Workflow
```
AdminUser -> Organization -> OrganizationSettings
    |            |                |
    v            v                v
User -> OrganizationMember -> Team
```

This comprehensive resource catalog enables complete enterprise Git infrastructure management through declarative Kubernetes manifests.