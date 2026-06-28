#!/usr/bin/env bash
# Self-contained e2e for provider-gitea on KIND, driven by uptest.
#
# Stands up (idempotently): a kind cluster -> Crossplane -> the provider package
# -> a REAL Gitea (latest, via the official gitea Helm chart) -> then runs uptest
# v2 (apply -> Ready -> import -> delete) over examples/e2e/*. The whole loop runs
# on a throwaway kind cluster with NO external dependency: the provider package
# is built locally and served from a throwaway local registry, and Gitea runs
# in-cluster (sqlite, no persistence). uptest-setup.sh mints an admin API token.
#
# This is the "works by design" guarantee: a green run proves every example MR
# reaches Ready, updates, and deletes cleanly against a real Gitea — which
# enforces real ids, 404s, SSH-key validation, dependencies and auth. The update
# step mutates every example carrying a uptest.upbound.io/update-parameter
# annotation and re-waits Ready (create/observe/update/import/delete all proven).
#
# Env knobs (optional, defaults from scripts/lib.sh):
#   KIND_CLUSTER (provider-gitea-dev)  PROVIDER (provider-gitea)
#   E2E_NS (provider-gitea-e2e)  KEEP (set to keep the kind cluster afterwards)
source "$(dirname "$0")/lib.sh"
require_cmd kind kubectl helm go docker
CHAINSAW="${CHAINSAW:-$(command -v chainsaw || true)}"
[ -n "$CHAINSAW" ] || die "required command not found: chainsaw"
KCTX="kind-${KIND_CLUSTER}"
E2E_NS="${E2E_NS:-provider-gitea-e2e}"
cd "$ROOT" || die "cannot cd to repo root $ROOT"

# uptest v2 (namespaced-aware) is not `go install`-able; fetch the released
# binary, cached by version.
UPTEST_VERSION="${UPTEST_VERSION:-v2.2.0}"
UPTEST="${UPTEST:-$(go env GOPATH)/bin/uptest-${UPTEST_VERSION}}"
if [ ! -x "$UPTEST" ]; then
  log "downloading uptest ${UPTEST_VERSION}"
  curl -fsSL "https://github.com/crossplane/uptest/releases/download/${UPTEST_VERSION}/uptest_$(go env GOOS)-$(go env GOARCH)" -o "$UPTEST"
  chmod +x "$UPTEST"
fi

# 1. kind + Crossplane (idempotent).
"${ROOT}/scripts/cluster-up.sh"

ORIG_CTX="$(kubectl config current-context 2>/dev/null || true)"
cleanup() {
  if [ "${KEEP:-false}" != "true" ]; then
    log "deleting kind cluster ${KIND_CLUSTER}"
    kind delete cluster --name "$KIND_CLUSTER" >/dev/null 2>&1 || true
    docker rm -f "${REG_NAME:-provider-gitea-e2e-registry}" >/dev/null 2>&1 || true
  fi
  [ -n "$ORIG_CTX" ] && kubectl config use-context "$ORIG_CTX" >/dev/null 2>&1 || true
}
trap cleanup EXIT
kubectl config use-context "$KCTX" >/dev/null

# 2. build the xpkg locally and serve it from a throwaway local OCI registry that
#    BOTH the host and the kind node can reach on the same ref. Crossplane always
#    resolves the package ref tag -> digest via the registry from inside a Pod, so
#    a node-local `kind load` image isn't enough; a dotted-host registry that
#    CoreDNS + containerd both resolve is the smallest thing that satisfies it.
REG_NAME="${REG_NAME:-provider-gitea-e2e-registry}"
REG_PORT="${REG_PORT:-5001}"
if [ -z "$(docker ps -q -f "name=^${REG_NAME}$")" ]; then
  log "starting local registry ${REG_NAME} on :${REG_PORT}"
  docker run -d --restart=always -p "127.0.0.1:${REG_PORT}:5000" --name "$REG_NAME" registry:2 >/dev/null
