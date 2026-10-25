#!/usr/bin/env bash
# Real-driver fixtures exercise registration, all dispatch modes, and recovery.
set -eu
exec python3 "$(dirname -- "${BASH_SOURCE[0]}")/test_census_budget.py" "$@"
