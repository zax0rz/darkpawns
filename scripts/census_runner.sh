#!/usr/bin/env bash
# census_runner.sh — detached finalization and verdicts, launched by census.py.
# Inputs are snapshotted before queueing. All builds and scenario/recheck workers
# use census_budget.py; this shell never owns a global exclusive census lock.
# Never run by hand.
set -u

run_dir=${CENSUS_RUN_DIR:?CENSUS_RUN_DIR is required}
runner=${CENSUS_RUNNER:?CENSUS_RUNNER is required}
repo_root=${CENSUS_REPO_ROOT:?CENSUS_REPO_ROOT is required}
oracle_bin=${CENSUS_ORACLE_BIN:?CENSUS_ORACLE_BIN is required}
oracle_sha=${CENSUS_ORACLE_SHA:?CENSUS_ORACLE_SHA is required}
allow_note=${CENSUS_ALLOW_NOTE:-}
scenarios=${CENSUS_SCENARIOS:-}
jobs=${CENSUS_JOBS:-36}
seed=${CENSUS_SEED:-1}
budget_tool="$repo_root/scripts/census_budget.py"
runner_command=("$runner")

pid=$$
started_iso=$(date '+%Y-%m-%dT%H:%M:%S%z')
started_s=$(date +%s)

# A cancelled runner drains separately leased engine groups through the registry
# and terminates its own dispatch group. No other run's state/locks are cleared.
killed() {
 trap '' TERM INT
	python3 "$budget_tool" cancel "$run_dir" || true
	printf 'died verdict=KILLED\n' >"$run_dir/state"
 kill -- "-$pid" 2>/dev/null || true
 exit 4
}
trap killed TERM INT


finish() {
	local verdict=$1 exit_code=$2
	local finished_s finished_iso
	finished_s=$(date +%s)
	finished_iso=$(date '+%Y-%m-%dT%H:%M:%S%z')
	local summary_line
	if [[ "${CENSUS_COMBINED:-0}" == 1 ]]; then
		summary_line=$(cat "$run_dir/combined-summary.txt")
	elif [[ "${CENSUS_CLAIMS:-0}" == 1 ]]; then
		summary_line=$(cat "$run_dir/claims-summary.txt")
	else
		summary_line=$(grep '^oracle-regression: ' "$run_dir/census.log" | tail -1)
	fi
	{
		printf '%s verdict=%s' "$summary_line" "$verdict"
		if [[ -s "$run_dir/recheck-results.tsv" ]]; then
			printf ' rechecked=%s' "$(awk -F '\t' '{ printf "%s%s", sep, $2; sep="," }' "$run_dir/recheck-results.tsv")"
		fi
		printf '\n'
		# DP-1427: the draw-parity tally is its own line; it never changes the
		# byte verdict above.
		grep '^oracle-draw-parity: ' "$run_dir/census.log" | tail -1 || true
	} >"$run_dir/summary.pending"
	printf '%s\n' "$exit_code" >"$run_dir/exit-code.pending"

	printf 'done verdict=%s\n' "$verdict" >"$run_dir/state.pending"
	write_manifest "$verdict" "$started_iso" "$finished_iso" "$summary_line"
	mv "$run_dir/MANIFEST.pending" "$run_dir/MANIFEST.md"
	mv "$run_dir/summary.pending" "$run_dir/summary.txt"
	mv "$run_dir/exit-code.pending" "$run_dir/exit-code"
	mv "$run_dir/state.pending" "$run_dir/state"
	python3 "$budget_tool" finish
	if command -v notify-send >/dev/null 2>&1; then
		notify-send "DP census ${run_dir##*/}" "$verdict" 2>/dev/null || true
	fi
	exit "$exit_code"
}

