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

# --- helmfile ----------------------------------------------------------------

# Run helmfile against deploy/helmfile.yaml.gotmpl with the project env loaded.
[group('helmfile')]
hf *args:
    #!/usr/bin/env bash
    set -euo pipefail
    source scripts/lib/env.sh

    helmfile --file deploy/helmfile.yaml.gotmpl {{ args }}

# Install or upgrade every enabled release. Safe to re-run.
[group('helmfile')]
apply *args:
    #!/usr/bin/env bash
    set -euo pipefail

    # --skip-diff-on-install is unconditional on purpose. helm-diff renders a release
    # against the live cluster, so a chart carrying custom resources whose CRDs arrive
    # in the same run (OpenTelemetryCollector, ServiceMonitor) can never diff on a
    # fresh cluster — the first apply fails with "ensure CRDs are installed first".
    #
    # Nothing is lost by always passing it: the flag only skips the diff for releases
    # being installed for the FIRST time, where the diff says "all of this is new"
    # anyway. Releases that already exist still diff normally, which is where a diff
    # earns its keep. Run `just hf template` when you want to inspect a first install.
    just hf apply --skip-diff-on-install {{ args }}

# List the releases this helmfile installs (declarative — does not query the cluster).
[group('helmfile')]
releases:
    #!/usr/bin/env bash
    set -euo pipefail

    # Reads the `installed:` field out of the rendered state file, so this answers
    # "what would I get" rather than "what is running".
    #
    # awk on the table output, not jq on --output json: awk is everywhere, jq is an
    # extra install. The header check means a future helmfile column reshuffle fails
    # loudly instead of silently printing nothing.
    just hf list | awk -F'\t' '
      NR == 1 {
        if ($4 !~ /^INSTALLED/) {
          print "helmfile list columns changed: expected INSTALLED in column 4" > "/dev/stderr"
          exit 1
        }
        next
      }
      $4 ~ /^true/ { gsub(/[[:space:]]+$/, "", $1); print $1 }
    '
