set shell := ["bash", "-euo", "pipefail", "-c"]

# Recipes run from the repo root, so paths below are repo-relative (matching helmfile).
#
# Env is NOT loaded via `set dotenv-load`: that reads a single file, which would give
# us .env only and silently drop the pins in deploy/environment/versions.env. Recipes
# source scripts/lib/env.sh instead, which is the same loader scripts/with-env.sh uses.

_default:
    @just --list

# --- cluster -----------------------------------------------------------------

# Delete and re-create the local Kind cluster, then point kubectl at it.
[group('cluster')]
kind-recreate:
    #!/usr/bin/env bash
    set -euo pipefail
    source scripts/lib/env.sh

    name="${DPL_KIND_CLUSTER_NAME:-dmp}"
    config="${DPL_KIND_CONFIG:-deploy/clusters/kind/kind-ingress.yaml}"

    echo "==> recreating kind cluster '${name}' from ${config}"
    kind delete cluster --name "${name}"
    kind create cluster --name "${name}" --config "${config}"
    kubectl config use-context "kind-${name}"

# Recreate whatever DPL_CLUSTER_TYPE points at (kind | k3s). Destructive.
[group('cluster')]
cluster-recreate:
    #!/usr/bin/env bash
    set -euo pipefail
    source scripts/lib/env.sh

    case "${DPL_CLUSTER_TYPE:-kind}" in
      kind)
        just kind-recreate
        ;;
      k3s)
        echo "k3s recreate is not migrated yet — the Go version targeted AWS" >&2
        echo "(ec2-user, context k3s-aws) and the lab is now Hetzner." >&2
        exit 1
        ;;
      *)
        echo "unsupported DPL_CLUSTER_TYPE '${DPL_CLUSTER_TYPE}'" >&2
        exit 1
        ;;
    esac
