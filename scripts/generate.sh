#!/usr/bin/env bash
# Generate the code/manifests Crossplane providers don't hand-write:
#   - deepcopy methods (controller-gen object)
#   - CRDs -> package/crds (controller-gen crd)
#   - managed-resource methodsets (angryjet)
# The tools are pinned in tools.go, fetched via `go run`. This regenerates a
# CLEAN package/crds containing ONLY the CRDs the Go types actually define (the
# v2 *.gitea.m.crossplane.io namespaced groups + the v1beta1 ProviderConfig) —
# the stale legacy *.gitea.crossplane.io and the broken empty-group `_*.yaml`
# artefacts are deliberately wiped first.
source "$(dirname "$0")/lib.sh"
require_cmd go
cd "$ROOT"

HDR="hack/boilerplate.go.txt"
[ -f "$HDR" ] || die "missing license header $HDR"

log "deepcopy (controller-gen)"
go run -mod=mod sigs.k8s.io/controller-tools/cmd/controller-gen \
  object:headerFile="$HDR" paths=./apis/...

log "wiping package/crds and regenerating from ./apis/..."
rm -f package/crds/*.yaml
go run -mod=mod sigs.k8s.io/controller-tools/cmd/controller-gen \
  crd:allowDangerousTypes=true,crdVersions=v1 paths=./apis/... \
  output:crd:artifacts:config=./package/crds

log "managed methodsets (angryjet)"
go run -mod=mod github.com/crossplane/crossplane-tools/cmd/angryjet \
  generate-methodsets --header-file="$HDR" ./apis/... 2>/dev/null || \
  warn "angryjet methodset generation skipped/failed (v2 namespaced methods are hand-written)"

ok "generation complete — review git diff under apis/ and package/crds/"
