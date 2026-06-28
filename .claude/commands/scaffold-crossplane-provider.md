# Scaffold a Crossplane Go Provider

This command guides you through building a complete Crossplane Go provider from scratch, using the `mosabastion/crossplane-provider-template` as the canonical reference. See `mosabastion/provider-gitea` for the gold-standard e2e pattern this command follows.

## Inputs needed from user

Before starting, confirm:
- **Target service**: What external system are you managing? (e.g. RustFS, Vault, some SaaS API)
- **Target repo**: Where does the new provider live? (e.g. `myorg/crossplane-provider-<service>`)
- **Resources to manage**: List the CRDs you need (e.g. User, Bucket, Policy, PolicyAttachment)
- **API docs / source**: URL or path to the service's API reference. If there's no SDK, reverse-engineer from source.
- **Is the service self-hostable via a Helm chart?** This determines your e2e strategy (see Phase 6).

## Testing philosophy

**Unit tests** use `httptest.NewServer` to mock the HTTP transport and test your client code in isolation.

**E2e tests must use the real service binary**, deployed via its official Helm chart into the kind cluster. A hand-written mock HTTP server is integration testing — it only proves your client is consistent with itself. Real e2e proves behavior against the actual service: real auth, real error payloads, real 404s, real API evolution.

See `mosabastion/provider-gitea/scripts/e2e.sh` as the reference implementation of this pattern.

## Phase 1 — API research

1. Read the service's API source or docs. Identify:
   - Base URL pattern and authentication scheme
   - CRUD endpoints for each resource you'll manage
   - Immutable fields (will be `external-name` or have CEL validation `self == oldSelf`)
   - Observation-only fields (go in `AtProvider`)
   - Any attach/detach semantics (separate managed resource, not just a field)

2. If the service uses AWS SigV4 (MinIO-compatible), use `github.com/aws/aws-sdk-go-v2/aws/signer/v4`. If it has an official Go SDK, prefer that. Otherwise write a thin HTTP client in `internal/clients/`.

## Phase 2 — Repo bootstrap

Fork or copy `mosabastion/crossplane-provider-template`. Adjust the module name:

```bash
git clone git@github.com:myorg/crossplane-provider-<service>.git
cd crossplane-provider-<service>
sed -i 's|github.com/mosabastion/crossplane-provider-template|github.com/myorg/crossplane-provider-<service>|g' go.mod
```

Key files from template to keep/adapt:
- `Makefile` — add a `deps` target: `go mod download && go mod tidy`
- `.github/workflows/ci.yml` — add `go mod tidy` as first step before `make generate`
- `.github/workflows/e2e.yml` — runs `scripts/e2e.sh`; no extra steps needed (helm install is inside the script)
- `.github/workflows/release.yml` — keep as-is
- `scripts/validate.sh` — exclude `go.sum` from generate-is-clean: `git diff --exit-code -- ':(exclude)go.sum'`

## Phase 3 — API types

For each managed resource, create `apis/<group>/v1alpha1/<resource>_types.go`:

```go
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,<service>}
type MyResource struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`
    Spec   MyResourceSpec   `json:"spec"`
    Status MyResourceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:generate=true
type MyResourceParameters struct { ... }

// +kubebuilder:object:generate=true
type MyResourceObservation struct { ... }

// +kubebuilder:object:generate=true
type MyResourceSpec struct {
    xpv1.ManagedResourceSpec `json:",inline"`
    ForProvider MyResourceParameters `json:"forProvider"`
}

