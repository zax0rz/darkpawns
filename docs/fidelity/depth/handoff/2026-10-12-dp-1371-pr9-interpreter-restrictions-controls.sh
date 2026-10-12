#!/usr/bin/env bash
# DP-1371 batch 9 — argument-level controls for the interpreter.c denial /
# bad-password / self-delete proof (pkg/session/interpreter_batch9_mudlog_test.go).
#
# The four refusal rows share one producer (refuseEntry, entry_restrictions.go:32),
# so their producer-specific coverage is a payload mutation at each caller, while
# the shared type/level/file arguments get one control each.
set -u

REPO=${REPO:-$(git rev-parse --show-toplevel 2>/dev/null)}
if [ -z "${REPO:-}" ] || [ ! -d "$REPO/pkg/session" ]; then
	echo "run from inside the repository, or set REPO=/path/to/repo" >&2
	exit 2
fi
cd "$REPO" || exit 2

OUT_DIR=${OUT_DIR:-${TMPDIR:-/tmp}/dp1371-pr9-controls}
mkdir -p "$OUT_DIR"

MUTABLE="pkg/session/entry_restrictions.go pkg/session/session_login.go pkg/session/menu.go"

restore() {
	for f in $MUTABLE; do
		if [ -f "$f.ctl9.bak" ]; then mv "$f.ctl9.bak" "$f"; fi
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
ER = "pkg/session/entry_restrictions.go"
SL = "pkg/session/session_login.go"
MN = "pkg/session/menu.go"
ENTRY = 'game.MudLog(log, game.MudlogNormal, game.LVL_GOD, true)'
BADPW = 'fmt.Sprintf("Bad PW: %s [%s]", rec.Name, s.MudHost())'
SELFDEL = 'fmt.Sprintf("%s (lev %d) has self-deleted.", name, level)'

def patch_after(path, anchor, old, new):
    shutil.copyfile(path, path + ".ctl9.bak")
    s = open(path).read()
    i = s.index(anchor)
    j = s.index(old, i)
    open(path, "w").write(s[:j] + new + s[j + len(old):])

M = {
    # producer-specific payload mutations at each refusal caller
    "ref-siteban-payload": lambda: patch_after(ER, "refuseNewSiteBan", "(siteban)", "(SITEBAN)"),
    "ref-wizlock-payload": lambda: patch_after(ER, "refuseNewWizlock", "(wizlock)", "(WIZLOCK)"),
    "ref-loginwiz-payload": lambda: patch_after(ER, "refuseReturningWizlock", "(wizlock)", "(WIZLOCK)"),
    "ref-conndenied-payload": lambda: patch_after(ER, "refuseReturningSiteBan", "denied from %s", "denied by %s"),
    # shared refusal producer arguments
    "ref-type": lambda: patch_after(ER, "func (s *Session) refuseEntry", "game.MudlogNormal", "game.MudlogBrief"),
    "ref-level": lambda: patch_after(ER, "func (s *Session) refuseEntry", "game.LVL_GOD, true)", "game.LVL_GOD - 1, true)"),
    "ref-file": lambda: patch_after(ER, "func (s *Session) refuseEntry", "game.LVL_GOD, true)", "game.LVL_GOD, false)"),
    # bad password (BRF at LVL_GOD)
    "badpw-type": lambda: patch_after(SL, BADPW, "game.MudlogBrief", "game.MudlogNormal"),
    "badpw-level": lambda: patch_after(SL, BADPW, "game.LVL_GOD, true)", "game.LVL_GOD - 1, true)"),
    "badpw-file": lambda: patch_after(SL, BADPW, "game.LVL_GOD, true)", "game.LVL_GOD, false)"),
    # self delete (NRM at LVL_GOD)
    "selfdel-type": lambda: patch_after(MN, SELFDEL, "game.MudlogNormal", "game.MudlogBrief"),
    "selfdel-level": lambda: patch_after(MN, SELFDEL, "game.LVL_GOD, true)", "game.LVL_GOD - 1, true)"),
    "selfdel-file": lambda: patch_after(MN, SELFDEL, "game.LVL_GOD, true)", "game.LVL_GOD, false)"),
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
	ref-siteban-payload) ctl "$1" '^TestEntryRestrictionMudlogProducers$' 'TestEntryRestrictionMudlogProducers/siteban_1826' 'file payload count' ;;
	ref-wizlock-payload) ctl "$1" '^TestEntryRestrictionMudlogProducers$' 'TestEntryRestrictionMudlogProducers/wizlock_new_1838' 'file payload count' ;;
	ref-loginwiz-payload) ctl "$1" '^TestEntryRestrictionMudlogProducers$' 'TestEntryRestrictionMudlogProducers/login_wizlock_1912' 'file payload count' ;;
	ref-conndenied-payload) ctl "$1" '^TestEntryRestrictionMudlogProducers$' 'TestEntryRestrictionMudlogProducers/conn_denied_1903' 'file payload count' ;;
	ref-type) ctl "$1" '^TestEntryRestrictionMudlogProducers$' 'TestEntryRestrictionMudlogProducers/siteban_1826' 'LOG1-only observer received' ;;
	ref-level) ctl "$1" '^TestEntryRestrictionMudlogProducers$' 'TestEntryRestrictionMudlogProducers/siteban_1826' 'level-33 observer received' ;;
	ref-file) ctl "$1" '^TestEntryRestrictionMudlogProducers$' 'TestEntryRestrictionMudlogProducers/siteban_1826' 'file payload count' ;;
	badpw-type) ctl "$1" '^TestBadPasswordMudlog$' 'TestBadPasswordMudlog' 'LOG1-only observer did not receive' ;;
	badpw-level) ctl "$1" '^TestBadPasswordMudlog$' 'TestBadPasswordMudlog' 'level-33 observer received' ;;
	badpw-file) ctl "$1" '^TestBadPasswordMudlog$' 'TestBadPasswordMudlog' 'file payload count' ;;
	selfdel-type) ctl "$1" '^TestSelfDeleteMudlog$' 'TestSelfDeleteMudlog' 'LOG1-only observer received' ;;
	selfdel-level) ctl "$1" '^TestSelfDeleteMudlog$' 'TestSelfDeleteMudlog' 'level-33 observer received' ;;
	selfdel-file) ctl "$1" '^TestSelfDeleteMudlog$' 'TestSelfDeleteMudlog' 'file payload count' ;;
	*) echo "unknown control: $1" >&2; return 2 ;;
	esac
}

if [ "${1:-}" = "--only" ] && [ -n "${2:-}" ]; then
	run_one "$2"
	exit $?
fi

rc=0
for id in ref-siteban-payload ref-wizlock-payload ref-loginwiz-payload \
	ref-conndenied-payload ref-type ref-level ref-file \
	badpw-type badpw-level badpw-file \
	selfdel-type selfdel-level selfdel-file; do
	run_one "$id" || rc=1
done
echo "raw control output retained in $OUT_DIR"
exit $rc
