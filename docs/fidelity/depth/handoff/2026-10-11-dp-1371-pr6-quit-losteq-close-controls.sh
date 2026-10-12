#!/usr/bin/env bash
# DP-1371 batch 6 — argument-level controls for the quit / LOSTEQ / socket-close
# mudlog proof:
#   pkg/game/quit_mudlog_test.go            (act.other.c:136 and :173)
#   pkg/session/comm_close_mudlog_test.go   (comm.c:2133)
#
# Each control mutates one argument of a producer call and must make the declared
# subtest fail on the declared assertion. Failures that are not that — a build
# error, an infrastructure failure (barrier/tap timeout, panic, race), a failure
# in a different test, or the declared test failing without the declared message —
# are rejected as CONTROL INVALID. Raw `go test` output per control is retained
# in OUT_DIR. The tree is restored on normal exit and on INT/TERM/HUP.
set -u

REPO=${REPO:-$(git rev-parse --show-toplevel 2>/dev/null)}
if [ -z "${REPO:-}" ] || [ ! -d "$REPO/pkg/session" ]; then
	echo "run from inside the repository, or set REPO=/path/to/repo" >&2
	exit 2
fi
cd "$REPO" || exit 2

OUT_DIR=${OUT_DIR:-${TMPDIR:-/tmp}/dp1371-pr6-controls}
mkdir -p "$OUT_DIR"

MUTABLE="pkg/game/quit.go pkg/session/manager.go"

restore() {
	for f in $MUTABLE; do
		if [ -f "$f.ctl6.bak" ]; then mv "$f.ctl6.bak" "$f"; fi
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
G = "pkg/game/quit.go"
S = "pkg/session/manager.go"
QUIT = 'fmt.Sprintf("%s has quit the game.", ch.GetName())'
LOST = 'fmt.Sprintf("LOSTEQ:%s has quit out of a save room.", ch.GetName())'
CLOSE = 'fmt.Sprintf("Closing link to: %s.", p.GetName())'

def patch_after(path, anchor, old, new):
    shutil.copyfile(path, path + ".ctl6.bak")
    s = open(path).read()
    i = s.index(anchor)
    j = s.index(old, i)
    open(path, "w").write(s[:j] + new + s[j + len(old):])

M = {
    # type NRM -> BRF: the LOG1-only observer starts receiving
    "quit-type": lambda: patch_after(G, QUIT, "MudlogNormal", "MudlogBrief"),
    "losteq-type": lambda: patch_after(G, LOST, "MudlogNormal", "MudlogBrief"),
    # The masking case: CMP is silent for a LOG2-only observer, so the LOSTEQ
    # line disappears while the sibling quit line still arrives. The receipt
    # assertion must be judged on LOSTEQ's own bytes or this passes.
    "losteq-cmp": lambda: patch_after(G, LOST, "MudlogNormal", "MudlogComplete"),
    "close-type": lambda: patch_after(S, CLOSE, "game.MudlogNormal", "game.MudlogBrief"),
    # level MAX(LVL_IMMORT, invis) -> LVL_IMMORT-1: the floor observer receives
    "quit-level": lambda: patch_after(G, QUIT, "max(lvlImmort, ch.GetInvisLevel())", "lvlImmort - 1"),
    "losteq-level": lambda: patch_after(G, LOST, "max(lvlImmort, ch.GetInvisLevel())", "lvlImmort - 1"),
    "close-level": lambda: patch_after(S, CLOSE, "max(game.LVL_IMMORT, p.GetInvisLevel())", "game.LVL_IMMORT - 1"),
    # invisibility ignored: the threshold drops to the floor
    "quit-invis": lambda: patch_after(G, QUIT, "max(lvlImmort, ch.GetInvisLevel())", "lvlImmort"),
    "losteq-invis": lambda: patch_after(G, LOST, "max(lvlImmort, ch.GetInvisLevel())", "lvlImmort"),
    "close-invis": lambda: patch_after(S, CLOSE, "max(game.LVL_IMMORT, p.GetInvisLevel())", "game.LVL_IMMORT"),
    # file TRUE -> FALSE: that producer's own file assertion must fail
    "quit-file": lambda: patch_after(G, QUIT, "max(lvlImmort, ch.GetInvisLevel()), true)", "max(lvlImmort, ch.GetInvisLevel()), false)"),
    "losteq-file": lambda: patch_after(G, LOST, "max(lvlImmort, ch.GetInvisLevel()), true)", "max(lvlImmort, ch.GetInvisLevel()), false)"),
    "close-file": lambda: patch_after(S, CLOSE, "max(game.LVL_IMMORT, p.GetInvisLevel()), true)", "max(game.LVL_IMMORT, p.GetInvisLevel()), false)"),
}
M[mid]()
print("mutated:", mid)
PY
}

