#!/usr/bin/env bash
# Build a Crossplane package (.xpkg) for ONE architecture, the correct way:
#   1. compile the provider binary for the arch,
#   2. build the runtime controller image LOCALLY (loaded into docker, NOT
#      pushed to the provider tag — that collision is the bug provider-gitea's
#      release.yml shipped: it pushed a runtime image to the provider tag with
#      no embedded CRDs, so the provider installs Healthy with 0 CRDs),
#   3. `crossplane xpkg build --embed-runtime-image` so package.yaml MERGES the
#      Provider meta + all CRDs and embeds the runtime.
# Echoes the produced .xpkg path. See crossplane-provider-template
# dev/docs/05-packaging.md and lessons-learned #5.
source "$(dirname "$0")/lib.sh"
require_cmd go docker
ARCH="${1:?usage: build-xpkg.sh <amd64|arm64>}"
CRANK="$(ensure_crossplane_cli)"
cd "$ROOT"

log "[$ARCH] build provider binary" >&2
mkdir -p "bin/linux_${ARCH}"
CGO_ENABLED=0 GOOS=linux GOARCH="${ARCH}" go build -trimpath \
  -ldflags "-X ${MODULE}/internal/version.Version=${VERSION}" \
  -o "bin/linux_${ARCH}/provider" ./cmd/provider

RUNTIME="${PROVIDER}-runtime:${VERSION}-${ARCH}"
log "[$ARCH] build runtime image locally (${RUNTIME})" >&2
docker build --platform "linux/${ARCH}" \
  --build-arg TARGETOS=linux --build-arg TARGETARCH="${ARCH}" \
  -t "${RUNTIME}" -f "cluster/images/${PROVIDER}/Dockerfile" . >&2

XPKG="${PROVIDER}-${ARCH}.xpkg"
log "[$ARCH] crossplane xpkg build -> ${XPKG}" >&2
"$CRANK" xpkg build \
  --package-root=package \
  --embed-runtime-image="${RUNTIME}" \
  --package-file="${XPKG}" >&2

echo "${XPKG}"
