#!/bin/sh
# deploy-site guard: refuse to deploy a verifying install stub before the
# public latest release actually serves install.sh.sig. Sequencing is
# release-first (v0.1.4+ uploads the .sig assets), site-second; deploying the
# stub early would break the public one-command install with a 404 on the
# signature. No-op while the stub does not verify signatures.
set -eu
for asset in install.sh install.ps1; do
    stub=$(dirname "$0")/../website/static/$asset
    url=https://github.com/zax0rz/darkpawns/releases/latest/download/$asset.sig
    if [ ! -f "$stub" ] || ! grep -q "$asset\.sig" "$stub"; then
        echo "deploy-site: $asset stub does not verify signatures; no release check needed."
        continue
    fi
    if curl -fsSIL --max-time 30 "$url" >/dev/null 2>&1; then
        echo "deploy-site: latest release serves $asset.sig."
        continue
    fi
    echo "deploy-site: $stub verifies $asset.sig, but the latest release does not serve it:" >&2
    echo "  $url" >&2
    echo "  Publish the release (with its .sig assets) before deploying the site." >&2
    exit 1
done
