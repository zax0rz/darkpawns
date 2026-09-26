#!/usr/bin/env bash
# oracle_regression.sh — full C-vs-Go scenario corpus.
#
# Usage: oracle_regression.sh [--workers N]
#
# --workers N (or ORACLE_REGRESSION_JOBS=N) sets how many scenarios run at once.
# Every worker is a separate dp-oracle-diff process on its own free ports, its
# own throwaway runtime directory, its own disposable C lib copy and its own
# Go world copy + database; scripts/oracle_regression_isolation.sh proves that
# holds before the fan-out is raised.
set -uo pipefail

repo_root=$(CDPATH= cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
go_bin=${ORACLE_REGRESSION_GO:-/usr/local/go/bin/go}
oracle_bin=${DP_ORACLE_BIN:-/home/zach/darkpawns-c-oracle/bin/circle}
scenario_timeout=${ORACLE_REGRESSION_TIMEOUT:-240s}
seed=${ORACLE_REGRESSION_SEED:-1}
jobs=${ORACLE_REGRESSION_JOBS:-4}

while (($# > 0)); do
	case $1 in
	--workers)
		if (($# < 2)); then
			printf 'oracle-regression: --workers needs a count\n' >&2
			exit 2
		fi
		jobs=$2
		shift 2
		;;
	--workers=*)
		jobs=${1#--workers=}
		shift
		;;
	-h | --help)
		printf 'usage: %s [--workers N]\n' "$(basename "$0")"
		exit 0
		;;
	*)
		printf 'oracle-regression: unknown argument: %s\n' "$1" >&2
		exit 2
		;;
	esac
done

if [[ ! -x "$go_bin" ]]; then
	printf 'oracle-regression: go binary is not executable: %s\n' "$go_bin" >&2
	exit 2
fi
if [[ ! -x "$oracle_bin" ]]; then
	printf 'oracle-regression: C oracle is not executable: %s\n' "$oracle_bin" >&2
	exit 2
fi
if ! command -v timeout >/dev/null 2>&1; then
	printf 'oracle-regression: timeout(1) is required\n' >&2
	exit 2
fi

mapfile -t scenarios < <(find "$repo_root/cmd/dp-oracle-diff/scenarios" -maxdepth 1 -type f -name '*.txt' -printf '%f\n' | sort)
if [[ -n "${ORACLE_REGRESSION_SCENARIOS:-}" ]]; then
	IFS=',' read -r -a requested_scenarios <<<"$ORACLE_REGRESSION_SCENARIOS"
	scenarios=()
	for scenario in "${requested_scenarios[@]}"; do
		if [[ ! -f "$repo_root/cmd/dp-oracle-diff/scenarios/$scenario.txt" ]]; then
			printf 'oracle-regression: scenario not found: %s\n' "$scenario" >&2
			exit 2
		fi
		scenarios+=("$scenario.txt")
	done
fi
if (( ${#scenarios[@]} == 0 )); then
	printf 'oracle-regression: no scenarios found\n' >&2
	exit 2
fi
if [[ ! "$jobs" =~ ^[1-9][0-9]*$ ]]; then
	printf 'oracle-regression: ORACLE_REGRESSION_JOBS must be a positive integer: %s\n' "$jobs" >&2
	exit 2
fi

script_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

run_started=$(date +%s)
run_started_ns=$(date +%s%N)
log_dir=$(mktemp -d "${TMPDIR:-/tmp}/dp-oracle-regression.XXXXXX")
result_dir="$log_dir/results"
mkdir -p -- "$result_dir"
cleanup() {
	rm -rf -- "$log_dir"
}
trap cleanup EXIT

# Build the harness and the server ONCE per corpus run. Per-scenario builds
# recompiled the server 900+ times per run — roughly half the wall time and
# repeated compile-storm memory spikes under the 4-way worker fan-out.
harness_bin="$log_dir/dp-oracle-diff"
server_bin="$log_dir/dp-server-prebuilt"
if ! "$go_bin" build -C "$repo_root" -o "$harness_bin" ./cmd/dp-oracle-diff; then
	printf 'oracle-regression: harness build failed\n' >&2
	exit 2
fi
if ! "$go_bin" build -C "$repo_root" -o "$server_bin" ./cmd/server; then
	printf 'oracle-regression: server prebuild failed\n' >&2
	exit 2
fi
export ORACLE_REGRESSION_SERVER="$server_bin"

export repo_root go_bin oracle_bin scenario_timeout seed log_dir result_dir harness_bin server_bin
# Ledger-backed expected-divergence baseline (see scripts/gen_expected_divergences.py).
# Entries cite blocked/excluded manifest rows; divergence without a row is FAIL,
# an entry that stops diverging is STALE. Never minted from observed behavior.
export EXPECTED_DIVERGENCES_FILE="$repo_root/cmd/dp-oracle-diff/expected_divergences.tsv"
export EXPECTED_DIVERGENCE_PINS_FILE="$repo_root/cmd/dp-oracle-diff/expected_divergence_pins.tsv"

# Optional coverage dump: when ORACLE_REGRESSION_DUMP names a directory, every
# scenario leaves its normalized C blocks behind (<dir>/<scenario>.txt) for
# cmd/dp-census-coverage. Default off, so results and timing are unchanged. Run
# `make census-coverage` against the directory afterwards.
if [[ -n "${ORACLE_REGRESSION_DUMP:-}" ]]; then
	mkdir -p -- "$ORACLE_REGRESSION_DUMP" || {
		printf 'oracle-regression: cannot create dump directory: %s\n' "$ORACLE_REGRESSION_DUMP" >&2
		exit 2
	}
	export ORACLE_REGRESSION_DUMP
fi

printf 'oracle-regression: %d scenarios, seed=%s, timeout=%s, jobs=%s%s\n' "${#scenarios[@]}" "$seed" "$scenario_timeout" "$jobs" "${ORACLE_REGRESSION_DUMP:+ dump=$ORACLE_REGRESSION_DUMP}"
printf '%s\0' "${scenarios[@]}" | xargs -0 -n1 -P "$jobs" "$script_dir"/oracle_regression_worker.sh

passed=0
expected=0
unstable=0
unpinnable=0
stale=0
failed=0
infra=0
timed_out=0
for scenario_file in "${scenarios[@]}"; do
	scenario=${scenario_file%.txt}
	result_file="$result_dir/$scenario"
	if [[ ! -s "$result_file" ]]; then
		failed=$((failed + 1))
		printf 'FAIL %s (no result; worker scheduling failure)\n' "$scenario" >&2
		continue
	fi
	case $(cut -f1 "$result_file") in
	PASS)
		passed=$((passed + 1))
		;;
	EXPECTED)
		expected=$((expected + 1))
		;;
	EXPECTED_UNSTABLE)
		# Ledger-backed divergence whose manifest row declares
		# stability=run-varying (e.g. the C-side accuse pointer anomaly).
		# Green, but surfaced so the standing roster stays visible.
		unstable=$((unstable + 1))
		printf 'EXPECTED_UNSTABLE %s (ledger-backed, declared run-varying)\n' "$scenario"
		;;
	UNPINNABLE)
		unpinnable=$((unpinnable + 1))
		printf 'UNPINNABLE %s (requires human clearance)\n' "$scenario" >&2
		;;
	STALE)
		stale=$((stale + 1))
		printf 'STALE %s (baseline expects divergence but scenario passed; reconcile ledger)\n' "$scenario" >&2
		;;
	TIMEOUT)
		timed_out=$((timed_out + 1))
		;;
	INFRA)
		infra=$((infra + 1))
		;;
	FAIL)
		failed=$((failed + 1))
		;;
	esac
done

run_finished=$(date +%s)
run_finished_ns=$(date +%s%N)
elapsed_ns=$((run_finished_ns - run_started_ns))
elapsed_seconds=$((elapsed_ns / 1000000000))
elapsed_remainder=$(( (elapsed_ns % 1000000000) / 1000000 ))

printf 'oracle-regression: scenarios=%d passed=%d expected=%d unpinnable=%d stale=%d failed=%d infra=%d timed_out=%d unstable=%d elapsed=%d.%03ds started=%s finished=%s\n' \
	"${#scenarios[@]}" "$passed" "$expected" "$unpinnable" "$stale" "$failed" "$infra" "$timed_out" "$unstable" "$elapsed_seconds" "$elapsed_remainder" \
	"$(date -d "@$run_started" '+%Y-%m-%dT%H:%M:%S%z')" \
	"$(date -d "@$run_finished" '+%Y-%m-%dT%H:%M:%S%z')"

if (( failed != 0 )); then
	exit 1
fi
if (( stale != 0 || unpinnable != 0 )); then
	exit 2
fi
if (( infra != 0 || timed_out != 0 )); then
	exit 1
fi
