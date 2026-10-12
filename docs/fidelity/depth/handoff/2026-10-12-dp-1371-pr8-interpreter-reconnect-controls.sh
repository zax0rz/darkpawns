#!/usr/bin/env bash
# DP-1371 batch 8 — argument-level controls for the interpreter.c reconnect /
# connect / new-player proof (pkg/session/interpreter_batch8_mudlog_test.go).
#
# Each control mutates one argument of one producer call and must make the
# declared subtest fail on the declared assertion. Build failures,
# infrastructure failures, other-test failures, and a declared test failing
# without the declared message are rejected as CONTROL INVALID.
set -u

REPO=${REPO:-$(git rev-parse --show-toplevel 2>/dev/null)}
if [ -z "${REPO:-}" ] || [ ! -d "$REPO/pkg/session" ]; then
	echo "run from inside the repository, or set REPO=/path/to/repo" >&2
	exit 2
fi
cd "$REPO" || exit 2

OUT_DIR=${OUT_DIR:-${TMPDIR:-/tmp}/dp1371-pr8-controls}
mkdir -p "$OUT_DIR"

MUTABLE="pkg/session/reconnect.go pkg/session/session_login.go pkg/session/char_creation.go"

restore() {
	for f in $MUTABLE; do
		if [ -f "$f.ctl8.bak" ]; then mv "$f.ctl8.bak" "$f"; fi
	done
}

on_signal() {
	echo "control run interrupted by signal — restoring the tree" >&2
	exit "$1"
}
trap 'on_signal 130' INT
trap 'on_signal 143' TERM
trap 'on_signal 129' HUP
trap restore EXIT

mutate() { # $1 = mutation id
	python3 - "$1" <<'PY'
import shutil, sys
mid = sys.argv[1]
RC = "pkg/session/reconnect.go"
SL = "pkg/session/session_login.go"
CC = "pkg/session/char_creation.go"
# The "has reconnected." payload appears twice in finishDupeCheck: the unswitch
# arm (first) and the linkdead/default arm (second, preceded by its Act).
UNSWITCH = 'fmt.Sprintf("%s [%s] has reconnected.", name, s.MudHost())'
LINKDEAD = 'game.Act(w, true, p, nil, nil, nil, "$n has reconnected.", "", game.ToRoom)'
USURP = 'fmt.Sprintf("%s has re-logged in ... disconnecting old socket.", name)'
CONNECT = 'fmt.Sprintf("%s [%s] has connected.", s.player.GetName(), s.MudHost())'
NEWPLAYER = 'fmt.Sprintf("%s [%s] new player.", s.charName, s.MudHost())'

def patch_after(path, anchor, old, new):
    shutil.copyfile(path, path + ".ctl8.bak")
    s = open(path).read()
    i = s.index(anchor)
    j = s.index(old, i)
    open(path, "w").write(s[:j] + new + s[j + len(old):])

M = {
    # reconnect: linkdead-resume arm (interpreter.c:1641)
    "recon-linkdead-type": lambda: patch_after(RC, LINKDEAD, "game.MudlogNormal", "game.MudlogBrief"),
    "recon-linkdead-level": lambda: patch_after(RC, LINKDEAD, "max(game.LVL_IMMORT, p.GetInvisLevel())", "game.LVL_IMMORT - 1"),
    "recon-linkdead-file": lambda: patch_after(RC, LINKDEAD, "max(game.LVL_IMMORT, p.GetInvisLevel()), true)", "max(game.LVL_IMMORT, p.GetInvisLevel()), false)"),
    # reconnect: usurp arm (interpreter.c:1650)
    "recon-usurp-type": lambda: patch_after(RC, USURP, "game.MudlogNormal", "game.MudlogBrief"),
    "recon-usurp-file": lambda: patch_after(RC, USURP, "max(game.LVL_IMMORT, p.GetInvisLevel()), true)", "max(game.LVL_IMMORT, p.GetInvisLevel()), false)"),
    # reconnect: unswitch arm (interpreter.c:1655)
    "recon-unswitch-level": lambda: patch_after(RC, UNSWITCH, "max(game.LVL_IMMORT, p.GetInvisLevel())", "game.LVL_IMMORT - 1"),
    "recon-unswitch-file": lambda: patch_after(RC, UNSWITCH, "max(game.LVL_IMMORT, p.GetInvisLevel()), true)", "max(game.LVL_IMMORT, p.GetInvisLevel()), false)"),
    # login connect (interpreter.c:1927): BRF at MAX(LVL_IMMORT, invis)
    "connect-type": lambda: patch_after(SL, CONNECT, "game.MudlogBrief", "game.MudlogNormal"),
    "connect-level": lambda: patch_after(SL, CONNECT, "max(game.LVL_IMMORT, s.player.GetInvisLevel())", "game.LVL_IMMORT - 1"),
    "connect-file": lambda: patch_after(SL, CONNECT, "max(game.LVL_IMMORT, s.player.GetInvisLevel()), true)", "max(game.LVL_IMMORT, s.player.GetInvisLevel()), false)"),
    # new player (interpreter.c:2155): NRM at a fixed LVL_IMMORT
    "newplayer-type": lambda: patch_after(CC, NEWPLAYER, "game.MudlogNormal", "game.MudlogBrief"),
    "newplayer-level": lambda: patch_after(CC, NEWPLAYER, "game.LVL_IMMORT, true)", "game.LVL_IMMORT - 1, true)"),
    "newplayer-file": lambda: patch_after(CC, NEWPLAYER, "game.LVL_IMMORT, true)", "game.LVL_IMMORT, false)"),
}
M[mid]()
print("mutated:", mid)
PY
}

