#!/usr/bin/env bash
set -euo pipefail

load_env_file() {
  local file="$1"

  if [[ ! -f "${file}" ]]; then
    return
  fi

  echo "Loading environment from ${file}"

  set -a
  # shellcheck disable=SC1090
  source "${file}"
  set +a
}

load_env_file "deploy/environment/versions.env"
load_env_file ".env"

: "${DPL_DEBEZIUM_VERSION:?DPL_DEBEZIUM_VERSION must be set in deploy/environment/versions.env, .env, or the environment}"
: "${DPL_ENV:=local}"
: "${DPL_DOMAIN:=platform.debezium.local}"
: "${DPL_NAMESPACE:=dmp}"
: "${DPL_IMAGE_TAG:=nightly}"
: "${DPL_IMAGE_TAG_CONDUCTOR:=nightly}"

# load_env_file exports everything it sources; these six may come from the defaults
# above instead, so export them explicitly. Every helmfile call below then inherits
# the same environment rather than repeating a prefix block per invocation.
export DPL_DEBEZIUM_VERSION DPL_ENV DPL_DOMAIN DPL_NAMESPACE DPL_IMAGE_TAG DPL_IMAGE_TAG_CONDUCTOR

HELMFILE=(helmfile --file deploy/helmfile.yaml.gotmpl)

# Charts are not linted individually: helmfile renders every release below, which
# applies the same per-environment values layer the deployment uses, and pipes the
# result through kubeconform. A per-chart loop lived here and validated the charts
# against different values than production, so it was removed.

echo "==> Helmfile lint"
"${HELMFILE[@]}" lint

# --- Release gating ---------------------------------------------------------
#
# The single most valuable invariant in this repo, and the one that was silently
# false for months: `installed:` defaults to TRUE in helmfile, so any release added
# without an explicit gate installs everywhere. A bare checkout once produced 14
# releases — Kafka, Mongo, SQL Server, Apicurio and the rest — where four were
# wanted.
#
# These lists are the contract behind `cp .env.example .env`: that, and nothing
# else, is what a new user gets. Adding a release means deciding which axis gates it
# (DPL_ENV for how, .Values.components for what) and, if it is genuinely part of the
# base, adding it here on purpose.

expected_base=(
  debezium-platform
  ingress-nginx
  kube-prometheus-stack
  opentelemetry-operator
)

installed_releases() {
  local mode="$1"

  # DPL_ENV and DPL_OPERATOR_MODE are pinned rather than inherited: a developer with
  # DPL_ENV=homelab in .env must still get the same verdict CI does.
  #
  # awk over the default table output rather than jq over --output json: awk is in
  # every base image (mawk on Ubuntu, so also under WSL) while jq is an extra install,
  # and this is the only thing in the repo that wanted it. The header check is what
  # buys back jq's stability — if helmfile ever reorders its columns this fails loudly
  # instead of quietly reporting that nothing is installed. Same awk as `just releases`.
  DPL_ENV=local DPL_OPERATOR_MODE="${mode}" "${HELMFILE[@]}" list \
    | awk -F'\t' '
        NR == 1 {
          if ($4 !~ /^INSTALLED/) {
            print "helmfile list columns changed: expected INSTALLED in column 4" > "/dev/stderr"
            exit 1
          }
          next
        }
        $4 ~ /^true/ { gsub(/[[:space:]]+$/, "", $1); print $1 }
      ' \
    | sort
}

assert_releases() {
  local mode="$1"
  shift

  local expected actual
  expected="$(printf '%s\n' "$@" | sort)"
  actual="$(installed_releases "${mode}")"

  if [[ "${actual}" != "${expected}" ]]; then
    echo "DPL_OPERATOR_MODE=${mode}: unexpected release set" >&2
    diff <(echo "${expected}") <(echo "${actual}") \
      --label expected --label actual --unified=99 >&2 || true
    return 1
  fi

  echo "DPL_OPERATOR_MODE=${mode}: $(echo "${actual}" | tr '\n' ' ')"
}

echo "==> Release gating (DPL_ENV=local, no .env overrides)"
assert_releases bundled "${expected_base[@]}"
assert_releases standalone debezium-operator "${expected_base[@]}"

# --- Operator packaging -----------------------------------------------------
#
# The operator ships either as a debezium-platform subchart (bundled) or as its own
# release (standalone) — exactly one, never both, never neither. Those are two
# separate switches that must always disagree, and they drifted before: aws had the
# subchart enabled AND the ungated standalone release, so it installed two operators
# into the same namespace. Counting the rendered Deployment catches that; the release
# check above cannot, because one of the two operators is a subchart.

assert_single_operator() {
  local mode="$1" count

  count="$(DPL_ENV=local DPL_OPERATOR_MODE="${mode}" "${HELMFILE[@]}" template \
    | grep -c 'image: quay.io/debezium/operator' || true)"

  if [[ "${count}" -ne 1 ]]; then
    echo "DPL_OPERATOR_MODE=${mode}: expected exactly 1 operator Deployment, rendered ${count}" >&2
    return 1
  fi

  echo "DPL_OPERATOR_MODE=${mode}: 1 operator"
}

echo "==> Operator packaging is exclusive"
assert_single_operator bundled
assert_single_operator standalone

# --- Per-environment rendering ----------------------------------------------

# Profiles, not venues: `local` is Kind with no TLS, `public` is a real cluster with a
# real domain and cert-manager. `aws` and `hetzner` dropped out when their values files
# were removed in the values consolidation — re-add them here the moment
# deploy/values/dmp/<env>.yaml.gotmpl comes back, or the check below stops covering them.
helmfile_envs=(
  local
  public
)

template_helmfile_env() {
  local env="$1"

  # Every environment needs its own dmp values; the rest of the tree is either
  # component-gated (and so not rendered by default) or covered by
  # missingFileHandler. A missing file here is an error rather than a skip — the
  # aws render sat commented out and unnoticed for months because it was a skip.
  local required="deploy/values/dmp/${env}.yaml.gotmpl"
  if [[ ! -f "${required}" ]]; then
    echo "DPL_ENV=${env} is in helmfile_envs but ${required} does not exist" >&2
    return 1
  fi

  echo "Rendering Helmfile with DPL_ENV=${env}"

  DPL_ENV="${env}" "${HELMFILE[@]}" template \
    | kubeconform \
        -schema-location default \
        -ignore-missing-schemas \
        -summary
}

echo "==> Helmfile template environments"
for env in "${helmfile_envs[@]}"; do
  template_helmfile_env "${env}"
done
