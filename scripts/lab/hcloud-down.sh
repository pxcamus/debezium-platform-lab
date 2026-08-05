#!/usr/bin/env bash
# Destroy the lab VM and its firewall.
#
# MAINTAINER TOOLING. See hcloud-up.sh for the full description.
#
#   scripts/lab/hcloud-down.sh        # list what will be deleted, then confirm
#   DPL_LAB_YES=true scripts/lab/hcloud-down.sh
#
# SAFETY: this only ever acts on resources carrying the label below, and it takes no
# server name as an argument. If your Hetzner project holds anything you care about, it
# cannot be reached from here — the label is set by hcloud-up.sh and by nothing else.
set -euo pipefail

readonly LAB_LABEL="lab=dbz-platform"

DPL_LAB_YES="${DPL_LAB_YES:-false}"

readonly STATE_DIR="${XDG_CACHE_HOME:-${HOME}/.cache}/dbz-lab"

die() {
  echo "error: $*" >&2
  exit 1
}

preflight() {
  command -v hcloud >/dev/null || die "hcloud CLI not found — https://github.com/hetznercloud/cli"

  if [[ -z "${HCLOUD_TOKEN:-}" ]] && [[ -z "$(hcloud context active 2>/dev/null)" ]]; then
    die "no Hetzner credentials — export HCLOUD_TOKEN, or run 'hcloud context create dbz-lab'"
  fi
}

# `hcloud <resource> list --selector` filters by label server-side, so nothing unlabelled
# can ever appear in these lists.
labelled() {
  hcloud "$1" list --selector "${LAB_LABEL}" -o noheader -o columns=name 2>/dev/null || true
}

main() {
  preflight

  local servers firewalls
  servers="$(labelled server)"
  firewalls="$(labelled firewall)"

  if [[ -z "${servers}" && -z "${firewalls}" ]]; then
    echo "Nothing labelled ${LAB_LABEL} in this project — nothing to delete."
    exit 0
  fi

  echo
  echo "  These resources are labelled ${LAB_LABEL} and will be DELETED:"
  echo
  # Written as `if` rather than `[[ … ]] && …`: under `set -e` a false test as the last
  # command of an AND-list exits the script, which would skip the firewall listing whenever
  # no server is present.
  if [[ -n "${servers}" ]]; then
    echo "${servers}" | sed 's/^/    server    /'
  fi
  if [[ -n "${firewalls}" ]]; then
    echo "${firewalls}" | sed 's/^/    firewall  /'
  fi
  echo

  if [[ "${DPL_LAB_YES}" != "true" ]]; then
    read -r -p "  Delete them? [y/N] " answer
    [[ "${answer}" == "y" || "${answer}" == "Y" ]] || die "aborted"
  fi

  # Servers first: a firewall still attached to a server cannot be deleted.
  local name
  while read -r name; do
    [[ -n "${name}" ]] || continue
    echo "Deleting server ${name}"
    hcloud server delete "${name}" >/dev/null
    rm -f "${STATE_DIR}/${name}.env"
  done <<<"${servers}"

  while read -r name; do
    [[ -n "${name}" ]] || continue
    echo "Deleting firewall ${name}"
    hcloud firewall delete "${name}" >/dev/null
  done <<<"${firewalls}"

  echo "Done. Nothing labelled ${LAB_LABEL} is still billing."
}

main "$@"
