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

run_started=$(date +%s)
run_started_ns=$(date +%s%N)
log_dir=$(mktemp -d "${TMPDIR:-/tmp}/dp-oracle-regression.XXXXXX")
result_dir="$log_dir/results"
mkdir -p -- "$result_dir"
cleanup() {
	rm -rf -- "$log_dir"
}
trap cleanup EXIT

run_one() {
	local scenario_file=$1
	scenario=${scenario_file%.txt}
	log_file="$log_dir/$scenario.log"
	(
		cd "$repo_root" || exit 125
		timeout --foreground --signal=TERM --kill-after=10s "$scenario_timeout" \
			env DP_ORACLE_BIN="$oracle_bin" "$go_bin" run ./cmd/dp-oracle-diff \
			--scenario "$scenario" --seed "$seed"
	) >"$log_file" 2>&1
	status=$?
	if [[ $status -eq 0 ]]; then
		printf 'PASS\t%s\n' "$scenario" >"$result_dir/$scenario"
		printf 'PASS %s\n' "$scenario"
		return 0
	fi

	if [[ $status -ne 124 && $status -ne 137 && $status -ne 143 ]] &&
		grep -Eq 'exited before readiness|did not log .*within|: EOF|connection (reset|closed)' "$log_file"; then
		printf 'RETRY %s once (infrastructure-shaped failure)\n' "$scenario" >&2
		mv -- "$log_file" "$log_file.attempt1"
		(
			cd "$repo_root" || exit 125
			timeout --foreground --signal=TERM --kill-after=10s "$scenario_timeout" \
				env DP_ORACLE_BIN="$oracle_bin" "$go_bin" run ./cmd/dp-oracle-diff \
				--scenario "$scenario" --seed "$seed"
		) >"$log_file" 2>&1
		status=$?
		if [[ $status -eq 0 ]]; then
			printf 'PASS %s (after infra retry)\n' "$scenario"
			printf 'PASS\t%s\n' "$scenario" >"$result_dir/$scenario"
			return 0
		fi
	fi

	if [[ $status -eq 124 || $status -eq 137 || $status -eq 143 ]]; then
		printf 'TIMEOUT\t%s\t%d\n' "$scenario" "$status" >"$result_dir/$scenario"
		printf 'TIMEOUT %s (exit %d; not classified as a content diff)\n' "$scenario" "$status" >&2
		return 0
	fi

	if grep -Eq 'exited before readiness|did not log .*within|: EOF|connection (reset|closed)' "$log_file"; then
		printf 'INFRA\t%s\t%d\n' "$scenario" "$status" >"$result_dir/$scenario"
		printf 'INFRA %s (exit %d after one retry; not classified as a content diff)\n' "$scenario" "$status" >&2
	else
		printf 'FAIL\t%s\t%d\n' "$scenario" "$status" >"$result_dir/$scenario"
		printf 'FAIL %s (exit %d)\n' "$scenario" "$status" >&2
	fi
	sed -n '1,120p' "$log_file" >&2
}
export -f run_one
export repo_root go_bin oracle_bin scenario_timeout seed log_dir result_dir

printf 'oracle-regression: %d scenarios, seed=%s, timeout=%s, jobs=%s\n' "${#scenarios[@]}" "$seed" "$scenario_timeout" "$jobs"
printf '%s\0' "${scenarios[@]}" | xargs -0 -n1 -P "$jobs" bash -c 'run_one "$1"' _

passed=0
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

printf 'oracle-regression: scenarios=%d passed=%d failed=%d infra=%d timed_out=%d elapsed=%d.%03ds started=%s finished=%s\n' \
	"${#scenarios[@]}" "$passed" "$failed" "$infra" "$timed_out" "$elapsed_seconds" "$elapsed_remainder" \
	"$(date -d "@$run_started" '+%Y-%m-%dT%H:%M:%S%z')" \
	"$(date -d "@$run_finished" '+%Y-%m-%dT%H:%M:%S%z')"

if (( failed != 0 || infra != 0 || timed_out != 0 )); then
	exit 1
fi
