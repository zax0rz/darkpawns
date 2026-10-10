#!/usr/bin/env bash
# deadcode-ratchet.sh — the unreachable-from-everything set only shrinks (R5b).
#
# Runs deadcode over the whole module with tests counted as roots
# (`-test ./...`; the conservative `./cmd/... ./tools/...` form is blind to
# own-package tests — see docs/cleanup/2026-10-09-dead-code-decisions.md) and
# fails when a function is unreachable from every program AND every test but
# is not baselined. A new entry must be wired, deleted, or added to the
# baseline with a class (d) reason (test seam, tooling hook).
#
# Failure is loud: deadcode's stderr is never discarded, and pipefail turns
# a tool failure (build error, bad pin, missing network) into a ratchet
# failure instead of an empty, silently-green run. The R5h controls in
# scripts/test_deadcode_ratchet.sh prove each failure mode with a stubbed
# DEADCODE command.
#
# --update regenerates docs/cleanup/deadcode-baseline.txt in place (make
# deadcode-ratchet-update). The entry list it writes is exact; the header
# comment is fixed here so an update cannot lose it.
set -euo pipefail

if [[ $# -gt 1 || ( $# -eq 1 && $1 != --update ) ]]; then
	echo "usage: $0 [--update]" >&2
	exit 2
fi
update=0
[[ ${1:-} = --update ]] && update=1

# Pinned, not @latest: the finding set must not drift with whatever upstream
# ships the day CI runs. v0.51.0 is the version this baseline was generated
# with; go.mod deliberately carries no golang.org/x/tools dependency, so the
# pin lives here. Bump it deliberately and regenerate the baseline in the
# same change.
#
# DEADCODE is an override for the R5h controls only: a single stub command
# that replaces the real invocation.
if [[ ${DEADCODE+x} = x ]]; then
	run_deadcode() { "$DEADCODE" "$@"; }
else
	run_deadcode() { go run golang.org/x/tools/cmd/deadcode@v0.51.0 "$@"; }
fi

BASELINE=${BASELINE:-docs/cleanup/deadcode-baseline.txt}
repo_root=$(CDPATH= cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
cd "$repo_root"

if [[ $update -eq 0 && ! -f $BASELINE ]]; then
	echo "deadcode-ratchet: baseline $BASELINE not found — run: make deadcode-ratchet-update" >&2
	exit 1
fi

# deadcode prints one finding per line and no header:
#   pkg/foo.go:12:3: unreachable func: Name
# Keys keep file + kind + symbol and drop line and column, so an edit to an
# unrelated line in a baselined file (which shifts every following position)
# cannot fail CI — only a genuinely new unreachable symbol can. Findings
# under node_modules (admin-ui vendors Go files there) are not ours to gate.
normalize() {
	sed -E -e '/\/node_modules\//d' \
		-e 's/^([^:]+):[0-9]+:[0-9]+: (unreachable [a-z]+): /\1: \2: /' |
		LC_ALL=C sort -u
}

current=$(mktemp)
base=$(mktemp)
trap 'rm -f "$current" "$base"' EXIT

# stderr passes through unredirected; with pipefail, a non-zero deadcode
# exit fails the pipeline here instead of producing an empty, green result.
if ! run_deadcode -test ./... | normalize >"$current"; then
	echo "deadcode-ratchet: deadcode failed — not a clean verdict (see errors above)" >&2
	exit 1
fi

if [[ $update -eq 1 ]]; then
	{
		cat <<'HEADER'
# deadcode baseline (ratchet): functions unreachable from every program and
# every test, per the pinned whole-module invocation:
#   go run golang.org/x/tools/cmd/deadcode@v0.51.0 -test ./...
# Keys are file + symbol with no line or column, so editing an unrelated line
# in a baselined file cannot fail CI. The baseline only shrinks: remove an
# entry when you wire or delete it, and add one only with a class (d) reason
# (test seam, tooling hook). Regenerate with: make deadcode-ratchet-update.
# See docs/cleanup/2026-10-09-dead-code-decisions.md for the rulings behind
# these entries and the 238-vs-108 whole-module finding.
HEADER
		cat "$current"
	} >"$BASELINE"
	entries=$(wc -l <"$current")
	echo "deadcode-ratchet: baseline updated ($((entries)) entries)"
	exit 0
fi

# grep exits 1 when the baseline holds only comments; that is a shrunken
# baseline, not an error.
{ grep -v '^#' "$BASELINE" || true; } | normalize >"$base"

new=$(LC_ALL=C comm -23 "$current" "$base")
if [[ -n $new ]]; then
	echo "deadcode-ratchet: NEW unreachable functions — wire, delete, or baseline them with a class (d) reason:" >&2
	printf '%s\n' "$new" >&2
	exit 1
fi

baselined=$(wc -l <"$base")
echo "deadcode-ratchet: ok ($((baselined)) baselined, 0 new)"
