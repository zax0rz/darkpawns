#!/usr/bin/env bash
# DP-1371 PR 5 — argument-level discriminator controls for the wizutil/skillset
# mudlog proof (pkg/session/wizutil_mudlog_discriminators_test.go).
#
# Companion to docs/fidelity/depth/handoff/2026-10-05-dp-1371-mudlog-wizutil-controls.py,
# which removes each producer wholesale and proves the producer calls exist. This
# script mutates a *single argument* of an existing call and proves the new
# discriminator assertions are not vacuous:
#   type:  BRF -> NRM  (the LOG1-only observer must stop receiving)
#   level: MAX(LVL_GOD, invis) -> 33  (the LVL_GOD-1 observer must start receiving)
#   invis: the shared helper stops applying MAX(LVL_GOD, invis)
#   tap:   the skillset call's BRF / -1 / true metadata, read through the feed
#
# R5h: a green must be able to fail. Each control applies one mutation, runs one
# test, and requires FAIL; the tree is restored afterwards, including on
# interrupt (EXIT/INT/TERM/HUP traps below).
set -u

REPO=${REPO:-$(git rev-parse --show-toplevel 2>/dev/null)}
if [ -z "${REPO:-}" ] || [ ! -d "$REPO/pkg/session" ]; then
	echo "run from inside the repository, or set REPO=/path/to/repo" >&2
	exit 2
fi
cd "$REPO" || exit 2

# Each control's raw `go test` output is retained beside the summary: the summary
# alone cannot show *why* a test failed, and a control is only valid when the
# failure is the expected discriminator assertion. OUT_DIR defaults to a temp dir;
# the release evidence copy lives with the census run.
OUT_DIR=${OUT_DIR:-${TMPDIR:-/tmp}/dp1371-pr5-controls}
mkdir -p "$OUT_DIR"

# Files this script may mutate. restore() is idempotent and safe to call at any
# point, including twice.
MUTABLE="pkg/session/wiz_system.go pkg/session/wiz_player.go"

restore() {
	for f in $MUTABLE; do
		if [ -f "$f.ctl5.bak" ]; then mv "$f.ctl5.bak" "$f"; fi
	done
}

# Interruption safety: a mutation must never outlive the script. The EXIT trap
# covers normal exit, `exit` inside the script, and any shell error path; the
# signal traps print a marker so an interrupted run is distinguishable from one
# that finished its control, then exit so the EXIT trap restores the tree.
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
W = "pkg/session/wiz_system.go"
P = "pkg/session/wiz_player.go"

def patch_after(path, anchor, old, new):
    shutil.copyfile(path, path + ".ctl5.bak")
    s = open(path).read()
    i = s.index(anchor)
    j = s.index(old, i)
    open(path, "w").write(s[:j] + new + s[j + len(old):])

M = {
    # type BRF -> NRM: the LOG1-only observer stops receiving
    "squelch-type": lambda: patch_after(W, "(GC) Squelch %s for %s by %s.", "game.MudlogBrief", "game.MudlogNormal"),
    "freeze-type": lambda: patch_after(W, "(GC) %s frozen by %s.", "game.MudlogBrief", "game.MudlogNormal"),
    "thaw-type": lambda: patch_after(W, "(GC) %s un-frozen by %s.", "game.MudlogBrief", "game.MudlogNormal"),
    # level MAX(LVL_GOD, invis) -> 33: the LVL_GOD-1 observer starts receiving
    "notitle-level": lambda: patch_after(W, "(GC) Notitle %s for %s by %s.", "wizutilMudlogLevel(s.player)", "33"),
    "squelch-level": lambda: patch_after(W, "(GC) Squelch %s for %s by %s.", "wizutilMudlogLevel(s.player)", "33"),
    "freeze-level": lambda: patch_after(W, "(GC) %s frozen by %s.", "wizutilMudlogLevel(s.player)", "33"),
    "thaw-level": lambda: patch_after(W, "(GC) %s un-frozen by %s.", "wizutilMudlogLevel(s.player)", "33"),
    # invisibility ignored: the shared helper stops applying MAX(LVL_GOD, invis)
    "wizutil-invis-ignored": lambda: patch_after(W, "func wizutilMudlogLevel", "max(game.LVL_GOD, actor.GetInvisLevel())", "game.LVL_GOD"),
    # skillset tap metadata (modify.c:334: BRF, -1, TRUE)
    "skillset-type": lambda: patch_after(P, "changed %s's %s to %d.", "game.MudlogBrief, -1, true", "game.MudlogNormal, -1, true"),
    "skillset-level": lambda: patch_after(P, "changed %s's %s to %d.", "-1, true", "0, true"),
    "skillset-file": lambda: patch_after(P, "changed %s's %s to %d.", "-1, true", "-1, false"),
}
M[mid]()
print("mutated:", mid)
PY
}

