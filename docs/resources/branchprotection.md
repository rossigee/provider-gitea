# BranchProtection

**API Version**: `branchprotection.gitea.m.crossplane.io/v1beta1`

Enterprise-grade branch protection with approval workflows.

## Spec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `forProvider.repository` | string | yes | Repository name |
| `forProvider.owner` | string | yes | Repository owner |
| `forProvider.branch` | string | yes | Branch name or pattern |
| `forProvider.enablePush` | bool | no | Allow direct pushes |
| `forProvider.enablePushWhitelist` | bool | no | Use push whitelist |
| `forProvider.pushWhitelistUsernames` | []string | no | Users allowed to push |
| `forProvider.pushWhitelistTeams` | []string | no | Teams allowed to push |
| `forProvider.enableMergeWhitelist` | bool | no | Use merge whitelist |
| `forProvider.mergeWhitelistUsernames` | []string | no | Users allowed to merge |
| `forProvider.mergeWhitelistTeams` | []string | no | Teams allowed to merge |
| `forProvider.enableStatusCheck` | bool | no | Require status checks |
| `forProvider.statusCheckContexts` | []string | no | Required status contexts |
| `forProvider.requiredApprovingReviewCount` | int32 | no | Required approvals |
| `forProvider.enableApprovingReviewWhitelist` | bool | no | Use approval whitelist |
| `forProvider.approvingWhitelistUsernames` | []string | no | Users who can approve |
| `forProvider.approvingWhitelistTeams` | []string | no | Teams who can approve |
| `forProvider.blockOnRejectedReviews` | bool | no | Block on rejected reviews |
| `forProvider.dismissStaleReviews` | bool | no | Dismiss stale reviews |
| `forProvider.requireCodeOwnerReviews` | bool | no | Require code owner reviews |
| `forProvider.blockOnOutdatedBranch` | bool | no | Block on outdated branch |

## Example

```yaml
apiVersion: branchprotection.gitea.m.crossplane.io/v1beta1
kind: BranchProtection
metadata:
  name: main-protection
  namespace: production
spec:
  forProvider:
    repository: my-repo
    owner: myorg
    branch: main
    enablePush: false
    enableStatusCheck: true
    requiredApprovingReviewCount: 2
    dismissStaleReviews: true
  providerConfigRef:
    name: default
```

## Behavior

- **Create**: Creates branch protection rules
- **Update**: Updates protection settings
- **Delete**: Removes branch protection

## Status Fields

- `status.atProvider.id` — Rule ID
- `status.atProvider.createdAt` — Creation timestamp
- `status.atProvider.updatedAt` — Last update timestamp
