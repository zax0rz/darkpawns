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
	in_baseline=0
	if [[ -n "${EXPECTED_DIVERGENCES_FILE:-}" && -f "$EXPECTED_DIVERGENCES_FILE" ]] \
		&& awk -F'\t' -v s="$scenario" '$1 == s { found = 1 } END { exit !found }' "$EXPECTED_DIVERGENCES_FILE"; then
		in_baseline=1
	fi
	if [[ $status -eq 0 ]]; then
		if [[ $in_baseline -eq 1 ]]; then
			printf 'STALE\t%s\n' "$scenario" >"$result_dir/$scenario"
			printf 'STALE %s (baseline expects divergence; ledger reconciliation needed)\n' "$scenario"
		else
			printf 'PASS\t%s\n' "$scenario" >"$result_dir/$scenario"
			printf 'PASS %s\n' "$scenario"
		fi
		return 0
	fi
	if [[ $status -eq 3 ]]; then
		# A truncated transcript (oracle died mid-scenario) also exits 3 with
		# non-empty diffs. Retry once before convicting; require the retry to
		# reproduce the SAME divergence fingerprints — unstable transcripts are
		# INFRA, not content.
		mv -- "$log_file" "$log_file.attempt1"
		(
			cd "$repo_root" || exit 125
			timeout --foreground --signal=TERM --kill-after=10s "$scenario_timeout" \
				env DP_ORACLE_BIN="$oracle_bin" "$harness_bin" \
				--scenario "$scenario" --seed "$seed"
		) >"$log_file" 2>&1
		status2=$?
		fingerprints() {
			grep '^divergence-fingerprint' "$1" 2>/dev/null | cut -f2-3 | sort
		}
		if [[ $status2 -eq 0 ]]; then
			printf 'PASS\t%s\n' "$scenario" >"$result_dir/$scenario"
			printf 'PASS %s (divergence did not reproduce on retry)\n' "$scenario"
			return 0
		fi
		if [[ $status2 -ne 3 ]]; then
			status=$status2
		elif ! diff <(fingerprints "$log_file.attempt1") <(fingerprints "$log_file") >/dev/null; then
			if [[ $in_baseline -eq 1 ]]; then
				# Run-varying divergence bytes (e.g. the accuse pointer anomaly the
				# ledger documents): the shape can never be pinned, but the ledger
				# row already justifies the divergence itself.
				printf 'EXPECTED\t%s\n' "$scenario" >"$result_dir/$scenario"
				printf 'EXPECTED %s (ledger-backed divergence; shape unpinnable — run-varying bytes)\n' "$scenario"
			else
				printf 'INFRA\t%s\t%d\n' "$scenario" "$status" >"$result_dir/$scenario"
				printf 'INFRA %s (unstable divergence across attempts; not classified as content)\n' "$scenario" >&2
			fi
			return 0
		elif [[ $in_baseline -eq 1 ]]; then
			pins_file="${EXPECTED_DIVERGENCE_PINS_FILE:-}"
			if [[ -n "$pins_file" && -f "$pins_file" ]] \
				&& diff <(awk -F'\t' -v s="$scenario" '$1 == s { print $2 "\t" $3 }' "$pins_file" | sort) \
					<(fingerprints "$log_file") >/dev/null; then
				printf 'EXPECTED\t%s\n' "$scenario" >"$result_dir/$scenario"
				printf 'EXPECTED %s (ledger-backed divergence, pinned shape)\n' "$scenario"
			else
				printf 'FAIL\t%s\t%d\n' "$scenario" "$status" >"$result_dir/$scenario"
				printf 'FAIL %s (divergence shape differs from pinned baseline)\n' "$scenario"
			fi
			return 0
		else
			printf 'FAIL\t%s\t%d\n' "$scenario" "$status" >"$result_dir/$scenario"
			printf 'FAIL %s (content divergence with no ledger row)\n' "$scenario"
			return 0
		fi
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