ctl() { # id, test regex, expected failing subtest, expected assertion substring
	local id=$1 test=$2 sub=$3 msg=$4
	local log="$OUT_DIR/$id.log"
	mutate "$id" || { echo "MUTATION FAILED: $id"; restore; return 1; }
	# -count=1 so a cached result can never stand in for a run.
	go test ./pkg/session/ -run "$test" -count=1 >"$log" 2>&1
	local rc=$?
	restore

	# Reject failures that are not the control's intended discriminator failing.
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

run_one() { # id -> the right test regex, subtest and assertion for that mutation
	case "$1" in
	squelch-type) ctl "$1" '^TestWizutilMudlogTypeDiscriminators$' 'TestWizutilMudlogTypeDiscriminators/squelch_is_BRF,_not_NRM' 'LOG1-only observer' ;;
	freeze-type) ctl "$1" '^TestWizutilMudlogTypeDiscriminators$' 'TestWizutilMudlogTypeDiscriminators/freeze_is_BRF,_not_NRM' 'LOG1-only observer' ;;
	thaw-type) ctl "$1" '^TestWizutilMudlogTypeDiscriminators$' 'TestWizutilMudlogTypeDiscriminators/thaw_is_BRF,_not_NRM' 'LOG1-only observer' ;;
	notitle-level) ctl "$1" '^TestWizutilMudlogLevelDiscriminators$' 'TestWizutilMudlogLevelDiscriminators/notitle_needs_LVL_GOD/lv_god_floor' 'level-33 observer received' ;;
	squelch-level) ctl "$1" '^TestWizutilMudlogLevelDiscriminators$' 'TestWizutilMudlogLevelDiscriminators/squelch_needs_LVL_GOD/lv_god_floor' 'level-33 observer received' ;;
	freeze-level) ctl "$1" '^TestWizutilMudlogLevelDiscriminators$' 'TestWizutilMudlogLevelDiscriminators/freeze_needs_LVL_GOD/lv_god_floor' 'level-33 observer received' ;;
	thaw-level) ctl "$1" '^TestWizutilMudlogLevelDiscriminators$' 'TestWizutilMudlogLevelDiscriminators/thaw_needs_LVL_GOD/lv_god_floor' 'level-33 observer received' ;;
	# Flattening the helper to LVL_GOD drops the invisibility threshold to 34, so
	# the 35-level observer (intended to be silent) receives first.
	wizutil-invis-ignored) ctl "$1" '^TestWizutilMudlogLevelDiscriminators$' 'TestWizutilMudlogLevelDiscriminators/notitle_needs_LVL_GOD/invis_raises_threshold' 'level-35 observer received' ;;
	skillset-type) ctl "$1" '^TestSkillsetMudlogTapMetadata$' 'TestSkillsetMudlogTapMetadata' 'tap type =' ;;
	skillset-level) ctl "$1" '^TestSkillsetMudlogTapMetadata$' 'TestSkillsetMudlogTapMetadata' 'tap level =' ;;
	skillset-file) ctl "$1" '^TestSkillsetMudlogTapMetadata$' 'TestSkillsetMudlogTapMetadata' 'tap to_file = false' ;;
	*) echo "unknown control: $1" >&2; return 2 ;;
	esac
}

if [ "${1:-}" = "--only" ] && [ -n "${2:-}" ]; then
	# Targeted mode: one control only. Used to verify that an interrupted run
	# restores the tree — kill the script while its mutation is applied and
	# `git status` must be clean.
	run_one "$2"
	exit $?
fi

rc=0
for id in squelch-type freeze-type thaw-type notitle-level squelch-level \
	freeze-level thaw-level wizutil-invis-ignored skillset-type skillset-level \
	skillset-file; do
	run_one "$id" || rc=1
done
echo "raw control output retained in $OUT_DIR"
exit $rc
