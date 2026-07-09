# provider-gitea

Crossplane provider for Gitea (`github.com/rossigee/provider-gitea`).

Managed resources live under `gitea.m.crossplane.io/v1beta1`. ProviderConfig lives under `gitea.crossplane.io/v1beta1`.

Resources: Organization, User, Repository, Webhook, BranchProtection, AccessToken, Label, Variable, Team, GitHook, OrganizationSettings, OrganizationSecret, RepositorySecret, RepositoryKey, RepositoryCollaborator, TeamMembership, TeamRepository.

## Commands

```bash
bash scripts/generate.sh   # regenerate deepcopy + CRDs (run after editing types)
go build ./...
go test -race ./...
bash scripts/validate.sh   # lint + check generated files are committed
```

## Key rules

- Every non-root API type (`*Spec`, `*Status`, `*Parameters`, `*Observation`) needs `// +kubebuilder:object:generate=true` — without it the CI build breaks after regeneration.
- `Get*` returns `(nil, nil)` on 404. `Delete*` returns nil on 404.
- `Observe` calls `cr.SetConditions(xpv1.Available())` when up-to-date.
- `Create` stamps `meta.SetExternalName(cr, id)` from the backend response.
- `Delete` returns `(managed.ExternalDelete{}, err)` — crossplane-runtime v2 signature.
- Do NOT add `// +versionName=...` to `groupversion_info.go` files — it causes the CRD version to diverge from the runtime `Version` constant.

## Gitea API quirks

- `GET /admin/users/{u}` returns 405 — use `GET /users/{u}` instead.
- SSH key CRUD uses admin endpoints (`/admin/users/{u}/keys[/{id}]`); `/users/{u}/keys` is read-only.
- No single-secret GET (405) for org secrets, repo secrets, or repo variables — list and match by name.
- GitHooks cannot be created (POST returns 405) — use PATCH to edit the existing hook script.
- `PATCH /admin/users/{u}` requires `login_name` and `source_id` fields even when not changing them.
- AccessToken list+match by id (no single-GET endpoint).

## E2e

Requires: `kind`, `kubectl`, `helm`, `chainsaw`.

```bash
bash scripts/e2e.sh        # full run: kind cluster + real Gitea (Helm) + uptest
KEEP=1 bash scripts/e2e.sh # keep cluster for inspection
```

E2e drives a real Gitea instance (official `gitea-charts/gitea`, sqlite, in-memory cache, no persistence). The setup script mints an admin API token and writes the ProviderConfig. E2e manifests are in `examples/e2e/`.

## Skills / agents

See `.claude/commands/` and `.claude/agents/` for slash commands and agent definitions.
