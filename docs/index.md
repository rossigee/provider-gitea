# Provider Gitea Documentation

A Crossplane v2 provider for managing Gitea resources with complete namespace isolation for multi-tenancy.

## Quick Links

- [Configuration](configuration.md) — Authentication and connection setup
- [Development](development.md) — Building, testing, and contributing

## Resource Documentation

### Core Resources

| Resource | API Group | Description |
|----------|-----------|-------------|
| [Repository](resources/repository.md) | `repository.gitea.m.crossplane.io/v1beta1` | Git repository management |
| [Organization](resources/organization.md) | `organization.gitea.m.crossplane.io/v1beta1` | Organization management |
| [User](resources/user.md) | `user.gitea.m.crossplane.io/v1beta1` | User account management |
| [Webhook](resources/webhook.md) | `webhook.gitea.m.crossplane.io/v1beta1` | Repository and org webhooks |
| [Team](resources/team.md) | `team.gitea.m.crossplane.io/v1beta1` | Team and permissions |
| [Label](resources/label.md) | `label.gitea.m.crossplane.io/v1beta1` | Issue and PR labels |
| [RepositoryCollaborator](resources/repositorycollaborator.md) | `repositorycollaborator.gitea.m.crossplane.io/v1beta1` | Repository collaboration |

### Security Resources

| Resource | API Group | Description |
|----------|-----------|-------------|
| [BranchProtection](resources/branchprotection.md) | `branchprotection.gitea.m.crossplane.io/v1beta1` | Branch protection rules |
| [RepositoryKey](resources/repositorykey.md) | `repositorykey.gitea.m.crossplane.io/v1beta1` | SSH deploy keys |
| [AccessToken](resources/accesstoken.md) | `accesstoken.gitea.m.crossplane.io/v1beta1` | API tokens |
| [RepositorySecret](resources/repositorysecret.md) | `repositorysecret.gitea.m.crossplane.io/v1beta1` | Repository secrets |
| [UserKey](resources/userkey.md) | `userkey.gitea.m.crossplane.io/v1beta1` | User SSH keys |
| [OrganizationMember](resources/organizationmember.md) | `organizationmember.gitea.m.crossplane.io/v1beta1` | Organization membership |
| [OrganizationSecret](resources/organizationsecret.md) | `organizationsecret.gitea.m.crossplane.io/v1beta1` | Organization secrets |

### CI/CD Resources

| Resource | API Group | Description |
|----------|-----------|-------------|
| [Action](resources/action.md) | `action.gitea.m.crossplane.io/v1beta1` | CI/CD workflows |
| [Runner](resources/runner.md) | `runner.gitea.m.crossplane.io/v1beta1` | Self-hosted runners |

### Administrative Resources

| Resource | API Group | Description |
|----------|-----------|-------------|
| [AdminUser](resources/adminuser.md) | `adminuser.gitea.m.crossplane.io/v1beta1` | Admin user accounts |
| [OrganizationSettings](resources/organizationsettings.md) | `organizationsettings.gitea.m.crossplane.io/v1beta1` | Organization settings |
| [GitHook](resources/githook.md) | `githook.gitea.m.crossplane.io/v1beta1` | Server-side Git hooks |
| [Issue](resources/issue.md) | `issue.gitea.m.crossplane.io/v1beta1` | Issue management |
| [PullRequest](resources/pullrequest.md) | `pullrequest.gitea.m.crossplane.io/v1beta1` | Pull request management |
| [Release](resources/release.md) | `release.gitea.m.crossplane.io/v1beta1` | Release management |

## Examples

See the `examples/` directory for complete working examples of each resource type.
