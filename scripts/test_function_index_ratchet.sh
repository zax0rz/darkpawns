#!/usr/bin/env bash
set -euo pipefail

fake_file="$(mktemp pkg/game/zz_fnindex_ratchet_fake_XXXX.go)"
cleanup() {
	rm -f "$fake_file"
}
trap cleanup EXIT

cat > "$fake_file" <<'EOF'
package game

// C: is_number()
func fnIndexRatchetFake() {}
EOF

if output="$(go run ./tools/fnindex -check 2>&1)"; then
	printf '%s\n' 'expected fake C association to fail the function-index ratchet' >&2
	exit 1
fi
if [[ "$output" != *"unbaselined C association: src/interpreter.c:is_number -> pkg/game/"* ]]; then
	printf '%s\n' 'function-index check failed for an unexpected reason:' "$output" >&2
	exit 1
fi
printf '%s\n' "$output"
printf '%s\n' 'EXPECTED RED: fake is_number association is not in baseline'

cleanup
trap - EXIT
go run ./tools/fnindex -check
