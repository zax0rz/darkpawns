#!/usr/bin/env bash
# Stable entrypoint; registration, queueing and run-bound reports live in census.py.
set -eu
# Keep census binaries on the same JSON implementation as Make and releases.
export GOEXPERIMENT=nojsonv2
exec python3 "$(dirname -- "${BASH_SOURCE[0]}")/census.py" "$@"
