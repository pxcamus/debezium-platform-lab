#!/usr/bin/env bash
# Check that the DPL_* variable surface is internally consistent.
#
#   scripts/validate-env.sh
#
# Two invariants, both guarding failure modes that are silent at runtime:
#
#   1. Every ${DPL_*} expanded into a DMP payload is declared in .env.example or
#      deploy/environment/versions.env. Expansion turns a missing variable into an
#      empty string rather than an error, so an undeclared name ships a malformed
#      DMP resource instead of failing.
#
#   2. No pre-DPL_ variable name survives. A leftover legacy name is inert — it is
#      read by nothing and reported by nothing.
#
# Run from the repo root.
set -euo pipefail

readonly ENV_EXAMPLE=".env.example"
readonly VERSIONS_ENV="deploy/environment/versions.env"

# Payload trees whose JSON is expanded from the environment.
readonly PAYLOAD_PATHS=(data-pipelines/config resources)

# Pre-DPL_ names. Kept explicit rather than pattern-matched so that legitimate
# internals (LAB_LABEL) and pod-level variables inside deploy/charts/ are not
# swept up by a prefix match.
readonly LEGACY_NAMES=(
  DBZ_ENV DBZ_DOMAIN DBZ_VERSION DBZ_NAMESPACE DBZ_IMAGE_TAG
  DBZ_IMAGE_TAG_CONDUCTOR DBZ_IMAGE_REGISTRY DBZ_IMAGE_REGISTRY_CONDUCTOR
  DBZ_OPERATOR_TAG
  CLUSTER_TYPE KIND_CLUSTER_NAME KIND_CONFIG
  K3S_HOST K3S_SSH_KEY K3S_SSH_USER K3S_CONTEXT
  K3S_LOCAL_KUBECONFIG K3S_REMOTE_KUBECONFIG
  LAB_NAME LAB_TYPE LAB_IMAGE LAB_LOCATION LAB_IDENTITY LAB_SSH_KEY
  LAB_OPEN_HTTP LAB_YES LAB_DOMAIN LAB_IP
  DMP_RESOURCE_PREFIX DMP_ENVIRONMENT DMP_BASE_URL
  MONGODB_URI MONGODB_DIRECT_CONNECTION POSTGRESQL_HOST
  KAFKA_DMP_BOOTSTRAP_SERVERS HTTPSERVER_DMP_CONNECTION_STRING
  MONGODB_DMP_CONNECTION_STRING
  LOG_LEVEL HELM_VERSION HELMFILE_VERSION
  KEYCLOAK_VERSION KEYCLOAK_OPERATOR_VERSION
  OTEL_OPERATOR_VERSION KUBE_PROMETHEUS_STACK_VERSION OAUTH2_PROXY_CHART_VERSION
)

# Files allowed to mention a legacy name: the transition guard names them on
# purpose, and deploy/charts/ has its own pod-level variables that never came
# from the environment.
readonly LEGACY_EXCLUDES=(
  ':(exclude)scripts/lib/env.sh'
  ':(exclude)scripts/validate-env.sh'
  ':(exclude)deploy/charts'
  ':(exclude).env.example'
)

fail=0

note() { echo "  $*"; }

# Indent every line of a multi-line string. shellcheck suggests ${var//…} instead,
# but that cannot prefix each line without newline gymnastics.
# shellcheck disable=SC2001
indent() { sed 's/^/  /'; }

check_payload_vars_declared() {
  echo "==> DMP payload variables are declared"

  if [[ ! -f "${ENV_EXAMPLE}" ]]; then
    echo "error: ${ENV_EXAMPLE} not found — run from the repo root" >&2
    exit 1
  fi

  # Declared: assignments in either env file, commented-out ones included — a
  # commented default still documents the name.
  local declared
  declared="$(cat "${ENV_EXAMPLE}" "${VERSIONS_ENV}" 2>/dev/null |
    sed -n 's/^[[:space:]]*#\{0,1\}[[:space:]]*\([A-Z][A-Z0-9_]*\)=.*/\1/p' |
    sort -u)"

  # Used: ${DPL_...} occurrences in the payload trees. Skip paths that do not
  # exist so the check survives trees being added or retired, and tolerate a
  # no-match grep (exit 1) without tripping pipefail.
  local -a present=()
  local path
  for path in "${PAYLOAD_PATHS[@]}"; do
    [[ -d "${path}" ]] && present+=("${path}")
  done

  if ((${#present[@]} == 0)); then
    note "none of ${PAYLOAD_PATHS[*]} present — nothing to check"
    return
  fi

  local used
  # The single quotes are deliberate: tr deletes the literal $ { } characters.
  # shellcheck disable=SC2016
  used="$( (grep -rhoE '\$\{DPL_[A-Z0-9_]+\}' "${present[@]}" || true) |
    tr -d '${}' | sort -u)"

  if [[ -z "${used}" ]]; then
    note "no \${DPL_*} placeholders found — nothing to check"
    return
  fi

  local undeclared
  undeclared="$(comm -23 <(echo "${used}") <(echo "${declared}") || true)"

  if [[ -n "${undeclared}" ]]; then
    echo "error: expanded in payloads but declared in neither ${ENV_EXAMPLE} nor ${VERSIONS_ENV}:" >&2
    echo "${undeclared}" | indent >&2
    echo "       these expand to an empty string, producing a malformed DMP resource." >&2
    fail=1
  else
    note "$(echo "${used}" | wc -l | tr -d ' ') payload variables, all declared"
  fi
}

check_no_legacy_names() {
  echo "==> no pre-DPL_ variable names survive"

  local found=0 name hits
  for name in "${LEGACY_NAMES[@]}"; do
    # -w so DPL_KIND_CONFIG does not match the legacy KIND_CONFIG.
    if hits="$(git grep -wnF -e "${name}" -- "${LEGACY_EXCLUDES[@]}" 2>/dev/null)"; then
      echo "error: legacy variable ${name} is still referenced:" >&2
      echo "${hits}" | indent >&2
      found=1
    fi
  done

  if ((found)); then
    note "legacy names are read by nothing — rename them to DPL_*"
    fail=1
  else
    note "${#LEGACY_NAMES[@]} legacy names checked, none present"
  fi
}

main() {
  check_payload_vars_declared
  check_no_legacy_names

  echo
  if ((fail)); then
    echo "FAILED"
    exit 1
  fi
  echo "OK"
}

main "$@"
