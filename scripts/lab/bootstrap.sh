#!/usr/bin/env bash
# Install the prerequisites this project needs, on a fresh Debian/Ubuntu machine.
#
# Provider-agnostic: this is the half of the lab tooling that matters. It runs on a Hetzner
# VM, a spare laptop, an EC2 instance or bare metal — hcloud-up.sh only exists to hand it a
# clean machine.
#
#   scp scripts/lab/bootstrap.sh lab@<host>:
#   ssh lab@<host> ./bootstrap.sh
#
# A fresh machine is worth something: it is the only honest test of what the project
# actually requires. Note what a bare box is missing before you run this — that list is the
# real prerequisite list, and a machine you bootstrapped immediately can no longer tell you
# anything.
#
# This is scaffolding, and should shrink as the toolchain gets a task runner that can
# install its own prerequisites.
#
# Deliberately NOT done here: raising fs.inotify limits. Fixing that silently would hide a
# requirement that belongs in the documentation.
set -euo pipefail

# Pinned to match .github/workflows/helm-validation.yaml, so the lab machine and CI
# validate charts with the same toolchain.
DPL_HELM_VERSION="${DPL_HELM_VERSION:-v4.2.3}"
DPL_HELMFILE_VERSION="${DPL_HELMFILE_VERSION:-1.7.1}"

# The justfile uses [group(...)], which needs just >= 1.27 — older packaged versions
# (including Ubuntu's) fail to parse it, so this is pinned rather than taken from apt.
DPL_JUST_VERSION="${DPL_JUST_VERSION:-1.58.0}"

die() {
  echo "error: $*" >&2
  exit 1
}

step() {
  echo
  echo "==> $*"
}

arch() {
  case "$(uname -m)" in
    x86_64) echo amd64 ;;
    aarch64 | arm64) echo arm64 ;;
    *) die "unsupported architecture: $(uname -m)" ;;
  esac
}

preflight() {
  [[ "$(uname -s)" == "Linux" ]] || die "this script targets Debian/Ubuntu; run it on the lab machine, not your workstation"
  command -v apt-get >/dev/null || die "no apt-get — this script targets Debian/Ubuntu"
  [[ "${EUID}" -ne 0 ]] || die "run as a normal user with sudo, not as root (the docker group matters)"
  sudo -n true 2>/dev/null || die "passwordless sudo required"

  if [[ "$(arch)" == "arm64" ]]; then
    echo "warning: the Debezium Platform release images are amd64-only; expect image pull failures." >&2
  fi
}

install_base() {
  step "Base packages"
  sudo apt-get update -qq
  sudo DEBIAN_FRONTEND=noninteractive apt-get install -y -qq curl ca-certificates git tar
}

install_docker() {
  if command -v docker >/dev/null; then
    echo "docker already installed"
    return
  fi

  step "Docker"
  curl -fsSL https://get.docker.com | sudo sh
  sudo usermod -aG docker "${USER}"
  echo "note: log out and back in before docker works without sudo"
}

install_kubectl() {
  if command -v kubectl >/dev/null; then
    echo "kubectl already installed"
    return
  fi

  step "kubectl"
  local version
  version="$(curl -fsSL https://dl.k8s.io/release/stable.txt)"
  curl -fsSLo /tmp/kubectl "https://dl.k8s.io/release/${version}/bin/linux/$(arch)/kubectl"
  sudo install -m 0755 /tmp/kubectl /usr/local/bin/kubectl
  rm -f /tmp/kubectl
}

install_helm() {
  if command -v helm >/dev/null; then
    echo "helm already installed"
    return
  fi

  step "Helm ${DPL_HELM_VERSION}"
  curl -fsSL https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 |
    sudo DESIRED_VERSION="${DPL_HELM_VERSION}" bash

  # helm-diff is not optional: `just apply` runs a diff for every release that already
  # exists, so a workstation without it works exactly once and then fails with
  # `unknown command "diff" for "helm"` on the second run.
  #
  # Helm 4 verifies plugins on install and helm-diff publishes no provenance, so the
  # plain command fails there; --verify=false is not a flag Helm 3 has, hence the
  # fallback. Failures are NOT swallowed — the previous version discarded the output and
  # reported the plugin as "already present or unavailable", which is how a bootstrapped
  # machine ends up without it and is told nothing is wrong.
  if helm plugin list 2>/dev/null | grep -q '^diff[[:space:]]'; then
    echo "helm-diff already installed"
  else
    helm plugin install https://github.com/databus23/helm-diff --verify=false ||
      helm plugin install https://github.com/databus23/helm-diff ||
      die "helm-diff install failed; \`just apply\` cannot upgrade an existing release without it"
  fi
}

install_helmfile() {
  if command -v helmfile >/dev/null; then
    echo "helmfile already installed"
    return
  fi

  step "helmfile ${DPL_HELMFILE_VERSION}"
  # Separate declaration: `local url=$(arch)` would mask a failing arch().
  local url
  url="https://github.com/helmfile/helmfile/releases/download/v${DPL_HELMFILE_VERSION}/helmfile_${DPL_HELMFILE_VERSION}_linux_$(arch).tar.gz"
  curl -fsSL "${url}" | tar -xz -C /tmp helmfile
  sudo install -m 0755 /tmp/helmfile /usr/local/bin/helmfile
  rm -f /tmp/helmfile
}

install_kind() {
  if command -v kind >/dev/null; then
    echo "kind already installed"
    return
  fi

  step "kind"
  curl -fsSLo /tmp/kind "https://github.com/kubernetes-sigs/kind/releases/latest/download/kind-linux-$(arch)"
  sudo install -m 0755 /tmp/kind /usr/local/bin/kind
  rm -f /tmp/kind
}

install_just() {
  if command -v just >/dev/null; then
    echo "just already installed"
    return
  fi

  step "just ${DPL_JUST_VERSION}"
  # just names its release assets by target triple, not the amd64/arm64 that arch() returns.
  local target
  case "$(uname -m)" in
    x86_64) target=x86_64-unknown-linux-musl ;;
    aarch64 | arm64) target=aarch64-unknown-linux-musl ;;
    *) die "unsupported architecture for just: $(uname -m)" ;;
  esac

  local url="https://github.com/casey/just/releases/download/${DPL_JUST_VERSION}/just-${DPL_JUST_VERSION}-${target}.tar.gz"
  curl -fsSL "${url}" | tar -xz -C /tmp just
  sudo install -m 0755 /tmp/just /usr/local/bin/just
  rm -f /tmp/just
}

summary() {
  cat <<EOF

  Installed:

    docker    $(docker --version 2>/dev/null || echo "not on PATH until you re-login")
    kubectl   $(kubectl version --client -o yaml 2>/dev/null | awk '/gitVersion/ {print $2; exit}')
    helm      $(helm version --short 2>/dev/null)
    helmfile  $(helmfile --version 2>/dev/null)
    kind      $(kind --version 2>/dev/null)
    just      $(just --version 2>/dev/null)

  Log out and back in so the docker group and PATH take effect, then:

    git clone https://github.com/pxcamus/debezium-platform-lab.git
    cd debezium-platform-lab

EOF
}

main() {
  preflight
  install_base
  install_docker
  install_kubectl
  install_helm
  install_helmfile
  install_kind
  install_just
  summary
}

main "$@"
