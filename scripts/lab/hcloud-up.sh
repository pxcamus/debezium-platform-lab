#!/usr/bin/env bash
# Create a disposable Hetzner Cloud VM for testing the operator experience on a machine
# that has never seen this project.
#
# MAINTAINER TOOLING — not part of the deployment path. Nobody deploying the platform
# needs this. It exists so "does a fresh machine work?" can be answered repeatedly, which
# is impossible on a machine you have already fixed by hand.
#
#   scripts/lab/hcloud-up.sh          # create, print how to connect
#   scripts/lab/hcloud-down.sh        # destroy everything it made
#
# THIS CREATES BILLABLE RESOURCES and they bill until you run hcloud-down.sh.
#
# Requires the hcloud CLI, authenticated either by exporting HCLOUD_TOKEN or by an active
# `hcloud context`. No credential is read from, or written to, this repository.
#
# Configuration (all optional except DPL_LAB_SSH_KEY):
#   DPL_LAB_SSH_KEY    name of an SSH key already uploaded to your Hetzner project (required)
#   DPL_LAB_IDENTITY   local private key to connect with (default: ssh's own defaults)
#   DPL_LAB_NAME       server name                       (default: dbz-lab)
#   DPL_LAB_TYPE       server type                       (default: cpx32 — 4 vCPU, 8 GB, amd64)
#   DPL_LAB_IMAGE      OS image                          (default: ubuntu-24.04)
#   DPL_LAB_LOCATION   datacenter                        (default: nbg1)
#   DPL_LAB_OPEN_HTTP  expose 80/443 to the internet     (default: false — your IP only)
#   DPL_LAB_YES        skip the confirmation prompt      (default: false)
#
# amd64 is deliberate: the Debezium Platform release images are amd64-only (see the
# README's known gaps), so an arm64 box would fail for reasons unrelated to what is
# being tested.
set -euo pipefail

# Every resource carries this label and hcloud-down.sh only ever acts on it. Not
# configurable — it is the guard that stops the teardown touching anything else in your
# project.
readonly LAB_LABEL="lab=dbz-platform"

DPL_LAB_NAME="${DPL_LAB_NAME:-dbz-lab}"
DPL_LAB_TYPE="${DPL_LAB_TYPE:-cpx32}"
DPL_LAB_IMAGE="${DPL_LAB_IMAGE:-ubuntu-24.04}"
DPL_LAB_LOCATION="${DPL_LAB_LOCATION:-nbg1}"
DPL_LAB_OPEN_HTTP="${DPL_LAB_OPEN_HTTP:-false}"
DPL_LAB_YES="${DPL_LAB_YES:-false}"

# ssh only offers the default identities (~/.ssh/id_*) plus whatever the agent holds. A lab
# key created under its own name is therefore never offered, ssh falls through to password
# auth, and cloud-init sets lock_passwd — so the wait below would time out on a machine that
# is in fact perfectly healthy. Set DPL_LAB_IDENTITY to the matching private key.
DPL_LAB_IDENTITY="${DPL_LAB_IDENTITY:-}"
ssh_identity_args=()
if [[ -n "${DPL_LAB_IDENTITY}" ]]; then
  ssh_identity_args=(-i "${DPL_LAB_IDENTITY}" -o IdentitiesOnly=yes)
fi

readonly FIREWALL_NAME="${DPL_LAB_NAME}-fw"
readonly STATE_DIR="${XDG_CACHE_HOME:-${HOME}/.cache}/dbz-lab"
readonly STATE_FILE="${STATE_DIR}/${DPL_LAB_NAME}.env"
readonly CLOUD_INIT="$(dirname "$0")/cloud-init.yaml"

die() {
  echo "error: $*" >&2
  exit 1
}

