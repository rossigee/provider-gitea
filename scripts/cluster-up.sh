#!/usr/bin/env bash
# Create a local kind cluster and install Crossplane (v2) into it, so the whole
# install/e2e loop is reproducible from nothing. Idempotent.
source "$(dirname "$0")/lib.sh"
require_cmd kind kubectl helm

if kind get clusters 2>/dev/null | grep -qx "$KIND_CLUSTER"; then
  ok "kind cluster '$KIND_CLUSTER' already exists"
else
  log "creating kind cluster '$KIND_CLUSTER'"
  kind create cluster --name "$KIND_CLUSTER"
fi
kubectl config use-context "kind-${KIND_CLUSTER}"

log "installing Crossplane via helm"
helm repo add crossplane-stable https://charts.crossplane.io/stable >/dev/null 2>&1 || true
helm repo update >/dev/null
helm upgrade --install crossplane crossplane-stable/crossplane \
  --namespace crossplane-system --create-namespace --wait

log "waiting for Crossplane to be ready"
kubectl -n crossplane-system rollout status deploy/crossplane --timeout=180s
ok "cluster '$KIND_CLUSTER' ready with Crossplane — context kind-${KIND_CLUSTER}"
