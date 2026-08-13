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

# Copy the kubeconfig off an existing k3s node and name its context locally.
[group('cluster')]
k3s-kubeconfig:
    #!/usr/bin/env bash
    set -euo pipefail
    source scripts/lib/env.sh

    # Reads one file, writes one file, never touches the cluster. The DPL_LAB_* are
    # fallbacks for the Hetzner path only — hcloud-up.sh records both in its state
    # file, so an explicit DPL_K3S_* always wins. docs/guides/k3s.md owns the details.
    host="${DPL_K3S_HOST:-${DPL_LAB_IP:-}}"
    key="${DPL_K3S_SSH_KEY:-${DPL_LAB_IDENTITY:-}}"
    user="${DPL_K3S_SSH_USER:-root}"
    remote_kc="${DPL_K3S_REMOTE_KUBECONFIG:-/etc/rancher/k3s/k3s.yaml}"
    context="${DPL_K3S_CONTEXT:-dmp-demo}"
    local_kc="${DPL_K3S_LOCAL_KUBECONFIG:-${HOME}/.kube/${context}.yaml}"

    if [[ -z "${host}" ]]; then
      echo "DPL_K3S_HOST is required — see docs/guides/k3s.md#point-kubectl-at-it" >&2
      exit 1
    fi

    # A quoted value in .env is never tilde-expanded on source, and an unexpanded ~
    # yields a literal ~ directory in the repo rather than an error.
    local_kc="${local_kc/#\~/$HOME}"

    # IdentitiesOnly: ssh otherwise offers its defaults plus the agent, and a key kept
    # under its own name is never tried.
    ssh_opts=(-o ConnectTimeout=15)
    if [[ -n "${key}" ]]; then
      ssh_opts+=(-i "${key/#\~/$HOME}" -o IdentitiesOnly=yes)
    fi

    # A root image may not have sudo installed, and does not need it.
    if [[ "${user}" == "root" ]]; then remote_sudo=""; else remote_sudo="sudo "; fi

    echo "==> ${user}@${host}:${remote_kc} -> ${local_kc}"
    mkdir -p "$(dirname "${local_kc}")"

    # k3s writes 127.0.0.1 as the server address, so rewrite it in flight. Staged through
    # a temp file so a failed fetch cannot truncate a kubeconfig that was working.
    ssh -n "${ssh_opts[@]}" "${user}@${host}" "${remote_sudo}cat ${remote_kc}" \
      | sed "s#127.0.0.1#${host}#" > "${local_kc}.tmp"
    mv "${local_kc}.tmp" "${local_kc}"
    chmod 600 "${local_kc}"

    kubectl --kubeconfig "${local_kc}" config rename-context default "${context}" >/dev/null
    kubectl --kubeconfig "${local_kc}" get nodes

    echo
    echo "    export KUBECONFIG=${local_kc}"

# Recreate whatever DPL_CLUSTER_TYPE points at. Destructive. Kind only.
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
        # Deliberately not implemented. A Kind cluster is disposable by construction; a
        # k3s cluster is a machine you brought, and the first act of a recreate would be
        # to uninstall it. Not this recipe's call to make.
        echo "cluster-recreate handles kind only — a k3s cluster is a machine you own." >&2
        echo "Already running k3s: just k3s-kubeconfig. Otherwise: docs/guides/k3s.md" >&2
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
releases *args:
    #!/usr/bin/env bash
    set -euo pipefail

    # Reads the `installed:` field out of the rendered state file, so this answers
    # "what would I get" rather than "what is running".
    #
    # awk on the table output, not jq on --output json: awk is everywhere, jq is an
    # extra install. The header check means a future helmfile column reshuffle fails
    # loudly instead of silently printing nothing.
    just hf list {{ args }} | awk -F'\t' '
      NR == 1 {
        if ($4 !~ /^INSTALLED/) {
          print "helmfile list columns changed: expected INSTALLED in column 4" > "/dev/stderr"
          exit 1
        }
        next
      }
      $4 ~ /^true/ { gsub(/[[:space:]]+$/, "", $1); print $1 }
    '

[group('helmfile')]
platform-sso *args:
    just apply --state-values-file {{ justfile_directory() }}/examples/k3s/platform-sso/components.yaml {{ args }}

[group('helmfile')]
platform-sso-releases:
    just releases --state-values-file {{ justfile_directory() }}/examples/k3s/platform-sso/components.yaml
