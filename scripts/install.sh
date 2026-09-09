#!/usr/bin/env sh
set -eu

repo="kuraudo-lab/teleskope"
install_dir="${INSTALL_DIR:-"$HOME/.local/bin"}"
curl_args="--http1.1 --retry 3 --retry-delay 2"

case "$(uname -s)" in
  Linux) os="linux" ;;
  Darwin) os="darwin" ;;
  *) echo "Unsupported operating system: $(uname -s)" >&2; exit 1 ;;
esac

case "$(uname -m)" in
  x86_64 | amd64) arch="amd64" ;;
  arm64 | aarch64) arch="arm64" ;;
  *) echo "Unsupported architecture: $(uname -m)" >&2; exit 1 ;;
esac

latest_url=$(curl $curl_args -fsSLI -o /dev/null -w '%{url_effective}' "https://github.com/$repo/releases/latest")
tag="${latest_url##*/}"
version="${tag#v}"
archive="teleskope_${version}_${os}_${arch}.tar.gz"
base_url="https://github.com/$repo/releases/download/$tag"
tmpdir=$(mktemp -d "${TMPDIR:-/tmp}/teleskope-install.XXXXXX")

cleanup() {
  rm -rf "$tmpdir"
}
trap cleanup EXIT INT TERM

curl $curl_args -fsSLo "$tmpdir/$archive" "$base_url/$archive"
curl $curl_args -fsSLo "$tmpdir/checksums.txt" "$base_url/checksums.txt"

expected=$(awk -v name="$archive" '$2 == name { print $1 }' "$tmpdir/checksums.txt")
if [ -z "$expected" ]; then
  echo "Checksum entry not found for $archive" >&2
  exit 1
fi

if command -v sha256sum >/dev/null 2>&1; then
  actual=$(sha256sum "$tmpdir/$archive" | awk '{ print $1 }')
elif command -v shasum >/dev/null 2>&1; then
  actual=$(shasum -a 256 "$tmpdir/$archive" | awk '{ print $1 }')
else
  echo "sha256sum or shasum is required to verify $archive" >&2
  exit 1
fi

if [ "$actual" != "$expected" ]; then
  echo "Checksum mismatch for $archive" >&2
  exit 1
fi

tar -xzf "$tmpdir/$archive" -C "$tmpdir" teleskope
mkdir -p "$install_dir"
install "$tmpdir/teleskope" "$install_dir/teleskope"

echo "Installed teleskope $version to $install_dir/teleskope"
echo "Run: teleskope --version"
