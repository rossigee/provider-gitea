# Provider Gitea

> [!WARNING]
> **The next release contains breaking API changes.** All existing CRs must be
> deleted and re-applied — CRD names and API versions change and in-place upgrade
> is not possible. See [BREAKING_CHANGES.md](BREAKING_CHANGES.md) for the full
> migration guide.
>
> Key changes: unified group (`gitea.m.crossplane.io/v1beta1` for all MRs,
> replacing the per-resource `<kind>.gitea.m.crossplane.io/v2` groups); ProviderConfig
> group (`gitea.crossplane.io`) is unchanged.

A Crossplane provider for declarative Gitea management: **15 namespaced
resource kinds**, each with a working reconciler proven end-to-end against a real
Gitea server in CI.

## Overview

This provider manages Gitea resources (repositories, organizations, users,
teams, labels, webhooks, secrets, branch protection, and more) as Kubernetes
custom resources. Every kind has a create/observe/update/delete controller, a
unit test, and — for all but the immutable kinds — a full
apply→Ready→update→import→delete run against a **real Gitea** (the official Helm
chart) on a kind cluster, wired into CI.

The controllers bake in the correctness lessons distilled in
[`crossplane-provider-template`](https://github.com/mosabastion/crossplane-provider-template)
`dev/docs/09-lessons-learned.md` — `Available()` set in `Observe`, not-found
classified off the typed HTTP status, real drift detection, external-name as the
authoritative identity for Observe/Update/Delete, a non-nil rate limiter, the v2
managed-methodset, and a package that can't ship Healthy-but-CRD-less.

## Resource catalog

All kinds use the namespaced group `gitea.m.crossplane.io/v1beta1` and must
carry `metadata.namespace`.

| Resource | Purpose | Example |
|----------|---------|---------|
| `Repository` | Git repository lifecycle | [repository.yaml](examples/e2e/repository.yaml) |
| `Organization` | Organization lifecycle | [organization.yaml](examples/e2e/organization.yaml) |
| `User` | User account lifecycle (admin API) | [user.yaml](examples/e2e/user.yaml) |
| `Team` | Org team + access control | [team.yaml](examples/e2e/team.yaml) |
| `Label` | Issue/PR labels (repo- or org-scoped) | [label.yaml](examples/e2e/label.yaml) |
| `Webhook` | Repository/org webhooks | [webhook.yaml](examples/e2e/webhook.yaml) |
| `GitHook` | Server-side Git hooks | [githook.yaml](examples/e2e/githook.yaml) |
| `BranchProtection` | Branch protection rules | [branchprotection.yaml](examples/e2e/branchprotection.yaml) |
| `RepositoryKey` | Repository SSH (deploy) keys | [repositorykey.yaml](examples/e2e/repositorykey.yaml) |
| `RepositoryCollaborator` | Repository access grants | [repositorycollaborator.yaml](examples/e2e/repositorycollaborator.yaml) |
| `OrganizationSettings` | Organization policy | [organizationsettings.yaml](examples/e2e/organizationsettings.yaml) |
| `Variable` | Actions variable (non-secret), repo- or org-scoped | [variable.yaml](examples/e2e/variable.yaml) |
| `RepositorySecret` 🔑 | Actions secret on a repo | [repositorysecret.yaml](examples/e2e/repositorysecret.yaml) |
| `OrganizationSecret` 🔑 | Actions secret on an org | [organizationsecret.yaml](examples/e2e/organizationsecret.yaml) |
| `AccessToken` 🔑 | Personal access token (PAT) | [accesstoken.yaml](examples/e2e/accesstoken.yaml) |

🔑 = takes a secret value via a Secret reference — see [Working with secret-bearing
resources](#working-with-secret-bearing-resources).

### What is intentionally NOT a resource

Some Gitea concepts don't fit a declarative managed-resource model and were
deliberately excluded; modelling them would produce resources that can't
reconcile:

| Concept | Why it's not a CRD |
|---------|--------------------|
| Action / workflow | A workflow is a file committed to `.gitea/workflows/` — git content, not an API object (Gitea has no create-workflow endpoint). |
| Runner | Registration needs a one-time token and a live `act_runner` agent — runtime registration, not desired state. |
| PullRequest | A transient event over two branches with divergent commits, not a managed resource. |
| Issue / Release | Content/tickets/tagged artifacts — not infrastructure config. |
| OrganizationMember | Gitea has no add-member endpoint; membership is a side-effect of team membership (use `Team`). |

Two kinds were merged/deduplicated rather than dropped:
- **AdminUser** merged into **`User`** (both drove `/admin/users`); `User` carries
  the union of fields (`maxRepoCreation`, `admin`, …).
- **DeployKey** and **UserKey** were removed in favour of **`RepositoryKey`**
  (DeployKey hit the identical `/repos/{owner}/{repo}/keys` endpoint).

## Lifecycle coverage

A green CI run is the "works by design" guarantee. `scripts/e2e.sh` installs a
real Gitea via the official Helm chart on a kind cluster and drives every example
through uptest:

| Stage | Coverage |
|-------|----------|
| Create / Observe / Import / Delete | **all 15 kinds**, against real Gitea |
| Update | exercised live for the mutable kinds (`Repository`, `Organization`, `Label`, `Team`, `User`) via `uptest.upbound.io/update-parameter`; unit-tested for the rest |

`RepositorySecret`, `OrganizationSecret`, `AccessToken`, and `RepositoryKey` have
no meaningful in-place update (the value is write-only / the key is immutable), so
their controllers treat the resource as up-to-date once it exists and skip the
live update step.

Crossplane **management policies** (`spec.managementPolicies`) are honoured by all
15 controllers — ObserveOnly, no-delete, pause, and partial-action modes — when
the provider is run with `--enable-management-policies`
(`feature.EnableBetaManagementPolicies`). The flag defaults off.

## Working with secret-bearing resources

**Secret values are never set inline.** Every field that carries a credential is
a Kubernetes Secret reference (`*SecretRef`), following the same convention as the
rest of the platform's providers. The reference is a selector with `namespace`,
`name`, and `key`:

| Resource | Field | Holds |
|----------|-------|-------|
| `User` | `spec.forProvider.passwordSecretRef` | the user's password (required on create) |
| `RepositorySecret` | `spec.forProvider.valueSecretRef` | the Actions secret value |
| `OrganizationSecret` | `spec.forProvider.valueSecretRef` | the Actions secret value |
| `AccessToken` | `spec.forProvider.passwordSecretRef` | the owning user's password (see below) |

Example — a `User` whose password comes from a Secret:

```yaml
apiVersion: v1
kind: Secret
metadata: {name: alice-password, namespace: default}
stringData: {password: "S3cure-Pass-123"}
---
apiVersion: gitea.m.crossplane.io/v1beta1
kind: User
metadata: {name: alice, namespace: default}
spec:
  forProvider:
    username: alice
    email: alice@example.com
    passwordSecretRef: {namespace: default, name: alice-password, key: password}
  providerConfigRef: {name: default, kind: ProviderConfig}
```

### AccessToken authenticates as the user (not the provider)

`AccessToken` is special. Gitea's token API (`/users/{user}/tokens`) **rejects the
ProviderConfig token and requires HTTP basic auth as the owning user**. So
`AccessToken` does not use the ProviderConfig's credentials — it authenticates as
`spec.forProvider.username` using the password in `passwordSecretRef`. Provide a
Secret with that user's password; the provider basic-auths as them to mint, read,
and revoke the token. The minted token value is written to the resource's
connection secret (Gitea returns it exactly once). This basic-auth path is reusable
for any future user-scoped resource.

## Quick start

1. Install the provider:
   ```bash
   kubectl crossplane install provider ghcr.io/mosabastion/provider-gitea:<tag>
   ```
2. Confirm the CRDs registered:
   ```bash
   kubectl get crds | grep gitea.m.crossplane.io   # expect 15
   ```
3. Create a ProviderConfig + apply a resource (see `examples/e2e/`):
   ```bash
   kubectl apply -f examples/e2e/repository.yaml
   kubectl get repository.gitea.m.crossplane.io -n <ns>
   ```

## Testing

```bash
make test          # unit tests (offline, table-driven per controller)
make validate      # the full static CI gate (build + test + lint + check-diff) — RUN BEFORE PUSHING
make e2e           # self-contained kind + REAL Gitea e2e (apply->Ready->update->import->delete)
make xpkg-verify   # assert the built package carries the Provider meta + all CRDs
```

- Every controller has a unit test asserting the correctness invariants
  (Available on the exists path, typed not-found, drift, external-name identity,
  idempotent delete).
- `make e2e` (and CI `e2e.yml`) install a real Gitea (latest, official Helm chart)
  on a throwaway kind cluster and drive every example through its lifecycle — no
  mock backend. See [docs/TESTING.md](docs/TESTING.md) for the harness details.
- `make validate` reproduces the exact CI gate locally; run it before every push.

## Development setup

After cloning, install the git hooks (prevents large-file / binary-artifact
commits):

```bash
./scripts/install-hooks.sh
```

## Documentation

- [Testing harness](docs/TESTING.md)
- [Configuration Guide](docs/CONFIGURATION.md)
- [Development Guide](docs/DEVELOPMENT.md)
- [Resource Reference](docs/RESOURCES.md)

## Registry

`ghcr.io/mosabastion/provider-gitea`
