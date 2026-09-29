#!/usr/bin/env bash
# census_runner.sh — the detached half of scripts/census.sh. Launched with
# setsid/nohup by census.sh start, with the census lock open on the FD named
# by CENSUS_LOCK_FD. Runs the regression script, rechecks INFRA/TIMEOUT rows,
# writes summary.txt, MANIFEST.md and exit-code, flips state, releases the
# lock, and notifies. Never run by hand.
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
lock_fd=${CENSUS_LOCK_FD:-9}
lock_file=${XDG_RUNTIME_DIR:-/tmp}/dp-census.lock

pid=$$
started_iso=$(date '+%Y-%m-%dT%H:%M:%S%z')
started_s=$(date +%s)

finish() {
	local verdict=$1 exit_code=$2
	local finished_s finished_iso
	finished_s=$(date +%s)
	finished_iso=$(date '+%Y-%m-%dT%H:%M:%S%z')
	local summary_line
	summary_line=$(grep '^oracle-regression: ' "$run_dir/census.log" | tail -1)
	printf '%s verdict=%s\n' "$summary_line" "$verdict" >"$run_dir/summary.txt"
	printf '%s\n' "$exit_code" >"$run_dir/exit-code"
	printf 'done verdict=%s\n' "$verdict" >"$run_dir/state"
	write_manifest "$verdict" "$started_iso" "$finished_iso" "$summary_line"
	# Release the lock and clear its body.
	flock -u "$lock_fd" 2>/dev/null || true
	: >"$lock_file" 2>/dev/null || true
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
		printf -- '- Seed: %s · Workers: %s · Selection: %s\n' "$seed" "$jobs" "$selection"
		printf -- '- Commands: `census.sh start --name %s` → `%s`' "${run_dir##*/}" "$(basename "$runner")"
		[[ -n "$scenarios" ]] && printf ' with ORACLE_REGRESSION_SCENARIOS=%s' "$scenarios"
		printf '\n'
		printf -- '- Started: %s · Finished: %s\n' "$started_iso" "$finished_iso"
		printf -- '- Summary: `%s`\n' "$summary_line"
		printf -- '- Verdict: **%s**\n' "$verdict"
		[[ -n "$recheck_note" ]] && printf -- '- Rechecked: %s\n' "$recheck_note"
		printf -- '\n## File sizes (du -b)\n\n'
		du -b -- "$run_dir"/* 2>/dev/null | sort -k2
		if [[ -d "$run_dir/census-dump" ]]; then
			printf '\n## census-dump\n\n'
			du -b -- "$run_dir/census-dump" 2>/dev/null
		fi
	} >"$run_dir/MANIFEST.md"
}

# ---------------------------------------------------------------------------
# 1. The census itself.
# ---------------------------------------------------------------------------
regression_env=(
	env
	ORACLE_REGRESSION_DUMP="$run_dir/census-dump"
	ORACLE_REGRESSION_JOBS="$jobs"
	ORACLE_REGRESSION_SEED="$seed"
	ORACLE_REGRESSION_RESULTS="$run_dir/results.tsv"
)
if [[ -n "$scenarios" ]]; then
	regression_env+=("ORACLE_REGRESSION_SCENARIOS=$scenarios")
fi
if ! "${regression_env[@]}" "$runner" >>"$run_dir/census.log" 2>&1; then
	# Non-zero exit alone doesn't decide the verdict: the summary counts do.
	:
fi

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
	ORACLE_REGRESSION_SCENARIOS="$recheck_names"
)
if ! "${recheck_env[@]}" "$runner" >"$run_dir/recheck.log" 2>&1; then
	:
fi
if [[ ! -s "$run_dir/recheck-results.tsv" ]]; then
	finish NOT_CLEAN 1
fi

recheck_bad=$(awk -F '\t' '$1 != "PASS" && $1 != "EXPECTED" && $1 != "EXPECTED_UNSTABLE" { print; exit }' "$run_dir/recheck-results.tsv")
if [[ -n "$recheck_bad" ]]; then
	finish NOT_CLEAN 1
fi
finish CLEAN_AFTER_RECHECK 0
