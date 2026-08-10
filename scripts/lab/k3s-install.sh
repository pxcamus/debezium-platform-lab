#!/usr/bin/env bash
# Install k3s on a fresh Linux machine, configured for DPL_ENV=public.
#
# Not Debian-specific — unlike bootstrap.sh, which shells out to apt-get, this needs only
# curl and systemd, so Fedora, RHEL and openSUSE work too. The k3s installer pulls the
# k3s-selinux policy itself when SELinux is enforcing; firewalld it leaves alone, which is
# checked for below because the resulting failure names nothing useful.
#
# THROWAWAY TOOLING — the same contract as the rest of scripts/lab. Nothing in the
# deployment path calls this, and no `just` recipe depends on it. It exists so a demo host
# can be rebuilt from nothing in one command and destroyed as soon as it has served its
# purpose. Treat the machine, its address and its DNS records as disposable.
#
# Runs ON the target machine. Pipe it over ssh from the repository root:
#
#   ssh lab@<ip> 'bash -s' < scripts/lab/k3s-install.sh
#
# You do NOT need bootstrap.sh on this host. helmfile runs from your workstation against
# the kubeconfig fetched at the end; bootstrap.sh installs the *workstation* toolchain and
# a Kind binary that a k3s host has no use for. Run it here only if you intend to drive the
# deployment from the box itself.
#
# Configuration (all optional):
#   DPL_K3S_CHANNEL  release channel                        (default: stable)
#   DPL_K3S_VERSION  exact version, overrides the channel   (default: unset)
#                    e.g. v1.33.4+k3s1 — pin this if a post has to stay reproducible
#   DPL_K3S_TLS_SAN  extra API server SANs, comma-separated (default: detected public IPv4)
#
# Two deliberate non-choices, both of which most k3s guides get wrong for this repo:
#
# 1. Traefik is LEFT INSTALLED. deploy/helmfile.yaml.gotmpl gates ingress-nginx to
#    DPL_ENV=local because Kind ships no controller; every public values file carries
#    `className: traefik` (zot, oauth2-proxy, apicurio, the keycloak chart). Passing
#    `--disable traefik` here leaves all of them unclaimed and nothing answers on 443.
#
# 2. --write-kubeconfig-mode is NOT set. Guides reach for 0644 so a non-root user can copy
#    the file, which makes cluster-admin credentials world-readable to every process on the
#    box. The fetch printed at the end uses `sudo cat` instead, so k3s keeps its 0600.
set -euo pipefail

DPL_K3S_CHANNEL="${DPL_K3S_CHANNEL:-stable}"
DPL_K3S_VERSION="${DPL_K3S_VERSION:-}"
DPL_K3S_TLS_SAN="${DPL_K3S_TLS_SAN:-}"

# Empty when already root, so the script works both as the cloud-init `lab` user and on a
# stock Hetzner image where you land as root.
SUDO=""

die() {
  echo "error: $*" >&2
  exit 1
}

step() {
  echo
  echo "==> $*"
}

preflight() {
  [[ "$(uname -s)" == "Linux" ]] ||
    die "this installs a k3s server; run it on the target machine, not your workstation"
  command -v curl >/dev/null || die "curl not found"
  command -v systemctl >/dev/null || die "systemd required — the k3s installer ships a unit"

  if [[ "${EUID}" -ne 0 ]]; then
    sudo -n true 2>/dev/null || die "passwordless sudo required (or run as root)"
    SUDO="sudo"
  fi

  # Re-running cannot change --tls-san on an existing install: the server certificate is
  # generated once and the flags live in the systemd unit. Fail rather than appear to work.
  if command -v k3s >/dev/null; then
    die "k3s is already installed — uninstall it first:
       sudo /usr/local/bin/k3s-uninstall.sh"
  fi
}

# The API server certificate otherwise covers only 127.0.0.1 and the private address, so a
# kubeconfig copied to your laptop fails TLS verification and you end up reaching for
# --insecure-skip-tls-verify. Resolved at runtime; no address is committed anywhere.
public_ip() {
  local ip
  ip="$(curl -fsS --max-time 10 https://ifconfig.me 2>/dev/null || true)"
  [[ -n "${ip}" ]] || ip="$(curl -fsS --max-time 10 https://api.ipify.org 2>/dev/null || true)"
  [[ "${ip}" =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]] ||
    die "could not determine this machine's public IPv4 address — set DPL_K3S_TLS_SAN"
  echo "${ip}"
}

# RPM-based distributions ship firewalld enabled. It drops pod-to-pod and pod-to-service
# traffic, and the symptom is never "firewall": DNS lookups inside the cluster time out,
# readiness probes fail, and pods restart in a loop. Warned rather than stopped, because on
# a host that is not disposable the right fix may well be the CIDR rules rather than
# switching the firewall off.
check_firewalld() {
  systemctl is-active --quiet firewalld 2>/dev/null || return 0

  cat >&2 <<'EOF'

warning: firewalld is running and will break cluster networking

         On a disposable host:

           sudo systemctl disable --now firewalld

         Otherwise open the default k3s CIDRs (see the k3s documentation for the
         per-distribution rules):

           sudo firewall-cmd --permanent --add-source=10.42.0.0/16
           sudo firewall-cmd --permanent --add-source=10.43.0.0/16
           sudo firewall-cmd --reload

EOF
}

