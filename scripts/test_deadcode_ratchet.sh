#!/usr/bin/env bash
# test_deadcode_ratchet.sh — R5h controls for scripts/deadcode-ratchet.sh.
# Every control names its break pair: the exact reverted fix that would turn
# it red. DEADCODE is stubbed and BASELINE points at a fixture, so no real
# analysis runs and the controls are deterministic.
set -u

repo_root=$(CDPATH= cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
ratchet=$repo_root/scripts/deadcode-ratchet.sh
work=$(mktemp -d "${TMPDIR:-/tmp}/dp-deadcode-test.XXXXXX")
trap 'rm -rf "$work"' EXIT

pass=0
fail=0
ok() {
	pass=$((pass + 1))
	printf 'ok %d - %s\n' "$pass" "$1"
}
not_ok() {
	fail=$((fail + 1))
	printf 'not_ok %d - %s\n' "$((pass + fail))" "$1"
}

# make_stub <file> <script-text> — write an executable stub.
make_stub() {
	printf '%s\n' "$2" >"$1"
	chmod +x "$1"
}

# run_ratchet <stub> <baseline-text> — run the ratchet against a stub
# deadcode and a fixture baseline; sets status and out (stdout + stderr).
run_ratchet() {
	printf '%s\n' "$2" >"$work/baseline.txt"
	out=$(DEADCODE="$1" BASELINE="$work/baseline.txt" "$ratchet" 2>&1)
	status=$?
}

# Two baselined symbols used across controls, as they appear in a normalized
# baseline (file + symbol, no position).
baseline_two='# fixture baseline
pkg/game/weather.go: unreachable func: WeatherChange
pkg/command/admin_commands.go: unreachable func: AdminCommands.RegisterCommands'

# Control 1 — a failing deadcode fails the ratchet, loudly (R5h).
# Break pair: without pipefail (and with stderr discarded, as in the first
# draft's `2>/dev/null | tail -n +2`), a dead tool produces an empty current
# set and the ratchet reports ok.
make_stub "$work/stub-fail" '#!/bin/sh
echo "stub-deadcode: pkg/game/broken.go:4:2: expected declaration, found }" >&2
exit 1'
run_ratchet "$work/stub-fail" "$baseline_two"
if [[ $status -ne 0 && $out == *stub-deadcode:* ]]; then
	ok "tool failure fails the ratchet with its stderr visible"
else
	not_ok "tool failure fails the ratchet with its stderr visible (status=$status out=$out)"
fi

# Control 2 — the FIRST finding counts (R5h).
# Break pair: deadcode prints no header, so the first draft's `tail -n +2`
# silently ate the first real finding — exactly the single-finding case here.
make_stub "$work/stub-one" '#!/bin/sh
cat <<"FINDINGS"
pkg/session/commands.go:12:5: unreachable func: NeverWired
FINDINGS'
run_ratchet "$work/stub-one" "$baseline_two"
if [[ $status -ne 0 && $out == *NeverWired* ]]; then
	ok "a single new first finding fails and is named"
else
	not_ok "a single new first finding fails and is named (status=$status out=$out)"
fi

# Control 3 — a new finding among baselined ones is named, baselined ones are not.
make_stub "$work/stub-new" '#!/bin/sh
cat <<"FINDINGS"
pkg/game/weather.go:214:6: unreachable func: WeatherChange
pkg/command/admin_commands.go:93:26: unreachable func: AdminCommands.RegisterCommands
pkg/game/new_code.go:5:1: unreachable func: BrandNew
FINDINGS'
run_ratchet "$work/stub-new" "$baseline_two"
if [[ $status -ne 0 && $out == *BrandNew* && $out != *WeatherChange* ]]; then
	ok "new finding is named and baselined neighbours are not flagged"
else
	not_ok "new finding is named and baselined neighbours are not flagged (status=$status out=$out)"
fi

# Control 4 — line drift on a baselined symbol stays green (R5h).
# Break pair: keys that include line/column make any unrelated edit above a
# baselined function fail CI. Both positions here differ from anything a
# text diff of the baseline could predict; only symbol keys match.
make_stub "$work/stub-drift" '#!/bin/sh
cat <<"FINDINGS"
pkg/game/weather.go:999:1: unreachable func: WeatherChange
pkg/command/admin_commands.go:1000:9: unreachable func: AdminCommands.RegisterCommands
FINDINGS'
run_ratchet "$work/stub-drift" "$baseline_two"
if [[ $status -eq 0 && $out == *ok* ]]; then
	ok "position drift on baselined symbols stays green"
else
	not_ok "position drift on baselined symbols stays green (status=$status out=$out)"
fi

# Control 5 — node_modules findings are not ours to gate.
# Break pair: without the node_modules filter, vendored Go files under
# admin-ui/node_modules list as new unreachable functions.
make_stub "$work/stub-vendor" '#!/bin/sh
cat <<"FINDINGS"
pkg/game/weather.go:214:6: unreachable func: WeatherChange
pkg/command/admin_commands.go:93:26: unreachable func: AdminCommands.RegisterCommands
admin-ui/node_modules/vendor.go:1:1: unreachable func: VendoredHelper
FINDINGS'
run_ratchet "$work/stub-vendor" "$baseline_two"
if [[ $status -eq 0 && $out != *VendoredHelper* ]]; then
	ok "node_modules findings are ignored"
else
	not_ok "node_modules findings are ignored (status=$status out=$out)"
fi

# Control 6 — a missing baseline is an error, not an empty comparison.
make_stub "$work/stub-one-copy" '#!/bin/sh
cat <<"FINDINGS"
pkg/game/weather.go:214:6: unreachable func: WeatherChange
FINDINGS'
out=$(DEADCODE="$work/stub-one-copy" BASELINE="$work/absent-baseline.txt" "$ratchet" 2>&1)
status=$?
if [[ $status -ne 0 && $out == *"not found"* ]]; then
	ok "missing baseline fails with a pointer to deadcode-ratchet-update"
else
	not_ok "missing baseline fails with a pointer to deadcode-ratchet-update (status=$status out=$out)"
fi

echo "1..$((pass + fail))"
if [[ $fail -gt 0 ]]; then
	echo "FAILED: $fail of $((pass + fail)) controls failed"
	exit 1
fi
echo "deadcode-ratchet controls: all $pass passed"
