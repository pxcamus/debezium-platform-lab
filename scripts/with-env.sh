#!/usr/bin/env bash
# Run any command with the project env loaded.
#
# deploy/environment/versions.env supplies the shared pins and .env overrides
# them. Raw helmfile / kubectl / skopeo read neither, so they render the
# committed defaults instead of your host overrides. Front them with this:
#
#   scripts/with-env.sh helmfile --file deploy/helmfile.yaml.gotmpl \
#     --selector app=debezium-platform template
#   scripts/with-env.sh kubectl get pods -n "${DBZ_NAMESPACE}"
#
# Run from the repo root (paths below are repo-relative, matching helmfile).
set -euo pipefail

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

if [[ $# -eq 0 ]]; then
  echo "usage: scripts/with-env.sh <command> [args...]" >&2
  exit 64
fi

exec "$@"