// +kubebuilder:object:generate=true
type MyResourceStatus struct {
    xpv1.ConditionedStatus `json:",inline"`
    AtProvider MyResourceObservation `json:"atProvider,omitempty"`
}
```

**CRITICAL — `+kubebuilder:object:generate=true` on every non-root type:** `controller-gen` only generates `DeepCopyInto` for types that have this marker (or `+kubebuilder:object:root=true`). Without it, the root type's deepcopy calls `in.Spec.DeepCopyInto(&out.Spec)` which resolves to the embedded type's method — wrong argument type — and the CI build fails (CI runs `scripts/generate.sh` before compiling). Every `*Parameters`, `*Observation`, `*Spec`, `*Status` and any nested struct must have the marker. Also applies to `ProviderConfigSpec`, `ProviderCredentials`, `ProviderConfigStatus` in `apis/v1alpha1/types.go`.

After adding types, always run `bash scripts/generate.sh` and verify `zz_generated.deepcopy.go` contains `func (in *MyResourceSpec) DeepCopyInto(...)` — if missing, the marker is absent or misplaced.

Also create `apis/<group>/v1alpha1/groupversion_info.go` with `SchemeGroupVersion`, `SchemeBuilder`, `AddToScheme`, and `<Resource>GroupVersionKind` constants.

**Do NOT create a `register.go` alongside `groupversion_info.go`** — they would redeclare the same `Group`, `Version`, `SchemeGroupVersion`, and `SchemeBuilder` vars, causing a compile error. Everything belongs in `groupversion_info.go`.

For `apis/v1alpha1/` (provider config group): include both `ProviderConfig` and `ProviderConfigUsage` types.

`ProviderConfigUsage` must implement `resource.TypedProviderConfigUsage`. Since the embedded `xpv1.TypedProviderConfigUsage` struct provides fields but no methods, add these explicitly in `providerconfig_managed.go`:

```go
func (p *ProviderConfigUsage) GetProviderConfigReference() xpv1.ProviderConfigReference {
    return p.TypedProviderConfigUsage.ProviderConfigReference
}
func (p *ProviderConfigUsage) SetProviderConfigReference(r xpv1.ProviderConfigReference) {
    p.TypedProviderConfigUsage.ProviderConfigReference = r
}
func (p *ProviderConfigUsage) GetResourceReference() xpv1.TypedReference {
    return p.TypedProviderConfigUsage.ResourceReference
}
func (p *ProviderConfigUsage) SetResourceReference(r xpv1.TypedReference) {
    p.TypedProviderConfigUsage.ResourceReference = r
}
```

If `controller-gen`/`angryjet` aren't available at commit time, hand-write `zz_generated.deepcopy.go` and `zz_generated.managed.go` following the template's widget example.

## Phase 4 — Client layer

```
internal/clients/
  <service>.go           # Config, ConfigFromSecret, NewClientFromProviderConfig
  admin/
    client.go            # Authenticated HTTP client (SigV4 / SDK / token)
    <resource>.go        # CRUD per resource; Get returns (nil, nil) on 404
    <resource>_test.go   # httptest.NewServer — unit tests only
```

Key patterns:
- `Get<Resource>` returns `(nil, nil)` on 404 — never error on not-found
- `Delete<Resource>` is idempotent — nil on 404
- `NewClientFromProviderConfig(ctx, kube, mg)` looks up ProviderConfig, reads the credentials Secret, returns a ready client

## Phase 5 — Controllers

For each resource, create `internal/controller/<resource>/<resource>.go`:

```go
type connector struct {
    kube  client.Client
    usage resource.ModernTracker  // resource.NewProviderConfigUsageTracker(kube, &apisv1alpha1.ProviderConfigUsage{})
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
    cr, ok := mg.(*v1alpha1.MyResource)
    if !ok {
        return nil, errors.New(errNotMyResource)
    }
    if err := c.usage.Track(ctx, cr); err != nil {  // pass cr, not mg — cr implements ModernManaged
        return nil, errors.Wrap(err, errTrackPCUsage)
    }
    client, err := clients.NewClientFromProviderConfig(ctx, c.kube, cr)
    return &external{client: client}, err
}

type external struct{ ... }
func (e *external) Disconnect(_ context.Context) error { return nil }
func (e *external) Observe(ctx, mg) (managed.ExternalObservation, error) { ... }
func (e *external) Create(ctx, mg) (managed.ExternalCreation, error)     { ... }
func (e *external) Update(ctx, mg) (managed.ExternalUpdate, error)       { ... }
func (e *external) Delete(ctx, mg) (managed.ExternalDelete, error)       { ... }
```

Observe: return `ResourceExists: false` if Get returns nil; set `AtProvider` from live state; call `cr.SetConditions(xpv1.Available())`; compute `ResourceUpToDate`.

Create: call `meta.SetExternalName(cr, <id>)` and return `managed.ExternalCreation{}`. There is no `ExternalNameAssigned` field — it was removed in crossplane-runtime v2.

Delete: return `(managed.ExternalDelete{}, err)` — the signature changed from `error` to `(ExternalDelete, error)` in crossplane-runtime v2.

Wire all controllers in `internal/controller/controller.go` and register in `cmd/provider/main.go`.

## Phase 6 — E2e test infrastructure

### Determine your e2e strategy

**Self-hostable service with a Helm chart** → install the real service in the kind cluster. This is the required approach. Follow the `mosabastion/provider-gitea` pattern exactly.

**Managed / SaaS service** → choose explicitly:
1. Gate the e2e workflow on a secret: `if: secrets.SERVICE_API_KEY != ''` — skip silently in forks/PRs.
2. Provision a throwaway account in CI if the service offers it.
3. No e2e — document this explicitly in CONTRIBUTING.md.

Never write a hand-crafted mock HTTP server as a substitute for e2e. It's integration testing.

### `scripts/e2e.sh` structure (follow provider-gitea exactly)

```
1. Download uptest binary (cached by version)
2. cluster-up.sh  — kind + Crossplane (idempotent)
3. Local OCI registry
     - Start registry:2 container
     - Wire containerd on each kind node (hosts.toml + /etc/hosts)
     - Patch CoreDNS so in-Pod image pulls resolve the registry hostname
