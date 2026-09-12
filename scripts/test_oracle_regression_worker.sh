#!/usr/bin/env bash
set -euo pipefail

repo_root=$(CDPATH= cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
worker="$repo_root/scripts/oracle_regression_worker.sh"
harness="$repo_root/scripts/testdata/oracle_regression_fake_harness.sh"
fake_go="$repo_root/scripts/testdata/oracle_regression_fake_go.sh"
test_root=$(mktemp -d "${TMPDIR:-/tmp}/dp-oracle-worker-test.XXXXXX")
trap 'rm -rf -- "$test_root"' EXIT

pass_count=0

fail() {
	printf 'FAIL: %s\n' "$1" >&2
	exit 1
}

assert_eq() {
	local want=$1
	local got=$2
	local what=$3
	if [[ "$want" != "$got" ]]; then
		fail "$what: want $(printf '%q' "$want"), got $(printf '%q' "$got")"
	fi
}

run_case() {
	local name=$1
	local sequence=$2
	local baseline=$3
	local pins=$4
	local want_result=$5
	local want_attempts=$6
	local want_logs=$7
	local case_dir="$test_root/$name"
	local log_dir="$case_dir/logs"
	local result_dir="$case_dir/results"
	local ledger="$case_dir/ledger.tsv"
	local pins_file="$case_dir/pins.tsv"
	local output status result attempts log_count

	mkdir -p -- "$log_dir" "$result_dir"
	if [[ "$baseline" == yes ]]; then
		printf 'scenario\tmanifest\tcase_id\tstatus\nfake\ttest.tsv\t%s\tblocked\n' "$name" >"$ledger"
	else
		printf 'scenario\tmanifest\tcase_id\tstatus\n' >"$ledger"
	fi
	if [[ -n "$pins" ]]; then
		printf 'scenario\tlabel\tsha256\tcitations\n%s\n' "$pins" >"$pins_file"
	else
		printf 'scenario\tlabel\tsha256\tcitations\n' >"$pins_file"
	fi

	set +e
	output=$(
		env \
			repo_root="$repo_root" \
			harness_bin="$harness" \
			oracle_bin=/bin/true \
			scenario_timeout=1s \
			seed=1 \
			log_dir="$log_dir" \
			result_dir="$result_dir" \
			EXPECTED_DIVERGENCES_FILE="$ledger" \
			EXPECTED_DIVERGENCE_PINS_FILE="$pins_file" \
			FAKE_SEQUENCE="$sequence" \
			FAKE_STATE_DIR="$case_dir/state" \
			"$worker" fake.txt 2>&1
	)
	status=$?
	set -e
	assert_eq 0 "$status" "$name worker exit"

	result=$(<"$result_dir/fake")
	assert_eq "$want_result" "$result" "$name result file"
	attempts=$(<"$case_dir/state/attempt-count")
	assert_eq "$want_attempts" "$attempts" "$name attempt count"
	log_count=$(find "$log_dir" -maxdepth 1 -type f -name 'fake.attempt*.log' -printf '%f\n' | wc -l)
	log_count=${log_count//[[:space:]]/}
	assert_eq "$want_logs" "$log_count" "$name preserved attempt-log count"
	for ((attempt = 1; attempt <= want_logs; attempt++)); do
		if [[ ! -s "$log_dir/fake.attempt$attempt.log" ]]; then
			fail "$name missing preserved attempt $attempt log"
		fi
	done
	if [[ "$output" == *"ERROR"* ]]; then
		fail "$name emitted an unexpected error: $output"
	fi
	pass_count=$((pass_count + 1))
}

run_aggregate_case() {
	local name=$1
	local scenario=$2
	local sequence=$3
	local want_status=$4
	local want_summary=$5
	local output status

	set +e
	output=$(
		env \
			ORACLE_REGRESSION_GO="$fake_go" \
			DP_ORACLE_BIN=/bin/true \
			ORACLE_REGRESSION_TIMEOUT=1s \
			ORACLE_REGRESSION_SEED=1 \
			ORACLE_REGRESSION_JOBS=1 \
			ORACLE_REGRESSION_SCENARIOS="$scenario" \
			FAKE_HARNESS_SOURCE="$harness" \
			FAKE_SEQUENCE="$sequence" \
			FAKE_STATE_DIR="$test_root/aggregate-$name/state" \
			"$repo_root/scripts/oracle_regression.sh" 2>&1
	)
	status=$?
	set -e
	assert_eq "$want_status" "$status" "$name aggregate exit"
	if [[ "$output" != *"$want_summary"* ]]; then
		fail "$name aggregate summary missing $(printf '%q' "$want_summary"): $output"
	fi
	pass_count=$((pass_count + 1))
}

one_a=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
one_b=bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb
two_a=1111111111111111111111111111111111111111111111111111111111111111
two_b=2222222222222222222222222222222222222222222222222222222222222222
two_c=3333333333333333333333333333333333333333333333333333333333333333
medit_a=134cb9c489e39d317ab67587e401d50b8f0382d5ceb9cfdd3050ddaceaaa2eda
medit_b=5d8d92b3c45a9d7f753223206d917b47609be74601ef7c19ed48866c87d0abf9
medit_c=0f8807f28888522a9809fedd3386d648b22fa17f977fbc3d9cf220f3acb576d3
medit_set="D:medit:$medit_a,medit abc:$medit_b,medit 999999:$medit_c"

pin_one() {
	printf 'fake\tcast\t%s\ttest.tsv:%s:blocked' "$1" "$2"
}

pin_two() {
	printf 'fake\tcast\t%s\ttest.tsv:%s:blocked\nfake\tdamage\t%s\ttest.tsv:%s:blocked' "$1" "$2" "$3" "$2"
}

run_case immediate-pass P no "" $'PASS\tfake' 1 1
run_case immediate-stale P yes "" $'STALE\tfake' 1 1
run_case immediate-pinned-confirmed "D:cast:$one_a|D:cast:$one_a" yes "$(pin_one "$one_a" immediate-pinned-confirmed)" $'EXPECTED\tfake' 2 2
run_case infra-pass "I|P" no "" $'PASS\tfake' 2 2
run_case infra-stale "I|P" yes "" $'STALE\tfake' 2 2
run_case infra-pinned-confirmed "I|D:cast:$one_a|D:cast:$one_a" yes "$(pin_one "$one_a" infra-pinned-confirmed)" $'EXPECTED\tfake' 3 3
run_case infra-unpinned-confirmed "I|D:cast:$one_a|D:cast:$one_a" no "" $'FAIL\tfake\t3' 3 3
run_case infra-pin-mismatch "I|D:cast:$one_a|D:cast:$one_a" yes "$(pin_one "$one_b" infra-pin-mismatch)" $'FAIL\tfake\t3' 3 3
run_case infra-single-divergence "I|D:cast:$one_a|I" yes "$(pin_one "$one_a" infra-single-divergence)" $'INFRA\tfake\t1' 3 3
run_case divergence-then-success "D:cast:$one_a|P" no "" $'INFRA\tfake\t3' 2 2
run_case baseline-divergence-then-success "D:cast:$one_a|P" yes "$(pin_one "$one_a" baseline-divergence-then-success)" $'STALE\tfake' 2 2
run_case unstable-divergence "D:cast:$one_a|D:cast:$one_b" yes "$(pin_one "$one_a" unstable-divergence)" $'UNPINNABLE\tfake' 2 2
run_case divergence-infra-confirmed "D:cast:$one_a|I|D:cast:$one_a" yes "$(pin_one "$one_a" divergence-infra-confirmed)" $'EXPECTED\tfake' 3 3
run_case repeated-infra "I|I|I" no "" $'INFRA\tfake\t1' 3 3
run_case timeout T no "" $'TIMEOUT\tfake\t124' 1 1
run_case infra-timeout "I|T" no "" $'TIMEOUT\tfake\t124' 2 2
run_case missing-fingerprints M no "" $'FAIL\tfake\t3' 1 1
run_case malformed-fingerprints X:cast:not-a-sha256 no "" $'FAIL\tfake\t3' 1 1
run_case multiblock-complete "I|D:cast:$two_a,damage:$two_b|D:cast:$two_a,damage:$two_b" yes "$(pin_two "$two_a" multiblock-complete "$two_b")" $'EXPECTED\tfake' 3 3
run_case multiblock-missing "I|D:cast:$two_a,damage:$two_b|D:cast:$two_a,damage:$two_b" yes "$(pin_one "$two_a" multiblock-missing)" $'FAIL\tfake\t3' 3 3
run_case multiblock-extra "I|D:cast:$two_a|D:cast:$two_a" yes "$(pin_two "$two_a" multiblock-extra "$two_b")" $'FAIL\tfake\t3' 3 3
run_case multiblock-changed "I|D:cast:$two_a,damage:$two_c|D:cast:$two_a,damage:$two_c" yes "$(pin_two "$two_a" multiblock-changed "$two_b")" $'FAIL\tfake\t3' 3 3

run_aggregate_case aggregate-pass yuball-depth P 0 'scenarios=1 passed=1 expected=0 unpinnable=0 stale=0 failed=0 infra=0 timed_out=0'
run_aggregate_case aggregate-stale medit-entry-depth P 2 'scenarios=1 passed=0 expected=0 unpinnable=0 stale=1 failed=0 infra=0 timed_out=0'
run_aggregate_case aggregate-expected medit-entry-depth "$medit_set|$medit_set" 0 'scenarios=1 passed=0 expected=1 unpinnable=0 stale=0 failed=0 infra=0 timed_out=0'
run_aggregate_case aggregate-fail medit-entry-depth "D:medit:$one_a|D:medit:$one_a" 1 'scenarios=1 passed=0 expected=0 unpinnable=0 stale=0 failed=1 infra=0 timed_out=0'
run_aggregate_case aggregate-unpinnable accuse-noarg-depth "D:accuse:$one_a|D:accuse:$one_b" 2 'scenarios=1 passed=0 expected=0 unpinnable=1 stale=0 failed=0 infra=0 timed_out=0'
run_aggregate_case aggregate-infra yuball-depth "I|I|I" 1 'scenarios=1 passed=0 expected=0 unpinnable=0 stale=0 failed=0 infra=1 timed_out=0'
run_aggregate_case aggregate-timeout yuball-depth T 1 'scenarios=1 passed=0 expected=0 unpinnable=0 stale=0 failed=0 infra=0 timed_out=1'

printf 'PASS: %d deterministic oracle regression cases\n' "$pass_count"
