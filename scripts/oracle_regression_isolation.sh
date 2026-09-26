#!/usr/bin/env bash
# oracle_regression_isolation.sh — prove the per-worker isolation of the oracle
# differential fan-out BEFORE raising --workers beyond a previously proven level.
#
# The census is driven by xargs -P, so N scenarios run as N independent
# dp-oracle-diff processes. This script runs N harness processes concurrently
# and proves, from kernel state (/proc, not from the runs' own logs):
#
#   1. ports      — every worker's C oracle port, its WHOD port (oracle+1) and
#                   its Go telnet/HTTP ports are unique across workers;
#   2. data dirs  — every worker's disposable C lib copy (-d) is its own;
#   3. world dirs — every worker's Go world copy (-world) is its own;
#   4. databases  — every worker's store (-db) is its own file;
#   5. processes  — no engine process survives the runs;
#   6. scratch    — no /tmp/dp-oracle-diff-* directory survives the runs.
#
# (4) needs a scenario that carries a store, i.e. one with [relogin:*] or
# <RESTART>: every other scenario deliberately points the Go port at an
# unreachable DSN, so its -db is the same constant in every worker. That is why
# the default scenario list is a relogin/restart vehicle pair.
#
# Usage: oracle_regression_isolation.sh [--workers N] [scenario ...]
set -uo pipefail

