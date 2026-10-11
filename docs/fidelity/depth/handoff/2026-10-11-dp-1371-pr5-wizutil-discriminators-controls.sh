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

ctl() { # id, test regex
	local id=$1 test=$2
	mutate "$id" || { echo "MUTATION FAILED: $id"; restore; return 1; }
	if go test ./pkg/session/ -run "$test" -count=1 >"/tmp/ctl5-$id.log" 2>&1; then
		echo "CONTROL BROKEN: $id — green after mutation (/tmp/ctl5-$id.log)"
		restore
		return 1
	fi
	echo "control ok: $id — failed as required"
	restore
}

if [ "${1:-}" = "--only" ] && [ -n "${2:-}" ]; then
	# Targeted mode: one control only. Used to verify that an interrupted run
	# restores the tree — kill the script while its mutation is applied and
	# `git status` must be clean. See the PR body / handoff notes.
	case "$2" in
	squelch-type | freeze-type | thaw-type) ctl "$2" '^TestWizutilMudlogTypeDiscriminators$' ;;
	notitle-level | squelch-level | freeze-level | thaw-level | wizutil-invis-ignored) ctl "$2" '^TestWizutilMudlogLevelDiscriminators$' ;;
	skillset-type | skillset-level | skillset-file) ctl "$2" '^TestSkillsetMudlogTapMetadata$' ;;
	*) echo "unknown control: $2" >&2; exit 2 ;;
	esac
	exit $?
fi

rc=0
ctl squelch-type '^TestWizutilMudlogTypeDiscriminators$' || rc=1
ctl freeze-type '^TestWizutilMudlogTypeDiscriminators$' || rc=1
ctl thaw-type '^TestWizutilMudlogTypeDiscriminators$' || rc=1
ctl notitle-level '^TestWizutilMudlogLevelDiscriminators$' || rc=1
ctl squelch-level '^TestWizutilMudlogLevelDiscriminators$' || rc=1
ctl freeze-level '^TestWizutilMudlogLevelDiscriminators$' || rc=1
ctl thaw-level '^TestWizutilMudlogLevelDiscriminators$' || rc=1
ctl wizutil-invis-ignored '^TestWizutilMudlogLevelDiscriminators$' || rc=1
ctl skillset-type '^TestSkillsetMudlogTapMetadata$' || rc=1
ctl skillset-level '^TestSkillsetMudlogTapMetadata$' || rc=1
ctl skillset-file '^TestSkillsetMudlogTapMetadata$' || rc=1
exit $rc