4. Build xpkg, verify-xpkg.sh, push to local registry
5. Install Provider package, wait Healthy
6. Assert ALL package/crds/*.yaml CRDs registered
     (Healthy with 0 CRDs = mis-packaged; silently invalid CRD = never works)
7. helm repo add <chart-repo>
   helm upgrade --install <release> <chart> -n <ns>
     --set <minimal values: no persistence, credentials, single replica>
     --wait --timeout=10m
   kubectl rollout status ...
8. kubectl create secret <credentials pointing at in-cluster service>
9. uptest e2e <examples/e2e/*.yaml> --setup-script=test/e2e/uptest-setup.sh
```

Expose the service image/version as an env var override so CI can pin it:
```bash
SERVICE_IMAGE="${SERVICE_IMAGE:-official/image:latest}" make e2e
```

### `test/e2e/uptest-setup.sh`

```bash
#!/usr/bin/env bash
set -euo pipefail
# 1. Wait for the helm-deployed service rollout
kubectl rollout status deploy/<release> -n <ns> --timeout=120s
# 2. Verify the API port responds from within the cluster
kubectl run healthcheck --image=curlimages/curl:latest \
  --restart=Never --rm --attach --timeout=30s \
  -- curl -sf http://<release>.<ns>.svc.cluster.local:<port>/health
# 3. Apply ProviderConfig (references the credentials secret injected by e2e.sh)
kubectl apply -f - <<EOF
apiVersion: <provider>.crossplane.io/v1alpha1
kind: ProviderConfig
metadata:
  name: default
spec:
  credentials:
    source: Secret
    secretRef:
      namespace: crossplane-system
      name: <service>-credentials
      key: credentials
EOF
echo "backend ready — uptest may proceed."
```

### `examples/e2e/`

One YAML per managed resource with minimal valid `forProvider`. Must be idempotent — uptest creates, asserts `Ready=True`, then deletes and asserts gone. Add `uptest.upbound.io/update-parameter` annotations on mutable fields to exercise the Update path.

## Phase 7 — CI wiring

`.github/workflows/ci.yml`:
1. `go mod tidy`
2. `make generate`
3. `make validate`
4. `make test-unit`
5. `make build`

`.github/workflows/e2e.yml`: runs `scripts/e2e.sh`. For SaaS services, gate on `if: secrets.SERVICE_KEY != ''`.

## Checklist

- [ ] All `Get*` methods return `(nil, nil)` on 404
- [ ] All `Delete*` methods are idempotent (nil on 404)
- [ ] `external-name` is set in `Create`; return `managed.ExternalCreation{}` (no `ExternalNameAssigned` field in v2)
- [ ] `Delete` returns `(managed.ExternalDelete, error)` — not just `error`
- [ ] `connector.usage` is `resource.ModernTracker`; `Track` is called with the typed `cr`, not `mg`
- [ ] `Observe` sets `AtProvider` from live state and computes `ResourceUpToDate`
- [ ] Unit tests use `httptest.NewServer` — client code only, not e2e
- [ ] **E2e installs the real service via its official Helm chart** — no mock servers
- [ ] `scripts/e2e.sh` asserts ALL CRDs registered (not just `Healthy=True`)
- [ ] `go mod tidy` runs before `make generate` in CI
- [ ] `scripts/validate.sh` excludes `go.sum` from diff check
- [ ] No `register.go` alongside `groupversion_info.go` — duplicate declarations won't compile
- [ ] `ProviderConfigUsage` has the 4 explicit methods (`GetProviderConfigReference`, `SetProviderConfigReference`, `GetResourceReference`, `SetResourceReference`)
- [ ] `ProviderConfigUsage` registered in `apis/v1alpha1/`
- [ ] All API groups have `groupversion_info.go` with GVK constants (no sibling `register.go`)
- [ ] `apis/apis.go` registers all scheme builders
- [ ] `cmd/provider/main.go` calls the controller Setup function
- [ ] `cluster/images/<provider>/Dockerfile` exists; release CI uses `docker buildx build --output type=docker,dest=runtime.tar` + `crossplane xpkg build --embed-runtime-image-tarball=runtime.tar` (NOT `--embed-runtime-image` which expects a Docker image name, not a file path; NOT `type=oci` which lacks `manifest.json`)
- [ ] `+kubebuilder:object:generate=true` on ALL non-root types in every API package (Parameters, Observation, Spec, Status, plus ProviderConfigSpec, ProviderCredentials, ProviderConfigStatus)
- [ ] After `scripts/generate.sh`: `zz_generated.deepcopy.go` has explicit `DeepCopyInto` for every `*Spec` and `*Status`