# Not fixed silently, following bootstrap.sh: raising these here would hide a requirement
# that belongs in the documentation. The failure mode is worth recognising though — pods
# stay Pending or controllers restart in a loop, with nothing in the events naming inotify.
check_inotify() {
  local instances watches

  instances="$(sysctl -n fs.inotify.max_user_instances 2>/dev/null || echo 0)"
  watches="$(sysctl -n fs.inotify.max_user_watches 2>/dev/null || echo 0)"

  if ((instances < 512)) || ((watches < 524288)); then
    cat >&2 <<EOF

warning: inotify limits are low for a cluster this size
         fs.inotify.max_user_instances = ${instances}  (want >= 512)
         fs.inotify.max_user_watches   = ${watches}  (want >= 524288)

         Raise them before deploying, or containerd and the operators will fail in ways
         that never mention inotify:

           sudo tee /etc/sysctl.d/99-inotify.conf >/dev/null <<'SYSCTL'
           fs.inotify.max_user_instances = 512
           fs.inotify.max_user_watches = 524288
SYSCTL
           sudo sysctl --system

EOF
  fi
}

install_k3s() {
  local ip="$1"

  # The detected address is always a SAN and DPL_K3S_TLS_SAN adds to it rather than
  # replacing it, so naming a hostname cannot cost you access by IP.
  local sans="${ip}${DPL_K3S_TLS_SAN:+,${DPL_K3S_TLS_SAN}}"
  local exec_args="" san

  # One --tls-san per entry; the installer passes the string to the unit verbatim.
  while IFS= read -r san; do
    if [[ -n "${san}" ]]; then
      exec_args+=" --tls-san ${san}"
    fi
  done <<<"${sans//,/$'\n'}"
  exec_args="${exec_args# }"

  if [[ -n "${DPL_K3S_VERSION}" ]]; then
    step "k3s ${DPL_K3S_VERSION} (SANs: ${sans})"
    curl -sfL https://get.k3s.io |
      INSTALL_K3S_VERSION="${DPL_K3S_VERSION}" \
        INSTALL_K3S_EXEC="${exec_args}" sh -
  else
    step "k3s from the ${DPL_K3S_CHANNEL} channel (SANs: ${sans})"
    curl -sfL https://get.k3s.io |
      INSTALL_K3S_CHANNEL="${DPL_K3S_CHANNEL}" \
        INSTALL_K3S_EXEC="${exec_args}" sh -
  fi
}

wait_for_node() {
  step "Waiting for the node to report Ready"
  ${SUDO} k3s kubectl wait --for=condition=Ready node --all --timeout=180s ||
    die "node did not become Ready — check 'sudo journalctl -u k3s -n 100'"

  # Traefik arrives as a HelmChart CR reconciled after the node is up, so it lags the wait
  # above. Its absence is the difference between a working 443 and a connection refused.
  step "Waiting for the bundled Traefik"
  ${SUDO} k3s kubectl -n kube-system rollout status deploy/traefik --timeout=180s ||
    echo "warning: Traefik is not ready yet — 'kubectl -n kube-system get helmchart' to see why" >&2
}

summary() {
  local ip="$1"

  cat <<EOF

  k3s is up: $(${SUDO} k3s kubectl version -o yaml 2>/dev/null | awk '/gitVersion/ {print $2; exit}')

  Fetch the kubeconfig FROM YOUR WORKSTATION. The server field on the node says
  127.0.0.1, so it is rewritten in flight — piping through sed avoids the sed -i
  incompatibility between macOS and Linux, and \`sudo cat\` avoids weakening the file mode:

    ssh <user>@${ip} 'sudo cat /etc/rancher/k3s/k3s.yaml' \\
      | sed 's#127.0.0.1#${ip}#' > ~/.kube/dmp-demo.yaml
    chmod 600 ~/.kube/dmp-demo.yaml
    kubectl --kubeconfig ~/.kube/dmp-demo.yaml config rename-context default dmp-demo

  Then, in the repository root .env:

    DPL_K3S_HOST=${ip}
    DPL_K3S_LOCAL_KUBECONFIG=~/.kube/dmp-demo.yaml
    DPL_K3S_CONTEXT=dmp-demo

  DNS — both records, because the wildcard does not cover the apex it hangs off:

    <DPL_DOMAIN>      A   ${ip}
    *.<DPL_DOMAIN>    A   ${ip}

  This address is ephemeral by design. Rebuilding the box changes it and both records
  have to follow. While iterating, point the issuer at Let's Encrypt staging — set
  acme.server in deploy/values/dns01-gandi/public.yaml.gotmpl — because production
  allows only 5 identical certificate requests per week and a few rebuilds will spend
  them. Switch to production for the run you actually screenshot.

  Tear the machine down when the post is out; nothing here is meant to outlive it.

EOF
}

main() {
  preflight

  local ip
  ip="$(public_ip)"

  # Before installing, so it is visible above the installer's own output rather than
  # buried under it.
  check_firewalld

  install_k3s "${ip}"
  wait_for_node
  check_inotify
  summary "${ip}"
}

main "$@"