ctl() { # pkg, id, test regex, expected failing subtest, expected assertion substring
	local pkg=$1 id=$2 test=$3 sub=$4 msg=$5
	local log="$OUT_DIR/$id.log"
	mutate "$id" || { echo "MUTATION FAILED: $id"; restore; return 1; }
	go test "$pkg" -run "$test" -count=1 >"$log" 2>&1
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
	quit-type) ctl ./pkg/game/ "$1" '^TestQuitMudlogTypeIsNRM$' 'TestQuitMudlogTypeIsNRM/quit_line' 'LOG1-only observer received' ;;
	losteq-type) ctl ./pkg/game/ "$1" '^TestQuitMudlogTypeIsNRM$' 'TestQuitMudlogTypeIsNRM/losteq_line' 'LOG1-only observer received' ;;
	losteq-cmp) ctl ./pkg/game/ "$1" '^TestQuitMudlogTypeIsNRM$' 'TestQuitMudlogTypeIsNRM/losteq_line' 'LOG2-only observer did not receive' ;;
	close-type) ctl ./pkg/session/ "$1" '^TestCloseSocketMudlogTypeIsNRM$' 'TestCloseSocketMudlogTypeIsNRM' 'LOG1-only observer received' ;;
	quit-level) ctl ./pkg/game/ "$1" '^TestQuitMudlogLevelDiscriminators$' 'TestQuitMudlogLevelDiscriminators/quit_line/lv_immort_floor' 'level-30 observer received' ;;
	losteq-level) ctl ./pkg/game/ "$1" '^TestQuitMudlogLevelDiscriminators$' 'TestQuitMudlogLevelDiscriminators/losteq_line/lv_immort_floor' 'level-30 observer received' ;;
	close-level) ctl ./pkg/session/ "$1" '^TestCloseSocketMudlogLevelDiscriminators$' 'TestCloseSocketMudlogLevelDiscriminators/lv_immort_floor' 'level-30 observer received' ;;
	quit-invis) ctl ./pkg/game/ "$1" '^TestQuitMudlogLevelDiscriminators$' 'TestQuitMudlogLevelDiscriminators/quit_line/invis_raises_threshold' 'level-35 observer received' ;;
	losteq-invis) ctl ./pkg/game/ "$1" '^TestQuitMudlogLevelDiscriminators$' 'TestQuitMudlogLevelDiscriminators/losteq_line/invis_raises_threshold' 'level-35 observer received' ;;
	close-invis) ctl ./pkg/session/ "$1" '^TestCloseSocketMudlogLevelDiscriminators$' 'TestCloseSocketMudlogLevelDiscriminators/invis_raises_threshold' 'level-35 observer received' ;;
	# file controls: each names its own producer's file assertion
	quit-file) ctl ./pkg/game/ "$1" '^TestQuitMudlogProducers$' 'TestQuitMudlogProducers/quit_line_from_a_temple' 'file payload count' ;;
	losteq-file) ctl ./pkg/game/ "$1" '^TestQuitMudlogProducers$' 'TestQuitMudlogProducers/losteq_line_from_an_unsafe_room' 'file LOSTEQ count' ;;
	close-file) ctl ./pkg/session/ "$1" '^TestCloseSocketMudlogProducer$' 'TestCloseSocketMudlogProducer' 'file payload count' ;;
	*) echo "unknown control: $1" >&2; return 2 ;;
	esac
}

if [ "${1:-}" = "--only" ] && [ -n "${2:-}" ]; then
	run_one "$2"
	exit $?
fi

rc=0
for id in quit-type losteq-type losteq-cmp close-type quit-level losteq-level \
	close-level quit-invis losteq-invis close-invis quit-file losteq-file \
	close-file; do
	run_one "$id" || rc=1
done
echo "raw control output retained in $OUT_DIR"
exit $rc
