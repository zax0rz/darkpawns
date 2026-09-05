# Depth handoff — 2026-08-28 — `do_flesh_alter`

## Queue position

- Main at session start: `0b4003d263001664bf743831a12aacb527271764` (sleep
  entry-gates attempt already closed in PR #706).
- Special-procedure inventory was exhausted before this session's command
  sweep; the one permitted `objmagic.sleep-entry-gates` attempt remains
  blocked for the potion/quaff arm, with its cast-sleep arm proven.
- This round's next command-table family was `alter`/`flesh`, the two
  interpreter registrations for `do_flesh_alter` at `src/interpreter.c:329`
  and `:446`.
- Next queue item after this round: `ambush` (`src/interpreter.c:332`).

## C call path and branch inventory

The command table requires `POS_FIGHTING`; both names dispatch to
`do_flesh_alter` in `src/new_cmds.c:1886-1938`. The handler first gates on
`GET_SKILL(SKILL_FLESH_ALTER)`, then draws `number(0, 101 +
(FIGHTING(ch) ? 10 : 0))`. Failure emits `You lose your concentration!`, sets
`WAIT_STATE(ch, PULSE_VIOLENCE*2)`, and calls `improve_skill`. Success selects
the weapon name from `flesh_alter_weapon()` (`:1828-1870`). The off branch
clears `AFF_FLESH_ALTER`, subtracts `(level/3)+1` hitroll and `(level/2)+1
damroll`, and sends the molecule/weapon-revert lines. The on branch sets the
affect, adds those bonuses, unequips `WEAR_WIELD` if present with the two
stop-using audience messages, and sends the transformed-hand lines.

## Proof and fixes

- Baseline RED exposed the existing generic `weapon` output and missing state
  transitions. The vehicle also exposed a Go-only skillset key mismatch:
  C's catalog name is `flesh alter`, while the Go handler reads
  `flesh_alter`.
- Go now normalizes that skillset key, mirrors the fighting-dependent roll and
  level weapon bands, toggles the affect and base combat stats, returns a
  wielded item to inventory with checked error logging, emits the ordered actor
  and room strings, and carries the C failure wait plus deferred improvement.
- `flesh-alter-gates` is green for both registered names (seed 1).
- `flesh-alter-depth` is green on seeds 1, 2, 3, 5, and 8, including wielded
  and bare-handed success plus toggle-off.
- `flesh-alter-failure` is green on seeds 1, 2, 3, 5, and 8.
- Because the passive observer stayed silent in this room vehicle, room-message
  composition is claimed only by `TestDoFleshAlterToggleAndUnequip`, not by a
  live audience proof.
- Focused tests and `make fidelity-depth` pass. Current depth report is
  `664 total, 644 proven/delegated, 6 blocked, 14 excluded`; `do_flesh_alter`
  is `10/10`; `interpreter` remains `0/1` because its `sacrifice.unreachable`
  row is excluded.

## Repository state and next action

No files under `src/` or `darkpawns-c-oracle/` were edited. The required alias
handoff from the immediately preceding merged slice is included alongside
this round's changes so the worktree is not left with an uncommitted dated
handoff. Finish the full build/vet/test/lint/gofumpt gates, open one
`glm/depth-flesh-alter` PR, and merge only if every GitHub check is green.
Then checkpoint main and continue with `ambush` in interpreter table order.
