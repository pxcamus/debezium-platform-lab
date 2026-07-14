#!/usr/bin/env bash
# =============================================================================
# vendor-keycloak-operator.sh
# -----------------------------------------------------------------------------
# WHY THIS EXISTS
#   Every other operator in this repo (cert-manager, cnpg, strimzi, otel) ships
#   as a published Helm chart, so helmfile just pulls it. The Keycloak Operator
#   does NOT — the project distributes it only as raw manifests at a GitHub URL
#   (one Deployment/RBAC file + two CRDs). Helmfile installs *charts*, not loose
#   URLs, so to manage the operator like all our other infra we wrap those
#   manifests in a local chart: deploy/charts/keycloak-operator/.
#
# WHAT IT DOES
#   Downloads the three upstream files, PINNED to KEYCLOAK_OPERATOR_VERSION, into
#   that local chart:
#     - crds/  : the two CustomResourceDefinitions (Keycloak, KeycloakRealmImport)
#     - templates/operator.yaml : the operator Deployment + RBAC + Service
#   It also applies the ONE edit the upstream file needs (see step 3 below).
#
#   Vendoring (committing the pinned files to git) keeps the repo self-contained
#   and reproducible: installs work offline, and `git diff` shows exactly what
#   changed when you bump the version — instead of a runtime `kubectl apply -f
#   <moving-url>` that nobody can audit.
#
# WHEN TO RUN IT
#   Once now, and again whenever you bump KEYCLOAK_OPERATOR_VERSION in
#   deploy/environment/versions.env. Commit the regenerated files.
#
# USAGE
#   scripts/vendor-keycloak-operator.sh                 # reads versions.env
#   KEYCLOAK_OPERATOR_VERSION=26.7.0 scripts/vendor-keycloak-operator.sh
# =============================================================================
set -euo pipefail

repo_root="$(cd "$(dirname "$0")/.." && pwd)"

# --- Resolve the pinned version ---------------------------------------------
# Prefer an explicit env var; otherwise read the pin from versions.env so the
# vendored files always match the version the rest of the repo deploys.
if [[ -z "${KEYCLOAK_OPERATOR_VERSION:-}" ]]; then
  versions_file="${repo_root}/deploy/environment/versions.env"
  if [[ -f "${versions_file}" ]]; then
    KEYCLOAK_OPERATOR_VERSION="$(grep -E '^KEYCLOAK_OPERATOR_VERSION=' "${versions_file}" | tail -1 | cut -d= -f2)"
  fi
fi
: "${KEYCLOAK_OPERATOR_VERSION:?Set KEYCLOAK_OPERATOR_VERSION (env or deploy/environment/versions.env)}"

version="${KEYCLOAK_OPERATOR_VERSION}"
base="https://raw.githubusercontent.com/keycloak/keycloak-k8s-resources/${version}/kubernetes"
chart_dir="${repo_root}/deploy/charts/keycloak-operator"

echo "==> Vendoring Keycloak Operator ${version}"
echo "    source: ${base}"
echo "    target: ${chart_dir}"
mkdir -p "${chart_dir}/crds" "${chart_dir}/templates"

# --- Step 1 & 2: the two CRDs -----------------------------------------------
# These teach Kubernetes about the `Keycloak` and `KeycloakRealmImport` kinds.
# They go in crds/ (not templates/) because Helm installs crds/ ONCE and never
# upgrades or deletes them — deleting a CRD would delete every CR (and thus your
# Keycloak). The tradeoff: after a version bump you may need to apply the new
# CRDs by hand. They are large (~500 KB each), so use server-side apply to dodge
# the client-side "metadata.annotations: Too long" limit:
#     kubectl apply --server-side -f deploy/charts/keycloak-operator/crds/
echo "--> [1/3] CRD: Keycloak"
curl -fsSL "${base}/keycloaks.k8s.keycloak.org-v1.yml" \
  -o "${chart_dir}/crds/keycloaks.k8s.keycloak.org-v1.yml"
echo "--> [2/3] CRD: KeycloakRealmImport"
curl -fsSL "${base}/keycloakrealmimports.k8s.keycloak.org-v1.yml" \
  -o "${chart_dir}/crds/keycloakrealmimports.k8s.keycloak.org-v1.yml"

# --- Step 3: the operator Deployment + RBAC ---------------------------------
# THE ONE EDIT: upstream kubernetes.yml hardcodes `namespace: keycloak` in the
# ClusterRoleBinding subject (it assumes you install into a `keycloak` namespace).
# We install into `identity`, so we retarget that single line to the chart's
# release namespace. Missing this would leave the operator without the RBAC it
# needs and it would silently fail to reconcile anything.
echo "--> [3/3] Operator Deployment + RBAC (retargeting ClusterRoleBinding namespace -> release namespace)"
tmp="$(mktemp)"
curl -fsSL "${base}/kubernetes.yml" -o "${tmp}"
sed 's/^\( *namespace: \)keycloak$/\1{{ .Release.Namespace }}/' "${tmp}" \
  > "${chart_dir}/templates/operator.yaml"
rm -f "${tmp}"

echo "==> Done. Review the changes with:  git diff --stat ${chart_dir#${repo_root}/}"
