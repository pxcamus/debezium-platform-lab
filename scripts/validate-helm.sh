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

# Charts are not linted individually: helmfile renders every release below, which
# applies the same per-environment values layer the deployment uses, and pipes the
# result through kubeconform. A per-chart loop lived here and validated the charts
# against different values than production, so it was removed.

echo "==> Helmfile lint"
DPL_DEBEZIUM_VERSION="${DPL_DEBEZIUM_VERSION}" \
DPL_ENV="${DPL_ENV}" \
DPL_DOMAIN="${DPL_DOMAIN}" \
DPL_NAMESPACE="${DPL_NAMESPACE}" \
DPL_IMAGE_TAG="${DPL_IMAGE_TAG}" \
DPL_IMAGE_TAG_CONDUCTOR="${DPL_IMAGE_TAG_CONDUCTOR}" \
helmfile --file deploy/helmfile.yaml.gotmpl lint

helmfile_envs=(
  local
#  aws
)

helmfile_env_required_files() {
  local env="$1"

  printf '%s\n' "deploy/values/dmp/${env}.yaml.gotmpl"
  printf '%s\n' "deploy/values/apicurio/${env}.yaml.gotmpl"
}

helmfile_env_is_complete() {
  local env="$1"
  local file

  while IFS= read -r file; do
    if [[ ! -f "${file}" ]]; then
      echo "Skipping DPL_ENV=${env}: missing ${file}"
      return 1
    fi
  done < <(helmfile_env_required_files "${env}")

  return 0
}

template_helmfile_env() {
  local env="$1"

  echo "Rendering Helmfile with DPL_ENV=${env}"

  DPL_DEBEZIUM_VERSION="${DPL_DEBEZIUM_VERSION}" \
  DPL_ENV="${env}" \
  DPL_DOMAIN="${DPL_DOMAIN}" \
  DPL_NAMESPACE="${DPL_NAMESPACE}" \
  DPL_IMAGE_TAG="${DPL_IMAGE_TAG}" \
  DPL_IMAGE_TAG_CONDUCTOR="${DPL_IMAGE_TAG_CONDUCTOR}" \
  helmfile --file deploy/helmfile.yaml.gotmpl template \
    | kubeconform \
        -schema-location default \
        -ignore-missing-schemas \
        -summary
}

echo "==> Helmfile template environments"
for env in "${helmfile_envs[@]}"; do
  if helmfile_env_is_complete "${env}"; then
    template_helmfile_env "${env}"
  fi
done