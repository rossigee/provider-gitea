#!/usr/bin/env bash
# uptest --setup-script: prepares the cluster for the e2e examples.
#   - deploys the in-cluster mock Gitea backend (test/e2e/mock-gitea.yaml),
#   - creates the credentials Secret the ProviderConfig points at (dummy token —
#     the mock ignores auth),
#   - applies the cluster-scoped ProviderConfig the example MRs reference,
#   - waits until the provider is Healthy.
#
# Run by scripts/e2e.sh via `uptest e2e --setup-script=...`; KUBECTL is exported
# by that wrapper so uptest and this script use the same kube-context.
set -aeuo pipefail
: "${KUBECTL:=kubectl}"
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
NS="${E2E_NS:-provider-gitea-e2e}"

echo "uptest-setup: deploying mock Gitea in ${NS}"
${KUBECTL} apply -f "${ROOT}/test/e2e/mock-gitea.yaml"
${KUBECTL} -n "$NS" rollout status deploy/gitea-mock --timeout=120s

echo "uptest-setup: credentials Secret + ProviderConfig"
${KUBECTL} -n "$NS" create secret generic gitea-mock-creds \
  --from-literal=token=e2e-dummy-token \
  --dry-run=client -o yaml | ${KUBECTL} apply -f -

# Secret the AdminUser example's passwordSecretRef points at (its controller
# reads the password from a k8s Secret).
${KUBECTL} -n "$NS" create secret generic uptest-admin-pw \
  --from-literal=password=ChangeMe-123 \
  --dry-run=client -o yaml | ${KUBECTL} apply -f -

# ProviderConfig is CLUSTER-scoped (apis/v1beta1, scope=Cluster); it carries no
# namespace. The MRs reference it by name. baseURL points at the in-cluster mock
# Service; the client appends /api/v1.
cat <<YAML | ${KUBECTL} apply -f -
apiVersion: gitea.crossplane.io/v1beta1
kind: ProviderConfig
metadata:
  name: e2e
spec:
  baseURL: http://gitea-mock.${NS}.svc.cluster.local:8080
  credentials:
    source: Secret
    secretRef:
      namespace: ${NS}
      name: gitea-mock-creds
      key: token
YAML

echo "uptest-setup: waiting until provider is Healthy"
${KUBECTL} wait provider.pkg --all --for condition=Healthy --timeout 5m
echo "uptest-setup: done"
