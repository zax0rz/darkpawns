#!/usr/bin/env bash
# oracle_regression_worker.sh — per-scenario worker for scripts/oracle_regression.sh.
# Invoked by xargs as: oracle_regression_worker.sh <scenario-file-name>
# Shared state arrives via exported variables (repo_root, harness_bin, oracle_bin,
# scenario_timeout, seed, log_dir, result_dir); replaces the old export -f
# round-trip whose bash reconstruction proved fragile.
set -u

main() {
	local scenario_file=$1
	scenario=${scenario_file%.txt}
	log_file="$log_dir/$scenario.log"
	(
		cd "$repo_root" || exit 125
		timeout --foreground --signal=TERM --kill-after=10s "$scenario_timeout" \
			env DP_ORACLE_BIN="$oracle_bin" "$harness_bin" \
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
				env DP_ORACLE_BIN="$oracle_bin" "$harness_bin" \
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

main "$1"
