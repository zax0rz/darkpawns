#!/usr/bin/env bash
set -uo pipefail

# Run the complete real-oracle census from a disposable checkout of origin/main.
# This keeps generated lib/data/world_state.json and other runtime files out of
# the developer checkout, which is important for scenarios without fixtures.

repo_root=$(CDPATH= cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
results_root=${DP_NIGHTLY_RESULTS_DIR:-/home/zach/dp-census-nightly}
go_bin=${ORACLE_REGRESSION_GO:-/usr/local/go/bin/go}
oracle_bin=${DP_ORACLE_BIN:-/home/zach/darkpawns-c-oracle/bin/circle}
run_id=$(date '+%Y%m%dT%H%M%S%z')-$$
run_dir="$results_root/$run_id"
worktree="$run_dir/main"
log_file="$run_dir/oracle-regression.log"
summary_file="$results_root/exit-codes.log"

mkdir -p -- "$run_dir/tmp"
exec > >(tee -a "$log_file") 2>&1

run_status=0
cleanup_status=0
main_revision=unknown

printf 'census-nightly: started=%s repo=%s\n' "$(date --iso-8601=seconds)" "$repo_root"

if ! git -C "$repo_root" fetch --quiet origin main; then
	printf 'census-nightly: failed to fetch origin/main\n'
	run_status=2
fi

if (( run_status == 0 )); then
	if ! main_revision=$(git -C "$repo_root" rev-parse --verify refs/remotes/origin/main); then
		printf 'census-nightly: origin/main is unavailable\n'
		run_status=2
	fi
fi

if (( run_status == 0 )); then
	printf 'census-nightly: revision=%s\n' "$main_revision"
	if ! git -C "$repo_root" worktree add --detach "$worktree" "$main_revision"; then
		printf 'census-nightly: failed to create disposable main worktree\n'
		run_status=2
	fi
fi

if (( run_status == 0 )); then
	(
		cd "$worktree" || exit 2
		TMPDIR="$run_dir/tmp" \
		DP_ORACLE_BIN="$oracle_bin" \
		ORACLE_REGRESSION_GO="$go_bin" \
		make oracle-regression
	)
	run_status=$?
	printf 'census-nightly: oracle-regression exit=%d\n' "$run_status"
fi

if [[ -d "$worktree" ]]; then
	if ! git -C "$repo_root" worktree remove --force "$worktree"; then
		printf 'census-nightly: failed to remove disposable worktree\n'
		cleanup_status=2
	fi
fi

if (( run_status == 0 && cleanup_status != 0 )); then
	run_status=$cleanup_status
fi

finished=$(date --iso-8601=seconds)
printf 'census-nightly: finished=%s overall-exit=%d\n' "$finished" "$run_status"
mkdir -p -- "$results_root"
printf '%s\trevision=%s\toracle-regression=%d\tcleanup=%d\toverall=%d\tresults=%s\n' \
	"$finished" "$main_revision" "$run_status" "$cleanup_status" "$run_status" "$run_dir" >> "$summary_file"

exit "$run_status"
