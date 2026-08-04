#!/bin/sh
#
# Install dmp-lab.
#
#   curl -fsSL https://raw.githubusercontent.com/pxcamus/debezium-platform-lab/main/install.sh | sh
#
# Environment:
#   DMP_LAB_VERSION       tag to install, e.g. v0.2.0 (default: the latest release)
#   DMP_LAB_INSTALL_DIR   where to put the binary (default: ~/.local/bin)
#
# Written for /bin/sh, and using only curl, tar and a sha256 tool. dmp-lab exists to report
# what a machine is missing, so its installer cannot assume the machine has anything: no Go,
# no jq, no bash.

set -eu

REPO="pxcamus/debezium-platform-lab"
BINARY="dmp-lab"
VERSION="${DMP_LAB_VERSION:-latest}"
INSTALL_DIR="${DMP_LAB_INSTALL_DIR:-${HOME}/.local/bin}"

fail() {
	echo "error: $*" >&2
	exit 1
}

need() {
	command -v "$1" >/dev/null 2>&1 || fail "$1 is required but was not found"
}

need curl
need tar

os="$(uname -s)"
case "${os}" in
Linux) os="linux" ;;
Darwin) os="darwin" ;;
*) fail "unsupported operating system: ${os}. Linux and macOS have released binaries; on anything else, build from source with 'go build ./cmd/dmp-lab'." ;;
esac

arch="$(uname -m)"
case "${arch}" in
x86_64 | amd64) arch="amd64" ;;
aarch64 | arm64) arch="arm64" ;;
*) fail "unsupported architecture: ${arch}" ;;
esac

archive="${BINARY}_${os}_${arch}.tar.gz"

# The /releases/latest/download/ path redirects to the newest release's asset, which keeps
# this script off the GitHub API and its unauthenticated rate limit — the one thing most
# likely to make a piped installer fail for reasons the user cannot act on.
if [ "${VERSION}" = "latest" ]; then
	base="https://github.com/${REPO}/releases/latest/download"
else
	base="https://github.com/${REPO}/releases/download/${VERSION}"
fi

tmp="$(mktemp -d)"
trap 'rm -rf "${tmp}"' EXIT INT TERM

echo "Downloading ${archive} (${VERSION})..."
curl -fsSL "${base}/${archive}" -o "${tmp}/${archive}" ||
	fail "could not download ${base}/${archive}"
curl -fsSL "${base}/checksums.txt" -o "${tmp}/checksums.txt" ||
	fail "could not download ${base}/checksums.txt"

# Verify before extracting, not after. An archive that fails its checksum should never have
# been unpacked anywhere, however briefly.
if command -v sha256sum >/dev/null 2>&1; then
	sha256="sha256sum"
elif command -v shasum >/dev/null 2>&1; then
	sha256="shasum -a 256"
else
	fail "no sha256 tool found (looked for sha256sum and shasum)"
fi

expected="$(grep " ${archive}\$" "${tmp}/checksums.txt" | awk '{print $1}')"
[ -n "${expected}" ] || fail "${archive} is not listed in checksums.txt"

actual="$(${sha256} "${tmp}/${archive}" | awk '{print $1}')"
[ "${expected}" = "${actual}" ] || fail "checksum mismatch for ${archive}
  expected ${expected}
  actual   ${actual}"

tar -xzf "${tmp}/${archive}" -C "${tmp}" "${BINARY}"

mkdir -p "${INSTALL_DIR}"
# Install by rename, so a running or in-use copy is replaced atomically rather than being
# truncated partway through a write.
mv "${tmp}/${BINARY}" "${INSTALL_DIR}/${BINARY}.new"
chmod +x "${INSTALL_DIR}/${BINARY}.new"
mv "${INSTALL_DIR}/${BINARY}.new" "${INSTALL_DIR}/${BINARY}"

echo "Installed ${INSTALL_DIR}/${BINARY}"

case ":${PATH}:" in
*":${INSTALL_DIR}:"*)
	echo
	echo "Run 'dmp-lab doctor' to check this machine."
	;;
*)
	echo
	echo "${INSTALL_DIR} is not on your PATH. Either add it:"
	echo "    export PATH=\"${INSTALL_DIR}:\${PATH}\""
	echo "or run it directly:"
	echo "    ${INSTALL_DIR}/${BINARY} doctor"
	;;
esac
