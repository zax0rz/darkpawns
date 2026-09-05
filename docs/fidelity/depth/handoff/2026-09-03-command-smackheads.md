# Depth-fidelity handoff — `smackheads`

Date: 2026-09-03

Branch: `glm/depth-smackheads`

Feature PR: #1248 (open; security and lint green, test still pending)

Feature commit: `a1f513264`

## Queue position and result

This session checked out `main`, pulled with `--ff-only`, ran
`make fidelity-depth`, reread `docs/fidelity/DEPTH_TESTING.md` and the newest
dated handoff, and audited the interpreter command table. The special-procedure
inventory remains exhausted. The one blocked row,
`objmagic.sleep-entry-gates`, remains blocked after its one allowed cast-sleep
outlaw/reagent vehicle and was not repicked.

The next unclaimed source-order row was `smackheads` at
`src/interpreter.c:711`. This slice is complete as an implementation and
evidence set, but its feature PR is intentionally unmerged because the hosted
test check remained pending after CI fired normally. Do not merge PR #1248 or
repick this row; continue with the next source-order family, `smell`, at
`src/interpreter.c:712`, while #1248 remains open. Main does not yet contain
the `smackheads` commit or manifest, so the main-branch report remains the
pre-slice frontier below.

Pre-slice main frontier: 3,752 total, 3,651 proven/delegated, 48 blocked, and
53 excluded; actionable completion 3,651/3,699 = 98.7%.

Feature-branch evidence frontier: 3,772 total, 3,671 proven/delegated, 48
blocked, and 53 excluded; actionable completion 3,671/3,719 = 98.7%.

## C call path and observable contract

The registered C row is:

```c
/* src/interpreter.c:711 */
{ "smackheads" , POS_FIGHTING, do_smackheads, 1, 0 },
```

`do_smackheads` in `src/new_cmds.c:966-1102` uses `half_chop`, checks the
skill, resolves two visible room targets, rejects missing/same/self targets,
then checks wielded hands, mount, actor fighting, victim fighting, and
peaceful-room state. Its miss arm draws the AC-adjusted `number(1,101)`, emits
the actor and room slip/duck acts, improves the skill, calls ordered zero
damage twice, and waits three violence pulses. Its success arm emits the
authored grab/bang acts, stuns both victims, calls `damage()` with
`3 * GET_LEVEL(ch)` in C order, waits both victims three pulses, improves the
skill, and waits the actor three pulses.

The Go port now preserves the two ordered targets, C half_chop remainder
parsing, gate bytes, formula, waits, combat enrollment, direct C-style damage
and death bytes, XP, and death cry. The dedicated damage seam avoids the
generic corpse announcement that C does not emit for this path.

## Evidence and implementation boundary

The durable evidence is:

- `cmd/dp-oracle-diff/scenarios/smackheads-no-skill-depth.txt`;
- `cmd/dp-oracle-diff/scenarios/smackheads-gates-depth.txt`;
- `cmd/dp-oracle-diff/scenarios/smackheads-outcome-depth.txt`;
- `cmd/dp-oracle-diff/scenarios/smackheads-wielded-depth.txt`;
- `cmd/dp-oracle-diff/scenarios/smackheads-mounted-depth.txt`;
- `cmd/dp-oracle-diff/scenarios/smackheads-peaceful-depth.txt`;
- `docs/fidelity/depth/smackheads.tsv`;
- `pkg/game/smackheads_depth_test.go`; and
- `pkg/session/smackheads_depth_test.go`.

The required main RED checks found the gangs-message punctuation/argument
divergence and the missing two-target damage/death path. The corrected
vehicles report no normalized divergence for no-skill, early gates, and
outcomes at seeds 1, 2, 3, 5, and 8; wielded, mounted, and peaceful vehicles
are green at seed 1. The outcome vehicle was inspected with `--show-oracle`
and reached the intended C hit/death block. A three-pulse live continuation
was deliberately not claimed: it exposed unrelated shared combat-engine
divergence after the direct two-damage boundary, while the result contract
unit test proves the command's waits and ordered targets. No file under `src/`
or `darkpawns-c-oracle/` was edited.

## Gates and review

The final local gates passed on the feature branch:

- `make fidelity-depth`
- `go build ./...`
- `go vet ./...`
- `go test ./...`
- `golangci-lint run ./...` (with the installed Go binary on PATH)
- `gofumpt -l .` clean
- `git diff --check`

PR #1248's hosted CI fired normally. Security and lint completed green; test
remained pending during the wait. The workflow was not retried because it did
fire, and the PR was not merged because all checks were not green.

This slice follows R1 (player-facing bytes), R2 (registered command surface),
R3 (seed/draw and state ordering), R4 (no invented behavior), R5/R5e (verify
the actual C path and let C win), and R5b/R5c (delegate shared dispatcher and
combat behavior rather than duplicating or fixing forward).

## Continuation

The next session must checkout `main`, pull with `--ff-only`, rerun
`make fidelity-depth`, reread the guide and newest handoff, and claim/audit
`smell` at `src/interpreter.c:712`. Do not repick `smackheads`, `slap`,
`slowns`, `slug`, or the shared `smile` social proof. Leave PR #1248 open until
its hosted test check resolves; never merge it while non-green.
