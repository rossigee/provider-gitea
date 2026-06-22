#!/usr/bin/env bash
# The CRD-verify gate provider-gitea's release.yml lacked: extract the built
# xpkg and assert package.yaml carries the Provider meta AND every CRD under
# package/crds/. A package that installs Healthy but registers 0 CRDs is exactly
# what this catches (lesson #5). Pass an .xpkg path, or it builds an amd64 one.
source "$(dirname "$0")/lib.sh"
require_cmd tar grep
cd "$ROOT"

XPKG="${1:-}"
[ -n "$XPKG" ] || XPKG="$("${ROOT}/scripts/build-xpkg.sh" amd64)"
[ -f "$XPKG" ] || die "xpkg not found: $XPKG"

EXPECTED="$(ls package/crds/*.yaml 2>/dev/null | wc -l | tr -d ' ')"
[ "$EXPECTED" -gt 0 ] || die "no CRDs under package/crds — run scripts/generate.sh"

rm -rf _verify && mkdir _verify
tar -xf "$XPKG" -C _verify
PKG_YAML=""
for blob in $(find _verify -type f); do
  if tar -tzf "$blob" 2>/dev/null | grep -qx 'package.yaml'; then
    tar -xzf "$blob" -C _verify package.yaml 2>/dev/null && PKG_YAML="_verify/package.yaml" && break
  fi
  if tar -tf "$blob" 2>/dev/null | grep -qx 'package.yaml'; then
    tar -xf "$blob" -C _verify package.yaml 2>/dev/null && PKG_YAML="_verify/package.yaml" && break
  fi
done
[ -n "$PKG_YAML" ] || die "could not locate package.yaml inside $XPKG"

PROV="$(grep -c '^kind: Provider$' "$PKG_YAML" || true)"
CRDS="$(grep -c '^kind: CustomResourceDefinition$' "$PKG_YAML" || true)"
log "package.yaml: Provider docs=$PROV  CustomResourceDefinition docs=$CRDS (expected $EXPECTED)"
[ "$PROV" -ge 1 ] || die "no Provider meta in package.yaml"
[ "$CRDS" -eq "$EXPECTED" ] || die "expected $EXPECTED CRDs in package.yaml, found $CRDS — packaging is broken"
ok "package.yaml carries the Provider meta + all $CRDS CRDs"