write_manifest() {
	local verdict=$1 started_iso=$2 finished_iso=$3 summary_line=$4
	local head dirty
	head=$(cat -- "$run_dir/go-head.txt" 2>/dev/null)
	if [[ -s "$run_dir/git-status.txt" ]] && [[ "$(cat -- "$run_dir/git-status.txt")" != "" ]]; then
		dirty=dirty
	else
		dirty=clean
	fi
	local selection=full
	[[ -n "$scenarios" ]] && selection="$scenarios"
	local recheck_note=
	if [[ -s "$run_dir/recheck-results.tsv" ]]; then
		recheck_note=$(awk -F '\t' '{ printf "%s ", $2 }' "$run_dir/recheck-results.tsv")
	fi
	{
		printf '# Census manifest — written by scripts/census.sh\n\n'
		if [[ -n "$allow_note" ]]; then
			printf '**%s**\n\n' "$allow_note"
		fi
		printf -- '- Go HEAD (captured before the run): `%s`\n' "$head"
		printf -- '- Tree: %s (see git-status.txt)\n' "$dirty"
		printf -- '- Oracle: `%s` sha256 `%s`\n' "$oracle_bin" "$oracle_sha"
		printf -- '- Reference oracle: %s\n' "$([[ -z "$allow_note" ]] && printf yes || printf NO)"
		printf -- '- Admission: host budget 72; per-run maximum 36; MemAvailable floor 4 GiB; see admission.json and admission-events.jsonl\n'
		printf -- '- Immutable inputs: source/ and source-hashes.json; captured before queueing\n'
		printf -- '- Seed: %s · Workers: %s · Selection: %s\n' "$seed" "$jobs" "$selection"
		printf -- '- Commands: `census.sh start --name %s` → `%s`' "${run_dir##*/}" "$(basename "$runner")"
		[[ -n "$scenarios" ]] && printf ' with ORACLE_REGRESSION_SCENARIOS=%s' "$scenarios"
		printf '\n'
		printf -- '- Started: %s · Finished: %s\n' "$started_iso" "$finished_iso"
		printf -- '- Summary: `%s`\n' "$summary_line"
		printf -- '- Verdict: **%s**\n' "$verdict"
		[[ -n "$recheck_note" ]] && printf -- '- Rechecked: %s\n' "$recheck_note"
		if [[ "${CENSUS_CLAIMS:-0}" == 1 || "${CENSUS_COMBINED:-0}" == 1 ]]; then
			printf '\n## Claims\n\n'
			printf -- '- Go build flags: `%s`\n' "${GOFLAGS:-default}"
			printf -- '- Manifest HEAD: `%s`\n' "$(cat "$run_dir/manifests-head.txt")"
			printf -- '- Enumeration: `claims.tsv` (manifest claims, unchanged across modes)\n'
			awk -F '\t' '{n[$1]++} END {for (s in n) printf "seed %s: %d pairs\n", s, n[s]}' "$run_dir/claims.tsv" | sort -k2,2n
		fi
		if [[ "${CENSUS_COMBINED:-0}" == 1 ]]; then
			printf '\n## Combined gate\n\n'
			printf -- '- Command: `census.sh start --combined --name %s --jobs %s`\n' "${run_dir##*/}" "$jobs"
			printf -- '- Pair schedule: `pairs.tsv`; worker artifacts: `seed*/initial-logs/`, `seed*/recheck-logs/`\n'
			printf -- '- Full: `%s` verdict=%s\n' "$(cat "$run_dir/full-summary.txt")" "$(cat "$run_dir/full-verdict")"
			printf -- '- Claims: `%s` verdict=%s\n' "$(cat "$run_dir/claims-summary.txt")" "$(cat "$run_dir/claims-verdict")"
		fi
		printf -- '\n## File sizes (du -b)\n\n'
		du -b -- "$run_dir"/* 2>/dev/null | \
            awk -F '\t' 'BEGIN { OFS="\t" } $2 !~ /\/(MANIFEST.pending|state|admission-events.jsonl)$/ { sub(/summary.pending$/, "summary.txt", $2); sub(/exit-code.pending$/, "exit-code", $2); sub(/state.pending$/, "state", $2); print }' | sort -k2
		if [[ -d "$run_dir/census-dump" ]]; then
			printf '\n## census-dump\n\n'
			du -b -- "$run_dir/census-dump" 2>/dev/null
		fi
	} >"$run_dir/MANIFEST.pending"
}

if [[ "${CENSUS_COMBINED:-0}" == 1 ]]; then
	python3 "$repo_root/scripts/combined_census.py" >>"$run_dir/census.log" 2>&1 9>&- &
	child=$!
	wait "$child"
	if [[ ! -s "$run_dir/combined-verdict" ]]; then
		printf 'oracle-combined: driver failed\n' >"$run_dir/combined-summary.txt"
		finish NOT_CLEAN 1
	fi
	verdict=$(cat "$run_dir/combined-verdict")
	[[ "$verdict" == NOT_CLEAN ]] && finish "$verdict" 1
	finish "$verdict" 0
fi

if [[ "${CENSUS_CLAIMS:-0}" == 1 ]]; then
	python3 "$repo_root/scripts/claims_census.py" >>"$run_dir/census.log" 2>&1 9>&- &
	child=$!
	wait "$child"
	if [[ ! -s "$run_dir/claims-verdict" ]]; then
		printf 'oracle-claims: driver failed\n' >"$run_dir/claims-summary.txt"
		finish NOT_CLEAN 1
	fi
	verdict=$(cat "$run_dir/claims-verdict")
	[[ "$verdict" == NOT_CLEAN ]] && finish "$verdict" 1
	finish "$verdict" 0
fi

# ---------------------------------------------------------------------------
# 1. The census itself.
# ---------------------------------------------------------------------------
regression_env=(
	env
	ORACLE_REGRESSION_DUMP="$run_dir/census-dump"
	ORACLE_REGRESSION_JOBS="$jobs"
	ORACLE_REGRESSION_SEED="$seed"
	ORACLE_REGRESSION_RESULTS="$run_dir/results.tsv"
 ORACLE_REGRESSION_LOG_ROOT="$run_dir/initial-logs"
)
if [[ -n "$scenarios" ]]; then
	regression_env+=("ORACLE_REGRESSION_SCENARIOS=$scenarios")
fi
if [[ "${CENSUS_DRAW_PARITY:-0}" == 1 ]]; then
	regression_env+=("ORACLE_REGRESSION_DRAW_PARITY=1")
fi
# Background + wait lets TERM interrupt immediately. Worker leases supervise
# their own child groups, including SIGKILL recovery by another registry client.
"${regression_env[@]}" "${runner_command[@]}" >>"$run_dir/census.log" 2>&1 9>&- &
child=$!
wait "$child"
# Non-zero exit alone doesn't decide the verdict: the summary counts do.
:

[[ -s "$run_dir/results.tsv" ]] || printf 'FAIL\t(no results file)\n' >"$run_dir/results.tsv"

# ---------------------------------------------------------------------------
# 2. Verdict. FAIL/STALE/UNPINNABLE rows are never rechecked (R5h).
# ---------------------------------------------------------------------------
counts=$(awk -F '\t' '
	$1 == "PASS" { p++ } $1 == "EXPECTED" { e++ } $1 == "EXPECTED_UNSTABLE" { u++ }
	$1 == "FAIL" { f++ } $1 == "STALE" { s++ } $1 == "UNPINNABLE" { n++ }
	$1 == "INFRA" { i++ } $1 == "TIMEOUT" { t++ }
	END {
		printf "%d %d %d %d %d %d %d %d", p + 0, e + 0, u + 0, f + 0, s + 0, n + 0, i + 0, t + 0
	}' "$run_dir/results.tsv")
read -r passed expected unstable failed stale unpinnable infra timed_out <<<"$counts"

if ((failed != 0 || stale != 0 || unpinnable != 0)); then
	finish NOT_CLEAN 1
fi
if ((infra == 0 && timed_out == 0)); then
	finish CLEAN 0
fi

# ---------------------------------------------------------------------------
# 3. INFRA/TIMEOUT recheck: those scenarios once, at --jobs 1.
# ---------------------------------------------------------------------------
recheck_names=$(awk -F '\t' '$1 == "INFRA" || $1 == "TIMEOUT" { printf "%s%s", sep, $2; sep="," }' "$run_dir/results.tsv")
if [[ -z "$recheck_names" ]]; then
	finish CLEAN 0
fi
recheck_env=(
	env
	ORACLE_REGRESSION_JOBS=1
	ORACLE_REGRESSION_SEED="$seed"
	ORACLE_REGRESSION_RESULTS="$run_dir/recheck-results.tsv"
 ORACLE_REGRESSION_LOG_ROOT="$run_dir/recheck-logs"
 ORACLE_REGRESSION_DUMP="$run_dir/recheck-dump"
	ORACLE_REGRESSION_SCENARIOS="$recheck_names"
)
if [[ "${CENSUS_DRAW_PARITY:-0}" == 1 ]]; then
	recheck_env+=("ORACLE_REGRESSION_DRAW_PARITY=1")
fi
# Same background+wait discipline as the main invocation.
"${recheck_env[@]}" "${runner_command[@]}" >"$run_dir/recheck.log" 2>&1 9>&- &
child=$!
wait "$child"
:
if [[ ! -s "$run_dir/recheck-results.tsv" ]]; then
	finish NOT_CLEAN 1
fi

recheck_bad=$(awk -F '\t' '$1 != "PASS" && $1 != "EXPECTED" && $1 != "EXPECTED_UNSTABLE" && $1 != "DRAW" { print; exit }' "$run_dir/recheck-results.tsv")
if [[ -n "$recheck_bad" ]]; then
	finish NOT_CLEAN 1
fi
finish CLEAN_AFTER_RECHECK 0
