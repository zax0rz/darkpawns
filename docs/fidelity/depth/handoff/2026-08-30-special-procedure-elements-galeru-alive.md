# Depth-fidelity handoff — `elements_galeru_alive` — 2026-08-30

## Frontier and queue position

- Started from clean, freshly pulled `main`; `make fidelity-depth` confirmed **1225 total, 1179 proven/delegated, 13 blocked, 33 excluded**, actionable **1179/1192 (98.9%)**.
- Consumed the next active C definition in file/registration order: `SPECIAL(elements_galeru_alive)` at `src/spec_procs3.c:1184-1216`, actively registered by `ASSIGNROOM(1394, elements_galeru_alive)` at `src/spec_assign.c:632`.
- Next active special: `elements_minion`, registered for mob 1313 at `src/spec_assign.c:195`, defined at `src/spec_procs3.c:1217-1240`.

## C call path and observable contract

The room special is reached through the current-room player-command dispatcher (`src/interpreter.c:1407-1456`) and the movement special hook (`src/act.movement.c:115`). A commandless invocation returns `FALSE` before scanning. For a command, C scans the current room for exact mob VNum 1315; any such Galeru leaves the room special at `FALSE`, allowing ordinary command handling. When Galeru is absent, C walks every character in the room—not only players—then for each character sends the exact direct buffer `"You begin to feel very dizzy and the world around you fades...\\r\\n\\n"`, emits the `TO_NOTVICT` departure Act, transfers to room 1395, runs `look_at_room`, emits the `TO_NOTVICT` arrival Act, and returns `TRUE`. The per-character sequence makes destination occupants visible to later looks; NPCs have no descriptor for direct bytes but still move and can produce observer Acts.

## Proof vehicles and result

- Live gate scenario: `cmd/dp-oracle-diff/scenarios/spec-proc-elements-galeru-alive.txt`.
- Dead branch/destination scenario: `cmd/dp-oracle-diff/scenarios/spec-proc-elements-galeru-alive-dead.txt`.
- `--show-oracle` was run on seed 1. Main RED showed the name-based Go gate missed live Galeru, the old handler omitted NPC/all-character processing, used incorrect direct spacing, and used non-canonical audience/transfer behavior.
- Both vehicles were GREEN for seeds **1, 2, 3, 5, and 8**.
- Focused proof in `pkg/game/spec_elements_galeru_alive_test.go` covers commandless/live gates, exact VNum-vs-keyword behavior, canonical relocation, and NPC movement. `pkg/game/spec_elements_galeru_column_test.go` was updated with the room-1395 fixture used by the shared test world.

## Go changes

- Replaced the name lookup with the exact mob VNum 1315 gate.
- Matched C’s exact direct bytes, all-character processing, audience-aware departure/arrival Acts, checked `PlayerTransfer`/`MobTransfer`, destination look, and procedure-specific C room-list framing.
- Removed the now-unused name-based helper.
- No files under `src/` or `darkpawns-c-oracle/` were edited.

## Manifest and gates

Added seven rows to `docs/fidelity/depth/spec-procs.tsv` for entry, live exact gate, ordinary fallthrough, dead branch, audience, relocation, and destination look, citing R1/R2/R3/R4/R5e.

Local gates passed: `make fidelity-depth`, `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run ./...`, `gofumpt -l .` clean, and `git diff --check`.

Branch `glm/spec-elements-galeru-alive` was committed as `c952a43ab`; PR **#788** passed `test`, `lint`, and `security` (build/deploy skipped) and was self-merged into `main` as `349bf03b5`.