ctl() { # id, test regex, expected failing subtest, expected assertion substring
	local id=$1 test=$2 sub=$3 msg=$4
	local log="$OUT_DIR/$id.log"
	mutate "$id" || { echo "MUTATION FAILED: $id"; restore; return 1; }
	go test ./pkg/session/ -run "$test" -count=1 >"$log" 2>&1
	local rc=$?
	restore

	if grep -qE '\[build failed\]|^# github\.com/|undefined:|cannot use|syntax error' "$log"; then
		echo "CONTROL INVALID: $id — build/compile failure, not a discriminator failure ($log)"
		return 1
	fi
	if grep -qE 'FIFO barrier|no tap event for|panic:|DATA RACE|timed out after' "$log"; then
		echo "CONTROL INVALID: $id — infrastructure failure (barrier/event timeout, panic or race) ($log)"
		return 1
	fi
	if [ "$rc" -eq 0 ]; then
		echo "CONTROL BROKEN: $id — test passed after mutation ($log)"
		return 1
	fi
	if ! grep -qF -- "--- FAIL: $sub" "$log"; then
		echo "CONTROL INVALID: $id — expected subtest '$sub' did not fail; other failure ($log)"
		return 1
	fi
	if ! grep -qF -- "$msg" "$log"; then
		echo "CONTROL INVALID: $id — '$sub' failed without the expected assertion '$msg' ($log)"
		return 1
	fi
	echo "control ok: $id — $sub failed on '$msg' (output: $log)"
}

run_one() {
	case "$1" in
	recon-linkdead-type) ctl "$1" '^TestReconnectMudlogProducers$' 'TestReconnectMudlogProducers/linkdead_resume_1641' 'LOG1-only observer received' ;;
	recon-linkdead-level) ctl "$1" '^TestReconnectMudlogLevelDiscriminators$' 'TestReconnectMudlogLevelDiscriminators/linkdead_1641/lv_immort_floor' 'level-30 observer received' ;;
	recon-linkdead-file) ctl "$1" '^TestReconnectMudlogProducers$' 'TestReconnectMudlogProducers/linkdead_resume_1641' 'file payload count' ;;
	recon-usurp-type) ctl "$1" '^TestReconnectMudlogProducers$' 'TestReconnectMudlogProducers/usurp_1650' 'LOG1-only observer received' ;;
	recon-usurp-file) ctl "$1" '^TestReconnectMudlogProducers$' 'TestReconnectMudlogProducers/usurp_1650' 'file payload count' ;;
	recon-unswitch-level) ctl "$1" '^TestReconnectMudlogLevelDiscriminators$' 'TestReconnectMudlogLevelDiscriminators/unswitch_1655/lv_immort_floor' 'level-30 observer received' ;;
	recon-unswitch-file) ctl "$1" '^TestReconnectMudlogProducers$' 'TestReconnectMudlogProducers/unswitch_1655' 'file payload count' ;;
	connect-type) ctl "$1" '^TestLoginConnectMudlog$' 'TestLoginConnectMudlog' 'LOG1-only observer did not receive' ;;
	connect-level) ctl "$1" '^TestLoginConnectMudlog$' 'TestLoginConnectMudlog' 'level-30 observer received' ;;
	connect-file) ctl "$1" '^TestLoginConnectMudlog$' 'TestLoginConnectMudlog' 'file payload count' ;;
	newplayer-type) ctl "$1" '^TestNewPlayerMudlog$' 'TestNewPlayerMudlog' 'LOG1-only observer received' ;;
	newplayer-level) ctl "$1" '^TestNewPlayerMudlog$' 'TestNewPlayerMudlog' 'level-30 observer received' ;;
	newplayer-file) ctl "$1" '^TestNewPlayerMudlog$' 'TestNewPlayerMudlog' 'file payload count' ;;
	*) echo "unknown control: $1" >&2; return 2 ;;
	esac
}

if [ "${1:-}" = "--only" ] && [ -n "${2:-}" ]; then
	run_one "$2"
	exit $?
fi

rc=0
for id in recon-linkdead-type recon-linkdead-level recon-linkdead-file \
	recon-usurp-type recon-usurp-file recon-unswitch-level recon-unswitch-file \
	connect-type connect-level connect-file \
	newplayer-type newplayer-level newplayer-file; do
	run_one "$id" || rc=1
done
echo "raw control output retained in $OUT_DIR"
exit $rc
