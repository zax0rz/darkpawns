#!/usr/bin/env bash
# oracle_regression_worker.sh — per-scenario worker for scripts/oracle_regression.sh.
# Invoked by xargs as: oracle_regression_worker.sh <scenario-file-name>
# Shared state arrives via exported variables (repo_root, harness_bin, oracle_bin,
# scenario_timeout, seed, log_dir, result_dir). Every scenario gets at most three
# process attempts: recovery and content confirmation share one budget.
set -u

main() {
	local scenario_file=$1
	local scenario=${scenario_file%.txt}
	local max_attempts=3
	local attempt=0
	local attempt_status=0
	local attempt_log
	local in_baseline=0
	local pins_file=${EXPECTED_DIVERGENCE_PINS_FILE:-}
	local fingerprint_dir="$log_dir/$scenario.fingerprints"
	local fingerprint_file
	local -a content_attempts=()

	mkdir -p -- "$fingerprint_dir"
	if [[ -n "${EXPECTED_DIVERGENCES_FILE:-}" && -f "$EXPECTED_DIVERGENCES_FILE" ]] \
		&& awk -F '\t' -v s="$scenario" '$1 == s { found = 1 } END { exit !found }' "$EXPECTED_DIVERGENCES_FILE"; then
		in_baseline=1
	fi

	write_result() {
		local kind=$1
		local status=${2:-}
		if [[ -n "$status" ]]; then
			printf '%s\t%s\t%s\n' "$kind" "$scenario" "$status" >"$result_dir/$scenario"
		else
			printf '%s\t%s\n' "$kind" "$scenario" >"$result_dir/$scenario"
		fi
	}

	is_timeout() {
		case $1 in
		124|137|143) return 0 ;;
		*) return 1 ;;
		esac
	}

	has_infra_signature() {
		grep -Eq 'exited before readiness|did not log .*within|: EOF|connection (reset|closed)' "$1"
	}

	run_attempt() {
		attempt=$((attempt + 1))
		attempt_log="$log_dir/$scenario.attempt$attempt.log"
		(
			cd "$repo_root" || exit 125
			timeout --foreground --signal=TERM --kill-after=10s "$scenario_timeout" \
				env DP_ORACLE_BIN="$oracle_bin" "$harness_bin" \
				--scenario "$scenario" --seed "$seed"
		) >"$attempt_log" 2>&1
		attempt_status=$?
	}

	write_success() {
		if [[ $in_baseline -eq 1 ]]; then
			write_result STALE
			printf 'STALE %s (baseline expects divergence; ledger reconciliation needed)\n' "$scenario"
		elif ((${#content_attempts[@]} != 0)); then
			# A successful content attempt after a divergent one is flaky-red, not
			# proof that the non-baselined divergence was harmless.
			write_result INFRA 3
			printf 'INFRA %s (divergence did not reproduce on content attempt)\n' "$scenario" >&2
		else
			write_result PASS
			printf 'PASS %s\n' "$scenario"
		fi
	}

	extract_fingerprints() {
		local input=$1
		local output=$2
		local raw="$output.raw"
		if ! awk -F '\t' '
			/^divergence-fingerprint\t/ {
				saw = 1
				if (NF != 3 || $2 == "" || length($3) != 64 || $3 !~ /^[0-9a-f]+$/ || seen[$2]++) {
					bad = 1
					next
				}
				print $2 "\t" $3
			}
			END { if (!saw || bad) exit 1 }
		' "$input" >"$raw"; then
			return 1
		fi
		sort -- "$raw" >"$output"
		return $?
	}

	pins_match() {
		local fingerprints=$1
		local expected="$fingerprint_dir/pins"
		local raw="$expected.raw"
		if [[ -z "$pins_file" || ! -f "$pins_file" ]]; then
			return 1
		fi
		if ! awk -F '\t' -v s="$scenario" '
			NR == 1 { next }
			$1 == s {
				found = 1
				if (NF != 4 || $2 == "" || length($3) != 64 || $3 !~ /^[0-9a-f]+$/ || seen[$2]++) {
					bad = 1
					next
				}
				print $2 "\t" $3
			}
			END { if (!found || bad) exit 1 }
		' "$pins_file" >"$raw"; then
			return 1
		fi
		sort -- "$raw" >"$expected" || return 1
		diff -- "$expected" "$fingerprints" >/dev/null
	}

	classify_content_pair() {
		local first=${content_attempts[0]}
		local second=${content_attempts[1]}
		local first_fingerprints="$fingerprint_dir/attempt$first"
		local second_fingerprints="$fingerprint_dir/attempt$second"
		if ! diff -- "$first_fingerprints" "$second_fingerprints" >/dev/null; then
			if [[ $in_baseline -eq 1 ]]; then
				write_result UNPINNABLE
				printf 'UNPINNABLE %s (ledger-backed divergence; shape unpinnable — run-varying bytes; requires human clearance)\n' "$scenario"
			else
				write_result INFRA 3
				printf 'INFRA %s (unstable divergence across content attempts; not classified as content)\n' "$scenario" >&2
			fi
			return
		fi
		if [[ $in_baseline -eq 1 ]]; then
			if pins_match "$second_fingerprints"; then
				write_result EXPECTED
				printf 'EXPECTED %s (ledger-backed divergence, pinned shape)\n' "$scenario"
			else
				write_result FAIL 3
				printf 'FAIL %s (divergence shape differs from pinned baseline)\n' "$scenario"
			fi
		else
			write_result FAIL 3
			printf 'FAIL %s (content divergence with no ledger row)\n' "$scenario"
		fi
	}

	while ((attempt < max_attempts)); do
		run_attempt
		if [[ $attempt_status -eq 0 ]]; then
			write_success
			return 0
		fi
		if is_timeout "$attempt_status"; then
			write_result TIMEOUT "$attempt_status"
			printf 'TIMEOUT %s (exit %d; not classified as a content diff)\n' "$scenario" "$attempt_status" >&2
			return 0
		fi
		if [[ $attempt_status -eq 3 ]]; then
			fingerprint_file="$fingerprint_dir/attempt$attempt"
			if ! extract_fingerprints "$attempt_log" "$fingerprint_file"; then
				write_result FAIL 3
				printf 'FAIL %s (missing or malformed divergence fingerprints)\n' "$scenario" >&2
				return 0
			fi
			content_attempts+=("$attempt")
			if ((${#content_attempts[@]} == 2)); then
				classify_content_pair
				return 0
			fi
			if ((attempt == max_attempts)); then
				write_result INFRA 3
				printf 'INFRA %s (content divergence lacked a confirming attempt within the %d-attempt budget)\n' "$scenario" "$max_attempts" >&2
				return 0
			fi
			continue
		fi
		if has_infra_signature "$attempt_log"; then
			if ((attempt == max_attempts)); then
				write_result INFRA "$attempt_status"
				printf 'INFRA %s (exit %d after the %d-attempt budget; not classified as a content diff)\n' "$scenario" "$attempt_status" "$max_attempts" >&2
				return 0
			fi
			printf 'RETRY %s (infrastructure-shaped attempt %d of %d)\n' "$scenario" "$attempt" "$max_attempts" >&2
			continue
		fi
		write_result FAIL "$attempt_status"
		printf 'FAIL %s (exit %d)\n' "$scenario" "$attempt_status" >&2
		sed -n '1,120p' "$attempt_log" >&2
		return 0
	done
}

main "$1"
