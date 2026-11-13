#!/usr/bin/env bash
# Misconfiguration scan of what this repository deploys, with Trivy.
#
#   scripts/security-scan.sh            gate: fail on HIGH/CRITICAL in this repo's charts
#   scripts/security-scan.sh --report   also print the findings in third-party charts
#
# Two scopes, on purpose:
#
#   - The charts under deploy/charts/ are written here, so their findings are this
#     repository's to fix or to accept. HIGH and CRITICAL fail the scan unless
#     .trivyignore.yaml accepts them with a reason.
#   - Third-party charts (ingress-nginx, the operators, kube-prometheus-stack, the
#     Debezium Platform chart itself) are reported and never fail the scan. Their
#     findings are mostly operator RBAC, which is what an operator is; they belong
#     upstream, and a gate nobody here can turn green is a gate that gets ignored.
#
# The scan runs on rendered manifests, not on the chart sources: several charts only
# render with the values the helmfile supplies, and the values are where a security
# context is set or dropped.
set -euo pipefail

report=false
[[ "${1:-}" == "--report" ]] && report=true

load_env_file() {
  local file="$1"
  [[ -f "${file}" ]] || return 0
  set -a
  # shellcheck disable=SC1090
  source "${file}"
  set +a
}

# versions.env only, never .env: the scan must give the same verdict on every machine.
load_env_file "deploy/environment/versions.env"
export DPL_DOMAIN="platform.debezium.local"
export DPL_NAMESPACE="dmp"
# Placeholders. The DNS-01 webhook values refuse to render without them, and nothing
# is deployed from here, so they only have to be present.
export DPL_GANDI_PAT="placeholder-for-rendering"
export DPL_ACME_EMAIL="scan@example.com"

HELMFILE=(helmfile --file deploy/helmfile.yaml.gotmpl)

work="$(mktemp -d)"
trap 'rm -rf "${work}"' EXIT

# Every component that needs no extra secret to render. Together with the two example
# selections below this reaches every chart under deploy/charts/ that the helmfile uses.
cat > "${work}/all-components.yaml" <<'YAML'
components:
  postgres: true
  mongodb: true
  sqlserver: true
  kafka: true
  kafka-ui: true
  kafka-connect: true
  http-server: true
  cdc-dashboard: true
  apicurio: true
YAML

render() {
  local name="$1" env="$2" components="$3"

  echo "Rendering ${name} (DPL_ENV=${env})"
  # helmfile narrates every repository and release on stderr. Keep it for the failure
  # case only, where it is the only place the cause is named.
  if ! DPL_ENV="${env}" "${HELMFILE[@]}" template \
      --state-values-file "${components}" \
      --output-dir "${work}/render/${name}" \
      --output-dir-template '{{ .OutputDir }}/{{ .Release.Name }}' \
      > /dev/null 2> "${work}/render-${name}.log"; then
    cat "${work}/render-${name}.log" >&2
    return 1
  fi
}

render local  local  "${work}/all-components.yaml"
render sso    public "$(pwd)/examples/k3s/platform-sso/components.yaml"
render ocp    ocp    "$(pwd)/examples/ocp/components.yaml"

# A rendered release is <release>/<chart name>/templates/. A chart is "ours" when a
# directory of that name exists under deploy/charts/. keycloak-operator is the
# exception: it is the upstream operator's manifests, vendored by
# scripts/vendor-keycloak-operator.sh, so it counts as third-party.
mkdir -p "${work}/own"
for chart_dir in deploy/charts/*/; do
  chart="$(basename "${chart_dir}")"
  [[ "${chart}" == "keycloak-operator" ]] && continue
  for rendered in "${work}"/render/*/*/"${chart}"; do
    [[ -d "${rendered}" ]] || continue
    profile="$(basename "$(dirname "$(dirname "${rendered}")")")"
    mkdir -p "${work}/own/${profile}"
    cp -r "${rendered}" "${work}/own/${profile}/${chart}"
  done
done

echo
echo "==> Charts in this repository (gate: HIGH, CRITICAL)"
find "${work}/own" -mindepth 2 -maxdepth 2 -type d -printf '%P\n' | sort | sed 's/^/  /'

status=0
(
  cd "${work}/own"
  trivy config \
    --quiet \
    --skip-version-check \
    --severity HIGH,CRITICAL \
    --ignorefile "${OLDPWD}/.trivyignore.yaml" \
    --exit-code 1 \
    .
) || status=$?

if [[ "${report}" == true ]]; then
  echo
  echo "==> Everything rendered, third-party charts included (report only)"
  (
    cd "${work}/render"
    trivy config \
      --quiet \
      --skip-version-check \
      --severity HIGH,CRITICAL \
      --exit-code 0 \
      .
  )
fi

if [[ "${status}" -ne 0 ]]; then
  echo
  echo "HIGH or CRITICAL findings in this repository's charts. Fix them, or accept them" >&2
  echo "in .trivyignore.yaml with a reason." >&2
  exit "${status}"
fi

echo
echo "OK"
