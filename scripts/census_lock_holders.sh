#!/usr/bin/env bash
# census_lock_holders.sh — print the processes still holding the census
# lock file open. Called by census.sh start when the lock stays taken after
# the recorded PID died (an orphaned child inherited the FD). One find(1)
# pass matches the FD links in-kernel (a per-FD readlink fork loop took
# 8+s on this machine — the very hang this script exists to prevent).
# Best effort; never run by hand.
set -u

lock_file="${1:-${XDG_RUNTIME_DIR:-/tmp}/dp-census.lock}"

[[ -r /proc ]] || exit 0
lock_real=$(readlink -f -- "$lock_file" 2>/dev/null)
[[ -n "$lock_real" ]] || exit 0

found=0
while IFS= read -r fd; do
	pid=${fd#/proc/}
	pid=${pid%%/*}
	cmdline=$(tr '\0' ' ' <"/proc/$pid/cmdline" 2>/dev/null | cut -c1-120)
	start=$(stat -c %y "/proc/$pid" 2>/dev/null | cut -d. -f1)
	printf '  pid %s (%s): %s\n' "$pid" "$start" "${cmdline:-unknown}"
	found=1
done < <(find /proc/[0-9]*/fd -maxdepth 1 -lname "$lock_real" 2>/dev/null)
((found == 0)) && printf '  (no process has the lock file open; the flock may be on a deleted inode)\n'

exit 0
