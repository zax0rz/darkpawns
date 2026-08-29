# 2026-08-29 — `horn` depth slice

## Frontier and queue

- Started from `main`, pulled the merged bank boundary, ran
  `make fidelity-depth`, and reread `docs/fidelity/DEPTH_TESTING.md` plus the
  newest handoff. The post-bank frontier was 959 total cases; this slice adds
  seven cases, yielding 966 total, 939 proven/delegated, 6 blocked, and 21
  excluded; actionable completion is 939/945 (99.4%).
- The C special inventory was refreshed across `src/spec_procs.c`,
  `src/spec_procs2.c`, and `src/spec_procs3.c`, then checked against the active
  `ASSIGNMOB`/`ASSIGNOBJ`/`ASSIGNROOM` tables in `src/spec_assign.c`. The next
  active, unclaimed definition in source order after `horn` is
  `normal_checker` in `src/spec_procs2.c:162`, registered for mob vnums
  18301–18304 at `src/spec_assign.c:411-414`.
- The unrelated untracked brief
  `docs/briefs/BRIEF-2026-08-28-economy-specproc-cluster.md` remains preserved.

## C call path and branch census

- `SPECIAL(horn)` is registered for object vnum 14415 at
  `src/spec_assign.c:558` and defined at `src/spec_procs.c:2401-2419`.
  `special()` dispatches room, equipment, inventory, mobile, and room-object
  specials before the command-table handler at `src/interpreter.c:1407-1475`.
- The procedure handles only exact `use`; it requires the concrete #14415
  object to be the actor's `WEAR_HOLD` object and requires C `isname` to match
  the command argument. A match sends one `send_to_zone` line to awake players
  in the same zone outside the actor's room, two actor-only `stc` lines, two
  object-substituted `TO_ROOM` lines, and returns `TRUE`. All gates and the
  `FALSE` ordinary-`do_use` fallthrough are covered.

## RED/GREEN evidence and port result

- RED on pre-fix `main`: `spec-proc-horn` exposed missing zone output, wrong
  object substitution and room audience, actor leakage, doubled CRLF bytes,
  and false interception for nonmatching/un-held arguments.
- GREEN on `glm/spec-horn`:
  `DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle /usr/local/go/bin/go run ./cmd/dp-oracle-diff --scenario spec-proc-horn --show-oracle`
  reports `result: no normalized divergence` for held `horn`/`silver`, the
  multi-word gate, ordinary fallthrough, and removal state.
- The Go fix adds an object-aware special receiver with compatibility adapters
  for existing object specials, implements the C horn audiences and bytes, and
  removes the unrelated Go-only `use`-as-skill fallback so C `do_use`
  fallthrough remains reachable. Focused coverage is in
  `TestSpecHornObjectReceiverAudienceAndGates` and the updated legacy horn
  regression test.
- No `src/` or `darkpawns-c-oracle/` file was edited.

## Verification and integration

- Feature-branch gates passed: `make fidelity-depth`, `go build ./...`,
  `go vet ./...`, `go test ./...`, `golangci-lint run ./...`, and clean
  `gofumpt -l .`.
- PR #753 (`glm/spec-horn`) received green lint, security, and test checks;
  build/deploy were skipped by workflow policy. It was squash-merged as
  `6502621cf`; no CI retry was needed because checks fired normally.

This slice applies R1 (player-facing bytes), R2 (registered object command
surface), R4 (no invented output), and R5/R5e (actual C dispatch/call path,
whole object-special receiver class, and oracle verification).

## Next queue item

Continue the active special-procedure inventory with `normal_checker` from
`src/spec_procs2.c:162`, using registered mob vnums 18301–18304. Do not re-pick
`horn`, `bank`, or any earlier claimed procedure. After the active inventory is
exhausted, attempt the single blocked `objmagic.sleep-entry-gates` vehicle,
then sweep remaining un-manifested command families in `src/interpreter.c`
table order.
