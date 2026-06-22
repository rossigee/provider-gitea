#!/usr/bin/env bash
# uptest --setup-script: prepares the cluster for the e2e examples, against a
# REAL Gitea (installed by scripts/e2e.sh via the gitea Helm chart):
#   - mints an admin API token (most controllers authenticate with this token
#     via the ProviderConfig) via `gitea admin user generate-access-token`,
#   - writes the token into the credentials Secret,
#   - creates the password Secrets the basic-auth examples reference: the admin
#     password Secret (AccessToken authenticates as gitea_admin via HTTP basic
#     auth — Gitea gates token CRUD on the owning user) and the User example's
#     password Secret (read via passwordSecretRef),
#   - applies the cluster-scoped ProviderConfig pointing at the in-cluster Gitea,
#   - waits until the provider is Healthy.
#
# Run by scripts/e2e.sh via `uptest e2e --setup-script=...`; KUBECTL is exported
# by that wrapper so uptest and this script use the same kube-context.
set -aeuo pipefail
: "${KUBECTL:=kubectl}"
# uptest runs this setup-script from its own working dir, so resolve the repo
# root from the script location for any file paths (e.g. the example manifests).
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
NS="${E2E_NS:-provider-gitea-e2e}"
GITEA_NS="${GITEA_NS:-gitea}"
GITEA_RELEASE="${GITEA_RELEASE:-gitea}"
GITEA_ADMIN="${GITEA_ADMIN_USER:-gitea_admin}"
# Must match the admin password scripts/e2e.sh sets on the Helm release, so the
# AccessToken example can basic-auth as gitea_admin.
GITEA_ADMIN_PASSWORD="${GITEA_ADMIN_PASSWORD:-Uptest-Admin-123}"

${KUBECTL} create namespace "$NS" --dry-run=client -o yaml | ${KUBECTL} apply -f - >/dev/null

echo "uptest-setup: minting admin API token in the gitea pod"
# generate-access-token --raw prints just the token. The token name must be
# unique per call, so include a fixed e2e name; on a fresh cluster this is the
# first one. `all` scope so every resource controller can act.
TOKEN="$(${KUBECTL} -n "$GITEA_NS" exec "deploy/${GITEA_RELEASE}" -c gitea -- \
  gitea admin user generate-access-token --username "$GITEA_ADMIN" \
  --token-name "e2e" --scopes all --raw 2>/dev/null | tr -d '\r\n' | tail -c 64)"
[ -n "$TOKEN" ] || { echo "uptest-setup: FAILED to mint gitea token" >&2; exit 1; }

echo "uptest-setup: credentials + password Secrets in ${NS}"
${KUBECTL} -n "$NS" create secret generic gitea-creds \
  --from-literal=token="$TOKEN" \
  --dry-run=client -o yaml | ${KUBECTL} apply -f -
# gitea_admin's real password — the AccessToken example basic-auths with this.
${KUBECTL} -n "$NS" create secret generic uptest-admin-pw \
  --from-literal=password="$GITEA_ADMIN_PASSWORD" \
  --dry-run=client -o yaml | ${KUBECTL} apply -f -
# Password for the User example (read via passwordSecretRef on create).
${KUBECTL} -n "$NS" create secret generic uptest-user-password \
  --from-literal=password=Uptest-User-123 \
  --dry-run=client -o yaml | ${KUBECTL} apply -f -
# Secret the RepositorySecret + OrganizationSecret examples' valueSecretRef point at.
${KUBECTL} -n "$NS" create secret generic uptest-secret-value \
  --from-literal=value=repo-secret-value \
  --dry-run=client -o yaml | ${KUBECTL} apply -f -

# Seed the user the RepositoryCollaborator example references, so it pre-exists
# before the MRs reconcile. uptest applies all examples in parallel, so a
# collaborator whose target user is itself a (parallel) MR can have its Create
# fire before the user exists -> 422 -> a create-pending deadlock. Seeding the
# dependency (as provider-harbor's e2e seeds an image) avoids the race.
${KUBECTL} -n "$GITEA_NS" exec "deploy/${GITEA_RELEASE}" -c gitea -- \
  gitea admin user create --username uptest-collab-user --password 'Collab-123!' \
  --email collab@local.domain --must-change-password=false 2>/dev/null || true

# ProviderConfig is CLUSTER-scoped (apis/v1beta1, scope=Cluster). baseURL points
# at the in-cluster Gitea HTTP Service; the client appends /api/v1.
cat <<YAML | ${KUBECTL} apply -f -
apiVersion: gitea.crossplane.io/v1beta1
kind: ProviderConfig
metadata:
  name: e2e
spec:
  baseURL: http://${GITEA_RELEASE}-http.${GITEA_NS}.svc.cluster.local:3000
  credentials:
    source: Secret
    secretRef:
      namespace: ${NS}
      name: gitea-creds
      key: token
YAML

echo "uptest-setup: waiting until provider is Healthy"
${KUBECTL} wait provider.pkg --all --for condition=Healthy --timeout 5m

# Pre-create the foundational resources almost everything else depends on (the
# repository and the organization) and wait for them Ready BEFORE uptest applies
# the full set. uptest applies all examples in parallel; a dependent whose parent
# repo/org does not exist yet gets a transient 404 on Create, and
# crossplane-runtime's create-pending safety can then refuse to retry
# ("cannot determine creation result"). Seeding the parents first removes that
# race (provider-harbor's e2e seeds its dependencies for the same reason). uptest
# then adopts these (already Ready) and still drives + deletes them.
echo "uptest-setup: pre-creating foundational repository + organization"
${KUBECTL} apply -f "${ROOT}/examples/e2e/repository.yaml" -f "${ROOT}/examples/e2e/organization.yaml" >/dev/null
${KUBECTL} -n "$NS" wait repository.repository.gitea.m.crossplane.io/uptest-repo \
  organization.organization.gitea.m.crossplane.io/uptest-org \
  --for condition=Ready --timeout 3m
echo "uptest-setup: done"
