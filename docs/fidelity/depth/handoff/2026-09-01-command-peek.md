# Depth-fidelity handoff — `peek`

Date: 2026-09-01
Branch: `glm/depth-peek`
Feature commit: `cc8b9efb4`
Feature PR: #1061 (merged to `main` as `64261fcaa`)

## Frontier

The clean-main frontier before this slice was 2,746 total cases: 2,675
proven/delegated, 22 blocked, and 49 excluded. After merging `peek`, it is
2,765 total: 2,694 proven/delegated, 22 blocked, and 49 excluded. Actionable
completion is 2,694/2,716 (99.2%). The special-procedure inventory remains
exhausted, and `objmagic.sleep-entry-gates` remains the single explicitly
blocked vehicle.

## C-first call path

The registration at `src/interpreter.c:602` is `{ "peek", POS_RESTING,
do_peek, 0, 0 }`. `src/act.other.c:1665-1697` first gates mortal class,
then runs `one_argument`, local visible-character lookup, the self-target
branch, and the mortal `number(1,101)` skill test. Failure calls `do_look`
with the original argument; success calls `look_at_char` and then
`improve_skill`. `src/act.informative.c:388-490` supplies character
description, player status, condition, equipment, authorized inventory peek,
and the reachable Kender post-look hook. `spec_procs2.c:594-650` is reached
only for mortal Kender viewers and NPC victims; it selects visible NPC
inventory items with `number(0,600)` and calls `do_steal(..., subcmd=1)`.

## RED and confirmed fixes

The RED vehicles exposed incorrect missing-target bytes, no NPC character
description, invented Go equipment/inventory summaries, broken self/fill-word
resolution, absent failure fallback semantics, and the missing target-audience
act emitted by the failed `do_look` path. The confirmed fixes use the C
one-argument boundary and exact messages, route character rendering through
the C-shaped look path, preserve raw object display and inventory RNG order,
emit the fallback audience only when the original `do_look` actually resolves
the same character, and implement the reachable Kender NPC inventory arm.
The mob parser also now preserves canonical detailed descriptions when a
standalone `~` terminates the one-line long description. No `src/` or
`darkpawns-c-oracle/` file was edited.

## Evidence

- Scenarios: `peek-gates-depth`, `peek-depth`, `peek-failure-depth`,
  `peek-success-depth`, `peek-higher-immortal-depth`, and
  `peek-kender-depth`.
- Manifest: `docs/fidelity/depth/peek.tsv` (19/19 proven rows).
- Focused tests: `pkg/session/peek_depth_test.go` and the canonical mob
  delimiter regression in `pkg/parser/mob_test.go`.
- Oracle matrices were green across the sampled seeds for the core gate,
  character, failure, and success vehicles; targeted higher-immortal and
  Kender vehicles were green at seeds 1 and 3 respectively. Draw logging
  confirmed the seed-3 `number(1,101)=101` failure boundary and matching C/Go
  stream before the fallback audience.
- Local gates passed: `make fidelity-depth`, `go build ./...`, `go vet ./...`,
  `go test ./...`, `golangci-lint run ./...`, and clean `gofumpt -l .`.
- Hosted checks for PR #1061 were green: lint, security, and test passed;
  build/deploy were skipped by the workflow. CI fired normally, so no retry
  was required.

## Fidelity rules

This slice follows R1 (player-facing bytes), R2 (the registered command
surface), R3 (deterministic draw and ordering parity), R4 (no invented
messages or unreachable player-victim Kender arm), and R5/R5e (verify the
actual C call path). The shared look and target behavior is retained at its
actual call boundary under R5b/R5c.

## Next queue position

Return to clean `main`, pull, run `make fidelity-depth`, reread the depth guide
and this newest handoff, then resweep `src/interpreter.c` against all depth
manifests. `peek` is claimed; `peer` is a shared `do_action` social and
`pick` is owned by the door manifest. The next unclaimed command family in
table order is `plot` at `interpreter.c:607`; do not re-pick any command
already owned by a manifest or delegated boundary.
