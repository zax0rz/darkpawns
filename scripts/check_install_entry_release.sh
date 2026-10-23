#!/bin/sh
# deploy-site guard: refuse to deploy a verifying install stub before the
# public latest release actually serves install.sh.sig. Sequencing is
# release-first (v0.1.4+ uploads the .sig assets), site-second; deploying the
# stub early would break the public one-command install with a 404 on the
# signature. No-op while the stub does not verify signatures.
set -eu
stub=$(dirname "$0")/../website/static/install.sh
url=https://github.com/zax0rz/darkpawns/releases/latest/download/install.sh.sig
if ! grep -q 'install\.sh\.sig' "$stub"; then
    echo "deploy-site: install stub does not verify signatures; no release check needed."
    exit 0
fi
if curl -fsSIL --max-time 30 "$url" >/dev/null 2>&1; then
    echo "deploy-site: latest release serves install.sh.sig."
    exit 0
fi
echo "deploy-site: $stub verifies install.sh.sig, but the latest release does not serve it:" >&2
echo "  $url" >&2
echo "  Publish the release (with its .sig assets) before deploying the site." >&2
exit 1
