#!/usr/bin/env bash
set -eu
# Compile storms are separate from scenario permits; one host build at a time.
go_bin=${ORACLE_REGRESSION_GO:-/usr/local/go/bin/go}
command=("$go_bin")
# Go's global -C must precede the build subcommand and all build flags.
if [[ ${1:-} == -C ]]; then
 command+=(-C "$2")
 shift 2
fi
command+=(build -p 4 "$@")
if [[ -n "${CENSUS_RUN_DIR:-}" ]]; then
 exec python3 "$(dirname -- "${BASH_SOURCE[0]}")/census_budget.py" exec build -- "${command[@]}"
fi
exec "${command[@]}"
