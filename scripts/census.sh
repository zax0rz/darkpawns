#!/usr/bin/env bash
# census.sh — start a census, leave it running, and read one line.
#
#   scripts/census.sh start --name <run-name> [--claims | --scenarios a,b,c] [--jobs N]
#   scripts/census.sh wait   [--run <dir>] [--max-seconds N]
#   scripts/census.sh status [--run <dir>]
#
# start returns within seconds (detached via setsid); wait blocks up to
# --max-seconds (default 540) and prints exactly one line; status prints the
# same line immediately. Exit codes shared by wait/status:
#   0  done, CLEAN or CLEAN_AFTER_RECHECK
#   1  done, NOT_CLEAN
#   3  still running when max-seconds hit (call wait again)
#   4  died / nothing to wait on
# start-specific: 2 bad usage / run dir exists, 5 another census holds the
# lock, 6 reference-oracle mismatch.
#
# The runner is a seam (CENSUS_RUNNER) so the tests can drive stubs.
set -u

repo_root=$(CDPATH= cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
lock_file="${XDG_RUNTIME_DIR:-/tmp}/dp-census.lock"
poll_seconds=${CENSUS_POLL_SECONDS:-15}
last_run_file="${XDG_RUNTIME_DIR:-/tmp}/dp-census.last"
runs_root=${ORACLE_RUNS_ROOT:-$HOME/Archives/darkpawns/oracle-runs}
runner=${CENSUS_RUNNER:-$repo_root/scripts/oracle_regression.sh}
reference_file=$repo_root/cmd/dp-oracle-diff/reference-oracle.sha256
oracle_bin=${DP_ORACLE_BIN:-/home/zach/darkpawns-c-oracle/bin/circle}
seed=${ORACLE_REGRESSION_SEED:-1}

die() {
	printf 'census: %s\n' "$1" >&2
	exit "${2:-2}"
}

usage() {
	printf 'usage: census.sh start --name <run> [--claims | --scenarios a,b,c] [--jobs N]\n' \
		'       census.sh wait [--run <dir>] [--max-seconds N]\n' \
		'       census.sh status [--run <dir>]\n' "${0##*/}" >&2
	exit 2
}

# ---------------------------------------------------------------------------
# Shared one-line reporting for wait/status.
# ---------------------------------------------------------------------------
report_run() {
	local run_dir=$1 max_seconds=$2
	local summary state pid started elapsed done total
	if [[ ! -d "$run_dir" ]]; then
		printf 'no census %s\n' "$run_dir"
		return 4
	fi
	if [[ -s "$run_dir/summary.txt" ]]; then
		local line verdict
		line=$(cat -- "$run_dir/summary.txt")
		verdict=$(printf '%s\n' "$line" | sed -n 's/.* verdict=\([A-Z_]*\).*/\1/p')
		printf '%s\n' "$line"
		[[ "$verdict" == CLEAN || "$verdict" == CLEAN_AFTER_RECHECK ]]
		return
	fi
	state=$(cat -- "$run_dir/state" 2>/dev/null || printf 'unknown')
	pid=$(printf '%s\n' "$state" | sed -n 's/^running pid=\([0-9]*\).*/\1/p')
	if [[ -n "$pid" ]] && ! kill -0 "$pid" 2>/dev/null; then
		printf 'died %s\n' "$run_dir"
		return 4
	fi
	if [[ "$state" != running* ]]; then
		# state was written to done but no summary followed: treat as died.
		printf 'died %s\n' "$run_dir"
		return 4
	fi
	if ((max_seconds <= 0)); then
		started=$(printf '%s\n' "$state" | sed -n 's/.*started=\(.*\)$/\1/p')
		elapsed=$(( $(date +%s) - $(date -d "$started" +%s) ))
		total=$(cat -- "$run_dir/total" 2>/dev/null || printf '?')
		if [[ -s "$run_dir/claims.tsv" ]]; then
			local claim_seed claim_done claim_total
			IFS=$'\t' read -r claim_seed claim_done claim_total <"$run_dir/claims-progress" 2>/dev/null || true
			printf 'running %ss seed %s %s/%s\n' "$elapsed" "${claim_seed:-pending}" "${claim_done:-0}" "${claim_total:-$total}"
		else
		done=$(grep -cE '^(PASS|FAIL|EXPECTED|EXPECTED_UNSTABLE|UNPINNABLE|STALE|TIMEOUT|INFRA|RETRY) ' "$run_dir/census.log" 2>/dev/null || printf 0)
		printf 'running %ss %s/%s\n' "$elapsed" "$done" "$total"
		fi
		return 3
	fi
	local deadline=$(( $(date +%s) + max_seconds ))
	while (( $(date +%s) < deadline )); do
		if [[ -s "$run_dir/summary.txt" ]]; then
			report_run "$run_dir" 0
			return
		fi
		pid=$(cat -- "$run_dir/pid" 2>/dev/null || printf 0)
		if ! kill -0 "$pid" 2>/dev/null; then
			printf 'died %s\n' "$run_dir"
			return 4
		fi
		sleep "$poll_seconds"
	done
	# One final check after the deadline.
	if [[ -s "$run_dir/summary.txt" ]]; then
		report_run "$run_dir" 0
		return
	fi
	if ! kill -0 "$(cat -- "$run_dir/pid" 2>/dev/null || printf 0)" 2>/dev/null; then
		printf 'died %s\n' "$run_dir"
		return 4
	fi
	report_run "$run_dir" 0
	return 3
}

resolve_run() {
	local requested=${1:-}
	if [[ -n "$requested" ]]; then
		printf '%s\n' "$requested"
		return
	fi
	# The run holding the lock, else the last one started, else the newest
	# run dir. The lock body names the active run; after the runner
	# finishes it is emptied. The last-run pointer is written by start, so
	# a finished run resolves even after its lock is gone — alphabetical
	# order would report a *different* run's result. The mtime fallback
	# covers a pointer lost to a reboot and ignores hand-made directories
	# without a state file.
	if [[ -s "$lock_file" ]]; then
		local lock_run
		lock_run=$(sed -n 's/^[0-9]*: //p' "$lock_file")
		if [[ -d "$lock_run" ]]; then
			printf '%s\n' "$lock_run"
			return
		fi
	fi
	if [[ -s "$last_run_file" ]]; then
		local last_run
		last_run=$(cat -- "$last_run_file")
		if [[ -d "$last_run" ]]; then
			printf '%s\n' "$last_run"
			return
		fi
	fi
	local best best_mtime=-1 mtime
	while IFS= read -r -d '' state; do
		mtime=$(stat -c %Y -- "$state" 2>/dev/null) || continue
		if ((mtime > best_mtime)); then
			best_mtime=$mtime
			best=$(dirname -- "$state")
		fi
	done < <(find "$runs_root" -mindepth 3 -maxdepth 3 -name state -type f -print0 2>/dev/null)
	[[ -n "${best:-}" ]] && printf '%s\n' "$best"
}

# ---------------------------------------------------------------------------
# start
# ---------------------------------------------------------------------------
do_start() {
	local name= scenarios= jobs=36 claims=0
	while (($# > 0)); do
		case $1 in
		--claims) claims=1; shift ;;
		--name) (($# >= 2)) || usage; name=$2; shift 2 ;;
		--name=*) name=${1#--name=}; shift ;;
		--scenarios) (($# >= 2)) || usage; scenarios=$2; shift 2 ;;
		--scenarios=*) scenarios=${1#--scenarios=}; shift ;;
		--jobs) (($# >= 2)) || usage; jobs=$2; shift 2 ;;
		--jobs=*) jobs=${1#--jobs=}; shift ;;
		*) usage ;;
		esac
	done
	[[ -n "$name" ]] || usage
	[[ "$claims" == 0 || -z "$scenarios" ]] || die "--claims and --scenarios are mutually exclusive"
	[[ "$name" != */* ]] || die "run name must not contain '/': $name"
	command -v flock >/dev/null 2>&1 || die "flock(1) is required"
	command -v setsid >/dev/null 2>&1 || die "setsid(1) is required"

	# 1. Lock. FD 9 stays open for the detached runner's lifetime. Append
	# mode: > would truncate the body before the stale check below reads it.
	exec 9>>"$lock_file"
	if ! flock -n 9; then
		local lock_pid lock_run
		lock_pid=$(sed -n 's/^\([0-9]*\):.*/\1/p' "$lock_file")
		lock_run=$(sed -n 's/^[0-9]*: //p' "$lock_file")
		if [[ -n "$lock_pid" ]] && ! kill -0 "$lock_pid" 2>/dev/null; then
			# The named holder is dead but the flock is still taken: an FD
			# inherited by an orphaned child. Wait at most 5s for it, then
			# refuse — never block without a limit.
			if flock -w 5 9; then
				printf 'census: stale lock (pid %s dead) — taken over\n' "$lock_pid" >&2
			else
				printf 'census: lock still held after the pid %s died; holders:\n' "$lock_pid" >&2
				"$repo_root/scripts/census_lock_holders.sh" >&2 || true
				exit 5
			fi
		else
			printf 'census: another census is running: %s\n' "$lock_run"
			exit 5
		fi
	else
		# The flock was free. A body naming a dead PID (or a finished run)
		# is a stale leftover from a killed runner: take it over.
		local lock_pid lock_run
		lock_pid=$(sed -n 's/^\([0-9]*\):.*/\1/p' "$lock_file")
		lock_run=$(sed -n 's/^[0-9]*: //p' "$lock_file")
		if [[ -n "$lock_pid" ]] && ! kill -0 "$lock_pid" 2>/dev/null 			&& [[ -n "$lock_run" ]] && [[ ! -s "$lock_run/summary.txt" ]]; then
			printf 'census: stale lock (pid %s dead) — taking over\n' "$lock_pid" >&2
		fi
	fi

	# 2. Reference check.
	local oracle_sha allow_note=
	if [[ -x "$oracle_bin" ]]; then
		oracle_sha=$(sha256sum -- "$oracle_bin" | cut -d' ' -f1)
	else
		die "oracle binary is not executable: $oracle_bin" 2
	fi
	if [[ "${CENSUS_ALLOW_NONREFERENCE:-}" != 1 ]]; then
		local reference
		reference=$(cat -- "$reference_file" 2>/dev/null)
		[[ -n "$reference" ]] || die "missing $reference_file"
		[[ "$oracle_sha" == "$reference" ]] || {
			printf 'census: oracle sha256 %s is not the reference %s\n' "$oracle_sha" "$reference" >&2
			printf 'census: set CENSUS_ALLOW_NONREFERENCE=1 to proceed anyway\n' >&2
			exit 6
		}
	else
		allow_note=' (NOT THE REFERENCE ORACLE)'
	fi

	# 3. Run directory.
	local run_dir
	run_dir="$runs_root/$(date +%F)/$name"
	if [[ -e "$run_dir" ]] && [[ -n "$(ls -A -- "$run_dir" 2>/dev/null)" ]]; then
		die "run directory exists and is not empty: $run_dir"
	fi
	mkdir -p -- "$run_dir" || die "cannot create $run_dir"
	git -C "$repo_root" rev-parse HEAD >"$run_dir/go-head.txt" 2>/dev/null || printf 'unknown\n' >"$run_dir/go-head.txt"
	git -C "$repo_root" status --porcelain >"$run_dir/git-status.txt" 2>/dev/null || printf 'unknown\n' >"$run_dir/git-status.txt"

	if [[ "$claims" == 1 ]]; then
		local claims_args=()
		[[ -z "${CENSUS_MANIFEST_DIR:-}" ]] || claims_args+=(--manifest-dir "$CENSUS_MANIFEST_DIR" --root "$CENSUS_MANIFEST_DIR")
		python3 "$repo_root/scripts/manifest_claims.py" "${claims_args[@]}" >"$run_dir/claims.tsv" || die "cannot enumerate manifest claims"
		[[ -s "$run_dir/claims.tsv" ]] || die "no oracle claims"
		cp "$run_dir/go-head.txt" "$run_dir/manifests-head.txt"
	fi

	# Worker count for a targeted run: the smaller of 36 and the count.
	local effective_jobs=$jobs
	if [[ -n "$scenarios" ]]; then
		local count
		count=$(printf '%s\n' "$scenarios" | tr ',' '\n' | grep -c .)
		((count < effective_jobs)) && effective_jobs=$count
	fi
	local total
	total=$(printf '%s\n' "$scenarios" | tr ',' '\n' | grep -c .)
	[[ -n "$scenarios" ]] || total=$(find "$repo_root/cmd/dp-oracle-diff/scenarios" -maxdepth 1 -type f -name '*.txt' | wc -l)
	[[ "$claims" == 0 ]] || total=$(wc -l <"$run_dir/claims.tsv")
	printf '%s\n' "$total" >"$run_dir/total"

	# 4. Launch detached. The runner keeps FD 9 (the lock) and releases it in
	# its EXIT trap, so the lock lives exactly as long as the census.
	local pid
	setsid nohup env \
		CENSUS_RUN_DIR="$run_dir" \
		CENSUS_RUNNER="$runner" \
		CENSUS_REPO_ROOT="$repo_root" \
		CENSUS_ORACLE_BIN="$oracle_bin" \
		CENSUS_ORACLE_SHA="$oracle_sha" \
		CENSUS_ALLOW_NOTE="$allow_note" \
		CENSUS_SCENARIOS="$scenarios" \
		CENSUS_JOBS="$effective_jobs" \
		CENSUS_SEED="$seed" \
		CENSUS_CLAIMS="$claims" \
		CENSUS_LOCK_FD=9 \
		CENSUS_TEST_WORK="${CENSUS_TEST_WORK:-}" \
		bash "$repo_root/scripts/census_runner.sh" </dev/null >"$run_dir/census.log" 2>&1 &
	pid=$!
	printf '%s\n' "$pid" >"$run_dir/pid"

	# 5. State + lock body + last-run pointer (wait/status resolve the
	# finished run through it once the lock is gone).
	printf 'running pid=%s started=%s\n' "$pid" "$(date '+%Y-%m-%dT%H:%M:%S%z')" >"$run_dir/state"
	printf '%s: %s\n' "$pid" "$run_dir" >"$lock_file"
	printf '%s\n' "$run_dir" >"$last_run_file"

	printf 'census started: %s\n' "$run_dir"
	exit 0
}

# ---------------------------------------------------------------------------
# wait / status
# ---------------------------------------------------------------------------
do_wait() {
	local run= max_seconds=540
	while (($# > 0)); do
		case $1 in
		--run) (($# >= 2)) || usage; run=$2; shift 2 ;;
		--run=*) run=${1#--run=}; shift ;;
		--max-seconds) (($# >= 2)) || usage; max_seconds=$2; shift 2 ;;
		--max-seconds=*) max_seconds=${1#--max-seconds=}; shift ;;
		*) usage ;;
		esac
	done
	[[ "$max_seconds" =~ ^[0-9]+$ ]] || die "--max-seconds needs a number"
	local run_dir
	run_dir=$(resolve_run "$run")
	[[ -n "$run_dir" ]] || {
		printf 'no census\n'
		exit 4
	}
	report_run "$run_dir" "$max_seconds"
}

do_status() {
	local run=
	while (($# > 0)); do
		case $1 in
		--run) (($# >= 2)) || usage; run=$2; shift 2 ;;
		--run=*) run=${1#--run=}; shift ;;
		*) usage ;;
		esac
	done
	local run_dir
	run_dir=$(resolve_run "$run")
	[[ -n "$run_dir" ]] || {
		printf 'no census\n'
		exit 4
	}
	report_run "$run_dir" 0
}

case ${1:-} in
start) shift; do_start "$@" ;;
wait) shift; do_wait "$@" ;;
status) shift; do_status "$@" ;;
*) usage ;;
esac