repo_root=$(CDPATH= cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
go_bin=${ORACLE_REGRESSION_GO:-/usr/local/go/bin/go}
oracle_bin=${DP_ORACLE_BIN:-/home/zach/darkpawns-c-oracle/bin/circle}
workers=4
scenarios=()

while (($# > 0)); do
	case $1 in
	--workers)
		(($# >= 2)) || {
			printf 'isolation: --workers needs a count\n' >&2
			exit 2
		}
		workers=$2
		shift 2
		;;
	--workers=*)
		workers=${1#--workers=}
		shift
		;;
	*)
		scenarios+=("$1")
		shift
		;;
	esac
done
if ((${#scenarios[@]} == 0)); then
	scenarios=(lifecycle-quit-reenter restart-gossip-review)
fi
if [[ ! $workers =~ ^[1-9][0-9]*$ ]]; then
	printf 'isolation: --workers must be a positive integer: %s\n' "$workers" >&2
	exit 2
fi
for required in "$go_bin" "$oracle_bin"; do
	if [[ ! -x $required ]]; then
		printf 'isolation: not executable: %s\n' "$required" >&2
		exit 2
	fi
done

tmp=$(mktemp -d "${TMPDIR:-/tmp}/dp-isolation.XXXXXX")
harness_bin=$tmp/dp-oracle-isolation-harness
server_bin=$tmp/dp-oracle-isolation-server
cleanup() { rm -rf -- "$tmp"; }
trap cleanup EXIT

# Distinctive binary names: the /proc cmdline matching below keys on these exact
# paths, so no unrelated circle/darkpawns process can be mistaken for a worker.
if ! "$go_bin" build -C "$repo_root" -o "$harness_bin" ./cmd/dp-oracle-diff; then
	printf 'isolation: harness build failed\n' >&2
	exit 2
fi
if ! "$go_bin" build -C "$repo_root" -o "$server_bin" ./cmd/server; then
	printf 'isolation: server build failed\n' >&2
	exit 2
fi

printf 'isolation: %d workers, %d scenarios\n' "$workers" "${#scenarios[@]}"

# Baseline for the scratch-leak check: a killed earlier run can leave its own
# directory behind, and that is not this run's leak.
scratch_before=$(find "${TMPDIR:-/tmp}" -maxdepth 1 -name 'dp-oracle-diff-*' -type d 2>/dev/null | wc -l)

pids=()
logs=()
for ((i = 0; i < workers; i++)); do
	scenario=${scenarios[$((i % ${#scenarios[@]}))]}
	log=$tmp/worker$i.log
	logs+=("$log")
	(
		cd "$repo_root" || exit 125
		exec env DP_ORACLE_BIN="$oracle_bin" ORACLE_REGRESSION_SERVER="$server_bin" \
			"$harness_bin" --scenario "$scenario" --seed 1
	) >"$log" 2>&1 &
	pids+=($!)
done


# Sample kernel state while the fan-out is live.
declare -A seen_oracle_port seen_whod_port seen_go_port seen_data_dir seen_world seen_db
oracle_procs=0
go_procs=0
samples=0
while :; do
	live=0
	for pid in "${pids[@]}"; do
		if kill -0 "$pid" 2>/dev/null; then live=1; fi
	done
	((live)) || break
	samples=$((samples + 1))
	for cmdline in /proc/[0-9]*/cmdline; do
		[[ -r $cmdline ]] || continue
		mapfile -d '' -t argv <"$cmdline" 2>/dev/null || continue
		((${#argv[@]} > 0)) || continue
		case ${argv[0]} in
		"$oracle_bin")
			# argv: <oracle> -d <lib> <port>
			oracle_procs=$((oracle_procs + 1))
			dir=""
			port=""
			for ((k = 1; k < ${#argv[@]}; k++)); do
				case ${argv[$k]} in
				-d) dir=${argv[$((k + 1))]} ;;
				*) [[ ${argv[$k]} =~ ^[0-9]+$ ]] && port=${argv[$k]} ;;
				esac
			done
			seen_oracle_port[$port]=1
			seen_whod_port[$((port + 1))]=1
			[[ -n $dir ]] && seen_data_dir[$dir]=1
			;;
		"$server_bin")
			# argv: <server> -world <dir> -port <http> -telnet-port <telnet> -db <url>
			go_procs=$((go_procs + 1))
			world=""
			db=""
			http=""
			telnet=""
			for ((k = 1; k < ${#argv[@]}; k++)); do
				case ${argv[$k]} in
				-world) world=${argv[$((k + 1))]} ;;
				-db) db=${argv[$((k + 1))]} ;;
				-port) http=${argv[$((k + 1))]} ;;
				-telnet-port) telnet=${argv[$((k + 1))]} ;;
				esac
			done
			seen_go_port[$http]=1
			seen_go_port[$telnet]=1
			[[ -n $world ]] && seen_world[$world]=1
			[[ -n $db ]] && seen_db[$db]=1
			;;
		esac
	done
	sleep 0.2
done

for pid in "${pids[@]}"; do
	wait "$pid" || true
done

status=0
report() {
	printf '%-12s distinct=%-4s expected>=%-4s no-empty-key=%s\n' "$1" "$2" "$3" "$4"
	if (($2 < $3)) || [[ $4 != yes ]]; then
		status=1
	fi
}

distinct_count() {
	local -n map=$1
	local -A inverted=()
	local key
	for key in "${!map[@]}"; do
		[[ -n $key ]] && inverted[$key]=1
	done
	printf '%s' "${#inverted[@]}"
}

has_empty_key() {
	local -n map=$1
	local key
	for key in "${!map[@]}"; do
		[[ -z $key ]] && return 0
	done
	return 1
}

printf 'isolation: samples=%d oracle_observations=%d go_observations=%d\n' \
	"$samples" "$oracle_procs" "$go_procs"
if ((samples == 0 || oracle_procs < workers || go_procs < workers)); then
	printf 'isolation: FAIL — the sample loop did not observe %d oracle and %d Go processes\n' "$workers" "$workers" >&2
	status=1
fi

# 1. Ports: the C oracle port, its WHOD neighbour and the Go telnet/HTTP ports
# must not collide, and no sampled process may have an unreadable port.
ports_unique=yes
if has_empty_key seen_oracle_port || has_empty_key seen_go_port; then ports_unique=no; fi
all_ports=()
for p in "${!seen_oracle_port[@]}" "${!seen_whod_port[@]}" "${!seen_go_port[@]}"; do
	[[ -n $p ]] && all_ports+=("$p")
done
port_total=${#all_ports[@]}
port_distinct=$(printf '%s\n' "${all_ports[@]}" | sort -u | wc -l)
if ((port_total != port_distinct)); then ports_unique=no; fi
report "ports" "$port_distinct" "$((workers * 3))" "$ports_unique"

# 2-4. Data directories, world copies and stores.
for pair in "data-dirs seen_data_dir" "world-dirs seen_world" "databases seen_db"; do
	set -- $pair
	count=$(distinct_count "$2")
	unique=yes
	has_empty_key "$2" && unique=no
	report "$1" "$count" "$workers" "$unique"
done

# 5. Process cleanup: no engine process may outlive its worker.
sleep 0.5
survivors=0
for cmdline in /proc/[0-9]*/cmdline; do
	[[ -r $cmdline ]] || continue
	mapfile -d '' -t argv <"$cmdline" 2>/dev/null || continue
	((${#argv[@]} > 0)) || continue
	case ${argv[0]} in
	"$oracle_bin" | "$server_bin") survivors=$((survivors + 1)) ;;
	esac
done
printf '%-12s survivors=%-4d expected=0\n' "processes" "$survivors"
((survivors == 0)) || status=1

# 6. Scratch cleanup: the harness removes its own work directory on exit, so the
# count must return to its pre-run baseline.
scratch_after=$(find "${TMPDIR:-/tmp}" -maxdepth 1 -name 'dp-oracle-diff-*' -type d 2>/dev/null | wc -l)
scratch_leaked=$((scratch_after - scratch_before))
printf '%-12s leaked=%-4d (before=%d after=%d) expected=0\n' \
	"scratch-dirs" "$scratch_leaked" "$scratch_before" "$scratch_after"
((scratch_leaked <= 0)) || status=1

green=0
for log in "${logs[@]}"; do
	if grep -q 'normalized divergence detected' "$log"; then
		printf 'worker result: divergence (%s)\n' "$(basename "$log")"
	else
		green=$((green + 1))
	fi
done
printf 'isolation: %d/%d workers green\n' "$green" "${#logs[@]}"

if ((status == 0)); then
	printf 'isolation: OK — ports, data directories, databases and cleanup are per-worker\n'
else
	printf 'isolation: FAILED — do not raise --workers on this host yet\n' >&2
fi
exit "$status"