fi
REG_HOST="${REG_HOST:-registry.e2e.local}"
docker network connect kind "$REG_NAME" 2>/dev/null || true
REG_IP="$(docker inspect -f '{{(index .NetworkSettings.Networks "kind").IPAddress}}' "$REG_NAME")"
[ -n "$REG_IP" ] || die "could not determine ${REG_NAME} IP on the kind network"
PUSH_REF="localhost:${REG_PORT}/${PROVIDER}:e2e"
REG_REF="${REG_HOST}:5000/${PROVIDER}:e2e"

log "pointing kind node containerd at ${REG_HOST}:5000 (plain http) + /etc/hosts"
for node in $(kind get nodes --name "$KIND_CLUSTER"); do
  docker exec "$node" mkdir -p "/etc/containerd/certs.d/${REG_HOST}:5000"
  cat <<HOSTS | docker exec -i "$node" cp /dev/stdin "/etc/containerd/certs.d/${REG_HOST}:5000/hosts.toml"
[host."http://${REG_HOST}:5000"]
  capabilities = ["pull", "resolve"]
  skip_verify = true
HOSTS
  docker exec "$node" sh -c "grep -q ' ${REG_HOST}\$' /etc/hosts || echo '${REG_IP} ${REG_HOST}' >> /etc/hosts"
done

log "adding ${REG_HOST} -> ${REG_IP} to CoreDNS (for the in-Pod digest resolve)"
CURRENT_CF="$(kubectl -n kube-system get configmap coredns -o jsonpath='{.data.Corefile}')"
if ! grep -q "$REG_HOST" <<<"$CURRENT_CF"; then
  NEW_CF="$(CURRENT_CF="$CURRENT_CF" REG_IP="$REG_IP" REG_HOST="$REG_HOST" python3 - <<'PY'
import os, sys
cf = os.environ["CURRENT_CF"]
block = "    hosts {\n        %s %s\n        fallthrough\n    }\n" % (os.environ["REG_IP"], os.environ["REG_HOST"])
out, done = [], False
for line in cf.splitlines(keepends=True):
    out.append(line)
    if not done and line.strip().startswith(".:53 {"):
        out.append(block); done = True
sys.stdout.write("".join(out))
PY
)"
  [ -n "$NEW_CF" ] || die "failed to build patched Corefile"
  kubectl -n kube-system create configmap coredns --from-literal=Corefile="$NEW_CF" \
    --dry-run=client -o yaml | kubectl -n kube-system apply -f -
  kubectl -n kube-system rollout restart deploy/coredns
  kubectl -n kube-system rollout status deploy/coredns --timeout=120s
fi

log "regenerating CRDs (must match what release.yml ships)"
"${ROOT}/scripts/generate.sh"
XPKG="$("${ROOT}/scripts/build-xpkg.sh" amd64)"
"${ROOT}/scripts/verify-xpkg.sh" "$XPKG"
log "pushing package to ${PUSH_REF} (pulled in-cluster as ${REG_REF})"
CRANK="$(ensure_crossplane_cli)"
# Push to the throwaway local registry with an ISOLATED docker config: the
# user's ~/.docker/config.json may set a credsStore (e.g. "desktop.exe" on
# Docker Desktop/WSL) whose credential helper isn't on PATH, which makes
# `crossplane xpkg push` fail with "docker-credential-...: executable file not
# found". The local registry needs no auth, so an empty config sidesteps it.
CLEAN_DOCKER_CFG="$(mktemp -d)"
echo '{}' > "${CLEAN_DOCKER_CFG}/config.json"
DOCKER_CONFIG="$CLEAN_DOCKER_CFG" "$CRANK" xpkg push -f "$XPKG" "$PUSH_REF"
rm -rf "$CLEAN_DOCKER_CFG"
cat <<EOF | kubectl apply -f -
apiVersion: pkg.crossplane.io/v1
kind: Provider
metadata:
  name: ${PROVIDER}
spec:
  package: ${REG_REF}
EOF
log "waiting for Installed + Healthy"
heal=""
for _ in $(seq 1 30); do
  heal="$(kubectl get provider.pkg "$PROVIDER" -o jsonpath='{.status.conditions[?(@.type=="Healthy")].status}' 2>/dev/null || true)"
  [ "$heal" = "True" ] && break
  sleep 5
