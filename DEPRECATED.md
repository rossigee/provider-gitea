# DEPRECATED: provider-gitea v1

⚠️ **This provider is deprecated and no longer under active development.**

## Migration to v2

This traditional Crossplane framework-based provider is being replaced with a modern kubebuilder-based implementation for better maintainability and developer experience.

### Why Deprecated?

- **High maintenance burden**: Traditional Crossplane framework requires extensive boilerplate
- **Complex contributor onboarding**: Requires deep Crossplane-specific knowledge
- **Limited tooling support**: Less IDE and debugging support compared to kubebuilder

### Migration Path

**For New Deployments:**
- Use the upcoming **provider-gitea-v2** built with kubebuilder
- Modern Kubernetes controller patterns
- Better developer experience and community contribution
- Faster development velocity for new features

### Current Status

- **Production Ready**: This v1 provider has 22 resource types and enterprise features
- **Maintenance Mode**: Critical bug fixes only, no new features
- **Sunset Plan**: Will be archived after v2 reaches feature parity

### v1 Features (Reference)

This provider includes 22 managed resource types:
- Repository, Organization, User, Webhook, DeployKey, Team, Label, RepositoryCollaborator
- BranchProtection, RepositoryKey, AccessToken, RepositorySecret, UserKey
- OrganizationMember, OrganizationSecret, Action, Runner
- AdminUser, OrganizationSettings, GitHook

### Timeline

- **Current**: v1 in maintenance mode
- **Q4 2025**: v2 development and testing
- **Q1 2026**: v2 production release
- **Q2 2026**: v1 deprecation and archive

For questions about migration or v2 development, please file an issue in the main crossplane-providers repository.