# Depth handoff — circle command

Date: 2026-08-30
Queue slice: `src/interpreter.c:378`, `circle` / `do_circle`
Starting main: `e6845919f`

## Queue decision

The special-procedure inventory across `src/spec_procs.c`,
`src/spec_procs2.c`, and `src/spec_procs3.c`, including active registration
tables in `src/spec_assign.c`, remains exhausted. The blocked
`objmagic.sleep-entry-gates` row was already attempted through cast-sleep and
remains blocked because C's `SPELL_SLEEP` entry is `TAR_NOT_SELF`; this session
did not repick it. The interpreter sweep therefore selected `circle`, the
first un-manifested family after the merged `charge` slice.

## C path and proof

`src/new_cmds.c:2391-2467` does `one_argument`, target lookup with the
FIGHTING fallback, self and busy gates, wielded piercing-weapon and mounted
gates, the awake `MOB_AWARE` notice/retaliation early return, the delayed
skill-zero message, the `number(1,101)` circle roll, and then
`hit(ch,vict,SKILL_CIRCLE)` or `damage(ch,vict,0,SKILL_CIRCLE)`. The delegated
`src/fight.c:1763-1888` hit path consumes the THAC0 d20 and weapon dice, uses
the integer victim-position multiplier and `backstab_mult(level)/3`, and
routes both outcomes through skill-message set 173. The command waits
`PULSE_VIOLENCE+2` only after the roll paths; the AWARE early return does not
wait.

The valid RED vehicle confirmed three main divergences: Go joined all target
words instead of C's first `one_argument` token, Go always took an invented
direct-success/literal-message path instead of C's two-stage hit and set-173
damage path, and Go omitted the combat aftermath because the pair was not
enrolled. Draw logging confirmed C order as circle roll → hit d20 → weapon
dice → set-173 message dice → improvement. A second AWARE vehicle confirmed
that an idle aware mob immediately swings through `hit(vict,ch)`, while an
already-fighting aware mob only emits the notice trio.

## Changes

- `pkg/command/skill_commands.go`: C `one_argument` parsing and faithful
  caller-side synchronous retaliation ordering.
- `pkg/game/skills.go`: result metadata for retaliation before a later skill
  message.
- `pkg/game/skill_combat.go`: C AWARE behavior, failed-circle retarget order,
  THAC0/weapon-dice hit path, exact integer damage multiplier, set-173 routing,
  deferred improvement, and combat enrollment.
- `pkg/game/damage_stubs.go`: confirmed circle/backstab numeric attack types
  for the C corpse category.
- `pkg/game/{skill_combat_test.go,skill_formulas_golden_test.go,damage_stubs_test.go}`
  and `pkg/command/{skill_commands_test.go,skill_draw_order_test.go}`:
  result, corpse, fallback, and draw-order regressions.
- `cmd/dp-oracle-diff/scenarios/{circle-depth,circle-aware-depth}.txt` and
  `docs/fidelity/depth/circle.tsv`: durable oracle vehicles and manifest rows.

`circle-depth` and `circle-aware-depth` are GREEN at seeds `1,2,3,5,8`.
Local `make fidelity-depth`, `go build ./...`, `go vet ./...`, `go test ./...`,
`golangci-lint run ./...`, and `gofumpt -l .` all pass. The frontier is 1,355
total cases: 1,301 proven/delegated, 14 blocked, and 40 excluded (98.9%
actionable completion).

All changes obey R1/R2/R3/R4 and R5/R5e; the C call-path and draw audit
follow R5b/R5c. Neither `src/` nor `darkpawns-c-oracle/` was edited.

## Next queue item

After this PR merges with all hosted checks green, return to clean `main`,
refresh the frontier, and take `checkload` at `src/interpreter.c:381` in
table order.
