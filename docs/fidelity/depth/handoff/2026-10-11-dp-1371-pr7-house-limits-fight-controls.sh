#!/usr/bin/env bash
# DP-1371 batch 7 — argument-level controls for the four-row proof:
#   pkg/game/house_mudlog_test.go        (house.c:570, house.c:699)
#   pkg/game/limits_idle_mudlog_test.go  (limits.c:450)
#   pkg/game/pk_death_mudlog_test.go     (fight.c:1681)
#
# Each control mutates one producer argument and must make the declared subtest
# fail on the declared assertion. Build failures, infrastructure failures,
# failures in another test, and a declared test failing without the declared
# message are rejected as CONTROL INVALID. Raw output per control is retained in
# OUT_DIR; the tree is restored on normal exit and on INT/TERM/HUP.
set -u

REPO=${REPO:-$(git rev-parse --show-toplevel 2>/dev/null)}
if [ -z "${REPO:-}" ] || [ ! -d "$REPO/pkg/game" ]; then
	echo "run from inside the repository, or set REPO=/path/to/repo" >&2
	exit 2
fi
cd "$REPO" || exit 2

OUT_DIR=${OUT_DIR:-${TMPDIR:-/tmp}/dp1371-pr7-controls}
mkdir -p "$OUT_DIR"

MUTABLE="pkg/game/house_control.go pkg/game/house_player.go pkg/game/limits_misc.go pkg/game/death.go"

restore() {
	for f in $MUTABLE; do
		if [ -f "$f.ctl7.bak" ]; then mv "$f.ctl7.bak" "$f"; fi
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
HC = "pkg/game/house_control.go"
HP = "pkg/game/house_player.go"
LM = "pkg/game/limits_misc.go"
DD = "pkg/game/death.go"
PAY = 'fmt.Sprintf("Payment for house %s collected by %s.", args[0], chName)'
TRA = 'fmt.Sprintf("%s transfered %s house to %s.", chName, hshr(ch), toTitle(toLower(arg)))'
IDLE = 'fmt.Sprintf("%s force-rented and extracted (idle).", p.Name)'
PK = 'fmt.Sprintf("%s killed by %s at %s", player.GetName(), killerName, roomName)'

def patch_after(path, anchor, old, new):
    shutil.copyfile(path, path + ".ctl7.bak")
    s = open(path).read()
    i = s.index(anchor)
    j = s.index(old, i)
    open(path, "w").write(s[:j] + new + s[j + len(old):])

M = {
    # house pay (NRM, max(LVL_IMMORT, invis), TRUE)
    "pay-type": lambda: patch_after(HC, PAY, "MudlogNormal", "MudlogBrief"),
    "pay-level": lambda: patch_after(HC, PAY, "max(LVL_IMMORT, ch.GetInvisLevel())", "LVL_IMMORT - 1"),
    "pay-invis": lambda: patch_after(HC, PAY, "max(LVL_IMMORT, ch.GetInvisLevel())", "LVL_IMMORT"),
    "pay-file": lambda: patch_after(HC, PAY, "max(LVL_IMMORT, ch.GetInvisLevel()), true)", "max(LVL_IMMORT, ch.GetInvisLevel()), false)"),
    # house transfer (same shape)
    "transfer-type": lambda: patch_after(HP, TRA, "MudlogNormal", "MudlogBrief"),
    "transfer-level": lambda: patch_after(HP, TRA, "max(LVL_IMMORT, ch.GetInvisLevel())", "LVL_IMMORT - 1"),
    "transfer-invis": lambda: patch_after(HP, TRA, "max(LVL_IMMORT, ch.GetInvisLevel())", "LVL_IMMORT"),
    "transfer-file": lambda: patch_after(HP, TRA, "max(LVL_IMMORT, ch.GetInvisLevel()), true)", "max(LVL_IMMORT, ch.GetInvisLevel()), false)"),
    # idle force-rent (CMP, fixed LVL_GOD, TRUE)
    "idle-type": lambda: patch_after(LM, IDLE, "MudlogComplete", "MudlogNormal"),
    "idle-level": lambda: patch_after(LM, IDLE, "LVL_GOD, true)", "LVL_GOD - 1, true)"),
    "idle-invis-term": lambda: patch_after(LM, IDLE, "LVL_GOD, true)", "max(LVL_IMMORT, p.GetInvisLevel()), true)"),
    "idle-file": lambda: patch_after(LM, IDLE, "LVL_GOD, true)", "LVL_GOD, false)"),
    # player death (BRF, fixed lvlImmort, TRUE)
    "pk-type": lambda: patch_after(DD, PK, "MudlogBrief, lvlImmort, true)", "MudlogNormal, lvlImmort, true)"),
    "pk-level": lambda: patch_after(DD, PK, "lvlImmort, true)", "lvlImmort - 1, true)"),
    "pk-invis-term": lambda: patch_after(DD, PK, "lvlImmort, true)", "max(LVL_IMMORT, player.GetInvisLevel()), true)"),
    "pk-file": lambda: patch_after(DD, PK, "lvlImmort, true)", "lvlImmort, false)"),
}
M[mid]()
print("mutated:", mid)
PY
}