done
[ "$heal" = "True" ] || die "provider did not become Healthy"
# Healthy is necessary but NOT sufficient: a mis-packaged provider goes Healthy
# with 0 CRDs, and an individually-invalid CRD (e.g. uniqueItems:true) silently
# fails to register while the provider stays Healthy — its controller then logs
# "no matches for kind" and that resource never works. So assert EVERY packaged
# CRD actually registered, not just ">0".
want="$(ls package/crds/*.yaml | wc -l | tr -d ' ')"
got="$(kubectl get crd -o name 2>/dev/null | grep -c 'gitea' || true)"
[ "$got" -eq "$want" ] || die "provider Healthy but only ${got}/${want} gitea CRDs registered — a CRD failed to install (check provider logs for 'no matches for kind')"
ok "installed: Healthy + ${got}/${want} CRDs registered"

# 3. install a REAL Gitea (latest, via the official chart) for the e2e to drive
#    — far stronger than a mock: it enforces real ids, 404s, key validation,
#    dependencies and auth. Lightweight config: sqlite + in-memory cache/session
#    + level queue, no bundled postgresql-ha/valkey-cluster, no persistence.
GITEA_NS="${GITEA_NS:-gitea}"
GITEA_RELEASE="${GITEA_RELEASE:-gitea}"
# Exported so the uptest --setup-script subprocess seeds the matching admin
# password Secret (the AccessToken example basic-auths as this user).
export GITEA_ADMIN_USER="${GITEA_ADMIN_USER:-gitea_admin}"
export GITEA_ADMIN_PASSWORD="${GITEA_ADMIN_PASSWORD:-Uptest-Admin-123}"
log "installing Gitea (chart gitea-charts/gitea) in ns ${GITEA_NS}"
helm repo add gitea-charts https://dl.gitea.com/charts/ >/dev/null 2>&1 || true
helm repo update gitea-charts >/dev/null 2>&1
kubectl create namespace "$GITEA_NS" --dry-run=client -o yaml | kubectl apply -f - >/dev/null
helm upgrade --install "$GITEA_RELEASE" gitea-charts/gitea -n "$GITEA_NS" \
  --set postgresql-ha.enabled=false \
  --set valkey-cluster.enabled=false \
  --set valkey.enabled=false \
  --set persistence.enabled=false \
  --set gitea.config.database.DB_TYPE=sqlite3 \
  --set gitea.config.session.PROVIDER=memory \
  --set gitea.config.cache.ADAPTER=memory \
  --set gitea.config.queue.TYPE=level \
  --set gitea.config.repository.DEFAULT_BRANCH=main \
  --set gitea.config.actions.ENABLED=true \
  --set gitea.config.security.DISABLE_GIT_HOOKS=false \
  --set gitea.admin.username="$GITEA_ADMIN_USER" \
  --set gitea.admin.password="$GITEA_ADMIN_PASSWORD" \
  --set gitea.admin.email=gitea@local.domain \
  --wait --timeout 10m >/dev/null
kubectl -n "$GITEA_NS" rollout status deploy/"$GITEA_RELEASE" --timeout=300s
ok "Gitea ready at ${GITEA_RELEASE}-http.${GITEA_NS}.svc:3000 (admin ${GITEA_ADMIN_USER})"

# 4. run uptest over the examples (apply -> Ready -> update -> import -> delete).
# The update step mutates each example that carries a
# uptest.upbound.io/update-parameter annotation (repository/organization/label/
# team/user) and re-waits Ready, exercising the Update path end-to-end; examples
# without the annotation (immutable/write-only kinds) just skip update.
log "running uptest e2e against the real Gitea backend"
LIST="$(ls examples/e2e/*.yaml | paste -sd, -)"

rc=0
KUBECTL="$(command -v kubectl)" CHAINSAW="$CHAINSAW" E2E_NS="$E2E_NS" \
  "$UPTEST" e2e "$LIST" \
  --setup-script="test/e2e/uptest-setup.sh" \
  --default-conditions=Ready --default-timeout=600s || rc=$?

[ "$rc" -eq 0 ] && ok "e2e PASSED" || warn "e2e FAILED (rc=$rc) — re-run with KEEP=true to inspect"
exit $rc
