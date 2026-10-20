#!/bin/sh
# Stable entry point. The release installer pins the publisher verification key.
# Keep the complete function before its invocation so an interrupted download
# cannot execute a partial body.
darkpawns_install() (
    set -eu
    installer=$(mktemp "${TMPDIR:-/tmp}/darkpawns-install.XXXXXXXX")
    trap 'rm -f "$installer"' EXIT
    trap 'exit 1' HUP INT TERM
    curl -fsSL --proto '=https' --proto-redir '=https' \
        https://github.com/zax0rz/darkpawns/releases/latest/download/install.sh \
        -o "$installer"
    sh "$installer" "$@"
)
darkpawns_install "$@"
