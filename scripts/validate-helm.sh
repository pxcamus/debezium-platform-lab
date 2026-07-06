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

: "${DBZ_VERSION:?DBZ_VERSION must be set in deploy/environment/versions.env, .env, or the environment}"
: "${DBZ_ENV:=local}"
: "${DBZ_DOMAIN:=platform.debezium.local}"
: "${DBZ_NAMESPACE:=dmp}"
: "${DBZ_IMAGE_TAG:=nightly}"
: "${DBZ_IMAGE_TAG_CONDUCTOR:=nightly}"

charts=(
  deploy/charts/apicurio-registry
  deploy/charts/cdc-dashboard
  deploy/charts/http-server
  deploy/charts/kafka-cluster
  deploy/charts/kafka-connect
  deploy/charts/mongodb-replica-set
  deploy/charts/mssql
  deploy/charts/postgresql-cluster
)

chart_values_args() {
  local chart="$1"

  case "${chart}" in
    deploy/charts/cdc-dashboard)
      printf '%s\n' "--values" "deploy/values/cdc-dashboard/local.yaml"
      ;;
    deploy/charts/mongodb-replica-set)
      printf '%s\n' "--values" "deploy/values/mongodb/values.yaml"
      ;;
    deploy/charts/mssql)
      printf '%s\n' "--values" "deploy/values/mssql/local.yaml"
      ;;
    deploy/charts/postgresql-cluster)
      if [[ -s deploy/values/postgresql/values.yaml ]]; then
        printf '%s\n' "--values" "deploy/values/postgresql/values.yaml"
      fi
      ;;
    deploy/charts/apicurio-registry)
      if [[ -s deploy/values/apicurio/local.yaml ]]; then
        printf '%s\n' "--values" "deploy/values/apicurio/local.yaml"
      fi
      ;;
  esac
}

# ... existing code ...

echo "==> Helmfile lint"
DBZ_VERSION="${DBZ_VERSION}" \
DBZ_ENV="${DBZ_ENV}" \
DBZ_DOMAIN="${DBZ_DOMAIN}" \
DBZ_NAMESPACE="${DBZ_NAMESPACE}" \
DBZ_IMAGE_TAG="${DBZ_IMAGE_TAG}" \
DBZ_IMAGE_TAG_CONDUCTOR="${DBZ_IMAGE_TAG_CONDUCTOR}" \
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
      echo "Skipping DBZ_ENV=${env}: missing ${file}"
      return 1
    fi
  done < <(helmfile_env_required_files "${env}")

  return 0
}

template_helmfile_env() {
  local env="$1"

  echo "Rendering Helmfile with DBZ_ENV=${env}"

  DBZ_VERSION="${DBZ_VERSION}" \
  DBZ_ENV="${env}" \
  DBZ_DOMAIN="${DBZ_DOMAIN}" \
  DBZ_NAMESPACE="${DBZ_NAMESPACE}" \
  DBZ_IMAGE_TAG="${DBZ_IMAGE_TAG}" \
  DBZ_IMAGE_TAG_CONDUCTOR="${DBZ_IMAGE_TAG_CONDUCTOR}" \
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