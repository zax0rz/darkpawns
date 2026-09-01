# Depth-fidelity handoff — `parry`

Date: 2026-09-01  
Branch: `glm/depth-parry`  
Feature commit: `3cbe9d0aa`  
Feature PR: #1057 (merged to `main` as `19217de92`)

## Frontier

The clean-main frontier before this slice was 2,717 total cases: 2,646
proven/delegated, 22 blocked, and 49 excluded. After merging `parry`, it is
2,732 total: 2,661 proven/delegated, 22 blocked, and 49 excluded. Actionable
completion is 2,661/2,683 (99.2%). The special-procedure inventory remains
exhausted; `objmagic.sleep-entry-gates` remains the single explicitly blocked
vehicle; the command-table sweep continues after `parry`.

## C-first call path

The registration at `src/interpreter.c:600` is `{ "parry", POS_DEAD,
do_parry, 0, 0 }`. The handler is `src/new_cmds.c:2340-2389`:

1. A player with no `SKILL_PARRY` receives `You're not good enough at
   swordplay to parry!` and returns for the player-command `subcmd=0` path.
2. The handler ignores the typed argument and checks `FIGHTING(ch)`; no
   opponent emits `But you aren't fighting anyone!`, while a non-mutual fight
   emits the exact misspelled `But noone's attacking you!`.
3. A mutual fight without `WEAR_WIELD` emits `Parry with what? You're
   unarmed!`.
4. The manual path draws `number(1,101)`, uses `GET_SKILL(ch,
   SKILL_PARRY)` as its probability, and on failure emits the outmaneuvered
   line, calls `improve_skill`, and sets `WAIT_STATE(ch,
   PULSE_VIOLENCE*3)`.
5. Success emits the actor line, then calls `act()` with `TO_ROOM` and
   `TO_VICT`, marks `IS_PARRIED(FIGHTING(ch))`, and sets
   `WAIT_STATE(ch,PULSE_VIOLENCE*2)`. `fight.c:1999-2007` consumes the
   one-round marker on the opponent's next combat turn.

The registered NPC fighter and paladin `subcmd=1` calls are already owned by
`mob.fighter-parry` and remain delegated under R5b/R5c; this slice covers the
unclaimed player command surface only.

## RED and confirmed fixes

The initial Go command surface had no player `parry` handler, so the RED
vehicles returned `Huh?!?` instead of the C gates and combat branches. The
confirmed fix registers `parry`, preserves the C gate order and ignored
argument, uses the canonical current fighting target, performs the
deterministic manual roll and improve-skill side effect, maps C wait states to
the Go pulse representation, delivers both C act audiences, and records the
combat engine's one-round parry marker. No oracle or C source was edited.

## Evidence

- Scenarios: `cmd/dp-oracle-diff/scenarios/parry-gates-depth.txt`,
  `parry-entry-depth.txt`, `parry-failure-depth.txt`, and `parry-depth.txt`.
- Manifest: `docs/fidelity/depth/parry.tsv` (15/15 proven or delegated rows).
- Focused tests: `pkg/session/parry_depth_test.go`.
- Oracle matrix: seeds `1, 2, 3, 5, 8` for all four scenarios, all
  `result: no normalized divergence`.
- `--show-oracle --seed 1` confirmed the C success audiences and marker-round
  output.
- Local gates passed: `make fidelity-depth`, `go build ./...`, `go vet ./...`,
  `go test ./...`, `golangci-lint run ./...`, and clean `gofumpt -l .`.
- Hosted checks for PR #1057 were green: lint, security, and test passed;
  build/deploy were skipped by the workflow. No retry was required because CI
  fired normally.

## Fidelity rules

This slice follows R1 (player-facing bytes), R2 (the registered command
surface), R3 (multi-seed determinism and state), R4 (no invented NPC/direct
surface), and R5/R5e (verify the actual C call path). The existing NPC
subcommand proof remains delegated under R5b/R5c rather than duplicated.

## Next queue position

Return to clean `main`, pull, run `make fidelity-depth`, reread the depth guide
and this newest handoff, then resweep `src/interpreter.c` against all depth
manifests. `parry` is claimed; `palm` is already covered as a special-procedure
row, so the next unclaimed command family in table order is `pardon` at
`interpreter.c:601`. Do not re-pick any command already owned by a manifest or
delegated boundary.
