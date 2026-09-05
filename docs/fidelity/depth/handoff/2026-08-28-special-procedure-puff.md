# Depth-fidelity handoff — puff special procedure

Date: 2026-08-28
Branch: `glm/spec-puff`
Starting main: `191004191` (guild_guard handoff)

## Queue position and inventory

This session refreshed `main`, ran `make fidelity-depth`, read
`docs/fidelity/DEPTH_TESTING.md`, and read the newest prior handoff. The
source-and-registration ordered next active procedure was `puff`, defined at
`src/spec_procs.c:611-718` and assigned to mob vnum 1 at
`src/spec_assign.c:182`.

Before this slice the frontier was 563 total: 550 proven/delegated, 2
blocked, and 11 excluded. This slice adds six rows. The resulting frontier is
569 total: 556 proven/delegated, 2 blocked, and 11 excluded; actionable
completion is 556/558 (99.6%).

## C path and branch claims

`mobile_activity()` skips fighting or asleep mobs and invokes the mob special
with `cmd == 0` (`src/mobact.c:68-93`). The player-command special path always
passes a nonzero canonical command (`src/interpreter.c:1407-1456`), so C's
`if (cmd) return(0)` is a real entry gate rather than a command vehicle.

After the negative-HP `do_say` branch, `SPECIAL(puff)` consumes exactly one
`number(0,90)` draw. Rolls 0-42 have the explicit C cases: exact punctuation-
selected `do_say` strings, five `act(..., TO_ROOM)` emotes, and handled silent
returns. Rolls 43-90 take C's `default: return(0)`. The `do_say` path uses
`src/act.comm.c:759-820`; emotes use `src/comm.c:2392-2555` with C's
hide-invisible flag. The Go implementation now mirrors the complete switch,
the inclusive draw range, punctuation verbs, room audience, and return
contract without carrying over the unrelated legacy script saying table.

## Evidence

Focused RED/GREEN coverage is in `pkg/game/spec_puff_test.go`:

- `TestSpecPuff_CommandAndDeathGates` proves cmd gating and the negative-HP
  death speech.
- `TestSpecPuff_CaseOutputsAndReturnContract` proves representative speech,
  emote, silent, and default cases with forced C rolls.
- `TestSpecPuff_CompleteCOutcomePartition` proves every roll from 0 through 90
  against the C handled/fallthrough partition.

The C-first live vehicle is
`cmd/dp-oracle-diff/scenarios/spec-proc-puff.txt`. On clean main at
`191004191`, seed 1 was RED: C emitted Puff's case-7 exclamation on the third
pulse while the old Go special emitted nothing. After the focused fix, the
vehicle is GREEN with `--show-oracle` for seeds 1, 2, 3, 5, and 8. The
vehicle spawns active vnum 1, strips its unrelated script, removes exits to
prevent post-special wandering, and crosses the 40-heartbeat mobile cadence
with eight `~dpclock pulse 40` pads; both actor and peer receive the C room
bytes.

## Verification and next queue item

The focused tests and all five live differential runs pass. Full repository
gates are run before the PR is opened: `go build ./...`, `go vet ./...`,
`go test ./...`, `golangci-lint run ./...`, clean `gofumpt -l .`, and
`make fidelity-depth`.

After this slice merges, refresh `main` and take the next active
source-and-registration ordered special procedure, `fido` at
`src/spec_procs.c:724` (first active assignment vnum 8063 at
`src/spec_assign.c:293-294`). The blocked `objmagic.sleep-entry-gates` row
remains queued after the special-procedure inventory is exhausted.

Rules applied: R1, R3, R4, R5b, and R5e.
