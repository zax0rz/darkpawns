#!/usr/bin/env bash
# Stable entrypoint; registration, queueing and run-bound reports live in census.py.
set -eu
exec python3 "$(dirname -- "${BASH_SOURCE[0]}")/census.py" "$@"
