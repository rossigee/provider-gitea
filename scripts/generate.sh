#!/usr/bin/env bash
# Generate the code/manifests Crossplane providers don't hand-write (deepcopy,
# CRDs, managed methodsets) by running the canonical `go generate` directives in
# apis/generate.go — the SAME ones the crossplane build submodule's `make
# reviewable` runs in CI. Using `go generate` (rather than a bespoke
# controller-gen invocation) keeps this script's output byte-identical to CI, so
# the check-diff job stays green. The directives use crd:maxDescLen=0; do not
# substitute different flags here or you will reintroduce package/crds drift.
source "$(dirname "$0")/lib.sh"
require_cmd go
cd "$ROOT"

# Wipe package/crds first so a renamed/removed kind can't leave a stale CRD
# behind (go generate overwrites, but does not delete).
log "wiping package/crds"
rm -f package/crds/*.yaml

log "go generate ./... (deepcopy + CRDs + managed methodsets)"
go generate ./...

ok "generation complete — review git diff under apis/ and package/crds/"
