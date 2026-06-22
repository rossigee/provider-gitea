#!/usr/bin/env bash
# Shared helpers for the scripts/ task runners. Source, don't execute.
#
# This harness is a self-contained alternative to the crossplane build submodule
# (build/makelib/*.mk): it builds a CORRECT xpkg (embedded runtime + verified
# CRDs, lesson #5) and drives a real apply->Ready->delete e2e on a throwaway
# kind cluster against a mock Gitea — so a green CI run proves the CRs actually
# work, by design. Ported from mosabastion/crossplane-provider-template.
set -euo pipefail

PROVIDER="${PROVIDER:-provider-gitea}"
MODULE="${MODULE:-github.com/rossigee/provider-gitea}"
REGISTRY="${REGISTRY:-ghcr.io/rossigee}"
IMAGE="${IMAGE:-${REGISTRY}/${PROVIDER}}"
VERSION="${VERSION:-$(git describe --tags --dirty --always 2>/dev/null || echo v0.0.0-dev)}"
PLATFORMS="${PLATFORMS:-amd64 arm64}"
CROSSPLANE_CLI_VERSION="${CROSSPLANE_CLI_VERSION:-v2.3.2}"
KIND_CLUSTER="${KIND_CLUSTER:-provider-gitea-dev}"
# API groups this provider's CRDs register under (used by the post-install
# CRD-presence assertion). Both the legacy cluster-scoped and the v2 namespaced
# variants share the "gitea.crossplane.io" substring.
CRD_GROUP="${CRD_GROUP:-gitea.crossplane.io}"

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

log()  { printf '\033[36m==>\033[0m %s\n' "$*"; }
ok()   { printf '\033[32m✓\033[0m %s\n' "$*"; }
warn() { printf '\033[33m! \033[0m%s\n' "$*" >&2; }
die()  { printf '\033[31m✗ %s\033[0m\n' "$*" >&2; exit 1; }

require_cmd() {
  for c in "$@"; do
    command -v "$c" >/dev/null 2>&1 || die "required command not found: $c"
  done
}

# Install the Crossplane CLI ("crank") into ./bin if not on PATH. Echoes the path.
ensure_crossplane_cli() {
  if command -v crossplane >/dev/null 2>&1; then echo "crossplane"; return; fi
  local dst="${ROOT}/bin/crossplane"
  mkdir -p "${ROOT}/bin"
  log "fetching crossplane CLI ${CROSSPLANE_CLI_VERSION}" >&2
  curl -fsSL "https://releases.crossplane.io/stable/${CROSSPLANE_CLI_VERSION}/bin/linux_amd64/crank" -o "$dst"
  chmod +x "$dst"
  echo "$dst"
}
