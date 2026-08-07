#!/usr/bin/env bash
# Run any command with the project env loaded.
#
# deploy/environment/versions.env supplies the shared pins and .env overrides
# them. Raw helmfile / kubectl / skopeo read neither, so they render the
# committed defaults instead of your host overrides. Front them with this:
#
#   scripts/with-env.sh helmfile --file deploy/helmfile.yaml.gotmpl \
#     --selector app=debezium-platform template
#   scripts/with-env.sh kubectl get pods -n "${DPL_NAMESPACE}"
#
# Run from the repo root (paths below are repo-relative, matching helmfile).
set -euo pipefail

# shellcheck source=scripts/lib/env.sh
source "$(dirname "${BASH_SOURCE[0]}")/lib/env.sh"

if [[ $# -eq 0 ]]; then
  echo "usage: scripts/with-env.sh <command> [args...]" >&2
  exit 64
fi

exec "$@"
