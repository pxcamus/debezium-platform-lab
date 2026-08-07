# shellcheck shell=bash
# Sourceable project env loader — sets variables, runs nothing.
#
#   source scripts/lib/env.sh
#
# Use this from justfile recipes and any script that needs the env in its own
# shell, including an interactive bash or zsh session. To front a single command
# instead, use scripts/with-env.sh, which is a thin wrapper around this file.
#
# Run from the repo root (paths below are repo-relative, matching helmfile).

load_env_file() {
  local file="$1"
  [[ -f "${file}" ]] || return 0
  set -a
  # shellcheck disable=SC1090
  source "${file}"
  set +a
}

# versions.env first (shared, non-sensitive fallback), .env last so it wins —
# shell last-wins matches godotenv's first-wins-with-.env-loaded-first.
load_env_file "deploy/environment/versions.env"
load_env_file ".env"

# Transition guard: every variable moved to a single DPL_ prefix. A legacy name
# left exported in your shell is now silently inert, which is exactly the failure
# this warns about. Delete this block once the rename has settled.
for _dpl_legacy in \
  DBZ_ENV DBZ_DOMAIN DBZ_VERSION DBZ_NAMESPACE DBZ_IMAGE_TAG DBZ_OPERATOR_TAG \
  CLUSTER_TYPE KIND_CLUSTER_NAME KIND_CONFIG \
  DMP_RESOURCE_PREFIX DMP_ENVIRONMENT DMP_BASE_URL \
  LAB_SSH_KEY LAB_NAME LAB_TYPE LAB_IMAGE LAB_LOCATION LAB_IDENTITY \
  K3S_HOST K3S_SSH_KEY K3S_SSH_USER K3S_CONTEXT \
  MONGODB_URI POSTGRESQL_HOST LOG_LEVEL; do
  # eval, not ${!name}: indirect expansion is bash-only and this file is meant to
  # be sourceable from an interactive zsh too.
  eval "_dpl_value=\${${_dpl_legacy}:-}"
  if [ -n "${_dpl_value}" ]; then
    echo "warning: ${_dpl_legacy} is set but no longer read — variables are now DPL_*" >&2
  fi
done
unset _dpl_legacy _dpl_value
