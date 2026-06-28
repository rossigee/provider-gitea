---
name: scaffold-provider
description: Scaffolds a complete new Crossplane Go provider from the mosabastion/crossplane-provider-template, including all boilerplate, wiring, and CI. Use when starting a brand-new provider repo.
tools: Read, Edit, Write, Bash
---

You are scaffolding a new Crossplane provider from the `mosabastion/crossplane-provider-template`. You know every pitfall from real implementations. Work methodically through the phases below.

Before starting, confirm:
- **Service name** (e.g. `rustfs`, `gitea`, `harbor`)
- **Module path** (e.g. `github.com/myorg/crossplane-provider-<service>`)
- **Resources to implement** (e.g. Bucket, User, Policy)
- **API groups** (e.g. `bucket`, `user`, `iam`)
- **Auth mechanism** (AWS SigV4 / SDK / API token / basic auth)
- **Self-hostable with Helm chart?** → determines e2e strategy

## Phase 1 — Module rename

```bash
find . -name "*.go" -o -name "go.mod" | xargs sed -i \
  's|github.com/mosabastion/crossplane-provider-template|<module-path>|g'
# Also rename the sample group
find . -type f | xargs sed -i 's|sample|<group>|g'  # adjust as needed
find apis/sample -name "*.go" | while read f; do
    mv "$f" "${f/sample/<group>}"
done
mv apis/sample apis/<group>
```

Update the module name in `go.mod` and run `go mod tidy`.

## Phase 2 — ProviderConfig

The template's `ProviderConfig` in `apis/v1alpha1/types.go` is already correct for most providers. Adapt:
- `credentials` spec to match your service's auth (endpoint, accessKey, secretKey, or token)
- The credentials Secret keys in `internal/clients/<service>.go`

**CRITICAL markers** — every non-root type in `apis/v1alpha1/types.go` needs:
```go
// +kubebuilder:object:generate=true
type ProviderConfigSpec struct { ... }

// +kubebuilder:object:generate=true
type ProviderCredentials struct { ... }

// +kubebuilder:object:generate=true
type ProviderConfigStatus struct { ... }
```
Without these, `controller-gen` generates `ProviderConfig.DeepCopyInto` that calls `in.Spec.DeepCopyInto(&out.Spec)` — which resolves to the embedded type's method (wrong signature) → CI compile error.

## Phase 3 — Client layer

Create `internal/clients/<service>.go`:
```go
type Config struct {
    Endpoint  string
    AccessKey string
    SecretKey string
    Insecure  bool
}

func ConfigFromSecret(data map[string][]byte) (Config, error) { ... }

func NewClientFromProviderConfig(ctx context.Context, kube client.Client, mg resource.Managed) (*admin.Client, error) {
    pc := mg.GetProviderConfigReference()
    // 1. look up ProviderConfig
    // 2. read credentials Secret
    // 3. return admin.New(cfg.Endpoint, cfg.AccessKey, cfg.SecretKey, cfg.Insecure)
}
```

If the service uses a Go SDK (e.g. madmin-go), wrap it. If it uses AWS SigV4, use `github.com/aws/aws-sdk-go-v2/aws/signer/v4`. If it's a plain REST API, write a thin HTTP client with token auth.

## Phase 4 — For each managed resource

Run the `implement-managed-resource` agent (or follow its steps manually):

1. Create `apis/<group>/v1alpha1/<resource>_types.go` with `+kubebuilder:object:generate=true` on all non-root types
2. Create `apis/<group>/v1alpha1/<resource>_managed.go` (hand-written — angryjet doesn't handle v2 namespaced MR)
3. Register in `apis/<group>/v1alpha1/groupversion_info.go` and `apis/apis.go`
4. Run `bash scripts/generate.sh` — verify `*Spec.DeepCopyInto` and `*Status.DeepCopyInto` are generated
5. Create `internal/clients/admin/<resource>.go` (Get returns nil/nil on 404; Delete is idempotent)
6. Create `internal/controller/<resource>/<resource>.go` (Observe sets Available(), Create stamps external-name, Delete returns (ExternalDelete, error))
7. Wire in `internal/controller/controller.go`
8. Write unit tests

## Phase 5 — E2e

For a self-hostable service with a Helm chart, follow `mosabastion/provider-gitea` exactly:

```
scripts/e2e.sh:
  1. kind cluster + Crossplane (cluster-up.sh)
  2. local OCI registry → wire kind containerd → patch CoreDNS
  3. build xpkg, push to local registry
  4. install Provider package, wait Healthy + ALL CRDs registered
  5. helm install <service> chart --wait
  6. apply ProviderConfig + credentials Secret (test/e2e/uptest-setup.sh)
  7. uptest e2e examples/e2e/*.yaml
```

Never use a hand-crafted mock server as a substitute for a real backend. It proves the client is consistent with itself, not with the actual service.

## Phase 6 — CI wiring

### `.github/workflows/ci.yml`
```yaml
- run: go mod tidy       # before generate — ensures go.sum is in sync
- run: bash scripts/generate.sh
- run: bash scripts/validate.sh
- run: go test -race ./...
- run: CGO_ENABLED=0 go build ./cmd/provider
```

### `.github/workflows/release.yml`
The template's `release.yml` is correct as-is after the fixes applied in June 2026. Key points:
- `docker/setup-buildx-action@v3` is required before the build step
- `docker buildx build --output type=docker,dest=runtime-${ARCH}.tar` — NOT `type=oci`
- `crossplane xpkg build --embed-runtime-image-tarball=runtime-${ARCH}.tar` — NOT `--embed-runtime-image`
- CRD count is verified before pushing

### `scripts/validate.sh`
Exclude `go.sum` from the generate-is-committed check:
```bash
git diff --exit-code -- ':(exclude)go.sum'
```

## Phase 7 — Final checks

```bash
bash scripts/generate.sh
go build ./...
go test -race ./...
bash scripts/validate.sh
```

## Provider-level checklist

- [ ] Module path renamed everywhere
- [ ] `+kubebuilder:object:generate=true` on ALL non-root types in all API packages
- [ ] `zz_generated.deepcopy.go` has `DeepCopyInto` for every `*Spec` and `*Status`
- [ ] `<resource>_managed.go` hand-written for every MR (8 methods each)
- [ ] `ProviderConfigUsage` has 4 explicit interface methods
- [ ] `apis/apis.go` registers all scheme builders
- [ ] No `register.go` alongside any `groupversion_info.go`
- [ ] All `Get*` return `(nil, nil)` on 404
- [ ] All `Delete*` return nil on 404
- [ ] All `Create*` stamp `meta.SetExternalName(cr, id)` from the backend response
- [ ] All `Delete` return `(managed.ExternalDelete{}, err)`
- [ ] All `Observe` call `cr.SetConditions(xpv1.Available())` when up-to-date
- [ ] Rate limiter non-nil in `Setup` (use `ratelimiter.NewGlobal(n)`)
- [ ] `ctrl.SetLogger(zl)` called unconditionally in `main.go`
- [ ] Release workflow uses `type=docker` tarball + `--embed-runtime-image-tarball`
- [ ] E2e installs the real service via Helm — no mock servers
