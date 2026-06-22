#!/usr/bin/env bash
# The full static gate, mirroring what CI runs — RUN THIS BEFORE PUSHING.
# CI failed once because lint (SA1019) + a generate/go.mod drift were not caught
# locally; this reproduces the lint and check-diff jobs so they're caught here.
source "$(dirname "$0")/lib.sh"
require_cmd go gofmt git
cd "$ROOT"

# golangci-lint (auto-discovers .golangci.yml). Fetch the pinned version into
# ./bin if not on PATH.
GCL_VERSION="${GCL_VERSION:-v2.12.2}"
GCL="$(command -v golangci-lint || true)"
if [ -z "$GCL" ]; then
  GCL="${ROOT}/bin/golangci-lint"
  if [ ! -x "$GCL" ]; then
    log "fetching golangci-lint ${GCL_VERSION}"
    curl -fsSL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b "${ROOT}/bin" "$GCL_VERSION" >/dev/null 2>&1 || true
  fi
fi

log "build"
go build ./...
ok "build"

log "unit tests"
go test ./... >/dev/null
ok "tests"

log "lint (golangci-lint run)"
"$GCL" run ./...
ok "lint clean"

# check-diff equivalent: regenerate (go generate) + tidy, then assert no diff —
# exactly what CI's `make reviewable` + `git diff --exit-code` does.
log "generated code + go.mod are current (check-diff)"
"${ROOT}/scripts/generate.sh" >/dev/null
go mod tidy
if ! git diff --quiet; then
  git --no-pager diff --stat
  die "working tree changed after generate + go mod tidy — commit the result (this is what CI's check-diff fails on)"
fi
ok "generated code + go.mod current"

# Workflow YAMLs must parse (a bad one fails as a 0s 'workflow file issue').
if command -v python3 >/dev/null 2>&1; then
  log "workflow YAML syntax"
  for f in .github/workflows/*.yml; do
    python3 -c "import yaml,sys; yaml.safe_load(open('$f'))" || die "invalid workflow YAML: $f"
  done
  ok "workflows parse"
fi

ok "validate passed — safe to push"
