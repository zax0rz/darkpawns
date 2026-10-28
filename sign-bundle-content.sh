#!/bin/bash
# Stage (and sign) one platform's bundle content for the portable archives.
# Called from the goreleaser builds post-hook with the platform as arguments:
#   sign-bundle-content.sh <goos> <goarch>
#
# The archive hook point does not exist in GoReleaser OSS, so the publisher
# record is signed over the bundle CONTENTS here — both executables plus the
# staged release-assets manifest — into dist/signed-bundles/<os>_<arch>/,
# which the archive file lists then include per platform. Everything GoReleaser
# computes afterwards (checksums, Homebrew, Scoop, winget, Docker) therefore
# sees the exact signed bytes; nothing is repacked after the fact.
#
# Key selection: DP_RELEASE_KEY_PATH is the production signing key (a path,
# written by the protected publish job); SNAPSHOT_PRIVATE_PEM/SNAPSHOT_PUBLIC_PEM
# is the disposable CI pair; with neither, the tree is staged unsigned and
# dev snapshots keep today's behavior.
set -euo pipefail

os="$1"
arch="$2"
root="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
staging="$root/release-assets"
out="$root/dist/signed-bundles/${os}_${arch}"

# Both executables of this platform must exist; builds may complete in any
# order, so earlier calls wait for the pair rather than signing half a bundle.
# The archive stage starts only after every build finished, so the last hook
# call per platform always finds both.
bins=()
for name in darkpawns darkpawns-server; do
  suffix=""
  [ "$os" = windows ] && suffix=".exe"
  candidates=( "$root"/dist/*_${os}_${arch}*/${name}${suffix} )
  [ -e "${candidates[0]}" ] || { echo "sign-bundle-content: waiting for ${name} (${os}/${arch})" >&2; exit 0; }
  bins+=( "${candidates[0]}" )
done

rm -rf "$out"
mkdir -p "$out"
cp -r "$staging" "$out/release-assets"
install -m 0755 "${bins[0]}" "$out/darkpawns"
install -m 0755 "${bins[1]}" "$out/darkpawns-server"

if [ -n "${DP_RELEASE_KEY_PATH:-}" ]; then
  # Production: dp-release verifies the private key matches the public key
  # pinned inside the launcher before signing anything.
  go run ./cmd/dp-release --sign-bundle "$out" --key "$DP_RELEASE_KEY_PATH"
elif [ -n "${SNAPSHOT_PRIVATE_PEM:-}" ] && [ -n "${SNAPSHOT_PUBLIC_PEM:-}" ]; then
  go run ./cmd/dp-release --sign-bundle "$out" --key "$SNAPSHOT_PRIVATE_PEM" --public "$SNAPSHOT_PUBLIC_PEM"
else
  echo "sign-bundle-content: no signing key configured; staging ${os}_${arch} unsigned (dev snapshot)" >&2
fi
echo "sign-bundle-content: staged ${os}_${arch}" >&2
