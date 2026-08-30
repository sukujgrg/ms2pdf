#!/bin/sh
# Install ms2pdf from GitHub Releases for this OS and CPU.
# Usage:
#   curl -fsSL https://github.com/sukujgrg/ms2pdf/releases/latest/download/install.sh | sh
#   curl -fsSL https://github.com/sukujgrg/ms2pdf/releases/latest/download/install.sh | sh -s -- v0.1.0
set -eu

REPO="sukujgrg/ms2pdf"
BIN="ms2pdf"
INSTALL_DIR="${MS2PDF_INSTALL_DIR:-${HOME}/.local/bin}"

die() {
	printf '%s\n' "$*" >&2
	exit 1
}

need_cmd() {
	command -v "$1" >/dev/null 2>&1 || die "need $1"
}

os=$(uname -s | tr '[:upper:]' '[:lower:]')
arch=$(uname -m)
case "$arch" in
x86_64 | amd64) arch=amd64 ;;
arm64 | aarch64) arch=arm64 ;;
*) die "unsupported architecture: $(uname -m)" ;;
esac
case "$os" in
darwin | linux) ;;
msys* | mingw* | cygwin*)
	die "on Windows use: irm https://github.com/${REPO}/releases/latest/download/install.ps1 | iex"
	;;
*) die "unsupported OS: $(uname -s)" ;;
esac

need_cmd curl
need_cmd tar
need_cmd mktemp
need_cmd mkdir
need_cmd install

version="${1:-${MS2PDF_VERSION:-latest}}"
if [ "$version" = "latest" ]; then
	loc=$(curl -fsSI "https://github.com/${REPO}/releases/latest" | tr -d '\r' | awk 'tolower($1) == "location:" { print $2; exit }')
	[ -n "$loc" ] || die "could not resolve latest release"
	version=${loc##*/}
fi
case "$version" in
v*) ;;
*) version="v${version}" ;;
esac

asset="${BIN}_${version}_${os}_${arch}.tar.gz"
base="https://github.com/${REPO}/releases/download/${version}"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

curl -fsSL -o "${tmp}/${asset}" "${base}/${asset}"
curl -fsSL -o "${tmp}/SHA256SUMS" "${base}/SHA256SUMS"

if command -v sha256sum >/dev/null 2>&1; then
	(cd "$tmp" && grep -F " ${asset}" SHA256SUMS | sha256sum -c -)
elif command -v shasum >/dev/null 2>&1; then
	(cd "$tmp" && grep -F " ${asset}" SHA256SUMS | shasum -a 256 -c -)
else
	die "need sha256sum or shasum"
fi

tar -xzf "${tmp}/${asset}" -C "$tmp"
[ -f "${tmp}/${BIN}" ] || die "archive did not contain ${BIN}"

mkdir -p "$INSTALL_DIR"
install -m 0755 "${tmp}/${BIN}" "${INSTALL_DIR}/${BIN}"
printf 'installed %s (%s)\n' "${INSTALL_DIR}/${BIN}" "$version"

case ":${PATH}:" in
*":${INSTALL_DIR}:"*) ;;
*)
	printf 'add %s to PATH, for example:\n  export PATH="%s:$PATH"\n' "$INSTALL_DIR" "$INSTALL_DIR"
	;;
esac
