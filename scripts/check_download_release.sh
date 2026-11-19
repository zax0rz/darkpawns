#!/bin/sh
# deploy-site guard: every GitHub release link the built download page carries
# must resolve. The page takes its version from the newest
# docs/releases/vX.Y.Z.md, and those notes land on main before the release is
# published, so a site deployed in between would link to assets that 404.
# Sequencing stays release-first, site-second.
set -eu
root=$(dirname "$0")/..
pages="$root/website-astro/dist/download/index.html $root/website-astro/dist/download.md"
for page in $pages; do
    if [ ! -f "$page" ]; then
        echo "deploy-site: $page is missing; build the site first." >&2
        exit 1
    fi
done
urls=$(grep -ohE 'https://github\.com/zax0rz/darkpawns/releases/(tag|download)/v[0-9]+\.[0-9]+\.[0-9]+[^"<>)[:space:]]*' $pages | sort -u)
if [ -z "$urls" ]; then
    echo "deploy-site: the download page has no versioned release links." >&2
    exit 1
fi
versions=$(printf '%s\n' "$urls" | sed -E 's#.*/(tag|download)/(v[0-9]+\.[0-9]+\.[0-9]+).*#\2#' | sort -u)
if [ "$(printf '%s\n' "$versions" | wc -l)" -ne 1 ]; then
    echo "deploy-site: the download page links more than one release:" >&2
    printf '  %s\n' $versions >&2
    exit 1
fi
failed=0
for url in $urls; do
    if ! curl -fsSIL --max-time 30 "$url" >/dev/null 2>&1; then
        echo "deploy-site: release link does not resolve: $url" >&2
        failed=1
    fi
done
if [ "$failed" -ne 0 ]; then
    echo "  Publish release $versions before deploying the site." >&2
    exit 1
fi
echo "deploy-site: download page links release $versions; all $(printf '%s\n' "$urls" | wc -l) links resolve."
