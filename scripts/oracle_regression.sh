#!/usr/bin/env bash
set -uo pipefail

repo_root=$(CDPATH= cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
go_bin=${ORACLE_REGRESSION_GO:-/usr/local/go/bin/go}
oracle_bin=${DP_ORACLE_BIN:-/home/zach/darkpawns-c-oracle/bin/circle}
scenario_timeout=${ORACLE_REGRESSION_TIMEOUT:-240s}
seed=${ORACLE_REGRESSION_SEED:-1}
jobs=${ORACLE_REGRESSION_JOBS:-4}

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

printf 'oracle-regression: %d scenarios, seed=%s, timeout=%s, jobs=%s\n' "${#scenarios[@]}" "$seed" "$scenario_timeout" "$jobs"
printf '%s\0' "${scenarios[@]}" | xargs -0 -n1 -P "$jobs" "$script_dir"/oracle_regression_worker.sh

passed=0
expected=0
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

printf 'oracle-regression: scenarios=%d passed=%d expected=%d stale=%d failed=%d infra=%d timed_out=%d elapsed=%d.%03ds started=%s finished=%s\n' \
	"${#scenarios[@]}" "$passed" "$expected" "$stale" "$failed" "$infra" "$timed_out" "$elapsed_seconds" "$elapsed_remainder" \
	"$(date -d "@$run_started" '+%Y-%m-%dT%H:%M:%S%z')" \
	"$(date -d "@$run_finished" '+%Y-%m-%dT%H:%M:%S%z')"

if (( failed != 0 )); then
	exit 1
fi
if (( stale != 0 )); then
	exit 2
fi
if (( infra != 0 || timed_out != 0 )); then
	exit 1
fi
