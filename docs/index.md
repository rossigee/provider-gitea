# Provider Gitea Documentation

A Crossplane v2 provider for managing Gitea resources with complete namespace isolation for multi-tenancy.

## Quick Links

- [Configuration](configuration.md) — Authentication and connection setup
- [Development](development.md) — Building, testing, and contributing

## Resource Documentation

### Core Resources

| Resource | API Group | Description |
|----------|-----------|-------------|
| [Repository](resources/repository.md) | `repository.gitea.m.crossplane.io/v2` | Git repository management |
| [Organization](resources/organization.md) | `organization.gitea.m.crossplane.io/v2` | Organization management |
| [User](resources/user.md) | `user.gitea.m.crossplane.io/v2` | User account management |
| [Webhook](resources/webhook.md) | `webhook.gitea.m.crossplane.io/v2` | Repository and org webhooks |
| [Team](resources/team.md) | `team.gitea.m.crossplane.io/v2` | Team and permissions |
| [Label](resources/label.md) | `label.gitea.m.crossplane.io/v2` | Issue and PR labels |
| [RepositoryCollaborator](resources/repositorycollaborator.md) | `repositorycollaborator.gitea.m.crossplane.io/v2` | Repository collaboration |

### Security Resources

| Resource | API Group | Description |
|----------|-----------|-------------|
| [BranchProtection](resources/branchprotection.md) | `branchprotection.gitea.m.crossplane.io/v2` | Branch protection rules |
| [RepositoryKey](resources/repositorykey.md) | `repositorykey.gitea.m.crossplane.io/v2` | SSH deploy keys |
| [AccessToken](resources/accesstoken.md) | `accesstoken.gitea.m.crossplane.io/v2` | API tokens |
| [RepositorySecret](resources/repositorysecret.md) | `repositorysecret.gitea.m.crossplane.io/v2` | Repository secrets |
| [UserKey](resources/userkey.md) | `userkey.gitea.m.crossplane.io/v2` | User SSH keys |
| [OrganizationMember](resources/organizationmember.md) | `organizationmember.gitea.m.crossplane.io/v2` | Organization membership |
| [OrganizationSecret](resources/organizationsecret.md) | `organizationsecret.gitea.m.crossplane.io/v2` | Organization secrets |

### CI/CD Resources

| Resource | API Group | Description |
|----------|-----------|-------------|
| [Action](resources/action.md) | `action.gitea.m.crossplane.io/v2` | CI/CD workflows |
| [Runner](resources/runner.md) | `runner.gitea.m.crossplane.io/v2` | Self-hosted runners |

### Administrative Resources

| Resource | API Group | Description |
|----------|-----------|-------------|
| [AdminUser](resources/adminuser.md) | `adminuser.gitea.m.crossplane.io/v2` | Admin user accounts |
| [OrganizationSettings](resources/organizationsettings.md) | `organizationsettings.gitea.m.crossplane.io/v2` | Organization settings |
| [GitHook](resources/githook.md) | `githook.gitea.m.crossplane.io/v2` | Server-side Git hooks |
| [Issue](resources/issue.md) | `issue.gitea.m.crossplane.io/v2` | Issue management |
| [PullRequest](resources/pullrequest.md) | `pullrequest.gitea.m.crossplane.io/v2` | Pull request management |
| [Release](resources/release.md) | `release.gitea.m.crossplane.io/v2` | Release management |

## Controller Coverage

Controllers are wired for **Repository, RepositoryKey, RepositorySecret, Organization, User, Webhook, BranchProtection**. The remaining API types have registered CRDs but no reconciler yet, so those resources will not be acted upon: AccessToken, Action, AdminUser, DeployKey, GitHook, Issue, Label, OrganizationMember, OrganizationSecret, OrganizationSettings, PullRequest, Release, RepositoryCollaborator, Runner, Team, UserKey.

## API Coverage Gaps

Gitea API surface not yet modeled at all: packages, container registry artifacts, wikis, project boards, milestones beyond issue fields, notifications, activity feeds, admin-wide user management beyond AdminUser CRUD, and OAuth2 applications.

## Examples

See the `examples/` directory for complete working examples of each resource type.
