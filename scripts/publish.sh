#!/usr/bin/env bash
# Build + verify + push a SINGLE multi-platform package index covering every
# arch in $PLATFORMS. This is the lesson-#5-correct release path: ONE artifact
# to the provider tag, runtime embedded, CRDs verified present. Requires GHCR
# login (docker login ghcr.io) and a real $VERSION (not the -dev default).
source "$(dirname "$0")/lib.sh"
require_cmd go docker
CRANK="$(ensure_crossplane_cli)"
cd "$ROOT"

case "$VERSION" in *-dev|*dirty*) die "refusing to publish VERSION=$VERSION — pass a real tag, e.g. make publish VERSION=v0.1.0";; esac

XPKG_FILES=()
for ARCH in $PLATFORMS; do
  xpkg="$("${ROOT}/scripts/build-xpkg.sh" "$ARCH")"
  "${ROOT}/scripts/verify-xpkg.sh" "$xpkg"
  XPKG_FILES+=("$xpkg")
done

REF="${IMAGE}:${VERSION}"
joined="$(IFS=,; echo "${XPKG_FILES[*]}")"
log "pushing multi-platform index ${REF} (${joined})"
"$CRANK" xpkg push -f "$joined" "$REF"
ok "pushed ${REF}"
