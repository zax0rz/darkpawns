# Depth-fidelity handoff — `elements_galeru_column` — 2026-08-30

## Frontier and queue position

- Started from a clean, freshly pulled `main` at the prior cylinder handoff.
- `make fidelity-depth`: **1218 total, 1172 proven/delegated, 13 blocked, 33 excluded**; actionable completion **1172/1185 (98.9%)**.
- Consumed the next active special-procedure definition in C file order: `SPECIAL(elements_galeru_column)` at `src/spec_procs3.c:1137-1182`, actively registered by `ASSIGNROOM(1372, elements_galeru_column)` at `src/spec_assign.c:631`.
- Next active special: `elements_galeru_alive`, registered at room 1394.

## C call path and observable contract

The room special is reached from the current-room player-command dispatcher (`src/interpreter.c:1407-1456`) and the movement special hook (`src/act.movement.c:115`). It has no command-text gate. It scans the exact room/object pairs `(1360,1300)`, `(1364,1301)`, `(1380,1302)`, and `(1384,1303)`; if any talisman is absent it returns `FALSE` without output or movement. When all four are present, it walks `world[ch->in_room].people` in C’s front-insert order, skips NPCs, and for each player sends the exact direct beam text, broadcasts the departure with `TO_NOTVICT`, transfers the player to room 1389, runs `look_at_room`, and broadcasts the arrival with `TO_NOTVICT`. The per-player sequencing means the second player’s destination look can see the first player already transferred. The C direct buffer is `"Four beams of colored light from the corners of the chamber converge around you.\\r\\n\\n"`.

## Proof vehicle and result

- Scenario: `cmd/dp-oracle-diff/scenarios/spec-proc-elements-galeru-column.txt`.
- It places all four exact talismans through the registered pillar-room vehicles, places a peer in room 1372, then triggers the room special with `say hello`.
- `--show-oracle` was run on seed 1 to confirm the intended C block. The initial RED exposed Go’s room-wide/self departure leak, missing destination look/arrival sequencing, and one extra direct-message line ending.
- Final oracle matrix was GREEN for seeds **1, 2, 3, 5, and 8**.
- Focused proof: `pkg/game/spec_elements_galeru_column_test.go` covers incomplete prerequisites, nil actor, NPC exclusion, exact direct bytes, audience isolation, sequential relocation, and destination look.

## Go changes

- Added nil and nil-item guards around the exact four-room prerequisite scan.
- Matched C’s actor-first/front-insert processing order deterministically.
- Sent the C direct beam buffer byte-for-byte, used audience-aware `Act` for departure/arrival, used checked `PlayerTransfer`, and called `lookAtRoom` after each transfer.
- No C oracle files under `src/` or `darkpawns-c-oracle/` were edited.

## Manifest and gates

Added six rows to `docs/fidelity/depth/spec-procs.tsv` for entry, nil actor, complete branch, audience, relocation, and destination look, citing R1/R2/R3/R4/R5e.

Local gates passed before commit: `make fidelity-depth`, `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run ./...`, `gofumpt -l .` clean, and `git diff --check`.

Branch `glm/spec-elements-galeru-column` was committed as `a6cebc415`; PR **#787** passed `test`, `lint`, and `security` (build/deploy skipped) and was self-merged into `main` as `313d1f065`.
