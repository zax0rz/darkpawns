#!/usr/bin/env bash
# Deterministic tests for census.sh wait's run identity (RO-014 follow-up):
# every report line names its run and the go-head it was captured against,
# and --name makes wait refuse to report a different run.
set -euo pipefail

repo_root=$(CDPATH= cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
census="$repo_root/scripts/census.sh"

tmp=$(mktemp -d "${TMPDIR:-/tmp}/dp-census-wait-test.XXXXXX")
trap 'rm -rf -- "$tmp"' EXIT

runs_root="$tmp/runs/2026-10-10"
run_dir="$runs_root/glm-fake-run"
mkdir -p -- "$run_dir"

# A finished run: go-head + summary with a verdict.
printf 'ca54e397ed79ebc5700ac6166223198bc04ed55a\n' >"$run_dir/go-head.txt"
printf 'oracle-combined: pairs=1 deduplicated=1 full=CLEAN claims=CLEAN elapsed=1.0s verdict=CLEAN\n' >"$run_dir/summary.txt"

# Point the resolver at the fake run through a private runtime dir so the
# test never touches the host's real census lock or last-run pointer.
runtime="$tmp/runtime"
mkdir -p -- "$runtime"
printf '1: %s\n' "$run_dir" >"$runtime/dp-census.lock"

pass_count=0

expect_exit() {
	local want=$1
	shift
	local got=0
	output=$(env ORACLE_RUNS_ROOT="$tmp/runs" XDG_RUNTIME_DIR="$runtime" "$@" 2>&1) || got=$?
	if [[ "$got" != "$want" ]]; then
		printf 'FAIL: exit %d, want %d: %s\n' "$got" "$want" "$output" >&2
		exit 1
	fi
	pass_count=$((pass_count + 1))
}

expect_out() {
	local want=$1
	if [[ "$output" != *"$want"* ]]; then
		printf 'FAIL: output missing %q: %s\n' "$want" "$output" >&2
		exit 1
	fi
	pass_count=$((pass_count + 1))
}

# A finished clean run names itself and its go-head on the one line.
expect_exit 0 "$census" wait --max-seconds 1
expect_out 'glm-fake-run @ca54e397e: oracle-combined:'

# The matching --name reports normally.
expect_exit 0 "$census" wait --name glm-fake-run --max-seconds 1
expect_out 'verdict=CLEAN'

# A different --name is refused with no verdict line (RO-014).
status=0
output=$(env ORACLE_RUNS_ROOT="$tmp/runs" XDG_RUNTIME_DIR="$runtime" \
	"$census" wait --name someone-elses-run --max-seconds=1 2>&1) || status=$?
if [[ "$status" != 4 ]]; then
	printf 'FAIL: refusal exit %d, want 4: %s\n' "$status" "$output" >&2
	exit 1
fi
if [[ "$output" != *"refusing: latest census is glm-fake-run, not someone-elses-run"* ]]; then
	printf 'FAIL: refusal message wrong: %s\n' "$output" >&2
	exit 1
fi
if [[ "$output" == *verdict=* ]]; then
	printf 'FAIL: refusal leaked a verdict line: %s\n' "$output" >&2
	exit 1
fi
pass_count=$((pass_count + 1))

# A running run carries the same identity prefix on its progress line.
printf 'running pid=%d started=%s\n' $$ "$(date '+%Y-%m-%d %H:%M:%S')" >"$run_dir/state"
printf 'running pid=%d started=%s\n' $$ "$(date '+%Y-%m-%d %H:%M:%S')" >"$run_dir/state.tmp"
rm -f -- "$run_dir/summary.txt"
printf '1\n' >"$run_dir/total"
expect_exit 3 "$census" wait --name glm-fake-run --max-seconds 0
expect_out 'glm-fake-run @ca54e397e: running'

printf 'PASS: %d census wait identity assertions\n' "$pass_count"