preflight() {
  command -v hcloud >/dev/null || die "hcloud CLI not found — https://github.com/hetznercloud/cli"
  command -v curl >/dev/null || die "curl not found"
  command -v ssh >/dev/null || die "ssh not found"

  # Either an exported token or an active context works. Never printed, never stored.
  if [[ -z "${HCLOUD_TOKEN:-}" ]] && [[ -z "$(hcloud context active 2>/dev/null)" ]]; then
    die "no Hetzner credentials — export HCLOUD_TOKEN, or run 'hcloud context create dbz-lab'"
  fi

  [[ -n "${DPL_LAB_SSH_KEY:-}" ]] || die "DPL_LAB_SSH_KEY must name an SSH key in your Hetzner project ('hcloud ssh-key list')"
  hcloud ssh-key describe "${DPL_LAB_SSH_KEY}" >/dev/null 2>&1 ||
    die "SSH key '${DPL_LAB_SSH_KEY}' not found in this project — see 'hcloud ssh-key list'"

  if [[ -n "${DPL_LAB_IDENTITY}" ]]; then
    [[ -f "${DPL_LAB_IDENTITY}" ]] || die "DPL_LAB_IDENTITY points at '${DPL_LAB_IDENTITY}', which does not exist"
  fi

  [[ -f "${CLOUD_INIT}" ]] || die "cloud-init file not found at ${CLOUD_INIT}"

  if hcloud server describe "${DPL_LAB_NAME}" >/dev/null 2>&1; then
    die "server '${DPL_LAB_NAME}' already exists — run scripts/lab/hcloud-down.sh first, or set DPL_LAB_NAME"
  fi

  check_server_type
}