ctl() { # id, test regex, expected failing subtest, expected assertion substring
	local id=$1 test=$2 sub=$3 msg=$4
	local log="$OUT_DIR/$id.log"
	mutate "$id" || { echo "MUTATION FAILED: $id"; restore; return 1; }
	go test ./pkg/game/ -run "$test" -count=1 >"$log" 2>&1
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
	pay-type) ctl "$1" '^TestHouseMudlogProducers$' 'TestHouseMudlogProducers/pay_house' 'LOG1-only observer received' ;;
	transfer-type) ctl "$1" '^TestHouseMudlogProducers$' 'TestHouseMudlogProducers/transfer' 'LOG1-only observer received' ;;
	pay-level) ctl "$1" '^TestHouseMudlogLevelDiscriminators$' 'TestHouseMudlogLevelDiscriminators/pay_house/lv_immort_floor' 'level-30 observer received' ;;
	transfer-level) ctl "$1" '^TestHouseMudlogLevelDiscriminators$' 'TestHouseMudlogLevelDiscriminators/transfer/lv_immort_floor' 'level-30 observer received' ;;
	pay-invis) ctl "$1" '^TestHouseMudlogLevelDiscriminators$' 'TestHouseMudlogLevelDiscriminators/pay_house/invis_raises_threshold' 'level-35 observer received' ;;
	transfer-invis) ctl "$1" '^TestHouseMudlogLevelDiscriminators$' 'TestHouseMudlogLevelDiscriminators/transfer/invis_raises_threshold' 'level-35 observer received' ;;
	pay-file) ctl "$1" '^TestHouseMudlogProducers$' 'TestHouseMudlogProducers/pay_house' 'file payload count' ;;
	transfer-file) ctl "$1" '^TestHouseMudlogProducers$' 'TestHouseMudlogProducers/transfer' 'file payload count' ;;
	idle-type) ctl "$1" '^TestIdleForceRentMudlogProducer$' 'TestIdleForceRentMudlogProducer' 'LOG2-only observer received' ;;
	idle-level) ctl "$1" '^TestIdleForceRentMudlogProducer$' 'TestIdleForceRentMudlogProducer' 'level-33 observer received' ;;
	idle-invis-term) ctl "$1" '^TestIdleForceRentMudlogLevelIsLVLGOD$' 'TestIdleForceRentMudlogLevelIsLVLGOD/invisibility_does_not_raise_it' 'suppressed the line for a level-34 observer' ;;
	idle-file) ctl "$1" '^TestIdleForceRentMudlogProducer$' 'TestIdleForceRentMudlogProducer' 'file payload count' ;;
	pk-type) ctl "$1" '^TestPKDeathMudlogTypeIsBRF$' 'TestPKDeathMudlogTypeIsBRF' 'LOG1-only observer did not receive' ;;
	pk-level) ctl "$1" '^TestPKDeathMudlogProducer$' 'TestPKDeathMudlogProducer' 'level-30 observer received' ;;
	pk-invis-term) ctl "$1" '^TestPKDeathMudlogLevelIsLVLIMMORT$' 'TestPKDeathMudlogLevelIsLVLIMMORT/killer_invisibility_does_not_raise_it' 'suppressed the line for a level-31 observer' ;;
	pk-file) ctl "$1" '^TestPKDeathMudlogProducer$' 'TestPKDeathMudlogProducer' 'file PK payload count' ;;
	*) echo "unknown control: $1" >&2; return 2 ;;
	esac
}

if [ "${1:-}" = "--only" ] && [ -n "${2:-}" ]; then
	run_one "$2"
	exit $?
fi

rc=0
for id in pay-type transfer-type pay-level transfer-level pay-invis transfer-invis \
	pay-file transfer-file idle-type idle-level idle-invis-term idle-file \
	pk-type pk-level pk-invis-term pk-file; do
	run_one "$id" || rc=1
done
echo "raw control output retained in $OUT_DIR"
exit $rc
