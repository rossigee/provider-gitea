# Breaking Changes

## Next release — API group consolidation + version alignment

> All existing CRs must be deleted and re-applied. CRD names and API versions
> change; in-place upgrade is not possible.

### 1. Unified API group for all managed resources

Every managed resource moves from a per-resource group to a single shared group,
and the version changes from `v2` to `v1beta1`.

| Before | After |
|--------|-------|
| `repository.gitea.m.crossplane.io/v2` | `gitea.m.crossplane.io/v1beta1` |
| `organization.gitea.m.crossplane.io/v2` | `gitea.m.crossplane.io/v1beta1` |
| `user.gitea.m.crossplane.io/v2` | `gitea.m.crossplane.io/v1beta1` |
| `team.gitea.m.crossplane.io/v2` | `gitea.m.crossplane.io/v1beta1` |
| `label.gitea.m.crossplane.io/v2` | `gitea.m.crossplane.io/v1beta1` |
| `webhook.gitea.m.crossplane.io/v2` | `gitea.m.crossplane.io/v1beta1` |
| `githook.gitea.m.crossplane.io/v2` | `gitea.m.crossplane.io/v1beta1` |
| `branchprotection.gitea.m.crossplane.io/v2` | `gitea.m.crossplane.io/v1beta1` |
| `repositorykey.gitea.m.crossplane.io/v2` | `gitea.m.crossplane.io/v1beta1` |
| `repositorycollaborator.gitea.m.crossplane.io/v2` | `gitea.m.crossplane.io/v1beta1` |
| `organizationsettings.gitea.m.crossplane.io/v2` | `gitea.m.crossplane.io/v1beta1` |
| `repositorysecret.gitea.m.crossplane.io/v2` | `gitea.m.crossplane.io/v1beta1` |
| `organizationsecret.gitea.m.crossplane.io/v2` | `gitea.m.crossplane.io/v1beta1` |
| `accesstoken.gitea.m.crossplane.io/v2` | `gitea.m.crossplane.io/v1beta1` |
| `variable.gitea.m.crossplane.io/v2` | `gitea.m.crossplane.io/v1beta1` |

**ProviderConfig is unchanged** — `gitea.crossplane.io/v1beta1` was already
correct (ProviderConfig is a provider identity resource, not a managed resource,
and must not carry `.m.`).

**Why — unified group:** the per-resource group (inherited from the upjet pattern)
is redundant — the kind already carries the resource name. A unified group is
simpler to RBAC, easier to `kubectl get`, and eliminates the
`repository.repository.gitea.m...` noise. With a unified group the compound kind
names (`RepositoryKey`, `OrganizationSecret`, `BranchProtection`) stand
unambiguously on their own.

**Why — `v1beta1` instead of `v2`:** Kubernetes API version conventions treat bare
`vN` (v1, v2, …) as **stable GA**. Using `v2` to signal "second-generation
rewrite" is a misuse — it tells the Crossplane package manager, conversion
webhooks, and consumers that this API is production-stable and fully supported.
`v1beta1` is the correct designation for a provider API that is functional but
still evolving.

**Migration:** update every manifest's `apiVersion` from
`<resource>.gitea.m.crossplane.io/v2` to `gitea.m.crossplane.io/v1beta1`. Kind
names are unchanged.