# Hetzner retires server types per location: cpx31 is still *supported* in nbg1 but no
# longer *available* there (it moved to US locations only), and the API says so only after
# the firewall has been created and the confirmation answered. Catch it before spending
# anything.
#
# Best-effort by design: if the listing cannot be read, the check is skipped rather than
# blocking. A brittle parser that hard-fails would be worse than the error it prevents.
check_server_type() {
  local listing line locations

  listing="$(hcloud server-type list 2>/dev/null || true)"
  [[ -n "${listing}" ]] || return 0

  line="$(awk -v type="${DPL_LAB_TYPE}" '$2 == type' <<<"${listing}")"
  [[ -n "${line}" ]] || die "unknown server type '${DPL_LAB_TYPE}' — see 'hcloud server-type list'"

  if [[ "${line}" == *" arm "* ]]; then
    echo "warning: ${DPL_LAB_TYPE} is arm64 and the Debezium Platform release images are amd64-only." >&2
  fi

  # The locations are the last column, after the memory and disk sizes: strip everything
  # up to and including the final " GB ".
  locations="${line##* GB }"
  locations="${locations// /}"

  if [[ ",${locations}," != *",${DPL_LAB_LOCATION},"* ]]; then
    die "server type '${DPL_LAB_TYPE}' is not available in '${DPL_LAB_LOCATION}'.
       Available in: ${locations//,/, }
       Pick another location with DPL_LAB_LOCATION, or another type with DPL_LAB_TYPE ('hcloud server-type list')."
  fi
}

# The firewall is scoped to whoever is running this, resolved at runtime so no address is
# ever committed. If your IP changes, tear down and recreate.
my_ip() {
  local ip
  ip="$(curl -fsS --max-time 10 https://ifconfig.me 2>/dev/null || true)"
  [[ -n "${ip}" ]] || ip="$(curl -fsS --max-time 10 https://api.ipify.org 2>/dev/null || true)"
  [[ "${ip}" =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]] || die "could not determine your public IPv4 address"
  echo "${ip}"
}

confirm() {
  local ip="$1"

  cat <<EOF

  This creates billable Hetzner Cloud resources:

    server     ${DPL_LAB_NAME}  (${DPL_LAB_TYPE}, ${DPL_LAB_IMAGE}, ${DPL_LAB_LOCATION})
    firewall   ${FIREWALL_NAME}
    ssh + k8s API reachable from ${ip}/32 only
    http/https $(if [[ "${DPL_LAB_OPEN_HTTP}" == "true" ]]; then echo "OPEN TO THE INTERNET"; else echo "${ip}/32 only"; fi)

  They bill by the hour until you run scripts/lab/hcloud-down.sh.

EOF

  if [[ "${DPL_LAB_YES}" == "true" ]]; then
    return 0
  fi

  read -r -p "  Create them? [y/N] " answer
  [[ "${answer}" == "y" || "${answer}" == "Y" ]] || die "aborted"
}

create_firewall() {
  local ip="$1"
  local http_source="${ip}/32"

  if [[ "${DPL_LAB_OPEN_HTTP}" == "true" ]]; then
    http_source="0.0.0.0/0"
  fi

  # Recreated from scratch every time rather than reconciled — it only exists between an
  # up and a down, so a stale one is a leftover from a failed run and is safe to replace.
  if hcloud firewall describe "${FIREWALL_NAME}" >/dev/null 2>&1; then
    echo "Removing stale firewall ${FIREWALL_NAME}"
    hcloud firewall delete "${FIREWALL_NAME}"
  fi

  echo "Creating firewall ${FIREWALL_NAME} (ssh + k8s API from ${ip}/32, http from ${http_source})"
  hcloud firewall create --name "${FIREWALL_NAME}" --label "${LAB_LABEL}" >/dev/null

  hcloud firewall add-rule "${FIREWALL_NAME}" --direction in --protocol tcp --port 22 \
    --source-ips "${ip}/32" --description "ssh" >/dev/null
  hcloud firewall add-rule "${FIREWALL_NAME}" --direction in --protocol tcp --port 6443 \
    --source-ips "${ip}/32" --description "kubernetes api" >/dev/null
  hcloud firewall add-rule "${FIREWALL_NAME}" --direction in --protocol tcp --port 80 \
    --source-ips "${http_source}" --description "http ingress" >/dev/null
  hcloud firewall add-rule "${FIREWALL_NAME}" --direction in --protocol tcp --port 443 \
    --source-ips "${http_source}" --description "https ingress" >/dev/null
}

create_server() {
  echo "Creating server ${DPL_LAB_NAME} (${DPL_LAB_TYPE}, ${DPL_LAB_IMAGE}, ${DPL_LAB_LOCATION})"
  hcloud server create \
    --name "${DPL_LAB_NAME}" \
    --type "${DPL_LAB_TYPE}" \
    --image "${DPL_LAB_IMAGE}" \
    --location "${DPL_LAB_LOCATION}" \
    --ssh-key "${DPL_LAB_SSH_KEY}" \
    --firewall "${FIREWALL_NAME}" \
    --label "${LAB_LABEL}" \
    --user-data-from-file "${CLOUD_INIT}" >/dev/null
}

# cloud-init creates the lab user after first boot, so the server being "running" is not
# the same as the account existing. Poll until it can actually be logged into.
wait_for_ssh() {
  local ip="$1"
  local deadline=$((SECONDS + 300))

  echo -n "Waiting for SSH as lab@${ip} "
  while ((SECONDS < deadline)); do
    if ssh "${ssh_identity_args[@]}" -o BatchMode=yes -o StrictHostKeyChecking=accept-new \
      -o ConnectTimeout=5 -o UserKnownHostsFile=/dev/null -o LogLevel=ERROR \
      "lab@${ip}" true 2>/dev/null; then
      echo " ready"
      return 0
    fi
    echo -n "."
    sleep 5
  done

  echo
  die "timed out waiting for SSH — the server exists at ${ip}.
     The usual cause is ssh not offering the right key rather than a broken machine: this
     waits with BatchMode, so a key ssh does not offer looks identical to a machine that
     never came up. Set DPL_LAB_IDENTITY to the private key matching '${DPL_LAB_SSH_KEY}' and
     re-run, or connect by hand to see the real error:

       ssh -v${DPL_LAB_IDENTITY:+ -i ${DPL_LAB_IDENTITY}} lab@${ip}

     If root works but lab does not, cloud-init has not finished — see
     'hcloud server describe ${DPL_LAB_NAME}'."
}

write_state() {
  local ip="$1"

  # Deliberately outside the repository: this holds a live host address and must never be
  # committable by accident.
  mkdir -p "${STATE_DIR}"
  cat >"${STATE_FILE}" <<EOF
DPL_LAB_NAME=${DPL_LAB_NAME}
DPL_LAB_IP=${ip}
DPL_LAB_SSH=lab@${ip}
DPL_LAB_IDENTITY=${DPL_LAB_IDENTITY}
DPL_LAB_DOMAIN=${ip//./-}.sslip.io
EOF
  echo "Wrote ${STATE_FILE}"
}

main() {
  preflight

  local ip
  ip="$(my_ip)"

  confirm "${ip}"
  create_firewall "${ip}"
  create_server

  local server_ip
  server_ip="$(hcloud server ip "${DPL_LAB_NAME}")"

  wait_for_ssh "${server_ip}"
  write_state "${server_ip}"

  cat <<EOF

  Ready.

    ssh${DPL_LAB_IDENTITY:+ -i ${DPL_LAB_IDENTITY}} lab@${server_ip}

  The machine is intentionally bare — no docker, no kubectl, no helm. That is the point:
  note what it is missing before you install anything, because a bootstrapped machine can
  no longer tell you.

  Bootstrap it with:

    ssh${DPL_LAB_IDENTITY:+ -i ${DPL_LAB_IDENTITY}} lab@${server_ip} 'bash -s' < scripts/lab/bootstrap.sh

  Ingress hostnames without touching /etc/hosts:

    DPL_DOMAIN=${server_ip//./-}.sslip.io

  Destroy it when you are done — it bills until then:

    scripts/lab/hcloud-down.sh

EOF
}

main "$@"
