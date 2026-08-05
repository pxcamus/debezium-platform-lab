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
HELM_VERSION="${HELM_VERSION:-v4.2.3}"
HELMFILE_VERSION="${HELMFILE_VERSION:-1.7.1}"

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

  step "Helm ${HELM_VERSION}"
  curl -fsSL https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 |
    sudo DESIRED_VERSION="${HELM_VERSION}" bash

  # helm-diff is what makes `helmfile diff` useful; helmfile prompts for it otherwise.
  helm plugin install https://github.com/databus23/helm-diff >/dev/null 2>&1 ||
    echo "note: helm-diff plugin already present or unavailable"
}

install_helmfile() {
  if command -v helmfile >/dev/null; then
    echo "helmfile already installed"
    return
  fi

  step "helmfile ${HELMFILE_VERSION}"
  local url="https://github.com/helmfile/helmfile/releases/download/v${HELMFILE_VERSION}/helmfile_${HELMFILE_VERSION}_linux_$(arch).tar.gz"
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

summary() {
  cat <<EOF

  Installed:

    docker    $(docker --version 2>/dev/null || echo "not on PATH until you re-login")
    kubectl   $(kubectl version --client -o yaml 2>/dev/null | awk '/gitVersion/ {print $2; exit}')
    helm      $(helm version --short 2>/dev/null)
    helmfile  $(helmfile --version 2>/dev/null)
    kind      $(kind --version 2>/dev/null)

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
  summary
}

main "$@"
