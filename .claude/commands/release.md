# Cut a Release

Builds and publishes a versioned Crossplane provider package to GHCR. The release workflow (`release.yml`) handles CI; this command covers the manual steps and the mental model behind each CI stage.

## Trigger

Push a semver tag to `main`:

```bash
git tag v0.x.y
git push origin v0.x.y
```

The workflow fires on `push: tags: 'v*'` and on `workflow_dispatch` for manual runs.

## What the CI does

```
1. go mod vendor
2. bash scripts/generate.sh          ← regenerates deepcopy + CRDs
3. docker buildx build               ← builds per-arch runtime image as Docker tarball
4. crossplane xpkg build             ← embeds runtime tarball into the xpkg
5. verify CRD count                  ← hard gate: prevents "Healthy with 0 CRDs"
6. crossplane xpkg push              ← publishes multi-arch OCI index
```

## Critical CI lessons (baked into release.yml)

### deepcopy regeneration breaks the build if markers are missing
`scripts/generate.sh` runs `controller-gen` before `go build`. If any non-root API type (`*Spec`, `*Status`, `*Parameters`, `*Observation`) is missing `// +kubebuilder:object:generate=true`, controller-gen does not generate `DeepCopyInto` for it. The root type's deepcopy then calls `in.Spec.DeepCopyInto(&out.Spec)` which resolves to the embedded type's method (wrong signature) → compile error.

**Fix:** every non-root type must have the marker. Verify with:
```bash
bash scripts/generate.sh
grep "func (in \*.*Spec) DeepCopyInto" apis/*/v1alpha1/zz_generated.deepcopy.go
```

### `--embed-runtime-image` is not a file path flag
`crossplane xpkg build --embed-runtime-image=<value>` expects a **Docker image name** (looked up in the local Docker daemon), not a binary file path. Passing `bin/linux_amd64/provider` causes:
```
No such image: bin/linux_amd64/provider:latest
```

**Correct pattern:**
```bash
# 1. Build the Go binary (cross-compiled)
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o bin/linux_amd64/provider ./cmd/provider

# 2. Wrap it in a Docker image, output as a Docker-format tarball
#    type=docker produces manifest.json at root (required by crossplane xpkg)
#    type=oci produces an OCI layout tarball — crossplane xpkg CANNOT read it
docker buildx build \
  --platform linux/amd64 \
  --file cluster/images/provider/Dockerfile \
  --output type=docker,dest=runtime-amd64.tar \
  .

# 3. Embed the tarball
crossplane xpkg build \
  --package-root=package \
  --embed-runtime-image-tarball=runtime-amd64.tar \
  --package-file=provider-amd64.xpkg
```

`docker/setup-buildx-action@v3` must precede this step so `docker buildx` is available.

### CRD count gate
The workflow counts CRDs in `package/crds/*.yaml` and verifies the built xpkg carries the same number. This catches the packaging bug where the Provider meta is present but CRDs are absent (lesson #5 in `dev/docs/09-lessons-learned.md`).

## If the release workflow fails

1. Get the failing step logs: `gh run view <run-id> --log-failed`
2. Common failures and fixes:

| Symptom | Cause | Fix |
|---|---|---|
| `cannot use &out.Spec (value of type *XSpec) as *v2.ManagedResourceSpec` | Missing `+kubebuilder:object:generate=true` on `XSpec` | Add the marker, run generate, commit |
| `No such image: bin/linux_amd64/provider:latest` | `--embed-runtime-image` given a file path | Use `--embed-runtime-image-tarball` + docker buildx |
| `file manifest.json not found in tar` | `docker buildx --output type=oci` used | Change to `type=docker` |
| `[amd64] expected N CRDs, found 0` | CRDs not in package/ | Run `bash scripts/generate.sh`, check package/crds/ |

3. Push a fix commit to main via PR, then delete and recreate the tag:

```bash
# After the fix is merged to main
git fetch origin main
git push origin :refs/tags/v0.x.y   # delete remote tag
git tag -d v0.x.y                   # delete local tag
git tag v0.x.y origin/main          # retag at new main tip
git push origin v0.x.y              # push — triggers a new release run
```

## Verify the published package

```bash
crossplane xpkg pull ghcr.io/<org>/crossplane-provider-<name>:v0.x.y
```

Or install directly:
```bash
kubectl crossplane install provider ghcr.io/<org>/crossplane-provider-<name>:v0.x.y
kubectl wait provider/crossplane-provider-<name> --for=condition=Healthy --timeout=120s
```
